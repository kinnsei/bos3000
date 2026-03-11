package fsclient

import (
	"testing"
	"time"
)

func TestManager_Pick(t *testing.T) {
	mock := NewMockFSClient(MockConfig{ALegResult: "answer", BLegResult: "answer", BridgeResult: "stable", BridgeDuration: 100 * time.Millisecond})
	mgr := NewFSClientManagerWithClient(mock, nil)

	client, err := mgr.Pick()
	if err != nil {
		t.Fatal(err)
	}
	if client != mock {
		t.Error("expected mock client to be picked")
	}
}

func TestManager_Unhealthy(t *testing.T) {
	mock := NewMockFSClient(MockConfig{})
	mgr := NewFSClientManagerWithClient(mock, nil)

	mgr.mu.Lock()
	mgr.healthy = false
	mgr.mu.Unlock()

	_, err := mgr.Pick()
	if err == nil {
		t.Fatal("expected error when unhealthy")
	}
}

func TestManager_DisconnectCallback(t *testing.T) {
	mock := NewMockFSClient(MockConfig{})
	called := make(chan struct{}, 1)
	mgr := NewFSClientManagerWithClient(mock, func() {
		called <- struct{}{}
	})

	mgr.handleDisconnect()

	select {
	case <-called:
		// ok
	case <-time.After(time.Second):
		t.Fatal("disconnect callback not called")
	}
}
