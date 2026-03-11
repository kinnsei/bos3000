package callback

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHub_NewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map is nil")
	}
	if hub.adminConns == nil {
		t.Error("adminConns map is nil")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel is nil")
	}
	if hub.register == nil {
		t.Error("register channel is nil")
	}
	if hub.unregister == nil {
		t.Error("unregister channel is nil")
	}
}

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Register a regular client.
	regularClient := &Client{
		hub:     hub,
		send:    make(chan []byte, 256),
		userID:  "user-1",
		isAdmin: false,
	}
	hub.register <- regularClient

	// Register an admin client.
	adminClient := &Client{
		hub:     hub,
		send:    make(chan []byte, 256),
		userID:  "admin-1",
		isAdmin: true,
	}
	hub.register <- adminClient

	// Give the hub goroutine time to process.
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if _, ok := hub.clients["user-1"][regularClient]; !ok {
		t.Error("regular client not registered in clients map")
	}
	if _, ok := hub.adminConns[adminClient]; !ok {
		t.Error("admin client not registered in adminConns map")
	}
	// Admin should NOT be in the regular clients map.
	if _, ok := hub.clients["admin-1"]; ok {
		t.Error("admin client should not be in regular clients map")
	}
}

func TestHub_BroadcastRouting(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	admin := &Client{hub: hub, send: make(chan []byte, 256), userID: "admin", isAdmin: true}
	userA := &Client{hub: hub, send: make(chan []byte, 256), userID: "user-a", isAdmin: false}
	userB := &Client{hub: hub, send: make(chan []byte, 256), userID: "user-b", isAdmin: false}

	hub.register <- admin
	hub.register <- userA
	hub.register <- userB

	time.Sleep(50 * time.Millisecond)

	event := &CallStatusEvent{
		CallID:      "call-123",
		UserID:      "user-a",
		Status:      StatusBridged,
		ALeg:        "+1111",
		BLeg:        "+2222",
		DurationSec: 10,
		Timestamp:   time.Now().Unix(),
	}

	hub.Broadcast(event)

	// Admin should receive the event.
	select {
	case msg := <-admin.send:
		var got CallStatusEvent
		if err := json.Unmarshal(msg, &got); err != nil {
			t.Fatalf("unmarshal admin msg: %v", err)
		}
		if got.CallID != "call-123" {
			t.Errorf("admin got call_id=%s, want call-123", got.CallID)
		}
	case <-time.After(time.Second):
		t.Error("admin did not receive event")
	}

	// User A should receive the event.
	select {
	case msg := <-userA.send:
		var got CallStatusEvent
		if err := json.Unmarshal(msg, &got); err != nil {
			t.Fatalf("unmarshal userA msg: %v", err)
		}
		if got.Status != StatusBridged {
			t.Errorf("userA got status=%s, want bridged", got.Status)
		}
	case <-time.After(time.Second):
		t.Error("userA did not receive event")
	}

	// User B should NOT receive the event.
	select {
	case <-userB.send:
		t.Error("userB should not have received event")
	case <-time.After(200 * time.Millisecond):
		// Expected: no message.
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{hub: hub, send: make(chan []byte, 256), userID: "user-x", isAdmin: false}
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	if _, ok := hub.clients["user-x"][client]; !ok {
		hub.mu.RUnlock()
		t.Fatal("client not registered")
	}
	hub.mu.RUnlock()

	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if conns, ok := hub.clients["user-x"]; ok && len(conns) > 0 {
		t.Error("client still registered after unregister")
	}
}
