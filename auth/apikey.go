package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"

	"encore.app/pkg/errcode"
)

// CreateAPIKeyResponse is returned when a new API key is created.
type CreateAPIKeyResponse struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Prefix string `json:"prefix"`
}

//encore:api auth method=POST path=/auth/api-keys
func (s *Service) CreateAPIKey(ctx context.Context) (*CreateAPIKeyResponse, error) {
	data := Data()
	if data == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to generate key"}
	}

	encoded := base64.RawURLEncoding.EncodeToString(raw)
	fullKey := "bos_" + encoded
	prefix := fullKey[:12]

	hash := sha256.Sum256([]byte(fullKey))
	keyHash := fmt.Sprintf("%x", hash)

	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, prefix, key_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`, data.UserID, prefix, keyHash).Scan(&id)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to create API key"}
	}

	return &CreateAPIKeyResponse{
		ID:     id,
		Key:    fullKey,
		Prefix: prefix,
	}, nil
}

// APIKeyInfo is a single API key entry (never includes full key or hash).
type APIKeyInfo struct {
	ID         int64      `json:"id"`
	Prefix     string     `json:"prefix"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// ListAPIKeysResponse wraps the list of API keys.
type ListAPIKeysResponse struct {
	Keys []APIKeyInfo `json:"keys"`
}

//encore:api auth method=GET path=/auth/api-keys
func (s *Service) ListAPIKeys(ctx context.Context) (*ListAPIKeysResponse, error) {
	data := Data()
	if data == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	rows, err := db.Query(ctx, `
		SELECT id, prefix, status, created_at, last_used_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, data.UserID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to list API keys"}
	}
	defer rows.Close()

	var keys []APIKeyInfo
	for rows.Next() {
		var k APIKeyInfo
		if err := rows.Scan(&k.ID, &k.Prefix, &k.Status, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to scan API key"}
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []APIKeyInfo{}
	}

	return &ListAPIKeysResponse{Keys: keys}, nil
}

// APIKeyIDParam is the path parameter for API key operations.
type APIKeyIDParam struct {
	ID int64 `json:"id"`
}

//encore:api auth method=POST path=/auth/api-keys/:id/reset
func (s *Service) ResetAPIKey(ctx context.Context, id int64) (*CreateAPIKeyResponse, error) {
	if err := s.verifyKeyOwnership(ctx, id); err != nil {
		return nil, err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to generate key"}
	}

	encoded := base64.RawURLEncoding.EncodeToString(raw)
	fullKey := "bos_" + encoded
	prefix := fullKey[:12]

	hash := sha256.Sum256([]byte(fullKey))
	keyHash := fmt.Sprintf("%x", hash)

	result, err := db.Exec(ctx, `
		UPDATE api_keys SET key_hash = $1, prefix = $2
		WHERE id = $3
	`, keyHash, prefix, id)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to reset API key"}
	}
	if result.RowsAffected() == 0 {
		return nil, &errs.Error{Code: errs.NotFound, Message: "API key not found"}
	}

	return &CreateAPIKeyResponse{
		ID:     id,
		Key:    fullKey,
		Prefix: prefix,
	}, nil
}

//encore:api auth method=DELETE path=/auth/api-keys/:id
func (s *Service) RevokeAPIKey(ctx context.Context, id int64) error {
	if err := s.verifyKeyOwnership(ctx, id); err != nil {
		return err
	}

	result, err := db.Exec(ctx, `
		UPDATE api_keys SET status = 'revoked'
		WHERE id = $1
	`, id)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to revoke API key"}
	}
	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "API key not found"}
	}
	return nil
}

// IPRequest is the request body for adding/removing IPs.
type IPRequest struct {
	IP string `json:"ip"`
}

// IPListResponse wraps the IP whitelist.
type IPListResponse struct {
	IPs []string `json:"ips"`
}

//encore:api auth method=POST path=/auth/api-keys/:id/ips
func (s *Service) AddIPWhitelist(ctx context.Context, id int64, req *IPRequest) (*IPListResponse, error) {
	if err := s.verifyKeyOwnership(ctx, id); err != nil {
		return nil, err
	}

	if err := validateIPOrCIDR(req.IP); err != nil {
		return nil, err
	}

	var ips []string
	err := db.QueryRow(ctx, `
		UPDATE api_keys SET ip_whitelist = array_append(ip_whitelist, $1)
		WHERE id = $2
		RETURNING ip_whitelist
	`, req.IP, id).Scan(&ips)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to add IP"}
	}

	return &IPListResponse{IPs: ips}, nil
}

//encore:api auth method=DELETE path=/auth/api-keys/:id/ips
func (s *Service) RemoveIPWhitelist(ctx context.Context, id int64, req *IPRequest) (*IPListResponse, error) {
	if err := s.verifyKeyOwnership(ctx, id); err != nil {
		return nil, err
	}

	var ips []string
	err := db.QueryRow(ctx, `
		UPDATE api_keys SET ip_whitelist = array_remove(ip_whitelist, $1)
		WHERE id = $2
		RETURNING ip_whitelist
	`, req.IP, id).Scan(&ips)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to remove IP"}
	}

	return &IPListResponse{IPs: ips}, nil
}

//encore:api auth method=GET path=/auth/api-keys/:id/ips
func (s *Service) ListIPWhitelist(ctx context.Context, id int64) (*IPListResponse, error) {
	if err := s.verifyKeyOwnership(ctx, id); err != nil {
		return nil, err
	}

	var ips []string
	err := db.QueryRow(ctx, `
		SELECT ip_whitelist FROM api_keys WHERE id = $1
	`, id).Scan(&ips)
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "API key not found"}
	}
	if ips == nil {
		ips = []string{}
	}

	return &IPListResponse{IPs: ips}, nil
}

// AdminCreateAPIKeyForUser creates (or regenerates) an API key for a specific user (admin only).
//
//encore:api auth method=POST path=/auth/admin/users/:userId/api-key
func (s *Service) AdminCreateAPIKeyForUser(ctx context.Context, userId int64) (*CreateAPIKeyResponse, error) {
	data := Data()
	if data == nil || data.Role != "admin" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "admin access required"}
	}

	// Revoke existing active keys for this user.
	_, _ = db.Exec(ctx, `UPDATE api_keys SET status = 'revoked' WHERE user_id = $1 AND status = 'active'`, userId)

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to generate key"}
	}

	encoded := base64.RawURLEncoding.EncodeToString(raw)
	fullKey := "bos_" + encoded
	prefix := fullKey[:12]

	hash := sha256.Sum256([]byte(fullKey))
	keyHash := fmt.Sprintf("%x", hash)

	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, prefix, key_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userId, prefix, keyHash).Scan(&id)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to create API key"}
	}

	return &CreateAPIKeyResponse{
		ID:     id,
		Key:    fullKey,
		Prefix: prefix,
	}, nil
}

// verifyKeyOwnership checks that the current user owns the API key, or is admin.
func (s *Service) verifyKeyOwnership(ctx context.Context, keyID int64) error {
	data := Data()
	if data == nil {
		return &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}
	if data.Role == "admin" {
		return nil
	}

	var ownerID int64
	err := db.QueryRow(ctx, `
		SELECT user_id FROM api_keys WHERE id = $1
	`, keyID).Scan(&ownerID)
	if err != nil {
		return &errs.Error{Code: errs.NotFound, Message: "API key not found"}
	}

	uid, _ := auth.UserID()
	if fmt.Sprintf("%d", ownerID) != string(uid) {
		return &errs.Error{Code: errs.PermissionDenied, Message: "not the owner of this API key"}
	}

	return nil
}

func validateIPOrCIDR(input string) error {
	if ip := net.ParseIP(input); ip != nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(input); err == nil {
		return nil
	}
	return errcode.NewError(errs.InvalidArgument, errcode.IPNotWhitelisted, "invalid IP address or CIDR")
}
