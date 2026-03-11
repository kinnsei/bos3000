package callback

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"encore.dev/rlog"
	"encore.dev/storage/sqldb"

	"encore.app/billing"
	"encore.app/callback/fsclient"
	"encore.app/pkg/types"
	"encore.app/recording"
	"encore.app/webhook"
)

// authDB references the auth service's database for webhook lookups.
var authDB = sqldb.Named("auth")

// runCall drives the full call lifecycle as a goroutine.
func (s *Service) runCall(ctx context.Context, call *CallbackCall) {
	v, ok := s.activeCalls.Load(call.CallID)
	if !ok {
		rlog.Error("active call not found", "call_id", call.CallID)
		return
	}
	ac := v.(*activeCall)

	// Publish webhook for initiating status
	s.publishWebhook(ctx, call)

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
		CallID:    call.CallID,
		Number:    call.ANumber,
		CallerID:  ptrOr(call.CallerID, ""),
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
	s.publishWebhook(ctx, call)

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
			s.publishWebhook(ctx, call)
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
		CallID:    call.CallID,
		Number:    call.BNumber,
		CallerID:  ptrOr(call.CallerID, ""),
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
	s.publishWebhook(ctx, call)

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
				s.publishWebhook(ctx, call)

				// Bridge A and B legs now that B has answered
				if err := s.fsClient.BridgeCall(ctx, aUUID, bUUID); err != nil {
					rlog.Error("bridge failed", "call_id", call.CallID, "error", err)
					_ = s.fsClient.HangupCall(ctx, aUUID, "NORMAL_CLEARING")
					_ = s.fsClient.HangupCall(ctx, bUUID, "NORMAL_CLEARING")
					reason := "bridge_failed"
					call.FailureReason = &reason
					s.finalizeCall(ctx, call, "failed")
					return
				}

			case event.EventName == "CHANNEL_BRIDGE":
				bridgeAt := event.Timestamp
				call.BridgeAt = &bridgeAt
				call.Status = "bridged"
				_ = updateCallStatus(ctx, call.CallID, "bridged")
				bridged = true

				// Start recording on both legs (best-effort)
				s.startRecording(ctx, call, aUUID, bUUID)

				s.publishWebhook(ctx, call)

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
					// Stop recording before finalization
					s.stopRecording(ctx, call, aUUID, bUUID)
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
			if bridged {
				s.stopRecording(ctx, call, aUUID, bUUID)
			}
			s.finalizeCall(ctx, call, "finished")
			return

		case <-ctx.Done():
			_ = s.fsClient.HangupCall(ctx, aUUID, "NORMAL_CLEARING")
			_ = s.fsClient.HangupCall(ctx, bUUID, "NORMAL_CLEARING")
			reason := "force_hangup"
			call.FailureReason = &reason
			if bridged {
				s.stopRecording(ctx, call, aUUID, bUUID)
			}
			s.finalizeCall(ctx, call, "failed")
			return
		}
	}
}

// startRecording initiates recording on both call legs. Errors are logged but non-blocking.
func (s *Service) startRecording(ctx context.Context, call *CallbackCall, aUUID, bUUID string) {
	if err := s.fsClient.StartRecording(ctx, aUUID, call.CallID, "a"); err != nil {
		rlog.Error("start recording A-leg failed", "call_id", call.CallID, "error", err)
	}
	if err := s.fsClient.StartRecording(ctx, bUUID, call.CallID, "b"); err != nil {
		rlog.Error("start recording B-leg failed", "call_id", call.CallID, "error", err)
	}
	recStatus := recording.StatusRecording
	call.RecordingStatus = &recStatus
	_, _ = db.Exec(ctx, `UPDATE callback_calls SET recording_status = $1 WHERE call_id = $2`,
		recStatus, call.CallID)
}

// stopRecording stops recording on both call legs. Errors are logged but non-blocking.
func (s *Service) stopRecording(ctx context.Context, call *CallbackCall, aUUID, bUUID string) {
	if err := s.fsClient.StopRecording(ctx, aUUID, call.CallID, "a"); err != nil {
		rlog.Error("stop recording A-leg failed", "call_id", call.CallID, "error", err)
	}
	if err := s.fsClient.StopRecording(ctx, bUUID, call.CallID, "b"); err != nil {
		rlog.Error("stop recording B-leg failed", "call_id", call.CallID, "error", err)
	}
}

// publishWebhook looks up the user's webhook config and publishes a webhook event.
// Errors are logged but non-blocking.
func (s *Service) publishWebhook(ctx context.Context, call *CallbackCall) {
	var webhookURL, webhookSecret *string
	err := authDB.QueryRow(ctx,
		`SELECT webhook_url, COALESCE(webhook_secret, '') FROM users WHERE id = $1`,
		call.UserID,
	).Scan(&webhookURL, &webhookSecret)
	if err != nil || webhookURL == nil || *webhookURL == "" {
		return
	}

	payload := buildWebhookPayload(call)
	secret := ""
	if webhookSecret != nil {
		secret = *webhookSecret
	}
	if err := webhook.CreateAndPublishWebhook(ctx, call.CallID, call.UserID, *webhookURL, secret, payload); err != nil {
		rlog.Error("webhook publish failed", "call_id", call.CallID, "status", call.Status, "error", err)
	}
}

// buildWebhookPayload constructs a WebhookPayload from the current call state.
func buildWebhookPayload(call *CallbackCall) *webhook.WebhookPayload {
	payload := &webhook.WebhookPayload{
		EventType:  "call." + call.Status,
		CallID:     call.CallID,
		Status:     call.Status,
		CustomData: call.CustomData,
		Timestamp:  time.Now(),
		ALeg: &webhook.LegDetail{
			Number:   call.ANumber,
			Status:   aLegStatus(call),
			DialAt:   call.ADialAt,
			AnswerAt: call.AAnswerAt,
			HangupAt: call.AHangupAt,
		},
	}

	if call.AHangupCause != nil {
		payload.ALeg.HangupCause = *call.AHangupCause
	}

	if call.BDialAt != nil {
		bLeg := &webhook.LegDetail{
			Number:   call.BNumber,
			Status:   bLegStatus(call),
			DialAt:   call.BDialAt,
			AnswerAt: call.BAnswerAt,
			HangupAt: call.BHangupAt,
		}
		if call.BHangupCause != nil {
			bLeg.HangupCause = *call.BHangupCause
		}
		payload.BLeg = bLeg
	}

	if call.BridgeAt != nil {
		bridgeDur := 0
		if call.BridgeDurationMs > 0 {
			bridgeDur = int(call.BridgeDurationMs / 1000)
		}
		payload.Bridge = &webhook.BridgeDetail{
			BridgedAt: call.BridgeAt,
			Duration:  bridgeDur,
		}
	}

	return payload
}

func aLegStatus(call *CallbackCall) string {
	if call.AHangupAt != nil {
		return "hangup"
	}
	if call.AAnswerAt != nil {
		return "answered"
	}
	if call.ADialAt != nil {
		return "dialing"
	}
	return "pending"
}

func bLegStatus(call *CallbackCall) string {
	if call.BHangupAt != nil {
		return "hangup"
	}
	if call.BAnswerAt != nil {
		return "answered"
	}
	if call.BDialAt != nil {
		return "dialing"
	}
	return "pending"
}

// finalizeCall calculates costs, classifies wastage, and persists final state.
func (s *Service) finalizeCall(_ context.Context, call *CallbackCall, status string) {
	v, ok := s.activeCalls.Load(call.CallID)
	if !ok {
		return
	}
	ac := v.(*activeCall)

	ac.finalized.Do(func() {
		// Use background context for finalization since callCtx may be cancelled.
		bgCtx := context.Background()
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
		if val, err := getSystemConfig(bgCtx, "bridge_broken_early_threshold_sec"); err == nil {
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
		finalResp, err := billing.Finalize(bgCtx, &billing.FinalizeParams{
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
		_, err = billing.ReleaseSlot(bgCtx, &billing.ReleaseSlotParams{
			UserID: call.UserID,
		})
		if err != nil {
			rlog.Error("billing release slot failed", "call_id", call.CallID, "error", err)
		}

		// Publish recording merge for bridged calls with active recording
		if call.BridgeAt != nil && call.RecordingStatus != nil && *call.RecordingStatus == recording.StatusRecording {
			mergingStatus := recording.StatusMerging
			call.RecordingStatus = &mergingStatus
			_, pubErr := recording.RecordingMergeTopic.Publish(bgCtx, &recording.RecordingMergeEvent{
				CallID:     call.CallID,
				CustomerID: call.UserID,
				AFilePath:  fmt.Sprintf("/var/lib/freeswitch/recordings/%s_a.wav", call.CallID),
				BFilePath:  fmt.Sprintf("/var/lib/freeswitch/recordings/%s_b.wav", call.CallID),
				Date:       time.Now().Format("2006-01-02"),
			})
			if pubErr != nil {
				rlog.Error("recording merge publish failed", "call_id", call.CallID, "error", pubErr)
				failedStatus := recording.StatusFailed
				call.RecordingStatus = &failedStatus
			}
		}

		// Persist final state
		if err := updateCallFinal(bgCtx, call); err != nil {
			rlog.Error("update call final failed", "call_id", call.CallID, "error", err)
		}

		// Publish final webhook with complete CDR data
		s.publishWebhook(bgCtx, call)

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
