package cmd

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/network"
	"github.com/ibrahmsql/gocat/internal/readline"
	"github.com/ibrahmsql/gocat/internal/signals"
	"github.com/ibrahmsql/gocat/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	interactive     bool
	blockSignals    bool
	localOnly       bool
	execCommand     string
	bindAddress     string
	listenKeepAlive bool
	maxConnections  int
	listenTimeout   time.Duration
	listenUseUDP    bool
	listenUseSCTP   bool
	listenForceIPv6 bool
	listenForceIPv4 bool
	listenUseSSL    bool
	sslKeyFile      string
	sslCertFile     string
	// Global flags for listen
	listenSendOnly     bool
	listenRecvOnly     bool
	listenOutputFile   string
	listenHexDumpFile  string
	listenAppendOutput bool
	listenNoShutdown   bool
	// Access control flags
	allowList []string
	denyList  []string
	allowFile string
	denyFile  string
	// Protocol flags for listen
	listenTelnetMode bool
	listenCRLFMode   bool
	listenZeroIOMode bool
)

var listenCmd = &cobra.Command{
	Use:     "listen [host] <port>",
	Aliases: []string{"l"},
	Short:   "Start a listener for incoming connections",
	Long:    `Start a TCP listener on the specified port and optionally host.`,
	Args:    cobra.RangeArgs(1, 2),
	Run:     runListen,
}

func init() {
	rootCmd.AddCommand(listenCmd)

	listenCmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive mode")
	listenCmd.Flags().BoolVar(&blockSignals, "block-signals", false, "Block exit signals like CTRL-C")
	listenCmd.Flags().BoolVar(&localOnly, "local", false, "Local interactive mode")
	listenCmd.Flags().StringVar(&execCommand, "listen-exec", "", "Execute command for each connection")
	listenCmd.Flags().StringVar(&bindAddress, "bind", "0.0.0.0", "Bind to specific address")
	listenCmd.Flags().BoolVar(&listenKeepAlive, "listen-keep-alive", false, "Keep connections alive")
	listenCmd.Flags().IntVar(&maxConnections, "listen-max-conn", 10, "Maximum concurrent connections")
	listenCmd.Flags().DurationVar(&listenTimeout, "listen-timeout", 0, "Connection timeout (0 = no timeout)")
	listenCmd.Flags().BoolVar(&listenUseUDP, "listen-udp", false, "Use UDP instead of TCP")
	listenCmd.Flags().BoolVar(&listenForceIPv6, "listen-ipv6", false, "Force IPv6")
	listenCmd.Flags().BoolVar(&listenForceIPv4, "listen-ipv4", false, "Force IPv4")
	listenCmd.Flags().BoolVar(&listenUseSSL, "listen-ssl", false, "Use SSL/TLS")
	listenCmd.Flags().StringVar(&sslKeyFile, "listen-ssl-key", "", "SSL private key file")
	listenCmd.Flags().StringVar(&sslCertFile, "listen-ssl-cert", "", "SSL certificate file")

	// Mark conflicting flags
	listenCmd.MarkFlagsMutuallyExclusive("interactive", "local")
	listenCmd.MarkFlagsMutuallyExclusive("listen-ipv4", "listen-ipv6")
}

func runListen(cmd *cobra.Command, args []string) {
	var host, port string

	if len(args) == 1 {
		host = bindAddress
		port = args[0]
	} else {
		host = args[0]
		port = args[1]
	}

	// Override local flags with global flags if set
	if globalSSL, _ := cmd.Root().PersistentFlags().GetBool("ssl"); globalSSL {
		listenUseSSL = true
	}
	if globalUDP, _ := cmd.Root().PersistentFlags().GetBool("udp"); globalUDP {
		listenUseUDP = true
	}
	if globalIPv4, _ := cmd.Root().PersistentFlags().GetBool("ipv4"); globalIPv4 {
		listenForceIPv4 = true
	}
	if globalIPv6, _ := cmd.Root().PersistentFlags().GetBool("ipv6"); globalIPv6 {
		listenForceIPv6 = true
	}
	if globalSCTP, _ := cmd.Root().PersistentFlags().GetBool("sctp"); globalSCTP {
		listenUseSCTP = true
	}
	if globalMaxConns, _ := cmd.Root().PersistentFlags().GetInt("max-conns"); globalMaxConns > 0 {
		maxConnections = globalMaxConns
	}
	if globalSSLCert, _ := cmd.Root().PersistentFlags().GetString("ssl-cert"); globalSSLCert != "" {
		sslCertFile = globalSSLCert
	}
	if globalSSLKey, _ := cmd.Root().PersistentFlags().GetString("ssl-key"); globalSSLKey != "" {
		sslKeyFile = globalSSLKey
	}
	// Data flow control flags for listen
	if globalSendOnly, _ := cmd.Root().PersistentFlags().GetBool("send-only"); globalSendOnly {
		listenSendOnly = true
	}
	if globalRecvOnly, _ := cmd.Root().PersistentFlags().GetBool("recv-only"); globalRecvOnly {
		listenRecvOnly = true
	}
	if globalNoShutdown, _ := cmd.Root().PersistentFlags().GetBool("no-shutdown"); globalNoShutdown {
		listenNoShutdown = true
	}
	// Output flags for listen
	if globalOutput, _ := cmd.Root().PersistentFlags().GetString("output"); globalOutput != "" {
		listenOutputFile = globalOutput
	}
	if globalHexDump, _ := cmd.Root().PersistentFlags().GetString("hex-dump"); globalHexDump != "" {
		listenHexDumpFile = globalHexDump
	}
	if globalAppend, _ := cmd.Root().PersistentFlags().GetBool("append-output"); globalAppend {
		listenAppendOutput = true
	}
	// Execution flags for listen
	if globalExec, _ := cmd.Root().PersistentFlags().GetString("exec"); globalExec != "" {
		execCommand = globalExec
	}
	if globalShExec, _ := cmd.Root().PersistentFlags().GetString("sh-exec"); globalShExec != "" {
		execCommand = "/bin/sh -c \"" + globalShExec + "\""
	}
	// Access control flags
	if globalAllow, _ := cmd.Root().PersistentFlags().GetStringSlice("allow"); len(globalAllow) > 0 {
		allowList = globalAllow
	}
	if globalDeny, _ := cmd.Root().PersistentFlags().GetStringSlice("deny"); len(globalDeny) > 0 {
		denyList = globalDeny
	}
	if globalAllowFile, _ := cmd.Root().PersistentFlags().GetString("allowfile"); globalAllowFile != "" {
		allowFile = globalAllowFile
	}
	if globalDenyFile, _ := cmd.Root().PersistentFlags().GetString("denyfile"); globalDenyFile != "" {
		denyFile = globalDenyFile
	}
	// Protocol flags for listen
	if globalTelnet, _ := cmd.Root().PersistentFlags().GetBool("telnet"); globalTelnet {
		listenTelnetMode = true
	}
	if globalCRLF, _ := cmd.Root().PersistentFlags().GetBool("crlf"); globalCRLF {
		listenCRLFMode = true
	}
	if globalZeroIO, _ := cmd.Root().PersistentFlags().GetBool("zero-io"); globalZeroIO {
		listenZeroIOMode = true
	}

	if err := listen(host, port); err != nil {
		logger.Fatal("Error: %v", err)
	}
}

func listen(host, port string) error {
	address := net.JoinHostPort(host, port)

	// Determine network type
	network := "tcp"
	if listenUseUDP {
		network = "udp"
	} else if listenUseSCTP {
		network = "sctp"
	}
	if listenForceIPv6 {
		network += "6"
	} else if listenForceIPv4 {
		network += "4"
	}

	logger.Debug("Listening on %s using %s protocol", address, network)
	if listenUseSSL {
		logger.Debug("SSL/TLS enabled for listening")
	}
	logger.Debug("Maximum connections: %d", maxConnections)

	var listener net.Listener
	var err error

	// Handle SCTP separately
	if listenUseSCTP {
		return handleSCTPListener(network, address)
	}

	// Handle SSL/TLS
	if listenUseSSL {
		listener, err = createTLSListener(network, address)
	} else if listenUseUDP {
		return handleUDPListener(network, address)
	} else {
		listener, err = net.Listen(network, address)
	}

	if err != nil {
		return fmt.Errorf("failed to bind to %s: %v", address, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			logger.Error("Error closing listener: %v", err)
		}
	}()

	theme := logger.GetCurrentTheme()
	if _, err := theme.Success.Printf("Listening on %s\n", address); err != nil {
		logger.Error("Error printing success message: %v", err)
	}

	// Handle multiple connections with semaphore
	connSemaphore := make(chan struct{}, maxConnections)
	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection: %v", err)
			continue
		}

		// Acquire semaphore slot
		connSemaphore <- struct{}{}
		wg.Add(1)

		go func(c net.Conn) {
			defer func() {
				if err := c.Close(); err != nil {
					logger.Error("Error closing connection: %v", err)
				}
				<-connSemaphore // Release semaphore slot
				wg.Done()
			}()

			handleConnection(c)
		}(conn)
	}
}

func createTLSListener(network, address string) (net.Listener, error) {
	if sslCertFile == "" || sslKeyFile == "" {
		return nil, fmt.Errorf("SSL certificate and key files are required for SSL mode")
	}

	cert, err := tls.LoadX509KeyPair(sslCertFile, sslKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSL certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12, // Secure minimum TLS version
	}

	return tls.Listen(network, address, tlsConfig)
}

func handleUDPListener(network, address string) error {
	udpAddr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	udpConn, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP: %v", err)
	}
	defer func() {
		if err := udpConn.Close(); err != nil {
			logger.Error("Error closing UDP connection: %v", err)
		}
	}()

	theme := logger.GetCurrentTheme()
	if _, err := theme.Success.Printf("Listening on %s (UDP)\n", address); err != nil {
		logger.Error("Error printing success message: %v", err)
	}

	buffer := make([]byte, 4096)
	for {
		n, clientAddr, err := udpConn.ReadFromUDP(buffer)
		if err != nil {
			logger.Error("UDP read error: %v", err)
			continue
		}

		theme := logger.GetCurrentTheme()
		if _, err := theme.Highlight.Printf("UDP packet from %s: %s\n", clientAddr, string(buffer[:n])); err != nil {
			logger.Error("Error printing highlight message: %v", err)
		}

		// Echo back for UDP
		if _, err := udpConn.WriteToUDP(buffer[:n], clientAddr); err != nil {
			logger.Error("UDP write error: %v", err)
		}
	}
}

func handleSCTPListener(netType, address string) error {
	// Check if SCTP is supported
	if !network.IsSCTPSupported() {
		return fmt.Errorf("SCTP protocol not supported on this platform")
	}

	// Parse SCTP address
	sctpAddr, err := network.ResolveSCTPAddr(netType, address)
	if err != nil {
		return fmt.Errorf("failed to resolve SCTP address: %w", err)
	}

	// Create SCTP listener
	listener, err := network.ListenSCTP(netType, sctpAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to bind SCTP: %w", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			logger.Error("Error closing SCTP listener: %v", err)
		}
	}()

	theme := logger.GetCurrentTheme()
	if _, err := theme.Success.Printf("Listening on %s (SCTP)\n", address); err != nil {
		logger.Error("Error printing success message: %v", err)
	}

	// Handle multiple connections with semaphore
	connSemaphore := make(chan struct{}, maxConnections)
	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept SCTP connection: %v", err)
			continue
		}

		// Acquire semaphore slot
		connSemaphore <- struct{}{}
		wg.Add(1)

		go func(c net.Conn) {
			defer func() {
				if err := c.Close(); err != nil {
					logger.Error("Error closing SCTP connection: %v", err)
				}
				<-connSemaphore // Release semaphore slot
				wg.Done()
			}()

			handleConnection(c)
		}(conn)
	}
}

func handleConnection(conn net.Conn) {
	// Set connection timeout if specified
	if listenTimeout > 0 {
		if err := conn.SetDeadline(time.Now().Add(listenTimeout)); err != nil {
			logger.Error("Error setting deadline: %v", err)
		}
	}

	// Configure keep-alive for TCP connections
	if listenKeepAlive && !listenUseUDP {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			if err := tcpConn.SetKeepAlive(true); err != nil {
				logger.Warn("Failed to enable keep-alive: %v", err)
			} else {
				if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
					logger.Warn("Failed to set keep-alive period: %v", err)
				}
			}
		}
	}

	theme := logger.GetCurrentTheme()
	if _, err := theme.Highlight.Printf("Connection received from %s\n", conn.RemoteAddr()); err != nil {
		logger.Error("Error printing highlight message: %v", err)
	}

	var err error
	if interactive {
		err = handleInteractive(conn)
	} else if localOnly {
		err = handleLocalInteractive(conn)
	} else {
		err = handleNormal(conn)
	}

	if err != nil {
		logger.Error("Connection handling error: %v", err)
	}
}

func handleNormal(conn net.Conn) error {
	if blockSignals {
		signals.BlockExitSignals()
	}

	if execCommand != "" {
		if _, err := conn.Write([]byte(execCommand + "\n")); err != nil {
			return fmt.Errorf("failed to send exec command: %v", err)
		}
	}

	// Start goroutines for bidirectional communication
	go func() {
		if _, err := io.Copy(conn, os.Stdin); err != nil {
			logger.Error("stdin copy error: %v", err)
		}
		os.Exit(0)
	}()

	if _, err := io.Copy(os.Stdout, conn); err != nil {
		return fmt.Errorf("stdout copy error: %v", err)
	}

	return nil
}

func handleInteractive(conn net.Conn) error {
	if blockSignals {
		signals.BlockExitSignals()
	}

	// Setup terminal for interactive mode on Unix systems
	if runtime.GOOS != "windows" {
		if termState, err := terminal.SetupTerminal(); err == nil {
			defer func() {
				if err := termState.Restore(); err != nil {
					logger.Error("Error restoring terminal state: %v", err)
				}
			}()
		}
	}

	// Create a PTY for better shell interaction
	shell := "/bin/sh"
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
	} else if os.Getenv("SHELL") != "" {
		shell = os.Getenv("SHELL")
	}

	cmd := exec.Command(shell, "-i")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start pty: %v", err)
	}
	defer func() {
		if err := ptmx.Close(); err != nil {
			logger.Error("Error closing pty: %v", err)
		}
	}()

	// Handle PTY size changes
	go func() {
		for {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				logger.Error("error resizing pty: %v", err)
			}
		}
	}()

	// Copy data between connection and PTY
	go func() {
		if _, err := io.Copy(ptmx, conn); err != nil {
			logger.Error("conn to pty copy error: %v", err)
		}
	}()

	if _, err := io.Copy(conn, ptmx); err != nil {
		return fmt.Errorf("pty to conn copy error: %v", err)
	}

	return nil
}

func handleLocalInteractive(conn net.Conn) error {
	// Start goroutine to read from connection and write to stdout
	go func() {
		if _, err := io.Copy(os.Stdout, conn); err != nil {
			logger.Error("connection read error: %v", err)
		}
	}()

	theme := logger.GetCurrentTheme()
	if _, err := theme.Highlight.Printf("Connection received\n"); err != nil {
		logger.Error("Error printing highlight message: %v", err)
	}

	// Create readline editor
	editor := readline.NewEditor()
	editor.SetPrompt(">> ")

	// Readline loop
	for {
		command, err := editor.Readline()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read command: %v", err)
		}

		if strings.TrimSpace(command) == "exit" {
			break
		}

		if _, err := conn.Write([]byte(command + "\n")); err != nil {
			return fmt.Errorf("failed to send command: %v", err)
		}
	}

	return nil
}
