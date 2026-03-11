package webhook

import (
	"testing"
	"time"
)

func TestRetryIntervalForAttempt(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 30 * time.Second},
		{2, 1 * time.Minute},
		{3, 5 * time.Minute},
		{4, 15 * time.Minute},
		{5, 1 * time.Hour},
		{6, 0}, // DLQ
		{0, 0}, // edge case
		{-1, 0},
	}

	for _, tc := range tests {
		got := retryIntervalForAttempt(tc.attempt)
		if got != tc.expected {
			t.Errorf("retryIntervalForAttempt(%d) = %v, want %v", tc.attempt, got, tc.expected)
		}
	}
}
