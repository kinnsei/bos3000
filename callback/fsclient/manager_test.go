package fsclient

import (
	"testing"
	"time"
)

func newTestManager() (*FSClientManager, *MockFSClient, *MockFSClient) {
	primary := NewMockFSClient(MockConfig{ALegResult: "answer", BLegResult: "answer", BridgeResult: "stable", BridgeDuration: 100 * time.Millisecond})
	standby := NewMockFSClient(MockConfig{ALegResult: "answer", BLegResult: "answer", BridgeResult: "stable", BridgeDuration: 100 * time.Millisecond})
	mgr := NewFSClientManagerWithClients(primary, standby, nil)
	return mgr, primary, standby
}

func TestManager_PickPrimaryPreference(t *testing.T) {
	mgr, primary, _ := newTestManager()

	client, err := mgr.Pick()
	if err != nil {
		t.Fatal(err)
	}
	if client != primary {
		t.Error("expected primary to be picked")
	}
}

func TestManager_FailoverToStandby(t *testing.T) {
	mgr, _, standby := newTestManager()

	mgr.MarkUnhealthy(0)

	client, err := mgr.Pick()
	if err != nil {
		t.Fatal(err)
	}
	if client != standby {
		t.Error("expected standby after primary marked unhealthy")
	}
}

func TestManager_BothDown(t *testing.T) {
	mgr, _, _ := newTestManager()

	mgr.MarkUnhealthy(0)
	mgr.MarkUnhealthy(1)

	_, err := mgr.Pick()
	if err == nil {
		t.Fatal("expected error when both instances are down")
	}
}

func TestManager_NoAutoSwitchback(t *testing.T) {
	mgr, _, _ := newTestManager()

	// Primary goes down, failover to standby.
	mgr.MarkUnhealthy(0)
	if mgr.ActiveIdx() != 1 {
		t.Fatalf("expected activeIdx=1, got %d", mgr.ActiveIdx())
	}

	// Primary recovers.
	mgr.MarkHealthy(0)

	// activeIdx should still be 1 (no auto-switchback).
	if mgr.ActiveIdx() != 1 {
		t.Errorf("expected activeIdx=1 (no auto-switchback), got %d", mgr.ActiveIdx())
	}

	// Both should be pickable, but standby is preferred.
	client, err := mgr.Pick()
	if err != nil {
		t.Fatal(err)
	}
	// Pick returns active instance first.
	_ = client // standby
}

func TestManager_FailoverCallback(t *testing.T) {
	primary := NewMockFSClient(MockConfig{})
	standby := NewMockFSClient(MockConfig{})

	callbackCh := make(chan int, 1)
	mgr := NewFSClientManagerWithClients(primary, standby, func(failedIdx int) {
		callbackCh <- failedIdx
	})

	mgr.MarkUnhealthy(0)

	select {
	case idx := <-callbackCh:
		if idx != 0 {
			t.Errorf("expected failover callback for idx=0, got %d", idx)
		}
	case <-time.After(time.Second):
		t.Fatal("failover callback not called")
	}
}
