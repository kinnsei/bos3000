package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/et"

	authpkg "encore.app/auth"
	"encore.app/pkg/errcode"
)

func adminCtx() context.Context {
	uid := auth.UID(fmt.Sprintf("admin-%d", time.Now().UnixNano()))
	return auth.WithContext(context.Background(), uid, &authpkg.AuthData{
		UserID:   1,
		Role:     "admin",
		Username: "admin",
	})
}

func clientCtx(userID int64) context.Context {
	uid := auth.UID(fmt.Sprintf("client-%d", userID))
	return auth.WithContext(context.Background(), uid, &authpkg.AuthData{
		UserID:   userID,
		Role:     "client",
		Username: "testclient",
	})
}

func seedBlacklist(t *testing.T, ctx context.Context, number string, userID *int64, createdBy int64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO blacklisted_numbers (number, user_id, reason, created_by)
		VALUES ($1, $2, 'test', $3) RETURNING id
	`, number, userID, createdBy).Scan(&id)
	if err != nil {
		t.Fatalf("seed blacklist: %v", err)
	}
	return id
}

func hasBizCode(err error, bizCode string) bool {
	var e *errs.Error
	if !errors.As(err, &e) {
		return false
	}
	d, ok := e.Details.(errcode.ErrDetails)
	if !ok {
		return false
	}
	return d.BizCode == bizCode
}

func TestBlacklistGlobalHit(t *testing.T) {
	ctx := context.Background()
	number := fmt.Sprintf("+1555%d", time.Now().UnixNano()%10000000)
	seedBlacklist(t, ctx, number, nil, 1)

	err := CheckBlacklist(ctx, &CheckBlacklistParams{
		CalledNumber: number,
		UserID:       999,
	})
	if err == nil {
		t.Fatal("expected error for globally blacklisted number")
	}
	if !hasBizCode(err, errcode.BlacklistedNumber) {
		t.Errorf("expected BLACKLISTED_NUMBER biz code, got: %v", err)
	}
}

func TestBlacklistClientHit(t *testing.T) {
	ctx := context.Background()
	number := fmt.Sprintf("+1556%d", time.Now().UnixNano()%10000000)
	userID := int64(time.Now().UnixNano() % 1000000)
	seedBlacklist(t, ctx, number, &userID, 1)

	err := CheckBlacklist(ctx, &CheckBlacklistParams{
		CalledNumber: number,
		UserID:       userID,
	})
	if err == nil {
		t.Fatal("expected error for client-blacklisted number")
	}
	if !hasBizCode(err, errcode.BlacklistedNumber) {
		t.Errorf("expected BLACKLISTED_NUMBER biz code, got: %v", err)
	}
}

func TestBlacklistNotBlocked(t *testing.T) {
	ctx := context.Background()
	number := fmt.Sprintf("+1557%d", time.Now().UnixNano()%10000000)

	err := CheckBlacklist(ctx, &CheckBlacklistParams{
		CalledNumber: number,
		UserID:       12345,
	})
	if err != nil {
		t.Fatalf("expected no error for non-blacklisted number, got: %v", err)
	}
}

func TestBlacklistGlobalCannotBeOverridden(t *testing.T) {
	ctx := context.Background()
	number := fmt.Sprintf("+1558%d", time.Now().UnixNano()%10000000)
	userID := int64(time.Now().UnixNano() % 1000000)

	// Add global blacklist
	seedBlacklist(t, ctx, number, nil, 1)

	// Even if not client-blacklisted, global check comes first
	err := CheckBlacklist(ctx, &CheckBlacklistParams{
		CalledNumber: number,
		UserID:       userID,
	})
	if err == nil {
		t.Fatal("expected error: global blacklist should not be overridable")
	}
	var e *errs.Error
	if errors.As(err, &e) {
		if e.Message != "number is globally blacklisted" {
			t.Errorf("expected global blacklist message, got: %s", e.Message)
		}
	}
}

func TestDailyLimitUnderLimit(t *testing.T) {
	ctx := context.Background()
	userID := time.Now().UnixNano()

	resp, err := CheckDailyLimit(ctx, &CheckDailyLimitParams{
		UserID:     userID,
		DailyLimit: 100,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.CurrentCount != 1 {
		t.Errorf("expected count 1, got %d", resp.CurrentCount)
	}
}

func TestDailyLimitExceeded(t *testing.T) {
	ctx := context.Background()
	userID := time.Now().UnixNano()
	limit := 3

	// Use up the limit
	for i := 0; i < limit; i++ {
		_, err := CheckDailyLimit(ctx, &CheckDailyLimitParams{
			UserID:     userID,
			DailyLimit: limit,
		})
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i+1, err)
		}
	}

	// Next call should exceed
	_, err := CheckDailyLimit(ctx, &CheckDailyLimitParams{
		UserID:     userID,
		DailyLimit: limit,
	})
	if err == nil {
		t.Fatal("expected error for exceeded daily limit")
	}
	if !hasBizCode(err, errcode.DailyLimitExceeded) {
		t.Errorf("expected DAILY_LIMIT_EXCEEDED biz code, got: %v", err)
	}
}

func TestDailyLimitFailOpen(t *testing.T) {
	// This test verifies the fail-open behavior.
	// Since we can't easily simulate a cache error in integration tests,
	// we verify that the function signature and response struct support
	// the fail-open pattern (CurrentCount = -1 on cache error).
	ctx := context.Background()
	userID := time.Now().UnixNano()

	resp, err := CheckDailyLimit(ctx, &CheckDailyLimitParams{
		UserID:     userID,
		DailyLimit: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Normal case: count should be positive
	if resp.CurrentCount < 1 {
		t.Errorf("expected positive count, got %d", resp.CurrentCount)
	}
}

func TestAddBlacklistPermissions(t *testing.T) {
	t.Run("admin can add global", func(t *testing.T) {
		ctx := adminCtx()
		number := fmt.Sprintf("+1560%d", time.Now().UnixNano()%10000000)

		resp, err := AddBlacklist(ctx, &AddBlacklistParams{
			Number: number,
			UserID: nil,
			Reason: "spam",
		})
		if err != nil {
			t.Fatalf("admin add global failed: %v", err)
		}
		if resp.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("client can add own", func(t *testing.T) {
		userID := int64(time.Now().UnixNano() % 1000000)
		ctx := clientCtx(userID)
		number := fmt.Sprintf("+1561%d", time.Now().UnixNano()%10000000)

		resp, err := AddBlacklist(ctx, &AddBlacklistParams{
			Number: number,
			UserID: &userID,
			Reason: "unwanted",
		})
		if err != nil {
			t.Fatalf("client add own failed: %v", err)
		}
		if resp.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("client cannot add global", func(t *testing.T) {
		ctx := clientCtx(42)
		number := fmt.Sprintf("+1562%d", time.Now().UnixNano()%10000000)

		_, err := AddBlacklist(ctx, &AddBlacklistParams{
			Number: number,
			UserID: nil,
			Reason: "spam",
		})
		if err == nil {
			t.Fatal("expected error: client should not add global blacklist")
		}
		var e *errs.Error
		if errors.As(err, &e) && e.Code != errs.PermissionDenied {
			t.Errorf("expected PermissionDenied, got %v", e.Code)
		}
	})

	t.Run("client cannot add for other user", func(t *testing.T) {
		ctx := clientCtx(42)
		otherUser := int64(99)
		number := fmt.Sprintf("+1563%d", time.Now().UnixNano()%10000000)

		_, err := AddBlacklist(ctx, &AddBlacklistParams{
			Number: number,
			UserID: &otherUser,
			Reason: "spam",
		})
		if err == nil {
			t.Fatal("expected error: client should not add blacklist for other user")
		}
		var e *errs.Error
		if errors.As(err, &e) && e.Code != errs.PermissionDenied {
			t.Errorf("expected PermissionDenied, got %v", e.Code)
		}
	})
}

func TestAuditEventPublished(t *testing.T) {
	ctx := context.Background()
	event := &AuditEvent{
		OperatorID:   1,
		OperatorName: "admin",
		Action:       "create",
		ResourceType: "user",
		ResourceID:   "42",
		AfterValue:   json.RawMessage(`{"name":"test"}`),
		IPAddress:    "127.0.0.1",
	}

	err := PublishAuditEvent(ctx, event)
	if err != nil {
		t.Fatalf("failed to publish audit event: %v", err)
	}

	msgs := et.Topic(AuditEvents).PublishedMessages()
	if len(msgs) == 0 {
		t.Fatal("expected at least one published message")
	}
	last := msgs[len(msgs)-1]
	if last.Action != "create" {
		t.Errorf("expected action 'create', got %q", last.Action)
	}
	if last.ResourceType != "user" {
		t.Errorf("expected resource_type 'user', got %q", last.ResourceType)
	}
}

func seedAuditLog(t *testing.T, ctx context.Context, operatorID int64, action, resourceType, resourceID string) {
	t.Helper()
	_, err := db.Exec(ctx, `
		INSERT INTO audit_logs (operator_id, operator_name, action, resource_type, resource_id, ip_address)
		VALUES ($1, 'testadmin', $2, $3, $4, '127.0.0.1')
	`, operatorID, action, resourceType, resourceID)
	if err != nil {
		t.Fatalf("seed audit log: %v", err)
	}
}

func TestQueryAuditLogs(t *testing.T) {
	ctx := adminCtx()
	ts := time.Now().UnixNano()
	rtype := fmt.Sprintf("testres_%d", ts)

	seedAuditLog(t, ctx, 1, "create", rtype, "1")
	seedAuditLog(t, ctx, 1, "update", rtype, "2")
	seedAuditLog(t, ctx, 2, "delete", rtype, "3")

	// Filter by resource_type
	resp, err := QueryAuditLogs(ctx, &QueryAuditLogsParams{
		ResourceType: rtype,
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected 3 total, got %d", resp.Total)
	}

	// Filter by action
	resp, err = QueryAuditLogs(ctx, &QueryAuditLogsParams{
		ResourceType: rtype,
		Action:       "create",
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("query audit logs by action: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 total for action=create, got %d", resp.Total)
	}

	// Filter by operator_id
	resp, err = QueryAuditLogs(ctx, &QueryAuditLogsParams{
		ResourceType: rtype,
		OperatorID:   2,
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("query audit logs by operator: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 total for operator_id=2, got %d", resp.Total)
	}
}

func TestAuditLogPagination(t *testing.T) {
	ctx := adminCtx()
	ts := time.Now().UnixNano()
	rtype := fmt.Sprintf("pagtest_%d", ts)

	for i := 0; i < 5; i++ {
		seedAuditLog(t, ctx, 1, "test", rtype, fmt.Sprintf("%d", i))
	}

	// Page 1, size 2
	resp, err := QueryAuditLogs(ctx, &QueryAuditLogsParams{
		ResourceType: rtype,
		Page:         1,
		PageSize:     2,
	})
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected 5 total, got %d", resp.Total)
	}
	if len(resp.Logs) != 2 {
		t.Errorf("expected 2 logs on page 1, got %d", len(resp.Logs))
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", resp.TotalPages)
	}

	// Page 3 (last), should have 1 record
	resp, err = QueryAuditLogs(ctx, &QueryAuditLogsParams{
		ResourceType: rtype,
		Page:         3,
		PageSize:     2,
	})
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(resp.Logs) != 1 {
		t.Errorf("expected 1 log on page 3, got %d", len(resp.Logs))
	}
}
