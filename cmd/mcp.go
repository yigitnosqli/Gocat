package cmd

import (
	"os"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpServerName string
	mcpVersion    string
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start Model Context Protocol (MCP) server",
	Long: `Start an MCP server that exposes all GoCat functionality to AI assistants.

The MCP server implements the Model Context Protocol, allowing AI assistants like
Claude to use GoCat's network tools, read resources, and access prompts.

Features:
  • 20+ network tools (scan, connect, proxy, websocket, etc.)
  • Real-time resources (metrics, system info, network status)
  • Interactive prompts (troubleshooting, security audit, guides)
  • Full GoCat functionality accessible to AI

The server communicates via stdio (standard input/output) for seamless
integration with MCP clients.

Example:
  # Start MCP server
  gocat mcp

  # Use with Claude Desktop (add to config)
  {
    "mcpServers": {
      "gocat": {
        "command": "gocat",
        "args": ["mcp"]
      }
    }
  }

Available Tools:
  • scan_ports           - Port scanning
  • connect              - TCP/UDP/SSL connections
  • listen               - Server listening
  • send_file            - File transfer (send)
  • receive_file         - File transfer (receive)
  • start_proxy          - Reverse proxy with load balancing
  • convert_protocol     - Protocol conversion
  • websocket_server     - WebSocket server
  • websocket_connect    - WebSocket client
  • unix_socket_listen   - Unix domain socket server
  • unix_socket_connect  - Unix domain socket client
  • start_metrics_server - Prometheus metrics exporter
  • create_ssh_tunnel    - SSH tunneling
  • dns_lookup           - DNS resolution
  • dns_tunnel           - DNS tunneling
  • multi_port_listen    - Multi-port listening

Available Resources:
  • gocat://system/info           - System information
  • gocat://system/capabilities   - Available features
  • gocat://network/interfaces    - Network interfaces
  • gocat://network/connections   - Active connections
  • gocat://metrics/prometheus    - Prometheus metrics
  • gocat://scan/results/latest   - Latest scan results
  • gocat://docs/commands         - Command documentation
  • gocat://docs/examples         - Usage examples

Available Prompts:
  • network_troubleshoot   - Network diagnostics
  • security_audit         - Security assessment
  • port_scan_analysis     - Scan result analysis
  • connection_guide       - Connection setup guide
  • proxy_setup            - Proxy configuration guide
  • file_transfer_guide    - File transfer guide
  • monitoring_setup       - Monitoring setup guide
  • tunnel_guide           - SSH tunnel guide`,
	Run: runMCPServer,
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().StringVar(&mcpServerName, "name", "gocat", "MCP server name")
	mcpCmd.Flags().StringVar(&mcpVersion, "version", version, "MCP server version")
}

func runMCPServer(cmd *cobra.Command, args []string) {
	// Disable logger for MCP mode - only JSON should go to stdout
	// Logs will go to stderr if needed
	logger.SetOutput(os.Stderr)
	
	logger.Info("Starting GoCat MCP server: %s v%s", mcpServerName, mcpVersion)
	logger.Debug("MCP Protocol: stdio-based communication")

	// Create MCP server
	server := mcp.NewMCPServer(mcpServerName, mcpVersion)

	// Register all GoCat tools
	logger.Debug("Registering GoCat tools...")
	mcp.RegisterGoCatTools(server)

	// Register all GoCat resources
	logger.Debug("Registering GoCat resources...")
	mcp.RegisterGoCatResources(server)

	// Register all GoCat prompts
	logger.Debug("Registering GoCat prompts...")
	mcp.RegisterGoCatPrompts(server)

	logger.Debug("MCP server ready - waiting for requests on stdin...")
	logger.Debug("Tools: 20+ network operations")
	logger.Debug("Resources: 10+ information sources")
	logger.Debug("Prompts: 8+ interactive guides")

	// Start server on stdin/stdout
	if err := server.Start(os.Stdin, os.Stdout); err != nil {
		logger.Fatal("MCP server error: %v", err)
	}
}
