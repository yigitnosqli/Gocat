package websocket

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultWebSocketConfig(t *testing.T) {
	config := DefaultWebSocketConfig()

	if config == nil {
		t.Fatal("DefaultWebSocketConfig should not return nil")
	}

	if config.ReadBufferSize <= 0 {
		t.Error("ReadBufferSize should be positive")
	}

	if config.WriteBufferSize <= 0 {
		t.Error("WriteBufferSize should be positive")
	}

	if config.CheckOrigin == nil {
		t.Error("CheckOrigin should not be nil")
	}
}

func TestSecureOriginChecker(t *testing.T) {
	checker := createSecureOriginChecker()

	// Test same-origin request
	req := httptest.NewRequest("GET", "http://localhost:8080/ws", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:8080")

	if !checker(req) {
		t.Error("Same-origin request should be allowed")
	}

	// Test localhost request
	req.Header.Set("Origin", "http://localhost:3000")
	if !checker(req) {
		t.Error("Localhost request should be allowed")
	}

	// Test malicious origin
	req.Header.Set("Origin", "http://evil.com")
	if checker(req) {
		t.Error("Malicious origin should be rejected")
	}
}

func TestNewWebSocketServer(t *testing.T) {
	config := DefaultWebSocketConfig()
	if config == nil {
		t.Fatal("DefaultWebSocketConfig should not return nil")
	}

	server := NewWebSocketServer(config)
	if server == nil {
		t.Fatal("NewWebSocketServer should not return nil")
	}

	if server.connections == nil {
		t.Error("Server connections map should be initialized")
	}

	// Test server shutdown
	server.Shutdown()
	time.Sleep(10 * time.Millisecond) // Give time for cleanup
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"localhost:8080", true},
		{"127.0.0.1", true},
		{"127.0.0.1:3000", true},
		{"::1", true},
		{"[::1]:8080", true},
		{"example.com", false},
		{"192.168.1.1", false},
	}

	for _, test := range tests {
		result := isLocalhost(test.host)
		if result != test.expected {
			t.Errorf("isLocalhost(%q) = %v, expected %v", test.host, result, test.expected)
		}
	}
}
