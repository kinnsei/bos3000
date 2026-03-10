package routing

import (
	"context"

	"encore.dev/beta/errs"

	"encore.app/pkg/errcode"
)

// --- PickBLeg: prefix matching with failover ---

type PickBLegParams struct {
	CalledNumber string `json:"called_number"`
}

type PickBLegResponse struct {
	GatewayID  int64  `json:"gateway_id"`
	Name       string `json:"name"`
	SIPAddress string `json:"sip_address"`
}

//encore:api private method=POST path=/routing/pick-b-leg
func (s *Service) PickBLeg(ctx context.Context, p *PickBLegParams) (*PickBLegResponse, error) {
	if len(p.CalledNumber) < 3 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "called_number must be at least 3 digits"}
	}
	prefix := p.CalledNumber[:3]

	// Find gateways matching the prefix, ordered by priority
	rows, err := db.Query(ctx, `
		SELECT g.id, g.name, g.sip_address, g.healthy, g.enabled, g.failover_gateway_id
		FROM gateways g
		JOIN gateway_prefixes gp ON g.id = gp.gateway_id
		WHERE gp.prefix = $1 AND g.enabled = true
		ORDER BY gp.priority ASC
	`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matched bool
	for rows.Next() {
		matched = true
		var id int64
		var name, sipAddr string
		var healthy, enabled bool
		var failoverID *int64
		if err := rows.Scan(&id, &name, &sipAddr, &healthy, &enabled, &failoverID); err != nil {
			return nil, err
		}
		if healthy {
			return &PickBLegResponse{GatewayID: id, Name: name, SIPAddress: sipAddr}, nil
		}
		// Try failover chain (max 3 hops)
		if resp := s.followFailover(ctx, failoverID, 3); resp != nil {
			return resp, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !matched {
		return nil, errcode.NewError(errs.NotFound, errcode.PrefixNotFound, "no gateway configured for prefix "+prefix)
	}
	return nil, errcode.NewError(errs.Unavailable, errcode.NoHealthyGateway, "all gateways for prefix "+prefix+" are unhealthy")
}

func (s *Service) followFailover(ctx context.Context, failoverID *int64, maxHops int) *PickBLegResponse {
	for hops := 0; hops < maxHops && failoverID != nil; hops++ {
		var id int64
		var name, sipAddr string
		var healthy, enabled bool
		var nextFailover *int64
		err := db.QueryRow(ctx, `
			SELECT id, name, sip_address, healthy, enabled, failover_gateway_id
			FROM gateways WHERE id = $1
		`, *failoverID).Scan(&id, &name, &sipAddr, &healthy, &enabled, &nextFailover)
		if err != nil {
			return nil
		}
		if healthy && enabled {
			return &PickBLegResponse{GatewayID: id, Name: name, SIPAddress: sipAddr}
		}
		failoverID = nextFailover
	}
	return nil
}

// --- Prefix CRUD ---

type AddPrefixParams struct {
	Prefix   string `json:"prefix"`
	Priority int    `json:"priority"`
}

type PrefixResponse struct {
	GatewayID int64  `json:"gateway_id"`
	Prefix    string `json:"prefix"`
	Priority  int    `json:"priority"`
}

//encore:api auth method=POST path=/routing/gateways/:id/prefixes
func (s *Service) AddPrefix(ctx context.Context, id int64, p *AddPrefixParams) (*PrefixResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}
	if p.Prefix == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "prefix is required"}
	}

	// Verify gateway exists
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM gateways WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &errs.Error{Code: errs.NotFound, Message: "gateway not found"}
	}

	_, err = db.Exec(ctx, `
		INSERT INTO gateway_prefixes (gateway_id, prefix, priority)
		VALUES ($1, $2, $3)
	`, id, p.Prefix, p.Priority)
	if err != nil {
		return nil, err
	}

	return &PrefixResponse{GatewayID: id, Prefix: p.Prefix, Priority: p.Priority}, nil
}

//encore:api auth method=DELETE path=/routing/gateways/:id/prefixes/:prefix
func (s *Service) RemovePrefix(ctx context.Context, id int64, prefix string) error {
	if err := requireAdmin(); err != nil {
		return err
	}

	result, err := db.Exec(ctx, `
		DELETE FROM gateway_prefixes WHERE gateway_id = $1 AND prefix = $2
	`, id, prefix)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "prefix not found"}
	}
	return nil
}

type ListPrefixesResponse struct {
	Prefixes []*PrefixResponse `json:"prefixes"`
}

//encore:api auth method=GET path=/routing/gateways/:id/prefixes
func (s *Service) ListPrefixes(ctx context.Context, id int64) (*ListPrefixesResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx, `
		SELECT gateway_id, prefix, priority
		FROM gateway_prefixes
		WHERE gateway_id = $1
		ORDER BY priority ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefixes []*PrefixResponse
	for rows.Next() {
		p := &PrefixResponse{}
		if err := rows.Scan(&p.GatewayID, &p.Prefix, &p.Priority); err != nil {
			return nil, err
		}
		prefixes = append(prefixes, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ListPrefixesResponse{Prefixes: prefixes}, nil
}

// GetPrefixMismatches returns mismatched prefixes between gateways and rate plans.
// Exported for testing the consistency check.
func (s *Service) GetPrefixMismatches(ctx context.Context) (gwOnly []string, rpOnly []string) {
	gwPrefixes := make(map[string]bool)
	rows, err := db.Query(ctx, `
		SELECT DISTINCT gp.prefix
		FROM gateway_prefixes gp
		JOIN gateways g ON g.id = gp.gateway_id
		WHERE g.type = 'b_leg'
	`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, nil
		}
		gwPrefixes[p] = true
	}

	rpPrefixes := make(map[string]bool)
	rpRows, err := billingDB.Query(ctx, `SELECT DISTINCT prefix FROM rate_plan_prefixes`)
	if err != nil {
		// billing DB not accessible — graceful handling
		return nil, nil
	}
	defer rpRows.Close()
	for rpRows.Next() {
		var p string
		if err := rpRows.Scan(&p); err != nil {
			return nil, nil
		}
		rpPrefixes[p] = true
	}

	for p := range gwPrefixes {
		if !rpPrefixes[p] {
			gwOnly = append(gwOnly, p)
		}
	}
	for p := range rpPrefixes {
		if !gwPrefixes[p] {
			rpOnly = append(rpOnly, p)
		}
	}
	return gwOnly, rpOnly
}

