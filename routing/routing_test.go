package routing

import (
	"context"
	"errors"
	"math"
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
