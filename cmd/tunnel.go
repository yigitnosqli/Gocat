package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	tunnelSSH         string
	tunnelLocal       string
	tunnelRemote      string
	tunnelReverse     bool
	tunnelDynamic     bool
	tunnelKeyFile     string
	tunnelPassword    string
	tunnelUser        string
	tunnelCompression bool
)

var tunnelCmd = &cobra.Command{
	Use:     "tunnel",
	Aliases: []string{"tun", "ssh-tunnel"},
	Short:   "Create SSH tunnels (local, remote, dynamic)",
	Long: `Create SSH tunnels for port forwarding through SSH connections.
Supports local forwarding, remote forwarding, and dynamic SOCKS proxy.

Examples:
  # Local port forwarding (access remote service locally)
  gocat tunnel --ssh user@server --local 8080 --remote localhost:80

  # Remote port forwarding (expose local service remotely)
  gocat tunnel --ssh user@server --reverse --local 3000 --remote 8080

  # Dynamic SOCKS proxy
  gocat tunnel --ssh user@server --dynamic 1080

  # With SSH key authentication
  gocat tunnel --ssh user@server --key ~/.ssh/id_rsa --local 8080 --remote 80
`,
	Run: runTunnel,
}

// init registers the tunnel subcommand and configures its command-line flags.
//
// It adds tunnelCmd to the root command, defines flags for SSH connection,
// local/remote addresses, mode toggles (reverse, dynamic), authentication
// options (key, password, user), and compression, and marks the "ssh" flag
// as required.
func init() {
	rootCmd.AddCommand(tunnelCmd)

	tunnelCmd.Flags().StringVar(&tunnelSSH, "ssh", "", "SSH server (user@host:port)")
	tunnelCmd.Flags().StringVar(&tunnelLocal, "local", "", "Local address:port")
	tunnelCmd.Flags().StringVar(&tunnelRemote, "remote", "", "Remote address:port")
	tunnelCmd.Flags().BoolVar(&tunnelReverse, "reverse", false, "Reverse tunnel (remote to local)")
	tunnelCmd.Flags().BoolVar(&tunnelDynamic, "dynamic", false, "Dynamic SOCKS proxy")
	tunnelCmd.Flags().StringVar(&tunnelKeyFile, "key", "", "SSH private key file")
	tunnelCmd.Flags().StringVar(&tunnelPassword, "password", "", "SSH password")
	tunnelCmd.Flags().StringVar(&tunnelUser, "user", "", "SSH username (overrides user@host)")
	tunnelCmd.Flags().BoolVar(&tunnelCompression, "compression", false, "Enable SSH compression")

	tunnelCmd.MarkFlagRequired("ssh")
}

// runTunnel establishes an SSH connection based on global flags and starts the selected tunnel mode.
// It parses the SSH target, constructs client authentication (key and/or password), connects to the SSH server,
// and dispatches to runLocalTunnel, runReverseTunnel, or runDynamicTunnel according to flags.
// The function logs a fatal error and exits if required flags are missing, authentication is not configured, or the SSH connection cannot be established.
func runTunnel(cmd *cobra.Command, args []string) {
	// Parse SSH connection string
	user, host, port := parseSSHConnection(tunnelSSH)
	if tunnelUser != "" {
		user = tunnelUser
	}

	logger.Info("Connecting to SSH server: %s@%s:%s", user, host, port)

	// Create SSH client config
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: createHostKeyCallback(),
		Timeout:         0,
	}

	// Add authentication methods
	if tunnelKeyFile != "" {
		key, err := os.ReadFile(tunnelKeyFile)
		if err != nil {
			logger.Fatal("Failed to read SSH key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			logger.Fatal("Failed to parse SSH key: %v", err)
		}

		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
		logger.Debug("Using SSH key authentication")
	}

	if tunnelPassword != "" {
		config.Auth = append(config.Auth, ssh.Password(tunnelPassword))
		logger.Debug("Using password authentication")
	}

	if len(config.Auth) == 0 {
		logger.Fatal("No authentication method specified (use --key or --password)")
	}

	// Connect to SSH server
	sshAddr := net.JoinHostPort(host, port)
	client, err := ssh.Dial("tcp", sshAddr, config)
	if err != nil {
		logger.Fatal("Failed to connect to SSH server: %v", err)
	}
	defer client.Close()

	logger.Info("SSH connection established")

	// Create tunnel based on mode
	if tunnelDynamic {
		// Dynamic SOCKS proxy
		if tunnelLocal == "" {
			logger.Fatal("--local required for dynamic SOCKS proxy")
		}
		runDynamicTunnel(client, tunnelLocal)
	} else if tunnelReverse {
		// Remote port forwarding
		if tunnelLocal == "" || tunnelRemote == "" {
			logger.Fatal("Both --local and --remote required for reverse tunnel")
		}
		runReverseTunnel(client, tunnelLocal, tunnelRemote)
	} else {
		// Local port forwarding
		if tunnelLocal == "" || tunnelRemote == "" {
			logger.Fatal("Both --local and --remote required for local tunnel")
		}
		runLocalTunnel(client, tunnelLocal, tunnelRemote)
	}
}

// cannot be loaded, it returns a callback that accepts any host key (insecure).
func createHostKeyCallback() ssh.HostKeyCallback {
	// Try to load known_hosts file
	knownHostsPath := os.Getenv("HOME") + "/.ssh/known_hosts"
	if _, err := os.Stat(knownHostsPath); err == nil {
		hostKeyCallback, err := knownhosts.New(knownHostsPath)
		if err == nil {
			logger.Debug("Using known_hosts file for host key verification")
			return hostKeyCallback
		}
		logger.Warn("Failed to load known_hosts: %v", err)
	}

	// Fallback to insecure (with warning)
	logger.Warn("⚠️  Host key verification disabled - connection may be insecure!")
	logger.Warn("⚠️  Consider using known_hosts file at: %s", knownHostsPath)
	return ssh.InsecureIgnoreHostKey()
}

// parseSSHConnection parses an SSH connection string of the form "user@host:port".
// If the user is omitted, the current OS user from $USER is used; if that is empty, "root" is used.
// If the port is omitted, "22" is used.
// It returns the parsed user, host, and port.
func parseSSHConnection(conn string) (user, host, port string) {
	// Parse user@host:port
	parts := strings.Split(conn, "@")
	if len(parts) == 2 {
		user = parts[0]
		conn = parts[1]
	} else {
		user = os.Getenv("USER")
		if user == "" {
			user = "root"
		}
	}

	// Parse host:port
	hostPort := strings.Split(conn, ":")
	host = hostPort[0]
	if len(hostPort) == 2 {
		port = hostPort[1]
	} else {
		port = "22"
	}

	return user, host, port
}

// runLocalTunnel starts a local TCP listener on localAddr and forwards each incoming connection to remoteAddr through the provided SSH client.
// It accepts connections in a loop and handles each connection concurrently; the listener is closed when the function returns.
func runLocalTunnel(client *ssh.Client, localAddr, remoteAddr string) {
	logger.Info("Starting local tunnel: %s -> %s", localAddr, remoteAddr)

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logger.Fatal("Failed to listen on %s: %v", localAddr, err)
	}
	defer listener.Close()

	logger.Info("Local tunnel listening on %s", localAddr)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		go handleLocalTunnelConnection(client, localConn, remoteAddr)
	}
}

// handleLocalTunnelConnection forwards data between an accepted local connection and a remote address over the provided SSH client.
// It dials the remote address through the SSH connection, performs bidirectional copying of data until one side closes, and ensures both connections are closed when finished.
func handleLocalTunnelConnection(client *ssh.Client, localConn net.Conn, remoteAddr string) {
	defer localConn.Close()

	// Connect to remote address through SSH
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		logger.Error("Failed to connect to remote %s: %v", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	logger.Debug("Tunnel established: %s <-> %s", localConn.RemoteAddr(), remoteAddr)

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	<-done
}

// runReverseTunnel starts a reverse SSH tunnel by asking the SSH server to listen on remoteAddr
// and forwarding each incoming remote connection to localAddr on the client side.
// remoteAddr and localAddr are network addresses (for example "host:port" or ":port").
// The function accepts connections in a loop and forwards them concurrently; it logs a fatal error
// if it fails to establish the remote listener.
func runReverseTunnel(client *ssh.Client, localAddr, remoteAddr string) {
	logger.Info("Starting reverse tunnel: %s <- %s", remoteAddr, localAddr)

	// Listen on remote server
	listener, err := client.Listen("tcp", remoteAddr)
	if err != nil {
		logger.Fatal("Failed to listen on remote %s: %v", remoteAddr, err)
	}
	defer listener.Close()

	logger.Info("Reverse tunnel listening on remote %s", remoteAddr)

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		go handleReverseTunnelConnection(remoteConn, localAddr)
	}
}

// handleReverseTunnelConnection establishes a TCP connection to localAddr and proxies data
// bidirectionally between the provided remoteConn and the newly created local connection
// until one side closes. Both connections are closed when the function returns; if dialing
// localAddr fails the function logs the error and returns.
func handleReverseTunnelConnection(remoteConn net.Conn, localAddr string) {
	defer remoteConn.Close()

	// Connect to local address
	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		logger.Error("Failed to connect to local %s: %v", localAddr, err)
		return
	}
	defer localConn.Close()

	logger.Debug("Reverse tunnel established: %s <-> %s", remoteConn.RemoteAddr(), localAddr)

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	<-done
}

// runDynamicTunnel starts a SOCKS5 proxy bound to localAddr that forwards proxied
// connections through the provided SSH client. It listens for incoming TCP
// connections on localAddr and handles each accepted connection concurrently.
func runDynamicTunnel(client *ssh.Client, localAddr string) {
	logger.Info("Starting dynamic SOCKS proxy on %s", localAddr)

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logger.Fatal("Failed to listen on %s: %v", localAddr, err)
	}
	defer listener.Close()

	logger.Info("SOCKS proxy listening on %s", localAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		go handleSOCKSConnection(client, conn)
	}
}

// handleSOCKSConnection handles a single SOCKS5 client connection on conn, negotiates the SOCKS5 handshake,
// resolves the target address (IPv4 or domain name), establishes a TCP connection to that target through
// the provided SSH client, and proxies data bidirectionally until one side closes.
//
// It supports IPv4 and domain-name address types; IPv6 is not supported. On failure to establish the
// remote connection a SOCKS failure response is sent. The client connection is closed when this function returns.
func handleSOCKSConnection(client *ssh.Client, conn net.Conn) {
	defer conn.Close()

	// Read SOCKS5 handshake
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		logger.Error("SOCKS handshake error: %v", err)
		return
	}

	// Check SOCKS version
	if n < 2 || buf[0] != 0x05 {
		logger.Error("Unsupported SOCKS version: %d", buf[0])
		return
	}

	// Send auth method (no auth)
	conn.Write([]byte{0x05, 0x00})

	// Read connection request
	n, err = conn.Read(buf)
	if err != nil {
		logger.Error("SOCKS request error: %v", err)
		return
	}

	if n < 7 || buf[0] != 0x05 {
		logger.Error("Invalid SOCKS request")
		return
	}

	// Parse target address
	var targetAddr string
	addrType := buf[3]

	switch addrType {
	case 0x01: // IPv4
		if n < 10 {
			logger.Error("Invalid IPv4 address")
			return
		}
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d",
			buf[4], buf[5], buf[6], buf[7],
			int(buf[8])<<8|int(buf[9]))

	case 0x03: // Domain name
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			logger.Error("Invalid domain name")
			return
		}
		domain := string(buf[5 : 5+domainLen])
		port := int(buf[5+domainLen])<<8 | int(buf[6+domainLen])
		targetAddr = fmt.Sprintf("%s:%d", domain, port)

	case 0x04: // IPv6
		logger.Error("IPv6 not yet supported in SOCKS proxy")
		return

	default:
		logger.Error("Unsupported address type: %d", addrType)
		return
	}

	logger.Debug("SOCKS connection to %s", targetAddr)

	// Connect through SSH
	remoteConn, err := client.Dial("tcp", targetAddr)
	if err != nil {
		logger.Error("Failed to connect to %s: %v", targetAddr, err)
		// Send failure response
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remoteConn.Close()

	// Send success response
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, conn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(conn, remoteConn)
		done <- struct{}{}
	}()

	<-done
}