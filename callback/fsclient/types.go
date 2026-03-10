package fsclient

import "time"

// OriginateParams contains parameters for originating a call leg.
type OriginateParams struct {
	CallID      string
	Number      string
	CallerID    string
	GatewayIP   string
	GatewayPort int
	MaxDuration int
	Vars        map[string]string
}

// CallEvent represents a FreeSWITCH channel event routed to the state machine.
type CallEvent struct {
	CallID      string
	UUID        string
	Leg         string // "A" or "B"
	EventName   string // "CHANNEL_ANSWER", "CHANNEL_BRIDGE", "CHANNEL_HANGUP"
	HangupCause string
	Timestamp   time.Time
}
