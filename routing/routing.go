package routing

import (
	"context"
	"sync"

	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
)

var db = sqldb.NewDatabase("routing", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

var billingDB = sqldb.Named("billing")

// WeightedGateway holds in-memory state for smooth weighted round-robin.
type WeightedGateway struct {
	ID            int64
	Name          string
	SIPAddress    string
	Weight        int
	CurrentWeight int
	Healthy       bool
	Enabled       bool
}

//encore:service
type Service struct {
	mu           sync.Mutex
	aLegGateways []*WeightedGateway
}

func initService() (*Service, error) {
	svc := &Service{}
	if err := svc.loadALegGateways(context.Background()); err != nil {
		return nil, err
	}
	svc.validatePrefixConsistency(context.Background())
	return svc, nil
}

func (s *Service) loadALegGateways(ctx context.Context) error {
	rows, err := db.Query(ctx, `
		SELECT id, name, sip_address, weight, healthy, enabled
		FROM gateways
		WHERE type = 'a_leg'
		ORDER BY id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var gateways []*WeightedGateway
	for rows.Next() {
		gw := &WeightedGateway{}
		if err := rows.Scan(&gw.ID, &gw.Name, &gw.SIPAddress, &gw.Weight, &gw.Healthy, &gw.Enabled); err != nil {
			return err
		}
		gateways = append(gateways, gw)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.aLegGateways = gateways
	s.mu.Unlock()
	return nil
}

func (s *Service) validatePrefixConsistency(ctx context.Context) {
	// Collect B-leg gateway prefixes from routing DB
	gwPrefixes := make(map[string]bool)
	rows, err := db.Query(ctx, `
		SELECT DISTINCT gp.prefix
		FROM gateway_prefixes gp
		JOIN gateways g ON g.id = gp.gateway_id
		WHERE g.type = 'b_leg'
	`)
	if err != nil {
		rlog.Warn("prefix consistency check skipped: routing DB query failed", "error", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rlog.Warn("prefix consistency check skipped: scan error", "error", err)
			return
		}
		gwPrefixes[p] = true
	}
	if err := rows.Err(); err != nil {
		rlog.Warn("prefix consistency check skipped: rows error", "error", err)
		return
	}

	// Read rate plan prefixes from billing DB
	rpPrefixes := make(map[string]bool)
	rpRows, err := billingDB.Query(ctx, `SELECT DISTINCT prefix FROM rate_plan_prefixes`)
	if err != nil {
		rlog.Warn("prefix consistency check skipped: billing DB not accessible", "error", err)
		return
	}
	defer rpRows.Close()
	for rpRows.Next() {
		var p string
		if err := rpRows.Scan(&p); err != nil {
			rlog.Warn("prefix consistency check skipped: billing scan error", "error", err)
			return
		}
		rpPrefixes[p] = true
	}
	if err := rpRows.Err(); err != nil {
		rlog.Warn("prefix consistency check skipped: billing rows error", "error", err)
		return
	}

	// Compare prefix sets
	for p := range gwPrefixes {
		if !rpPrefixes[p] {
			rlog.Warn("prefix in gateway but not in rate plans", "prefix", p)
		}
	}
	for p := range rpPrefixes {
		if !gwPrefixes[p] {
			rlog.Warn("prefix in rate plans but not in gateway", "prefix", p)
		}
	}
}
