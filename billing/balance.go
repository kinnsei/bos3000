package billing

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"

	authpkg "encore.app/auth"
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

// --- Admin Billing Operations ---

// TopupParams contains the parameters for an admin balance topup.
type TopupParams struct {
	Amount      types.Money `json:"amount"`
	Description string      `json:"description"`
}

func (p *TopupParams) Validate() error {
	if p.Amount <= 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "amount must be positive"}
	}
	return nil
}

// TopupResponse contains the result of a topup.
type TopupResponse struct {
	Balance types.Money `json:"balance"`
	TxID    int64       `json:"tx_id"`
}

// Topup adds balance to a user's account (admin only).
//
//encore:api auth method=POST path=/billing/accounts/:userId/topup
func (s *Service) Topup(ctx context.Context, userId int64, p *TopupParams) (*TopupResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int64
	err = tx.QueryRow(ctx, `
		SELECT balance FROM billing_accounts WHERE user_id = $1 FOR UPDATE
	`, userId).Scan(&balance)
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "billing account not found"}
		}
		return nil, fmt.Errorf("select account: %w", err)
	}

	newBalance := balance + int64(p.Amount)
	_, err = tx.Exec(ctx, `
		UPDATE billing_accounts SET balance = $1, updated_at = NOW() WHERE user_id = $2
	`, newBalance, userId)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	var txID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, amount, balance_after, description)
		VALUES ($1, 'topup', $2, $3, $4)
		RETURNING id
	`, userId, int64(p.Amount), newBalance, p.Description).Scan(&txID)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &TopupResponse{Balance: types.Money(newBalance), TxID: txID}, nil
}

// DeductParams contains the parameters for an admin balance deduction.
type DeductParams struct {
	Amount      types.Money `json:"amount"`
	Description string      `json:"description"`
}

func (p *DeductParams) Validate() error {
	if p.Amount <= 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "amount must be positive"}
	}
	return nil
}

// DeductResponse contains the result of a deduction.
type DeductResponse struct {
	Balance types.Money `json:"balance"`
	TxID    int64       `json:"tx_id"`
}

// Deduct subtracts balance from a user's account (admin only).
//
//encore:api auth method=POST path=/billing/accounts/:userId/deduct
func (s *Service) Deduct(ctx context.Context, userId int64, p *DeductParams) (*DeductResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int64
	err = tx.QueryRow(ctx, `
		SELECT balance FROM billing_accounts WHERE user_id = $1 FOR UPDATE
	`, userId).Scan(&balance)
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "billing account not found"}
		}
		return nil, fmt.Errorf("select account: %w", err)
	}

	newBalance := balance - int64(p.Amount)
	_, err = tx.Exec(ctx, `
		UPDATE billing_accounts SET balance = $1, updated_at = NOW() WHERE user_id = $2
	`, newBalance, userId)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	var txID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, amount, balance_after, description)
		VALUES ($1, 'deduction', $2, $3, $4)
		RETURNING id
	`, userId, int64(p.Amount), newBalance, p.Description).Scan(&txID)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &DeductResponse{Balance: types.Money(newBalance), TxID: txID}, nil
}

// --- Account Access ---

// AccountResponse contains billing account details.
type AccountResponse struct {
	UserID        int64       `json:"user_id"`
	Balance       types.Money `json:"balance"`
	CreditLimit   types.Money `json:"credit_limit"`
	MaxConcurrent int         `json:"max_concurrent"`
	RatePlanID    *int64      `json:"rate_plan_id,omitempty"`
	Status        string      `json:"status"`
}

// GetAccount retrieves a billing account. Admin: any user. Client: own account only.
//
//encore:api auth method=GET path=/billing/accounts/:userId
func (s *Service) GetAccount(ctx context.Context, userId int64) (*AccountResponse, error) {
	data := authpkg.Data()
	if data == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	// Client can only view own account
	if data.Role != "admin" && data.UserID != userId {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "access denied"}
	}

	var resp AccountResponse
	err := db.QueryRow(ctx, `
		SELECT user_id, balance, credit_limit, max_concurrent, rate_plan_id, status
		FROM billing_accounts WHERE user_id = $1
	`, userId).Scan(&resp.UserID, &resp.Balance, &resp.CreditLimit,
		&resp.MaxConcurrent, &resp.RatePlanID, &resp.Status)
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "billing account not found"}
		}
		return nil, fmt.Errorf("select account: %w", err)
	}

	return &resp, nil
}

// --- Transaction Listing ---

// ListTransactionsParams contains query parameters for listing transactions.
type ListTransactionsParams struct {
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
	Type     string `query:"type"`
	DateFrom string `query:"date_from"`
	DateTo   string `query:"date_to"`
}

// TransactionEntry represents a single transaction.
type TransactionEntry struct {
	ID           int64       `json:"id"`
	UserID       int64       `json:"user_id"`
	Type         string      `json:"type"`
	Amount       types.Money `json:"amount"`
	BalanceAfter types.Money `json:"balance_after"`
	ReferenceID  *string     `json:"reference_id,omitempty"`
	Description  *string     `json:"description,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

// ListTransactionsResponse contains paginated transactions.
type ListTransactionsResponse struct {
	Transactions []TransactionEntry `json:"transactions"`
	Page         int                `json:"page"`
	PageSize     int                `json:"page_size"`
	Total        int                `json:"total"`
}

// ListTransactions returns paginated transactions. Admin: any user. Client: own only.
//
//encore:api auth method=GET path=/billing/accounts/:userId/transactions
func (s *Service) ListTransactions(ctx context.Context, userId int64, p *ListTransactionsParams) (*ListTransactionsResponse, error) {
	data := authpkg.Data()
	if data == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	if data.Role != "admin" && data.UserID != userId {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "access denied"}
	}

	page := max(p.Page, 1)
	pageSize := p.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build dynamic query with filters
	where := "WHERE user_id = $1"
	args := []any{userId}
	argIdx := 2

	if p.Type != "" {
		where += " AND type = $" + strconv.Itoa(argIdx)
		args = append(args, p.Type)
		argIdx++
	}
	if p.DateFrom != "" {
		where += " AND created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, p.DateFrom)
		argIdx++
	}
	if p.DateTo != "" {
		where += " AND created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, p.DateTo)
		argIdx++
	}

	// Count total
	var total int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM transactions "+where, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count transactions: %w", err)
	}

	// Fetch page
	limitArgs := append(args, pageSize, offset)
	rows, err := db.Query(ctx, `
		SELECT id, user_id, type, amount, balance_after, reference_id, description, created_at
		FROM transactions `+where+`
		ORDER BY created_at DESC
		LIMIT $`+strconv.Itoa(argIdx)+` OFFSET $`+strconv.Itoa(argIdx+1),
		limitArgs...)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var txns []TransactionEntry
	for rows.Next() {
		var t TransactionEntry
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter,
			&t.ReferenceID, &t.Description, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}

	return &ListTransactionsResponse{
		Transactions: txns,
		Page:         page,
		PageSize:     pageSize,
		Total:        total,
	}, nil
}

// --- Account Creation ---

// CreateAccountParams contains the parameters for creating a billing account.
type CreateAccountParams struct {
	UserID        int64       `json:"user_id"`
	CreditLimit   types.Money `json:"credit_limit"`
	MaxConcurrent int         `json:"max_concurrent"`
	RatePlanID    *int64      `json:"rate_plan_id,omitempty"`
}

func (p *CreateAccountParams) Validate() error {
	if p.UserID <= 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "user_id is required"}
	}
	if p.MaxConcurrent < 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "max_concurrent must be non-negative"}
	}
	return nil
}

// CreateAccountResponse contains the created billing account.
type CreateAccountResponse struct {
	UserID        int64       `json:"user_id"`
	Balance       types.Money `json:"balance"`
	CreditLimit   types.Money `json:"credit_limit"`
	MaxConcurrent int         `json:"max_concurrent"`
	RatePlanID    *int64      `json:"rate_plan_id,omitempty"`
	Status        string      `json:"status"`
}

// CreateAccount creates a billing account for a user (admin only).
//
//encore:api auth method=POST path=/billing/accounts
func (s *Service) CreateAccount(ctx context.Context, p *CreateAccountParams) (*CreateAccountResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	var resp CreateAccountResponse
	err := db.QueryRow(ctx, `
		INSERT INTO billing_accounts (user_id, balance, credit_limit, max_concurrent, rate_plan_id)
		VALUES ($1, 0, $2, $3, $4)
		RETURNING user_id, balance, credit_limit, max_concurrent, rate_plan_id, status
	`, p.UserID, int64(p.CreditLimit), p.MaxConcurrent, p.RatePlanID).Scan(
		&resp.UserID, &resp.Balance, &resp.CreditLimit,
		&resp.MaxConcurrent, &resp.RatePlanID, &resp.Status)
	if err != nil {
		return nil, fmt.Errorf("insert billing account: %w", err)
	}

	return &resp, nil
}

