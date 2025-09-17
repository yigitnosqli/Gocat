package network

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides bandwidth limiting functionality
type RateLimiter struct {
	limiter *rate.Limiter
	burst   int
}

// with a minimum of 1024 bytes.
func NewRateLimiter(rateStr string) (*RateLimiter, error) {
	if rateStr == "" {
		return nil, nil // No rate limiting
	}

	bytesPerSecond, err := parseRateString(rateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid rate format: %w", err)
	}

	// Set burst to 10% of rate or minimum 1KB
	burst := int(float64(bytesPerSecond) * 0.1)
	if burst < 1024 {
		burst = 1024
	}

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
		burst:   burst,
	}, nil
}

// parseRateString parses a human-friendly rate string (e.g. "1MB/s", "500KB/s", "1.5MB/s")
// and returns the equivalent bytes-per-second value.
//
// The input is case-insensitive and may optionally end with "/s". If no unit is
// supplied the value is interpreted as bytes. Supported units are B/BYTE/BYTES,
// K/KB/KBYTE/KBYTES (1024), M/MB/MBYTE/MBYTES (1024^2) and G/GB/GBYTE/GBYTES (1024^3).
// Returns an error if the numeric portion cannot be parsed or the unit is unknown.
func parseRateString(rateStr string) (int64, error) {
	rateStr = strings.ToUpper(strings.TrimSpace(rateStr))

	// Remove /s suffix if present
	if strings.HasSuffix(rateStr, "/S") {
		rateStr = rateStr[:len(rateStr)-2]
	}

	// Parse number and unit
	var value float64
	var unit string

	for i, char := range rateStr {
		if (char < '0' || char > '9') && char != '.' {
			var err error
			value, err = strconv.ParseFloat(rateStr[:i], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", rateStr[:i])
			}
			unit = rateStr[i:]
			break
		}
	}

	// If no unit found, assume bytes
	if unit == "" {
		var err error
		value, err = strconv.ParseFloat(rateStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", rateStr)
		}
		unit = "B"
	}

	// Convert to bytes per second
	var multiplier int64
	switch unit {
	case "B", "BYTE", "BYTES":
		multiplier = 1
	case "K", "KB", "KBYTE", "KBYTES":
		multiplier = 1024
	case "M", "MB", "MBYTE", "MBYTES":
		multiplier = 1024 * 1024
	case "G", "GB", "GBYTE", "GBYTES":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return int64(value * float64(multiplier)), nil
}

// Wait waits for permission to transfer n bytes
func (rl *RateLimiter) Wait(ctx context.Context, n int) error {
	if rl == nil {
		return nil // No rate limiting
	}
	return rl.limiter.WaitN(ctx, n)
}

// Allow checks if n bytes can be transferred immediately
func (rl *RateLimiter) Allow(n int) bool {
	if rl == nil {
		return true // No rate limiting
	}
	return rl.limiter.AllowN(time.Now(), n)
}

// RateLimitedReader wraps an io.Reader with rate limiting
type RateLimitedReader struct {
	reader  io.Reader
	limiter *RateLimiter
	ctx     context.Context
}

// NewRateLimitedReader returns a RateLimitedReader that wraps the provided io.Reader.
// If limiter is nil the returned reader performs no rate limiting. The returned
// reader uses context.Background() for limiter waits; use NewRateLimitedReaderWithContext
// to supply a custom context.
func NewRateLimitedReader(reader io.Reader, limiter *RateLimiter) *RateLimitedReader {
	return &RateLimitedReader{
		reader:  reader,
		limiter: limiter,
		ctx:     context.Background(),
	}
}

// NewRateLimitedReaderWithContext returns a RateLimitedReader that wraps the provided io.Reader and uses the given context for limiter waits.
// If limiter is nil the returned reader performs reads without rate limiting.
func NewRateLimitedReaderWithContext(ctx context.Context, reader io.Reader, limiter *RateLimiter) *RateLimitedReader {
	return &RateLimitedReader{
		reader:  reader,
		limiter: limiter,
		ctx:     ctx,
	}
}

// Read implements io.Reader with rate limiting
func (r *RateLimitedReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 && r.limiter != nil {
		if waitErr := r.limiter.Wait(r.ctx, n); waitErr != nil {
			return n, waitErr
		}
	}
	return n, err
}

// RateLimitedWriter wraps an io.Writer with rate limiting
type RateLimitedWriter struct {
	writer  io.Writer
	limiter *RateLimiter
	ctx     context.Context
}

// NewRateLimitedWriter returns a RateLimitedWriter that wraps the provided io.Writer.
// The returned writer uses a background context for limiter waits. If limiter is nil,
// writes through without rate limiting.
func NewRateLimitedWriter(writer io.Writer, limiter *RateLimiter) *RateLimitedWriter {
	return &RateLimitedWriter{
		writer:  writer,
		limiter: limiter,
		ctx:     context.Background(),
	}
}

// NewRateLimitedWriterWithContext creates a RateLimitedWriter that wraps the provided io.Writer and
// uses the supplied context for limiter waits. If limiter is nil the returned writer performs no
// rate limiting.
func NewRateLimitedWriterWithContext(ctx context.Context, writer io.Writer, limiter *RateLimiter) *RateLimitedWriter {
	return &RateLimitedWriter{
		writer:  writer,
		limiter: limiter,
		ctx:     ctx,
	}
}

// Write implements io.Writer with rate limiting
func (w *RateLimitedWriter) Write(p []byte) (n int, err error) {
	if w.limiter != nil {
		if waitErr := w.limiter.Wait(w.ctx, len(p)); waitErr != nil {
			return 0, waitErr
		}
	}
	return w.writer.Write(p)
}

// RateLimitedConn wraps a net.Conn with rate limiting
type RateLimitedConn struct {
	conn         io.ReadWriteCloser
	readLimiter  *RateLimiter
	writeLimiter *RateLimiter
	ctx          context.Context
}

// NewRateLimitedConn returns a RateLimitedConn that wraps conn and applies separate optional rate limits for reads and writes.
// If readLimiter or writeLimiter is nil, the corresponding direction is not rate limited. The returned connection uses a
// background context for limiter waits.
func NewRateLimitedConn(conn io.ReadWriteCloser, readLimiter, writeLimiter *RateLimiter) *RateLimitedConn {
	return &RateLimitedConn{
		conn:         conn,
		readLimiter:  readLimiter,
		writeLimiter: writeLimiter,
		ctx:          context.Background(),
	}
}

// NewRateLimitedConnWithContext creates a RateLimitedConn that wraps the provided io.ReadWriteCloser
// and enforces separate read and write rate limits using the given context.
// If readLimiter or writeLimiter is nil the corresponding direction is not rate limited.
func NewRateLimitedConnWithContext(ctx context.Context, conn io.ReadWriteCloser, readLimiter, writeLimiter *RateLimiter) *RateLimitedConn {
	return &RateLimitedConn{
		conn:         conn,
		readLimiter:  readLimiter,
		writeLimiter: writeLimiter,
		ctx:          ctx,
	}
}

// Read implements io.Reader with rate limiting
func (c *RateLimitedConn) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	if n > 0 && c.readLimiter != nil {
		if waitErr := c.readLimiter.Wait(c.ctx, n); waitErr != nil {
			return n, waitErr
		}
	}
	return n, err
}

// Write implements io.Writer with rate limiting
func (c *RateLimitedConn) Write(p []byte) (n int, err error) {
	if c.writeLimiter != nil {
		if waitErr := c.writeLimiter.Wait(c.ctx, len(p)); waitErr != nil {
			return 0, waitErr
		}
	}
	return c.conn.Write(p)
}

// Close implements io.Closer
func (c *RateLimitedConn) Close() error {
	return c.conn.Close()
}
