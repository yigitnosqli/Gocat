package relay

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// RelayMode defines different relay operation modes
type RelayMode int

const (
	// ModeBidirectional relays data in both directions
	ModeBidirectional RelayMode = iota
	// ModeForward only forwards data from source to destination
	ModeForward
	// ModeReverse only forwards data from destination to source
	ModeReverse
)

// Relay handles connection relaying/proxying
type Relay struct {
	Mode          RelayMode
	BufferSize    int
	Timeout       time.Duration
	ShowStats     bool
	bytesForward  int64
	bytesReverse  int64
	mutex         sync.RWMutex
}

// NewRelay creates a new relay instance
func NewRelay() *Relay {
	return &Relay{
		Mode:       ModeBidirectional,
		BufferSize: 32768, // 32KB default
		Timeout:    0,     // No timeout by default
		ShowStats:  true,
	}
}

// RelayConnections relays data between two connections
func (r *Relay) RelayConnections(conn1, conn2 net.Conn) error {
	logger.Info("Starting relay between %s and %s", conn1.RemoteAddr(), conn2.RemoteAddr())
	
	var wg sync.WaitGroup
	errChan := make(chan error, 2)
	
	// Set timeouts if specified
	if r.Timeout > 0 {
		deadline := time.Now().Add(r.Timeout)
		conn1.SetDeadline(deadline)
		conn2.SetDeadline(deadline)
	}
	
	// Forward direction: conn1 -> conn2
	if r.Mode == ModeBidirectional || r.Mode == ModeForward {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bytes, err := r.copyData(conn2, conn1, "forward")
			r.mutex.Lock()
			r.bytesForward += bytes
			r.mutex.Unlock()
			if err != nil {
				errChan <- fmt.Errorf("forward relay error: %v", err)
			}
		}()
	}
	
	// Reverse direction: conn2 -> conn1
	if r.Mode == ModeBidirectional || r.Mode == ModeReverse {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bytes, err := r.copyData(conn1, conn2, "reverse")
			r.mutex.Lock()
			r.bytesReverse += bytes
			r.mutex.Unlock()
			if err != nil {
				errChan <- fmt.Errorf("reverse relay error: %v", err)
			}
		}()
	}
	
	// Wait for completion or error
	go func() {
		wg.Wait()
		close(errChan)
	}()
	
	// Collect any errors
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}
	
	// Show statistics
	if r.ShowStats {
		r.printStats()
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("relay errors: %v", errors)
	}
	
	return nil
}

// copyData copies data from src to dst with statistics
func (r *Relay) copyData(dst io.Writer, src io.Reader, direction string) (int64, error) {
	buffer := make([]byte, r.BufferSize)
	var totalBytes int64
	
	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err == io.EOF {
				logger.Debug("EOF reached in %s direction", direction)
				return totalBytes, nil
			}
			return totalBytes, err
		}
		
		written, err := dst.Write(buffer[:n])
		if err != nil {
			return totalBytes, err
		}
		
		totalBytes += int64(written)
		
		if r.ShowStats && totalBytes%1048576 == 0 { // Every MB
			logger.Debug("Relayed %d MB in %s direction", totalBytes/1048576, direction)
		}
	}
}

// printStats prints relay statistics
func (r *Relay) printStats() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	logger.Info("Relay Statistics:")
	logger.Info("  Forward bytes: %d", r.bytesForward)
	logger.Info("  Reverse bytes: %d", r.bytesReverse)
	logger.Info("  Total bytes: %d", r.bytesForward+r.bytesReverse)
}

// GetStats returns current relay statistics
func (r *Relay) GetStats() (forward, reverse int64) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.bytesForward, r.bytesReverse
}

// ProxyServer implements a simple proxy server
type ProxyServer struct {
	ListenAddr string
	TargetAddr string
	relay      *Relay
}

// NewProxyServer creates a new proxy server
func NewProxyServer(listenAddr, targetAddr string) *ProxyServer {
	return &ProxyServer{
		ListenAddr: listenAddr,
		TargetAddr: targetAddr,
		relay:      NewRelay(),
	}
}

// Start starts the proxy server
func (ps *ProxyServer) Start() error {
	listener, err := net.Listen("tcp", ps.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", ps.ListenAddr, err)
	}
	defer listener.Close()
	
	logger.Info("Proxy server listening on %s, forwarding to %s", ps.ListenAddr, ps.TargetAddr)
	
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection: %v", err)
			continue
		}
		
		go ps.handleConnection(clientConn)
	}
}

// handleConnection handles a single proxy connection
func (ps *ProxyServer) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()
	
	logger.Info("New proxy connection from %s", clientConn.RemoteAddr())
	
	// Connect to target
	targetConn, err := net.Dial("tcp", ps.TargetAddr)
	if err != nil {
		logger.Error("Failed to connect to target %s: %v", ps.TargetAddr, err)
		return
	}
	defer targetConn.Close()
	
	// Start relaying
	if err := ps.relay.RelayConnections(clientConn, targetConn); err != nil {
		logger.Error("Relay error: %v", err)
	}
	
	logger.Info("Proxy connection closed")
}

// PortForwarder implements port forwarding functionality
type PortForwarder struct {
	LocalAddr  string
	RemoteAddr string
	relay      *Relay
}

// NewPortForwarder creates a new port forwarder
func NewPortForwarder(localAddr, remoteAddr string) *PortForwarder {
	return &PortForwarder{
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		relay:      NewRelay(),
	}
}

// Forward starts port forwarding
func (pf *PortForwarder) Forward() error {
	listener, err := net.Listen("tcp", pf.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", pf.LocalAddr, err)
	}
	defer listener.Close()
	
	logger.Info("Port forwarding from %s to %s", pf.LocalAddr, pf.RemoteAddr)
	
	for {
		localConn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection: %v", err)
			continue
		}
		
		go pf.handleForward(localConn)
	}
}

// handleForward handles a single port forward connection
func (pf *PortForwarder) handleForward(localConn net.Conn) {
	defer localConn.Close()
	
	logger.Debug("Forwarding connection from %s", localConn.RemoteAddr())
	
	// Connect to remote
	remoteConn, err := net.Dial("tcp", pf.RemoteAddr)
	if err != nil {
		logger.Error("Failed to connect to remote %s: %v", pf.RemoteAddr, err)
		return
	}
	defer remoteConn.Close()
	
	// Start relaying
	if err := pf.relay.RelayConnections(localConn, remoteConn); err != nil {
		logger.Error("Forward error: %v", err)
	}
	
	logger.Debug("Forward connection closed")
}

// TunnelServer implements a simple tunneling server
type TunnelServer struct {
	ListenAddr string
	relay      *Relay
	clients    map[string]net.Conn
	mutex      sync.RWMutex
}

// NewTunnelServer creates a new tunnel server
func NewTunnelServer(listenAddr string) *TunnelServer {
	return &TunnelServer{
		ListenAddr: listenAddr,
		relay:      NewRelay(),
		clients:    make(map[string]net.Conn),
	}
}

// Start starts the tunnel server
func (ts *TunnelServer) Start() error {
	listener, err := net.Listen("tcp", ts.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", ts.ListenAddr, err)
	}
	defer listener.Close()
	
	logger.Info("Tunnel server listening on %s", ts.ListenAddr)
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection: %v", err)
			continue
		}
		
		go ts.handleTunnel(conn)
	}
}

// handleTunnel handles a tunnel connection
func (ts *TunnelServer) handleTunnel(conn net.Conn) {
	defer conn.Close()
	
	clientID := conn.RemoteAddr().String()
	logger.Info("New tunnel client: %s", clientID)
	
	ts.mutex.Lock()
	ts.clients[clientID] = conn
	ts.mutex.Unlock()
	
	defer func() {
		ts.mutex.Lock()
		delete(ts.clients, clientID)
		ts.mutex.Unlock()
		logger.Info("Tunnel client disconnected: %s", clientID)
	}()
	
	// Keep connection alive and handle tunnel commands
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				logger.Error("Tunnel read error: %v", err)
			}
			break
		}
		
		// Echo back for now (simple tunnel)
		if _, err := conn.Write(buffer[:n]); err != nil {
			logger.Error("Tunnel write error: %v", err)
			break
		}
	}
}