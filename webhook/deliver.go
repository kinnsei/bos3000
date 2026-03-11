package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"encore.dev/pubsub"
	"encore.dev/rlog"
)

var _ = pubsub.NewSubscription(
	WebhookTopic, "webhook-delivery-worker",
	pubsub.SubscriptionConfig[*WebhookEvent]{
		Handler: pubsub.MethodHandler((*Service).DeliverWebhook),
		RetryPolicy: &pubsub.RetryPolicy{
			MaxRetries: 0,
		},
	},
)

// DeliverWebhook attempts to deliver a webhook to the customer's endpoint.
func (s *Service) DeliverWebhook(ctx context.Context, event *WebhookEvent) error {
	// Update status to delivering and increment attempt count.
	_, _ = webhookDB.Exec(ctx,
		`UPDATE webhook_deliveries SET status = $1, attempt_count = attempt_count + 1,
		last_attempt_at = NOW(), updated_at = NOW() WHERE id = $2`,
		StatusDelivering, event.DeliveryID)

	// Compute signature.
	signature := SignPayload(event.Secret, event.Payload)

	// Create HTTP request.
	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, event.WebhookURL, bytes.NewReader(event.Payload))
	if err != nil {
		return s.handleDeliveryFailure(ctx, event.DeliveryID, fmt.Sprintf("create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Delivery-ID", strconv.FormatInt(event.DeliveryID, 10))

	resp, err := httpClient.Do(req)
	if err != nil {
		return s.handleDeliveryFailure(ctx, event.DeliveryID, fmt.Sprintf("http error: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, _ = webhookDB.Exec(ctx,
			`UPDATE webhook_deliveries SET status = $1, delivered_at = NOW(), updated_at = NOW()
			WHERE id = $2`, StatusDelivered, event.DeliveryID)
		return nil
	}

	// Read up to 1KB of response body for error context.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	lastError := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	return s.handleDeliveryFailure(ctx, event.DeliveryID, lastError)
}

func (s *Service) handleDeliveryFailure(ctx context.Context, deliveryID int64, lastError string) error {
	var attemptCount int
	err := webhookDB.QueryRow(ctx,
		"SELECT attempt_count FROM webhook_deliveries WHERE id = $1", deliveryID,
	).Scan(&attemptCount)
	if err != nil {
		rlog.Warn("lookup attempt count", "delivery_id", deliveryID, "error", err)
		return nil // Don't trigger Pub/Sub retry.
	}

	interval := retryIntervalForAttempt(attemptCount)
	if interval > 0 {
		_, _ = webhookDB.Exec(ctx,
			`UPDATE webhook_deliveries SET status = $1, next_retry_at = NOW() + $2::interval,
			last_error = $3, updated_at = NOW() WHERE id = $4`,
			StatusRetrying, fmt.Sprintf("%d seconds", int(interval.Seconds())), lastError, deliveryID)
	} else {
		_, _ = webhookDB.Exec(ctx,
			`UPDATE webhook_deliveries SET status = $1, dlq_at = NOW(),
			last_error = $2, updated_at = NOW() WHERE id = $3`,
			StatusDLQ, lastError, deliveryID)
	}

	// Always return nil — retry scheduling is managed by DB + cron, not Pub/Sub.
	return nil
}
