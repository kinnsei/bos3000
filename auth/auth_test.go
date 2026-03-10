package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"golang.org/x/crypto/bcrypt"
)

func createTestUser(t *testing.T, ctx context.Context, email, password, role string) int64 {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	var id int64
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, email, "testuser_"+role, string(hash), role).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return id
}

func testService() *Service {
	return &Service{jwtSecret: []byte("test-jwt-secret-for-testing-only")}
}

func TestAdminCookieAuth(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	email := fmt.Sprintf("admin-%d@test.com", time.Now().UnixNano())
	createTestUser(t, ctx, email, "adminpass123", "admin")

	body, _ := json.Marshal(AdminLoginRequest{Email: email, Password: "adminpass123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	svc.AdminLogin(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var sessionCookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie should be HttpOnly")
	}
	if !sessionCookie.Secure {
		t.Error("session cookie should be Secure")
	}

	// Verify JWT claims in cookie
	claims, err := svc.parseJWT(sessionCookie.Value)
	if err != nil {
		t.Fatalf("failed to parse JWT from cookie: %v", err)
	}
	if role, _ := claims["role"].(string); role != "admin" {
		t.Errorf("expected role=admin, got %s", role)
	}
	if sub, _ := claims["sub"].(string); sub == "" {
		t.Error("expected sub claim to be set")
	}
}

func TestClientAuth(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	email := fmt.Sprintf("client-%d@test.com", time.Now().UnixNano())
	createTestUser(t, ctx, email, "clientpass123", "client")

	resp, err := svc.ClientLogin(ctx, &ClientLoginRequest{Email: email, Password: "clientpass123"})
	if err != nil {
		t.Fatalf("client login failed: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected token in response")
	}
	if resp.ExpiresAt == "" {
		t.Fatal("expected expires_at in response")
	}

	// Verify JWT claims
	claims, err := svc.parseJWT(resp.Token)
	if err != nil {
		t.Fatalf("failed to parse JWT: %v", err)
	}
	if role, _ := claims["role"].(string); role != "client" {
		t.Errorf("expected role=client, got %s", role)
	}
}

func TestAuthHandlerDispatch(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	// Create admin user and get a JWT
	adminEmail := fmt.Sprintf("admin-handler-%d@test.com", time.Now().UnixNano())
	adminID := createTestUser(t, ctx, adminEmail, "pass123", "admin")
	adminToken, _, err := svc.createJWT(adminID, "admin", "testuser_admin")
	if err != nil {
		t.Fatal(err)
	}

	// Create client user and get a JWT
	clientEmail := fmt.Sprintf("client-handler-%d@test.com", time.Now().UnixNano())
	clientID := createTestUser(t, ctx, clientEmail, "pass123", "client")
	clientToken, _, err := svc.createJWT(clientID, "client", "testuser_client")
	if err != nil {
		t.Fatal(err)
	}

	// Test session cookie dispatch (admin only)
	t.Run("session_cookie", func(t *testing.T) {
		uid, data, err := svc.AuthHandler(ctx, &AuthParams{SessionCookie: adminToken})
		if err != nil {
			t.Fatalf("auth handler failed: %v", err)
		}
		if uid != auth.UID(fmt.Sprintf("%d", adminID)) {
			t.Errorf("expected uid=%d, got %s", adminID, uid)
		}
		if data.Role != "admin" {
			t.Errorf("expected role=admin, got %s", data.Role)
		}
	})

	// Test session cookie rejected for non-admin
	t.Run("session_cookie_client_rejected", func(t *testing.T) {
		_, _, err := svc.AuthHandler(ctx, &AuthParams{SessionCookie: clientToken})
		if err == nil {
			t.Fatal("expected error for client session cookie")
		}
	})

	// Test Bearer token dispatch
	t.Run("bearer_token", func(t *testing.T) {
		uid, data, err := svc.AuthHandler(ctx, &AuthParams{Authorization: "Bearer " + clientToken})
		if err != nil {
			t.Fatalf("auth handler failed: %v", err)
		}
		if uid != auth.UID(fmt.Sprintf("%d", clientID)) {
			t.Errorf("expected uid=%d, got %s", clientID, uid)
		}
		if data.Role != "client" {
			t.Errorf("expected role=client, got %s", data.Role)
		}
	})

	// Test no credentials
	t.Run("no_credentials", func(t *testing.T) {
		_, _, err := svc.AuthHandler(ctx, &AuthParams{})
		if err == nil {
			t.Fatal("expected error for no credentials")
		}
	})
}

func TestInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	email := fmt.Sprintf("invalid-%d@test.com", time.Now().UnixNano())
	createTestUser(t, ctx, email, "correctpass", "client")

	// Wrong password
	_, err := svc.ClientLogin(ctx, &ClientLoginRequest{Email: email, Password: "wrongpass"})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	// Non-existent user
	_, err = svc.ClientLogin(ctx, &ClientLoginRequest{Email: "nonexistent@test.com", Password: "pass"})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}

	// Wrong role (admin trying client login)
	adminEmail := fmt.Sprintf("admin-invalid-%d@test.com", time.Now().UnixNano())
	createTestUser(t, ctx, adminEmail, "adminpass", "admin")
	_, err = svc.ClientLogin(ctx, &ClientLoginRequest{Email: adminEmail, Password: "adminpass"})
	if err == nil {
		t.Fatal("expected error for wrong role")
	}
}
