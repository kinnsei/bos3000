package webhook

import (
	"context"
	"time"

	"encore.dev/cron"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
)

// RetryIntervals defines the exact backoff schedule per CONTEXT.md decision.
// Encore Pub/Sub RetryPolicy uses automatic exponential doubling which cannot
// match these exact intervals, so we implement custom retry via cron polling.
var RetryIntervals = []time.Duration{
	30 * time.Second, // attempt 1
	1 * time.Minute,  // attempt 2
	5 * time.Minute,  // attempt 3
	15 * time.Minute, // attempt 4
	1 * time.Hour,    // attempt 5
}

func retryIntervalForAttempt(attempt int) time.Duration {
	if attempt < 1 || attempt > len(RetryIntervals) {
		return 0 // signals DLQ
	}
	return RetryIntervals[attempt-1]
}

var authDB = sqldb.Named("auth")

var _ = cron.NewJob("webhook-retry-poll", cron.JobConfig{
	Title:    "Poll webhook deliveries due for retry",
	Every:    1 * cron.Minute,
	Endpoint: PollRetries,
})

// PollRetries checks for webhook deliveries due for retry and re-publishes them.
//
//encore:api private
func PollRetries(ctx context.Context) error {
	rows, err := webhookDB.Query(ctx,
		`SELECT id, webhook_url, payload, user_id
		FROM webhook_deliveries
		WHERE status = 'retrying' AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC
		LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, userID int64
		var webhookURL string
		var payload []byte

		if err := rows.Scan(&id, &webhookURL, &payload, &userID); err != nil {
			rlog.Warn("scan retry row", "error", err)
			continue
		}

		// Look up user's current webhook secret.
		var secret string
		err := authDB.QueryRow(ctx,
			"SELECT COALESCE(webhook_secret, '') FROM users WHERE id = $1", userID,
		).Scan(&secret)
		if err != nil {
			rlog.Warn("lookup webhook secret for retry", "user_id", userID, "error", err)
			continue
		}

		_, err = WebhookTopic.Publish(ctx, &WebhookEvent{
			DeliveryID: id,
			WebhookURL: webhookURL,
			Secret:     secret,
			Payload:    payload,
		})
		if err != nil {
			rlog.Warn("re-publish webhook for retry", "delivery_id", id, "error", err)
			continue
		}

		_, _ = webhookDB.Exec(ctx,
			"UPDATE webhook_deliveries SET status = $1, updated_at = NOW() WHERE id = $2",
			StatusPending, id)
		count++
	}

	if count > 0 {
		rlog.Info("re-published webhook deliveries for retry", "count", count)
	}
	return nil
}
