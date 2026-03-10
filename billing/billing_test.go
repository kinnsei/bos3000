package billing

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
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
