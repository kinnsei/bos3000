package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"encore.dev/beta/errs"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"encore.app/pkg/errcode"
)

// AdminLoginRequest is the request body for admin login.
type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

//encore:api public raw method=POST path=/auth/admin/login
func (s *Service) AdminLogin(w http.ResponseWriter, req *http.Request) {
	var body AdminLoginRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, `{"code":"invalid_argument","message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	user, err := s.authenticateUser(req.Context(), body.Email, body.Password, "admin")
	if err != nil {
		http.Error(w, `{"code":"unauthenticated","message":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	tokenStr, expiresAt, err := s.createJWT(user.id, user.role, user.username)
	if err != nil {
		http.Error(w, `{"code":"internal","message":"failed to create token"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":    "login successful",
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// ClientLoginRequest is the request body for client login.
type ClientLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ClientLoginResponse is the response body for client login.
type ClientLoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

//encore:api public method=POST path=/auth/client/login
func (s *Service) ClientLogin(ctx context.Context, req *ClientLoginRequest) (*ClientLoginResponse, error) {
	user, err := s.authenticateUser(ctx, req.Email, req.Password, "client")
	if err != nil {
		return nil, err
	}

	tokenStr, expiresAt, err := s.createJWT(user.id, user.role, user.username)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to create token",
		}
	}

	return &ClientLoginResponse{
		Token:     tokenStr,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

type userRecord struct {
	id           int64
	email        string
	username     string
	role         string
	passwordHash string
}

func (s *Service) authenticateUser(ctx context.Context, email, password, expectedRole string) (*userRecord, error) {
	var user userRecord
	err := db.QueryRow(ctx, `
		SELECT id, email, username, role, password_hash
		FROM users
		WHERE email = $1
	`, email).Scan(&user.id, &user.email, &user.username, &user.role, &user.passwordHash)
	if err != nil {
		return nil, errcode.NewError(errs.Unauthenticated, errcode.InvalidCredentials, "invalid credentials")
	}

	if user.role != expectedRole {
		return nil, errcode.NewError(errs.Unauthenticated, errcode.InvalidCredentials, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.passwordHash), []byte(password)); err != nil {
		return nil, errcode.NewError(errs.Unauthenticated, errcode.InvalidCredentials, "invalid credentials")
	}

	return &user, nil
}

func (s *Service) createJWT(userID int64, role, username string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":      fmt.Sprintf("%d", userID),
		"role":     role,
		"username": username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenStr, expiresAt, nil
}
