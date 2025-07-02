package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateHelp handles help mode input
func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.switchToMode(ModeMenu)
		return m, nil

	case "up", "k":
		// Scroll up in help content
		return m, nil

	case "down", "j":
		// Scroll down in help content
		return m, nil
	}

	return m, nil
}

// viewHelp renders the help interface
func (m Model) viewHelp() string {
	var content strings.Builder

	// Title
	title := HeaderStyle.Render("❓ GoCat Help & Documentation")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Overview
	overview := m.renderHelpOverview()
	content.WriteString(overview)
	content.WriteString("\n\n")

	// Module help
	moduleHelp := m.renderModuleHelp()
	content.WriteString(moduleHelp)
	content.WriteString("\n\n")

	// Keyboard shortcuts
	shortcuts := m.renderKeyboardShortcuts()
	content.WriteString(shortcuts)
	content.WriteString("\n\n")

	// Examples
	examples := m.renderExamples()
	content.WriteString(examples)

	return content.String()
}

// renderHelpOverview renders the help overview
func (m Model) renderHelpOverview() string {
	var overview strings.Builder

	overviewTitle := InfoStyle.Render("Overview:")
	overview.WriteString(overviewTitle)
	overview.WriteString("\n")

	description := `GoCat is a powerful network utility tool with a beautiful terminal interface.
It provides multiple networking capabilities including connection management,
listening for incoming connections, real-time chat, network brokering,
and comprehensive port scanning.`

	overview.WriteString(MutedStyle.Render(description))
	overview.WriteString("\n\n")

	features := []string{
		"🔗 Connect to remote hosts with multiple protocols",
		"👂 Listen for incoming connections",
		"💬 Real-time chat communication",
		"🔄 Network traffic brokering and proxying",
		"🔍 Comprehensive port and service scanning",
		"✨ Beautiful terminal UI with colors and styling",
		"🚀 Fast and efficient networking operations",
	}

	for _, feature := range features {
		overview.WriteString(InfoStyle.Render("  " + feature))
		overview.WriteString("\n")
	}

	return overview.String()
}

// renderModuleHelp renders help for each module
func (m Model) renderModuleHelp() string {
	var moduleHelp strings.Builder

	moduleTitle := InfoStyle.Render("Module Documentation:")
	moduleHelp.WriteString(moduleTitle)
	moduleHelp.WriteString("\n\n")

	// Connect module
	connectTitle := SuccessStyle.Render("🔗 Connect Module:")
	moduleHelp.WriteString(connectTitle)
	moduleHelp.WriteString("\n")
	connectDesc := `Establish connections to remote hosts using various protocols.
Supports TCP, UDP, HTTP, HTTPS, and SSH connections.
Configure host, port, and protocol before connecting.`
	moduleHelp.WriteString(MutedStyle.Render(connectDesc))
	moduleHelp.WriteString("\n\n")

	// Listen module
	listenTitle := SuccessStyle.Render("👂 Listen Module:")
	moduleHelp.WriteString(listenTitle)
	moduleHelp.WriteString("\n")
	listenDesc := `Listen for incoming connections on specified ports.
Monitor active connections and view real-time logs.
Supports TCP, UDP, and HTTP protocols.`
	moduleHelp.WriteString(MutedStyle.Render(listenDesc))
	moduleHelp.WriteString("\n\n")

	// Chat module
	chatTitle := SuccessStyle.Render("💬 Chat Module:")
	moduleHelp.WriteString(chatTitle)
	moduleHelp.WriteString("\n")
	chatDesc := `Real-time text communication over network connections.
Requires an active connection established via Connect module.
Supports message history and session statistics.`
	moduleHelp.WriteString(MutedStyle.Render(chatDesc))
	moduleHelp.WriteString("\n\n")

	// Broker module
	brokerTitle := SuccessStyle.Render("🔄 Broker Module:")
	moduleHelp.WriteString(brokerTitle)
	moduleHelp.WriteString("\n")
	brokerDesc := `Act as a network proxy/broker between clients and servers.
Supports TCP proxy, HTTP proxy, and SOCKS5 modes.
Monitor traffic statistics and active connections.`
	moduleHelp.WriteString(MutedStyle.Render(brokerDesc))
	moduleHelp.WriteString("\n\n")

	// Scan module
	scanTitle := SuccessStyle.Render("🔍 Scan Module:")
	moduleHelp.WriteString(scanTitle)
	moduleHelp.WriteString("\n")
	scanDesc := `Comprehensive network and port scanning capabilities.
Supports TCP Connect, SYN, UDP, and Service scans.
Detect open ports, services, and banners.`
	moduleHelp.WriteString(MutedStyle.Render(scanDesc))

	return moduleHelp.String()
}

// renderKeyboardShortcuts renders keyboard shortcuts
func (m Model) renderKeyboardShortcuts() string {
	var shortcuts strings.Builder

	shortcutsTitle := InfoStyle.Render("Keyboard Shortcuts:")
	shortcuts.WriteString(shortcutsTitle)
	shortcuts.WriteString("\n\n")

	// Global shortcuts
	globalTitle := WarningStyle.Render("Global Shortcuts:")
	shortcuts.WriteString(globalTitle)
	shortcuts.WriteString("\n")
	globalShortcuts := []string{
		"q, Ctrl+C: Quit application",
		"Esc: Return to main menu",
		"?: Show help (context-sensitive)",
		"↑/↓, j/k: Navigate menus and lists",
		"Enter, Space: Select/Activate",
		"Tab: Switch between input fields",
	}
	for _, shortcut := range globalShortcuts {
		shortcuts.WriteString(MutedStyle.Render("  " + shortcut))
		shortcuts.WriteString("\n")
	}
	shortcuts.WriteString("\n")

	// Module-specific shortcuts
	moduleTitle := WarningStyle.Render("Module-Specific Shortcuts:")
	shortcuts.WriteString(moduleTitle)
	shortcuts.WriteString("\n")
	moduleShortcuts := []string{
		"s: Start/Stop (Listen, Broker, Scan modules)",
		"c: Clear logs/results/connections",
		"e: Export results (Scan module)",
		"Enter: Send message (Chat module)",
		"Backspace: Delete character (input fields)",
		"Ctrl+L: Clear chat history (Chat module)",
	}
	for _, shortcut := range moduleShortcuts {
		shortcuts.WriteString(MutedStyle.Render("  " + shortcut))
		shortcuts.WriteString("\n")
	}

	return shortcuts.String()
}

// renderExamples renders usage examples
func (m Model) renderExamples() string {
	var examples strings.Builder

	examplesTitle := InfoStyle.Render("Usage Examples:")
	examples.WriteString(examplesTitle)
	examples.WriteString("\n\n")

	// Example scenarios
	exampleScenarios := []struct {
		title string
		steps []string
	}{
		{
			title: "Basic TCP Connection:",
			steps: []string{
				"1. Select 'Connect' from main menu",
				"2. Enter target host (e.g., localhost)",
				"3. Enter port (e.g., 8080)",
				"4. Select TCP protocol",
				"5. Press Enter to connect",
			},
		},
		{
			title: "Port Scanning:",
			steps: []string{
				"1. Select 'Scan' from main menu",
				"2. Enter target host (e.g., 192.168.1.1)",
				"3. Set port range (e.g., 1-1000)",
				"4. Choose scan type (TCP Connect)",
				"5. Press 's' to start scanning",
			},
		},
		{
			title: "Network Brokering:",
			steps: []string{
				"1. Select 'Broker' from main menu",
				"2. Configure broker port (e.g., 8080)",
				"3. Set target host (e.g., localhost:9090)",
				"4. Choose proxy mode (TCP Proxy)",
				"5. Press 's' to start broker",
			},
		},
	}

	for _, scenario := range exampleScenarios {
		examples.WriteString(SuccessStyle.Render(scenario.title))
		examples.WriteString("\n")
		for _, step := range scenario.steps {
			examples.WriteString(MutedStyle.Render("  " + step))
			examples.WriteString("\n")
		}
		examples.WriteString("\n")
	}

	// Footer
	footer := InfoStyle.Render("For more information, visit: https://github.com/ibrahmsql/gocat")
	examples.WriteString(footer)

	return examples.String()
}
