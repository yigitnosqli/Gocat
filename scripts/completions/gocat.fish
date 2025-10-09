# Fish shell completion for GoCat
# Save this to ~/.config/fish/completions/gocat.fish

# Main commands
complete -c gocat -f -n __fish_use_subcommand -a connect -d "Connect to remote host"
complete -c gocat -f -n __fish_use_subcommand -a listen -d "Start listener for incoming connections"
complete -c gocat -f -n __fish_use_subcommand -a scan -d "Port scanner for network reconnaissance"
complete -c gocat -f -n __fish_use_subcommand -a transfer -d "File transfer operations"
complete -c gocat -f -n __fish_use_subcommand -a tunnel -d "Create SSH tunnels"
complete -c gocat -f -n __fish_use_subcommand -a dns-tunnel -d "DNS tunneling for firewall bypass"
complete -c gocat -f -n __fish_use_subcommand -a convert -d "Convert between network protocols"
complete -c gocat -f -n __fish_use_subcommand -a proxy -d "HTTP reverse proxy"
complete -c gocat -f -n __fish_use_subcommand -a multi-listen -d "Listen on multiple ports"
complete -c gocat -f -n __fish_use_subcommand -a broker -d "Connection broker mode"
complete -c gocat -f -n __fish_use_subcommand -a chat -d "Chat server mode"
complete -c gocat -f -n __fish_use_subcommand -a script -d "Lua script management"
complete -c gocat -f -n __fish_use_subcommand -a tui -d "Terminal user interface"
complete -c gocat -f -n __fish_use_subcommand -a version -d "Show version information"

# Global flags
complete -c gocat -l help -d "Show help"
complete -c gocat -s h -l help -d "Show help"
complete -c gocat -s v -l verbose -d "Verbose output"
complete -c gocat -l debug -d "Enable debug output"
complete -c gocat -l no-color -d "Disable colored output"
complete -c gocat -l config -r -d "Configuration file path"
complete -c gocat -l theme -r -d "Color theme file path"

# Network protocols
complete -c gocat -s 4 -l ipv4 -d "Use IPv4 only"
complete -c gocat -s 6 -l ipv6 -d "Use IPv6 only"
complete -c gocat -s u -l udp -d "Use UDP protocol"
complete -c gocat -l sctp -d "Use SCTP protocol"
complete -c gocat -l ssl -d "Use SSL/TLS"
complete -c gocat -l ssl-verify -d "Verify SSL certificate"
complete -c gocat -l ssl-cert -r -d "SSL certificate file"
complete -c gocat -l ssl-key -r -d "SSL private key file"

# Connection options
complete -c gocat -s l -l listen -d "Listen mode"
complete -c gocat -s k -l keep-open -d "Keep listening for connections"
complete -c gocat -s w -l wait -r -d "Connection timeout"
complete -c gocat -l proxy -r -d "Proxy URL (socks5:// or http://)"

# Connect command specific
complete -c gocat -n "__fish_seen_subcommand_from connect" -l shell -r -d "Shell to use"
complete -c gocat -n "__fish_seen_subcommand_from connect" -l connect-timeout -r -d "Connection timeout"
complete -c gocat -n "__fish_seen_subcommand_from connect" -l retry -r -d "Retry attempts"
complete -c gocat -n "__fish_seen_subcommand_from connect" -l connect-ssl -d "Use SSL/TLS"

# Listen command specific
complete -c gocat -n "__fish_seen_subcommand_from listen" -l interactive -d "Interactive mode"
complete -c gocat -n "__fish_seen_subcommand_from listen" -l local -d "Local interactive mode"
complete -c gocat -n "__fish_seen_subcommand_from listen" -l listen-exec -r -d "Execute command"
complete -c gocat -n "__fish_seen_subcommand_from listen" -l bind -r -d "Bind address"
complete -c gocat -n "__fish_seen_subcommand_from listen" -l listen-max-conn -r -d "Max connections"

# Scan command specific
complete -c gocat -n "__fish_seen_subcommand_from scan" -l ports -r -d "Port range (e.g., 1-1000)"
complete -c gocat -n "__fish_seen_subcommand_from scan" -l scan-timeout -r -d "Scan timeout"
complete -c gocat -n "__fish_seen_subcommand_from scan" -l concurrency -r -d "Concurrent scans"
complete -c gocat -n "__fish_seen_subcommand_from scan" -l open -d "Show only open ports"

# Transfer command specific
complete -c gocat -n "__fish_seen_subcommand_from transfer" -a "send receive" -d "Transfer mode"
complete -c gocat -n "__fish_seen_subcommand_from transfer" -s f -l file -r -d "File to transfer"
complete -c gocat -n "__fish_seen_subcommand_from transfer" -s o -l output -r -d "Output file"
complete -c gocat -n "__fish_seen_subcommand_from transfer" -l progress -d "Show progress"
complete -c gocat -n "__fish_seen_subcommand_from transfer" -l checksum -d "Verify checksum"
complete -c gocat -n "__fish_seen_subcommand_from transfer" -l compress -d "Compress data"

# Tunnel command specific
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l ssh -r -d "SSH server (user@host:port)"
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l local -r -d "Local address:port"
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l remote -r -d "Remote address:port"
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l reverse -d "Reverse tunnel"
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l dynamic -d "Dynamic SOCKS proxy"
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l key -r -d "SSH private key"
complete -c gocat -n "__fish_seen_subcommand_from tunnel" -l password -r -d "SSH password"

# DNS Tunnel command specific
complete -c gocat -n "__fish_seen_subcommand_from dns-tunnel" -l domain -r -d "Tunnel domain"
complete -c gocat -n "__fish_seen_subcommand_from dns-tunnel" -l server -d "Server mode"
complete -c gocat -n "__fish_seen_subcommand_from dns-tunnel" -l client -d "Client mode"
complete -c gocat -n "__fish_seen_subcommand_from dns-tunnel" -l listen -r -d "Listen address"
complete -c gocat -n "__fish_seen_subcommand_from dns-tunnel" -l target -r -d "Target address"

# Convert command specific
complete -c gocat -n "__fish_seen_subcommand_from convert" -l from -r -d "Source protocol:address"
complete -c gocat -n "__fish_seen_subcommand_from convert" -l to -r -d "Target protocol:address"
complete -c gocat -n "__fish_seen_subcommand_from convert" -l buffer -r -d "Buffer size"

# Proxy command specific
complete -c gocat -n "__fish_seen_subcommand_from proxy" -l listen -r -d "Listen address"
complete -c gocat -n "__fish_seen_subcommand_from proxy" -l target -r -d "Target backend URL"
complete -c gocat -n "__fish_seen_subcommand_from proxy" -l backends -r -d "Backend list"
complete -c gocat -n "__fish_seen_subcommand_from proxy" -l health-check -r -d "Health check path"
complete -c gocat -n "__fish_seen_subcommand_from proxy" -l lb-algorithm -r -d "Load balancing algorithm" -a "round-robin least-connections ip-hash"

# Multi-listen command specific
complete -c gocat -n "__fish_seen_subcommand_from multi-listen" -l ports -r -d "Port list"
complete -c gocat -n "__fish_seen_subcommand_from multi-listen" -l range -r -d "Port range"
complete -c gocat -n "__fish_seen_subcommand_from multi-listen" -l stats -d "Show statistics"

# Broker command specific
complete -c gocat -n "__fish_seen_subcommand_from broker" -s m -l max-conns -r -d "Max connections"

# Chat command specific
complete -c gocat -n "__fish_seen_subcommand_from chat" -s m -l max-conns -r -d "Max connections"
complete -c gocat -n "__fish_seen_subcommand_from chat" -s r -l room -r -d "Chat room name"

# Script command specific
complete -c gocat -n "__fish_seen_subcommand_from script" -a "run list info validate" -d "Script operation"
complete -c gocat -n "__fish_seen_subcommand_from script; and __fish_seen_subcommand_from run" -s a -l args -r -d "Script arguments"
complete -c gocat -n "__fish_seen_subcommand_from script; and __fish_seen_subcommand_from run" -l timeout -r -d "Execution timeout"

# TUI command specific
complete -c gocat -n "__fish_seen_subcommand_from tui" -a "connect listen chat broker scan help" -d "TUI mode"

# Common file completions
complete -c gocat -n "__fish_seen_subcommand_from transfer" -F
complete -c gocat -n "__fish_seen_subcommand_from script" -F -a "*.lua"
complete -c gocat -l config -F -a "*.yml *.yaml *.json"
complete -c gocat -l theme -F -a "*.yml *.yaml"
complete -c gocat -l ssl-cert -F -a "*.pem *.crt"
complete -c gocat -l ssl-key -F -a "*.pem *.key"
