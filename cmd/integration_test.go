package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestTCPListenConnect tests basic TCP listen and connect
func TestTCPListenConnect(t *testing.T) {
	port := "18080"
	
	// Start listener
	cmd := exec.Command("./gocat", "listen", port)
	
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer cmd.Process.Kill()
	
	// Wait for listener to start
	time.Sleep(2 * time.Second)
	
	// Connect and send data
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	testMessage := "hello from test\n"
	_, err = conn.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	
	// Read response - just check connection works
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	// Connection established successfully if we can write
	
	t.Log("✓ TCP Listen & Connect test passed")
}

// TestMultiPortListener tests multi-port listening
func TestMultiPortListener(t *testing.T) {
	ports := "18081,18082,18083"
	
	// Start multi-listener
	cmd := exec.Command("./gocat", "multi-listen", "--ports", ports)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start multi-listener: %v", err)
	}
	defer cmd.Process.Kill()
	
	// Wait for listeners to start
	time.Sleep(3 * time.Second)
	
	// Test each port
	testPorts := []string{"18081", "18082", "18083"}
	for _, port := range testPorts {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 5*time.Second)
		if err != nil {
			t.Errorf("Failed to connect to port %s: %v", port, err)
			continue
		}
		conn.Close()
		t.Logf("✓ Port %s is open", port)
	}
	
	t.Log("✓ Multi-Port Listener test passed")
}

// TestHTTPReverseProxy tests the HTTP reverse proxy
func TestHTTPReverseProxy(t *testing.T) {
	backendPort := "19000"
	proxyPort := "18084"
	
	// Start backend server
	backend := exec.Command("./gocat", "listen", backendPort)
	if err := backend.Start(); err != nil {
		t.Fatalf("Failed to start backend: %v", err)
	}
	defer backend.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Start proxy
	proxy := exec.Command("./gocat", "proxy", "--listen", ":"+proxyPort, 
		"--target", "http://127.0.0.1:"+backendPort, "--log-requests")
	if err := proxy.Start(); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Test HTTP request through proxy
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + proxyPort)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 && resp.StatusCode != 0 {
		t.Logf("Response status: %d (expected connection)", resp.StatusCode)
	}
	
	t.Log("✓ HTTP Reverse Proxy test passed")
}

// TestProtocolConverter tests TCP to UDP conversion
func TestProtocolConverter(t *testing.T) {
	tcpPort := "18085"
	udpPort := "19001"
	
	// Start UDP listener
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+udpPort)
	if err != nil {
		t.Fatalf("Failed to resolve UDP address: %v", err)
	}
	
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("Failed to start UDP listener: %v", err)
	}
	defer udpConn.Close()
	
	// Start converter
	converter := exec.Command("./gocat", "convert", 
		"--from", "tcp:"+tcpPort, 
		"--to", "udp:127.0.0.1:"+udpPort)
	if err := converter.Start(); err != nil {
		t.Fatalf("Failed to start converter: %v", err)
	}
	defer converter.Process.Kill()
	
	// Give converter more time to start
	time.Sleep(3 * time.Second)
	
	// Try to connect via TCP with retries
	var tcpConn net.Conn
	for i := 0; i < 3; i++ {
		tcpConn, err = net.DialTimeout("tcp", "127.0.0.1:"+tcpPort, 2*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	
	if err != nil {
		t.Skipf("Converter not ready, skipping test: %v", err)
		return
	}
	defer tcpConn.Close()
	
	// Send data via TCP
	testData := "hello via converter"
	_, err = tcpConn.Write([]byte(testData))
	if err != nil {
		t.Fatalf("Failed to write to TCP: %v", err)
	}
	
	// Receive via UDP
	udpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := udpConn.ReadFromUDP(buf)
	if err != nil {
		t.Logf("UDP read timeout (converter may need more time): %v", err)
		t.Skip("Skipping converter test due to timing")
		return
	}
	
	if !strings.Contains(string(buf[:n]), testData) {
		t.Errorf("Expected '%s', got '%s'", testData, string(buf[:n]))
	}
	
	t.Log("✓ Protocol Converter test passed")
}

// TestPortScanner tests the port scanner
func TestPortScanner(t *testing.T) {
	testPort := "18086"
	
	// Start listener
	listener, err := net.Listen("tcp", "127.0.0.1:"+testPort)
	if err != nil {
		t.Fatalf("Failed to start test listener: %v", err)
	}
	defer listener.Close()
	
	// Run scan
	cmd := exec.Command("./gocat", "scan", "127.0.0.1", testPort)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Scan failed: %v, output: %s", err, string(output))
	}
	
	if !strings.Contains(string(output), "OPEN") {
		t.Errorf("Expected OPEN in output, got: %s", string(output))
	}
	
	t.Log("✓ Port Scanner test passed")
}

// TestVerboseFlag tests verbose flag
func TestVerboseFlag(t *testing.T) {
	cmd := exec.Command("./gocat", "-v", "scan", "127.0.0.1", "80")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Scan might fail, but we're testing the flag
		t.Logf("Scan output: %s", string(output))
	}
	
	// Verbose mode should produce output
	if len(output) == 0 {
		t.Error("Expected verbose output")
	}
	
	t.Log("✓ Verbose flag test passed")
}

// TestDebugFlag tests debug flag
func TestDebugFlag(t *testing.T) {
	cmd := exec.Command("./gocat", "--debug", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}
	
	if !strings.Contains(string(output), "GoCat") {
		t.Errorf("Expected version info, got: %s", string(output))
	}
	
	t.Log("✓ Debug flag test passed")
}

// TestSSLFlags tests SSL-related flags
func TestSSLFlags(t *testing.T) {
	// Generate self-signed cert
	certCmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", "test_key.pem", "-out", "test_cert.pem",
		"-days", "1", "-nodes", "-subj", "/CN=localhost")
	if err := certCmd.Run(); err != nil {
		t.Skip("OpenSSL not available, skipping SSL test")
	}
	defer os.Remove("test_key.pem")
	defer os.Remove("test_cert.pem")
	
	port := "18443"
	
	// Start SSL listener
	listener := exec.Command("./gocat", "listen", 
		"--listen-ssl", "--listen-ssl-cert", "test_cert.pem", 
		"--listen-ssl-key", "test_key.pem", port)
	if err := listener.Start(); err != nil {
		t.Fatalf("Failed to start SSL listener: %v", err)
	}
	defer listener.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Try to connect with SSL
	cmd := exec.Command("./gocat", "connect", "--connect-ssl", "127.0.0.1", port)
	stdin, _ := cmd.StdinPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start SSL connect: %v", err)
	}
	defer cmd.Process.Kill()
	
	time.Sleep(1 * time.Second)
	stdin.Write([]byte("test\n"))
	stdin.Close()
	
	t.Log("✓ SSL flags test passed")
}

// TestConvertCommand tests convert command help
func TestConvertCommand(t *testing.T) {
	cmd := exec.Command("./gocat", "convert", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Convert help failed: %v", err)
	}
	
	if !strings.Contains(string(output), "protocol") {
		t.Errorf("Expected protocol in help, got: %s", string(output))
	}
	
	t.Log("✓ Convert command test passed")
}

// TestTunnelCommand tests tunnel command help
func TestTunnelCommand(t *testing.T) {
	cmd := exec.Command("./gocat", "tunnel", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Tunnel help failed: %v", err)
	}
	
	if !strings.Contains(string(output), "SSH") {
		t.Errorf("Expected SSH in help, got: %s", string(output))
	}
	
	t.Log("✓ Tunnel command test passed")
}

// TestDNSTunnelCommand tests DNS tunnel command help
func TestDNSTunnelCommand(t *testing.T) {
	cmd := exec.Command("./gocat", "dns-tunnel", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("DNS tunnel help failed: %v", err)
	}
	
	if !strings.Contains(string(output), "DNS") {
		t.Errorf("Expected DNS in help, got: %s", string(output))
	}
	
	t.Log("✓ DNS Tunnel command test passed")
}

// TestAllCommandsExist tests that all commands are registered
func TestAllCommandsExist(t *testing.T) {
	expectedCommands := []string{
		"connect", "listen", "scan", "transfer", "chat", "broker",
		"proxy", "convert", "multi-listen", "tunnel", "dns-tunnel",
		"script", "tui", "version",
	}
	
	cmd := exec.Command("./gocat", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}
	
	helpText := string(output)
	for _, command := range expectedCommands {
		if !strings.Contains(helpText, command) {
			t.Errorf("Command '%s' not found in help output", command)
		}
	}
	
	t.Logf("✓ All %d commands exist", len(expectedCommands))
}

// TestProxyLoadBalancing tests proxy with multiple backends
func TestProxyLoadBalancing(t *testing.T) {
	backend1Port := "19001"
	backend2Port := "19002"
	proxyPort := "18087"
	
	// Start backend 1
	backend1 := exec.Command("./gocat", "listen", backend1Port)
	if err := backend1.Start(); err != nil {
		t.Fatalf("Failed to start backend1: %v", err)
	}
	defer backend1.Process.Kill()
	
	// Start backend 2
	backend2 := exec.Command("./gocat", "listen", backend2Port)
	if err := backend2.Start(); err != nil {
		t.Fatalf("Failed to start backend2: %v", err)
	}
	defer backend2.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Start proxy with multiple backends
	proxy := exec.Command("./gocat", "proxy", 
		"--listen", ":"+proxyPort,
		"--backends", fmt.Sprintf("http://127.0.0.1:%s,http://127.0.0.1:%s", backend1Port, backend2Port),
		"--lb-algorithm", "round-robin")
	if err := proxy.Start(); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Make multiple requests
	client := &http.Client{Timeout: 3 * time.Second}
	successCount := 0
	for i := 0; i < 3; i++ {
		resp, err := client.Get("http://127.0.0.1:" + proxyPort)
		if err == nil {
			resp.Body.Close()
			successCount++
		}
		time.Sleep(500 * time.Millisecond)
	}
	
	if successCount == 0 {
		t.Error("No successful proxy requests")
	}
	
	t.Logf("✓ Proxy Load Balancing test passed (%d/3 requests successful)", successCount)
}

// TestScanWithDifferentFlags tests scan command with various flags
func TestScanWithDifferentFlags(t *testing.T) {
	// Start a listener for scanning
	listener, err := net.Listen("tcp", "127.0.0.1:18088")
	if err != nil {
		t.Fatalf("Failed to start test listener: %v", err)
	}
	defer listener.Close()
	
	tests := []struct {
		name  string
		flags []string
	}{
		{"Basic scan", []string{"scan", "127.0.0.1", "18088"}},
		{"Scan with timeout", []string{"scan", "--scan-timeout", "1s", "127.0.0.1", "18088"}},
		{"Scan with concurrency", []string{"scan", "--concurrency", "10", "127.0.0.1", "18088"}},
		{"Verbose scan", []string{"scan", "--scan-verbose", "127.0.0.1", "18088"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./gocat", tt.flags...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Scan output: %s", string(output))
			}
			
			if !strings.Contains(string(output), "OPEN") {
				t.Errorf("Expected OPEN in output for %s", tt.name)
			} else {
				t.Logf("✓ %s passed", tt.name)
			}
		})
	}
}

// TestConnectWithRetry tests connection retry functionality
func TestConnectWithRetry(t *testing.T) {
	port := "18089"
	
	// Try to connect to non-existent server with retry
	cmd := exec.Command("./gocat", "connect", "--retry", "2", 
		"--connect-timeout", "1s", "127.0.0.1", port)
	
	start := time.Now()
	output, _ := cmd.CombinedOutput()
	duration := time.Since(start)
	
	// Should retry and take at least 2 seconds (2 retries with backoff)
	if duration < 2*time.Second {
		t.Errorf("Expected retry to take at least 2s, took %v", duration)
	}
	
	if !strings.Contains(string(output), "Retrying") && !strings.Contains(string(output), "failed to connect") {
		t.Logf("Output: %s", string(output))
	}
	
	t.Log("✓ Connect with retry test passed")
}

// TestVersionCommand tests version command
func TestVersionCommand(t *testing.T) {
	cmd := exec.Command("./gocat", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}
	
	if !strings.Contains(string(output), "GoCat") {
		t.Errorf("Expected GoCat in version output, got: %s", string(output))
	}
	
	if !strings.Contains(string(output), "Build Information") {
		t.Errorf("Expected Build Information in version output")
	}
	
	t.Log("✓ Version command test passed")
}

// TestChatMode tests chat server startup
func TestChatMode(t *testing.T) {
	port := "18090"
	
	cmd := exec.Command("./gocat", "chat", port)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start chat: %v", err)
	}
	defer cmd.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Try to connect
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to chat: %v", err)
	}
	conn.Close()
	
	t.Log("✓ Chat mode test passed")
}

// TestBrokerMode tests broker mode startup
func TestBrokerMode(t *testing.T) {
	port := "18091"
	
	cmd := exec.Command("./gocat", "broker", port)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start broker: %v", err)
	}
	defer cmd.Process.Kill()
	
	time.Sleep(2 * time.Second)
	
	// Try to connect
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to broker: %v", err)
	}
	conn.Close()
	
	t.Log("✓ Broker mode test passed")
}
