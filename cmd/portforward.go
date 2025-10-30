package cmd

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	pfLocalPort   int
	pfRemoteHost  string
	pfProtocol    string
	pfMaxConns    int
	pfIdleTimeout time.Duration
	pfBufferSize  int
	pfVerbose     bool
)

// Port forwarding statistics
type ForwardStats struct {
	TotalConnections  int64
	ActiveConnections int64
	BytesForwarded    int64
	ErrorCount        int64
	mu                sync.RWMutex
}

var stats = &ForwardStats{}

var portforwardCmd = &cobra.Command{
	Use:     "portforward",
	Aliases: []string{"pf", "forward"},
	Short:   "Port forwarding / redirection",
	Long: `Forward connections from a local port to a remote host:port.
	
Supports TCP and UDP port forwarding with connection pooling and statistics.`,
	Example: `  # Forward local port 8080 to remote server
  gocat portforward -l 8080 -r example.com:80
  
  # UDP port forwarding
  gocat portforward -l 53 -r 8.8.8.8:53 --udp
  
  # With connection limit
  gocat portforward -l 3306 -r db.internal:3306 --max-conns 10`,
	Run: runPortForward,
}

func init() {
	rootCmd.AddCommand(portforwardCmd)

	portforwardCmd.Flags().IntVar(&pfLocalPort, "local-port", 0, "Local port to listen on")
	portforwardCmd.Flags().StringVarP(&pfRemoteHost, "remote", "r", "", "Remote host:port to forward to")
	portforwardCmd.Flags().StringVar(&pfProtocol, "protocol", "tcp", "Protocol (tcp/udp)")
	portforwardCmd.Flags().IntVar(&pfMaxConns, "max-conns", 100, "Maximum concurrent connections")
	portforwardCmd.Flags().DurationVar(&pfIdleTimeout, "idle-timeout", 5*time.Minute, "Idle connection timeout")
	portforwardCmd.Flags().IntVar(&pfBufferSize, "buffer-size", 32*1024, "Buffer size for data transfer")
	portforwardCmd.Flags().BoolVar(&pfVerbose, "pf-verbose", false, "Verbose output")

	portforwardCmd.MarkFlagRequired("local-port")
	portforwardCmd.MarkFlagRequired("remote")
}

func runPortForward(cmd *cobra.Command, args []string) {
	if pfLocalPort == 0 || pfRemoteHost == "" {
		logger.Fatal("Both --local and --remote flags are required")
	}

	// Parse remote host and port
	remoteHost, remotePort, err := net.SplitHostPort(pfRemoteHost)
	if err != nil {
		// Try adding default port
		remoteHost = pfRemoteHost
		remotePort = fmt.Sprintf("%d", pfLocalPort)
		pfRemoteHost = net.JoinHostPort(remoteHost, remotePort)
	}

	logger.Info("Starting port forwarding: :%d -> %s", pfLocalPort, pfRemoteHost)

	// Start statistics reporter
	go reportPortForwardStats()

	switch pfProtocol {
	case "tcp":
		if err := forwardTCP(); err != nil {
			logger.Fatal("TCP forwarding failed: %v", err)
		}
	case "udp":
		if err := forwardUDP(); err != nil {
			logger.Fatal("UDP forwarding failed: %v", err)
		}
	default:
		logger.Fatal("Unknown protocol: %s", pfProtocol)
	}
}

func forwardTCP() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", pfLocalPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", pfLocalPort, err)
	}
	defer listener.Close()

	logger.Info("TCP port forwarding active on :%d", pfLocalPort)

	// Connection limiter
	connLimiter := make(chan struct{}, pfMaxConns)
	for i := 0; i < pfMaxConns; i++ {
		connLimiter <- struct{}{}
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		select {
		case <-connLimiter:
			go handleTCPConnection(conn, connLimiter)
		default:
			logger.Warn("Connection limit reached, rejecting connection from %s", conn.RemoteAddr())
			conn.Close()
		}
	}
}

func handleTCPConnection(clientConn net.Conn, limiter chan struct{}) {
	defer func() {
		clientConn.Close()
		limiter <- struct{}{} // Return token
		updateStats(func(s *ForwardStats) {
			s.ActiveConnections--
		})
	}()

	updateStats(func(s *ForwardStats) {
		s.TotalConnections++
		s.ActiveConnections++
	})

	if pfVerbose {
		logger.Debug("New connection from %s", clientConn.RemoteAddr())
	}

	// Connect to remote
	remoteConn, err := net.DialTimeout("tcp", pfRemoteHost, 10*time.Second)
	if err != nil {
		logger.Error("Failed to connect to %s: %v", pfRemoteHost, err)
		updateStats(func(s *ForwardStats) {
			s.ErrorCount++
		})
		return
	}
	defer remoteConn.Close()

	if pfVerbose {
		logger.Debug("Connected to remote %s", pfRemoteHost)
	}

	// Set idle timeout
	if pfIdleTimeout > 0 {
		clientConn.SetDeadline(time.Now().Add(pfIdleTimeout))
		remoteConn.SetDeadline(time.Now().Add(pfIdleTimeout))
	}

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Remote
	go func() {
		defer wg.Done()
		n, _ := copyBuffer(remoteConn, clientConn)
		updateStats(func(s *ForwardStats) {
			s.BytesForwarded += n
		})
	}()

	// Remote -> Client
	go func() {
		defer wg.Done()
		n, _ := copyBuffer(clientConn, remoteConn)
		updateStats(func(s *ForwardStats) {
			s.BytesForwarded += n
		})
	}()

	wg.Wait()
}

func forwardUDP() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", pfLocalPort))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %v", pfLocalPort, err)
	}
	defer conn.Close()

	logger.Info("UDP port forwarding active on :%d", pfLocalPort)

	// UDP connection map
	connections := make(map[string]*net.UDPConn)
	var connMu sync.RWMutex

	buffer := make([]byte, pfBufferSize)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			logger.Error("UDP read error: %v", err)
			continue
		}

		go handleUDPPacket(conn, clientAddr, buffer[:n], connections, &connMu)
	}
}

func handleUDPPacket(localConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte,
	connections map[string]*net.UDPConn, connMu *sync.RWMutex) {

	clientKey := clientAddr.String()

	// Get or create remote connection
	connMu.RLock()
	remoteConn, exists := connections[clientKey]
	connMu.RUnlock()

	if !exists {
		remoteAddr, err := net.ResolveUDPAddr("udp", pfRemoteHost)
		if err != nil {
			logger.Error("Failed to resolve remote address: %v", err)
			return
		}

		remoteConn, err = net.DialUDP("udp", nil, remoteAddr)
		if err != nil {
			logger.Error("Failed to connect to remote: %v", err)
			return
		}

		connMu.Lock()
		connections[clientKey] = remoteConn
		connMu.Unlock()

		// Start reverse forwarding
		go func() {
			buffer := make([]byte, pfBufferSize)
			for {
				n, err := remoteConn.Read(buffer)
				if err != nil {
					connMu.Lock()
					delete(connections, clientKey)
					connMu.Unlock()
					remoteConn.Close()
					return
				}

				localConn.WriteToUDP(buffer[:n], clientAddr)
				updateStats(func(s *ForwardStats) {
					s.BytesForwarded += int64(n)
				})
			}
		}()
	}

	// Forward packet to remote
	_, err := remoteConn.Write(data)
	if err != nil {
		logger.Error("Failed to forward UDP packet: %v", err)
		updateStats(func(s *ForwardStats) {
			s.ErrorCount++
		})
		return
	}

	updateStats(func(s *ForwardStats) {
		s.BytesForwarded += int64(len(data))
	})
}

func copyBuffer(dst net.Conn, src net.Conn) (int64, error) {
	buf := make([]byte, pfBufferSize)
	var total int64

	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, err := dst.Write(buf[:nr])
			if err != nil {
				return total, err
			}
			total += int64(nw)

			// Reset deadline on activity
			if pfIdleTimeout > 0 {
				src.SetDeadline(time.Now().Add(pfIdleTimeout))
				dst.SetDeadline(time.Now().Add(pfIdleTimeout))
			}
		}
		if err != nil {
			if err != io.EOF {
				return total, err
			}
			break
		}
	}

	return total, nil
}

func updateStats(fn func(*ForwardStats)) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	fn(stats)
}

func reportPortForwardStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats.mu.RLock()
		if stats.TotalConnections > 0 {
			color.Cyan("\n📊 Port Forwarding Statistics:")
			fmt.Printf("  Total Connections: %d\n", stats.TotalConnections)
			fmt.Printf("  Active Connections: %d\n", stats.ActiveConnections)
			fmt.Printf("  Data Forwarded: %s\n", formatBytes(stats.BytesForwarded))
			fmt.Printf("  Errors: %d\n", stats.ErrorCount)
		}
		stats.mu.RUnlock()
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
