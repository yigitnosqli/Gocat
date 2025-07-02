package cmd

import (
	"net"
	"testing"
	"time"
)

func TestConnect(t *testing.T) {
	// Test invalid port
	err := connect("127.0.0.1", "99999", "/bin/sh")
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

func TestDialWithOptions(t *testing.T) {
	// Test with invalid address
	_, err := dialWithOptions("tcp", "invalid:99999")
	if err == nil {
		t.Error("Expected error for invalid address")
	}
}

func TestHostPortParsing(t *testing.T) {
	tests := []struct {
		args    []string
		expHost string
		expPort string
	}{
		{[]string{"8080"}, "127.0.0.1", "8080"},
		{[]string{"192.168.1.1", "9090"}, "192.168.1.1", "9090"},
	}

	for _, test := range tests {
		var host, port string
		if len(test.args) == 1 {
			host = "127.0.0.1"
			port = test.args[0]
		} else {
			host = test.args[0]
			port = test.args[1]
		}

		if host != test.expHost || port != test.expPort {
			t.Errorf("Expected %s:%s, got %s:%s", test.expHost, test.expPort, host, port)
		}
	}
}

func TestRetryLogic(t *testing.T) {
	// Save original values
	origRetryCount := retryCount
	origTimeout := timeout

	// Set test values
	retryCount = 2
	timeout = 100 * time.Millisecond

	// Restore original values
	defer func() {
		retryCount = origRetryCount
		timeout = origTimeout
	}()

	// Test connection to non-existent service
	start := time.Now()
	err := connect("127.0.0.1", "12345", "/bin/sh")
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected connection to fail")
	}

	// Should have taken at least the timeout duration * retry attempts
	expectedMinDuration := timeout * time.Duration(retryCount+1)
	if duration < expectedMinDuration {
		t.Errorf("Expected at least %v, got %v", expectedMinDuration, duration)
	}
}

func BenchmarkConnect(b *testing.B) {
	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			b.Logf("Error closing listener: %v", err)
		}
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			if err := conn.Close(); err != nil {
				b.Logf("Error closing connection: %v", err)
			}
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just test the dial part, not the shell execution
		conn, err := dialWithOptions("tcp", addr.String())
		if err != nil {
			b.Error(err)
			continue
		}
		if err := conn.Close(); err != nil {
			b.Logf("Error closing connection: %v", err)
		}
	}
}
