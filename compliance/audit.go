package compliance

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/cron"
	"encore.dev/pubsub"
	"encore.dev/rlog"

	authpkg "encore.app/auth"
)

// AuditEvent represents an action to be recorded in the audit log.
type AuditEvent struct {
	OperatorID   int64           `json:"operator_id"`
	OperatorName string          `json:"operator_name"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	BeforeValue  json.RawMessage `json:"before_value,omitempty"`
	AfterValue   json.RawMessage `json:"after_value,omitempty"`
	IPAddress    string          `json:"ip_address"`
}

// AuditEvents is the Pub/Sub topic for async audit log writes.
var AuditEvents = pubsub.NewTopic[*AuditEvent]("audit-events", pubsub.TopicConfig{
	DeliveryGuarantee: pubsub.AtLeastOnce,
})

// Subscription to write audit events to the database.
var _ = pubsub.NewSubscription(AuditEvents, "write-audit-log",
	pubsub.SubscriptionConfig[*AuditEvent]{
		Handler: pubsub.MethodHandler((*Service).HandleAuditEvent),
	},
)

// HandleAuditEvent inserts an audit event into the database.
func (s *Service) HandleAuditEvent(ctx context.Context, event *AuditEvent) error {
	_, err := db.Exec(ctx, `
		INSERT INTO audit_logs (operator_id, operator_name, action, resource_type, resource_id, before_value, after_value, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.OperatorID, event.OperatorName, event.Action, event.ResourceType, event.ResourceID, event.BeforeValue, event.AfterValue, event.IPAddress)
	if err != nil {
		rlog.Error("failed to insert audit log", "err", err)
		return err
	}
	return nil
}

// PublishAuditEvent publishes an audit event to the topic for async processing.
//
//encore:api private method=POST path=/compliance/audit
func PublishAuditEvent(ctx context.Context, event *AuditEvent) error {
	_, err := AuditEvents.Publish(ctx, event)
	return err
}

// AuditLogEntry represents a single audit log row.
type AuditLogEntry struct {
	ID           int64           `json:"id"`
	OperatorID   int64           `json:"operator_id"`
	OperatorName string          `json:"operator_name"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	BeforeValue  json.RawMessage `json:"before_value,omitempty"`
	AfterValue   json.RawMessage `json:"after_value,omitempty"`
	IPAddress    string          `json:"ip_address"`
	CreatedAt    time.Time       `json:"created_at"`
}

// QueryAuditLogsParams defines filters for audit log queries.
type QueryAuditLogsParams struct {
	OperatorID   int64  `query:"operator_id"`
	Action       string `query:"action"`
	ResourceType string `query:"resource_type"`
	DateFrom     string `query:"date_from"`
	DateTo       string `query:"date_to"`
	Page         int    `query:"page"`
	PageSize     int    `query:"page_size"`
}

// QueryAuditLogsResponse contains paginated audit log results.
type QueryAuditLogsResponse struct {
	Logs       []*AuditLogEntry `json:"logs"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// QueryAuditLogs returns paginated, filtered audit logs. Admin only.
//
//encore:api auth method=GET path=/compliance/audit-logs
func QueryAuditLogs(ctx context.Context, p *QueryAuditLogsParams) (*QueryAuditLogsResponse, error) {
	data := authpkg.Data()
	if data == nil || data.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	page := max(p.Page, 1)
	pageSize := p.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if p.OperatorID != 0 {
		where += " AND operator_id = $" + strconv.Itoa(argIdx)
		args = append(args, p.OperatorID)
		argIdx++
	}
	if p.Action != "" {
		where += " AND action = $" + strconv.Itoa(argIdx)
		args = append(args, p.Action)
		argIdx++
	}
	if p.ResourceType != "" {
		where += " AND resource_type = $" + strconv.Itoa(argIdx)
		args = append(args, p.ResourceType)
		argIdx++
	}
	if p.DateFrom != "" {
		where += " AND created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, p.DateFrom)
		argIdx++
	}
	if p.DateTo != "" {
		where += " AND created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, p.DateTo)
		argIdx++
	}

	// Count total
	var total int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to count audit logs"}
	}

	// Fetch page
	offset := (page - 1) * pageSize
	queryArgs := append(args, pageSize, offset)
	rows, err := db.Query(ctx,
		"SELECT id, operator_id, operator_name, action, resource_type, resource_id, before_value, after_value, ip_address, created_at FROM audit_logs "+
			where+" ORDER BY created_at DESC LIMIT $"+strconv.Itoa(argIdx)+" OFFSET $"+strconv.Itoa(argIdx+1),
		queryArgs...)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to query audit logs"}
	}
	defer rows.Close()

	var logs []*AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		if err := rows.Scan(&e.ID, &e.OperatorID, &e.OperatorName, &e.Action, &e.ResourceType, &e.ResourceID, &e.BeforeValue, &e.AfterValue, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to scan audit log"}
		}
		logs = append(logs, &e)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return &QueryAuditLogsResponse{
		Logs:       logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Cleanup cron: delete audit logs older than 90 days.
var _ = cron.NewJob("audit-log-cleanup", cron.JobConfig{
	Title:    "Clean up old audit logs",
	Schedule: "0 3 * * *",
	Endpoint: CleanupAuditLogs,
})

//encore:api private
func CleanupAuditLogs(ctx context.Context) error {
	result, err := db.Exec(ctx, `DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days'`)
	if err != nil {
		rlog.Error("audit log cleanup failed", "err", err)
		return err
	}
	count := result.RowsAffected()
	if count > 0 {
		rlog.Info("audit log cleanup complete", "deleted", count)
	}
	return nil
}
