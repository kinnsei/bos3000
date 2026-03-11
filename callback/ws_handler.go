package callback

import (
	"fmt"
	"net/http"

	"encore.dev/rlog"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var secrets struct {
	JWTSecret string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler upgrades HTTP connections to WebSocket for real-time call
// status push. Clients authenticate via a JWT token in the query string.
//
//encore:api public raw path=/ws/calls
func (s *Service) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, isAdmin, err := validateWSToken(token)
	if err != nil {
		rlog.Warn("ws auth failed", "err", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		rlog.Error("ws upgrade failed", "err", err)
		return
	}

	client := &Client{
		hub:     s.hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		userID:  userID,
		isAdmin: isAdmin,
	}

	s.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// validateWSToken parses a JWT and returns the user ID and admin flag.
func validateWSToken(tokenStr string) (userID string, isAdmin bool, err error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secrets.JWTSecret), nil
	})
	if err != nil {
		return "", false, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", false, fmt.Errorf("invalid claims")
	}

	uid, ok := claims["user_id"].(string)
	if !ok || uid == "" {
		return "", false, fmt.Errorf("missing user_id claim")
	}

	if admin, ok := claims["is_admin"].(bool); ok {
		isAdmin = admin
	}

	return uid, isAdmin, nil
}
