package callback

import (
	"context"
	"encoding/json"

	"encore.dev/beta/errs"
	"github.com/google/uuid"

	"encore.app/billing"
	"encore.app/compliance"
	"encore.app/callback/fsclient"
	"encore.app/routing"
)

// InitiateCallbackParams contains the request body for initiating a callback.
type InitiateCallbackParams struct {
	ANumber          string          `json:"a_number"`
	BNumber          string          `json:"b_number"`
	CallerID         *string         `json:"caller_id,omitempty"`
	MaxDuration      *int            `json:"max_duration,omitempty"`
	CustomData       json.RawMessage `json:"custom_data,omitempty"`
	PreferredGateway *string         `json:"preferred_gateway,omitempty"`
	CallbackURL      *string         `json:"callback_url,omitempty"`
}

func (p *InitiateCallbackParams) Validate() error {
	if p.ANumber == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "a_number is required"}
	}
	if p.BNumber == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "b_number is required"}
	}
	if len(p.CustomData) > 1024 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "custom_data must be <= 1024 bytes"}
	}
	if len(p.CustomData) > 0 && !json.Valid(p.CustomData) {
		return &errs.Error{Code: errs.InvalidArgument, Message: "custom_data must be valid JSON"}
	}
	if p.MaxDuration != nil && *p.MaxDuration <= 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "max_duration must be positive"}
	}
	return nil
}

// InitiateCallbackResponse contains the response for a callback initiation.
type InitiateCallbackResponse struct {
	CallID string `json:"call_id"`
	Status string `json:"status"`
}

// InitiateCallback starts a new callback call.
//
//encore:api auth method=POST path=/callbacks
func (s *Service) InitiateCallback(ctx context.Context, p *InitiateCallbackParams) (*InitiateCallbackResponse, error) {
	// Validate explicitly so direct calls (e.g. tests) also run validation.
	if err := p.Validate(); err != nil {
		return nil, err
	}

	ad := getAuthData(ctx)
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	// Check blacklist
	if err := compliance.CheckBlacklist(ctx, &compliance.CheckBlacklistParams{
		CalledNumber: p.BNumber,
		UserID:       ad.UserID,
	}); err != nil {
		return nil, err
	}

	// Check daily limit (get from billing account)
	acct, err := billing.GetAccount(ctx, ad.UserID)
	if err != nil {
		return nil, err
	}

	if _, err := compliance.CheckDailyLimit(ctx, &compliance.CheckDailyLimitParams{
		UserID:     ad.UserID,
		DailyLimit: acct.MaxConcurrent * 1000, // generous daily limit based on concurrency
	}); err != nil {
		return nil, err
	}

	// Resolve rates
	calledPrefix := p.BNumber
	if len(calledPrefix) > 3 {
		calledPrefix = calledPrefix[:3]
	}
	rateResp, err := billing.ResolveRate(ctx, &billing.ResolveRateParams{
		UserID:       ad.UserID,
		CalledPrefix: calledPrefix,
	})
	if err != nil {
		return nil, err
	}

	callID := uuid.New().String()

	// Pre-deduct
	preDeductResp, err := billing.PreDeduct(ctx, &billing.PreDeductParams{
		UserID:   ad.UserID,
		CallID:   callID,
		ALegRate: rateResp.ALegRate,
		BLegRate: rateResp.BLegRate,
	})
	if err != nil {
		return nil, err
	}

	// Acquire concurrent slot
	_, err = billing.AcquireSlot(ctx, &billing.AcquireSlotParams{
		UserID:        ad.UserID,
		MaxConcurrent: acct.MaxConcurrent,
	})
	if err != nil {
		// Refund pre-deduction
		_, _ = billing.Finalize(ctx, &billing.FinalizeParams{
			UserID:          ad.UserID,
			CallID:          callID,
			ALegDurationSec: 0,
			BLegDurationSec: 0,
			ALegRate:        rateResp.ALegRate,
			BLegRate:        rateResp.BLegRate,
			PreDeductAmount: preDeductResp.Amount,
		})
		return nil, err
	}

	// Pick A-leg gateway
	aLeg, err := routing.PickALeg(ctx)
	if err != nil {
		// Release slot and refund
		_, _ = billing.ReleaseSlot(ctx, &billing.ReleaseSlotParams{UserID: ad.UserID})
		_, _ = billing.Finalize(ctx, &billing.FinalizeParams{
			UserID: ad.UserID, CallID: callID,
			PreDeductAmount: preDeductResp.Amount,
			ALegRate: rateResp.ALegRate, BLegRate: rateResp.BLegRate,
		})
		return nil, err
	}

	// Pick B-leg gateway
	bLeg, err := routing.PickBLeg(ctx, &routing.PickBLegParams{CalledNumber: p.BNumber})
	if err != nil {
		_, _ = billing.ReleaseSlot(ctx, &billing.ReleaseSlotParams{UserID: ad.UserID})
		_, _ = billing.Finalize(ctx, &billing.FinalizeParams{
			UserID: ad.UserID, CallID: callID,
			PreDeductAmount: preDeductResp.Amount,
			ALegRate: rateResp.ALegRate, BLegRate: rateResp.BLegRate,
		})
		return nil, err
	}

	// Select caller ID
	callerID := ""
	if p.CallerID != nil {
		callerID = *p.CallerID
	} else {
		didResp, err := routing.SelectDID(ctx, &routing.SelectDIDParams{UserID: ad.UserID})
		if err == nil {
			callerID = didResp.Number
		}
	}

	// Build call record
	call := &CallbackCall{
		CallID:          callID,
		UserID:          ad.UserID,
		ANumber:         p.ANumber,
		BNumber:         p.BNumber,
		CallerID:        strPtr(callerID),
		MaxDuration:     p.MaxDuration,
		CustomData:      p.CustomData,
		CallbackURL:     p.CallbackURL,
		Status:          "initiating",
		AGatewayName:    &aLeg.SIPAddress,
		AGatewayID:      &aLeg.GatewayID,
		BGatewayName:    &bLeg.SIPAddress,
		BGatewayID:      &bLeg.GatewayID,
		ALegRate:        int64(rateResp.ALegRate),
		BLegRate:        int64(rateResp.BLegRate),
		PreDeductAmount: int64(preDeductResp.Amount),
	}

	if err := insertCall(ctx, call); err != nil {
		_, _ = billing.ReleaseSlot(ctx, &billing.ReleaseSlotParams{UserID: ad.UserID})
		_, _ = billing.Finalize(ctx, &billing.FinalizeParams{
			UserID: ad.UserID, CallID: callID,
			PreDeductAmount: preDeductResp.Amount,
			ALegRate: rateResp.ALegRate, BLegRate: rateResp.BLegRate,
		})
		return nil, err
	}

	// Create active call and launch state machine
	callCtx, cancel := context.WithCancel(context.Background())
	ac := &activeCall{
		call:    call,
		cancel:  cancel,
		eventCh: make(chan fsclient.CallEvent, 10),
	}
	s.activeCalls.Store(call.CallID, ac)

	go s.runCall(callCtx, call)

	return &InitiateCallbackResponse{
		CallID: callID,
		Status: "initiating",
	}, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
