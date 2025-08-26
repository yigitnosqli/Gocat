package network

import (
	"net"
	"testing"
	"time"
)

func TestDefaultConnectionOptions(t *testing.T) {
	opts := DefaultConnectionOptions()

	if opts == nil {
		t.Fatal("DefaultConnectionOptions should not return nil")
	}

	if opts.Timeout <= 0 {
		t.Error("Default timeout should be positive")
	}

	if opts.BufferSize <= 0 {
		t.Error("Default buffer size should be positive")
	}
}

func TestConnectionTypeValidation(t *testing.T) {
	validTypes := []ConnectionType{
		ConnectionTypeTCP,
		ConnectionTypeUDP,
		ConnectionTypeTLS,
		ConnectionTypeUnix,
	}

	for _, connType := range validTypes {
		if string(connType) == "" {
			t.Errorf("Connection type %v should not be empty", connType)
		}
	}
}

func TestNewDialer(t *testing.T) {
	opts := &ConnectionOptions{
		Host:     "localhost",
		Port:     8080,
		Protocol: ConnectionTypeTCP,
		Timeout:  5 * time.Second,
	}

	dialer := NewDialer(opts)
	if dialer == nil {
		t.Fatal("NewDialer should not return nil")
	}

	if dialer.options != opts {
		t.Error("Dialer should store the provided options")
	}
}

func TestConnectionStatsInitialization(t *testing.T) {
	// Create a test connection for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(client, DefaultConnectionOptions())
	if conn == nil {
		t.Fatal("NewConnection should not return nil")
	}

	stats := conn.Stats()
	if stats.ConnectedAt.IsZero() {
		t.Error("ConnectedAt should be set")
	}

	if stats.BytesRead != 0 {
		t.Error("Initial BytesRead should be 0")
	}

	if stats.BytesWritten != 0 {
		t.Error("Initial BytesWritten should be 0")
	}
}

func TestConnectionContextCancellation(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(client, DefaultConnectionOptions())
	ctx := conn.Context()

	if ctx == nil {
		t.Fatal("Connection context should not be nil")
	}

	// Close connection and verify context is cancelled
	conn.Close()

	select {
	case <-ctx.Done():
		// Context should be cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled when connection is closed")
	}
}
