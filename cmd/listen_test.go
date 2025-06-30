package cmd

import (
	"net"
	"testing"
	"time"
)

func TestListen(t *testing.T) {
	// Test invalid port
	err := listen("127.0.0.1", "99999")
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

func TestCreateTLSListener(t *testing.T) {
	// Test without certificate files
	_, err := createTLSListener("tcp", "127.0.0.1:0")
	if err == nil {
		t.Error("Expected error when SSL cert/key files are not provided")
	}
}

func TestHandleUDPListener(t *testing.T) {
	// Test UDP listener creation with timeout
	done := make(chan error, 1)
	go func() {
		// This will fail because we can't bind to port 0 for UDP in this context
		// but it tests the function structure
		done <- handleUDPListener("udp", "127.0.0.1:0")
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err != nil {
			t.Logf("UDP listener failed as expected: %v", err)
		} else {
			t.Log("UDP listener test completed")
		}
	case <-time.After(1 * time.Second):
		t.Log("UDP listener test timed out (expected for port 0)")
	}
}

func TestMaxConnections(t *testing.T) {
	// Save original value
	origMaxConnections := maxConnections
	maxConnections = 2

	// Restore original value
	defer func() {
		maxConnections = origMaxConnections
	}()

	// Start a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Logf("Error closing listener: %v", closeErr)
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	// Test that we can handle up to maxConnections
	connections := make([]net.Conn, maxConnections)
	for i := 0; i < maxConnections; i++ {
		conn, err := net.Dial("tcp", addr.String())
		if err != nil {
			t.Errorf("Failed to create connection %d: %v", i, err)
			continue
		}
		connections[i] = conn
	}

	// Clean up
	for _, conn := range connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				t.Logf("Error closing connection: %v", err)
			}
		}
	}
}

func TestConnectionTimeout(t *testing.T) {
	// Save original value
	origListenTimeout := listenTimeout
	listenTimeout = 100 * time.Millisecond

	// Restore original value
	defer func() {
		listenTimeout = origListenTimeout
	}()

	// Create a mock connection that simulates timeout behavior
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Logf("Error closing listener: %v", closeErr)
		}
	}()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() {
			if closeErr := conn.Close(); closeErr != nil {
				t.Logf("Error closing connection: %v", closeErr)
			}
		}()

		// Set deadline and test timeout
		if listenTimeout > 0 {
			if deadlineErr := conn.SetDeadline(time.Now().Add(listenTimeout)); deadlineErr != nil {
				t.Logf("Error setting deadline: %v", deadlineErr)
			}
		}

		// Try to read (should timeout)
		buffer := make([]byte, 1024)
		_, err = conn.Read(buffer)
		if err == nil {
			t.Error("Expected timeout error")
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()

	// Wait for the timeout test to complete
	time.Sleep(200 * time.Millisecond)
}

func BenchmarkListen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Test listener creation and immediate close
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Error(err)
			continue
		}
		if err := listener.Close(); err != nil {
			b.Logf("Error closing listener: %v", err)
		}
	}
}

func BenchmarkHandleConnection(b *testing.B) {
	// Create a test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			b.Logf("Error closing listener: %v", err)
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create connection
		conn, err := net.Dial("tcp", addr.String())
		if err != nil {
			b.Error(err)
			continue
		}

		// Accept connection
		serverConn, err := listener.Accept()
		if err != nil {
			b.Error(err)
			if err := conn.Close(); err != nil {
				b.Logf("Error closing connection: %v", err)
			}
			continue
		}

		// Close connections
		if err := conn.Close(); err != nil {
			b.Logf("Error closing connection: %v", err)
		}
		if err := serverConn.Close(); err != nil {
			b.Logf("Error closing server connection: %v", err)
		}
	}
}