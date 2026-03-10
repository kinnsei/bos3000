package callback

import (
	"context"
	"strconv"
	"time"

	"encore.dev/rlog"

	"encore.app/billing"
	"encore.app/callback/fsclient"
	"encore.app/pkg/types"
)

// runCall drives the full call lifecycle as a goroutine.
func (s *Service) runCall(ctx context.Context, call *CallbackCall) {
	v, ok := s.activeCalls.Load(call.CallID)
	if !ok {
		rlog.Error("active call not found", "call_id", call.CallID)
		return
	}
	ac := v.(*activeCall)

	// Read park timeout from config
	parkTimeoutSec := 60
	if val, err := getSystemConfig(ctx, "park_timeout_sec"); err == nil {
		if parsed, err := strconv.Atoi(val); err == nil {
			parkTimeoutSec = parsed
		}
	}

	// Read max duration
	maxDurationSec := 3600
	if call.MaxDuration != nil && *call.MaxDuration > 0 {
		maxDurationSec = *call.MaxDuration
	} else if val, err := getSystemConfig(ctx, "default_max_duration_sec"); err == nil {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxDurationSec = parsed
		}
	}

	// 1. A-leg originate
	aUUID, err := s.fsClient.OriginateALeg(ctx, fsclient.OriginateParams{
		CallID:   call.CallID,
		Number:   call.ANumber,
		CallerID: ptrOr(call.CallerID, ""),
		GatewayIP: ptrOr(call.AGatewayName, ""),
	})
	if err != nil {
		reason := "a_originate_failed"
		call.FailureReason = &reason
		s.finalizeCall(ctx, call, "failed")
		return
	}

	call.AfsUUID = &aUUID
	now := time.Now()
	call.ADialAt = &now
	call.Status = "a_dialing"
	_ = updateCallStatus(ctx, call.CallID, "a_dialing")

	// 2. Wait for A-leg event
	parkTimer := time.NewTimer(time.Duration(parkTimeoutSec) * time.Second)
	defer parkTimer.Stop()

	select {
	case event := <-ac.eventCh:
		if event.EventName == "CHANNEL_ANSWER" && event.Leg == "A" {
			answerAt := event.Timestamp
			call.AAnswerAt = &answerAt
			call.Status = "a_connected"
			_ = updateCallStatus(ctx, call.CallID, "a_connected")
		} else if event.EventName == "CHANNEL_HANGUP" {
			hangupAt := event.Timestamp
			call.AHangupAt = &hangupAt
			call.AHangupCause = &event.HangupCause
			reason := "a_hangup_" + event.HangupCause
			call.FailureReason = &reason
			s.finalizeCall(ctx, call, "failed")
			return
		}
	case <-parkTimer.C:
		_ = s.fsClient.HangupCall(ctx, aUUID, "ALLOTTED_TIMEOUT")
		reason := "a_park_timeout"
		call.FailureReason = &reason
		s.finalizeCall(ctx, call, "failed")
		return
	case <-ctx.Done():
		_ = s.fsClient.HangupCall(ctx, aUUID, "NORMAL_CLEARING")
		reason := "force_hangup"
		call.FailureReason = &reason
		s.finalizeCall(ctx, call, "failed")
		return
	}

	// 3. A-leg answered -> originate B-leg
	bUUID, err := s.fsClient.OriginateBLegAndBridge(ctx, aUUID, fsclient.OriginateParams{
		CallID:   call.CallID,
		Number:   call.BNumber,
		CallerID: ptrOr(call.CallerID, ""),
		GatewayIP: ptrOr(call.BGatewayName, ""),
	})
	if err != nil {
		// B originate failed - hang up A, mark wastage
		_ = s.fsClient.HangupCall(ctx, aUUID, "NORMAL_CLEARING")
		hangupAt := time.Now()
		call.AHangupAt = &hangupAt
		reason := "b_originate_failed"
		call.FailureReason = &reason
		s.finalizeCall(ctx, call, "failed")
		return
	}

	call.BfsUUID = &bUUID
	bDialAt := time.Now()
	call.BDialAt = &bDialAt
	call.Status = "b_dialing"
	_ = updateCallStatus(ctx, call.CallID, "b_dialing")

	// 4. Wait for B-leg events (answer, bridge, hangup) with max duration guard
	bridged := false
	maxDurationTimer := time.NewTimer(time.Duration(maxDurationSec) * time.Second)
	defer maxDurationTimer.Stop()

	for {
		select {
		case event := <-ac.eventCh:
			switch {
			case event.EventName == "CHANNEL_ANSWER" && event.Leg == "B":
				bAnswerAt := event.Timestamp
				call.BAnswerAt = &bAnswerAt
				call.Status = "b_connected"
				_ = updateCallStatus(ctx, call.CallID, "b_connected")

			case event.EventName == "CHANNEL_BRIDGE":
				bridgeAt := event.Timestamp
				call.BridgeAt = &bridgeAt
				call.Status = "bridged"
				_ = updateCallStatus(ctx, call.CallID, "bridged")
				bridged = true

			case event.EventName == "CHANNEL_HANGUP":
				hangupAt := event.Timestamp
				if event.Leg == "A" {
					call.AHangupAt = &hangupAt
					call.AHangupCause = &event.HangupCause
				} else {
					call.BHangupAt = &hangupAt
					call.BHangupCause = &event.HangupCause
				}
				if bridged {
					call.BridgeEndAt = &hangupAt
					s.finalizeCall(ctx, call, "finished")
					return
				}
				// B hung up before bridge
				_ = s.fsClient.HangupCall(ctx, aUUID, "NORMAL_CLEARING")
				aHangup := time.Now()
				call.AHangupAt = &aHangup
				s.finalizeCall(ctx, call, "failed")
				return
			}

		case <-maxDurationTimer.C:
			_ = s.fsClient.HangupCall(ctx, aUUID, "ALLOTTED_TIMEOUT")
			_ = s.fsClient.HangupCall(ctx, bUUID, "ALLOTTED_TIMEOUT")
			hangupBy := "system"
			call.HangupBy = &hangupBy
			s.finalizeCall(ctx, call, "finished")
			return

		case <-ctx.Done():
			_ = s.fsClient.HangupCall(ctx, aUUID, "NORMAL_CLEARING")
			_ = s.fsClient.HangupCall(ctx, bUUID, "NORMAL_CLEARING")
			reason := "force_hangup"
			call.FailureReason = &reason
			s.finalizeCall(ctx, call, "failed")
			return
		}
	}
}

// finalizeCall calculates costs, classifies wastage, and persists final state.
func (s *Service) finalizeCall(ctx context.Context, call *CallbackCall, status string) {
	v, ok := s.activeCalls.Load(call.CallID)
	if !ok {
		return
	}
	ac := v.(*activeCall)

	ac.finalized.Do(func() {
		now := time.Now()

		// Calculate A-leg duration
		if call.AAnswerAt != nil {
			hangup := now
			if call.AHangupAt != nil {
				hangup = *call.AHangupAt
			}
			call.ADurationMs = hangup.Sub(*call.AAnswerAt).Milliseconds()
			if call.ADurationMs < 0 {
				call.ADurationMs = 0
			}
		}

		// Calculate B-leg duration
		if call.BAnswerAt != nil {
			hangup := now
			if call.BHangupAt != nil {
				hangup = *call.BHangupAt
			}
			call.BDurationMs = hangup.Sub(*call.BAnswerAt).Milliseconds()
			if call.BDurationMs < 0 {
				call.BDurationMs = 0
			}
		}

		// Calculate bridge duration
		if call.BridgeAt != nil {
			end := now
			if call.BridgeEndAt != nil {
				end = *call.BridgeEndAt
			}
			call.BridgeDurationMs = end.Sub(*call.BridgeAt).Milliseconds()
			if call.BridgeDurationMs < 0 {
				call.BridgeDurationMs = 0
			}
		}

		// Classify wastage
		thresholdSec := int64(10)
		if val, err := getSystemConfig(ctx, "bridge_broken_early_threshold_sec"); err == nil {
			if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
				thresholdSec = parsed
			}
		}

		wastageType, wastageCost := classifyWastage(call, thresholdSec)
		if wastageType != "" {
			call.WastageType = &wastageType
			call.WastageCost = &wastageCost
			wDur := call.ADurationMs
			if wastageType == "bridge_broken_early" {
				wDur = call.BridgeDurationMs
			}
			call.WastageDurationMs = &wDur
		}

		call.Status = status

		// Finalize billing
		aLegDurSec := call.ADurationMs / 1000
		bLegDurSec := call.BDurationMs / 1000
		finalResp, err := billing.Finalize(ctx, &billing.FinalizeParams{
			UserID:          call.UserID,
			CallID:          call.CallID,
			ALegDurationSec: aLegDurSec,
			BLegDurationSec: bLegDurSec,
			ALegRate:        types.Money(call.ALegRate),
			BLegRate:        types.Money(call.BLegRate),
			PreDeductAmount: types.Money(call.PreDeductAmount),
		})
		if err != nil {
			rlog.Error("billing finalize failed", "call_id", call.CallID, "error", err)
		} else {
			call.ALegCost = int64(finalResp.ALegCost)
			call.BLegCost = int64(finalResp.BLegCost)
			call.TotalCost = int64(finalResp.TotalCost)
		}

		// Release concurrent slot
		_, err = billing.ReleaseSlot(ctx, &billing.ReleaseSlotParams{
			UserID: call.UserID,
		})
		if err != nil {
			rlog.Error("billing release slot failed", "call_id", call.CallID, "error", err)
		}

		// Persist final state
		if err := updateCallFinal(ctx, call); err != nil {
			rlog.Error("update call final failed", "call_id", call.CallID, "error", err)
		}

		// Remove from active calls
		s.activeCalls.Delete(call.CallID)
	})
}

func ptrOr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}
