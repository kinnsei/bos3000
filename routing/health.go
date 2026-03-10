package routing

import (
	"context"
	"net"
	"strings"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/cron"
	"encore.dev/rlog"

	authpkg "encore.app/auth"
)

var _ = cron.NewJob("gateway-health-check", cron.JobConfig{
	Title:    "Check gateway health",
	Every:    1 * cron.Minute,
	Endpoint: RunHealthCheck,
})

type HealthCheckResult struct {
	GatewayID  int64  `json:"gateway_id"`
	Name       string `json:"name"`
	SIPAddress string `json:"sip_address"`
	Healthy    bool   `json:"healthy"`
	Failures   int    `json:"failures"`
}

type RunHealthCheckResponse struct {
	Results []HealthCheckResult `json:"results"`
}

//encore:api private method=POST path=/routing/health-check
func (s *Service) RunHealthCheck(ctx context.Context) (*RunHealthCheckResponse, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, type, sip_address, health_check_failures, max_health_failures
		FROM gateways
		WHERE enabled = true
		ORDER BY id
	`)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to query gateways")
	}
	defer rows.Close()

	type gwInfo struct {
		ID          int64
		Name        string
		Type        string
		SIPAddress  string
		Failures    int
		MaxFailures int
	}
	var gateways []gwInfo
	for rows.Next() {
		var gw gwInfo
		if err := rows.Scan(&gw.ID, &gw.Name, &gw.Type, &gw.SIPAddress, &gw.Failures, &gw.MaxFailures); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to scan gateway")
		}
		gateways = append(gateways, gw)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "rows iteration error")
	}

	var results []HealthCheckResult
	now := time.Now()

	for _, gw := range gateways {
		reachable := checkTCP(gw.SIPAddress)

		var newFailures int
		var healthy bool
		if reachable {
			newFailures = 0
			healthy = true
		} else {
			newFailures = gw.Failures + 1
			healthy = newFailures < gw.MaxFailures
		}

		_, err := db.Exec(ctx, `
			UPDATE gateways
			SET healthy = $1, health_check_failures = $2, last_health_check = $3, updated_at = $3
			WHERE id = $4
		`, healthy, newFailures, now, gw.ID)
		if err != nil {
			rlog.Warn("failed to update gateway health", "gateway_id", gw.ID, "error", err)
			continue
		}

		// Log status changes
		wasHealthy := gw.Failures < gw.MaxFailures
		if wasHealthy && !healthy {
			rlog.Warn("gateway marked unhealthy", "gateway_id", gw.ID, "name", gw.Name, "failures", newFailures)
		} else if !wasHealthy && healthy {
			rlog.Info("gateway recovered", "gateway_id", gw.ID, "name", gw.Name)
		}

		results = append(results, HealthCheckResult{
			GatewayID:  gw.ID,
			Name:       gw.Name,
			SIPAddress: gw.SIPAddress,
			Healthy:    healthy,
			Failures:   newFailures,
		})

		// Update in-memory A-leg state
		if gw.Type == "a_leg" {
			s.mu.Lock()
			for _, wg := range s.aLegGateways {
				if wg.ID == gw.ID {
					wg.Healthy = healthy
					break
				}
			}
			s.mu.Unlock()
		}
	}

	return &RunHealthCheckResponse{Results: results}, nil
}

type ManualHealthCheckResponse struct {
	HealthCheckResult
}

//encore:api auth method=POST path=/routing/gateways/:id/health
func (s *Service) ManualHealthCheck(ctx context.Context, id int64) (*ManualHealthCheckResponse, error) {
	ad := authpkg.Data()
	if ad == nil || ad.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin only"}
	}

	var gw struct {
		Name        string
		Type        string
		SIPAddress  string
		Failures    int
		MaxFailures int
	}
	err := db.QueryRow(ctx, `
		SELECT name, type, sip_address, health_check_failures, max_health_failures
		FROM gateways WHERE id = $1
	`, id).Scan(&gw.Name, &gw.Type, &gw.SIPAddress, &gw.Failures, &gw.MaxFailures)
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "gateway not found"}
	}

	reachable := checkTCP(gw.SIPAddress)

	var newFailures int
	var healthy bool
	if reachable {
		newFailures = 0
		healthy = true
	} else {
		newFailures = gw.Failures + 1
		healthy = newFailures < gw.MaxFailures
	}

	now := time.Now()
	_, err = db.Exec(ctx, `
		UPDATE gateways
		SET healthy = $1, health_check_failures = $2, last_health_check = $3, updated_at = $3
		WHERE id = $4
	`, healthy, newFailures, now, id)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update gateway")
	}

	// Update in-memory A-leg state
	if gw.Type == "a_leg" {
		s.mu.Lock()
		for _, wg := range s.aLegGateways {
			if wg.ID == id {
				wg.Healthy = healthy
				break
			}
		}
		s.mu.Unlock()
	}

	return &ManualHealthCheckResponse{
		HealthCheckResult: HealthCheckResult{
			GatewayID:  id,
			Name:       gw.Name,
			SIPAddress: gw.SIPAddress,
			Healthy:    healthy,
			Failures:   newFailures,
		},
	}, nil
}

// checkTCP attempts a TCP connection to the SIP address with a 5s timeout.
// SIP addresses may be in formats like "sip:host:port" or "host:port".
func checkTCP(sipAddress string) bool {
	addr := sipAddress
	// Strip sip: prefix if present
	addr = strings.TrimPrefix(addr, "sip:")
	// If no port, default to 5060
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = addr + ":5060"
	}
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
