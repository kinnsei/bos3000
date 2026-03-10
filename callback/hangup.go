package callback

import (
	"context"
	"errors"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
)

// ForceHangupResponse contains the result of a force hangup.
type ForceHangupResponse struct {
	CallID  string `json:"call_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ForceHangup terminates an active call.
//
//encore:api auth method=POST path=/callbacks/:id/hangup
func (s *Service) ForceHangup(ctx context.Context, id string) (*ForceHangupResponse, error) {
	ad := getAuthData(ctx)
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	// Verify call exists and ownership
	call, err := getCall(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "call not found"}
		}
		return nil, err
	}

	if ad.Role != "admin" && call.UserID != ad.UserID {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "access denied"}
	}

	if call.Status == "finished" || call.Status == "failed" {
		return nil, &errs.Error{Code: errs.FailedPrecondition, Message: "call already terminated"}
	}

	// Set hangup_by based on role
	hangupBy := ad.Role
	call.HangupBy = &hangupBy

	// Cancel the state machine context
	if v, ok := s.activeCalls.Load(id); ok {
		ac := v.(*activeCall)
		ac.call.HangupBy = &hangupBy
		ac.cancel()
	}

	// Also directly hang up the call legs
	if call.AfsUUID != nil {
		_ = s.fsClient.HangupCall(ctx, *call.AfsUUID, "NORMAL_CLEARING")
	}
	if call.BfsUUID != nil {
		_ = s.fsClient.HangupCall(ctx, *call.BfsUUID, "NORMAL_CLEARING")
	}

	return &ForceHangupResponse{
		CallID:  id,
		Status:  "finished",
		Message: "Call terminated",
	}, nil
}
