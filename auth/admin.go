package auth

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"golang.org/x/crypto/bcrypt"
)

// ---- GET /auth/me ----

// MeResponse contains the current authenticated user's info.
type MeResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

//encore:api auth method=GET path=/auth/me
func (s *Service) Me(ctx context.Context) (*MeResponse, error) {
	data := Data()

	var resp MeResponse
	err := db.QueryRow(ctx, `
		SELECT id, username, email, role, status
		FROM users WHERE id = $1
	`, data.UserID).Scan(&resp.UserID, &resp.Username, &resp.Email, &resp.Role, &resp.Status)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to fetch user"}
	}

	return &resp, nil
}

// ---- GET /auth/admin/users ----

// ListUsersParams are the query parameters for listing users.
type ListUsersParams struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
	Status string `query:"status"`
}

// UserSummary is a single user in a list response.
type UserSummary struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	Balance       int64     `json:"balance"`
	CreditLimit   int64     `json:"credit_limit"`
	MaxConcurrent int       `json:"max_concurrent"`
	DailyLimit    int       `json:"daily_limit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ListUsersResponse wraps a paginated list of users.
type ListUsersResponse struct {
	Users []UserSummary `json:"users"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

//encore:api auth method=GET path=/auth/admin/users
func (s *Service) ListUsers(ctx context.Context, p *ListUsersParams) (*ListUsersResponse, error) {
	data := Data()
	if data.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	limit := p.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	page := p.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	search := "%" + p.Search + "%"

	// Count total matching rows.
	var total int
	if p.Status != "" {
		err := db.QueryRow(ctx, `
			SELECT COUNT(*) FROM users
			WHERE (email ILIKE $1 OR username ILIKE $1) AND status = $2
		`, search, p.Status).Scan(&total)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to count users"}
		}
	} else {
		err := db.QueryRow(ctx, `
			SELECT COUNT(*) FROM users
			WHERE email ILIKE $1 OR username ILIKE $1
		`, search).Scan(&total)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to count users"}
		}
	}

	// Fetch page of users.
	users, err := s.queryUserSummaries(ctx, search, p.Status, limit, offset)
	if err != nil {
		return nil, err
	}

	return &ListUsersResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func scanUserSummaries(rows *sqldb.Rows) ([]UserSummary, error) {
	defer rows.Close()
	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.Role, &u.Status,
			&u.Balance, &u.CreditLimit, &u.MaxConcurrent, &u.DailyLimit,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to scan user"}
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to iterate users"}
	}
	if users == nil {
		users = []UserSummary{}
	}
	return users, nil
}

func (s *Service) queryUserSummaries(ctx context.Context, search, status string, limit, offset int) ([]UserSummary, error) {
	if status != "" {
		rows, err := db.Query(ctx, `
			SELECT id, email, username, role, status, balance, credit_limit,
			       max_concurrent, daily_limit, created_at, updated_at
			FROM users
			WHERE (email ILIKE $1 OR username ILIKE $1) AND status = $2
			ORDER BY id ASC
			LIMIT $3 OFFSET $4
		`, search, status, limit, offset)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to query users"}
		}
		return scanUserSummaries(rows)
	}

	rows, err := db.Query(ctx, `
		SELECT id, email, username, role, status, balance, credit_limit,
		       max_concurrent, daily_limit, created_at, updated_at
		FROM users
		WHERE email ILIKE $1 OR username ILIKE $1
		ORDER BY id ASC
		LIMIT $2 OFFSET $3
	`, search, limit, offset)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to query users"}
	}
	return scanUserSummaries(rows)
}

// ---- GET /auth/admin/users/:id ----

// UserDetail is the full user record returned by GetUser.
type UserDetail struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	Balance       int64     `json:"balance"`
	CreditLimit   int64     `json:"credit_limit"`
	MaxConcurrent int       `json:"max_concurrent"`
	DailyLimit    int       `json:"daily_limit"`
	RatePlanID    *int64    `json:"rate_plan_id"`
	ALegRate      *int64    `json:"a_leg_rate"`
	BLegRate      *int64    `json:"b_leg_rate"`
	WebhookURL    string    `json:"webhook_url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

//encore:api auth method=GET path=/auth/admin/users/:id
func (s *Service) GetUser(ctx context.Context, id int64) (*UserDetail, error) {
	data := Data()
	if data.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	var u UserDetail
	err := db.QueryRow(ctx, `
		SELECT id, email, username, role, status, balance, credit_limit,
		       max_concurrent, daily_limit, rate_plan_id, a_leg_rate, b_leg_rate,
		       COALESCE(webhook_url, ''), created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.Role, &u.Status,
		&u.Balance, &u.CreditLimit, &u.MaxConcurrent, &u.DailyLimit,
		&u.RatePlanID, &u.ALegRate, &u.BLegRate,
		&u.WebhookURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "user not found"}
	}

	return &u, nil
}

// ---- POST /auth/admin/users ----

// CreateUserRequest is the request body for creating a new client user.
type CreateUserRequest struct {
	Email         string `json:"email"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	CreditLimit   int64  `json:"credit_limit"`
	MaxConcurrent int    `json:"max_concurrent"`
	DailyLimit    int    `json:"daily_limit"`
	RatePlanID    *int64 `json:"rate_plan_id"`
	ALegRate      *int64 `json:"a_leg_rate"`
	BLegRate      *int64 `json:"b_leg_rate"`
	WebhookURL    string `json:"webhook_url"`
}

func (r *CreateUserRequest) Validate() error {
	if r.Email == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "email is required"}
	}
	if r.Username == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "username is required"}
	}
	if r.Password == "" || len(r.Password) < 8 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "password must be at least 8 characters"}
	}
	return nil
}

// CreateUserResponse is the response body after creating a user.
type CreateUserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

//encore:api auth method=POST path=/auth/admin/users
func (s *Service) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	data := Data()
	if data.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to hash password"}
	}

	var id int64
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, role, balance, credit_limit,
		                    max_concurrent, daily_limit, rate_plan_id, a_leg_rate, b_leg_rate,
		                    webhook_url, status)
		VALUES ($1, $2, $3, 'client', 0, $4, $5, $6, $7, $8, $9, $10, 'active')
		RETURNING id
	`, req.Email, req.Username, string(hash),
		req.CreditLimit, req.MaxConcurrent, req.DailyLimit,
		req.RatePlanID, req.ALegRate, req.BLegRate, req.WebhookURL,
	).Scan(&id)
	if err != nil {
		return nil, &errs.Error{Code: errs.AlreadyExists, Message: "user with this email already exists"}
	}

	return &CreateUserResponse{
		ID:       id,
		Email:    req.Email,
		Username: req.Username,
	}, nil
}

// ---- POST /auth/admin/users/:id/freeze ----

//encore:api auth method=POST path=/auth/admin/users/:id/freeze
func (s *Service) FreezeUser(ctx context.Context, id int64) error {
	data := Data()
	if data.Role != "admin" {
		return &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	result, err := db.Exec(ctx, `
		UPDATE users SET status = 'frozen', updated_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, id)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to freeze user"}
	}
	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "user not found or not in active status"}
	}

	return nil
}

// ---- POST /auth/admin/users/:id/unfreeze ----

//encore:api auth method=POST path=/auth/admin/users/:id/unfreeze
func (s *Service) UnfreezeUser(ctx context.Context, id int64) error {
	data := Data()
	if data.Role != "admin" {
		return &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	result, err := db.Exec(ctx, `
		UPDATE users SET status = 'active', updated_at = NOW()
		WHERE id = $1 AND status = 'frozen'
	`, id)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to unfreeze user"}
	}
	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "user not found or not in frozen status"}
	}

	return nil
}
