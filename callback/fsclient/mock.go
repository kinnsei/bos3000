package fsclient

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// MockConfig controls the behavior of MockFSClient for testing.
type MockConfig struct {
	ALegResult    string        // "answer", "no_answer", "reject", "error"
	BLegResult    string        // "answer", "reject", "error", "busy"
	BridgeResult  string        // "stable", "early_hangup"
	BridgeDuration time.Duration // simulated bridge duration for billing calc
	ALegHangupCause string
	BLegHangupCause string
}

type mockCallState struct {
	uuid   string
	callID string
	leg    string
	active bool
}

// MockFSClient implements FSClient for testing with configurable scenarios.
type MockFSClient struct {
	config   MockConfig
	mu       sync.RWMutex
	handlers map[string][]func(CallEvent)
	callsMu  sync.Mutex
	calls    map[string]*mockCallState
}

// NewMockFSClient creates a MockFSClient with the given config.
func NewMockFSClient(config MockConfig) *MockFSClient {
	return &MockFSClient{
		config:   config,
		handlers: make(map[string][]func(CallEvent)),
		calls:    make(map[string]*mockCallState),
	}
}

func (m *MockFSClient) OriginateALeg(ctx context.Context, params OriginateParams) (string, error) {
	if m.config.ALegResult == "error" {
		return "", fmt.Errorf("a-leg originate failed: system error")
	}

	uuid := "mock-a-" + params.CallID
	m.callsMu.Lock()
	m.calls[uuid] = &mockCallState{uuid: uuid, callID: params.CallID, leg: "A", active: true}
	m.callsMu.Unlock()

	go func() {
		runtime.Gosched()
		now := time.Now()
		switch m.config.ALegResult {
		case "answer":
			m.fireEvent(CallEvent{
				CallID:    params.CallID,
				UUID:      uuid,
				Leg:       "A",
				EventName: "CHANNEL_ANSWER",
				Timestamp: now,
			})
		case "no_answer":
			cause := m.config.ALegHangupCause
			if cause == "" {
				cause = "NO_ANSWER"
			}
			m.fireEvent(CallEvent{
				CallID:      params.CallID,
				UUID:        uuid,
				Leg:         "A",
				EventName:   "CHANNEL_HANGUP",
				HangupCause: cause,
				Timestamp:   now,
			})
		case "reject":
			cause := m.config.ALegHangupCause
			if cause == "" {
				cause = "CALL_REJECTED"
			}
			m.fireEvent(CallEvent{
				CallID:      params.CallID,
				UUID:        uuid,
				Leg:         "A",
				EventName:   "CHANNEL_HANGUP",
				HangupCause: cause,
				Timestamp:   now,
			})
		}
	}()

	return uuid, nil
}

func (m *MockFSClient) OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) (string, error) {
	switch m.config.BLegResult {
	case "error":
		return "", fmt.Errorf("b-leg originate failed: system error")
	case "reject":
		return "", fmt.Errorf("b-leg originate failed: call rejected")
	case "busy":
		return "", fmt.Errorf("b-leg originate failed: user busy")
	}

	uuid := "mock-b-" + params.CallID
	m.callsMu.Lock()
	m.calls[uuid] = &mockCallState{uuid: uuid, callID: params.CallID, leg: "B", active: true}
	m.callsMu.Unlock()

	go func() {
		runtime.Gosched()
		now := time.Now()

		// B answers
		m.fireEvent(CallEvent{
			CallID:    params.CallID,
			UUID:      uuid,
			Leg:       "B",
			EventName: "CHANNEL_ANSWER",
			Timestamp: now,
		})

		runtime.Gosched()

		// Bridge established
		m.fireEvent(CallEvent{
			CallID:    params.CallID,
			UUID:      uuid,
			Leg:       "B",
			EventName: "CHANNEL_BRIDGE",
			Timestamp: now,
		})

		runtime.Gosched()

		// Bridge ends
		cause := m.config.BLegHangupCause
		if cause == "" {
			cause = "NORMAL_CLEARING"
		}
		m.fireEvent(CallEvent{
			CallID:      params.CallID,
			UUID:        uuid,
			Leg:         "B",
			EventName:   "CHANNEL_HANGUP",
			HangupCause: cause,
			Timestamp:   now,
		})
	}()

	return uuid, nil
}

func (m *MockFSClient) HangupCall(ctx context.Context, uuid string, cause string) error {
	m.callsMu.Lock()
	cs, ok := m.calls[uuid]
	if ok {
		cs.active = false
	}
	m.callsMu.Unlock()

	if ok {
		m.fireEvent(CallEvent{
			CallID:      cs.callID,
			UUID:        uuid,
			Leg:         cs.leg,
			EventName:   "CHANNEL_HANGUP",
			HangupCause: cause,
			Timestamp:   time.Now(),
		})
	}
	return nil
}

func (m *MockFSClient) StartRecording(ctx context.Context, uuid string, callID string, leg string) error {
	return nil
}

func (m *MockFSClient) StopRecording(ctx context.Context, uuid string, callID string, leg string) error {
	return nil
}

func (m *MockFSClient) RegisterEventHandler(eventName string, handler func(CallEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[eventName] = append(m.handlers[eventName], handler)
}

func (m *MockFSClient) fireEvent(event CallEvent) {
	m.mu.RLock()
	handlers := m.handlers[event.EventName]
	m.mu.RUnlock()
	for _, h := range handlers {
		h(event)
	}
}
