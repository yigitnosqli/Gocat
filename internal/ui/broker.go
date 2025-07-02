package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BrokerConnection represents a broker connection
type BrokerConnection struct {
	ID       string
	ClientA  string
	ClientB  string
	Status   string
	Started  time.Time
	Bytes    int64
}

// updateBroker handles broker mode input
func (m Model) updateBroker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.switchToMode(ModeMenu)
		return m, nil

	case "?":
		m.switchToMode(ModeHelp)
		return m, nil

	case "s":
		// Start/Stop broker
		if m.listening {
			m.listening = false
			m.setSuccess("Broker stopped")
		} else {
			m.listening = true
			m.setSuccess("Broker started on port 8080")
		}
		return m, nil

	case "c":
		// Clear connections
		m.setSuccess("All connections cleared")
		return m, nil
	}

	return m, nil
}

// viewBroker renders the broker interface
func (m Model) viewBroker() string {
	var content strings.Builder

	// Title
	title := HeaderStyle.Render("🔄 Network Broker")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Broker configuration
	config := m.renderBrokerConfig()
	content.WriteString(config)
	content.WriteString("\n\n")

	// Broker status and controls
	controls := m.renderBrokerControls()
	content.WriteString(controls)
	content.WriteString("\n\n")

	// Active broker connections
	connections := m.renderBrokerConnections()
	content.WriteString(connections)
	content.WriteString("\n\n")

	// Traffic statistics
	stats := m.renderBrokerStats()
	content.WriteString(stats)

	return content.String()
}

// renderBrokerConfig renders broker configuration
func (m Model) renderBrokerConfig() string {
	var config strings.Builder

	// Listen port
	portLabel := InfoStyle.Render("Broker Port:")
	portValue := BoxStyle.Width(15).Render("8080")
	portRow := lipgloss.JoinHorizontal(lipgloss.Left, portLabel, "  ", portValue)
	config.WriteString(portRow)
	config.WriteString("\n")

	// Target host
	targetLabel := InfoStyle.Render("Target Host:")
	targetValue := BoxStyle.Width(25).Render("localhost:9090")
	targetRow := lipgloss.JoinHorizontal(lipgloss.Left, targetLabel, "  ", targetValue)
	config.WriteString(targetRow)
	config.WriteString("\n")

	// Broker mode
	modeLabel := InfoStyle.Render("Mode:")
	modes := []string{"TCP Proxy", "HTTP Proxy", "SOCKS5"}
	modeButtons := make([]string, len(modes))

	for i, mode := range modes {
		if i == 0 { // TCP Proxy selected by default
			modeButtons[i] = ActiveButtonStyle.Render(mode)
		} else {
			modeButtons[i] = ButtonStyle.Render(mode)
		}
	}

	modeRow := lipgloss.JoinHorizontal(lipgloss.Left,
		modeLabel, "  ",
		lipgloss.JoinHorizontal(lipgloss.Left, modeButtons...),
	)
	config.WriteString(modeRow)

	return config.String()
}

// renderBrokerControls renders broker control buttons
func (m Model) renderBrokerControls() string {
	var controls strings.Builder

	// Status indicator
	var status string
	if m.listening {
		status = SuccessStyle.Render("● Broker Active")
	} else {
		status = MutedStyle.Render("● Broker Inactive")
	}
	controls.WriteString(status)
	controls.WriteString("\n\n")

	// Control buttons
	var startStopBtn string
	if m.listening {
		startStopBtn = ErrorStyle.Render("[S] Stop Broker")
	} else {
		startStopBtn = SuccessStyle.Render("[S] Start Broker")
	}

	clearBtn := WarningStyle.Render("[C] Clear Connections")
	helpBtn := InfoStyle.Render("[?] Help")
	backBtn := MutedStyle.Render("[Esc] Back")

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Left,
		startStopBtn, "  ",
		clearBtn, "  ",
		helpBtn, "  ",
		backBtn,
	)
	controls.WriteString(buttonRow)

	return controls.String()
}

// renderBrokerConnections renders active broker connections
func (m Model) renderBrokerConnections() string {
	var connections strings.Builder

	connTitle := InfoStyle.Render("Active Broker Connections:")
	connections.WriteString(connTitle)
	connections.WriteString("\n")

	if m.listening {
		// Sample broker connections
		sampleConnections := []BrokerConnection{
			{ID: "broker-001", ClientA: "192.168.1.100:45678", ClientB: "localhost:9090", Status: "Active", Started: time.Now().Add(-10 * time.Minute), Bytes: 1024000},
			{ID: "broker-002", ClientA: "10.0.0.50:33445", ClientB: "localhost:9090", Status: "Active", Started: time.Now().Add(-5 * time.Minute), Bytes: 512000},
			{ID: "broker-003", ClientA: "172.16.0.25:55123", ClientB: "localhost:9090", Status: "Idle", Started: time.Now().Add(-2 * time.Minute), Bytes: 256000},
		}

		for _, conn := range sampleConnections {
			duration := time.Since(conn.Started).Round(time.Second)
			bytesFormatted := formatBytes(conn.Bytes)

			var statusStyle lipgloss.Style
			switch conn.Status {
			case "Active":
				statusStyle = SuccessStyle
			case "Idle":
				statusStyle = WarningStyle
			default:
				statusStyle = MutedStyle
			}

			connInfo := fmt.Sprintf("● %s: %s ↔ %s [%s] - %s - %v",
				conn.ID, conn.ClientA, conn.ClientB,
				statusStyle.Render(conn.Status), bytesFormatted, duration)
			connections.WriteString("  " + connInfo)
			connections.WriteString("\n")
		}
	} else {
		connections.WriteString(MutedStyle.Render("  No active broker connections"))
		connections.WriteString("\n")
	}

	return connections.String()
}

// renderBrokerStats renders broker statistics
func (m Model) renderBrokerStats() string {
	var stats strings.Builder

	statsTitle := InfoStyle.Render("Broker Statistics:")
	stats.WriteString(statsTitle)
	stats.WriteString("\n")

	if m.listening {
		// Sample statistics
		totalConnections := MutedStyle.Render("  Total connections: 15")
		activeConnections := SuccessStyle.Render("  Active connections: 3")
		totalBytes := MutedStyle.Render("  Total bytes transferred: 15.2 MB")
		uptime := MutedStyle.Render("  Broker uptime: 2h 15m")

		stats.WriteString(totalConnections)
		stats.WriteString("\n")
		stats.WriteString(activeConnections)
		stats.WriteString("\n")
		stats.WriteString(totalBytes)
		stats.WriteString("\n")
		stats.WriteString(uptime)
	} else {
		stats.WriteString(MutedStyle.Render("  No statistics available"))
	}

	return stats.String()
}

// formatBytes formats byte count into human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}