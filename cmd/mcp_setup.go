package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpSetupList   bool
	mcpSetupRemove bool
	mcpSetupClient string
)

// mcpSetupCmd represents the mcp setup command
var mcpSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup GoCat MCP server for AI clients",
	Long: `Automatically configure GoCat MCP server for AI clients like Claude Desktop, Cursor, etc.

This command will:
  • Detect installed AI clients (Claude, Cursor, Continue, Zed, Windsurf)
  • Show current configuration status
  • Add GoCat MCP server to selected client's configuration
  • Verify the setup

Examples:
  # Interactive setup (shows menu)
  gocat mcp setup

  # List detected clients and status
  gocat mcp setup --list

  # Setup for specific client
  gocat mcp setup --client claude

  # Remove from specific client
  gocat mcp setup --client claude --remove

  # Setup for all detected clients
  gocat mcp setup --client all`,
	Run: runMCPSetup,
}

func init() {
	mcpCmd.AddCommand(mcpSetupCmd)

	mcpSetupCmd.Flags().BoolVar(&mcpSetupList, "list", false, "List detected clients and status")
	mcpSetupCmd.Flags().BoolVar(&mcpSetupRemove, "remove", false, "Remove GoCat from client configuration")
	mcpSetupCmd.Flags().StringVar(&mcpSetupClient, "client", "", "Specific client to configure (claude, cursor, continue, zed, windsurf, all)")
}

func runMCPSetup(cmd *cobra.Command, args []string) {
	logger.Info("🔧 GoCat MCP Setup")
	logger.Info("")

	// Get GoCat executable path
	gocatPath, err := mcp.GetGoCatPath()
	if err != nil {
		logger.Error("Failed to determine GoCat path: %v", err)
		logger.Info("Using 'gocat' as command (make sure it's in PATH)")
		gocatPath = "gocat"
	} else {
		logger.Debug("GoCat path: %s", gocatPath)
	}

	// Detect clients
	clients := mcp.DetectMCPClients()

	if len(clients) == 0 {
		logger.Warn("No MCP clients detected on your system.")
		logger.Info("")
		logger.Info("Supported clients:")
		logger.Info("  • Claude Desktop")
		logger.Info("  • Cursor")
		logger.Info("  • Continue")
		logger.Info("  • Zed Editor")
		logger.Info("  • Windsurf")
		logger.Info("")
		logger.Info("Install one of these clients to use GoCat's MCP features.")
		return
	}

	// Handle --list flag
	if mcpSetupList {
		showDetailedStatus(clients, gocatPath)
		return
	}

	// Handle --remove flag
	if mcpSetupRemove {
		if mcpSetupClient == "" {
			logger.Error("Please specify --client when using --remove")
			return
		}
		removeFromClients(clients, mcpSetupClient)
		return
	}

	// Handle --client flag
	if mcpSetupClient != "" {
		if mcpSetupClient == "all" {
			setupAllClients(clients, gocatPath)
		} else {
			setupSpecificClient(clients, mcpSetupClient, gocatPath)
		}
		return
	}

	// Interactive mode
	runInteractiveSetup(clients, gocatPath)
}

func showDetailedStatus(clients []mcp.MCPClient, gocatPath string) {
	fmt.Println()
	fmt.Println("📊 MCP Client Status")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	for i, client := range clients {
		fmt.Printf("%d. %s\n", i+1, client.Name)
		fmt.Printf("   ├─ Config: %s\n", client.ConfigPath)
		
		status := checkClientStatus(client)
		fmt.Printf("   ├─ Status: %s\n", status)
		
		if !client.Installed {
			fmt.Printf("   └─ ⚠️  Application not detected\n")
		} else {
			fmt.Printf("   └─ ✓ Application detected\n")
		}
		fmt.Println()
	}

	fmt.Println("Command to use:", gocatPath, "mcp")
	fmt.Println()
}

func checkClientStatus(client mcp.MCPClient) string {
	if !mcp.FileExists(client.ConfigPath) {
		return "❌ Not configured"
	}

	data, err := os.ReadFile(client.ConfigPath)
	if err != nil {
		return "❌ Cannot read config"
	}

	var config mcp.ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "❌ Invalid config"
	}

	if _, exists := config.MCPServers["gocat"]; exists {
		return "✅ Configured"
	}

	return "❌ Not configured"
}

func runInteractiveSetup(clients []mcp.MCPClient, gocatPath string) {
	fmt.Println()
	fmt.Println("🤖 Interactive MCP Setup")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Select a client to configure:")
	fmt.Println()

	// Show menu
	for i, client := range clients {
		status := checkClientStatus(client)
		installedMark := ""
		if !client.Installed {
			installedMark = " ⚠️"
		}
		fmt.Printf("  %d. %s %s%s\n", i+1, client.Name, status, installedMark)
	}
	fmt.Printf("  %d. Configure all clients\n", len(clients)+1)
	fmt.Printf("  0. Exit\n")
	fmt.Println()

	// Read choice
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your choice: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil {
		logger.Error("Invalid input")
		return
	}

	if choice == 0 {
		logger.Info("Setup cancelled")
		return
	}

	if choice == len(clients)+1 {
		// Configure all
		fmt.Println()
		setupAllClients(clients, gocatPath)
		return
	}

	if choice < 1 || choice > len(clients) {
		logger.Error("Invalid choice")
		return
	}

	// Configure selected client
	selectedClient := clients[choice-1]
	fmt.Println()
	logger.Info("Configuring %s...", selectedClient.Name)
	fmt.Println()

	if err := mcp.AddToClient(selectedClient, gocatPath); err != nil {
		logger.Error("Failed to configure: %v", err)
		return
	}

	fmt.Println()
	logger.Info("✅ Setup completed!")
	fmt.Println()
	showPostSetupInstructions(selectedClient)
}

func setupAllClients(clients []mcp.MCPClient, gocatPath string) {
	logger.Info("Configuring all detected clients...")
	fmt.Println()

	successCount := 0
	failCount := 0

	for _, client := range clients {
		logger.Info("→ %s", client.Name)
		
		if err := mcp.AddToClient(client, gocatPath); err != nil {
			logger.Error("  Failed: %v", err)
			failCount++
		} else {
			successCount++
		}
		fmt.Println()
	}

	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("Summary: %d configured, %d failed", successCount, failCount)
	
	if successCount > 0 {
		fmt.Println()
		logger.Info("✅ Setup completed!")
		fmt.Println()
		logger.Info("Next steps:")
		logger.Info("  1. Restart your AI client(s)")
		logger.Info("  2. Open a conversation")
		logger.Info("  3. Ask: 'Can you scan example.com ports 80,443?'")
	}
}

func setupSpecificClient(clients []mcp.MCPClient, clientName string, gocatPath string) {
	// Find client by name (case insensitive)
	clientName = strings.ToLower(clientName)
	var selectedClient *mcp.MCPClient

	for i := range clients {
		if strings.Contains(strings.ToLower(clients[i].Name), clientName) {
			selectedClient = &clients[i]
			break
		}
	}

	if selectedClient == nil {
		logger.Error("Client not found: %s", clientName)
		logger.Info("")
		logger.Info("Available clients:")
		for _, c := range clients {
			logger.Info("  • %s", strings.ToLower(strings.Fields(c.Name)[0]))
		}
		return
	}

	logger.Info("Configuring %s...", selectedClient.Name)
	fmt.Println()

	if err := mcp.AddToClient(*selectedClient, gocatPath); err != nil {
		logger.Error("Failed: %v", err)
		return
	}

	fmt.Println()
	logger.Info("✅ Setup completed!")
	fmt.Println()
	showPostSetupInstructions(*selectedClient)
}

func removeFromClients(clients []mcp.MCPClient, clientName string) {
	if clientName == "all" {
		logger.Info("Removing GoCat from all clients...")
		fmt.Println()

		for _, client := range clients {
			logger.Info("→ %s", client.Name)
			
			if err := mcp.RemoveFromClient(client); err != nil {
				logger.Error("  Failed: %v", err)
			}
			fmt.Println()
		}

		logger.Info("✅ Removal completed!")
		return
	}

	// Find and remove from specific client
	clientName = strings.ToLower(clientName)
	var selectedClient *mcp.MCPClient

	for i := range clients {
		if strings.Contains(strings.ToLower(clients[i].Name), clientName) {
			selectedClient = &clients[i]
			break
		}
	}

	if selectedClient == nil {
		logger.Error("Client not found: %s", clientName)
		return
	}

	logger.Info("Removing GoCat from %s...", selectedClient.Name)
	fmt.Println()

	if err := mcp.RemoveFromClient(*selectedClient); err != nil {
		logger.Error("Failed: %v", err)
		return
	}

	logger.Info("✅ Removal completed!")
}

func showPostSetupInstructions(client mcp.MCPClient) {
	logger.Info("📝 Next Steps:")
	logger.Info("")
	logger.Info("  1. Restart %s", client.Name)
	logger.Info("  2. Open a new conversation/chat")
	logger.Info("  3. Try these commands:")
	logger.Info("")
	logger.Info("     \"Can you scan example.com ports 80,443?\"")
	logger.Info("     \"Set up a reverse proxy for 3 backends\"")
	logger.Info("     \"Check if my server is reachable\"")
	logger.Info("     \"Create a WebSocket echo server\"")
	logger.Info("")
	logger.Info("  Config location: %s", client.ConfigPath)
	logger.Info("")
}
