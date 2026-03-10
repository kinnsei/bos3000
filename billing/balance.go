package billing

import (
	"context"
	"fmt"

	"encore.dev/beta/errs"

	"encore.app/pkg/errcode"
	"encore.app/pkg/types"
)

// PreDeductParams contains the parameters for pre-deducting balance before a call.
type PreDeductParams struct {
	UserID   int64       `json:"user_id"`
	CallID   string      `json:"call_id"`
	ALegRate types.Money `json:"a_leg_rate"`
	BLegRate types.Money `json:"b_leg_rate"`
}

// PreDeductResponse contains the pre-deduction result.
type PreDeductResponse struct {
	Amount types.Money `json:"amount"`
	TxID   int64       `json:"tx_id"`
}

// PreDeduct atomically pre-deducts balance for a call (30 min at combined rate).
//
//encore:api private method=POST path=/billing/pre-deduct
func (s *Service) PreDeduct(ctx context.Context, p *PreDeductParams) (*PreDeductResponse, error) {
	// 30 min at combined per-minute rate
	preDeductAmount := types.Money((int64(p.ALegRate) + int64(p.BLegRate)) * 30)

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance, creditLimit int64
	err = tx.QueryRow(ctx, `
		SELECT balance, credit_limit FROM billing_accounts
		WHERE user_id = $1 FOR UPDATE
	`, p.UserID).Scan(&balance, &creditLimit)
	if err != nil {
		return nil, fmt.Errorf("select account: %w", err)
	}

	if balance+creditLimit < int64(preDeductAmount) {
		return nil, errcode.NewError(errs.FailedPrecondition, errcode.InsufficientBalance,
			"insufficient balance for pre-deduction")
	}

	newBalance := balance - int64(preDeductAmount)
	_, err = tx.Exec(ctx, `
		UPDATE billing_accounts SET balance = $1, updated_at = NOW()
		WHERE user_id = $2
	`, newBalance, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	var txID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, amount, balance_after, reference_id, description)
		VALUES ($1, 'pre_deduct', $2, $3, $4, $5)
		RETURNING id
	`, p.UserID, int64(preDeductAmount), newBalance, p.CallID, "Pre-deduction for call "+p.CallID).Scan(&txID)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &PreDeductResponse{
		Amount: preDeductAmount,
		TxID:   txID,
	}, nil
}

// AcquireSlotParams contains the parameters for acquiring a concurrent call slot.
type AcquireSlotParams struct {
	UserID        int64 `json:"user_id"`
	MaxConcurrent int   `json:"max_concurrent"`
}

// AcquireSlotResponse contains the current slot count after acquisition.
type AcquireSlotResponse struct {
	CurrentSlots int `json:"current_slots"`
}

// AcquireSlot increments the concurrent call counter and checks against the limit.
//
//encore:api private method=POST path=/billing/acquire-slot
func (s *Service) AcquireSlot(ctx context.Context, p *AcquireSlotParams) (*AcquireSlotResponse, error) {
	newVal, err := concurrentSlots.Increment(ctx, p.UserID, 1)
	if err != nil {
		return nil, fmt.Errorf("increment slot: %w", err)
	}

	if int(newVal) > p.MaxConcurrent {
		// Roll back the increment
		if _, decErr := concurrentSlots.Increment(ctx, p.UserID, -1); decErr != nil {
			return nil, fmt.Errorf("rollback slot increment: %w", decErr)
		}
		return nil, errcode.NewError(errs.ResourceExhausted, errcode.ConcurrencyLimitExceeded,
			"concurrent call limit exceeded")
	}

	return &AcquireSlotResponse{CurrentSlots: int(newVal)}, nil
}

// ReleaseSlotParams contains the parameters for releasing a concurrent call slot.
type ReleaseSlotParams struct {
	UserID int64 `json:"user_id"`
}

// ReleaseSlotResponse confirms successful slot release.
type ReleaseSlotResponse struct {
	Success bool `json:"success"`
}

// ReleaseSlot decrements the concurrent call counter, flooring at 0.
//
//encore:api private method=POST path=/billing/release-slot
func (s *Service) ReleaseSlot(ctx context.Context, p *ReleaseSlotParams) (*ReleaseSlotResponse, error) {
	val, err := concurrentSlots.Increment(ctx, p.UserID, -1)
	if err != nil {
		// Key might not exist, that's fine
		return &ReleaseSlotResponse{Success: true}, nil
	}

	// Floor at 0
	if val < 0 {
		if setErr := concurrentSlots.Set(ctx, p.UserID, 0); setErr != nil {
			return nil, fmt.Errorf("floor slot to 0: %w", setErr)
		}
	}

	return &ReleaseSlotResponse{Success: true}, nil
}

// FinalizeParams contains the parameters for finalizing a call's billing.
type FinalizeParams struct {
	UserID           int64       `json:"user_id"`
	CallID           string      `json:"call_id"`
	ALegDurationSec  int64       `json:"a_leg_duration_sec"`
	BLegDurationSec  int64       `json:"b_leg_duration_sec"`
	ALegRate         types.Money `json:"a_leg_rate"`
	BLegRate         types.Money `json:"b_leg_rate"`
	PreDeductAmount  types.Money `json:"pre_deduct_amount"`
}

// FinalizeResponse contains the final billing breakdown.
type FinalizeResponse struct {
	ALegCost  types.Money `json:"a_leg_cost"`
	BLegCost  types.Money `json:"b_leg_cost"`
	TotalCost types.Money `json:"total_cost"`
	Refund    types.Money `json:"refund"`
}

// Finalize calculates actual call cost and reconciles with pre-deduction.
// A-leg: 6-second blocks. B-leg: 60-second blocks.
//
//encore:api private method=POST path=/billing/finalize
func (s *Service) Finalize(ctx context.Context, p *FinalizeParams) (*FinalizeResponse, error) {
	// A-leg: ceil to 6s blocks, per-minute rate
	aLegCost := types.Money(0)
	if p.ALegDurationSec > 0 {
		aLegBlocks := types.CeilDiv(p.ALegDurationSec, 6)
		aLegCost = types.Money(aLegBlocks * 6 * int64(p.ALegRate) / 60)
	}

	// B-leg: ceil to 60s blocks, per-minute rate
	bLegCost := types.Money(0)
	if p.BLegDurationSec > 0 {
		bLegBlocks := types.CeilDiv(p.BLegDurationSec, 60)
		bLegCost = types.Money(bLegBlocks * 60 * int64(p.BLegRate) / 60)
	}

	actualCost := aLegCost + bLegCost
	diff := p.PreDeductAmount - actualCost

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int64
	err = tx.QueryRow(ctx, `
		SELECT balance FROM billing_accounts
		WHERE user_id = $1 FOR UPDATE
	`, p.UserID).Scan(&balance)
	if err != nil {
		return nil, fmt.Errorf("select account: %w", err)
	}

	var refund types.Money
	var txType string
	var txAmount int64

	if diff > 0 {
		// Refund the difference
		refund = diff
		txType = "refund"
		txAmount = int64(diff)
		balance += int64(diff)
	} else if diff < 0 {
		// Charge additional
		txType = "finalize"
		txAmount = int64(-diff)
		balance -= int64(-diff)
	} else {
		txType = "finalize"
		txAmount = 0
	}

	_, err = tx.Exec(ctx, `
		UPDATE billing_accounts SET balance = $1, updated_at = NOW()
		WHERE user_id = $2
	`, balance, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (user_id, type, amount, balance_after, reference_id, description)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.UserID, txType, txAmount, balance, p.CallID,
		fmt.Sprintf("Finalize call %s: a_leg=%d b_leg=%d", p.CallID, int64(aLegCost), int64(bLegCost)))
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &FinalizeResponse{
		ALegCost:  aLegCost,
		BLegCost:  bLegCost,
		TotalCost: actualCost,
		Refund:    refund,
	}, nil
}
