package webhook

import (
	"encoding/json"
	"time"
)

const (
	StatusPending    = "pending"
	StatusDelivering = "delivering"
	StatusDelivered  = "delivered"
	StatusRetrying   = "retrying"
	StatusFailed     = "failed"
	StatusDLQ        = "dlq"
)

// WebhookEvent is published to trigger webhook delivery.
type WebhookEvent struct {
	DeliveryID int64           `json:"delivery_id"`
	WebhookURL string          `json:"webhook_url"`
	Secret     string          `json:"-" encore:"sensitive"`
	Payload    json.RawMessage `json:"payload"`
}

// WebhookPayload is the actual payload sent to the customer's endpoint.
type WebhookPayload struct {
	EventType  string          `json:"event_type"`
	CallID     string          `json:"call_id"`
	Status     string          `json:"status"`
	ALeg       *LegDetail      `json:"a_leg"`
	BLeg       *LegDetail      `json:"b_leg,omitempty"`
	Bridge     *BridgeDetail   `json:"bridge,omitempty"`
	CustomData json.RawMessage `json:"custom_data,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// LegDetail contains details about one call leg.
type LegDetail struct {
	Number      string     `json:"number"`
	Status      string     `json:"status"`
	DialAt      *time.Time `json:"dial_at,omitempty"`
	AnswerAt    *time.Time `json:"answer_at,omitempty"`
	HangupAt    *time.Time `json:"hangup_at,omitempty"`
	HangupCause string     `json:"hangup_cause,omitempty"`
}

// BridgeDetail contains details about the call bridge.
type BridgeDetail struct {
	BridgedAt *time.Time `json:"bridged_at,omitempty"`
	Duration  int        `json:"duration"`
}
