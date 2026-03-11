package auth

import (
	"context"

	"encore.dev/beta/errs"
	"golang.org/x/crypto/bcrypt"
)

// ---- PUT /auth/profile ----

// UpdateProfileRequest is the request body for updating the current user's profile.
type UpdateProfileRequest struct {
	Username   *string `json:"username"`
	WebhookURL *string `json:"webhook_url"`
}

// UpdateProfileResponse contains the updated profile fields.
type UpdateProfileResponse struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url"`
}

//encore:api auth method=PUT path=/auth/profile
func (s *Service) UpdateProfile(ctx context.Context, req *UpdateProfileRequest) (*UpdateProfileResponse, error) {
	data := Data()

	// Build SET clause dynamically based on provided fields.
	if req.Username == nil && req.WebhookURL == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "at least one field must be provided"}
	}

	var resp UpdateProfileResponse

	if req.Username != nil && req.WebhookURL != nil {
		err := db.QueryRow(ctx, `
			UPDATE users SET username = $1, webhook_url = $2, updated_at = NOW()
			WHERE id = $3
			RETURNING username, email, webhook_url
		`, *req.Username, *req.WebhookURL, data.UserID).Scan(&resp.Username, &resp.Email, &resp.WebhookURL)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to update profile"}
		}
	} else if req.Username != nil {
		err := db.QueryRow(ctx, `
			UPDATE users SET username = $1, updated_at = NOW()
			WHERE id = $2
			RETURNING username, email, webhook_url
		`, *req.Username, data.UserID).Scan(&resp.Username, &resp.Email, &resp.WebhookURL)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to update profile"}
		}
	} else {
		err := db.QueryRow(ctx, `
			UPDATE users SET webhook_url = $1, updated_at = NOW()
			WHERE id = $2
			RETURNING username, email, webhook_url
		`, *req.WebhookURL, data.UserID).Scan(&resp.Username, &resp.Email, &resp.WebhookURL)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to update profile"}
		}
	}

	return &resp, nil
}

// ---- POST /auth/profile/password ----

// ChangePasswordRequest is the request body for changing the current user's password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (r *ChangePasswordRequest) Validate() error {
	if r.CurrentPassword == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "current_password is required"}
	}
	if r.NewPassword == "" || len(r.NewPassword) < 8 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "new_password must be at least 8 characters"}
	}
	if r.CurrentPassword == r.NewPassword {
		return &errs.Error{Code: errs.InvalidArgument, Message: "new password must be different from current password"}
	}
	return nil
}

//encore:api auth method=POST path=/auth/profile/password
func (s *Service) ChangePassword(ctx context.Context, req *ChangePasswordRequest) error {
	data := Data()

	// Fetch current password hash.
	var currentHash string
	err := db.QueryRow(ctx, `
		SELECT password_hash FROM users WHERE id = $1
	`, data.UserID).Scan(&currentHash)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to fetch user"}
	}

	// Verify current password.
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		return &errs.Error{Code: errs.Unauthenticated, Message: "current password is incorrect"}
	}

	// Hash new password.
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to hash password"}
	}

	// Update.
	_, err = db.Exec(ctx, `
		UPDATE users SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`, string(newHash), data.UserID)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to update password"}
	}

	return nil
}
