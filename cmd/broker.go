package cmd

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	brokerPort     string
	brokerClients  map[string]net.Conn
	brokerMutex    sync.RWMutex
	brokerMaxConns int
)

var brokerCmd = &cobra.Command{
	Use:   "broker [port]",
	Short: "Start a broker mode for relaying connections",
	Long: `Start GoCat in broker mode. This mode allows multiple clients to connect
and relay data between them. Useful for creating a central hub for communication.`,
	Args: cobra.ExactArgs(1),
	Run:  runBroker,
}

func init() {
	rootCmd.AddCommand(brokerCmd)
	brokerCmd.Flags().IntVarP(&brokerMaxConns, "max-conns", "m", 10, "Maximum number of concurrent connections")
	brokerClients = make(map[string]net.Conn)
}

func runBroker(cmd *cobra.Command, args []string) {
	brokerPort = args[0]

	// Override with global flags if set
	if globalMaxConns, _ := cmd.Root().PersistentFlags().GetInt("max-conns"); globalMaxConns > 0 {
		brokerMaxConns = globalMaxConns
	}

	logger.Info("Starting broker mode on port %s (max connections: %d)", brokerPort, brokerMaxConns)

	if err := startBroker(brokerPort); err != nil {
		logger.Fatal("Broker error: %v", err)
	}
}

func startBroker(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to start broker listener: %w", err)
	}
	defer listener.Close()

	logger.Info("Broker listening on :%s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection: %v", err)
			continue
		}

		brokerMutex.Lock()
		if len(brokerClients) >= brokerMaxConns {
			logger.Warn("Maximum connections reached, rejecting %s", conn.RemoteAddr())
			if err := conn.Close(); err != nil {
				logger.Error("Failed to close connection: %v", err)
			}
			brokerMutex.Unlock()
			continue
		}

		clientID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().Unix())
		brokerClients[clientID] = conn
		brokerMutex.Unlock()

		logger.Info("Client connected: %s (ID: %s)", conn.RemoteAddr(), clientID)
		go handleBrokerClient(clientID, conn)
	}
}

func handleBrokerClient(clientID string, conn net.Conn) {
	defer func() {
		brokerMutex.Lock()
		delete(brokerClients, clientID)
		brokerMutex.Unlock()
		if err := conn.Close(); err != nil {
			logger.Error("Failed to close connection: %v", err)
		}
		logger.Info("Client disconnected: %s", clientID)
	}()

	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			logger.Debug("Client %s read error: %v", clientID, err)
			return
		}

		data := buffer[:n]
		logger.Debug("Received %d bytes from %s", n, clientID)

		// Broadcast to all other clients
		brokerMutex.RLock()
		for otherID, otherConn := range brokerClients {
			if otherID != clientID {
				if _, err := otherConn.Write(data); err != nil {
					logger.Error("Failed to write to client %s: %v", otherID, err)
				}
			}
		}
		brokerMutex.RUnlock()
	}
}
