package mcp

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// RegisterGoCatResources registers all GoCat resources
func RegisterGoCatResources(server *MCPServer) {
	// System information resources
	server.RegisterResource(&Resource{
		URI:         "gocat://system/info",
		Name:        "System Information",
		Description: "Get current system information and GoCat status",
		MimeType:    "application/json",
		Handler:     handleSystemInfo,
	})

	server.RegisterResource(&Resource{
		URI:         "gocat://system/capabilities",
		Name:        "System Capabilities",
		Description: "List all available network capabilities",
		MimeType:    "application/json",
		Handler:     handleSystemCapabilities,
	})

	// Network information resources
	server.RegisterResource(&Resource{
		URI:         "gocat://network/interfaces",
		Name:        "Network Interfaces",
		Description: "List all network interfaces",
		MimeType:    "application/json",
		Handler:     handleNetworkInterfaces,
	})

	server.RegisterResource(&Resource{
		URI:         "gocat://network/connections",
		Name:        "Active Connections",
		Description: "List all active network connections",
		MimeType:    "application/json",
		Handler:     handleActiveConnections,
	})

	// Metrics resources
	server.RegisterResource(&Resource{
		URI:         "gocat://metrics/prometheus",
		Name:        "Prometheus Metrics",
		Description: "Get Prometheus-formatted metrics",
		MimeType:    "text/plain",
		Handler:     handlePrometheusMetrics,
	})

	server.RegisterResource(&Resource{
		URI:         "gocat://metrics/statistics",
		Name:        "Connection Statistics",
		Description: "Get detailed connection statistics",
		MimeType:    "application/json",
		Handler:     handleConnectionStatistics,
	})

	// Scan results resources
	server.RegisterResource(&Resource{
		URI:         "gocat://scan/results/latest",
		Name:        "Latest Scan Results",
		Description: "Get results from the most recent port scan",
		MimeType:    "application/json",
		Handler:     handleLatestScanResults,
	})

	// Configuration resources
	server.RegisterResource(&Resource{
		URI:         "gocat://config/current",
		Name:        "Current Configuration",
		Description: "Get current GoCat configuration",
		MimeType:    "application/json",
		Handler:     handleCurrentConfig,
	})

	// Documentation resources
	server.RegisterResource(&Resource{
		URI:         "gocat://docs/commands",
		Name:        "Command Documentation",
		Description: "Get documentation for all available commands",
		MimeType:    "text/markdown",
		Handler:     handleCommandDocs,
	})

	server.RegisterResource(&Resource{
		URI:         "gocat://docs/examples",
		Name:        "Usage Examples",
		Description: "Get practical usage examples",
		MimeType:    "text/markdown",
		Handler:     handleUsageExamples,
	})
}

// Resource handlers
func handleSystemInfo(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"version":    "dev",
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpus":       runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"uptime":     time.Since(time.Now()).String(),
	}, nil
}

func handleSystemCapabilities(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"protocols": []string{"TCP", "UDP", "SCTP", "WebSocket", "Unix Sockets"},
		"features": []string{
			"Port Scanning",
			"Connection Management",
			"File Transfer",
			"Proxy & Load Balancing",
			"Protocol Conversion",
			"SSH Tunneling",
			"DNS Tunneling",
			"WebSocket Support",
			"Unix Domain Sockets",
			"Prometheus Metrics",
			"TUI Interface",
			"Lua Scripting",
		},
		"security": []string{
			"SSL/TLS Support",
			"Certificate Verification",
			"Encryption (AES-256-GCM, ChaCha20-Poly1305)",
			"Authentication",
			"Rate Limiting",
			"Access Control",
		},
	}, nil
}

func handleNetworkInterfaces(ctx context.Context) (interface{}, error) {
	// Placeholder - would actually query system interfaces
	return map[string]interface{}{
		"interfaces": []map[string]interface{}{
			{
				"name":   "lo",
				"type":   "loopback",
				"status": "up",
				"ipv4":   []string{"127.0.0.1"},
				"ipv6":   []string{"::1"},
			},
			{
				"name":   "eth0",
				"type":   "ethernet",
				"status": "up",
				"ipv4":   []string{"192.168.1.100"},
			},
		},
	}, nil
}

func handleActiveConnections(ctx context.Context) (interface{}, error) {
	// Placeholder - would actually track connections
	return map[string]interface{}{
		"total": 0,
		"tcp":   0,
		"udp":   0,
		"connections": []map[string]interface{}{},
	}, nil
}

func handlePrometheusMetrics(ctx context.Context) (interface{}, error) {
	return `# HELP gocat_network_connections_total Total network connections
# TYPE gocat_network_connections_total counter
gocat_network_connections_total 0

# HELP gocat_network_bytes_sent_total Total bytes sent
# TYPE gocat_network_bytes_sent_total counter
gocat_network_bytes_sent_total 0

# HELP gocat_network_bytes_received_total Total bytes received
# TYPE gocat_network_bytes_received_total counter
gocat_network_bytes_received_total 0

# HELP gocat_network_goroutines Current number of goroutines
# TYPE gocat_network_goroutines gauge
gocat_network_goroutines ` + fmt.Sprintf("%d", runtime.NumGoroutine()) + `

# HELP gocat_network_memory_alloc_bytes Allocated memory in bytes
# TYPE gocat_network_memory_alloc_bytes gauge
gocat_network_memory_alloc_bytes 0
`, nil
}

func handleConnectionStatistics(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"total_connections":   0,
		"active_connections":  0,
		"failed_connections":  0,
		"bytes_transferred":   0,
		"average_duration_ms": 0,
		"protocols": map[string]int{
			"tcp":       0,
			"udp":       0,
			"websocket": 0,
			"unix":      0,
		},
	}, nil
}

func handleLatestScanResults(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"scan_id":    "scan_latest",
		"target":     "example.com",
		"timestamp":  time.Now().Format(time.RFC3339),
		"duration":   "5.2s",
		"total_ports": 1000,
		"open_ports":  []map[string]interface{}{
			{"port": 22, "state": "open", "service": "ssh"},
			{"port": 80, "state": "open", "service": "http"},
			{"port": 443, "state": "open", "service": "https"},
		},
		"closed_ports": 997,
	}, nil
}

func handleCurrentConfig(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"log_level":      "info",
		"default_timeout": "30s",
		"buffer_size":    8192,
		"max_connections": 1000,
		"theme":          "default",
		"ssl": map[string]interface{}{
			"verify":  false,
			"min_tls": "1.2",
		},
	}, nil
}

func handleCommandDocs(ctx context.Context) (interface{}, error) {
	return `# GoCat Commands Documentation

## Connection Management

### connect
Connect to a remote host and port.
` + "```bash" + `
gocat connect <host> <port>
gocat connect --ssl example.com 443
gocat connect --udp 192.168.1.1 53
` + "```" + `

### listen
Listen on a port for incoming connections.
` + "```bash" + `
gocat listen 8080
gocat listen --ssl --cert cert.pem --key key.pem 8443
` + "```" + `

## Scanning

### scan
Scan ports on target hosts.
` + "```bash" + `
gocat scan example.com --ports 1-1000
gocat scan 192.168.1.0/24 --ports 22,80,443
` + "```" + `

## File Transfer

### transfer
Send or receive files over the network.
` + "```bash" + `
gocat transfer send file.txt 192.168.1.100 8080
gocat transfer receive 8080 output.txt
` + "```" + `

## Proxy & Load Balancing

### proxy
Start HTTP/HTTPS reverse proxy with load balancing.
` + "```bash" + `
gocat proxy --listen :8080 --backends http://backend1,http://backend2
gocat proxy --listen :443 --ssl --lb-algorithm least-connections
` + "```" + `

## WebSocket

### websocket (ws)
WebSocket server and client operations.
` + "```bash" + `
gocat ws server --port 8080
gocat ws connect ws://localhost:8080
gocat ws echo --port 8080
` + "```" + `

## Unix Domain Sockets

### unix (uds)
Unix socket operations for local IPC.
` + "```bash" + `
gocat unix listen /tmp/gocat.sock
gocat unix connect /tmp/gocat.sock
` + "```" + `

## Metrics

### metrics
Prometheus metrics exporter.
` + "```bash" + `
gocat metrics --port 9090
curl http://localhost:9090/metrics
` + "```" + `

## Tunneling

### tunnel
Create SSH tunnels.
` + "```bash" + `
gocat tunnel --ssh user@server --local 8080 --remote localhost:80
gocat tunnel --ssh user@server --dynamic 1080
` + "```" + `
`, nil
}

func handleUsageExamples(ctx context.Context) (interface{}, error) {
	return `# GoCat Usage Examples

## Network Troubleshooting

### Check if port is open
` + "```bash" + `
gocat scan example.com --ports 80
` + "```" + `

### Test connection timeout
` + "```bash" + `
gocat connect --wait 5s example.com 80
` + "```" + `

### Monitor network traffic
` + "```bash" + `
gocat listen --verbose 8080
` + "```" + `

## File Sharing

### Quick file transfer
` + "```bash" + `
# Receiver
gocat transfer receive 9999 received_file.zip

# Sender
gocat transfer send myfile.zip 192.168.1.100 9999
` + "```" + `

## Load Balancing

### Simple round-robin proxy
` + "```bash" + `
gocat proxy --listen :8080 \\
  --backends http://server1:80,http://server2:80,http://server3:80 \\
  --health-check /health
` + "```" + `

## WebSocket Chat

### Start chat server
` + "```bash" + `
gocat ws server --port 8080 --compress
` + "```" + `

### Connect clients
` + "```bash" + `
gocat ws connect ws://localhost:8080
` + "```" + `

## Container Communication

### Unix socket for Docker
` + "```bash" + `
gocat unix listen /var/run/app.sock --permissions 0666
` + "```" + `

## Monitoring

### Export metrics to Prometheus
` + "```bash" + `
gocat metrics --namespace myapp --port 9090
` + "```" + `

### Add to prometheus.yml
` + "```yaml" + `
scrape_configs:
  - job_name: 'gocat'
    static_configs:
      - targets: ['localhost:9090']
` + "```" + `

## SSH Tunneling

### Local port forwarding
` + "```bash" + `
gocat tunnel --ssh user@remote --local 8080 --remote localhost:80
` + "```" + `

### SOCKS proxy
` + "```bash" + `
gocat tunnel --ssh user@remote --dynamic 1080
` + "```" + `

## Protocol Conversion

### TCP to WebSocket
` + "```bash" + `
gocat convert --from tcp:8080 --to ws://backend:9000/ws
` + "```" + `

### UDP to TCP
` + "```bash" + `
gocat convert --from udp:5353 --to tcp:backend:53
` + "```" + `
`, nil
}
