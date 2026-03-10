package fsclient

import (
	"context"
	"sync"
	"testing"
	"time"
)

func collectEvents(m *MockFSClient, eventNames ...string) chan CallEvent {
	ch := make(chan CallEvent, 20)
	for _, name := range eventNames {
		m.RegisterEventHandler(name, func(e CallEvent) {
			ch <- e
		})
	}
	return ch
}

func waitEvent(t *testing.T, ch chan CallEvent, timeout time.Duration) CallEvent {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(timeout):
		t.Fatal("timed out waiting for event")
		return CallEvent{}
	}
}

func TestMock_ALegAnswer(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "answer"})
	ch := collectEvents(m, "CHANNEL_ANSWER", "CHANNEL_HANGUP")

	uuid, err := m.OriginateALeg(context.Background(), OriginateParams{CallID: "call-1"})
	if err != nil {
		t.Fatal(err)
	}
	if uuid != "mock-a-call-1" {
		t.Errorf("unexpected uuid: %s", uuid)
	}

	e := waitEvent(t, ch, time.Second)
	if e.EventName != "CHANNEL_ANSWER" {
		t.Errorf("expected CHANNEL_ANSWER, got %s", e.EventName)
	}
	if e.Leg != "A" {
		t.Errorf("expected leg A, got %s", e.Leg)
	}
}

func TestMock_ALegReject(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "reject"})
	ch := collectEvents(m, "CHANNEL_ANSWER", "CHANNEL_HANGUP")

	_, err := m.OriginateALeg(context.Background(), OriginateParams{CallID: "call-2"})
	if err != nil {
		t.Fatal(err)
	}

	e := waitEvent(t, ch, time.Second)
	if e.EventName != "CHANNEL_HANGUP" {
		t.Errorf("expected CHANNEL_HANGUP, got %s", e.EventName)
	}
	if e.HangupCause != "CALL_REJECTED" {
		t.Errorf("expected CALL_REJECTED, got %s", e.HangupCause)
	}
}

func TestMock_ALegNoAnswer(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "no_answer"})
	ch := collectEvents(m, "CHANNEL_ANSWER", "CHANNEL_HANGUP")

	_, err := m.OriginateALeg(context.Background(), OriginateParams{CallID: "call-3"})
	if err != nil {
		t.Fatal(err)
	}

	e := waitEvent(t, ch, time.Second)
	if e.EventName != "CHANNEL_HANGUP" {
		t.Errorf("expected CHANNEL_HANGUP, got %s", e.EventName)
	}
	if e.HangupCause != "NO_ANSWER" {
		t.Errorf("expected NO_ANSWER, got %s", e.HangupCause)
	}
}

func TestMock_ALegError(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "error"})
	_, err := m.OriginateALeg(context.Background(), OriginateParams{CallID: "call-4"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMock_BLegSuccessAndBridge(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "answer", BLegResult: "answer", BridgeResult: "stable"})
	ch := collectEvents(m, "CHANNEL_ANSWER", "CHANNEL_BRIDGE", "CHANNEL_HANGUP")

	_, err := m.OriginateBLegAndBridge(context.Background(), "mock-a-1", OriginateParams{CallID: "call-5"})
	if err != nil {
		t.Fatal(err)
	}

	// Expect: B CHANNEL_ANSWER, CHANNEL_BRIDGE, CHANNEL_HANGUP
	e1 := waitEvent(t, ch, time.Second)
	if e1.EventName != "CHANNEL_ANSWER" || e1.Leg != "B" {
		t.Errorf("expected B CHANNEL_ANSWER, got %s %s", e1.Leg, e1.EventName)
	}

	e2 := waitEvent(t, ch, time.Second)
	if e2.EventName != "CHANNEL_BRIDGE" {
		t.Errorf("expected CHANNEL_BRIDGE, got %s", e2.EventName)
	}

	e3 := waitEvent(t, ch, time.Second)
	if e3.EventName != "CHANNEL_HANGUP" {
		t.Errorf("expected CHANNEL_HANGUP, got %s", e3.EventName)
	}
}

func TestMock_BLegReject(t *testing.T) {
	m := NewMockFSClient(MockConfig{BLegResult: "reject"})
	_, err := m.OriginateBLegAndBridge(context.Background(), "mock-a-1", OriginateParams{CallID: "call-6"})
	if err == nil {
		t.Fatal("expected error for B-leg reject")
	}
}

func TestMock_BLegBusy(t *testing.T) {
	m := NewMockFSClient(MockConfig{BLegResult: "busy"})
	_, err := m.OriginateBLegAndBridge(context.Background(), "mock-a-1", OriginateParams{CallID: "call-7"})
	if err == nil {
		t.Fatal("expected error for B-leg busy")
	}
}

func TestMock_HangupCall(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "answer"})
	ch := collectEvents(m, "CHANNEL_ANSWER", "CHANNEL_HANGUP")

	uuid, _ := m.OriginateALeg(context.Background(), OriginateParams{CallID: "call-8"})
	waitEvent(t, ch, time.Second) // consume CHANNEL_ANSWER

	err := m.HangupCall(context.Background(), uuid, "NORMAL_CLEARING")
	if err != nil {
		t.Fatal(err)
	}

	e := waitEvent(t, ch, time.Second)
	if e.EventName != "CHANNEL_HANGUP" {
		t.Errorf("expected CHANNEL_HANGUP, got %s", e.EventName)
	}
	if e.HangupCause != "NORMAL_CLEARING" {
		t.Errorf("expected NORMAL_CLEARING, got %s", e.HangupCause)
	}
}

func TestMock_RecordingNoOp(t *testing.T) {
	m := NewMockFSClient(MockConfig{})
	if err := m.StartRecording(context.Background(), "uuid", "call", "A"); err != nil {
		t.Fatal(err)
	}
	if err := m.StopRecording(context.Background(), "uuid", "call", "A"); err != nil {
		t.Fatal(err)
	}
}

func TestMock_ConcurrentSafety(t *testing.T) {
	m := NewMockFSClient(MockConfig{ALegResult: "answer"})
	collectEvents(m, "CHANNEL_ANSWER", "CHANNEL_HANGUP")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = m.OriginateALeg(context.Background(), OriginateParams{
				CallID: "concurrent-" + string(rune('0'+i)),
			})
		}(i)
	}
	wg.Wait()
}
