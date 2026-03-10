package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
	"encore.app/pkg/types"
)

func createTestAccount(t *testing.T, ctx context.Context, balance, creditLimit int64, maxConcurrent int) int64 {
	t.Helper()
	userID := time.Now().UnixNano()
	_, err := db.Exec(ctx, `
		INSERT INTO billing_accounts (user_id, balance, credit_limit, max_concurrent)
		VALUES ($1, $2, $3, $4)
	`, userID, balance, creditLimit, maxConcurrent)
	if err != nil {
		t.Fatalf("failed to create test account: %v", err)
	}
	return userID
}

func getBalance(t *testing.T, ctx context.Context, userID int64) int64 {
	t.Helper()
	var balance int64
	err := db.QueryRow(ctx, `SELECT balance FROM billing_accounts WHERE user_id = $1`, userID).Scan(&balance)
	if err != nil {
		t.Fatalf("failed to get balance: %v", err)
	}
	return balance
}

func TestPreDeductSufficientBalance(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 100000, 0, 10) // balance=100000 (1000.00 in cents)

	resp, err := PreDeduct(ctx, &PreDeductParams{
		UserID:   userID,
		CallID:   fmt.Sprintf("call-%d", userID),
		ALegRate: 100, // 1.00 per minute
		BLegRate: 200, // 2.00 per minute
	})
	if err != nil {
		t.Fatalf("PreDeduct failed: %v", err)
	}

	// Pre-deduction = (100 + 200) * 30 = 9000
	expectedAmount := int64(9000)
	if int64(resp.Amount) != expectedAmount {
		t.Errorf("expected amount %d, got %d", expectedAmount, resp.Amount)
	}
	if resp.TxID == 0 {
		t.Error("expected non-zero TxID")
	}

	// Balance should be 100000 - 9000 = 91000
	balance := getBalance(t, ctx, userID)
	if balance != 91000 {
		t.Errorf("expected balance 91000, got %d", balance)
	}
}

func TestPreDeductInsufficientBalance(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 100, 0, 10) // very low balance

	_, err := PreDeduct(ctx, &PreDeductParams{
		UserID:   userID,
		CallID:   fmt.Sprintf("call-%d", userID),
		ALegRate: 100,
		BLegRate: 200,
	})
	if err == nil {
		t.Fatal("expected error for insufficient balance, got nil")
	}

	// Balance should be unchanged
	balance := getBalance(t, ctx, userID)
	if balance != 100 {
		t.Errorf("expected balance unchanged at 100, got %d", balance)
	}
}

func TestPreDeductConcurrentSerializedByRowLock(t *testing.T) {
	ctx := context.Background()
	// Enough for 2 pre-deductions but not 3
	// Each pre-deduction = (100+200)*30 = 9000, so 20000 allows exactly 2
	userID := createTestAccount(t, ctx, 20000, 0, 10)

	var wg sync.WaitGroup
	results := make(chan error, 3)

	for i := range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := PreDeduct(ctx, &PreDeductParams{
				UserID:   userID,
				CallID:   fmt.Sprintf("call-%d-%d", userID, i),
				ALegRate: 100,
				BLegRate: 200,
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var successes, failures int
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	if successes != 2 {
		t.Errorf("expected 2 successes, got %d", successes)
	}
	if failures != 1 {
		t.Errorf("expected 1 failure, got %d", failures)
	}
}

func TestAcquireSlotUnderLimit(t *testing.T) {
	ctx := context.Background()
	userID := time.Now().UnixNano()

	resp, err := AcquireSlot(ctx, &AcquireSlotParams{
		UserID:        userID,
		MaxConcurrent: 5,
	})
	if err != nil {
		t.Fatalf("AcquireSlot failed: %v", err)
	}
	if resp.CurrentSlots != 1 {
		t.Errorf("expected 1 slot, got %d", resp.CurrentSlots)
	}

	// Acquire a second slot
	resp, err = AcquireSlot(ctx, &AcquireSlotParams{
		UserID:        userID,
		MaxConcurrent: 5,
	})
	if err != nil {
		t.Fatalf("AcquireSlot second call failed: %v", err)
	}
	if resp.CurrentSlots != 2 {
		t.Errorf("expected 2 slots, got %d", resp.CurrentSlots)
	}
}

func TestAcquireSlotExceedsLimit(t *testing.T) {
	ctx := context.Background()
	userID := time.Now().UnixNano()

	// Fill up to max (2)
	for i := range 2 {
		_, err := AcquireSlot(ctx, &AcquireSlotParams{
			UserID:        userID,
			MaxConcurrent: 2,
		})
		if err != nil {
			t.Fatalf("AcquireSlot %d failed: %v", i, err)
		}
	}

	// Third should fail
	_, err := AcquireSlot(ctx, &AcquireSlotParams{
		UserID:        userID,
		MaxConcurrent: 2,
	})
	if err == nil {
		t.Fatal("expected error for exceeded concurrency limit, got nil")
	}
}

func TestReleaseSlot(t *testing.T) {
	ctx := context.Background()
	userID := time.Now().UnixNano()

	// Acquire 2 slots
	for range 2 {
		_, err := AcquireSlot(ctx, &AcquireSlotParams{
			UserID:        userID,
			MaxConcurrent: 5,
		})
		if err != nil {
			t.Fatalf("AcquireSlot failed: %v", err)
		}
	}

	// Release one
	resp, err := ReleaseSlot(ctx, &ReleaseSlotParams{UserID: userID})
	if err != nil {
		t.Fatalf("ReleaseSlot failed: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}

	// Should be able to acquire up to max again
	// We had 2, released 1 = 1 remaining, so with max=2 we can acquire 1 more
	acqResp, err := AcquireSlot(ctx, &AcquireSlotParams{
		UserID:        userID,
		MaxConcurrent: 2,
	})
	if err != nil {
		t.Fatalf("AcquireSlot after release failed: %v", err)
	}
	if acqResp.CurrentSlots != 2 {
		t.Errorf("expected 2 slots after release+acquire, got %d", acqResp.CurrentSlots)
	}
}

// --- Finalize Tests ---

func TestFinalizeWithRefund(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 100000, 0, 10)

	// Pre-deduct first
	_, err := PreDeduct(ctx, &PreDeductParams{
		UserID: userID, CallID: fmt.Sprintf("fin-refund-%d", userID),
		ALegRate: 100, BLegRate: 200,
	})
	if err != nil {
		t.Fatalf("PreDeduct failed: %v", err)
	}
	// Pre-deduction = (100+200)*30 = 9000, balance = 91000

	// Finalize with short call (actual < pre-deducted)
	resp, err := Finalize(ctx, &FinalizeParams{
		UserID: userID, CallID: fmt.Sprintf("fin-refund-%d", userID),
		ALegDurationSec: 10, BLegDurationSec: 30,
		ALegRate: 100, BLegRate: 200,
		PreDeductAmount: 9000,
	})
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	// A-leg: ceil(10/6)=2 blocks, cost = 2*6*100/60 = 20
	// B-leg: ceil(30/60)=1 block, cost = 1*60*200/60 = 200
	// Total = 220, refund = 9000-220 = 8780
	if resp.ALegCost != 20 {
		t.Errorf("expected a_leg_cost 20, got %d", resp.ALegCost)
	}
	if resp.BLegCost != 200 {
		t.Errorf("expected b_leg_cost 200, got %d", resp.BLegCost)
	}
	if resp.Refund != 8780 {
		t.Errorf("expected refund 8780, got %d", resp.Refund)
	}

	balance := getBalance(t, ctx, userID)
	// 91000 + 8780 = 99780
	if balance != 99780 {
		t.Errorf("expected balance 99780, got %d", balance)
	}
}

func TestFinalizeZeroDuration(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 50000, 0, 10)

	_, err := PreDeduct(ctx, &PreDeductParams{
		UserID: userID, CallID: fmt.Sprintf("fin-zero-%d", userID),
		ALegRate: 100, BLegRate: 200,
	})
	if err != nil {
		t.Fatalf("PreDeduct failed: %v", err)
	}
	// balance = 50000 - 9000 = 41000

	resp, err := Finalize(ctx, &FinalizeParams{
		UserID: userID, CallID: fmt.Sprintf("fin-zero-%d", userID),
		ALegDurationSec: 0, BLegDurationSec: 0,
		ALegRate: 100, BLegRate: 200,
		PreDeductAmount: 9000,
	})
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	if resp.TotalCost != 0 {
		t.Errorf("expected total cost 0, got %d", resp.TotalCost)
	}
	if resp.Refund != 9000 {
		t.Errorf("expected full refund 9000, got %d", resp.Refund)
	}

	balance := getBalance(t, ctx, userID)
	if balance != 50000 {
		t.Errorf("expected balance restored to 50000, got %d", balance)
	}
}

func TestFinalizeBlockRounding(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 100000, 0, 10)

	// Test exact 6s and 60s boundaries
	resp, err := Finalize(ctx, &FinalizeParams{
		UserID: userID, CallID: fmt.Sprintf("fin-round-%d", userID),
		ALegDurationSec: 6, BLegDurationSec: 60,
		ALegRate: 600, BLegRate: 600,
		PreDeductAmount: 100000,
	})
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	// A-leg: ceil(6/6)=1 block, cost = 1*6*600/60 = 60
	// B-leg: ceil(60/60)=1 block, cost = 1*60*600/60 = 600
	if resp.ALegCost != 60 {
		t.Errorf("expected a_leg_cost 60 for 6s at 600/min, got %d", resp.ALegCost)
	}
	if resp.BLegCost != 600 {
		t.Errorf("expected b_leg_cost 600 for 60s at 600/min, got %d", resp.BLegCost)
	}

	// Test non-boundary: 7s should round up to 2 blocks of 6s
	userID2 := createTestAccount(t, ctx, 100000, 0, 10)
	resp2, err := Finalize(ctx, &FinalizeParams{
		UserID: userID2, CallID: fmt.Sprintf("fin-round2-%d", userID2),
		ALegDurationSec: 7, BLegDurationSec: 61,
		ALegRate: 600, BLegRate: 600,
		PreDeductAmount: 100000,
	})
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	// A-leg: ceil(7/6)=2 blocks, cost = 2*6*600/60 = 120
	// B-leg: ceil(61/60)=2 blocks, cost = 2*60*600/60 = 1200
	if resp2.ALegCost != 120 {
		t.Errorf("expected a_leg_cost 120 for 7s at 600/min, got %d", resp2.ALegCost)
	}
	if resp2.BLegCost != 1200 {
		t.Errorf("expected b_leg_cost 1200 for 61s at 600/min, got %d", resp2.BLegCost)
	}
}

// --- Rate Plan Tests ---

func adminCtx(t *testing.T) context.Context {
	t.Helper()
	return auth.WithContext(context.Background(), auth.UID("1"), &authpkg.AuthData{
		UserID: 1, Role: "admin", Username: "admin",
	})
}

func clientCtx(t *testing.T) context.Context {
	t.Helper()
	return auth.WithContext(context.Background(), auth.UID("2"), &authpkg.AuthData{
		UserID: 2, Role: "client", Username: "client",
	})
}

func moneyPtr(v types.Money) *types.Money { return &v }

func TestCreateRatePlanUniform(t *testing.T) {
	ctx := adminCtx(t)
	resp, err := CreateRatePlan(ctx, &CreateRatePlanParams{
		Name:         fmt.Sprintf("uniform-%d", time.Now().UnixNano()),
		Mode:         "uniform",
		UniformARate: moneyPtr(100),
		UniformBRate: moneyPtr(200),
		Description:  "test uniform plan",
	})
	if err != nil {
		t.Fatalf("CreateRatePlan failed: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Mode != "uniform" {
		t.Errorf("expected mode uniform, got %s", resp.Mode)
	}
	if resp.UniformARate == nil || *resp.UniformARate != 100 {
		t.Errorf("expected uniform_a_rate 100, got %v", resp.UniformARate)
	}
	if resp.UniformBRate == nil || *resp.UniformBRate != 200 {
		t.Errorf("expected uniform_b_rate 200, got %v", resp.UniformBRate)
	}
}

func TestCreateRatePlanPrefix(t *testing.T) {
	ctx := adminCtx(t)
	plan, err := CreateRatePlan(ctx, &CreateRatePlanParams{
		Name: fmt.Sprintf("prefix-%d", time.Now().UnixNano()),
		Mode: "prefix",
	})
	if err != nil {
		t.Fatalf("CreateRatePlan failed: %v", err)
	}
	if plan.Mode != "prefix" {
		t.Errorf("expected mode prefix, got %s", plan.Mode)
	}

	// Add a prefix rate
	pfx, err := AddPrefixRate(ctx, plan.ID, &AddPrefixRateParams{
		Prefix: "+86", ARate: 150, BRate: 250,
	})
	if err != nil {
		t.Fatalf("AddPrefixRate failed: %v", err)
	}
	if pfx.Prefix != "+86" {
		t.Errorf("expected prefix +86, got %s", pfx.Prefix)
	}
	if pfx.ARate != 150 || pfx.BRate != 250 {
		t.Errorf("expected rates 150/250, got %d/%d", pfx.ARate, pfx.BRate)
	}
}

func TestResolveRateUserLevelPriority(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 100000, 0, 10)

	// Set user-level rates directly
	_, err := db.Exec(ctx, `
		UPDATE billing_accounts SET a_leg_rate=500, b_leg_rate=600 WHERE user_id=$1
	`, userID)
	if err != nil {
		t.Fatalf("set user rates: %v", err)
	}

	resp, err := ResolveRate(ctx, &ResolveRateParams{
		UserID: userID, CalledPrefix: "+86",
	})
	if err != nil {
		t.Fatalf("ResolveRate failed: %v", err)
	}
	if resp.Source != "user" {
		t.Errorf("expected source 'user', got %s", resp.Source)
	}
	if resp.ALegRate != 500 || resp.BLegRate != 600 {
		t.Errorf("expected rates 500/600, got %d/%d", resp.ALegRate, resp.BLegRate)
	}
}

func TestResolveRateUniformPlan(t *testing.T) {
	ctx := context.Background()
	aCtx := adminCtx(t)

	// Create uniform plan
	plan, err := CreateRatePlan(aCtx, &CreateRatePlanParams{
		Name:         fmt.Sprintf("resolve-uniform-%d", time.Now().UnixNano()),
		Mode:         "uniform",
		UniformARate: moneyPtr(300),
		UniformBRate: moneyPtr(400),
	})
	if err != nil {
		t.Fatalf("CreateRatePlan failed: %v", err)
	}

	// Create account with plan, no user-level rates
	userID := createTestAccount(t, ctx, 100000, 0, 10)
	_, err = db.Exec(ctx, `
		UPDATE billing_accounts SET rate_plan_id=$1 WHERE user_id=$2
	`, plan.ID, userID)
	if err != nil {
		t.Fatalf("set rate plan: %v", err)
	}

	resp, err := ResolveRate(ctx, &ResolveRateParams{
		UserID: userID, CalledPrefix: "+1",
	})
	if err != nil {
		t.Fatalf("ResolveRate failed: %v", err)
	}
	if resp.Source != "plan_uniform" {
		t.Errorf("expected source 'plan_uniform', got %s", resp.Source)
	}
	if resp.ALegRate != 300 || resp.BLegRate != 400 {
		t.Errorf("expected rates 300/400, got %d/%d", resp.ALegRate, resp.BLegRate)
	}
}

func TestResolveRatePrefixPlan(t *testing.T) {
	ctx := context.Background()
	aCtx := adminCtx(t)

	// Create prefix plan with a specific prefix
	plan, err := CreateRatePlan(aCtx, &CreateRatePlanParams{
		Name: fmt.Sprintf("resolve-prefix-%d", time.Now().UnixNano()),
		Mode: "prefix",
	})
	if err != nil {
		t.Fatalf("CreateRatePlan failed: %v", err)
	}
	_, err = AddPrefixRate(aCtx, plan.ID, &AddPrefixRateParams{
		Prefix: "+44", ARate: 700, BRate: 800,
	})
	if err != nil {
		t.Fatalf("AddPrefixRate failed: %v", err)
	}

	userID := createTestAccount(t, ctx, 100000, 0, 10)
	_, err = db.Exec(ctx, `
		UPDATE billing_accounts SET rate_plan_id=$1 WHERE user_id=$2
	`, plan.ID, userID)
	if err != nil {
		t.Fatalf("set rate plan: %v", err)
	}

	resp, err := ResolveRate(ctx, &ResolveRateParams{
		UserID: userID, CalledPrefix: "+44",
	})
	if err != nil {
		t.Fatalf("ResolveRate failed: %v", err)
	}
	if resp.Source != "plan_prefix" {
		t.Errorf("expected source 'plan_prefix', got %s", resp.Source)
	}
	if resp.ALegRate != 700 || resp.BLegRate != 800 {
		t.Errorf("expected rates 700/800, got %d/%d", resp.ALegRate, resp.BLegRate)
	}
}

func TestResolveRateNoRateFound(t *testing.T) {
	ctx := context.Background()
	userID := createTestAccount(t, ctx, 100000, 0, 10)

	_, err := ResolveRate(ctx, &ResolveRateParams{
		UserID: userID, CalledPrefix: "+999",
	})
	if err == nil {
		t.Fatal("expected error for no rate found, got nil")
	}

	var errResp *errs.Error
	if !errors.As(err, &errResp) {
		t.Fatalf("expected *errs.Error, got %T", err)
	}
	if errResp.Code != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errResp.Code)
	}
}

func TestAdminOnlyRatePlanAccess(t *testing.T) {
	ctx := clientCtx(t)

	_, err := CreateRatePlan(ctx, &CreateRatePlanParams{
		Name:         "should-fail",
		Mode:         "uniform",
		UniformARate: moneyPtr(100),
		UniformBRate: moneyPtr(200),
	})
	if err == nil {
		t.Fatal("expected PermissionDenied for non-admin, got nil")
	}

	var errResp *errs.Error
	if !errors.As(err, &errResp) {
		t.Fatalf("expected *errs.Error, got %T", err)
	}
	if errResp.Code != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errResp.Code)
	}
}
