package compliance

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/storage/cache"

	"encore.app/pkg/errcode"
)

// DailyCallKey is the cache key for daily call counts per user.
type DailyCallKey struct {
	UserID int64
	Date   string
}

var dailyCalls = cache.NewIntKeyspace[DailyCallKey](complianceCache, cache.KeyspaceConfig{
	KeyPattern:    "daily-calls/:UserID/:Date",
	DefaultExpiry: cache.ExpireIn(25 * time.Hour),
})

// CheckDailyLimitParams contains the parameters for checking the daily call limit.
type CheckDailyLimitParams struct {
	UserID     int64 `json:"user_id"`
	DailyLimit int   `json:"daily_limit"`
}

// CheckDailyLimitResponse contains the current daily call count.
type CheckDailyLimitResponse struct {
	CurrentCount int `json:"current_count"`
}

// CheckDailyLimit checks and increments the daily call counter. Returns error if limit exceeded.
// Fails open on Redis errors (allows the call but logs a warning).
//
//encore:api private method=POST path=/compliance/check-daily-limit
func (s *Service) CheckDailyLimit(ctx context.Context, p *CheckDailyLimitParams) (*CheckDailyLimitResponse, error) {
	today := time.Now().Format("2006-01-02")
	key := DailyCallKey{UserID: p.UserID, Date: today}

	count, err := dailyCalls.Increment(ctx, key, 1)
	if err != nil {
		// Fail open: allow the call but log warning
		rlog.Warn("daily limit cache error, failing open", "user_id", p.UserID, "error", err)
		return &CheckDailyLimitResponse{CurrentCount: -1}, nil
	}

	if int(count) > p.DailyLimit {
		// Undo optimistic increment
		if _, decErr := dailyCalls.Increment(ctx, key, -1); decErr != nil {
			rlog.Warn("failed to decrement daily counter after limit exceeded", "user_id", p.UserID, "error", decErr)
		}
		return nil, errcode.NewError(errs.ResourceExhausted, errcode.DailyLimitExceeded,
			"daily call limit exceeded")
	}

	return &CheckDailyLimitResponse{CurrentCount: int(count)}, nil
}
