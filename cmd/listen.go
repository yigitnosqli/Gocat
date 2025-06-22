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
	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/readline"
	"github.com/ibrahmsql/gocat/internal/signals"
	"github.com/ibrahmsql/gocat/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	interactive      bool
	blockSignals     bool
	localInteractive bool
	execCmd          string
	bindAddress      string
	listenKeepAlive  bool
	maxConnections   int
	connTimeout      time.Duration
	listenUseUDP     bool
	listenForceIPv6  bool
	listenForceIPv4  bool
	listenUseSSL     bool
	sslKeyFile       string
	sslCertFile      string
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

	listenCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")
	listenCmd.Flags().BoolVarP(&blockSignals, "block-signals", "b", false, "Block exit signals like CTRL-C")
	listenCmd.Flags().BoolVarP(&localInteractive, "local", "l", false, "Local interactive mode")
	listenCmd.Flags().StringVarP(&execCmd, "exec", "e", "", "Execute command for each connection")
	listenCmd.Flags().StringVar(&bindAddress, "bind", "0.0.0.0", "Bind to specific address")
	listenCmd.Flags().BoolVarP(&listenKeepAlive, "keep-alive", "k", false, "Keep connections alive")
	listenCmd.Flags().IntVarP(&maxConnections, "max-conn", "m", 10, "Maximum concurrent connections")
	listenCmd.Flags().DurationVarP(&connTimeout, "timeout", "t", 0, "Connection timeout (0 = no timeout)")
	listenCmd.Flags().BoolVarP(&listenUseUDP, "udp", "u", false, "Use UDP instead of TCP")
	listenCmd.Flags().BoolVarP(&listenForceIPv6, "ipv6", "6", false, "Force IPv6")
	listenCmd.Flags().BoolVarP(&listenForceIPv4, "ipv4", "4", false, "Force IPv4")
	listenCmd.Flags().BoolVarP(&listenUseSSL, "ssl", "S", false, "Use SSL/TLS")
	listenCmd.Flags().StringVarP(&sslKeyFile, "ssl-key", "K", "", "SSL private key file")
	listenCmd.Flags().StringVarP(&sslCertFile, "ssl-cert", "C", "", "SSL certificate file")

	// Mark conflicting flags
	listenCmd.MarkFlagsMutuallyExclusive("interactive", "local")
	listenCmd.MarkFlagsMutuallyExclusive("ipv4", "ipv6")
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
	}
	if listenForceIPv6 {
		network += "6"
	} else if listenForceIPv4 {
		network += "4"
	}

	var listener net.Listener
	var err error

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
	defer listener.Close()

	color.Green("Listening on %s", address)

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
				c.Close()
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
	defer udpConn.Close()

	color.Green("Listening on %s (UDP)", address)

	buffer := make([]byte, 4096)
	for {
		n, clientAddr, err := udpConn.ReadFromUDP(buffer)
		if err != nil {
			logger.Error("UDP read error: %v", err)
			continue
		}

		color.Cyan("UDP packet from %s: %s", clientAddr, string(buffer[:n]))

		// Echo back for UDP
		if _, err := udpConn.WriteToUDP(buffer[:n], clientAddr); err != nil {
			logger.Error("UDP write error: %v", err)
		}
	}
}

func handleConnection(conn net.Conn) {
	// Set connection timeout if specified
	if connTimeout > 0 {
		conn.SetDeadline(time.Now().Add(connTimeout))
	}

	// Configure keep-alive for TCP connections
	if listenKeepAlive && !listenUseUDP {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}
	}

	color.Cyan("Connection received from %s", conn.RemoteAddr())

	var err error
	if interactive {
		err = handleInteractive(conn)
	} else if localInteractive {
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

	if execCmd != "" {
		if _, err := conn.Write([]byte(execCmd + "\n")); err != nil {
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
			defer termState.Restore()
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
	defer ptmx.Close()

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

	color.Cyan("Connection received")

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
