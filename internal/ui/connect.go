package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConnectState represents the state of the connect mode
type ConnectState struct {
	host          string
	port          string
	protocol      string
	focused       int // 0: host, 1: port, 2: protocol
	protocols     []string
	protocolIndex int
}

// updateConnect handles connect mode input
func (m Model) updateConnect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.switchToMode(ModeMenu)
		return m, nil

	case "?":
		m.switchToMode(ModeHelp)
		return m, nil

	case "tab":
		// Cycle through input fields
		// This is a simplified version - in real implementation,
		// you'd track which field is focused

	case "enter":
		// Attempt connection
		m.setSuccess("Attempting to connect...")
		m.connected = true
		return m, nil

	default:
		// Handle text input
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}

	return m, nil
}

// viewConnect renders the connect interface
func (m Model) viewConnect() string {
	var content strings.Builder

	// Title
	title := HeaderStyle.Render("🔗 Connect to Remote Host")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Connection form
	form := m.renderConnectionForm()
	content.WriteString(form)
	content.WriteString("\n\n")

	// Connection status
	if m.connected {
		status := SuccessStyle.Render("✓ Connected successfully!")
		content.WriteString(status)
	} else {
		status := MutedStyle.Render("● Not connected")
		content.WriteString(status)
	}
	content.WriteString("\n\n")

	// Instructions
	instructions := []string{
		"Tab: Switch between fields",
		"Enter: Connect",
		"Esc: Back to menu",
		"?: Help",
	}

	for _, instruction := range instructions {
		content.WriteString(HelpStyle.Render("  " + instruction))
		content.WriteString("\n")
	}

	// Recent connections
	content.WriteString("\n")
	recentTitle := InfoStyle.Render("Recent Connections:")
	content.WriteString(recentTitle)
	content.WriteString("\n")

	recentConnections := []string{
		"localhost:8080 (TCP)",
		"192.168.1.100:22 (SSH)",
		"example.com:443 (HTTPS)",
	}

	for _, conn := range recentConnections {
		content.WriteString(MutedStyle.Render("  • " + conn))
		content.WriteString("\n")
	}

	return content.String()
}

// renderConnectionForm renders the connection input form
func (m Model) renderConnectionForm() string {
	var form strings.Builder

	// Host input
	hostLabel := InfoStyle.Render("Host:")
	hostInput := BoxStyle.Width(30).Render("localhost")
	hostRow := lipgloss.JoinHorizontal(lipgloss.Left, hostLabel, "  ", hostInput)
	form.WriteString(hostRow)
	form.WriteString("\n")

	// Port input
	portLabel := InfoStyle.Render("Port:")
	portInput := BoxStyle.Width(30).Render("8080")
	portRow := lipgloss.JoinHorizontal(lipgloss.Left, portLabel, "  ", portInput)
	form.WriteString(portRow)
	form.WriteString("\n")

	// Protocol selection
	protocolLabel := InfoStyle.Render("Protocol:")
	protocols := []string{"TCP", "UDP", "HTTP", "HTTPS", "SSH"}
	protocolButtons := make([]string, len(protocols))

	for i, protocol := range protocols {
		if i == 0 { // TCP selected by default
			protocolButtons[i] = ActiveButtonStyle.Render(protocol)
		} else {
			protocolButtons[i] = ButtonStyle.Render(protocol)
		}
	}

	protocolRow := lipgloss.JoinHorizontal(lipgloss.Left,
		protocolLabel, "  ",
		lipgloss.JoinHorizontal(lipgloss.Left, protocolButtons...),
	)
	form.WriteString(protocolRow)
	form.WriteString("\n")

	// Connect button
	form.WriteString("\n")
	connectBtn := ButtonStyle.Width(20).Align(lipgloss.Center).Render("🔗 Connect")
	form.WriteString(lipgloss.NewStyle().Width(50).Align(lipgloss.Center).Render(connectBtn))

	return form.String()
}
