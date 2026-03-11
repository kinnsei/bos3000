package webhook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
)

// ConfigureWebhookParams contains the webhook URL to set.
type ConfigureWebhookParams struct {
	WebhookURL string `json:"webhook_url"`
}

// ConfigureWebhookResponse contains the configured webhook details.
type ConfigureWebhookResponse struct {
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

// ConfigureWebhook sets or clears the user's webhook URL.
//
//encore:api auth method=PUT path=/webhooks/config
func (s *Service) ConfigureWebhook(ctx context.Context, p *ConfigureWebhookParams) (*ConfigureWebhookResponse, error) {
	ad := authpkg.Data()
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	if p.WebhookURL == "" {
		_, _ = authDB.Exec(ctx,
			"UPDATE users SET webhook_url = NULL WHERE id = $1", ad.UserID)
		return &ConfigureWebhookResponse{}, nil
	}

	if len(p.WebhookURL) < 9 || p.WebhookURL[:8] != "https://" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "webhook_url must start with https://"}
	}

	// Check if user already has a secret.
	var existingSecret *string
	_ = authDB.QueryRow(ctx,
		"SELECT webhook_secret FROM users WHERE id = $1", ad.UserID,
	).Scan(&existingSecret)

	secret := ""
	if existingSecret == nil || *existingSecret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		secret = hex.EncodeToString(b)
		_, err := authDB.Exec(ctx,
			"UPDATE users SET webhook_url = $1, webhook_secret = $2 WHERE id = $3",
			p.WebhookURL, secret, ad.UserID)
		if err != nil {
			return nil, err
		}
	} else {
		secret = *existingSecret
		_, err := authDB.Exec(ctx,
			"UPDATE users SET webhook_url = $1 WHERE id = $2",
			p.WebhookURL, ad.UserID)
		if err != nil {
			return nil, err
		}
	}

	return &ConfigureWebhookResponse{
		WebhookURL:    p.WebhookURL,
		WebhookSecret: secret,
	}, nil
}

// DeliverySummary is a condensed delivery record for list views.
type DeliverySummary struct {
	ID           int64      `json:"id"`
	CallID       string     `json:"call_id"`
	EventType    string     `json:"event_type"`
	Status       string     `json:"status"`
	AttemptCount int        `json:"attempt_count"`
	LastError    *string    `json:"last_error,omitempty"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ListDeliveriesParams contains query parameters for listing deliveries.
type ListDeliveriesParams struct {
	Page     int `query:"page"`
	PageSize int `query:"page_size"`
}

// ListDeliveriesResponse contains paginated delivery results.
type ListDeliveriesResponse struct {
	Deliveries []DeliverySummary `json:"deliveries"`
	Total      int               `json:"total"`
}

// ListDeliveries returns the user's webhook delivery history.
//
//encore:api auth method=GET path=/webhooks/deliveries
func (s *Service) ListDeliveries(ctx context.Context, p *ListDeliveriesParams) (*ListDeliveriesResponse, error) {
	ad := authpkg.Data()
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	page := max(p.Page, 1)
	pageSize := p.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := webhookDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM webhook_deliveries WHERE user_id = $1", ad.UserID,
	).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := webhookDB.Query(ctx,
		`SELECT id, call_id, event_type, status, attempt_count, last_error, delivered_at, created_at
		FROM webhook_deliveries WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, ad.UserID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deliveries := []DeliverySummary{}
	for rows.Next() {
		var d DeliverySummary
		if err := rows.Scan(&d.ID, &d.CallID, &d.EventType, &d.Status,
			&d.AttemptCount, &d.LastError, &d.DeliveredAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}

	return &ListDeliveriesResponse{Deliveries: deliveries, Total: total}, nil
}

// ResetWebhookSecret generates a new webhook secret for the user.
//
//encore:api auth method=POST path=/webhooks/secret/reset
func (s *Service) ResetWebhookSecret(ctx context.Context) (*ConfigureWebhookResponse, error) {
	ad := authpkg.Data()
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	newSecret := hex.EncodeToString(b)

	var webhookURL *string
	err := authDB.QueryRow(ctx,
		"UPDATE users SET webhook_secret = $1 WHERE id = $2 RETURNING webhook_url",
		newSecret, ad.UserID,
	).Scan(&webhookURL)
	if err != nil {
		return nil, err
	}

	url := ""
	if webhookURL != nil {
		url = *webhookURL
	}

	return &ConfigureWebhookResponse{
		WebhookURL:    url,
		WebhookSecret: newSecret,
	}, nil
}
