package callback

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
)

// LegStatus contains the status details for one call leg.
type LegStatus struct {
	DialAt      *time.Time `json:"dial_at,omitempty"`
	AnswerAt    *time.Time `json:"answer_at,omitempty"`
	HangupAt    *time.Time `json:"hangup_at,omitempty"`
	HangupCause *string    `json:"hangup_cause,omitempty"`
	DurationMs  int64      `json:"duration_ms"`
}

// BridgeStatus contains bridge timing details.
type BridgeStatus struct {
	At         *time.Time `json:"at,omitempty"`
	EndAt      *time.Time `json:"end_at,omitempty"`
	DurationMs int64      `json:"duration_ms"`
}

// BillingStatus contains billing breakdown.
type BillingStatus struct {
	ALegRate        int64 `json:"a_leg_rate"`
	BLegRate        int64 `json:"b_leg_rate"`
	PreDeductAmount int64 `json:"pre_deduct_amount"`
	ALegCost        int64 `json:"a_leg_cost"`
	BLegCost        int64 `json:"b_leg_cost"`
	TotalCost       int64 `json:"total_cost"`
}

// WastageStatus contains wastage details.
type WastageStatus struct {
	Type       *string `json:"type,omitempty"`
	Cost       *int64  `json:"cost,omitempty"`
	DurationMs *int64  `json:"duration_ms,omitempty"`
}

// CallStatusResponse contains the full status of a callback call.
type CallStatusResponse struct {
	CallID        string          `json:"call_id"`
	Status        string          `json:"status"`
	ANumber       string          `json:"a_number"`
	BNumber       string          `json:"b_number"`
	CallerID      *string         `json:"caller_id,omitempty"`
	ALeg          LegStatus       `json:"a_leg"`
	BLeg          LegStatus       `json:"b_leg"`
	Bridge        BridgeStatus    `json:"bridge"`
	Billing       BillingStatus   `json:"billing"`
	Wastage       WastageStatus   `json:"wastage"`
	HangupBy      *string         `json:"hangup_by,omitempty"`
	FailureReason *string         `json:"failure_reason,omitempty"`
	CustomData    json.RawMessage `json:"custom_data,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// GetCallStatus returns the current status of a callback call.
//
//encore:api auth method=GET path=/callbacks/:id
func (s *Service) GetCallStatus(ctx context.Context, id string) (*CallStatusResponse, error) {
	ad := getAuthData(ctx)
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	// Try active call first for fresher data
	if v, ok := s.activeCalls.Load(id); ok {
		ac := v.(*activeCall)
		call := ac.call
		if ad.Role != "admin" && call.UserID != ad.UserID {
			return nil, &errs.Error{Code: errs.NotFound, Message: "call not found"}
		}
		return callToResponse(call), nil
	}

	// Fallback to DB
	call, err := getCall(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "call not found"}
		}
		return nil, err
	}

	if ad.Role != "admin" && call.UserID != ad.UserID {
		return nil, &errs.Error{Code: errs.NotFound, Message: "call not found"}
	}

	return callToResponse(call), nil
}

func callToResponse(call *CallbackCall) *CallStatusResponse {
	return &CallStatusResponse{
		CallID:   call.CallID,
		Status:   call.Status,
		ANumber:  call.ANumber,
		BNumber:  call.BNumber,
		CallerID: call.CallerID,
		ALeg: LegStatus{
			DialAt:      call.ADialAt,
			AnswerAt:    call.AAnswerAt,
			HangupAt:    call.AHangupAt,
			HangupCause: call.AHangupCause,
			DurationMs:  call.ADurationMs,
		},
		BLeg: LegStatus{
			DialAt:      call.BDialAt,
			AnswerAt:    call.BAnswerAt,
			HangupAt:    call.BHangupAt,
			HangupCause: call.BHangupCause,
			DurationMs:  call.BDurationMs,
		},
		Bridge: BridgeStatus{
			At:         call.BridgeAt,
			EndAt:      call.BridgeEndAt,
			DurationMs: call.BridgeDurationMs,
		},
		Billing: BillingStatus{
			ALegRate:        call.ALegRate,
			BLegRate:        call.BLegRate,
			PreDeductAmount: call.PreDeductAmount,
			ALegCost:        call.ALegCost,
			BLegCost:        call.BLegCost,
			TotalCost:       call.TotalCost,
		},
		Wastage: WastageStatus{
			Type:       call.WastageType,
			Cost:       call.WastageCost,
			DurationMs: call.WastageDurationMs,
		},
		HangupBy:      call.HangupBy,
		FailureReason: call.FailureReason,
		CustomData:    call.CustomData,
		CreatedAt:     call.CreatedAt,
		UpdatedAt:     call.UpdatedAt,
	}
}
