package pipe

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// PipeData pipes data between two io.ReadWriter interfaces with graceful shutdown
func PipeData(conn1, conn2 io.ReadWriter) {
	PipeDataWithContext(context.Background(), conn1, conn2)
}

// PipeDataWithContext pipes data between two io.ReadWriter interfaces with context support
func PipeDataWithContext(ctx context.Context, conn1, conn2 io.ReadWriter) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in PipeDataWithContext: %v", r)
		}
	}()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(2)

	// Copy from conn1 to conn2
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic in pipe goroutine (conn1->conn2): %v", r)
			}
			wg.Done()
			cancel() // Cancel context when this goroutine exits
		}()
		if _, err := copyWithContext(ctx, conn2, conn1); err != nil {
			if ctx.Err() == nil { // Only log if not cancelled
				logger.Warn("Connection lost: %v", err)
			}
		}
	}()

	// Copy from conn2 to conn1
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic in pipe goroutine (conn2->conn1): %v", r)
			}
			wg.Done()
			cancel() // Cancel context when this goroutine exits
		}()
		if _, err := copyWithContext(ctx, conn1, conn2); err != nil {
			if ctx.Err() == nil { // Only log if not cancelled
				logger.Warn("Connection lost: %v", err)
			}
		}
	}()

	wg.Wait()
}

// copyWithContext copies data with context cancellation support
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in copyWithContext: %v", r)
		}
	}()

	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		// Set read timeout if possible
		if conn, ok := src.(interface{ SetReadDeadline(time.Time) error }); ok {
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return written, er
			}
			break
		}
	}
	return written, nil
}

// PipeWithBuffer pipes data with a custom buffer size and graceful shutdown
func PipeWithBuffer(dst io.Writer, src io.Reader, bufferSize int) error {
	return PipeWithBufferContext(context.Background(), dst, src, bufferSize)
}

// PipeWithBufferContext pipes data with a custom buffer size and context support
func PipeWithBufferContext(ctx context.Context, dst io.Writer, src io.Reader, bufferSize int) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in PipeWithBufferContext: %v", r)
		}
	}()

	if bufferSize <= 0 {
		bufferSize = 32 * 1024 // Default 32KB
	}

	buffer := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Set read timeout if possible
		if conn, ok := src.(interface{ SetReadDeadline(time.Time) error }); ok {
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		}

		n, err := src.Read(buffer)
		if err != nil {
			if err == io.EOF {
				return nil // Normal termination
			}
			// Check if it's a timeout error and context is still valid
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				continue // Continue on timeout if context is still valid
			}
			return err
		}

		if n > 0 {
			if _, err := dst.Write(buffer[:n]); err != nil {
				return err
			}

			// Flush if possible
			if flusher, ok := dst.(interface{ Flush() error }); ok {
				if err := flusher.Flush(); err != nil {
					return fmt.Errorf("flush error: %v", err)
				}
			}
		}
	}
}
