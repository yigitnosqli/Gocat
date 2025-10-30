# 🤖 GoCat MCP Server

**Model Context Protocol (MCP) integration for GoCat**

Expose all GoCat network tools to AI assistants like Claude, enabling them to perform network operations, troubleshooting, and security assessments.

---

## 🎯 What is MCP?

**Model Context Protocol** is Anthropic's standard for connecting AI assistants to external tools and data sources. GoCat's MCP server allows AI assistants to:

- 🔧 **Execute network tools** (scan ports, connect, proxy, etc.)
- 📊 **Read resources** (metrics, system info, scan results)
- 💡 **Access prompts** (troubleshooting guides, security audits)

---

## 🚀 Quick Start

### Automatic Setup (Recommended)

The easiest way to get started:

```bash
# Interactive setup - shows menu of detected clients
gocat mcp setup

# List all detected AI clients
gocat mcp setup --list

# Setup for specific client
gocat mcp setup --client claude

# Setup for all detected clients at once
gocat mcp setup --client all
```

**Supported AI Clients:**
- ✅ Claude Desktop
- ✅ Cursor
- ✅ Continue
- ✅ Zed Editor  
- ✅ Windsurf

The setup command will:
1. 🔍 Detect installed AI clients
2. 📝 Show current configuration status
3. ⚙️ Automatically configure selected clients
4. ✅ Verify the setup

### Manual Setup

If you prefer manual configuration:

#### 1. Start MCP Server

```bash
gocat mcp
```

The server listens on **stdin/stdout** for MCP protocol messages.

#### 2. Configure Your AI Client

**Claude Desktop** - Add to config file:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

**Cursor** - Add to:
- `~/.config/Cursor/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`

**Configuration:**
```json
{
  "mcpServers": {
    "gocat": {
      "command": "/path/to/gocat",
      "args": ["mcp"]
    }
  }
}
```

#### 3. Restart & Use

1. Restart your AI client
2. Open a new conversation
3. Try these commands:

```
Can you scan example.com ports 1-1000?

Can you check if port 443 is open on example.com?

Set up a reverse proxy with load balancing for 3 backends

Create a WebSocket server on port 8080
```

---

## 🛠️ Available Tools (20+)

### Network Scanning
- **scan_ports** - Scan TCP/UDP ports with concurrency
- **scan_network** - Scan entire network (CIDR)

### Connections
- **connect** - TCP/UDP/SCTP connection with SSL
- **listen** - Start server (TCP/UDP/SSL)

### File Transfer
- **send_file** - Send files with compression/checksum
- **receive_file** - Receive files on port

### Proxy & Load Balancing
- **start_proxy** - Reverse proxy with health checks
- **convert_protocol** - TCP↔UDP, HTTP↔WebSocket
- **start_broker** - Connection broker/relay

### WebSocket
- **websocket_server** - Start WS server
- **websocket_connect** - Connect to WS

### Unix Sockets
- **unix_socket_listen** - Unix domain socket server
- **unix_socket_connect** - Connect to Unix socket

### Tunneling
- **create_ssh_tunnel** - Local/remote/dynamic SOCKS

### DNS
- **dns_lookup** - DNS resolution
- **dns_tunnel** - Covert DNS tunneling

### Monitoring
- **start_metrics_server** - Prometheus metrics
- **get_metrics** - Current metrics

### Multi-Port
- **multi_port_listen** - Listen on multiple ports

---

## 📊 Available Resources

AI assistants can read these resources:

| URI | Description |
|-----|-------------|
| `gocat://system/info` | System information |
| `gocat://system/capabilities` | Feature list |
| `gocat://network/interfaces` | Network interfaces |
| `gocat://network/connections` | Active connections |
| `gocat://metrics/prometheus` | Prometheus metrics |
| `gocat://metrics/statistics` | Connection stats |
| `gocat://scan/results/latest` | Latest scan results |
| `gocat://config/current` | Current config |
| `gocat://docs/commands` | Command docs |
| `gocat://docs/examples` | Usage examples |

---

## 💡 Available Prompts

Interactive guides for AI assistants:

- **network_troubleshoot** - Diagnose network issues
- **security_audit** - Perform security assessment
- **port_scan_analysis** - Analyze scan results
- **connection_guide** - Connection setup help
- **proxy_setup** - Proxy configuration guide
- **file_transfer_guide** - File transfer help
- **monitoring_setup** - Monitoring setup guide
- **tunnel_guide** - SSH tunnel guide

---

## 📝 Example Conversations

### Network Troubleshooting

**User:** *"I can't connect to my server at example.com, can you help diagnose?"*

**Claude with GoCat:**
```
I'll help diagnose the issue. Let me:

1. Check if the host is reachable
   [scan_ports: example.com, ports: 22,80,443]

2. Test specific connection
   [connect: example.com, port: 22, timeout: 5]

3. Analyze results
   [network_troubleshoot prompt with findings]
```

### Security Audit

**User:** *"Audit the security of my server at 192.168.1.100"*

**Claude with GoCat:**
```
I'll perform a comprehensive security audit:

1. Port scan (all ports)
   [scan_ports: 192.168.1.100, ports: 1-65535]

2. Check SSL/TLS
   [connect: 192.168.1.100, port: 443, ssl: true]

3. Analyze open services
   [security_audit prompt with findings]

4. Provide recommendations
```

### Proxy Setup

**User:** *"Set up a load balancer for 3 web servers"*

**Claude with GoCat:**
```
I'll set up a reverse proxy with load balancing:

[start_proxy:
  listen_port: 8080,
  backends: ["http://server1:80", "http://server2:80", "http://server3:80"],
  algorithm: "round-robin",
  health_check: "/health"
]

Your load balancer is running on :8080
```

---

## 🔧 Manual Testing

Test MCP server manually:

### 1. Initialize

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | gocat mcp
```

### 2. List Tools

```bash
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | gocat mcp
```

### 3. Call Tool

```bash
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"scan_ports","arguments":{"host":"example.com","ports":"80,443"}}}' | gocat mcp
```

### 4. Read Resource

```bash
echo '{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"gocat://system/info"}}' | gocat mcp
```

### 5. Get Prompt

```bash
echo '{"jsonrpc":"2.0","id":5,"method":"prompts/get","params":{"name":"network_troubleshoot","arguments":{"target":"example.com"}}}' | gocat mcp
```

---

## 🏗️ Architecture

```
┌─────────────────┐
│  Claude / AI    │
│   Assistant     │
└────────┬────────┘
         │ MCP Protocol
         │ (JSON-RPC over stdio)
         ▼
┌─────────────────┐
│  GoCat MCP      │
│    Server       │
├─────────────────┤
│ • Tools         │◄─── 20+ network tools
│ • Resources     │◄─── 10+ info sources
│ • Prompts       │◄─── 8+ guides
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  GoCat Core     │
│  Functionality  │
└─────────────────┘
```

---

## 🔒 Security Considerations

### Safe by Default
- MCP server runs with **same permissions** as user
- No privileged operations by default
- All commands **explicitly authorized** by user

### Best Practices
1. **Review AI actions** before approval
2. **Limit network access** if needed
3. **Monitor logs** for unusual activity
4. **Use firewall rules** for production

### Isolation
Consider running in container:
```bash
docker run -it gocat mcp
```

---

## 📖 Integration Examples

### With Other MCP Servers

GoCat can work alongside other MCP servers:

```json
{
  "mcpServers": {
    "gocat": {
      "command": "gocat",
      "args": ["mcp"]
    },
    "filesystem": {
      "command": "mcp-server-filesystem",
      "args": ["/path/to/allowed/files"]
    },
    "postgres": {
      "command": "mcp-server-postgres",
      "args": ["postgresql://localhost/mydb"]
    }
  }
}
```

Now AI can:
- Use **GoCat** for network operations
- Use **filesystem** for file operations
- Use **postgres** for database queries

---

## 🎓 Use Cases

### DevOps & SRE
- **"Check if all services are running"** → Port scans
- **"Set up load balancer"** → Proxy configuration
- **"Monitor network metrics"** → Prometheus metrics

### Security Engineering
- **"Audit this server"** → Security assessment
- **"Scan for open ports"** → Port enumeration
- **"Test SSL configuration"** → TLS verification

### Network Troubleshooting
- **"Why can't I connect?"** → Diagnostic workflow
- **"Is the firewall blocking?"** → Connection tests
- **"Check network latency"** → Performance tests

### Development
- **"Create WebSocket server"** → Quick WS setup
- **"Forward port through SSH"** → Tunnel creation
- **"Test my API"** → HTTP connectivity

---

## 🐛 Troubleshooting

### MCP Server Not Starting

```bash
# Check GoCat installation
which gocat

# Test manually
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | gocat mcp

# Check logs
gocat mcp --debug
```

### Claude Can't Find Tools

1. Restart Claude Desktop
2. Check config file path
3. Verify `gocat` in PATH
4. Check Claude logs

### Tool Execution Fails

- Check permissions
- Verify network access
- Review error messages
- Test tool manually: `gocat scan example.com --ports 80`

---

## 🚀 Advanced Configuration

### Custom Server Name

```bash
gocat mcp --name "my-gocat" --version "1.0.0"
```

### With Logging

```bash
gocat mcp --debug 2> gocat-mcp.log
```

### In Container

```dockerfile
FROM golang:1.21-alpine
COPY gocat /usr/local/bin/
ENTRYPOINT ["gocat", "mcp"]
```

---

## 📚 Resources

- **MCP Specification**: https://spec.modelcontextprotocol.io/
- **GoCat Documentation**: `gocat --help`
- **MCP Servers**: https://github.com/modelcontextprotocol/servers

---

## 🤝 Contributing

Add new tools to `internal/mcp/tools.go`:

```go
server.RegisterTool(&Tool{
    Name:        "my_tool",
    Description: "My awesome network tool",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param": map[string]interface{}{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param"},
    },
    Handler: handleMyTool,
})
```

---

## 📄 License

Same as GoCat main project.

---

## 🎉 Get Started

```bash
# 1. Build GoCat with MCP support
go build -o gocat

# 2. Configure Claude Desktop
# Edit claude_desktop_config.json

# 3. Restart Claude

# 4. Ask Claude:
# "Use GoCat to scan example.com ports 1-1000"
```

**Happy AI-powered networking! 🚀🤖**
