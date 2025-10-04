package cmd

import (
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ibrahmsql/gocat/internal/logger"
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

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVar(&convertFrom, "from", "", "Source protocol and address (e.g., tcp:8080, udp:8080, http:8080)")
	convertCmd.Flags().StringVar(&convertTo, "to", "", "Target protocol and address (e.g., tcp:host:9000, udp:host:9000)")
	convertCmd.Flags().IntVar(&convertBuffer, "buffer", 8192, "Buffer size for data transfer")
	
	convertCmd.MarkFlagRequired("from")
	convertCmd.MarkFlagRequired("to")
}

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

func parseProtocolAddress(addr string) (string, string) {
	parts := strings.SplitN(addr, ":", 2)
	if len(parts) != 2 {
		logger.Fatal("Invalid protocol address format: %s (expected protocol:address)", addr)
	}
	return parts[0], parts[1]
}

// TCP to UDP conversion
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

// UDP to TCP conversion
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

// TCP to TCP (simple proxy)
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

// UDP to UDP (simple proxy)
func udpToUDP(listenAddr, targetAddr string) {
	logger.Info("UDP->UDP proxy listening on %s, forwarding to %s", listenAddr, targetAddr)
	
	// Similar to udpToTCP but with UDP target
	// Implementation similar to above
	logger.Warn("UDP->UDP conversion not yet implemented")
}

// TCP to WebSocket
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

// HTTP to WebSocket
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

// WebSocket to TCP
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

// WebSocket to HTTP
func webSocketToHTTP(wsAddr, httpURL string) {
	logger.Warn("WebSocket->HTTP conversion not yet fully implemented")
	logger.Debug("Would listen on %s and forward to %s", wsAddr, httpURL)
	// This would require buffering and request/response matching
	// TODO: Implement WebSocket message to HTTP request conversion
}
