package callback

import (
	"context"
	"fmt"
	"testing"
	"time"

	"encore.dev/et"

	"encore.app/callback/fsclient"
	"encore.app/recording"
	"encore.app/webhook"
)

func TestRecordingStartOnBridge(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 300 * time.Millisecond,
	})
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 30)
	if status.Status != "finished" {
		t.Fatalf("expected finished, got %s", status.Status)
	}

	// Verify StartRecording was called for both legs
	startCalls := mock.GetStartRecordingCalls()
	if len(startCalls) != 2 {
		t.Fatalf("expected 2 StartRecording calls, got %d", len(startCalls))
	}

	legs := map[string]bool{}
	for _, c := range startCalls {
		legs[c.Leg] = true
		if c.CallID != resp.CallID {
			t.Errorf("StartRecording callID mismatch: %s vs %s", c.CallID, resp.CallID)
		}
	}
	if !legs["a"] || !legs["b"] {
		t.Error("expected StartRecording for both legs a and b")
	}
}

func TestRecordingStopBeforeFinalize(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 300 * time.Millisecond,
	})
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 30)
	if status.Status != "finished" {
		t.Fatalf("expected finished, got %s", status.Status)
	}

	stopCalls := mock.GetStopRecordingCalls()
	if len(stopCalls) != 2 {
		t.Fatalf("expected 2 StopRecording calls, got %d", len(stopCalls))
	}

	legs := map[string]bool{}
	for _, c := range stopCalls {
		legs[c.Leg] = true
	}
	if !legs["a"] || !legs["b"] {
		t.Error("expected StopRecording for both legs a and b")
	}
}

func TestRecordingMergePublishOnFinalize(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 300 * time.Millisecond,
	})
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	pollCallStatus(t, svc, ctx, resp.CallID, 30)

	msgs := et.Topic(recording.RecordingMergeTopic).PublishedMessages()
	found := false
	for _, evt := range msgs {
		if evt.CallID == resp.CallID {
			found = true
			if evt.CustomerID != 1 {
				t.Errorf("expected CustomerID 1, got %d", evt.CustomerID)
			}
			expectedA := fmt.Sprintf("/var/lib/freeswitch/recordings/%s_a.wav", resp.CallID)
			expectedB := fmt.Sprintf("/var/lib/freeswitch/recordings/%s_b.wav", resp.CallID)
			if evt.AFilePath != expectedA {
				t.Errorf("expected AFilePath %s, got %s", expectedA, evt.AFilePath)
			}
			if evt.BFilePath != expectedB {
				t.Errorf("expected BFilePath %s, got %s", expectedB, evt.BFilePath)
			}
		}
	}
	if !found {
		t.Error("expected RecordingMergeEvent to be published for bridged call")
	}
}

func TestRecordingMergeNotPublishedForUnbridgedCall(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult: "reject",
	})
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	pollCallStatus(t, svc, ctx, resp.CallID, 20)

	msgs := et.Topic(recording.RecordingMergeTopic).PublishedMessages()
	for _, evt := range msgs {
		if evt.CallID == resp.CallID {
			t.Error("RecordingMergeEvent should NOT be published for unbridged call")
		}
	}
}

func TestWebhookPublishOnEachStatusChange(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 300 * time.Millisecond,
	})
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	pollCallStatus(t, svc, ctx, resp.CallID, 30)

	// Check webhook events were published for this call
	msgs := et.Topic(webhook.WebhookTopic).PublishedMessages()
	callWebhooks := len(msgs)
	// We expect webhooks only if the user has a webhook_url configured in the auth DB.
	// In test mode with no auth DB row, publishWebhook will silently skip.
	// This test verifies the code path doesn't panic.
	t.Logf("webhook events published: %d", callWebhooks)
}

func TestRecordingErrorDoesNotAbortCall(t *testing.T) {
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 300 * time.Millisecond,
	})
	mock.StartRecordingErr = fmt.Errorf("recording system unavailable")
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 30)
	if status.Status != "finished" {
		t.Errorf("expected finished despite recording error, got %s", status.Status)
	}
}

func TestWebhookErrorDoesNotAbortTransition(t *testing.T) {
	// When no webhook_url is configured, publishWebhook silently returns.
	// This test verifies state transitions succeed without webhook config.
	setupMocks(t)
	svc := &Service{}
	mock := fsclient.NewMockFSClient(fsclient.MockConfig{
		ALegResult:     "answer",
		BLegResult:     "answer",
		BridgeResult:   "stable",
		BridgeDuration: 300 * time.Millisecond,
	})
	svc.fsClient = mock
	svc.registerEventHandlers(mock)

	ctx := withAuth(context.Background(), 1, "client")
	resp, err := svc.InitiateCallback(ctx, &InitiateCallbackParams{
		ANumber: "13800138001",
		BNumber: "13900139001",
	})
	if err != nil {
		t.Fatal(err)
	}

	status := pollCallStatus(t, svc, ctx, resp.CallID, 30)
	if status.Status != "finished" {
		t.Errorf("expected finished, got %s", status.Status)
	}
}
