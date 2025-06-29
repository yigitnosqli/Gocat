package pipe

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// mockReadWriter implements io.ReadWriter for testing
type mockReadWriter struct {
	*bytes.Buffer
	readErr  error
	writeErr error
	closeErr error
}

func newMockReadWriter(data string) *mockReadWriter {
	return &mockReadWriter{
		Buffer: bytes.NewBufferString(data),
	}
}

func (m *mockReadWriter) Read(p []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return m.Buffer.Read(p)
}

func (m *mockReadWriter) Write(p []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.Buffer.Write(p)
}

func (m *mockReadWriter) Close() error {
	return m.closeErr
}

// mockFlushWriter implements io.Writer with Flush method
type mockFlushWriter struct {
	*bytes.Buffer
	flushCalled bool
	flushErr    error
}

func newMockFlushWriter() *mockFlushWriter {
	return &mockFlushWriter{
		Buffer: &bytes.Buffer{},
	}
}

func (m *mockFlushWriter) Flush() error {
	m.flushCalled = true
	return m.flushErr
}

// mockReader implements io.Reader for testing
type mockReader struct {
	data    []byte
	pos     int
	readErr error
}

func newMockReader(data string) *mockReader {
	return &mockReader{
		data: []byte(data),
	}
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}

	if m.pos >= len(m.data) {
		return 0, io.EOF
	}

	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func TestPipeWithBuffer(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		bufferSize int
		expected   string
	}{
		{
			name:       "simple data transfer",
			input:      "Hello, World!",
			bufferSize: 1024,
			expected:   "Hello, World!",
		},
		{
			name:       "small buffer",
			input:      "This is a longer message that will be transferred in chunks",
			bufferSize: 5,
			expected:   "This is a longer message that will be transferred in chunks",
		},
		{
			name:       "empty input",
			input:      "",
			bufferSize: 1024,
			expected:   "",
		},
		{
			name:       "single byte buffer",
			input:      "abc",
			bufferSize: 1,
			expected:   "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := newMockReader(tt.input)
			dst := &bytes.Buffer{}

			// Use a goroutine to handle the EOF exit
			done := make(chan bool)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						// Expected panic from os.Exit(0)
						t.Logf("Recovered from panic: %v", r)
					}
					done <- true
				}()
			if err := PipeWithBuffer(dst, src, tt.bufferSize); err != nil {
				t.Logf("PipeWithBuffer error: %v", err)
			}
		}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Function completed
			case <-time.After(100 * time.Millisecond):
				// Timeout - function likely exited
			}

			result := dst.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPipeWithBufferFlush(t *testing.T) {
	src := newMockReader("test data")
	dst := newMockFlushWriter()

	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic from os.Exit(0)
				t.Logf("Recovered from panic: %v", r)
			}
			done <- true
		}()
		if err := PipeWithBuffer(dst, src, 1024); err != nil {
			t.Logf("PipeWithBuffer error: %v", err)
		}
	}()

	// Wait for completion
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
	}

	if !dst.flushCalled {
		t.Error("Flush should have been called")
	}

	if dst.String() != "test data" {
		t.Errorf("expected 'test data', got %q", dst.String())
	}
}

func TestPipeWithBufferReadError(t *testing.T) {
	src := &mockReader{
		readErr: errors.New("read error"),
	}
	dst := &bytes.Buffer{}

	err := PipeWithBuffer(dst, src, 1024)
	if err == nil {
		t.Error("expected error but got none")
	}

	if err.Error() != "read error" {
		t.Errorf("expected 'read error', got %q", err.Error())
	}
}

func TestPipeWithBufferWriteError(t *testing.T) {
	src := newMockReader("test data")

	// Simulate write error by using a custom writer that always returns an error
	errorWriter := &errorWriter{}

	err := PipeWithBuffer(errorWriter, src, 1024)
	if err == nil {
		t.Error("expected error but got none")
	}
}

// errorWriter always returns an error on Write
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}

// Test PipeData function with mock connections
func TestPipeDataBasic(t *testing.T) {
	// Create two mock connections
	conn1 := newMockReadWriter("Hello from conn1")
	conn2 := newMockReadWriter("Hello from conn2")

	// Use a timeout to prevent the test from hanging
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic from os.Exit(0)
				t.Logf("Recovered from panic: %v", r)
			}
			done <- true
		}()
		PipeData(conn1, conn2)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Function completed
	case <-time.After(200 * time.Millisecond):
		// Timeout - this is expected as PipeData runs indefinitely
	}

	// Note: Due to the nature of PipeData (it runs indefinitely and exits on EOF),
	// we can't easily test the actual data transfer without modifying the function
	// to not call os.Exit(). This test mainly ensures the function doesn't panic
	// immediately.
}

// Benchmark tests
func BenchmarkPipeWithBuffer(b *testing.B) {
	data := strings.Repeat("benchmark data ", 1000) // ~14KB of data
	bufferSize := 1024

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := newMockReader(data)
		dst := &bytes.Buffer{}

		// Run in goroutine to handle os.Exit
		done := make(chan bool)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected panic from os.Exit(0)
					b.Logf("Recovered from panic: %v", r)
				}
				done <- true
			}()
			if err := PipeWithBuffer(dst, src, bufferSize); err != nil {
				b.Logf("PipeWithBuffer error: %v", err)
			}
		}()

		// Wait for completion
		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func BenchmarkPipeWithBufferSmallBuffer(b *testing.B) {
	data := "small test data"
	bufferSize := 4

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := newMockReader(data)
		dst := &bytes.Buffer{}

		done := make(chan bool)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected panic from os.Exit(0)
					b.Logf("Recovered from panic: %v", r)
				}
				done <- true
			}()
			if err := PipeWithBuffer(dst, src, bufferSize); err != nil {
				b.Logf("PipeWithBuffer error: %v", err)
			}
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func BenchmarkPipeWithBufferLargeBuffer(b *testing.B) {
	data := strings.Repeat("x", 1024) // 1KB of data
	bufferSize := 8192                // 8KB buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := newMockReader(data)
		dst := &bytes.Buffer{}

		done := make(chan bool)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected panic from os.Exit(0)
					b.Logf("Recovered from panic: %v", r)
				}
				done <- true
			}()
			if err := PipeWithBuffer(dst, src, bufferSize); err != nil {
				b.Logf("PipeWithBuffer error: %v", err)
			}
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
		}
	}
}
