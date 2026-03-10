package routing

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"

	"encore.dev/beta/errs"
)

func seedGateways(t *testing.T, ctx context.Context, gateways []struct {
	Name       string
	SIPAddress string
	Weight     int
	Healthy    bool
	Enabled    bool
}) {
	t.Helper()
	for _, gw := range gateways {
		_, err := db.Exec(ctx, `
			INSERT INTO gateways (name, type, sip_address, weight, healthy, enabled)
			VALUES ($1, 'a_leg', $2, $3, $4, $5)
		`, gw.Name, gw.SIPAddress, gw.Weight, gw.Healthy, gw.Enabled)
		if err != nil {
			t.Fatalf("failed to seed gateway %s: %v", gw.Name, err)
		}
	}
}

func clearGateways(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := db.Exec(ctx, `DELETE FROM gateway_prefixes`)
	if err != nil {
		t.Fatalf("failed to clear gateway_prefixes: %v", err)
	}
	_, err = db.Exec(ctx, `DELETE FROM gateways`)
	if err != nil {
		t.Fatalf("failed to clear gateways: %v", err)
	}
}

func TestPickALegWeightedDistribution(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	seedGateways(t, ctx, []struct {
		Name       string
		SIPAddress string
		Weight     int
		Healthy    bool
		Enabled    bool
	}{
		{"GW-A", "sip:a@example.com", 5, true, true},
		{"GW-B", "sip:b@example.com", 3, true, true},
		{"GW-C", "sip:c@example.com", 2, true, true},
	})

	svc := &Service{}
	if err := svc.loadALegGateways(ctx); err != nil {
		t.Fatalf("loadALegGateways: %v", err)
	}

	counts := map[string]int{}
	iterations := 1000
	for i := range iterations {
		resp, err := svc.PickALeg(ctx)
		if err != nil {
			t.Fatalf("PickALeg iteration %d: %v", i, err)
		}
		counts[resp.Name]++
	}

	// Expected ratios: A=50%, B=30%, C=20%
	totalWeight := 10.0
	expected := map[string]float64{
		"GW-A": 5.0 / totalWeight,
		"GW-B": 3.0 / totalWeight,
		"GW-C": 2.0 / totalWeight,
	}

	for name, ratio := range expected {
		actual := float64(counts[name]) / float64(iterations)
		if math.Abs(actual-ratio) > 0.05 {
			t.Errorf("gateway %s: expected ratio ~%.2f, got %.2f (count=%d)", name, ratio, actual, counts[name])
		}
	}
}

func TestPickALegSkipsUnhealthy(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	seedGateways(t, ctx, []struct {
		Name       string
		SIPAddress string
		Weight     int
		Healthy    bool
		Enabled    bool
	}{
		{"GW-Healthy", "sip:healthy@example.com", 1, true, true},
		{"GW-Unhealthy", "sip:unhealthy@example.com", 1, false, true},
		{"GW-Disabled", "sip:disabled@example.com", 1, true, false},
	})

	svc := &Service{}
	if err := svc.loadALegGateways(ctx); err != nil {
		t.Fatalf("loadALegGateways: %v", err)
	}

	for i := range 100 {
		resp, err := svc.PickALeg(ctx)
		if err != nil {
			t.Fatalf("PickALeg iteration %d: %v", i, err)
		}
		if resp.Name != "GW-Healthy" {
			t.Fatalf("expected GW-Healthy, got %s", resp.Name)
		}
	}
}

func TestPickALegNoHealthy(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	seedGateways(t, ctx, []struct {
		Name       string
		SIPAddress string
		Weight     int
		Healthy    bool
		Enabled    bool
	}{
		{"GW-Down1", "sip:down1@example.com", 1, false, true},
		{"GW-Down2", "sip:down2@example.com", 1, true, false},
	})

	svc := &Service{}
	if err := svc.loadALegGateways(ctx); err != nil {
		t.Fatalf("loadALegGateways: %v", err)
	}

	_, err := svc.PickALeg(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var errResp *errs.Error
	if !errors.As(err, &errResp) {
		t.Fatalf("expected *errs.Error, got %T", err)
	}
	if errResp.Code != errs.Unavailable {
		t.Errorf("expected Unavailable, got %v", errResp.Code)
	}
}

// --- B-leg tests ---

func seedBLegGateway(t *testing.T, ctx context.Context, name, sipAddr string, healthy bool, failoverID *int64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO gateways (name, type, sip_address, weight, healthy, enabled, failover_gateway_id)
		VALUES ($1, 'b_leg', $2, 1, $3, true, $4)
		RETURNING id
	`, name, sipAddr, healthy, failoverID).Scan(&id)
	if err != nil {
		t.Fatalf("failed to seed b-leg gateway %s: %v", name, err)
	}
	return id
}

func seedPrefix(t *testing.T, ctx context.Context, gatewayID int64, prefix string, priority int) {
	t.Helper()
	_, err := db.Exec(ctx, `
		INSERT INTO gateway_prefixes (gateway_id, prefix, priority)
		VALUES ($1, $2, $3)
	`, gatewayID, prefix, priority)
	if err != nil {
		t.Fatalf("failed to seed prefix %s for gateway %d: %v", prefix, gatewayID, err)
	}
}

func TestPickBLegPrefixMatch(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	gwID := seedBLegGateway(t, ctx, "BLeg-138", "sip:138@carrier.com", true, nil)
	seedPrefix(t, ctx, gwID, "138", 1)

	svc := &Service{}
	resp, err := svc.PickBLeg(ctx, &PickBLegParams{CalledNumber: "13812345678"})
	if err != nil {
		t.Fatalf("PickBLeg: %v", err)
	}
	if resp.GatewayID != gwID {
		t.Errorf("expected gateway %d, got %d", gwID, resp.GatewayID)
	}
	if resp.Name != "BLeg-138" {
		t.Errorf("expected name BLeg-138, got %s", resp.Name)
	}
}

func TestPickBLegFailover(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	// Create failover gateway first (healthy)
	failoverID := seedBLegGateway(t, ctx, "BLeg-Failover", "sip:failover@carrier.com", true, nil)

	// Create primary gateway (unhealthy) with failover
	primaryID := seedBLegGateway(t, ctx, "BLeg-Primary", "sip:primary@carrier.com", false, &failoverID)
	seedPrefix(t, ctx, primaryID, "139", 1)

	svc := &Service{}
	resp, err := svc.PickBLeg(ctx, &PickBLegParams{CalledNumber: "13912345678"})
	if err != nil {
		t.Fatalf("PickBLeg: %v", err)
	}
	if resp.GatewayID != failoverID {
		t.Errorf("expected failover gateway %d, got %d", failoverID, resp.GatewayID)
	}
	if resp.Name != "BLeg-Failover" {
		t.Errorf("expected BLeg-Failover, got %s", resp.Name)
	}
}

func TestPickBLegUnknownPrefix(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	svc := &Service{}
	_, err := svc.PickBLeg(ctx, &PickBLegParams{CalledNumber: "99912345678"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var errResp *errs.Error
	if !errors.As(err, &errResp) {
		t.Fatalf("expected *errs.Error, got %T", err)
	}
	if errResp.Code != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errResp.Code)
	}
}

func TestPrefixConsistencyWarning(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	// Setup a B-leg gateway with prefix "138" but no matching rate plan prefix
	gwID := seedBLegGateway(t, ctx, "BLeg-Mismatch", "sip:mismatch@carrier.com", true, nil)
	seedPrefix(t, ctx, gwID, "138", 1)

	svc := &Service{}
	gwOnly, _ := svc.GetPrefixMismatches(ctx)

	// "138" should be in gateway-only list since no rate plan has it
	if !slices.Contains(gwOnly, "138") {
		t.Errorf("expected prefix '138' in gateway-only mismatches, got %v", gwOnly)
	}
}
