package callback

// CallStatusEvent is the real-time event pushed to WebSocket clients
// whenever a call's status changes.
type CallStatusEvent struct {
	CallID      string `json:"call_id"`
	UserID      string `json:"user_id"`
	Status      string `json:"status"` // initiating, a_dialing, a_connected, b_dialing, bridged, finished, failed
	ALeg        string `json:"a_leg"`
	BLeg        string `json:"b_leg"`
	DurationSec int    `json:"duration_sec"`
	Timestamp   int64  `json:"timestamp"`
}

// Status constants for CallStatusEvent.
const (
	StatusInitiating = "initiating"
	StatusADialing   = "a_dialing"
	StatusAConnected = "a_connected"
	StatusBDialing   = "b_dialing"
	StatusBridged    = "bridged"
	StatusFinished   = "finished"
	StatusFailed     = "failed"
)
