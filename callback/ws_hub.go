package callback

import (
	"encoding/json"
	"sync"
	"time"

	"encore.dev/rlog"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 54 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client represents a single WebSocket connection.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	userID  string
	isAdmin bool
}

// Hub manages all WebSocket clients and broadcasts events.
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*Client]bool // user_id -> set of clients
	adminConns map[*Client]bool
	broadcast  chan *CallStatusEvent
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		adminConns: make(map[*Client]bool),
		broadcast:  make(chan *CallStatusEvent, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub event loop. Should be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if client.isAdmin {
				h.adminConns[client] = true
			} else {
				if h.clients[client.userID] == nil {
					h.clients[client.userID] = make(map[*Client]bool)
				}
				h.clients[client.userID][client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if client.isAdmin {
				if _, ok := h.adminConns[client]; ok {
					delete(h.adminConns, client)
					close(client.send)
				}
			} else {
				if conns, ok := h.clients[client.userID]; ok {
					if _, exists := conns[client]; exists {
						delete(conns, client)
						close(client.send)
						if len(conns) == 0 {
							delete(h.clients, client.userID)
						}
					}
				}
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				rlog.Error("failed to marshal call status event", "err", err)
				continue
			}

			h.mu.RLock()
			// Send to all admin connections.
			for client := range h.adminConns {
				select {
				case client.send <- data:
				default:
					// Buffer full; will be cleaned up on write error.
				}
			}
			// Send to the specific user's connections.
			if conns, ok := h.clients[event.UserID]; ok {
				for client := range conns {
					select {
					case client.send <- data:
					default:
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a CallStatusEvent to all relevant WebSocket clients.
func (h *Hub) Broadcast(event *CallStatusEvent) {
	h.broadcast <- event
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
// It reads and discards all incoming messages; its main purpose is to
// detect disconnection and handle pong messages.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}
