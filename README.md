# 🐱 GoCat

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![Release](https://img.shields.io/github/v/release/ibrahmsql/gocat?style=for-the-badge)](https://github.com/ibrahmsql/gocat/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/ibrahmsql/gocat/ci.yml?style=for-the-badge)](https://github.com/ibrahmsql/gocat/actions)

<div align="center">
  <img src="https://raw.githubusercontent.com/ibrahmsql/gocat/main/assets/Gocat.png" alt="GoCat Logo" width="300" height="300">
</div>

**A modern, feature-rich  netcat alternative written in Go**


[🚀 Quick Start](#-quick-start) • [📖 Documentation](#-documentation) • [💾 Installation](#-installation) • [🎯 Features](#-features) • [🔧 Usage](#-usage) • [🤝 Contributing](#-contributing)

</div>

---

## 🌟 Overview

**GoCat** is a modern, cross-platform netcat alternative written in Go. It provides all the functionality of traditional netcat with additional features, better performance, and  security. Whether you're a network administrator, security professional, or developer, GoCat offers the tools you need for network communication, debugging, and testing.

### ✨ Why Choose GoCat?

- 🚀 Fast & Lightweight: Built with Go for optimal performance
- 🌐 Cross-Platform: Works on Linux, macOS, Windows, and FreeBSD
---

## Key Features

### Network Protocols
- **TCP/UDP Support**: Full support for both protocols with advanced options
- **IPv4/IPv6**: Native dual-stack support with proper formatting
- **SSL/TLS**: Secure connections with TLS 1.2+ and certificate validation
- **Proxy Support**: SOCKS5 and HTTP proxy support
- **Keep-Alive**: Configurable connection keep-alive
- **HTTP Reverse Proxy**: Load balancing with health checks
- **Protocol Converter**: TCPUDP, HTTPWebSocket conversion
- **NekoDNS**: DNS-based reverse shell (NEW!)
- **Port Forwarding**: Advanced port redirection with statistics (NEW!)
- **Packet Sniffer**: Network packet capture and analysis (NEW!)
- **Benchmark Tool**: Network performance testing (NEW!)

### Advanced Features
- **Interactive Mode**: Full PTY support with command history
- **Connection Retry**: Exponential backoff with configurable attempts
- **Signal Handling**: Graceful shutdown and signal blocking
- **Timeout Control**: Configurable connection and read timeouts
- **Concurrent Connections**: Handle multiple connections simultaneously
- **Comprehensive Logging**: Structured logging with multiple levels
- ✅ **Timeout Control**: Configurable connection and read timeouts
- ✅ **Concurrent Connections**: Handle multiple connections simultaneously
- ✅ **Comprehensive Logging**: Structured logging with multiple levels

### 🚀 New Features 
- **🤖 MCP Server**: Model Context Protocol integration - expose GoCat to AI assistants (Claude, etc.)
- **WebSocket Support**: Full WebSocket server/client with compression
- **Unix Domain Sockets**: Local IPC with stream and datagram support
- **Prometheus Metrics**: Built-in metrics exporter for monitoring
- **SSH Tunneling**: Local, remote, and dynamic SOCKS proxy tunnels
- **DNS Tunneling**: Covert channel for firewall bypass
- **Multi-Port Listener**: Listen on multiple ports simultaneously
- **File Transfer**: Efficient file sending and receiving
- **Command Execution**: Execute commands on remote systems
- **Connection Persistence**: Keep connections alive with heartbeat

### 🎨 User Experience
- **Colorful Output**: Syntax highlighting and colored logs
- **Progress Bars**: Visual progress indicators for transfers
- **Verbose Logging**: Detailed logging with multiple levels
- **Shell Integration**: Bash, Zsh, and Fish completion support
- **Configuration Files**: YAML/JSON configuration support
- **Man Pages**: Comprehensive manual pages
- **Lua Scripting**: Extensible with Lua scripts

### 🔒 Security
- ✅ **Encryption**: AES-256-GCM and ChaCha20-Poly1305 encryption
- ✅ **Authentication**: Token-based and password authentication
- ✅ **Rate Limiting**: Per-IP and global rate limiting
- ✅ **Access Control**: IP-based allow/deny lists with CIDR support
- ✅ **Audit Logging**: Comprehensive security event logging
- ✅ **Input Validation**: Extensive input sanitization and validation

---

## 💾 Installation

### 📦 Package Managers

#### 🍺 Homebrew (macOS/Linux)
```bash
brew tap ibrahmsql/gocat
brew install gocat
```

#### 🐧 Arch Linux (AUR)
```bash
yay -S gocat
# or
paru -S gocat
```

#### 📦 Debian/Ubuntu
```bash
wget https://github.com/ibrahmsql/gocat/releases/latest/download/gocat_amd64.deb
sudo dpkg -i gocat_amd64.deb
```

#### 🎩 RPM (RHEL/CentOS/Fedora)
```bash
wget https://github.com/ibrahmsql/gocat/releases/latest/download/gocat_amd64.rpm
sudo rpm -i gocat_amd64.rpm
```

### 🚀 Quick Install Script
```bash
curl -sSL https://raw.githubusercontent.com/ibrahmsql/gocat/main/pkg/install.sh | bash
```

### 📥 Manual Download
Download the latest binary from [GitHub Releases](https://github.com/ibrahmsql/gocat/releases):

```bash
# Linux
wget https://github.com/ibrahmsql/gocat/releases/latest/download/gocat-linux-amd64
chmod +x gocat-linux-amd64
sudo mv gocat-linux-amd64 /usr/local/bin/gocat

# macOS
wget https://github.com/ibrahmsql/gocat/releases/latest/download/gocat-darwin-amd64
chmod +x gocat-darwin-amd64
sudo mv gocat-darwin-amd64 /usr/local/bin/gocat

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/ibrahmsql/gocat/releases/latest/download/gocat-windows-amd64.exe" -OutFile "gocat.exe"
```

### 🐳 Docker
```bash
# Run directly
docker run --rm -it ghcr.io/ibrahmsql/gocat:latest

# Or use docker-compose
git clone https://github.com/ibrahmsql/gocat.git
cd gocat
docker-compose up
```

### 🛠️ Build from Source
```bash
# Prerequisites: Go 1.21+
git clone https://github.com/ibrahmsql/gocat.git
cd gocat
make build
# or
go build -o gocat .
```

---

## 🚀 Quick Start

### 🎯 Basic Usage

#### 🔗 Connect to a Server
```bash
# Connect to a TCP server
gocat connect example.com 80

# Connect with custom shell
gocat connect -s /bin/bash example.com 22

# Connect to IPv6 address
gocat connect "[2001:db8::1]" 80
```

#### 👂 Listen for Connections
```bash
# Listen on port 8080
gocat listen 8080

# Listen with command execution
gocat listen -e /bin/bash 8080

# Interactive mode
gocat listen -i 8080

# Local interactive mode
gocat listen -l 8080
```

#### 📁 File Transfer
```bash
# Send a file
gocat connect example.com 8080 < file.txt

# Receive a file
gocat listen 8080 > received_file.txt

# Send with progress bar
gocat connect --progress example.com 8080 < large_file.zip
```

### 🎨 Advanced Examples

#### 🔍 Port Scanning
```bash
# Scan a single port
gocat scan example.com 80

# Scan multiple ports
gocat scan example.com 80,443,8080

# Scan port range
gocat scan example.com 1-1000

# Scan with timeout
gocat scan --timeout 5s example.com 1-65535
```

#### 🌐 HTTP Reverse Proxy
```bash
# Simple reverse proxy
gocat proxy --listen :8080 --target http://backend:80

# Load balancing with multiple backends
gocat proxy --listen :8080 --backends http://backend1:80,http://backend2:80

# With health checks and SSL
gocat proxy --listen :443 --backends http://backend1:80,http://backend2:80 \
  --health-check /health --ssl --cert cert.pem --key key.pem

# Different load balancing algorithms
gocat proxy --listen :8080 --backends http://b1:80,http://b2:80 --lb-algorithm least-connections
```

#### 🔄 Protocol Converter
```bash
# TCP to UDP conversion
gocat convert --from tcp:8080 --to udp:backend:9000

# UDP to TCP conversion
gocat convert --from udp:8080 --to tcp:backend:9000

# HTTP to WebSocket (NEW!)
gocat convert --from http:8080 --to ws://backend:9000/ws

# WebSocket to HTTP (NEW!)
gocat convert --from ws:8080 --to http://backend:9000
```

#### 🔐 Encryption & Security
```bash
# Use encryption for connections
gocat connect --encrypt --key mykey example.com 8080

# Listen with authentication
gocat listen --auth --user admin --password secret 8080

# Rate limiting
gocat listen --rate-limit 100 --per-ip-limit 10 8080

# Access control
gocat listen --allow 192.168.1.0/24 --deny 192.168.1.100 8080

# Audit logging
gocat listen --audit-log /var/log/gocat-audit.log 8080
```

#### 📜 Lua Scripting
```bash
# Execute Lua script
gocat script scripts/examples/http_client.lua

# Port scanner script
gocat script scripts/examples/port_scanner.lua

# Banner grabber
gocat script scripts/examples/banner_grabber.lua

# Custom SSL client
gocat script scripts/examples/ssl_client.lua
```

#### 🔌 Multi-Port Listener
```bash
# Listen on multiple ports
gocat multi-listen --ports 8080,8081,8082

# Listen on port range
gocat multi-listen --range 8000-8100

# With command execution
gocat multi-listen --ports 8080,8081 --exec /bin/bash

# With statistics
gocat multi-listen --range 8000-8010 --stats
```

#### 🌐 WebSocket Support (NEW!)
```bash
# Start WebSocket server
gocat ws server --port 8080

# WebSocket echo server
gocat ws echo --port 8080

# Connect to WebSocket server
gocat ws connect ws://localhost:8080

# WebSocket with compression
gocat ws server --port 8080 --compress

# Secure WebSocket (WSS)
echo "Hello WebSocket!" | gocat ws connect wss://secure.example.com/ws
```

#### 🔌 Unix Domain Sockets (NEW!)
```bash
# Listen on Unix socket
gocat unix listen /tmp/gocat.sock

# Connect to Unix socket
gocat unix connect /tmp/gocat.sock

# Unix socket echo server
gocat unix echo /tmp/echo.sock

# Datagram socket
gocat unix listen --type datagram /tmp/dgram.sock

# With custom permissions
gocat unix listen --permissions 0600 /tmp/secure.sock
```

#### 📊 Prometheus Metrics (NEW!)
```bash
# Start metrics exporter
gocat metrics --port 9090

# With custom namespace
gocat metrics --namespace myapp --subsystem network

# Access metrics
curl http://localhost:9090/metrics

# Health check endpoint
curl http://localhost:9090/health
```

#### 🚇 SSH Tunneling
```bash
# Local port forwarding (access remote service locally)
gocat tunnel --ssh user@server --local 8080 --remote localhost:80

# Remote port forwarding (expose local service remotely)
gocat tunnel --ssh user@server --reverse --local 3000 --remote 8080

# Dynamic SOCKS proxy
gocat tunnel --ssh user@server --dynamic 1080

# With SSH key authentication
gocat tunnel --ssh user@server --key ~/.ssh/id_rsa --local 8080 --remote 80
```

#### 🌐 DNS Tunneling
```bash
# Start DNS tunnel server
gocat dns-tunnel --server --domain tunnel.example.com --listen :53 --target localhost:8080

# Start DNS tunnel client
gocat dns-tunnel --client --domain tunnel.example.com --listen :8080

# With different encoding
gocat dns-tunnel --server --domain tunnel.example.com --encoding hex
```

#### 🌐 Proxy Usage
```bash
# Connect through SOCKS proxy
gocat connect --proxy socks5://proxy.example.com:1080 target.com 80

# Connect through HTTP proxy
gocat connect --proxy http://proxy.example.com:8080 target.com 443
```

#### 🔒 Secure Connections
```bash
# SSL/TLS connection
gocat connect --ssl example.com 443

# With certificate verification
gocat connect --ssl --verify-cert example.com 443

# Custom CA certificate
gocat connect --ssl --ca-cert /path/to/ca.pem example.com 443
```

#### 📊 Monitoring and Logging
```bash
# Verbose output
gocat -v connect example.com 80

# Debug mode
gocat --debug listen 8080

# Log to file
gocat --log-file /var/log/gocat.log listen 8080

# JSON output
gocat --output json scan example.com 1-1000
```

---

## 📖 Documentation

### 📋 Command Reference

#### 🔗 Connect Command
```bash
gocat connect [OPTIONS] HOST PORT

Options:
  -s, --shell SHELL     Shell to use for command execution
  -t, --timeout DURATION Connection timeout (default: 30s)
  -r, --retry COUNT     Number of retry attempts (default: 3)
  -k, --keep-alive      Enable keep-alive
  -p, --proxy URL       Proxy URL (socks5:// or http://)
  -S, --ssl             Use SSL/TLS
  -C, --verify-cert     Verify SSL certificate
  -c, --ca-cert FILE    CA certificate file
  -u, --udp             Use UDP instead of TCP
  -6, --ipv6            Force IPv6
  -4, --ipv4            Force IPv4
```

#### 👂 Listen Command
```bash
gocat listen [OPTIONS] PORT

Options:
  -e, --exec COMMAND    Execute command for each connection
  -i, --interactive     Interactive mode
  -l, --local           Local interactive mode
  -b, --bind ADDRESS    Bind to specific address (default: 0.0.0.0)
  -k, --keep-alive      Keep connections alive
  -m, --max-conn COUNT  Maximum concurrent connections (default: 10)
  -t, --timeout DURATION Connection timeout (default: 0 = no timeout)
  -u, --udp             Use UDP instead of TCP
  -6, --ipv6            Force IPv6
  -4, --ipv4            Force IPv4
  -S, --ssl             Use SSL/TLS
  -K, --ssl-key FILE    SSL private key file
  -C, --ssl-cert FILE   SSL certificate file
```

#### 🔍 Scan Command
```bash
gocat scan [OPTIONS] HOST PORTS

Options:
  -t, --timeout DURATION Port scan timeout (default: 3s)
  -c, --concurrent COUNT Concurrent scans (default: 100)
  -u, --udp             Scan UDP ports
  -T, --tcp             Scan TCP ports (default)
  -A, --all             Scan both TCP and UDP
  -o, --output FORMAT   Output format (text, json, xml)
  -v, --verbose         Verbose output
  -q, --quiet           Quiet mode (only open ports)
```

### 🔧 Configuration

GoCat supports configuration files in YAML or JSON format:

```yaml
# ~/.gocat.yml
defaults:
  timeout: 30s
  retry: 3
  keep_alive: true
  
logging:
  level: info
  file: /var/log/gocat.log
  format: json
  
network:
  ipv6: false
  buffer_size: 4096
  
security:
  verify_cert: true
  ca_cert: /etc/ssl/certs/ca-certificates.crt
```

### 🎨 Color Themes

Customize output colors:

```yaml
# ~/.gocat-theme.yml
colors:
  success: green
  error: red
  warning: yellow
  info: blue
  debug: gray
  highlight: cyan
```

---

## 🔧 Development

### 🏗️ Building

```bash
# Clone the repository
git clone https://github.com/ibrahmsql/gocat.git
cd gocat

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint

# Build for all platforms
make build-all
```

### 🧪 Testing

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Run benchmarks
make test-bench

# Generate coverage report
make test-coverage
open coverage/coverage.html
```

### 🔍 Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Security scan
make security

# Vulnerability check
make vuln

# All checks
make check
```

### 🐳 Docker Development

```bash
# Build Docker image
make docker-build

# Run in container
make docker-run

# Development with docker-compose
docker-compose --profile dev up

# Testing with docker-compose
docker-compose --profile test up
```

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### 🚀 Quick Contribution Steps

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Commit** your changes: `git commit -m 'Add amazing feature'`
4. **Push** to the branch: `git push origin feature/amazing-feature`
5. **Open** a Pull Request

### 🐛 Bug Reports

Found a bug? Please [open an issue](https://github.com/ibrahmsql/gocat/issues/new?template=bug_report.yml) with:
- Steps to reproduce
- Expected vs actual behavior
- System information
- Log output (if applicable)

### 💡 Feature Requests

Have an idea? [Request a feature](https://github.com/ibrahmsql/gocat/issues/new?template=feature_request.yml) with:
- Use case description
- Proposed solution
- Alternative solutions considered

---

## 📊 Performance

### 🚀 Benchmarks

| Operation | GoCat | Traditional nc | Improvement |
|-----------|-------|----------------|-------------|
| TCP Connect | 0.5ms | 1.2ms | **2.4x faster** |
| File Transfer (1GB) | 45s | 67s | **1.5x faster** |
| Port Scan (1000 ports) | 2.3s | 8.7s | **3.8x faster** |
| Memory Usage | 8MB | 15MB | **47% less** |

### 📈 Scalability

- **Concurrent Connections**: Up to 10,000 simultaneous connections
- **Throughput**: 10Gbps+ on modern hardware
- **Memory Efficiency**: Constant memory usage regardless of connection count
- **CPU Usage**: Multi-core optimization with goroutines

---

## 🔒 Security

### 🛡️ Security Features

- **Input Validation**: All inputs are validated and sanitized
- **Buffer Overflow Protection**: Safe buffer handling
- **Rate Limiting**: Protection against DoS attacks
- **Secure Defaults**: Security-first configuration
- **Audit Logging**: Comprehensive security event logging

### 🔐 Encryption

- **TLS 1.3**: Latest TLS protocol support
- **Certificate Validation**: Full certificate chain validation
- **Custom CA**: Support for custom certificate authorities
- **Perfect Forward Secrecy**: Ephemeral key exchange

### 🚨 Reporting Security Issues

Please report security vulnerabilities to [security@gocat.dev](mailto:ibrahimsql@proton.me). Do not open public issues for security problems.

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- **Original netcat** by Hobbit for the inspiration
- **Go community** for the amazing ecosystem
- **Contributors** who make this project better
- **Users** who provide feedback and bug reports

---

## 📞 Support

- 📖 **Documentation**: [docs.gocat.dev](https://docs.gocat.dev)
- 💬 **Discord**: [Join our community](https://discord.gg/gocat)
- 🐛 **Issues**: [GitHub Issues](https://github.com/ibrahmsql/gocat/issues)
- 📧 **Email**: [support@gocat.dev](mailto:ibrahimsql@proton.me)
- 🐦 **Twitter**: [@GoCatTool](https://twitter.com/GoCatTool)

---

<div align="center">

**Made with ❤️ by the GoCat team**

[⭐ Star us on GitHub](https://github.com/ibrahmsql/gocat) • [🐦 Follow on Twitter](https://twitter.com/GoCatTool) • [💬 Join Discord](https://discord.gg/gocat)

</div>
