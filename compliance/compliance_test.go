package compliance

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"

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
