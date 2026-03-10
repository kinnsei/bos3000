package routing

import (
	"context"
	"time"

	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
	"encore.app/pkg/errcode"
)

func requireAdmin() error {
	data := authpkg.Data()
	if data == nil || data.Role != "admin" {
		return &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}
	return nil
}

// --- PickALeg: smooth weighted round-robin ---

type PickALegResponse struct {
	GatewayID  int64  `json:"gateway_id"`
	Name       string `json:"name"`
	SIPAddress string `json:"sip_address"`
}

//encore:api private method=POST path=/routing/pick-a-leg
func (s *Service) PickALeg(ctx context.Context) (*PickALegResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Filter healthy + enabled gateways
	var candidates []*WeightedGateway
	totalWeight := 0
	for _, gw := range s.aLegGateways {
		if gw.Healthy && gw.Enabled {
			candidates = append(candidates, gw)
			totalWeight += gw.Weight
		}
	}

	if len(candidates) == 0 {
		return nil, errcode.NewError(errs.Unavailable, errcode.NoHealthyGateway, "no healthy A-leg gateways available")
	}

	// Smooth weighted round-robin (nginx algorithm)
	var best *WeightedGateway
	for _, gw := range candidates {
		gw.CurrentWeight += gw.Weight
		if best == nil || gw.CurrentWeight > best.CurrentWeight {
			best = gw
		}
	}
	best.CurrentWeight -= totalWeight

	return &PickALegResponse{
		GatewayID:  best.ID,
		Name:       best.Name,
		SIPAddress: best.SIPAddress,
	}, nil
}

// --- Gateway Admin CRUD ---

type CreateGatewayParams struct {
	Name              string  `json:"name"`
	Type              string  `json:"type"`
	SIPAddress        string  `json:"sip_address"`
	Weight            int     `json:"weight"`
	FailoverGatewayID *int64  `json:"failover_gateway_id,omitempty"`
	Carrier           *string `json:"carrier,omitempty"`
	MaxHealthFailures *int    `json:"max_health_failures,omitempty"`
}

type GatewayResponse struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	SIPAddress          string     `json:"sip_address"`
	Weight              int        `json:"weight"`
	Healthy             bool       `json:"healthy"`
	Enabled             bool       `json:"enabled"`
	FailoverGatewayID   *int64     `json:"failover_gateway_id,omitempty"`
	Carrier             *string    `json:"carrier,omitempty"`
	HealthCheckFailures int        `json:"health_check_failures"`
	MaxHealthFailures   int        `json:"max_health_failures"`
	LastHealthCheck     *time.Time `json:"last_health_check,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

//encore:api auth method=POST path=/routing/gateways
func (s *Service) CreateGateway(ctx context.Context, p *CreateGatewayParams) (*GatewayResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	if p.Type != "a_leg" && p.Type != "b_leg" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "type must be 'a_leg' or 'b_leg'"}
	}
	if p.Weight < 1 {
		p.Weight = 1
	}

	maxFail := 3
	if p.MaxHealthFailures != nil {
		maxFail = *p.MaxHealthFailures
	}

	var gw GatewayResponse
	err := db.QueryRow(ctx, `
		INSERT INTO gateways (name, type, sip_address, weight, failover_gateway_id, carrier, max_health_failures)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, type, sip_address, weight, healthy, enabled,
			failover_gateway_id, carrier, health_check_failures, max_health_failures,
			last_health_check, created_at, updated_at
	`, p.Name, p.Type, p.SIPAddress, p.Weight, p.FailoverGatewayID, p.Carrier, maxFail).Scan(
		&gw.ID, &gw.Name, &gw.Type, &gw.SIPAddress, &gw.Weight, &gw.Healthy, &gw.Enabled,
		&gw.FailoverGatewayID, &gw.Carrier, &gw.HealthCheckFailures, &gw.MaxHealthFailures,
		&gw.LastHealthCheck, &gw.CreatedAt, &gw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Reload A-leg gateways if we added one
	if p.Type == "a_leg" {
		if err := s.loadALegGateways(ctx); err != nil {
			return nil, err
		}
	}

	return &gw, nil
}

type UpdateGatewayParams struct {
	Name              *string `json:"name,omitempty"`
	SIPAddress        *string `json:"sip_address,omitempty"`
	Weight            *int    `json:"weight,omitempty"`
	FailoverGatewayID *int64  `json:"failover_gateway_id,omitempty"`
	Carrier           *string `json:"carrier,omitempty"`
	MaxHealthFailures *int    `json:"max_health_failures,omitempty"`
}

//encore:api auth method=PUT path=/routing/gateways/:id
func (s *Service) UpdateGateway(ctx context.Context, id int64, p *UpdateGatewayParams) (*GatewayResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	var gw GatewayResponse
	err := db.QueryRow(ctx, `
		UPDATE gateways SET
			name = COALESCE($2, name),
			sip_address = COALESCE($3, sip_address),
			weight = COALESCE($4, weight),
			failover_gateway_id = COALESCE($5, failover_gateway_id),
			carrier = COALESCE($6, carrier),
			max_health_failures = COALESCE($7, max_health_failures),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, type, sip_address, weight, healthy, enabled,
			failover_gateway_id, carrier, health_check_failures, max_health_failures,
			last_health_check, created_at, updated_at
	`, id, p.Name, p.SIPAddress, p.Weight, p.FailoverGatewayID, p.Carrier, p.MaxHealthFailures).Scan(
		&gw.ID, &gw.Name, &gw.Type, &gw.SIPAddress, &gw.Weight, &gw.Healthy, &gw.Enabled,
		&gw.FailoverGatewayID, &gw.Carrier, &gw.HealthCheckFailures, &gw.MaxHealthFailures,
		&gw.LastHealthCheck, &gw.CreatedAt, &gw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Reload if it could be an A-leg
	if gw.Type == "a_leg" {
		if err := s.loadALegGateways(ctx); err != nil {
			return nil, err
		}
	}

	return &gw, nil
}

type ListGatewaysParams struct {
	Type string `query:"type"`
}

type ListGatewaysResponse struct {
	Gateways []*GatewayResponse `json:"gateways"`
}

//encore:api auth method=GET path=/routing/gateways
func (s *Service) ListGateways(ctx context.Context, p *ListGatewaysParams) (*ListGatewaysResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, name, type, sip_address, weight, healthy, enabled,
			failover_gateway_id, carrier, health_check_failures, max_health_failures,
			last_health_check, created_at, updated_at
		FROM gateways
	`
	args := []any{}
	if p.Type != "" {
		query += " WHERE type = $1"
		args = append(args, p.Type)
	}
	query += " ORDER BY id"

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []*GatewayResponse
	for rows.Next() {
		gw := &GatewayResponse{}
		if err := rows.Scan(
			&gw.ID, &gw.Name, &gw.Type, &gw.SIPAddress, &gw.Weight, &gw.Healthy, &gw.Enabled,
			&gw.FailoverGatewayID, &gw.Carrier, &gw.HealthCheckFailures, &gw.MaxHealthFailures,
			&gw.LastHealthCheck, &gw.CreatedAt, &gw.UpdatedAt,
		); err != nil {
			return nil, err
		}
		gateways = append(gateways, gw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ListGatewaysResponse{Gateways: gateways}, nil
}

type ToggleGatewayParams struct {
	Enabled bool `json:"enabled"`
}

type ToggleGatewayResponse struct {
	ID      int64 `json:"id"`
	Enabled bool  `json:"enabled"`
}

//encore:api auth method=POST path=/routing/gateways/:id/toggle
func (s *Service) ToggleGateway(ctx context.Context, id int64, p *ToggleGatewayParams) (*ToggleGatewayResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	var gwType string
	var enabled bool
	err := db.QueryRow(ctx, `
		UPDATE gateways SET enabled = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING type, enabled
	`, id, p.Enabled).Scan(&gwType, &enabled)
	if err != nil {
		return nil, err
	}

	if gwType == "a_leg" {
		if err := s.loadALegGateways(ctx); err != nil {
			return nil, err
		}
	}

	return &ToggleGatewayResponse{ID: id, Enabled: enabled}, nil
}
