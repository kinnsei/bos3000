package callback

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"encore.app/callback/fsclient"
)

// CallbackCall maps to the callback_calls table.
type CallbackCall struct {
	ID     int64  `json:"id"`
	CallID string `json:"call_id"`
	UserID int64  `json:"user_id"`

	// Request params
	ANumber     string          `json:"a_number"`
	BNumber     string          `json:"b_number"`
	CallerID    *string         `json:"caller_id,omitempty"`
	MaxDuration *int            `json:"max_duration,omitempty"`
	CustomData  json.RawMessage `json:"custom_data,omitempty"`
	CallbackURL *string         `json:"callback_url,omitempty"`

	Status string `json:"status"`

	// A-leg
	AfsUUID      *string    `json:"a_fs_uuid,omitempty"`
	AGatewayName *string    `json:"a_gateway_name,omitempty"`
	AGatewayID   *int64     `json:"a_gateway_id,omitempty"`
	ADialAt      *time.Time `json:"a_dial_at,omitempty"`
	AAnswerAt    *time.Time `json:"a_answer_at,omitempty"`
	AHangupAt    *time.Time `json:"a_hangup_at,omitempty"`
	AHangupCause *string    `json:"a_hangup_cause,omitempty"`
	ADurationMs  int64      `json:"a_duration_ms"`

	// B-leg
	BfsUUID      *string    `json:"b_fs_uuid,omitempty"`
	BGatewayName *string    `json:"b_gateway_name,omitempty"`
	BGatewayID   *int64     `json:"b_gateway_id,omitempty"`
	BDialAt      *time.Time `json:"b_dial_at,omitempty"`
	BAnswerAt    *time.Time `json:"b_answer_at,omitempty"`
	BHangupAt    *time.Time `json:"b_hangup_at,omitempty"`
	BHangupCause *string    `json:"b_hangup_cause,omitempty"`
	BDurationMs  int64      `json:"b_duration_ms"`

	// Bridge
	BridgeAt         *time.Time `json:"bridge_at,omitempty"`
	BridgeEndAt      *time.Time `json:"bridge_end_at,omitempty"`
	BridgeDurationMs int64      `json:"bridge_duration_ms"`

	// Billing (fen)
	ALegRate        int64 `json:"a_leg_rate"`
	BLegRate        int64 `json:"b_leg_rate"`
	PreDeductAmount int64 `json:"pre_deduct_amount"`
	ALegCost        int64 `json:"a_leg_cost"`
	BLegCost        int64 `json:"b_leg_cost"`
	TotalCost       int64 `json:"total_cost"`

	// Wastage
	WastageType       *string `json:"wastage_type,omitempty"`
	WastageCost       *int64  `json:"wastage_cost,omitempty"`
	WastageDurationMs *int64  `json:"wastage_duration_ms,omitempty"`

	// Termination
	HangupBy      *string `json:"hangup_by,omitempty"`
	FailureReason *string `json:"failure_reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// activeCall tracks an in-progress call in memory.
type activeCall struct {
	call      *CallbackCall
	cancel    context.CancelFunc
	eventCh   chan fsclient.CallEvent
	finalized sync.Once
}
