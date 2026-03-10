package routing

import (
	"context"
	"errors"
	"math"
	"net"
	"slices"
	"testing"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
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

// --- DID tests ---

func clearDIDs(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := db.Exec(ctx, `DELETE FROM did_numbers`)
	if err != nil {
		t.Fatalf("failed to clear did_numbers: %v", err)
	}
}

func seedDID(t *testing.T, ctx context.Context, number string, userID *int64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO did_numbers (number, user_id, status)
		VALUES ($1, $2, 'available')
		RETURNING id
	`, number, userID).Scan(&id)
	if err != nil {
		t.Fatalf("failed to seed DID %s: %v", number, err)
	}
	return id
}

func adminCtx() context.Context {
	return auth.WithContext(context.Background(), "admin-1", &authpkg.AuthData{
		UserID:   1,
		Role:     "admin",
		Username: "admin",
	})
}

func TestSelectDIDUserPool(t *testing.T) {
	ctx := context.Background()
	clearDIDs(t, ctx)

	userID := int64(42)
	seedDID(t, ctx, "+18001111111", &userID)
	seedDID(t, ctx, "+18001111112", &userID)

	// Also seed a public DID
	seedDID(t, ctx, "+18009999999", nil)

	svc := &Service{}
	resp, err := svc.SelectDID(ctx, &SelectDIDParams{UserID: 42})
	if err != nil {
		t.Fatalf("SelectDID: %v", err)
	}

	// Should select from user's pool
	if resp.Number != "+18001111111" && resp.Number != "+18001111112" {
		t.Errorf("expected user DID, got %s", resp.Number)
	}
}

func TestSelectDIDPublicFallback(t *testing.T) {
	ctx := context.Background()
	clearDIDs(t, ctx)

	// Only seed public DIDs (no user assignment)
	seedDID(t, ctx, "+18005550001", nil)

	svc := &Service{}
	resp, err := svc.SelectDID(ctx, &SelectDIDParams{UserID: 99})
	if err != nil {
		t.Fatalf("SelectDID: %v", err)
	}

	if resp.Number != "+18005550001" {
		t.Errorf("expected public DID +18005550001, got %s", resp.Number)
	}
}

func TestSelectDIDNoneAvailable(t *testing.T) {
	ctx := context.Background()
	clearDIDs(t, ctx)

	svc := &Service{}
	_, err := svc.SelectDID(ctx, &SelectDIDParams{UserID: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var errResp *errs.Error
	if !errors.As(err, &errResp) {
		t.Fatalf("expected *errs.Error, got %T", err)
	}
	if errResp.Code != errs.ResourceExhausted {
		t.Errorf("expected ResourceExhausted, got %v", errResp.Code)
	}
}

func TestImportDIDs(t *testing.T) {
	ctx := adminCtx()
	clearDIDs(t, ctx)

	resp, err := ImportDIDs(ctx, &ImportDIDsParams{
		Numbers: []string{"+18001110001", "+18001110002", "+18001110003"},
	})
	if err != nil {
		t.Fatalf("ImportDIDs: %v", err)
	}
	if resp.Imported != 3 {
		t.Errorf("expected 3 imported, got %d", resp.Imported)
	}
	if resp.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", resp.Skipped)
	}

	// Import again with overlap
	resp, err = ImportDIDs(ctx, &ImportDIDsParams{
		Numbers: []string{"+18001110002", "+18001110003", "+18001110004"},
	})
	if err != nil {
		t.Fatalf("ImportDIDs second call: %v", err)
	}
	if resp.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", resp.Imported)
	}
	if resp.Skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", resp.Skipped)
	}
}

func TestAssignUnassignDID(t *testing.T) {
	ctx := adminCtx()
	clearDIDs(t, ctx)

	// Import a DID
	_, err := ImportDIDs(ctx, &ImportDIDsParams{
		Numbers: []string{"+18007770001"},
	})
	if err != nil {
		t.Fatalf("ImportDIDs: %v", err)
	}

	// List to get the ID
	listResp, err := ListDIDs(ctx, &ListDIDsParams{Page: 1})
	if err != nil {
		t.Fatalf("ListDIDs: %v", err)
	}
	if len(listResp.DIDs) != 1 {
		t.Fatalf("expected 1 DID, got %d", len(listResp.DIDs))
	}
	didID := listResp.DIDs[0].ID

	// Assign to user 42
	assignResp, err := AssignDID(ctx, didID, &AssignDIDParams{UserID: 42})
	if err != nil {
		t.Fatalf("AssignDID: %v", err)
	}
	if assignResp.UserID != 42 {
		t.Errorf("expected user_id 42, got %d", assignResp.UserID)
	}

	// Verify user pool membership via SelectDID
	selectResp, err := SelectDID(context.Background(), &SelectDIDParams{UserID: 42})
	if err != nil {
		t.Fatalf("SelectDID after assign: %v", err)
	}
	if selectResp.Number != "+18007770001" {
		t.Errorf("expected +18007770001, got %s", selectResp.Number)
	}

	// Unassign
	_, err = UnassignDID(ctx, didID)
	if err != nil {
		t.Fatalf("UnassignDID: %v", err)
	}

	// Verify no longer in user pool (should fallback to public pool)
	selectResp, err = SelectDID(context.Background(), &SelectDIDParams{UserID: 42})
	if err != nil {
		t.Fatalf("SelectDID after unassign: %v", err)
	}
	// It's now in public pool, so still accessible but via public fallback
	if selectResp.Number != "+18007770001" {
		t.Errorf("expected +18007770001 from public pool, got %s", selectResp.Number)
	}
}

// --- Health check tests ---

func seedGatewayFull(t *testing.T, ctx context.Context, name, gwType, sipAddr string, healthy, enabled bool, failures, maxFailures int) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO gateways (name, type, sip_address, weight, healthy, enabled, health_check_failures, max_health_failures)
		VALUES ($1, $2, $3, 1, $4, $5, $6, $7)
		RETURNING id
	`, name, gwType, sipAddr, healthy, enabled, failures, maxFailures).Scan(&id)
	if err != nil {
		t.Fatalf("failed to seed gateway %s: %v", name, err)
	}
	return id
}

func getGatewayHealth(t *testing.T, ctx context.Context, id int64) (bool, int) {
	t.Helper()
	var healthy bool
	var failures int
	err := db.QueryRow(ctx, `SELECT healthy, health_check_failures FROM gateways WHERE id = $1`, id).Scan(&healthy, &failures)
	if err != nil {
		t.Fatalf("failed to get gateway health: %v", err)
	}
	return healthy, failures
}

func TestHealthCheckMarksUnhealthy(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	// Use an unreachable address so TCP check fails
	gwID := seedGatewayFull(t, ctx, "HC-Fail", "a_leg", "sip:192.0.2.1:5060", true, true, 0, 3)

	svc := &Service{}
	if err := svc.loadALegGateways(ctx); err != nil {
		t.Fatalf("loadALegGateways: %v", err)
	}

	// Run health check 3 times to accumulate failures
	for i := range 3 {
		resp, err := svc.RunHealthCheck(ctx)
		if err != nil {
			t.Fatalf("RunHealthCheck iteration %d: %v", i, err)
		}
		if len(resp.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(resp.Results))
		}
	}

	healthy, failures := getGatewayHealth(t, ctx, gwID)
	if healthy {
		t.Error("expected gateway to be unhealthy after 3 failures")
	}
	if failures != 3 {
		t.Errorf("expected 3 failures, got %d", failures)
	}

	// Verify in-memory state updated
	svc.mu.Lock()
	for _, gw := range svc.aLegGateways {
		if gw.ID == gwID && gw.Healthy {
			t.Error("expected in-memory gateway to be unhealthy")
		}
	}
	svc.mu.Unlock()
}

func TestHealthCheckRecovery(t *testing.T) {
	ctx := context.Background()
	clearGateways(t, ctx)

	// Start with an unhealthy gateway that has accumulated failures
	// Use localhost with a listener to simulate a reachable host
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer ln.Close()

	gwID := seedGatewayFull(t, ctx, "HC-Recover", "a_leg", ln.Addr().String(), false, true, 5, 3)

	svc := &Service{}
	if err := svc.loadALegGateways(ctx); err != nil {
		t.Fatalf("loadALegGateways: %v", err)
	}

	// Run health check — should succeed since we have a listener
	resp, err := svc.RunHealthCheck(ctx)
	if err != nil {
		t.Fatalf("RunHealthCheck: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if !resp.Results[0].Healthy {
		t.Error("expected result to show healthy")
	}
	if resp.Results[0].Failures != 0 {
		t.Errorf("expected 0 failures, got %d", resp.Results[0].Failures)
	}

	// Verify DB state
	healthy, failures := getGatewayHealth(t, ctx, gwID)
	if !healthy {
		t.Error("expected gateway to be healthy in DB")
	}
	if failures != 0 {
		t.Errorf("expected 0 failures in DB, got %d", failures)
	}

	// Verify in-memory state updated
	svc.mu.Lock()
	for _, gw := range svc.aLegGateways {
		if gw.ID == gwID && !gw.Healthy {
			t.Error("expected in-memory gateway to be healthy")
		}
	}
	svc.mu.Unlock()
}
