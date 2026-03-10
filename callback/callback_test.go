package callback

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"encore.dev/et"

	authpkg "encore.app/auth"
	"encore.app/billing"
	"encore.app/callback/fsclient"
	"encore.app/compliance"
	"encore.app/pkg/types"
	"encore.app/routing"
)

// setupMocks configures mock endpoints for all Phase 1 service calls.
func setupMocks(t *testing.T) {
	t.Helper()

	et.MockEndpoint(billing.ResolveRate, func(ctx context.Context, p *billing.ResolveRateParams) (*billing.ResolveRateResponse, error) {
		return &billing.ResolveRateResponse{ALegRate: 60, BLegRate: 120, Source: "mock"}, nil
	})

	et.MockEndpoint(billing.PreDeduct, func(ctx context.Context, p *billing.PreDeductParams) (*billing.PreDeductResponse, error) {
		return &billing.PreDeductResponse{Amount: 5400, TxID: 1}, nil
	})

	et.MockEndpoint(billing.AcquireSlot, func(ctx context.Context, p *billing.AcquireSlotParams) (*billing.AcquireSlotResponse, error) {
		return &billing.AcquireSlotResponse{CurrentSlots: 1}, nil
	})

	et.MockEndpoint(billing.ReleaseSlot, func(ctx context.Context, p *billing.ReleaseSlotParams) (*billing.ReleaseSlotResponse, error) {
		return &billing.ReleaseSlotResponse{Success: true}, nil
	})

	et.MockEndpoint(billing.Finalize, func(ctx context.Context, p *billing.FinalizeParams) (*billing.FinalizeResponse, error) {
		aLegCost := types.Money(0)
		if p.ALegDurationSec > 0 {
			aLegCost = types.Money(types.CeilDiv(p.ALegDurationSec, 6) * 6 * int64(p.ALegRate) / 60)
		}
		bLegCost := types.Money(0)
		if p.BLegDurationSec > 0 {
			bLegCost = types.Money(types.CeilDiv(p.BLegDurationSec, 60) * 60 * int64(p.BLegRate) / 60)
		}
		return &billing.FinalizeResponse{
			ALegCost: aLegCost, BLegCost: bLegCost,
			TotalCost: aLegCost + bLegCost, Refund: p.PreDeductAmount - aLegCost - bLegCost,
		}, nil
	})

	et.MockEndpoint(billing.GetAccount, func(ctx context.Context, userId int64) (*billing.AccountResponse, error) {
		return &billing.AccountResponse{
			UserID: userId, Balance: 100000, CreditLimit: 0,
			MaxConcurrent: 10, Status: "active",
		}, nil
	})

	et.MockEndpoint(routing.PickALeg, func(ctx context.Context) (*routing.PickALegResponse, error) {
		return &routing.PickALegResponse{GatewayID: 1, Name: "gw-a-1", SIPAddress: "10.0.0.1:5060"}, nil
	})

	et.MockEndpoint(routing.PickBLeg, func(ctx context.Context, p *routing.PickBLegParams) (*routing.PickBLegResponse, error) {
		return &routing.PickBLegResponse{GatewayID: 2, Name: "gw-b-1", SIPAddress: "10.0.0.2:5060"}, nil
	})

	et.MockEndpoint(routing.SelectDID, func(ctx context.Context, p *routing.SelectDIDParams) (*routing.SelectDIDResponse, error) {
		return &routing.SelectDIDResponse{Number: "10000000000", DIDID: 1}, nil
	})

	et.MockEndpoint(compliance.CheckBlacklist, func(ctx context.Context, p *compliance.CheckBlacklistParams) error {
		return nil
	})

	et.MockEndpoint(compliance.CheckDailyLimit, func(ctx context.Context, p *compliance.CheckDailyLimitParams) (*compliance.CheckDailyLimitResponse, error) {
		return &compliance.CheckDailyLimitResponse{CurrentCount: 1}, nil
	})
}

func withAuth(ctx context.Context, userID int64, role string) context.Context {
	return withAuthData(ctx, &authpkg.AuthData{
		UserID:   userID,
		Role:     role,
		Username: "testuser",
	})
}

func setMockFSClient(t *testing.T, svc *Service, config fsclient.MockConfig) {
	t.Helper()
	mockClient := fsclient.NewMockFSClient(config)
	svc.fsClient = mockClient

	for _, eventName := range []string{"CHANNEL_ANSWER", "CHANNEL_BRIDGE", "CHANNEL_HANGUP"} {
		mockClient.RegisterEventHandler(eventName, func(event fsclient.CallEvent) {
			if v, ok := svc.activeCalls.Load(event.CallID); ok {
				ac := v.(*activeCall)
				select {
				case ac.eventCh <- event:
				default:
				}
			}
		})
	}
}

func pollCallStatus(t *testing.T, svc *Service, ctx context.Context, callID string, maxRetries int) *CallStatusResponse {
	t.Helper()
	for i := 0; i < maxRetries; i++ {
		resp, err := svc.GetCallStatus(ctx, callID)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Status == "finished" || resp.Status == "failed" {
			return resp
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("call did not reach terminal status")
	return nil
}

func TestFullCallbackFlow(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	setMockFSClient(t, svc, fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 30 * time.Second,
	})

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "initiating" {
		t.Errorf("expected initiating, got %s", resp.Status)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 20)
	if status.Status != "finished" {
		t.Errorf("expected finished, got %s", status.Status)
	}
}

func TestALegReject(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	setMockFSClient(t, svc, fsclient.MockConfig{ALegResult: "reject"})

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 20)
	if status.Status != "failed" {
		t.Errorf("expected failed, got %s", status.Status)
	}
	if status.Wastage.Type != nil {
		t.Errorf("expected no wastage, got %v", *status.Wastage.Type)
	}
}

func TestALegError(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	setMockFSClient(t, svc, fsclient.MockConfig{ALegResult: "error"})

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 20)
	if status.Status != "failed" {
		t.Errorf("expected failed, got %s", status.Status)
	}
}

func TestAConnectedBFailed(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	setMockFSClient(t, svc, fsclient.MockConfig{
		ALegResult: "answer",
		BLegResult: "reject",
	})

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 20)
	if status.Status != "failed" {
		t.Errorf("expected failed, got %s", status.Status)
	}
	if status.Wastage.Type == nil || *status.Wastage.Type != "a_connected_b_failed" {
		t.Errorf("expected a_connected_b_failed wastage, got %v", status.Wastage.Type)
	}
}

func TestInitiateValidation_MissingANumber(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	ctx := withAuth(context.Background(), 1, "client")

	_, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		BNumber: "13900139001",
	})
	if err == nil {
		t.Fatal("expected validation error for missing a_number")
	}
}

func TestInitiateValidation_CustomDataTooLarge(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	ctx := withAuth(context.Background(), 1, "client")

	largeData := make(json.RawMessage, 2048)
	for i := range largeData {
		largeData[i] = 'a'
	}

	_, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber:    "13800138001",
		BNumber:    "13900139001",
		CustomData: largeData,
	})
	if err == nil {
		t.Fatal("expected validation error for large custom_data")
	}
}
