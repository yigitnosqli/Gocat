package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/metrics"
)

// WebSocketServer represents a WebSocket server
type WebSocketServer struct {
	mu          sync.RWMutex
	connections map[string]*WebSocketConnection
	upgrader    websocket.Upgrader
	logger      *logger.Logger
	metrics     *metrics.Metrics
	broadcast   chan []byte
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	ctx         context.Context
	cancel      context.CancelFunc
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Server   *WebSocketServer
	LastPing time.Time
	UserData map[string]interface{}
	mu       sync.RWMutex
}

// WebSocketConfig holds WebSocket server configuration
type WebSocketConfig struct {
	ReadBufferSize    int
	WriteBufferSize   int
	HandshakeTimeout  time.Duration
	PingPeriod        time.Duration
	PongWait          time.Duration
	WriteWait         time.Duration
	MaxMessageSize    int64
	CheckOrigin       func(r *http.Request) bool
	EnableCompression bool
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  10 * time.Second,
		PingPeriod:        54 * time.Second,
		PongWait:          60 * time.Second,
		WriteWait:         10 * time.Second,
		MaxMessageSize:    512,
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(config *WebSocketConfig) *WebSocketServer {
	if config == nil {
		config = DefaultWebSocketConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &WebSocketServer{
		connections: make(map[string]*WebSocketConnection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:    config.ReadBufferSize,
			WriteBufferSize:   config.WriteBufferSize,
			HandshakeTimeout:  config.HandshakeTimeout,
			CheckOrigin:       config.CheckOrigin,
			EnableCompression: config.EnableCompression,
		},
		logger:     logger.GetDefaultLogger(),
		metrics:    metrics.NewMetrics(),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketConnection),
		unregister: make(chan *WebSocketConnection),
		ctx:        ctx,
		cancel:     cancel,
	}

	go server.run()
	return server
}

// HandleWebSocket handles WebSocket upgrade requests
func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.ErrorWithFields("WebSocket upgrade failed", map[string]interface{}{
			"error":       err.Error(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		})
		return
	}

	// Generate unique connection ID
	connID := fmt.Sprintf("%s_%d", r.RemoteAddr, time.Now().UnixNano())

	client := &WebSocketConnection{
		ID:       connID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Server:   s,
		LastPing: time.Now(),
		UserData: make(map[string]interface{}),
	}

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	s.logger.InfoWithFields("WebSocket connection established", map[string]interface{}{
		"connection_id": connID,
		"remote_addr":   r.RemoteAddr,
	})
}

// run handles the main server loop
func (s *WebSocketServer) run() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case client := <-s.register:
			s.mu.Lock()
			s.connections[client.ID] = client
			s.mu.Unlock()
			s.metrics.IncrementConnectionsActive()

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.connections[client.ID]; ok {
				delete(s.connections, client.ID)
				close(client.Send)
			}
			s.mu.Unlock()
			s.metrics.DecrementConnectionsActive()

		case message := <-s.broadcast:
			s.mu.RLock()
			for _, client := range s.connections {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(s.connections, client.ID)
				}
			}
			s.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (s *WebSocketServer) Broadcast(message []byte) {
	select {
	case s.broadcast <- message:
	default:
		s.logger.Warn("Broadcast channel is full, message dropped")
	}
}

// SendToClient sends a message to a specific client
func (s *WebSocketServer) SendToClient(clientID string, message []byte) error {
	s.mu.RLock()
	client, exists := s.connections[clientID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	select {
	case client.Send <- message:
		return nil
	default:
		return fmt.Errorf("client %s send channel is full", clientID)
	}
}

// GetConnections returns all active connections
func (s *WebSocketServer) GetConnections() map[string]*WebSocketConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*WebSocketConnection)
	for id, conn := range s.connections {
		result[id] = conn
	}
	return result
}

// GetConnectionCount returns the number of active connections
func (s *WebSocketServer) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// Shutdown gracefully shuts down the WebSocket server
func (s *WebSocketServer) Shutdown() error {
	s.cancel()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all connections
	for _, client := range s.connections {
		client.Close()
	}

	s.logger.Info("WebSocket server shutdown complete")
	return nil
}

// readPump pumps messages from the websocket connection to the hub
func (c *WebSocketConnection) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.LastPing = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Server.logger.ErrorWithFields("WebSocket read error", map[string]interface{}{
					"connection_id": c.ID,
					"error":         err.Error(),
				})
			}
			break
		}

		c.Server.metrics.AddBytesReceived(int64(len(message)))

		// Echo the message back (can be customized)
		c.Send <- message
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *WebSocketConnection) writePump() {
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

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

			c.Server.metrics.AddBytesSent(int64(len(message)))

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close closes the WebSocket connection
func (c *WebSocketConnection) Close() error {
	return c.Conn.Close()
}

// SetUserData sets user-defined data for the connection
func (c *WebSocketConnection) SetUserData(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserData[key] = value
}

// GetUserData gets user-defined data from the connection
func (c *WebSocketConnection) GetUserData(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.UserData[key]
	return value, exists
}
