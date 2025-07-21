package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

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
	broadcast   chan WebSocketMessage
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	ctx         context.Context
	cancel      context.CancelFunc
	config      *WebSocketConfig
}

// WebSocketMessage represents a message with its type
type WebSocketMessage struct {
	Data []byte
	Type int // websocket.TextMessage or websocket.BinaryMessage
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan WebSocketMessage
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
		// CheckOrigin is critical for WebSocket security in production environments.
		// This function validates the Origin header to prevent Cross-Site WebSocket Hijacking (CSWSH) attacks.
		//
		// SECURITY WARNING: The current implementation rejects all origins by default.
		// For production use, you MUST configure this based on your deployment:
		//
		// 1. For same-origin only: Check if r.Header.Get("Origin") matches your domain
		// 2. For specific domains: Maintain a whitelist of allowed origins
		// 3. For development: You may temporarily allow localhost/127.0.0.1
		//
		// Example implementations:
		// - Same origin: return r.Header.Get("Origin") == "https://yourdomain.com"
		// - Whitelist: allowedOrigins := []string{"https://app.com", "https://admin.app.com"}
		//             origin := r.Header.Get("Origin")
		//             for _, allowed := range allowedOrigins { if origin == allowed { return true } }
		//             return false
		// - Development: return strings.Contains(r.Header.Get("Origin"), "localhost") ||
		//                      strings.Contains(r.Header.Get("Origin"), "127.0.0.1")
		CheckOrigin: func(r *http.Request) bool {
			// SECURE DEFAULT: Reject all origins
			// TODO: Configure this function based on your deployment requirements
			return false
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
		metrics:    metrics.GetGlobalMetrics(),
		broadcast:  make(chan WebSocketMessage),
		register:   make(chan *WebSocketConnection),
		unregister: make(chan *WebSocketConnection),
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
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
		Send:     make(chan WebSocketMessage, 256),
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
			// Collect clients to delete while holding read lock
			var clientsToDelete []string
			s.mu.RLock()
			for _, client := range s.connections {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					clientsToDelete = append(clientsToDelete, client.ID)
				}
			}
			s.mu.RUnlock()

			// Delete clients under write lock to prevent race conditions
			if len(clientsToDelete) > 0 {
				s.mu.Lock()
				for _, clientID := range clientsToDelete {
					delete(s.connections, clientID)
				}
				s.mu.Unlock()
			}
		}
	}
}

// isTextMessage determines if a message contains valid UTF-8 text
func isTextMessage(data []byte) bool {
	// Simple heuristic: check if the data is valid UTF-8
	// and doesn't contain null bytes (common in binary data)
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return utf8.Valid(data)
}

// Broadcast sends a message to all connected clients
func (s *WebSocketServer) Broadcast(message []byte) {
	// Determine message type
	msgType := websocket.TextMessage
	if !isTextMessage(message) {
		msgType = websocket.BinaryMessage
	}

	websocketMsg := WebSocketMessage{Data: message, Type: msgType}
	select {
	case s.broadcast <- websocketMsg:
	default:
		s.logger.Warn("Broadcast channel is full, message dropped")
	}
}

// BroadcastWithType sends a message with specified type to all connected clients
func (s *WebSocketServer) BroadcastWithType(message []byte, messageType int) {
	websocketMsg := WebSocketMessage{Data: message, Type: messageType}
	select {
	case s.broadcast <- websocketMsg:
	default:
		s.logger.Warn("Broadcast channel is full, message dropped")
	}
}

// SendToClient sends a message to a specific client
func (s *WebSocketServer) SendToClient(clientID string, message []byte) error {
	// Determine message type
	msgType := websocket.TextMessage
	if !isTextMessage(message) {
		msgType = websocket.BinaryMessage
	}

	return s.SendToClientWithType(clientID, message, msgType)
}

// SendToClientWithType sends a message with specified type to a specific client
func (s *WebSocketServer) SendToClientWithType(clientID string, message []byte, messageType int) error {
	s.mu.RLock()
	client, exists := s.connections[clientID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	websocketMsg := WebSocketMessage{Data: message, Type: messageType}
	select {
	case client.Send <- websocketMsg:
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

	c.Conn.SetReadLimit(c.Server.config.MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(c.Server.config.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Server.config.PongWait))
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
		// Determine message type based on content (simple heuristic)
		msgType := websocket.TextMessage
		if !isTextMessage(message) {
			msgType = websocket.BinaryMessage
		}
		c.Send <- WebSocketMessage{Data: message, Type: msgType}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *WebSocketConnection) writePump() {
	ticker := time.NewTicker(c.Server.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Server.config.WriteWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// For binary messages, send each message separately to preserve boundaries
			if message.Type == websocket.BinaryMessage {
				if err := c.Conn.WriteMessage(message.Type, message.Data); err != nil {
					return
				}
				c.Server.metrics.AddBytesSent(int64(len(message.Data)))

				// Send any additional binary messages separately
				n := len(c.Send)
				for i := 0; i < n; i++ {
					queuedMsg := <-c.Send
					if err := c.Conn.WriteMessage(queuedMsg.Type, queuedMsg.Data); err != nil {
						return
					}
					c.Server.metrics.AddBytesSent(int64(len(queuedMsg.Data)))
				}
			} else {
				// For text messages, batch them with newline separators
				w, err := c.Conn.NextWriter(message.Type)
				if err != nil {
					return
				}
				w.Write(message.Data)

				// Add queued text messages to the current websocket message
				n := len(c.Send)
				for i := 0; i < n; i++ {
					queuedMsg := <-c.Send
					if queuedMsg.Type == websocket.TextMessage {
						w.Write([]byte{'\n'})
						w.Write(queuedMsg.Data)
					} else {
						// If we encounter a binary message while batching text, close current writer
						// and send the binary message separately
						if err := w.Close(); err != nil {
							return
						}
						if err := c.Conn.WriteMessage(queuedMsg.Type, queuedMsg.Data); err != nil {
							return
						}
						c.Server.metrics.AddBytesSent(int64(len(queuedMsg.Data)))
						// Continue with remaining messages
						for j := i + 1; j < n; j++ {
							remaining := <-c.Send
							if err := c.Conn.WriteMessage(remaining.Type, remaining.Data); err != nil {
								return
							}
							c.Server.metrics.AddBytesSent(int64(len(remaining.Data)))
						}
						c.Server.metrics.AddBytesSent(int64(len(message.Data)))
						return
					}
				}

				if err := w.Close(); err != nil {
					return
				}
				c.Server.metrics.AddBytesSent(int64(len(message.Data)))
			}

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
