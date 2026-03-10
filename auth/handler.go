package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"strings"

	"encore.dev"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/golang-jwt/jwt/v5"

	"encore.app/pkg/errcode"
)

//encore:authhandler
func (s *Service) AuthHandler(ctx context.Context, p *AuthParams) (auth.UID, *AuthData, error) {
	// Dispatch order: session cookie -> Bearer token -> API key
	if p.SessionCookie != "" {
		return s.validateSessionCookie(ctx, p.SessionCookie)
	}
	if p.Authorization != "" && strings.HasPrefix(p.Authorization, "Bearer ") {
		token := strings.TrimPrefix(p.Authorization, "Bearer ")
		return s.validateBearerToken(ctx, token)
	}
	if p.APIKey != "" {
		return s.validateAPIKey(ctx, p.APIKey)
	}
	return "", nil, &errs.Error{
		Code:    errs.Unauthenticated,
		Message: "no authentication credentials provided",
	}
}

func (s *Service) validateSessionCookie(ctx context.Context, cookieValue string) (auth.UID, *AuthData, error) {
	claims, err := s.parseJWT(cookieValue)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "invalid session cookie",
		}
	}
	role, _ := claims["role"].(string)
	if role != "admin" {
		return "", nil, &errs.Error{
			Code:    errs.PermissionDenied,
			Message: "session cookies are only valid for admin users",
		}
	}
	sub, _ := claims["sub"].(string)
	username, _ := claims["username"].(string)
	return auth.UID(sub), &AuthData{
		UserID:   parseUserID(sub),
		Role:     role,
		Username: username,
	}, nil
}

func (s *Service) validateBearerToken(ctx context.Context, tokenStr string) (auth.UID, *AuthData, error) {
	claims, err := s.parseJWT(tokenStr)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "invalid bearer token",
		}
	}
	sub, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	username, _ := claims["username"].(string)
	return auth.UID(sub), &AuthData{
		UserID:   parseUserID(sub),
		Role:     role,
		Username: username,
	}, nil
}

func (s *Service) validateAPIKey(ctx context.Context, apiKey string) (auth.UID, *AuthData, error) {
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := fmt.Sprintf("%x", hash)

	var userID int64
	var role, username, status string
	var ipWhitelist []string
	err := db.QueryRow(ctx, `
		SELECT ak.status, ak.ip_whitelist, u.id, u.role, u.username
		FROM api_keys ak
		JOIN users u ON u.id = ak.user_id
		WHERE ak.key_hash = $1
	`, keyHash).Scan(&status, &ipWhitelist, &userID, &role, &username)
	if err != nil {
		return "", nil, errcode.NewError(errs.Unauthenticated, errcode.InvalidCredentials, "invalid API key")
	}
	if status != "active" {
		return "", nil, errcode.NewError(errs.Unauthenticated, errcode.APIKeyRevoked, "API key has been revoked")
	}

	if len(ipWhitelist) > 0 {
		if err := checkIPWhitelist(ipWhitelist); err != nil {
			return "", nil, err
		}
	}

	// Update last_used_at
	_, _ = db.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE key_hash = $1`, keyHash)

	uid := fmt.Sprintf("%d", userID)
	return auth.UID(uid), &AuthData{
		UserID:   userID,
		Role:     role,
		Username: username,
	}, nil
}

func checkIPWhitelist(whitelist []string) error {
	req := encore.CurrentRequest()
	var remoteIP string
	if xff := req.Headers.Get("X-Forwarded-For"); xff != "" {
		remoteIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	} else if xri := req.Headers.Get("X-Real-Ip"); xri != "" {
		remoteIP = strings.TrimSpace(xri)
	}

	if remoteIP == "" {
		return errcode.NewError(errs.PermissionDenied, errcode.IPNotWhitelisted, "could not determine client IP")
	}

	clientIP := net.ParseIP(remoteIP)
	if clientIP == nil {
		return errcode.NewError(errs.PermissionDenied, errcode.IPNotWhitelisted, "invalid client IP")
	}

	for _, entry := range whitelist {
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				continue
			}
			if cidr.Contains(clientIP) {
				return nil
			}
		} else {
			if net.ParseIP(entry) != nil && entry == clientIP.String() {
				return nil
			}
		}
	}
	return errcode.NewError(errs.PermissionDenied, errcode.IPNotWhitelisted, "IP address not in whitelist")
}

func (s *Service) parseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

func parseUserID(sub string) int64 {
	var id int64
	fmt.Sscanf(sub, "%d", &id)
	return id
}
