package callback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"encore.dev/config"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"

	"encore.app/callback/fsclient"
)

var db = sqldb.NewDatabase("callback", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

// CallbackConfig holds FreeSWITCH connection configuration.
type CallbackConfig struct {
	FSMode            config.String // "mock" or "real"
	FSPrimaryAddress  config.String
	FSPrimaryPassword config.String
	FSStandbyAddress  config.String
	FSStandbyPassword config.String
}

var cfg = config.Load[*CallbackConfig]()

//encore:service
type Service struct {
	fsClient    fsclient.FSClient
	fsManager   *fsclient.FSClientManager
	activeCalls sync.Map // map[string]*activeCall
	hub         *Hub
}

func initService() (*Service, error) {
	hub := NewHub()
	go hub.Run()

	svc := &Service{
		hub: hub,
	}

	fsMode := cfg.FSMode()
	if fsMode == "real" {
		primaryAddr := cfg.FSPrimaryAddress()
		primaryPwd := cfg.FSPrimaryPassword()
		standbyAddr := cfg.FSStandbyAddress()
		standbyPwd := cfg.FSStandbyPassword()

		manager := fsclient.NewFSClientManager(primaryAddr, primaryPwd, standbyAddr, standbyPwd, func(failedIdx int) {
			rlog.Error("FreeSWITCH failover triggered", "failed_idx", failedIdx)
			// Finalize all in-flight calls on the failed instance
			svc.activeCalls.Range(func(key, value any) bool {
				ac := value.(*activeCall)
				reason := "fs_connection_lost"
				ac.call.FailureReason = &reason
				svc.finalizeCall(context.Background(), ac.call, "failed")
				return true
			})
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := manager.Connect(ctx); err != nil {
			return nil, fmt.Errorf("fsclient manager connect: %w", err)
		}

		svc.fsManager = manager

		// Pick initial client for event handler registration
		client, err := manager.Pick()
		if err != nil {
			return nil, fmt.Errorf("fsclient manager pick: %w", err)
		}
		svc.fsClient = client

		// Register event handlers on all managed instances
		svc.registerEventHandlers(client)

		rlog.Info("running in real FSClient mode",
			"primary", primaryAddr, "standby", standbyAddr)
	} else {
		mockClient := fsclient.NewMockFSClient(fsclient.MockConfig{
			ALegResult:     "answer",
			BLegResult:     "answer",
			BridgeResult:   "stable",
			BridgeDuration: 30 * time.Second,
		})
		svc.fsClient = mockClient
		svc.registerEventHandlers(mockClient)
		rlog.Info("running in mock FSClient mode")
	}

	return svc, nil
}

// registerEventHandlers sets up event routing from FSClient to active call channels.
func (s *Service) registerEventHandlers(client fsclient.FSClient) {
	for _, eventName := range []string{"CHANNEL_ANSWER", "CHANNEL_BRIDGE", "CHANNEL_HANGUP"} {
		client.RegisterEventHandler(eventName, func(event fsclient.CallEvent) {
			if v, ok := s.activeCalls.Load(event.CallID); ok {
				ac := v.(*activeCall)
				select {
				case ac.eventCh <- event:
				default:
					rlog.Warn("event channel full, dropping event",
						"call_id", event.CallID, "event", event.EventName)
				}
			} else {
				rlog.Warn("orphan event, call not in active map",
					"call_id", event.CallID, "event", event.EventName, "uuid", event.UUID)
			}
		})
	}
}

// Shutdown gracefully terminates the callback service.
func (s *Service) Shutdown(force context.Context) {
	rlog.Info("callback service shutting down, finalizing in-flight calls")

	// Finalize all remaining active calls
	s.activeCalls.Range(func(key, value any) bool {
		ac := value.(*activeCall)
		reason := "service_shutdown"
		ac.call.FailureReason = &reason
		ac.cancel()
		return true
	})

	// Wait briefly for goroutines to finish
	select {
	case <-time.After(5 * time.Second):
	case <-force.Done():
	}
}

// --- DB helpers ---

func insertCall(ctx context.Context, call *CallbackCall) error {
	return db.QueryRow(ctx, `
		INSERT INTO callback_calls (
			call_id, user_id, a_number, b_number, caller_id, max_duration,
			custom_data, callback_url, status,
			a_gateway_name, a_gateway_id, b_gateway_name, b_gateway_id,
			a_leg_rate, b_leg_rate, pre_deduct_amount
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id, created_at, updated_at
	`, call.CallID, call.UserID, call.ANumber, call.BNumber, call.CallerID,
		call.MaxDuration, call.CustomData, call.CallbackURL, call.Status,
		call.AGatewayName, call.AGatewayID, call.BGatewayName, call.BGatewayID,
		call.ALegRate, call.BLegRate, call.PreDeductAmount,
	).Scan(&call.ID, &call.CreatedAt, &call.UpdatedAt)
}

func updateCallStatus(ctx context.Context, callID, status string) error {
	_, err := db.Exec(ctx, `
		UPDATE callback_calls SET status = $1, updated_at = NOW()
		WHERE call_id = $2
	`, status, callID)
	return err
}

func updateCallFinal(ctx context.Context, call *CallbackCall) error {
	_, err := db.Exec(ctx, `
		UPDATE callback_calls SET
			status = $1,
			a_fs_uuid = $2, a_dial_at = $3, a_answer_at = $4, a_hangup_at = $5,
			a_hangup_cause = $6, a_duration_ms = $7,
			b_fs_uuid = $8, b_dial_at = $9, b_answer_at = $10, b_hangup_at = $11,
			b_hangup_cause = $12, b_duration_ms = $13,
			bridge_at = $14, bridge_end_at = $15, bridge_duration_ms = $16,
			a_leg_cost = $17, b_leg_cost = $18, total_cost = $19,
			wastage_type = $20, wastage_cost = $21, wastage_duration_ms = $22,
			hangup_by = $23, failure_reason = $24,
			recording_status = $25,
			updated_at = NOW()
		WHERE call_id = $26
	`, call.Status,
		call.AfsUUID, call.ADialAt, call.AAnswerAt, call.AHangupAt,
		call.AHangupCause, call.ADurationMs,
		call.BfsUUID, call.BDialAt, call.BAnswerAt, call.BHangupAt,
		call.BHangupCause, call.BDurationMs,
		call.BridgeAt, call.BridgeEndAt, call.BridgeDurationMs,
		call.ALegCost, call.BLegCost, call.TotalCost,
		call.WastageType, call.WastageCost, call.WastageDurationMs,
		call.HangupBy, call.FailureReason,
		call.RecordingStatus,
		call.CallID,
	)
	return err
}

func getCall(ctx context.Context, callID string) (*CallbackCall, error) {
	var call CallbackCall
	err := db.QueryRow(ctx, `
		SELECT id, call_id, user_id, a_number, b_number, caller_id, max_duration,
			custom_data, callback_url, status,
			a_fs_uuid, a_gateway_name, a_gateway_id, a_dial_at, a_answer_at, a_hangup_at,
			a_hangup_cause, a_duration_ms,
			b_fs_uuid, b_gateway_name, b_gateway_id, b_dial_at, b_answer_at, b_hangup_at,
			b_hangup_cause, b_duration_ms,
			bridge_at, bridge_end_at, bridge_duration_ms,
			a_leg_rate, b_leg_rate, pre_deduct_amount, a_leg_cost, b_leg_cost, total_cost,
			recording_status, recording_key, recording_a_key, recording_b_key,
			wastage_type, wastage_cost, wastage_duration_ms,
			hangup_by, failure_reason,
			created_at, updated_at
		FROM callback_calls WHERE call_id = $1
	`, callID).Scan(
		&call.ID, &call.CallID, &call.UserID, &call.ANumber, &call.BNumber,
		&call.CallerID, &call.MaxDuration, &call.CustomData, &call.CallbackURL, &call.Status,
		&call.AfsUUID, &call.AGatewayName, &call.AGatewayID,
		&call.ADialAt, &call.AAnswerAt, &call.AHangupAt,
		&call.AHangupCause, &call.ADurationMs,
		&call.BfsUUID, &call.BGatewayName, &call.BGatewayID,
		&call.BDialAt, &call.BAnswerAt, &call.BHangupAt,
		&call.BHangupCause, &call.BDurationMs,
		&call.BridgeAt, &call.BridgeEndAt, &call.BridgeDurationMs,
		&call.ALegRate, &call.BLegRate, &call.PreDeductAmount,
		&call.ALegCost, &call.BLegCost, &call.TotalCost,
		&call.RecordingStatus, &call.RecordingKey, &call.RecordingAKey, &call.RecordingBKey,
		&call.WastageType, &call.WastageCost, &call.WastageDurationMs,
		&call.HangupBy, &call.FailureReason,
		&call.CreatedAt, &call.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get call: %w", err)
	}
	return &call, nil
}

func getSystemConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := db.QueryRow(ctx, `
		SELECT config_value FROM system_configs WHERE config_key = $1
	`, key).Scan(&value)
	return value, err
}
