package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	"encore.dev/pubsub"
	"encore.dev/storage/sqldb"
)

var webhookDB = sqldb.NewDatabase("webhook", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

// WebhookTopic triggers webhook delivery workers.
var WebhookTopic = pubsub.NewTopic[*WebhookEvent]("webhook-delivery", pubsub.TopicConfig{
	DeliveryGuarantee: pubsub.AtLeastOnce,
})

//encore:service
type Service struct{}

func initService() (*Service, error) {
	return &Service{}, nil
}

// CreateAndPublishWebhook creates a delivery record and publishes it for processing.
func CreateAndPublishWebhook(ctx context.Context, callID string, userID int64, webhookURL, secret string, payload *WebhookPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	var deliveryID int64
	err = webhookDB.QueryRow(ctx,
		`INSERT INTO webhook_deliveries (call_id, user_id, event_type, status, webhook_url, payload, next_retry_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id`,
		callID, userID, payload.EventType, StatusPending, webhookURL, data,
	).Scan(&deliveryID)
	if err != nil {
		return fmt.Errorf("insert webhook delivery: %w", err)
	}

	_, err = WebhookTopic.Publish(ctx, &WebhookEvent{
		DeliveryID: deliveryID,
		WebhookURL: webhookURL,
		Secret:     secret,
		Payload:    data,
	})
	if err != nil {
		return fmt.Errorf("publish webhook event: %w", err)
	}
	return nil
}
