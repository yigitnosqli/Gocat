package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ibrahmsql/gocat/internal/logger"
)

// WebSocketToHTTPConverter converts WebSocket messages to HTTP requests
type WebSocketToHTTPConverter struct {
	listenAddr string
	targetURL  string
	server     *http.Server
	upgrader   websocket.Upgrader
	client     *http.Client
	mu         sync.RWMutex
	active     int
}

// MessageEnvelope wraps WebSocket messages with metadata
type MessageEnvelope struct {
	ID        string            `json:"id,omitempty"`
	Method    string            `json:"method"`            // HTTP method (default: POST)
	Path      string            `json:"path,omitempty"`    // Optional path to append to target URL
	Headers   map[string]string `json:"headers,omitempty"` // Additional HTTP headers
	Body      interface{}       `json:"body"`              // Message payload
	Timestamp time.Time         `json:"timestamp,omitempty"`
}

// ResponseEnvelope wraps HTTP responses
type ResponseEnvelope struct {
	ID         string            `json:"id,omitempty"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// NewWebSocketToHTTPConverter creates a new WebSocket to HTTP converter
func NewWebSocketToHTTPConverter(listenAddr, targetURL string) (*WebSocketToHTTPConverter, error) {
	if listenAddr == "" {
		return nil, fmt.Errorf("listen address cannot be empty")
	}
	if targetURL == "" {
		return nil, fmt.Errorf("target URL cannot be empty")
	}

	return &WebSocketToHTTPConverter{
		listenAddr: listenAddr,
		targetURL:  targetURL,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins (can be restricted in production)
			},
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

// Start starts the WebSocket server
func (c *WebSocketToHTTPConverter) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", c.handleWebSocket)

	c.server = &http.Server{
		Addr:         c.listenAddr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("WebSocket->HTTP converter ready to accept connections")
	return c.server.ListenAndServe()
}

// Shutdown gracefully shuts down the converter
func (c *WebSocketToHTTPConverter) Shutdown() error {
	if c.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.server.Shutdown(ctx)
}

// handleWebSocket handles incoming WebSocket connections
func (c *WebSocketToHTTPConverter) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	c.mu.Lock()
	c.active++
	connID := c.active
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.active--
		c.mu.Unlock()
	}()

	logger.Info("WebSocket connection #%d established from %s", connID, r.RemoteAddr)

	// Handle messages from this connection
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error: %v", err)
			}
			break
		}

		logger.Debug("Received WebSocket message (type: %d, size: %d bytes)", messageType, len(message))

		// Convert and forward to HTTP
		response := c.convertAndForward(message, messageType)

		// Send response back through WebSocket
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, responseData); err != nil {
			logger.Error("Failed to send response: %v", err)
			break
		}
	}

	logger.Info("WebSocket connection #%d closed", connID)
}

// convertAndForward converts a WebSocket message to HTTP request and forwards it
func (c *WebSocketToHTTPConverter) convertAndForward(message []byte, _ int) *ResponseEnvelope {
	response := &ResponseEnvelope{
		Timestamp: time.Now(),
	}

	// Try to parse as JSON envelope
	var envelope MessageEnvelope
	if err := json.Unmarshal(message, &envelope); err != nil {
		// Not a JSON envelope, treat as raw data
		envelope = MessageEnvelope{
			Method: "POST",
			Body:   string(message),
		}
	}

	// Set defaults
	if envelope.Method == "" {
		envelope.Method = "POST"
	}
	if envelope.Timestamp.IsZero() {
		envelope.Timestamp = time.Now()
	}

	response.ID = envelope.ID

	// Build target URL
	targetURL := c.targetURL
	if envelope.Path != "" {
		targetURL = targetURL + envelope.Path
	}

	// Prepare request body
	var bodyReader io.Reader
	switch v := envelope.Body.(type) {
	case string:
		bodyReader = bytes.NewBufferString(v)
	case []byte:
		bodyReader = bytes.NewBuffer(v)
	default:
		// Marshal to JSON
		bodyData, err := json.Marshal(envelope.Body)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to marshal body: %v", err)
			response.StatusCode = 400
			return response
		}
		bodyReader = bytes.NewBuffer(bodyData)
	}

	// Create HTTP request
	req, err := http.NewRequest(envelope.Method, targetURL, bodyReader)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to create request: %v", err)
		response.StatusCode = 500
		return response
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoCat-WebSocket-Converter/1.0")
	req.Header.Set("X-Forwarded-Proto", "websocket")
	req.Header.Set("X-Original-Timestamp", envelope.Timestamp.Format(time.RFC3339))

	// Add custom headers from envelope
	for key, value := range envelope.Headers {
		req.Header.Set(key, value)
	}

	// Send HTTP request
	logger.Debug("Forwarding to HTTP: %s %s", envelope.Method, targetURL)
	httpResp, err := c.client.Do(req)
	if err != nil {
		response.Error = fmt.Sprintf("HTTP request failed: %v", err)
		response.StatusCode = 503
		return response
	}
	defer httpResp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to read response: %v", err)
		response.StatusCode = 500
		return response
	}

	// Build response
	response.StatusCode = httpResp.StatusCode
	response.Body = string(bodyBytes)
	response.Headers = make(map[string]string)

	for key, values := range httpResp.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	logger.Debug("HTTP response received: status=%d, body_size=%d", httpResp.StatusCode, len(bodyBytes))

	return response
}

// Stats returns converter statistics
func (c *WebSocketToHTTPConverter) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"listen_addr":        c.listenAddr,
		"target_url":         c.targetURL,
		"active_connections": c.active,
	}
}
