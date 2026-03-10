package callback

import "encore.app/pkg/types"

// classifyWastage determines if a call has wastage and calculates the cost.
func classifyWastage(call *CallbackCall, bridgeBrokenThresholdSec int64) (wastageType string, wastageCost int64) {
	// Case 1: A connected but B never answered
	if call.AAnswerAt != nil && call.BAnswerAt == nil {
		// A-leg waited for B but B failed. Cost = A-leg cost during wait.
		if call.AHangupAt != nil {
			durationSec := int64(call.AHangupAt.Sub(*call.AAnswerAt).Seconds())
			if durationSec < 1 {
				durationSec = 1
			}
			cost := calculateCost(durationSec, call.ALegRate, 6)
			return "a_connected_b_failed", cost
		}
		return "a_connected_b_failed", 0
	}

	// Case 2: Bridge established but broken early
	if call.BridgeAt != nil && call.BridgeEndAt != nil {
		bridgeDurationSec := int64(call.BridgeEndAt.Sub(*call.BridgeAt).Seconds())
		if bridgeDurationSec < bridgeBrokenThresholdSec {
			// Both legs cost during the short bridge
			aCost := calculateCost(bridgeDurationSec, call.ALegRate, 6)
			bCost := calculateCost(bridgeDurationSec, call.BLegRate, 60)
			return "bridge_broken_early", aCost + bCost
		}
	}

	return "", 0
}

// calculateCost calculates cost in fen using billing blocks.
// Formula: blocks = ceil(durationSec / blockSizeSec), cost = blocks * blockSizeSec * ratePerMin / 60
func calculateCost(durationSec, ratePerMin, blockSizeSec int64) int64 {
	if durationSec <= 0 || ratePerMin <= 0 {
		return 0
	}
	blocks := types.CeilDiv(durationSec, blockSizeSec)
	return blocks * blockSizeSec * ratePerMin / 60
}
