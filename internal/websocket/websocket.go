package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zakirkun/isekai/pkg/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a WebSocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan Message
	Hub  *Hub
	mu   sync.Mutex
}

// Hub maintains active WebSocket connections
type Hub struct {
	clients    map[string]*Client
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	log        *logger.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(log *logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		log:        log,
	}
}

// Run starts the hub
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.log.Infof("WebSocket client registered: %s", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			h.log.Infof("WebSocket client unregistered: %s", client.ID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client.ID)
					close(client.Send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()

		case <-ctx.Done():
			h.log.Info("WebSocket hub shutting down")
			return
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message Message) {
	h.broadcast <- message
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, message Message) bool {
	h.mu.RLock()
	client, exists := h.clients[clientID]
	h.mu.RUnlock()

	if !exists {
		return false
	}

	select {
	case client.Send <- message:
		return true
	default:
		return false
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.log.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Handle received message (echo for now)
		c.Hub.Broadcast(msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.Conn.WriteJSON(message)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWS handles WebSocket requests
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, clientID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.log.Errorf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan Message, 256),
		Hub:  hub,
	}

	hub.register <- client

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}
