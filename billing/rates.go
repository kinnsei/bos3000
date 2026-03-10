package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"

	authpkg "encore.app/auth"
	"encore.app/pkg/errcode"
	"encore.app/pkg/types"
)

func requireAdmin() error {
	data := authpkg.Data()
	if data == nil || data.Role != "admin" {
		return &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}
	return nil
}

// --- Rate Plan CRUD ---

type CreateRatePlanParams struct {
	Name         string       `json:"name"`
	Mode         string       `json:"mode"`
	UniformARate *types.Money `json:"uniform_a_rate,omitempty"`
	UniformBRate *types.Money `json:"uniform_b_rate,omitempty"`
	Description  string       `json:"description,omitempty"`
}

func (p *CreateRatePlanParams) Validate() error {
	if p.Name == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "name is required"}
	}
	if p.Mode != "uniform" && p.Mode != "prefix" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "mode must be 'uniform' or 'prefix'"}
	}
	if p.Mode == "uniform" && (p.UniformARate == nil || p.UniformBRate == nil) {
		return &errs.Error{Code: errs.InvalidArgument, Message: "uniform rates required for uniform mode"}
	}
	return nil
}

type RatePlanResponse struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Mode         string       `json:"mode"`
	UniformARate *types.Money `json:"uniform_a_rate,omitempty"`
	UniformBRate *types.Money `json:"uniform_b_rate,omitempty"`
	Description  string       `json:"description,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func scanRatePlan(scanner interface{ Scan(...any) error }) (*RatePlanResponse, error) {
	var resp RatePlanResponse
	var ua, ub *int64
	err := scanner.Scan(&resp.ID, &resp.Name, &resp.Mode, &ua, &ub,
		&resp.Description, &resp.CreatedAt, &resp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if ua != nil {
		v := types.Money(*ua)
		resp.UniformARate = &v
	}
	if ub != nil {
		v := types.Money(*ub)
		resp.UniformBRate = &v
	}
	return &resp, nil
}

// CreateRatePlan creates a new rate plan (admin only).
//
//encore:api auth method=POST path=/billing/rate-plans
func (s *Service) CreateRatePlan(ctx context.Context, p *CreateRatePlanParams) (*RatePlanResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	var aRate, bRate *int64
	if p.UniformARate != nil {
		v := int64(*p.UniformARate)
		aRate = &v
	}
	if p.UniformBRate != nil {
		v := int64(*p.UniformBRate)
		bRate = &v
	}

	resp, err := scanRatePlan(db.QueryRow(ctx, `
		INSERT INTO rate_plans (name, mode, uniform_a_rate, uniform_b_rate, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, mode, uniform_a_rate, uniform_b_rate, description, created_at, updated_at
	`, p.Name, p.Mode, aRate, bRate, p.Description))
	if err != nil {
		return nil, fmt.Errorf("insert rate plan: %w", err)
	}
	return resp, nil
}

type UpdateRatePlanParams struct {
	Name         string       `json:"name"`
	Mode         string       `json:"mode"`
	UniformARate *types.Money `json:"uniform_a_rate,omitempty"`
	UniformBRate *types.Money `json:"uniform_b_rate,omitempty"`
	Description  string       `json:"description,omitempty"`
}

// UpdateRatePlan updates an existing rate plan (admin only).
//
//encore:api auth method=PUT path=/billing/rate-plans/:id
func (s *Service) UpdateRatePlan(ctx context.Context, id int64, p *UpdateRatePlanParams) (*RatePlanResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	var aRate, bRate *int64
	if p.UniformARate != nil {
		v := int64(*p.UniformARate)
		aRate = &v
	}
	if p.UniformBRate != nil {
		v := int64(*p.UniformBRate)
		bRate = &v
	}

	resp, err := scanRatePlan(db.QueryRow(ctx, `
		UPDATE rate_plans SET name=$1, mode=$2, uniform_a_rate=$3, uniform_b_rate=$4, description=$5, updated_at=NOW()
		WHERE id=$6
		RETURNING id, name, mode, uniform_a_rate, uniform_b_rate, description, created_at, updated_at
	`, p.Name, p.Mode, aRate, bRate, p.Description, id))
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "rate plan not found"}
		}
		return nil, fmt.Errorf("update rate plan: %w", err)
	}
	return resp, nil
}

type ListRatePlansResponse struct {
	Plans []RatePlanResponse `json:"plans"`
}

// ListRatePlans lists all rate plans (admin only).
//
//encore:api auth method=GET path=/billing/rate-plans
func (s *Service) ListRatePlans(ctx context.Context) (*ListRatePlansResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx, `
		SELECT id, name, mode, uniform_a_rate, uniform_b_rate, description, created_at, updated_at
		FROM rate_plans ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query rate plans: %w", err)
	}
	defer rows.Close()

	var plans []RatePlanResponse
	for rows.Next() {
		p, err := scanRatePlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rate plan: %w", err)
		}
		plans = append(plans, *p)
	}
	return &ListRatePlansResponse{Plans: plans}, nil
}

// GetRatePlan gets a single rate plan by ID (admin only).
//
//encore:api auth method=GET path=/billing/rate-plans/:id
func (s *Service) GetRatePlan(ctx context.Context, id int64) (*RatePlanResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	resp, err := scanRatePlan(db.QueryRow(ctx, `
		SELECT id, name, mode, uniform_a_rate, uniform_b_rate, description, created_at, updated_at
		FROM rate_plans WHERE id=$1
	`, id))
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "rate plan not found"}
		}
		return nil, fmt.Errorf("get rate plan: %w", err)
	}
	return resp, nil
}

// --- Prefix Rates ---

type AddPrefixRateParams struct {
	Prefix string      `json:"prefix"`
	ARate  types.Money `json:"a_rate"`
	BRate  types.Money `json:"b_rate"`
}

type PrefixRateResponse struct {
	ID         int64       `json:"id"`
	RatePlanID int64       `json:"rate_plan_id"`
	Prefix     string      `json:"prefix"`
	ARate      types.Money `json:"a_rate"`
	BRate      types.Money `json:"b_rate"`
}

// AddPrefixRate adds a prefix rate entry to a rate plan (admin only).
//
//encore:api auth method=POST path=/billing/rate-plans/:id/prefixes
func (s *Service) AddPrefixRate(ctx context.Context, id int64, p *AddPrefixRateParams) (*PrefixRateResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	if p.Prefix == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "prefix is required"}
	}

	var resp PrefixRateResponse
	err := db.QueryRow(ctx, `
		INSERT INTO rate_plan_prefixes (rate_plan_id, prefix, a_rate, b_rate)
		VALUES ($1, $2, $3, $4)
		RETURNING id, rate_plan_id, prefix, a_rate, b_rate
	`, id, p.Prefix, int64(p.ARate), int64(p.BRate)).Scan(
		&resp.ID, &resp.RatePlanID, &resp.Prefix, &resp.ARate, &resp.BRate,
	)
	if err != nil {
		return nil, fmt.Errorf("insert prefix rate: %w", err)
	}
	return &resp, nil
}

// RemovePrefixRate removes a prefix rate from a rate plan (admin only).
//
//encore:api auth method=DELETE path=/billing/rate-plans/:id/prefixes/:prefix
func (s *Service) RemovePrefixRate(ctx context.Context, id int64, prefix string) error {
	if err := requireAdmin(); err != nil {
		return err
	}

	result, err := db.Exec(ctx, `
		DELETE FROM rate_plan_prefixes WHERE rate_plan_id=$1 AND prefix=$2
	`, id, prefix)
	if err != nil {
		return fmt.Errorf("delete prefix rate: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "prefix rate not found"}
	}
	return nil
}

// --- Rate Resolution ---

type ResolveRateParams struct {
	UserID       int64  `json:"user_id"`
	CalledPrefix string `json:"called_prefix"`
}

type ResolveRateResponse struct {
	ALegRate types.Money `json:"a_leg_rate"`
	BLegRate types.Money `json:"b_leg_rate"`
	Source   string      `json:"source"`
}

// ResolveRate resolves billing rates for a user/prefix with priority: user > plan_uniform > plan_prefix.
//
//encore:api private method=POST path=/billing/resolve-rate
func (s *Service) ResolveRate(ctx context.Context, p *ResolveRateParams) (*ResolveRateResponse, error) {
	var userARate, userBRate *int64
	var ratePlanID *int64
	err := db.QueryRow(ctx, `
		SELECT a_leg_rate, b_leg_rate, rate_plan_id FROM billing_accounts WHERE user_id=$1
	`, p.UserID).Scan(&userARate, &userBRate, &ratePlanID)
	if err != nil {
		return nil, fmt.Errorf("select billing account: %w", err)
	}

	// Step 1: User-level rates (highest priority)
	if userARate != nil && userBRate != nil {
		return &ResolveRateResponse{
			ALegRate: types.Money(*userARate),
			BLegRate: types.Money(*userBRate),
			Source:   "user",
		}, nil
	}

	// Step 2: Rate plan
	if ratePlanID != nil {
		var mode string
		var uniformA, uniformB *int64
		err := db.QueryRow(ctx, `
			SELECT mode, uniform_a_rate, uniform_b_rate FROM rate_plans WHERE id=$1
		`, *ratePlanID).Scan(&mode, &uniformA, &uniformB)
		if err != nil {
			return nil, fmt.Errorf("select rate plan: %w", err)
		}

		if mode == "uniform" && uniformA != nil && uniformB != nil {
			return &ResolveRateResponse{
				ALegRate: types.Money(*uniformA),
				BLegRate: types.Money(*uniformB),
				Source:   "plan_uniform",
			}, nil
		}

		if mode == "prefix" {
			var aRate, bRate int64
			err := db.QueryRow(ctx, `
				SELECT a_rate, b_rate FROM rate_plan_prefixes
				WHERE rate_plan_id=$1 AND prefix=$2
			`, *ratePlanID, p.CalledPrefix).Scan(&aRate, &bRate)
			if err == nil {
				return &ResolveRateResponse{
					ALegRate: types.Money(aRate),
					BLegRate: types.Money(bRate),
					Source:   "plan_prefix",
				}, nil
			}
		}
	}

	// Step 3: No rate found
	return nil, errcode.NewError(errs.NotFound, errcode.PrefixNotFound,
		"no rate configured for this user/prefix")
}

// --- User Rate Config ---

type SetUserRateConfigParams struct {
	RatePlanID *int64       `json:"rate_plan_id,omitempty"`
	ALegRate   *types.Money `json:"a_leg_rate,omitempty"`
	BLegRate   *types.Money `json:"b_leg_rate,omitempty"`
}

type SetUserRateConfigResponse struct {
	Success bool `json:"success"`
}

// SetUserRateConfig updates a user's rate configuration (admin only).
//
//encore:api auth method=PUT path=/billing/accounts/:userId/rate-config
func (s *Service) SetUserRateConfig(ctx context.Context, userId int64, p *SetUserRateConfigParams) (*SetUserRateConfigResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	var aRate, bRate *int64
	if p.ALegRate != nil {
		v := int64(*p.ALegRate)
		aRate = &v
	}
	if p.BLegRate != nil {
		v := int64(*p.BLegRate)
		bRate = &v
	}

	result, err := db.Exec(ctx, `
		UPDATE billing_accounts SET rate_plan_id=$1, a_leg_rate=$2, b_leg_rate=$3, updated_at=NOW()
		WHERE user_id=$4
	`, p.RatePlanID, aRate, bRate, userId)
	if err != nil {
		return nil, fmt.Errorf("update rate config: %w", err)
	}
	if result.RowsAffected() == 0 {
		return nil, &errs.Error{Code: errs.NotFound, Message: "billing account not found"}
	}
	return &SetUserRateConfigResponse{Success: true}, nil
}
