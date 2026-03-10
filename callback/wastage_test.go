package callback

import (
	"testing"
	"time"
)

func TestClassifyWastage_AConnectedBFailed(t *testing.T) {
	now := time.Now()
	aAnswer := now.Add(-30 * time.Second)
	aHangup := now
	call := &CallbackCall{
		AAnswerAt: &aAnswer,
		AHangupAt: &aHangup,
		BAnswerAt: nil,
		ALegRate:  60, // 60 fen/min
	}
	wType, wCost := classifyWastage(call, 10)
	if wType != "a_connected_b_failed" {
		t.Errorf("expected a_connected_b_failed, got %s", wType)
	}
	if wCost <= 0 {
		t.Errorf("expected positive wastage cost, got %d", wCost)
	}
}

func TestClassifyWastage_BridgeBrokenEarly(t *testing.T) {
	now := time.Now()
	aAnswer := now.Add(-20 * time.Second)
	bAnswer := now.Add(-15 * time.Second)
	bridgeAt := now.Add(-5 * time.Second)
	bridgeEnd := now
	call := &CallbackCall{
		AAnswerAt:        &aAnswer,
		BAnswerAt:        &bAnswer,
		BridgeAt:         &bridgeAt,
		BridgeEndAt:      &bridgeEnd,
		BridgeDurationMs: 5000,
		ALegRate:         60,
		BLegRate:         120,
	}
	wType, wCost := classifyWastage(call, 10) // 5s < 10s threshold
	if wType != "bridge_broken_early" {
		t.Errorf("expected bridge_broken_early, got %s", wType)
	}
	if wCost <= 0 {
		t.Errorf("expected positive wastage cost, got %d", wCost)
	}
}

func TestClassifyWastage_NormalBridge(t *testing.T) {
	now := time.Now()
	aAnswer := now.Add(-60 * time.Second)
	bAnswer := now.Add(-55 * time.Second)
	bridgeAt := now.Add(-50 * time.Second)
	bridgeEnd := now
	call := &CallbackCall{
		AAnswerAt:        &aAnswer,
		BAnswerAt:        &bAnswer,
		BridgeAt:         &bridgeAt,
		BridgeEndAt:      &bridgeEnd,
		BridgeDurationMs: 50000,
		ALegRate:         60,
		BLegRate:         120,
	}
	wType, wCost := classifyWastage(call, 10)
	if wType != "" {
		t.Errorf("expected no wastage, got %s", wType)
	}
	if wCost != 0 {
		t.Errorf("expected 0 cost, got %d", wCost)
	}
}

func TestClassifyWastage_ANeverConnected(t *testing.T) {
	call := &CallbackCall{AAnswerAt: nil}
	wType, wCost := classifyWastage(call, 10)
	if wType != "" {
		t.Errorf("expected no wastage, got %s", wType)
	}
	if wCost != 0 {
		t.Errorf("expected 0 cost, got %d", wCost)
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name                     string
		durSec, rate, block, want int64
	}{
		{"7s at 60/min 6s blocks", 7, 60, 6, 12},
		{"0 duration", 0, 60, 6, 0},
		{"exactly 6s", 6, 60, 6, 6},
		{"1s rounds up to 1 block", 1, 60, 6, 6},
		{"60s at 120/min 60s blocks", 60, 120, 60, 120},
		{"1s at 120/min 60s blocks", 1, 120, 60, 120},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCost(tt.durSec, tt.rate, tt.block)
			if got != tt.want {
				t.Errorf("calculateCost(%d,%d,%d) = %d, want %d",
					tt.durSec, tt.rate, tt.block, got, tt.want)
			}
		})
	}
}
