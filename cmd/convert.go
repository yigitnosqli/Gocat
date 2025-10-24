package cmd

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ibrahmsql/gocat/internal/logger"
	wsconv "github.com/ibrahmsql/gocat/internal/websocket"
	"github.com/spf13/cobra"
)

var (
	convertFrom   string
	convertTo     string
	convertBuffer int
)

var convertCmd = &cobra.Command{
	Use:     "convert",
	Aliases: []string{"conv", "protocol-convert"},
	Short:   "Convert between different network protocols",
	Long: `Convert network traffic between different protocols.
Supports TCP, UDP, HTTP, and WebSocket conversions.

Examples:
  # TCP to UDP
  gocat convert --from tcp:8080 --to udp:9000

  # UDP to TCP
  gocat convert --from udp:8080 --to tcp:9000

  # HTTP to WebSocket
  gocat convert --from http:8080 --to ws://backend:9000/ws

  # WebSocket to TCP
  gocat convert --from ws:8080 --to tcp:backend:9000
`,
	Run: runConvert,
}

// init registers the "convert" command with the root command and defines its CLI flags.
//
// It adds the --from and --to string flags for specifying source and target
// protocol:address pairs (required), and the --buffer int flag for configuring
// the data transfer buffer size.
func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVar(&convertFrom, "from", "", "Source protocol and address (e.g., tcp:8080, udp:8080, http:8080)")
	convertCmd.Flags().StringVar(&convertTo, "to", "", "Target protocol and address (e.g., tcp:host:9000, udp:host:9000)")
	convertCmd.Flags().IntVar(&convertBuffer, "buffer", 8192, "Buffer size for data transfer")

	convertCmd.MarkFlagRequired("from")
	convertCmd.MarkFlagRequired("to")
}

// runConvert parses the global convertFrom and convertTo flags, logs the conversion start, and dispatches to the appropriate protocol conversion handler.
// It invokes the corresponding converter (e.g., tcpToUDP, httpToWebSocket) and exits with a fatal log if the source or target protocol is unsupported.
func runConvert(cmd *cobra.Command, args []string) {
	fromProto, fromAddr := parseProtocolAddress(convertFrom)
	toProto, toAddr := parseProtocolAddress(convertTo)

	logger.Info("Starting protocol converter: %s:%s -> %s:%s", fromProto, fromAddr, toProto, toAddr)

	switch fromProto {
	case "tcp":
		switch toProto {
		case "udp":
			tcpToUDP(fromAddr, toAddr)
		case "tcp":
			tcpToTCP(fromAddr, toAddr)
		case "ws", "websocket":
			tcpToWebSocket(fromAddr, toAddr)
		default:
			logger.Fatal("Unsupported conversion: tcp -> %s", toProto)
		}

	case "udp":
		switch toProto {
		case "tcp":
			udpToTCP(fromAddr, toAddr)
		case "udp":
			udpToUDP(fromAddr, toAddr)
		default:
			logger.Fatal("Unsupported conversion: udp -> %s", toProto)
		}

	case "http":
		switch toProto {
		case "ws", "websocket":
			httpToWebSocket(fromAddr, toAddr)
		default:
			logger.Fatal("Unsupported conversion: http -> %s", toProto)
		}

	case "ws", "websocket":
		switch toProto {
		case "tcp":
			webSocketToTCP(fromAddr, toAddr)
		case "http":
			webSocketToHTTP(fromAddr, toAddr)
		default:
			logger.Fatal("Unsupported conversion: websocket -> %s", toProto)
		}

	default:
		logger.Fatal("Unsupported source protocol: %s", fromProto)
	}
}

// parseProtocolAddress splits an input of the form "protocol:address" and returns
// the protocol and the address parts.
//
// If the input does not contain a single ':' separator, the function logs a fatal
// error and exits the program.
func parseProtocolAddress(addr string) (string, string) {
	parts := strings.SplitN(addr, ":", 2)
	if len(parts) != 2 {
		logger.Fatal("Invalid protocol address format: %s (expected protocol:address)", addr)
	}
	return parts[0], parts[1]
}

// tcpToUDP starts a TCP listener on tcpAddr and, for each accepted connection,
// launches a goroutine to forward bidirectional traffic between that TCP
// connection and the UDP address udpAddr via handleTCPToUDP.
// It logs a fatal error and exits if it cannot start listening on tcpAddr.
func tcpToUDP(tcpAddr, udpAddr string) {
	listener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		logger.Fatal("Failed to listen on TCP %s: %v", tcpAddr, err)
	}
	defer listener.Close()

	logger.Info("TCP->UDP converter listening on %s, forwarding to %s", tcpAddr, udpAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		go handleTCPToUDP(conn, udpAddr)
	}
}

// handleTCPToUDP bridges data between a TCP connection and a remote UDP address.
//
// It forwards bytes read from the TCP connection to the UDP address and forwards
// packets read from the UDP connection back to the TCP connection until either
// side closes or an I/O error occurs. The TCP connection is closed when the
// function returns; the UDP connection is created for the duration of the
// function. Errors encountered while reading or writing are logged.
func handleTCPToUDP(tcpConn net.Conn, udpAddr string) {
	defer tcpConn.Close()

	udpConn, err := net.Dial("udp", udpAddr)
	if err != nil {
		logger.Error("Failed to connect to UDP %s: %v", udpAddr, err)
		return
	}
	defer udpConn.Close()

	logger.Debug("TCP->UDP: %s -> %s", tcpConn.RemoteAddr(), udpAddr)

	var wg sync.WaitGroup
	wg.Add(2)

	// TCP to UDP
	go func() {
		defer wg.Done()
		buf := make([]byte, convertBuffer)
		for {
			n, err := tcpConn.Read(buf)
			if err != nil {
				return
			}
			if _, err := udpConn.Write(buf[:n]); err != nil {
				logger.Error("UDP write error: %v", err)
				return
			}
		}
	}()

	// UDP to TCP
	go func() {
		defer wg.Done()
		buf := make([]byte, convertBuffer)
		for {
			n, err := udpConn.Read(buf)
			if err != nil {
				return
			}
			if _, err := tcpConn.Write(buf[:n]); err != nil {
				logger.Error("TCP write error: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}

// udpToTCP listens for UDP packets on udpAddr and forwards each UDP client's datagrams
// over a dedicated TCP connection to tcpAddr.
//
// For each distinct UDP client address it creates (and reuses) a TCP connection to the
// target address. Datagrams received from a UDP client are written to that client's
// TCP connection, and responses read from the TCP connection are sent back to the
// originating UDP client. When a TCP connection closes or encounters an error it is
// closed and removed from the client map; the function continues serving other clients.
func udpToTCP(udpAddr, tcpAddr string) {
	udpConn, err := net.ListenPacket("udp", udpAddr)
	if err != nil {
		logger.Fatal("Failed to listen on UDP %s: %v", udpAddr, err)
	}
	defer udpConn.Close()

	logger.Info("UDP->TCP converter listening on %s, forwarding to %s", udpAddr, tcpAddr)

	clients := make(map[string]net.Conn)
	var mu sync.Mutex

	buf := make([]byte, convertBuffer)
	for {
		n, clientAddr, err := udpConn.ReadFrom(buf)
		if err != nil {
			logger.Error("UDP read error: %v", err)
			continue
		}

		clientKey := clientAddr.String()

		mu.Lock()
		tcpConn, exists := clients[clientKey]
		if !exists {
			tcpConn, err = net.Dial("tcp", tcpAddr)
			if err != nil {
				logger.Error("Failed to connect to TCP %s: %v", tcpAddr, err)
				mu.Unlock()
				continue
			}
			clients[clientKey] = tcpConn

			// Handle TCP responses
			go func(conn net.Conn, addr net.Addr) {
				defer func() {
					conn.Close()
					mu.Lock()
					delete(clients, addr.String())
					mu.Unlock()
				}()

				respBuf := make([]byte, convertBuffer)
				for {
					n, err := conn.Read(respBuf)
					if err != nil {
						return
					}
					udpConn.WriteTo(respBuf[:n], addr)
				}
			}(tcpConn, clientAddr)
		}
		mu.Unlock()

		if _, err := tcpConn.Write(buf[:n]); err != nil {
			logger.Error("TCP write error: %v", err)
		}
	}
}

// tcpToTCP starts a TCP proxy that listens on listenAddr and forwards each incoming connection to targetAddr.
// For each accepted client it dials the target and copies data bidirectionally between the client and target until either side closes.
// It logs the listening state, calls logger.Fatal if the initial listen fails, and logs accept/connect/runtime errors.
func tcpToTCP(listenAddr, targetAddr string) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal("Failed to listen on TCP %s: %v", listenAddr, err)
	}
	defer listener.Close()

	logger.Info("TCP->TCP proxy listening on %s, forwarding to %s", listenAddr, targetAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			target, err := net.Dial("tcp", targetAddr)
			if err != nil {
				logger.Error("Failed to connect to %s: %v", targetAddr, err)
				return
			}
			defer target.Close()

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				io.Copy(target, c)
			}()

			go func() {
				defer wg.Done()
				io.Copy(c, target)
			}()

			wg.Wait()
		}(conn)
	}
}

// udpToUDP starts a UDP proxy that listens on listenAddr and forwards packets to targetAddr.
// It creates a UDP listener and forwards all received packets to the target UDP address.
func udpToUDP(listenAddr, targetAddr string) {
	logger.Info("UDP->UDP proxy listening on %s, forwarding to %s", listenAddr, targetAddr)

	// Resolve target address
	targetUDPAddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		logger.Fatal("Failed to resolve target UDP address %s: %v", targetAddr, err)
	}

	// Listen on UDP
	listenUDPAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		logger.Fatal("Failed to resolve listen UDP address %s: %v", listenAddr, err)
	}

	udpConn, err := net.ListenUDP("udp", listenUDPAddr)
	if err != nil {
		logger.Fatal("Failed to listen on UDP %s: %v", listenAddr, err)
	}
	defer udpConn.Close()

	logger.Info("UDP->UDP converter started")

	// Map to track client connections
	type clientInfo struct {
		addr       *net.UDPAddr
		targetConn *net.UDPConn
		lastSeen   time.Time
	}
	
	clients := make(map[string]*clientInfo)
	var clientsMu sync.Mutex

	// Cleanup goroutine for stale connections
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			clientsMu.Lock()
			now := time.Now()
			for key, info := range clients {
				if now.Sub(info.lastSeen) > 2*time.Minute {
					logger.Debug("Cleaning up stale UDP client: %s", key)
					info.targetConn.Close()
					delete(clients, key)
				}
			}
			clientsMu.Unlock()
		}
	}()

	buffer := make([]byte, 65535)
	for {
		n, clientAddr, err := udpConn.ReadFromUDP(buffer)
		if err != nil {
			logger.Error("UDP read error: %v", err)
			continue
		}

		clientKey := clientAddr.String()

		clientsMu.Lock()
		client, exists := clients[clientKey]
		if !exists {
			// Create new connection to target for this client
			targetConn, err := net.DialUDP("udp", nil, targetUDPAddr)
			if err != nil {
				logger.Error("Failed to dial target UDP: %v", err)
				clientsMu.Unlock()
				continue
			}

			client = &clientInfo{
				addr:       clientAddr,
				targetConn: targetConn,
				lastSeen:   time.Now(),
			}
			clients[clientKey] = client

			// Start goroutine to read responses from target
			go func(c *clientInfo) {
				respBuffer := make([]byte, 65535)
				for {
					n, err := c.targetConn.Read(respBuffer)
					if err != nil {
						if !isClosedError(err) {
							logger.Error("Target UDP read error: %v", err)
						}
						return
					}

					// Send response back to original client
					if _, err := udpConn.WriteToUDP(respBuffer[:n], c.addr); err != nil {
						logger.Error("Failed to write response to client: %v", err)
						return
					}

					clientsMu.Lock()
					c.lastSeen = time.Now()
					clientsMu.Unlock()
				}
			}(client)

			logger.Debug("New UDP client: %s", clientKey)
		} else {
			client.lastSeen = time.Now()
		}
		clientsMu.Unlock()

		// Forward packet to target
		if _, err := client.targetConn.Write(buffer[:n]); err != nil {
			logger.Error("Failed to forward UDP packet: %v", err)
			continue
		}

		logger.Debug("Forwarded %d bytes from %s to %s", n, clientAddr, targetAddr)
	}
}

// isClosedError checks if the error is due to a closed connection
func isClosedError(err error) bool {
	return err != nil && (err.Error() == "use of closed network connection" || 
		err == io.EOF)
}

// tcpToWebSocket starts a TCP listener on tcpAddr and forwards each accepted TCP connection to a backend WebSocket at wsURL.
// tcpAddr is the address to listen on (for example ":8080" or "0.0.0.0:9000").
// wsURL is the target WebSocket URL (for example "ws://host:port/path").
// For each incoming TCP connection the function delegates forwarding to handleTCPToWebSocket and continues accepting new connections.
func tcpToWebSocket(tcpAddr, wsURL string) {
	listener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		logger.Fatal("Failed to listen on TCP %s: %v", tcpAddr, err)
	}
	defer listener.Close()

	logger.Info("TCP->WebSocket converter listening on %s, forwarding to %s", tcpAddr, wsURL)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		go handleTCPToWebSocket(conn, wsURL)
	}
}

// handleTCPToWebSocket bridges data between a local TCP connection and a remote WebSocket URL.
// It forwards raw bytes read from the TCP connection as binary WebSocket messages to the given
// wsURL and writes binary WebSocket messages received from wsURL back to the TCP connection.
// Both the TCP and WebSocket connections are closed when the bridge ends; runtime errors are logged.
func handleTCPToWebSocket(tcpConn net.Conn, wsURL string) {
	defer tcpConn.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		logger.Error("Failed to connect to WebSocket %s: %v", wsURL, err)
		return
	}
	defer wsConn.Close()

	logger.Debug("TCP->WebSocket: %s -> %s", tcpConn.RemoteAddr(), wsURL)

	var wg sync.WaitGroup
	wg.Add(2)

	// TCP to WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, convertBuffer)
		for {
			n, err := tcpConn.Read(buf)
			if err != nil {
				return
			}
			if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				logger.Error("WebSocket write error: %v", err)
				return
			}
		}
	}()

	// WebSocket to TCP
	go func() {
		defer wg.Done()
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			if _, err := tcpConn.Write(message); err != nil {
				logger.Error("TCP write error: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}

// The function blocks serving requests and logs fatal on server errors. It returns after ListenAndServe fails.
func httpToWebSocket(httpAddr, wsURL string) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clientWS, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade error: %v", err)
			return
		}
		defer clientWS.Close()

		backendWS, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			logger.Error("Failed to connect to backend WebSocket: %v", err)
			return
		}
		defer backendWS.Close()

		var wg sync.WaitGroup
		wg.Add(2)

		// Client to backend
		go func() {
			defer wg.Done()
			for {
				msgType, message, err := clientWS.ReadMessage()
				if err != nil {
					return
				}
				if err := backendWS.WriteMessage(msgType, message); err != nil {
					return
				}
			}
		}()

		// Backend to client
		go func() {
			defer wg.Done()
			for {
				msgType, message, err := backendWS.ReadMessage()
				if err != nil {
					return
				}
				if err := clientWS.WriteMessage(msgType, message); err != nil {
					return
				}
			}
		}()

		wg.Wait()
	})

	logger.Info("HTTP->WebSocket converter listening on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		logger.Fatal("HTTP server error: %v", err)
	}
}

// webSocketToTCP upgrades incoming HTTP requests at wsAddr to WebSocket connections and proxies bidirectional binary data between each WebSocket client and a TCP server at tcpAddr.
//
// For each upgraded WebSocket connection a TCP connection to tcpAddr is established; messages received from the WebSocket are written to the TCP connection, and bytes read from the TCP connection are sent back to the WebSocket as binary messages. The function starts an HTTP server that listens on wsAddr and blocks until the server stops; errors are logged and fatal errors terminate the process.
func webSocketToTCP(wsAddr, tcpAddr string) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade error: %v", err)
			return
		}
		defer wsConn.Close()

		tcpConn, err := net.Dial("tcp", tcpAddr)
		if err != nil {
			logger.Error("Failed to connect to TCP %s: %v", tcpAddr, err)
			return
		}
		defer tcpConn.Close()

		var wg sync.WaitGroup
		wg.Add(2)

		// WebSocket to TCP
		go func() {
			defer wg.Done()
			for {
				_, message, err := wsConn.ReadMessage()
				if err != nil {
					return
				}
				if _, err := tcpConn.Write(message); err != nil {
					return
				}
			}
		}()

		// TCP to WebSocket
		go func() {
			defer wg.Done()
			buf := make([]byte, convertBuffer)
			for {
				n, err := tcpConn.Read(buf)
				if err != nil {
					return
				}
				if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}()

		wg.Wait()
	})

	logger.Info("WebSocket->TCP converter listening on %s", wsAddr)
	if err := http.ListenAndServe(wsAddr, nil); err != nil {
		logger.Fatal("HTTP server error: %v", err)
	}
}

// webSocketToHTTP listens for WebSocket connections on wsAddr and forwards their messages to the HTTP endpoint at httpURL.
func webSocketToHTTP(wsAddr, httpURL string) {
	logger.Info("Starting WebSocket->HTTP converter: %s -> %s", wsAddr, httpURL)

	converter, err := wsconv.NewWebSocketToHTTPConverter(wsAddr, httpURL)
	if err != nil {
		logger.Error("Failed to create WebSocket->HTTP converter: %v", err)
		return
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down WebSocket->HTTP converter...")
		converter.Shutdown()
	}()

	logger.Info("WebSocket->HTTP converter started successfully")

	// Keep the converter running
	select {}
}
