package mcp

import (
	"context"
	"fmt"
)

// RegisterGoCatPrompts registers all GoCat prompts
func RegisterGoCatPrompts(server *MCPServer) {
	// Network troubleshooting prompts
	server.RegisterPrompt(&Prompt{
		Name:        "network_troubleshoot",
		Description: "Diagnose network connectivity issues",
		Arguments: []PromptArgument{
			{
				Name:        "target",
				Description: "Target host or IP to diagnose",
				Required:    true,
			},
			{
				Name:        "symptoms",
				Description: "Observed symptoms or error messages",
				Required:    false,
			},
		},
		Handler: handleNetworkTroubleshoot,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "security_audit",
		Description: "Perform network security audit",
		Arguments: []PromptArgument{
			{
				Name:        "target",
				Description: "Target host or network to audit",
				Required:    true,
			},
			{
				Name:        "scope",
				Description: "Audit scope: basic, standard, or comprehensive",
				Required:    false,
			},
		},
		Handler: handleSecurityAudit,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "port_scan_analysis",
		Description: "Analyze port scan results and provide insights",
		Arguments: []PromptArgument{
			{
				Name:        "target",
				Description: "Target that was scanned",
				Required:    true,
			},
			{
				Name:        "open_ports",
				Description: "List of discovered open ports",
				Required:    false,
			},
		},
		Handler: handlePortScanAnalysis,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "connection_guide",
		Description: "Guide for establishing specific types of connections",
		Arguments: []PromptArgument{
			{
				Name:        "protocol",
				Description: "Protocol type (tcp, udp, websocket, unix, etc.)",
				Required:    true,
			},
			{
				Name:        "scenario",
				Description: "Use case scenario",
				Required:    false,
			},
		},
		Handler: handleConnectionGuide,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "proxy_setup",
		Description: "Guide for setting up reverse proxy and load balancing",
		Arguments: []PromptArgument{
			{
				Name:        "backends",
				Description: "Number of backend servers",
				Required:    true,
			},
			{
				Name:        "features",
				Description: "Required features (ssl, health-check, etc.)",
				Required:    false,
			},
		},
		Handler: handleProxySetup,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "file_transfer_guide",
		Description: "Guide for efficient file transfer",
		Arguments: []PromptArgument{
			{
				Name:        "file_size",
				Description: "Approximate file size",
				Required:    false,
			},
			{
				Name:        "network_type",
				Description: "Network type (lan, wan, internet)",
				Required:    false,
			},
		},
		Handler: handleFileTransferGuide,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "monitoring_setup",
		Description: "Guide for setting up monitoring and metrics",
		Arguments: []PromptArgument{
			{
				Name:        "platform",
				Description: "Monitoring platform (prometheus, grafana, etc.)",
				Required:    false,
			},
		},
		Handler: handleMonitoringSetup,
	})

	server.RegisterPrompt(&Prompt{
		Name:        "tunnel_guide",
		Description: "Guide for creating SSH tunnels",
		Arguments: []PromptArgument{
			{
				Name:        "tunnel_type",
				Description: "Tunnel type (local, remote, dynamic)",
				Required:    true,
			},
			{
				Name:        "use_case",
				Description: "Specific use case",
				Required:    false,
			},
		},
		Handler: handleTunnelGuide,
	})
}

// Prompt handlers
func handleNetworkTroubleshoot(ctx context.Context, args map[string]string) (string, error) {
	target := args["target"]
	symptoms := args["symptoms"]

	prompt := fmt.Sprintf(`# Network Troubleshooting for %s

I need to diagnose connectivity issues with %s.

## Symptoms
%s

## Diagnostic Steps

1. **Basic Connectivity Test**
   - Check if the host is reachable
   - Test DNS resolution
   - Verify routing

2. **Port Scanning**
   - Scan common ports (22, 80, 443, etc.)
   - Check if services are listening
   - Identify firewall rules

3. **Connection Analysis**
   - Test TCP handshake
   - Check TLS/SSL if applicable
   - Monitor connection timeouts

4. **Network Path Analysis**
   - Trace route to destination
   - Identify bottlenecks
   - Check packet loss

## Recommended GoCat Commands

` + "```bash" + `
# Test basic connectivity
gocat scan %s --ports 22,80,443

# Test specific port
gocat connect --wait 5s %s <port>

# Comprehensive scan
gocat scan %s --ports 1-65535 --concurrency 500

# Monitor connection
gocat connect --verbose %s <port>
` + "```" + `

Please provide the results of these tests for further analysis.
`, target, target, symptoms, target, target, target, target)

	return prompt, nil
}

func handleSecurityAudit(ctx context.Context, args map[string]string) (string, error) {
	target := args["target"]
	scope := args["scope"]
	if scope == "" {
		scope = "standard"
	}

	prompt := fmt.Sprintf(`# Security Audit for %s

Performing %s security audit.

## Audit Checklist

### 1. Port and Service Discovery
- [ ] Scan all TCP ports
- [ ] Scan common UDP ports
- [ ] Identify running services
- [ ] Check for unnecessary open ports

### 2. TLS/SSL Configuration
- [ ] Verify certificate validity
- [ ] Check TLS version support
- [ ] Test cipher suite configuration
- [ ] Verify certificate chain

### 3. Access Control
- [ ] Test authentication mechanisms
- [ ] Check for default credentials
- [ ] Verify IP-based restrictions
- [ ] Test rate limiting

### 4. Protocol Security
- [ ] Check for unencrypted protocols
- [ ] Verify secure protocol versions
- [ ] Test for protocol downgrade attacks

## GoCat Audit Commands

` + "```bash" + `
# Full port scan
gocat scan %s --ports 1-65535 --concurrency 1000

# SSL/TLS test
gocat connect --ssl --verify-cert %s 443

# Service enumeration
gocat scan %s --ports 21,22,23,25,80,110,143,443,993,995,3306,5432,8080,8443

# UDP service scan
gocat scan --udp %s --ports 53,123,161,500

# WebSocket security test
gocat ws connect wss://%s

# Unix socket enumeration
find /tmp /var/run -type s -ls
` + "```" + `

## Security Recommendations
Based on the scan results, I will provide specific security recommendations.
`, target, scope, target, target, target, target, target)

	return prompt, nil
}

func handlePortScanAnalysis(ctx context.Context, args map[string]string) (string, error) {
	target := args["target"]
	openPorts := args["open_ports"]

	prompt := fmt.Sprintf(`# Port Scan Analysis for %s

## Discovered Open Ports
%s

## Analysis Framework

### 1. Service Identification
For each open port:
- Identify the service
- Check known vulnerabilities
- Verify if service should be exposed

### 2. Risk Assessment
- Critical services (22, 3389, etc.)
- Web services (80, 443, 8080, etc.)
- Database ports (3306, 5432, etc.)
- Unnecessary services

### 3. Common Port Analysis

**SSH (22)**
- Remote access port
- Should use key-based auth
- Disable root login

**HTTP (80), HTTPS (443)**
- Web server ports
- Check for HTTPS redirect
- Verify TLS configuration

**Database Ports (3306, 5432, 27017)**
- Should NOT be publicly accessible
- Use firewalls or VPN
- Verify authentication

**RDP (3389)**
- Windows remote desktop
- High security risk if exposed
- Use VPN or whitelist IPs

## Detailed Investigation Commands

` + "```bash" + `
# Banner grabbing
for port in <open_ports>; do
    gocat connect %s $port
done

# SSL/TLS analysis for HTTPS services
gocat connect --ssl --verify-cert %s 443

# WebSocket check
gocat ws connect ws://%s:port

# Connection test with timeout
gocat connect --wait 3s %s <port>
` + "```" + `

## Recommendations
Based on these findings, I will provide specific security recommendations.
`, target, openPorts, target, target, target, target)

	return prompt, nil
}

func handleConnectionGuide(ctx context.Context, args map[string]string) (string, error) {
	protocol := args["protocol"]
	scenario := args["scenario"]

	guides := map[string]string{
		"tcp": `# TCP Connection Guide

## Basic TCP Connection
` + "```bash" + `
gocat connect example.com 80
` + "```" + `

## With SSL/TLS
` + "```bash" + `
gocat connect --ssl example.com 443
` + "```" + `

## With Timeout and Retry
` + "```bash" + `
gocat connect --wait 10s --retry 5 example.com 80
` + "```" + `

## Through Proxy
` + "```bash" + `
gocat connect --proxy socks5://proxy:1080 example.com 80
` + "```" + `
`,
		"udp": `# UDP Connection Guide

## Basic UDP Connection
` + "```bash" + `
gocat connect --udp example.com 53
` + "```" + `

## UDP Server
` + "```bash" + `
gocat listen --udp 5353
` + "```" + `

## UDP to TCP Conversion
` + "```bash" + `
gocat convert --from udp:5353 --to tcp:dns-server:53
` + "```" + `
`,
		"websocket": `# WebSocket Connection Guide

## WebSocket Server
` + "```bash" + `
gocat ws server --port 8080 --compress
` + "```" + `

## WebSocket Client
` + "```bash" + `
gocat ws connect ws://example.com:8080
gocat ws connect wss://secure.example.com/ws
` + "```" + `

## WebSocket Echo Server
` + "```bash" + `
gocat ws echo --port 8080
` + "```" + `
`,
		"unix": `# Unix Socket Connection Guide

## Unix Socket Server
` + "```bash" + `
gocat unix listen /tmp/app.sock --permissions 0660
` + "```" + `

## Unix Socket Client
` + "```bash" + `
gocat unix connect /tmp/app.sock
` + "```" + `

## Datagram Socket
` + "```bash" + `
gocat unix listen --type datagram /tmp/dgram.sock
` + "```" + `
`,
	}

	guide, exists := guides[protocol]
	if !exists {
		guide = "Protocol guide not found. Available: tcp, udp, websocket, unix"
	}

	if scenario != "" {
		guide += fmt.Sprintf("\n## Scenario: %s\n", scenario)
	}

	return guide, nil
}

func handleProxySetup(ctx context.Context, args map[string]string) (string, error) {
	backends := args["backends"]
	features := args["features"]

	prompt := fmt.Sprintf(`# Reverse Proxy Setup Guide

## Configuration
- Backend servers: %s
- Features: %s

## Basic Proxy Setup

` + "```bash" + `
gocat proxy --listen :8080 --backends http://backend1,http://backend2,http://backend3
` + "```" + `

## With SSL/TLS

` + "```bash" + `
gocat proxy --listen :443 --ssl --cert cert.pem --key key.pem \\
  --backends http://backend1,http://backend2
` + "```" + `

## With Health Checks

` + "```bash" + `
gocat proxy --listen :8080 \\
  --backends http://backend1:80,http://backend2:80 \\
  --health-check /health \\
  --log-requests
` + "```" + `

## Load Balancing Algorithms

### Round Robin (Default)
` + "```bash" + `
gocat proxy --listen :8080 --backends http://b1,http://b2 --lb-algorithm round-robin
` + "```" + `

### Least Connections
` + "```bash" + `
gocat proxy --listen :8080 --backends http://b1,http://b2 --lb-algorithm least-connections
` + "```" + `

### IP Hash (Session Affinity)
` + "```bash" + `
gocat proxy --listen :8080 --backends http://b1,http://b2 --lb-algorithm ip-hash
` + "```" + `

## Advanced Features

### Connection Limits
` + "```bash" + `
gocat proxy --listen :8080 --backends http://b1,http://b2 --max-connections 5000
` + "```" + `

### Custom Headers
` + "```bash" + `
gocat proxy --listen :8080 --backends http://b1,http://b2 --modify-headers
` + "```" + `

## Monitoring
Monitor the proxy with Prometheus metrics:
` + "```bash" + `
gocat metrics --port 9090
curl http://localhost:9090/metrics
` + "```" + `
`, backends, features)

	return prompt, nil
}

func handleFileTransferGuide(ctx context.Context, args map[string]string) (string, error) {
	return `# File Transfer Guide

## Quick File Transfer

### Receiver
` + "```bash" + `
gocat transfer receive 9999 output.txt
` + "```" + `

### Sender
` + "```bash" + `
gocat transfer send file.txt 192.168.1.100 9999
` + "```" + `

## With Compression

` + "```bash" + `
# Sender
gocat transfer send --compress largefile.zip 192.168.1.100 9999

# Receiver
gocat transfer receive 9999 largefile.zip
` + "```" + `

## With Checksum Verification

` + "```bash" + `
gocat transfer send --checksum file.dat 192.168.1.100 9999
` + "```" + `

## Resume Interrupted Transfer

` + "```bash" + `
gocat transfer send --resume file.iso 192.168.1.100 9999
` + "```" + `

## Through SSH Tunnel

` + "```bash" + `
# Create tunnel
gocat tunnel --ssh user@remote --local 9999 --remote localhost:9999

# Transfer through tunnel
gocat transfer send file.txt localhost 9999
` + "```" + `

## Performance Tips

1. **Use compression for large text files**
2. **Skip checksum for LAN transfers (faster)**
3. **Use larger buffer for high-bandwidth networks**
4. **Consider UDP for real-time streaming**
`, nil
}

func handleMonitoringSetup(ctx context.Context, args map[string]string) (string, error) {
	platform := args["platform"]
	if platform == "" {
		platform = "prometheus"
	}

	return fmt.Sprintf(`# Monitoring Setup Guide - %s

## Start Metrics Exporter

` + "```bash" + `
gocat metrics --port 9090 --namespace myapp
` + "```" + `

## Prometheus Configuration

Add to prometheus.yml:
` + "```yaml" + `
scrape_configs:
  - job_name: 'gocat'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:9090']
        labels:
          app: 'gocat'
          environment: 'production'
` + "```" + `

## Available Metrics

- gocat_network_connections_total - Total connections
- gocat_network_bytes_sent_total - Total bytes sent
- gocat_network_bytes_received_total - Total bytes received
- gocat_network_goroutines - Active goroutines
- gocat_network_memory_alloc_bytes - Memory usage
- gocat_network_gc_runs_total - Garbage collection runs

## Grafana Dashboard

Import the GoCat dashboard:
1. Add Prometheus data source
2. Import dashboard JSON
3. Configure refresh interval

## Alert Rules

Example Prometheus alert:
` + "```yaml" + `
groups:
  - name: gocat
    rules:
      - alert: HighConnectionRate
        expr: rate(gocat_network_connections_total[5m]) > 1000
        annotations:
          summary: "High connection rate detected"
` + "```" + `

## Health Check Endpoint

` + "```bash" + `
curl http://localhost:9090/health
` + "```" + `
`, platform), nil
}

func handleTunnelGuide(ctx context.Context, args map[string]string) (string, error) {
	tunnelType := args["tunnel_type"]
	useCase := args["use_case"]

	guides := map[string]string{
		"local": `# Local Port Forwarding

Access remote service through SSH tunnel.

## Basic Local Forward
` + "```bash" + `
gocat tunnel --ssh user@remote --local 8080 --remote localhost:80
` + "```" + `

Now access http://localhost:8080 to reach remote:80

## Multiple Forwards
` + "```bash" + `
gocat tunnel --ssh user@remote --local 8080 --remote db:3306
gocat tunnel --ssh user@remote --local 8443 --remote web:443
` + "```" + `

## Use Cases
- Access internal database
- Reach web service behind firewall
- Connect to remote API
`,
		"remote": `# Remote Port Forwarding

Expose local service to remote network.

## Basic Remote Forward
` + "```bash" + `
gocat tunnel --ssh user@remote --reverse --local 3000 --remote 8080
` + "```" + `

Now remote:8080 forwards to localhost:3000

## Use Cases
- Share local development server
- Expose local API to remote team
- Temporary public access
`,
		"dynamic": `# Dynamic SOCKS Proxy

Create a SOCKS proxy through SSH.

## Basic SOCKS Proxy
` + "```bash" + `
gocat tunnel --ssh user@remote --dynamic 1080
` + "```" + `

## Configure Applications

### Firefox
Settings → Network → SOCKS Host: localhost:1080

### Curl
` + "```bash" + `
curl --socks5 localhost:1080 http://example.com
` + "```" + `

### Chrome
` + "```bash" + `
google-chrome --proxy-server="socks5://localhost:1080"
` + "```" + `

## Use Cases
- Browse through remote network
- Access geo-restricted content
- Secure public WiFi browsing
`,
	}

	guide, exists := guides[tunnelType]
	if !exists {
		guide = "Tunnel type not found. Available: local, remote, dynamic"
	}

	if useCase != "" {
		guide += fmt.Sprintf("\n## Your Use Case: %s\n", useCase)
	}

	return guide, nil
}
