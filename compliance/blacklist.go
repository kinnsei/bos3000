package compliance

import (
	"context"
	"fmt"
	"strconv"

	"encore.dev/beta/errs"

	authpkg "encore.app/auth"
	"encore.app/pkg/errcode"
)

// CheckBlacklistParams contains the parameters for checking if a number is blacklisted.
type CheckBlacklistParams struct {
	CalledNumber string `json:"called_number"`
	UserID       int64  `json:"user_id"`
}

// CheckBlacklist checks if a number is globally or client-level blacklisted.
//
//encore:api private method=POST path=/compliance/check-blacklist
func (s *Service) CheckBlacklist(ctx context.Context, p *CheckBlacklistParams) error {
	var globalHit bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM blacklisted_numbers WHERE number = $1 AND user_id IS NULL)
	`, p.CalledNumber).Scan(&globalHit)
	if err != nil {
		return fmt.Errorf("check global blacklist: %w", err)
	}
	if globalHit {
		return errcode.NewError(errs.FailedPrecondition, errcode.BlacklistedNumber,
			"number is globally blacklisted")
	}

	var clientHit bool
	err = db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM blacklisted_numbers WHERE number = $1 AND user_id = $2)
	`, p.CalledNumber, p.UserID).Scan(&clientHit)
	if err != nil {
		return fmt.Errorf("check client blacklist: %w", err)
	}
	if clientHit {
		return errcode.NewError(errs.FailedPrecondition, errcode.BlacklistedNumber,
			"number is blacklisted for this account")
	}

	return nil
}

// AddBlacklistParams contains the parameters for adding a number to the blacklist.
type AddBlacklistParams struct {
	Number string `json:"number"`
	UserID *int64 `json:"user_id"`
	Reason string `json:"reason"`
}

// AddBlacklistResponse contains the ID of the created blacklist entry.
type AddBlacklistResponse struct {
	ID int64 `json:"id"`
}

// AddBlacklist adds a number to the blacklist. Admins can add global or client-level entries.
// Clients can only add entries for their own account.
//
//encore:api auth method=POST path=/compliance/blacklist
func (s *Service) AddBlacklist(ctx context.Context, p *AddBlacklistParams) (*AddBlacklistResponse, error) {
	ad := authpkg.Data()
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	if ad.Role != "admin" {
		// Clients can only add to their own blacklist
		if p.UserID == nil {
			return nil, &errs.Error{Code: errs.PermissionDenied, Message: "only admins can add global blacklist entries"}
		}
		if *p.UserID != ad.UserID {
			return nil, &errs.Error{Code: errs.PermissionDenied, Message: "can only add blacklist entries for your own account"}
		}
	}

	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO blacklisted_numbers (number, user_id, reason, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, p.Number, p.UserID, p.Reason, ad.UserID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert blacklist: %w", err)
	}

	return &AddBlacklistResponse{ID: id}, nil
}

// RemoveBlacklist removes a blacklist entry by ID. Admins can remove any entry;
// clients can only remove their own entries.
//
//encore:api auth method=DELETE path=/compliance/blacklist/:id
func (s *Service) RemoveBlacklist(ctx context.Context, id int64) error {
	ad := authpkg.Data()
	if ad == nil {
		return &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	var query string
	var args []any
	if ad.Role == "admin" {
		query = `DELETE FROM blacklisted_numbers WHERE id = $1`
		args = []any{id}
	} else {
		query = `DELETE FROM blacklisted_numbers WHERE id = $1 AND created_by = $2`
		args = []any{id, ad.UserID}
	}

	result, err := db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete blacklist: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "blacklist entry not found"}
	}
	return nil
}

// ListBlacklistParams contains the query parameters for listing blacklist entries.
type ListBlacklistParams struct {
	UserID     int64 `query:"user_id"`     // 0 means no filter
	GlobalOnly bool  `query:"global_only"`
}

// BlacklistEntry represents a single blacklist entry.
type BlacklistEntry struct {
	ID        int64  `json:"id"`
	Number    string `json:"number"`
	UserID    *int64 `json:"user_id"`
	Reason    string `json:"reason"`
	CreatedBy int64  `json:"created_by"`
}

// ListBlacklistResponse contains the list of blacklist entries.
type ListBlacklistResponse struct {
	Entries []BlacklistEntry `json:"entries"`
}

// ListBlacklist lists blacklist entries. Admins see all; clients see own + global.
//
//encore:api auth method=GET path=/compliance/blacklist
func (s *Service) ListBlacklist(ctx context.Context, p *ListBlacklistParams) (*ListBlacklistResponse, error) {
	ad := authpkg.Data()
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	var query string
	var args []any
	argIdx := 1

	if ad.Role == "admin" {
		if p.GlobalOnly {
			query = `SELECT id, number, user_id, reason, created_by FROM blacklisted_numbers WHERE user_id IS NULL ORDER BY id`
		} else if p.UserID != 0 {
			query = `SELECT id, number, user_id, reason, created_by FROM blacklisted_numbers WHERE user_id IS NULL OR user_id = $` + strconv.Itoa(argIdx) + ` ORDER BY id`
			args = append(args, p.UserID)
		} else {
			query = `SELECT id, number, user_id, reason, created_by FROM blacklisted_numbers ORDER BY id`
		}
	} else {
		// Client sees own + global
		query = `SELECT id, number, user_id, reason, created_by FROM blacklisted_numbers WHERE user_id IS NULL OR user_id = $` + strconv.Itoa(argIdx) + ` ORDER BY id`
		args = append(args, ad.UserID)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query blacklist: %w", err)
	}
	defer rows.Close()

	var entries []BlacklistEntry
	for rows.Next() {
		var e BlacklistEntry
		if err := rows.Scan(&e.ID, &e.Number, &e.UserID, &e.Reason, &e.CreatedBy); err != nil {
			return nil, fmt.Errorf("scan blacklist: %w", err)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []BlacklistEntry{}
	}

	return &ListBlacklistResponse{Entries: entries}, nil
}
