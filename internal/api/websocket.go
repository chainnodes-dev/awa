package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/events"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Hub manages all active WebSocket connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
	bus     events.Bus
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
}

func NewHub(bus events.Bus) *Hub {
	return &Hub{
		clients: make(map[*wsClient]bool),
		bus:     bus,
	}
}

// Start subscribes to the event bus and fans out to all connected clients.
func (h *Hub) Start(ctx context.Context) {
	ch, cancel := h.bus.Subscribe(ctx)
	_ = cancel // runs until process exits

	go func() {
		for event := range ch {
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			h.broadcast(data)
		}
	}()
}

func (h *Hub) broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// drop slow clients
		}
	}
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
// The caller must supply a valid JWT in the "token" query parameter:
//
//	ws://host/ws?token=<access_token>
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, jwtSvc *auth.JWTService) {
	token := r.URL.Query().Get("token")
	if _, err := jwtSvc.ValidateToken(token); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("WebSocket upgrade error", "error", err)
		return
	}

	client := &wsClient{
		conn: conn,
		send: make(chan []byte, 256),
		done: make(chan struct{}),
	}

	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()

	slog.Debug("WebSocket client connected", "total", len(h.clients))

	go client.writePump()
	go func() {
		client.readPump()
		h.mu.Lock()
		delete(h.clients, client)
		h.mu.Unlock()
		close(client.done)
		slog.Debug("WebSocket client disconnected", "remaining", len(h.clients))
	}()
}

// writePump sends queued messages to the WebSocket connection.
func (c *wsClient) writePump() {
	defer c.conn.Close()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

// readPump discards incoming messages (clients are read-only in this version).
func (c *wsClient) readPump() {
	defer c.conn.Close()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
