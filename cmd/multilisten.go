package cmd

import (
	"io"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	multiPorts       []string
	multiPortRange   string
	multiExec        string
	multiMaxConns    int
	multiTimeout     time.Duration
	multiShowStats   bool
	multiBindAddress string
)

type multiListenStats struct {
	portStats map[int]*portStats
	mu        sync.RWMutex
}

type portStats struct {
	Port            int
	TotalConns      int64
	ActiveConns     int64
	BytesReceived   int64
	BytesSent       int64
	LastConnection  time.Time
}

var mlStats = &multiListenStats{
	portStats: make(map[int]*portStats),
}

var multiListenCmd = &cobra.Command{
	Use:     "multi-listen",
	Aliases: []string{"ml", "multi"},
	Short:   "Listen on multiple ports simultaneously",
	Long: `Listen on multiple ports at the same time and handle connections.
Supports port ranges, individual ports, and different handlers per port.

Examples:
  # Listen on multiple ports
  gocat multi-listen --ports 8080,8081,8082

  # Listen on a port range
  gocat multi-listen --range 8000-8100

  # Execute command for each connection
  gocat multi-listen --ports 8080,8081 --exec /bin/bash

  # With connection limits
  gocat multi-listen --range 8000-8010 --max-connections 1000
`,
	Run: runMultiListen,
}

func init() {
	rootCmd.AddCommand(multiListenCmd)

	multiListenCmd.Flags().StringSliceVar(&multiPorts, "ports", nil, "Comma-separated list of ports (e.g., 8080,8081,8082)")
	multiListenCmd.Flags().StringVar(&multiPortRange, "range", "", "Port range (e.g., 8000-8100)")
	multiListenCmd.Flags().StringVar(&multiExec, "exec", "", "Command to execute for each connection")
	multiListenCmd.Flags().IntVar(&multiMaxConns, "max-connections", 1000, "Maximum concurrent connections per port")
	multiListenCmd.Flags().DurationVar(&multiTimeout, "timeout", 0, "Connection timeout (0 = no timeout)")
	multiListenCmd.Flags().BoolVar(&multiShowStats, "stats", true, "Show statistics")
	multiListenCmd.Flags().StringVar(&multiBindAddress, "bind", "0.0.0.0", "Bind address")
}

func runMultiListen(cmd *cobra.Command, args []string) {
	// Parse ports
	var ports []int
	
	// Add individual ports
	for _, portStr := range multiPorts {
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil {
			logger.Fatal("Invalid port: %s", portStr)
		}
		ports = append(ports, port)
	}

	// Add port range
	if multiPortRange != "" {
		rangePorts, err := parsePortRange(multiPortRange)
		if err != nil {
			logger.Fatal("Invalid port range: %v", err)
		}
		ports = append(ports, rangePorts...)
	}

	if len(ports) == 0 {
		logger.Fatal("No ports specified. Use --ports or --range")
	}

	logger.Info("Starting multi-port listener on %d ports", len(ports))

	// Start stats reporter if enabled
	if multiShowStats {
		go reportMultiListenStats()
	}

	// Start listeners
	var wg sync.WaitGroup
	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			startPortListener(p)
		}(port)
	}

	logger.Info("All listeners started. Press Ctrl+C to stop.")
	wg.Wait()
}

func startPortListener(port int) {
	address := net.JoinHostPort(multiBindAddress, strconv.Itoa(port))
	
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("Failed to listen on port %d: %v", port, err)
		return
	}
	defer listener.Close()

	// Initialize stats
	mlStats.mu.Lock()
	mlStats.portStats[port] = &portStats{
		Port: port,
	}
	mlStats.mu.Unlock()

	logger.Info("Listening on %s", address)

	// Connection semaphore
	connSemaphore := make(chan struct{}, multiMaxConns)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error on port %d: %v", port, err)
			continue
		}

		// Acquire semaphore
		connSemaphore <- struct{}{}

		go func(c net.Conn, p int) {
			defer func() {
				c.Close()
				<-connSemaphore
			}()
			handleMultiListenConnection(c, p)
		}(conn, port)
	}
}

func handleMultiListenConnection(conn net.Conn, port int) {
	// Update stats
	mlStats.mu.Lock()
	if stats, ok := mlStats.portStats[port]; ok {
		stats.TotalConns++
		stats.ActiveConns++
		stats.LastConnection = time.Now()
	}
	mlStats.mu.Unlock()

	defer func() {
		mlStats.mu.Lock()
		if stats, ok := mlStats.portStats[port]; ok {
			stats.ActiveConns--
		}
		mlStats.mu.Unlock()
	}()

	// Set timeout if specified
	if multiTimeout > 0 {
		conn.SetDeadline(time.Now().Add(multiTimeout))
	}

	logger.Debug("Connection on port %d from %s", port, conn.RemoteAddr())

	if multiExec != "" {
		// Execute command
		handleExecConnection(conn, port)
	} else {
		// Echo mode
		handleEchoConnection(conn, port)
	}
}

func handleExecConnection(conn net.Conn, port int) {
	shell := multiExec
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd.exe"
		} else {
			shell = "/bin/sh"
		}
	}

	cmd := exec.Command(shell)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Error("Failed to create stdin pipe: %v", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to create stdout pipe: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("Failed to create stderr pipe: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Error("Failed to start command: %v", err)
		return
	}

	// Copy data
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		io.Copy(stdin, conn)
		stdin.Close()
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(conn, stdout)
		mlStats.mu.Lock()
		if stats, ok := mlStats.portStats[port]; ok {
			stats.BytesSent += n
		}
		mlStats.mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		n, _ := io.Copy(conn, stderr)
		mlStats.mu.Lock()
		if stats, ok := mlStats.portStats[port]; ok {
			stats.BytesSent += n
		}
		mlStats.mu.Unlock()
	}()

	wg.Wait()
	cmd.Wait()
}

func handleEchoConnection(conn net.Conn, port int) {
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Debug("Read error on port %d: %v", port, err)
			}
			break
		}

		mlStats.mu.Lock()
		if stats, ok := mlStats.portStats[port]; ok {
			stats.BytesReceived += int64(n)
		}
		mlStats.mu.Unlock()

		// Echo back
		written, err := conn.Write(buf[:n])
		if err != nil {
			logger.Debug("Write error on port %d: %v", port, err)
			break
		}

		mlStats.mu.Lock()
		if stats, ok := mlStats.portStats[port]; ok {
			stats.BytesSent += int64(written)
		}
		mlStats.mu.Unlock()
	}
}

func reportMultiListenStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mlStats.mu.RLock()
		theme := logger.GetCurrentTheme()
		theme.Info.Println("\n╔═══════════════════════════════════════════════════════════════╗")
		theme.Info.Println("║          Multi-Port Listener Statistics                    ║")
		theme.Info.Println("╠═══════════════════════════════════════════════════════════════╣")
		
		for port, stats := range mlStats.portStats {
			theme.Success.Printf("║ Port %-5d │ ", port)
			theme.Highlight.Printf("Total: %-6d │ ", stats.TotalConns)
			theme.Info.Printf("Active: %-4d │ ", stats.ActiveConns)
			theme.Debug.Printf("RX: %-8d │ TX: %-8d ║\n", stats.BytesReceived, stats.BytesSent)
			if !stats.LastConnection.IsZero() {
				theme.Debug.Printf("║           └─ Last connection: %s                          ║\n",
					stats.LastConnection.Format("2006-01-02 15:04:05"))
			}
		}
		
		theme.Info.Println("╚═══════════════════════════════════════════════════════════════╝")
		mlStats.mu.RUnlock()
	}
}
