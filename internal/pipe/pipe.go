package pipe

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// PipeData pipes data between two io.ReadWriter interfaces
func PipeData(conn1, conn2 io.ReadWriter) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Copy from conn1 to conn2
	go func() {
		defer wg.Done()
		if _, err := io.Copy(conn2, conn1); err != nil {
			logger.Warn("Connection lost")
			os.Exit(0)
		}
	}()

	// Copy from conn2 to conn1
	go func() {
		defer wg.Done()
		if _, err := io.Copy(conn1, conn2); err != nil {
			logger.Warn("Connection lost")
			os.Exit(0)
		}
	}()

	wg.Wait()
}

// PipeWithBuffer pipes data with a custom buffer size
func PipeWithBuffer(dst io.Writer, src io.Reader, bufferSize int) error {
	buffer := make([]byte, bufferSize)
	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err == io.EOF {
				logger.Warn("Connection lost")
				os.Exit(0)
			}
			return err
		}

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
