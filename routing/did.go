package routing

import (
	"context"
	"strconv"
	"strings"
	"time"

	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
	"encore.app/pkg/errcode"
)

// --- SelectDID: user pool > public pool priority ---

type SelectDIDParams struct {
	UserID int64 `json:"user_id"`
}

type SelectDIDResponse struct {
	Number string `json:"number"`
	DIDID  int64  `json:"did_id"`
}

//encore:api private method=POST path=/routing/select-did
func (s *Service) SelectDID(ctx context.Context, p *SelectDIDParams) (*SelectDIDResponse, error) {
	// Step 1: Try user's dedicated pool
	var resp SelectDIDResponse
	err := db.QueryRow(ctx, `
		SELECT id, number FROM did_numbers
		WHERE user_id = $1 AND status = 'available'
		ORDER BY RANDOM() LIMIT 1
	`, p.UserID).Scan(&resp.DIDID, &resp.Number)
	if err == nil {
		return &resp, nil
	}

	// Step 2: Fallback to public pool (user_id IS NULL)
	err = db.QueryRow(ctx, `
		SELECT id, number FROM did_numbers
		WHERE user_id IS NULL AND status = 'available'
		ORDER BY RANDOM() LIMIT 1
	`).Scan(&resp.DIDID, &resp.Number)
	if err == nil {
		return &resp, nil
	}

	return nil, errcode.NewError(errs.ResourceExhausted, "NO_DID_AVAILABLE", "no DID numbers available")
}

// --- ImportDIDs: bulk import ---

type ImportDIDsParams struct {
	Numbers []string `json:"numbers"`
	UserID  *int64   `json:"user_id,omitempty"`
}

type ImportDIDsResponse struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

//encore:api auth method=POST path=/routing/did-import
func (s *Service) ImportDIDs(ctx context.Context, p *ImportDIDsParams) (*ImportDIDsResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	if len(p.Numbers) == 0 {
		return &ImportDIDsResponse{}, nil
	}

	// Build bulk INSERT with ON CONFLICT DO NOTHING
	var sb strings.Builder
	sb.WriteString("INSERT INTO did_numbers (number, user_id) VALUES ")
	args := make([]any, 0, len(p.Numbers)+1)
	args = append(args, p.UserID) // $1

	for i, num := range p.Numbers {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("($" + strconv.Itoa(i+2) + ", $1)")
		args = append(args, num)
	}
	sb.WriteString(" ON CONFLICT (number) DO NOTHING")

	result, err := db.Exec(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}

	imported := int(result.RowsAffected())
	return &ImportDIDsResponse{
		Imported: imported,
		Skipped:  len(p.Numbers) - imported,
	}, nil
}

// --- AssignDID ---

type AssignDIDParams struct {
	UserID int64 `json:"user_id"`
}

type AssignDIDResponse struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`
}

//encore:api auth method=POST path=/routing/dids/:id/assign
func (s *Service) AssignDID(ctx context.Context, id int64, p *AssignDIDParams) (*AssignDIDResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	result, err := db.Exec(ctx, `
		UPDATE did_numbers SET user_id = $1 WHERE id = $2
	`, p.UserID, id)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() == 0 {
		return nil, &errs.Error{Code: errs.NotFound, Message: "DID not found"}
	}

	return &AssignDIDResponse{ID: id, UserID: p.UserID}, nil
}

// --- UnassignDID ---

type UnassignDIDResponse struct {
	ID int64 `json:"id"`
}

//encore:api auth method=POST path=/routing/dids/:id/unassign
func (s *Service) UnassignDID(ctx context.Context, id int64) (*UnassignDIDResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	result, err := db.Exec(ctx, `
		UPDATE did_numbers SET user_id = NULL WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() == 0 {
		return nil, &errs.Error{Code: errs.NotFound, Message: "DID not found"}
	}

	return &UnassignDIDResponse{ID: id}, nil
}

// --- ListDIDs ---

type ListDIDsParams struct {
	UserID int64  `query:"user_id"`
	Status string `query:"status"`
	Page   int    `query:"page"`
}

type DIDItem struct {
	ID        int64     `json:"id"`
	Number    string    `json:"number"`
	UserID    *int64    `json:"user_id,omitempty"`
	Status    string    `json:"status"`
	Region    *string   `json:"region,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ListDIDsResponse struct {
	DIDs       []*DIDItem `json:"dids"`
	Page       int        `json:"page"`
	TotalCount int        `json:"total_count"`
}

//encore:api auth method=GET path=/routing/dids
func (s *Service) ListDIDs(ctx context.Context, p *ListDIDsParams) (*ListDIDsResponse, error) {
	if err := requireAdmin(); err != nil {
		return nil, err
	}

	pageSize := 50
	page := max(p.Page, 1)
	offset := (page - 1) * pageSize

	// Build dynamic WHERE clause
	where := []string{}
	args := []any{}
	argIdx := 1

	if p.UserID != 0 {
		where = append(where, "user_id = $"+strconv.Itoa(argIdx))
		args = append(args, p.UserID)
		argIdx++
	}
	if p.Status != "" {
		where = append(where, "status = $"+strconv.Itoa(argIdx))
		args = append(args, p.Status)
		argIdx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	// Count total
	var total int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM did_numbers"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Fetch page
	query := "SELECT id, number, user_id, status, region, created_at FROM did_numbers" +
		whereClause + " ORDER BY id LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dids []*DIDItem
	for rows.Next() {
		d := &DIDItem{}
		if err := rows.Scan(&d.ID, &d.Number, &d.UserID, &d.Status, &d.Region, &d.CreatedAt); err != nil {
			return nil, err
		}
		dids = append(dids, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ListDIDsResponse{
		DIDs:       dids,
		Page:       page,
		TotalCount: total,
	}, nil
}

// --- MyDIDs (portal: get current user's assigned DIDs) ---

type MyDIDsResponse struct {
	DIDs []string `json:"dids"`
}

//encore:api auth method=GET path=/routing/my-dids
func (s *Service) MyDIDs(ctx context.Context) (*MyDIDsResponse, error) {
	data := authpkg.Data()
	if data == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	rows, err := db.Query(ctx, `
		SELECT number FROM did_numbers
		WHERE user_id = $1 AND status = 'assigned'
		ORDER BY number
	`, data.UserID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to list DIDs"}
	}
	defer rows.Close()

	var dids []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to scan DID"}
		}
		dids = append(dids, n)
	}
	if dids == nil {
		dids = []string{}
	}

	return &MyDIDsResponse{DIDs: dids}, nil
}
