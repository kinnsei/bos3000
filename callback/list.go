package callback

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"encore.dev/beta/errs"
)

// ListCallbacksParams contains query parameters for listing callbacks.
type ListCallbacksParams struct {
	Status      string `query:"status"`
	ANumber     string `query:"a_number"`
	BNumber     string `query:"b_number"`
	WastageType string `query:"wastage_type"`
	UserID      int64  `query:"user_id"`
	StartTime   string `query:"start_time"`
	EndTime     string `query:"end_time"`
	Page        int    `query:"page"`
	PageSize    int    `query:"page_size"`
}

// CallSummary is a condensed version of a call for list views.
type CallSummary struct {
	CallID          string          `json:"call_id"`
	UserID          int64           `json:"user_id"`
	ANumber         string          `json:"a_number"`
	BNumber         string          `json:"b_number"`
	Status          string          `json:"status"`
	BridgeDurationMs int64          `json:"bridge_duration_ms"`
	TotalCost       int64           `json:"total_cost"`
	WastageType     *string         `json:"wastage_type,omitempty"`
	HangupBy        *string         `json:"hangup_by,omitempty"`
	FailureReason   *string         `json:"failure_reason,omitempty"`
	CustomData      json.RawMessage `json:"custom_data,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ListCallbacksResponse contains paginated callback results.
type ListCallbacksResponse struct {
	Items    []CallSummary `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// ListCallbacks returns a paginated list of callback calls (CDR).
//
//encore:api auth method=GET path=/callbacks
func (s *Service) ListCallbacks(ctx context.Context, p *ListCallbacksParams) (*ListCallbacksResponse, error) {
	ad := getAuthData(ctx)
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	page := max(p.Page, 1)
	pageSize := p.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build dynamic WHERE
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	// Client isolation: force user_id filter
	if ad.Role != "admin" {
		where += " AND user_id = $" + strconv.Itoa(argIdx)
		args = append(args, ad.UserID)
		argIdx++
	} else if p.UserID != 0 {
		where += " AND user_id = $" + strconv.Itoa(argIdx)
		args = append(args, p.UserID)
		argIdx++
	}

	if p.Status != "" {
		where += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, p.Status)
		argIdx++
	}
	if p.ANumber != "" {
		where += " AND a_number = $" + strconv.Itoa(argIdx)
		args = append(args, p.ANumber)
		argIdx++
	}
	if p.BNumber != "" {
		where += " AND b_number = $" + strconv.Itoa(argIdx)
		args = append(args, p.BNumber)
		argIdx++
	}
	if p.WastageType != "" {
		where += " AND wastage_type = $" + strconv.Itoa(argIdx)
		args = append(args, p.WastageType)
		argIdx++
	}
	if p.StartTime != "" {
		where += " AND created_at >= $" + strconv.Itoa(argIdx)
		args = append(args, p.StartTime)
		argIdx++
	}
	if p.EndTime != "" {
		where += " AND created_at <= $" + strconv.Itoa(argIdx)
		args = append(args, p.EndTime)
		argIdx++
	}

	// Count total
	var total int
	err := db.QueryRow(ctx,
		"SELECT COUNT(*) FROM callback_calls "+where, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Fetch page
	query := `SELECT call_id, user_id, a_number, b_number, status,
		bridge_duration_ms, total_cost, wastage_type, hangup_by, failure_reason,
		custom_data, created_at
		FROM callback_calls ` + where +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIdx) +
		` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CallSummary
	for rows.Next() {
		var item CallSummary
		if err := rows.Scan(
			&item.CallID, &item.UserID, &item.ANumber, &item.BNumber, &item.Status,
			&item.BridgeDurationMs, &item.TotalCost, &item.WastageType,
			&item.HangupBy, &item.FailureReason,
			&item.CustomData, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []CallSummary{}
	}

	return &ListCallbacksResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
