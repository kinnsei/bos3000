package webhook

import (
	"context"
	"time"

	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
)

// DLQEntry represents a dead-letter queue entry.
type DLQEntry struct {
	ID           int64     `json:"id"`
	CallID       string    `json:"call_id"`
	UserID       int64     `json:"user_id"`
	EventType    string    `json:"event_type"`
	WebhookURL   string    `json:"webhook_url"`
	AttemptCount int       `json:"attempt_count"`
	LastError    *string   `json:"last_error,omitempty"`
	DLQAt        time.Time `json:"dlq_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListDLQParams contains query parameters for listing DLQ entries.
type ListDLQParams struct {
	Page     int `query:"page"`
	PageSize int `query:"page_size"`
}

// ListDLQResponse contains paginated DLQ results.
type ListDLQResponse struct {
	Items    []DLQEntry `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// ListDLQ returns paginated dead-letter queue entries (admin only).
//
//encore:api auth method=GET path=/admin/webhooks/dlq
func (s *Service) ListDLQ(ctx context.Context, p *ListDLQParams) (*ListDLQResponse, error) {
	ad := authpkg.Data()
	if ad == nil || ad.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	page := max(p.Page, 1)
	pageSize := p.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := webhookDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM webhook_deliveries WHERE status = 'dlq'",
	).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := webhookDB.Query(ctx,
		`SELECT id, call_id, user_id, event_type, webhook_url, attempt_count,
			last_error, dlq_at, created_at
		FROM webhook_deliveries WHERE status = 'dlq'
		ORDER BY dlq_at DESC LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []DLQEntry{}
	for rows.Next() {
		var e DLQEntry
		if err := rows.Scan(&e.ID, &e.CallID, &e.UserID, &e.EventType, &e.WebhookURL,
			&e.AttemptCount, &e.LastError, &e.DLQAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}

	return &ListDLQResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// RetryDLQ manually retries a DLQ entry (admin only).
//
//encore:api auth method=POST path=/admin/webhooks/dlq/:id/retry
func (s *Service) RetryDLQ(ctx context.Context, id int64) error {
	ad := authpkg.Data()
	if ad == nil || ad.Role != "admin" {
		return &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	var webhookURL string
	var payload []byte
	var userID int64
	var status string
	err := webhookDB.QueryRow(ctx,
		`SELECT webhook_url, payload, user_id, status FROM webhook_deliveries WHERE id = $1`, id,
	).Scan(&webhookURL, &payload, &userID, &status)
	if err != nil {
		return &errs.Error{Code: errs.NotFound, Message: "delivery not found"}
	}
	if status != StatusDLQ {
		return &errs.Error{Code: errs.FailedPrecondition, Message: "delivery is not in DLQ"}
	}

	// Reset for retry.
	_, _ = webhookDB.Exec(ctx,
		`UPDATE webhook_deliveries SET status = $1, attempt_count = 0,
		dlq_retry_count = dlq_retry_count + 1, dlq_at = NULL,
		next_retry_at = NOW(), updated_at = NOW() WHERE id = $2`,
		StatusPending, id)

	// Fetch current webhook secret.
	var secret string
	_ = authDB.QueryRow(ctx,
		"SELECT COALESCE(webhook_secret, '') FROM users WHERE id = $1", userID,
	).Scan(&secret)

	_, err = WebhookTopic.Publish(ctx, &WebhookEvent{
		DeliveryID: id,
		WebhookURL: webhookURL,
		Secret:     secret,
		Payload:    payload,
	})
	return err
}
