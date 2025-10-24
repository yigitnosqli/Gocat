package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	wsServerPort     string
	wsServerPath     string
	wsClientURL      string
	wsReadBufferSize int
	wsWriteBufferSize int
	wsEnableCompression bool
	wsOrigin         string
	wsPingInterval   time.Duration
	wsPongTimeout    time.Duration
	wsMaxMessageSize int64
)

// websocketCmd represents the WebSocket parent command
var websocketCmd = &cobra.Command{
	Use:   "websocket",
	Aliases: []string{"ws"},
	Short: "WebSocket server and client operations",
	Long: `WebSocket server and client for bidirectional communication.

Examples:
  # Start WebSocket server
  gocat ws server --port 8080

  # Connect to WebSocket server
  gocat ws connect ws://localhost:8080

  # WebSocket with compression
  gocat ws server --port 8080 --compress

  # WebSocket echo server
  gocat ws echo --port 8080`,
}

// wsServerCmd handles WebSocket server
var wsServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a WebSocket server",
	Long: `Start a WebSocket server that accepts connections and relays data.

The server will accept WebSocket connections and relay data between
stdin/stdout and the WebSocket connection.`,
	RunE: runWSServer,
}

// wsClientCmd handles WebSocket client
var wsClientCmd = &cobra.Command{
	Use:   "connect [ws://host:port/path]",
	Aliases: []string{"client", "c"},
	Short: "Connect to a WebSocket server",
	Long: `Connect to a WebSocket server and relay data between stdin/stdout.

Examples:
  gocat ws connect ws://localhost:8080
  gocat ws connect wss://secure.example.com/ws
  echo "Hello" | gocat ws connect ws://localhost:8080`,
	Args: cobra.ExactArgs(1),
	RunE: runWSClient,
}

// wsEchoCmd handles WebSocket echo server
var wsEchoCmd = &cobra.Command{
	Use:   "echo",
	Short: "Start a WebSocket echo server",
	Long:  `Start a WebSocket echo server that echoes back all received messages.`,
	RunE:  runWSEcho,
}

func init() {
	rootCmd.AddCommand(websocketCmd)
	websocketCmd.AddCommand(wsServerCmd)
	websocketCmd.AddCommand(wsClientCmd)
	websocketCmd.AddCommand(wsEchoCmd)

	// Server flags
	wsServerCmd.Flags().StringVar(&wsServerPort, "port", "8080", "Port to listen on")
	wsServerCmd.Flags().StringVar(&wsServerPath, "path", "/", "WebSocket endpoint path")
	wsServerCmd.Flags().BoolVar(&wsEnableCompression, "compress", false, "Enable WebSocket compression")
	wsServerCmd.Flags().IntVar(&wsReadBufferSize, "read-buffer", 4096, "Read buffer size")
	wsServerCmd.Flags().IntVar(&wsWriteBufferSize, "write-buffer", 4096, "Write buffer size")
	wsServerCmd.Flags().Int64Var(&wsMaxMessageSize, "max-message-size", 512*1024, "Maximum message size in bytes")
	wsServerCmd.Flags().DurationVar(&wsPingInterval, "ping-interval", 30*time.Second, "Ping interval")
	wsServerCmd.Flags().DurationVar(&wsPongTimeout, "pong-timeout", 60*time.Second, "Pong timeout")

	// Client flags
	wsClientCmd.Flags().StringVar(&wsOrigin, "origin", "", "Origin header for WebSocket handshake")
	wsClientCmd.Flags().BoolVar(&wsEnableCompression, "compress", false, "Enable WebSocket compression")
	wsClientCmd.Flags().IntVar(&wsReadBufferSize, "read-buffer", 4096, "Read buffer size")
	wsClientCmd.Flags().IntVar(&wsWriteBufferSize, "write-buffer", 4096, "Write buffer size")
	wsClientCmd.Flags().DurationVar(&wsPingInterval, "ping-interval", 30*time.Second, "Ping interval")

	// Echo server flags
	wsEchoCmd.Flags().StringVar(&wsServerPort, "port", "8080", "Port to listen on")
	wsEchoCmd.Flags().StringVar(&wsServerPath, "path", "/", "WebSocket endpoint path")
	wsEchoCmd.Flags().BoolVar(&wsEnableCompression, "compress", false, "Enable WebSocket compression")
}

func runWSServer(cmd *cobra.Command, args []string) error {
	logger.Info("Starting WebSocket server on port %s%s", wsServerPort, wsServerPath)

	upgrader := websocket.Upgrader{
		ReadBufferSize:    wsReadBufferSize,
		WriteBufferSize:   wsWriteBufferSize,
		EnableCompression: wsEnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}

	http.HandleFunc(wsServerPath, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		logger.Info("WebSocket connection established from %s", r.RemoteAddr)

		// Configure connection
		conn.SetReadLimit(wsMaxMessageSize)
		conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
			return nil
		})

		// Start ping ticker
		ticker := time.NewTicker(wsPingInterval)
		defer ticker.Stop()

		// Handle bidirectional communication
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Read from WebSocket, write to stdout
		go func() {
			for {
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						logger.Error("WebSocket read error: %v", err)
					}
					cancel()
					return
				}

				if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
					if _, err := os.Stdout.Write(message); err != nil {
						logger.Error("Stdout write error: %v", err)
						cancel()
						return
					}
				}
			}
		}()

		// Read from stdin, write to WebSocket
		go func() {
			buffer := make([]byte, 4096)
			for {
				n, err := os.Stdin.Read(buffer)
				if err != nil {
					if err != io.EOF {
						logger.Error("Stdin read error: %v", err)
					}
					cancel()
					return
				}

				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
						logger.Error("WebSocket write error: %v", err)
						cancel()
						return
					}
				}
			}
		}()

		// Ping loop
		go func() {
			for {
				select {
				case <-ticker.C:
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						cancel()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		<-ctx.Done()
		logger.Info("WebSocket connection closed")
	})

	// Start server
	server := &http.Server{
		Addr:         ":" + wsServerPort,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down WebSocket server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	logger.Info("WebSocket server listening on http://localhost:%s%s", wsServerPort, wsServerPath)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func runWSClient(cmd *cobra.Command, args []string) error {
	url := args[0]
	logger.Info("Connecting to WebSocket server: %s", url)

	dialer := websocket.Dialer{
		ReadBufferSize:    wsReadBufferSize,
		WriteBufferSize:   wsWriteBufferSize,
		EnableCompression: wsEnableCompression,
		HandshakeTimeout:  10 * time.Second,
	}

	headers := http.Header{}
	if wsOrigin != "" {
		headers.Set("Origin", wsOrigin)
	}

	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	logger.Info("WebSocket connection established")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start ping ticker
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	// Read from WebSocket, write to stdout
	go func() {
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error("WebSocket read error: %v", err)
				}
				cancel()
				return
			}

			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				if _, err := os.Stdout.Write(message); err != nil {
					logger.Error("Stdout write error: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// Read from stdin, write to WebSocket
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Stdin read error: %v", err)
				}
				cancel()
				return
			}

			if n > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
					logger.Error("WebSocket write error: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// Ping loop
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for interrupt or context cancellation
	select {
	case <-sigChan:
		logger.Info("Interrupted, closing connection...")
	case <-ctx.Done():
	}

	// Clean close
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(100 * time.Millisecond)

	return nil
}

func runWSEcho(cmd *cobra.Command, args []string) error {
	logger.Info("Starting WebSocket echo server on port %s%s", wsServerPort, wsServerPath)

	upgrader := websocket.Upgrader{
		ReadBufferSize:    wsReadBufferSize,
		WriteBufferSize:   wsWriteBufferSize,
		EnableCompression: wsEnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	http.HandleFunc(wsServerPath, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		logger.Info("Echo connection established from %s", r.RemoteAddr)

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error("Read error: %v", err)
				}
				break
			}

			logger.Debug("Echoing %d bytes (type: %d)", len(message), messageType)

			if err := conn.WriteMessage(messageType, message); err != nil {
				logger.Error("Write error: %v", err)
				break
			}
		}

		logger.Info("Echo connection closed")
	})

	server := &http.Server{
		Addr:         ":" + wsServerPort,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down echo server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	logger.Info("WebSocket echo server listening on http://localhost:%s%s", wsServerPort, wsServerPath)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
