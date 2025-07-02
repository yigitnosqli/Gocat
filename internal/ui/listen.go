package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListenState represents the state of the listen mode
type ListenState struct {
	port        string
	protocol    string
	connections []Connection
	logMessages []LogMessage
}

// Connection represents an active connection
type Connection struct {
	ID        string
	RemoteIP  string
	Port      string
	Protocol  string
	StartTime time.Time
	Status    string
}

// LogMessage represents a log entry
type LogMessage struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// updateListen handles listen mode input
func (m Model) updateListen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		// Start/Stop listening
		if m.listening {
			m.listening = false
			m.setSuccess("Stopped listening")
		} else {
			m.listening = true
			m.setSuccess("Started listening on port 8080")
		}
		return m, nil

	case "c":
		// Clear logs
		m.messages = []string{}
		m.setSuccess("Logs cleared")
		return m, nil
	}

	return m, nil
}

// viewListen renders the listen interface
func (m Model) viewListen() string {
	var content strings.Builder

	// Title
	title := HeaderStyle.Render("👂 Listen for Incoming Connections")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Listen configuration
	config := m.renderListenConfig()
	content.WriteString(config)
	content.WriteString("\n\n")

	// Status and controls
	controls := m.renderListenControls()
	content.WriteString(controls)
	content.WriteString("\n\n")

	// Active connections
	connections := m.renderActiveConnections()
	content.WriteString(connections)
	content.WriteString("\n\n")

	// Live logs
	logs := m.renderLiveLogs()
	content.WriteString(logs)

	return content.String()
}

// renderListenConfig renders the listening configuration
func (m Model) renderListenConfig() string {
	var config strings.Builder

	// Port configuration
	portLabel := InfoStyle.Render("Listen Port:")
	portValue := BoxStyle.Width(15).Render("8080")
	portRow := lipgloss.JoinHorizontal(lipgloss.Left, portLabel, "  ", portValue)
	config.WriteString(portRow)
	config.WriteString("\n")

	// Protocol configuration
	protocolLabel := InfoStyle.Render("Protocol:")
	protocols := []string{"TCP", "UDP", "HTTP"}
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
	config.WriteString(protocolRow)

	return config.String()
}

// renderListenControls renders the control buttons and status
func (m Model) renderListenControls() string {
	var controls strings.Builder

	// Status indicator
	var status string
	if m.listening {
		status = StatusListening()
	} else {
		status = MutedStyle.Render("● Not listening")
	}
	controls.WriteString(status)
	controls.WriteString("\n\n")

	// Control buttons
	var startStopBtn string
	if m.listening {
		startStopBtn = ErrorStyle.Render("[S] Stop Listening")
	} else {
		startStopBtn = SuccessStyle.Render("[S] Start Listening")
	}

	clearBtn := WarningStyle.Render("[C] Clear Logs")
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

// renderActiveConnections renders the list of active connections
func (m Model) renderActiveConnections() string {
	var connections strings.Builder

	connTitle := InfoStyle.Render("Active Connections:")
	connections.WriteString(connTitle)
	connections.WriteString("\n")

	if m.listening {
		// Sample connections
		sampleConnections := []Connection{
			{ID: "conn-001", RemoteIP: "192.168.1.100", Port: "8080", Protocol: "TCP", StartTime: time.Now().Add(-5 * time.Minute), Status: "Active"},
			{ID: "conn-002", RemoteIP: "10.0.0.50", Port: "8080", Protocol: "TCP", StartTime: time.Now().Add(-2 * time.Minute), Status: "Active"},
		}

		for _, conn := range sampleConnections {
			duration := time.Since(conn.StartTime).Round(time.Second)
			connInfo := fmt.Sprintf("● %s:%s (%s) - %s - %v",
				conn.RemoteIP, conn.Port, conn.Protocol, conn.Status, duration)
			connections.WriteString(SuccessStyle.Render("  " + connInfo))
			connections.WriteString("\n")
		}
	} else {
		connections.WriteString(MutedStyle.Render("  No active connections"))
		connections.WriteString("\n")
	}

	return connections.String()
}

// renderLiveLogs renders the live log messages
func (m Model) renderLiveLogs() string {
	var logs strings.Builder

	logTitle := InfoStyle.Render("Live Logs:")
	logs.WriteString(logTitle)
	logs.WriteString("\n")

	if m.listening {
		// Sample log messages
		sampleLogs := []LogMessage{
			{Timestamp: time.Now().Add(-3 * time.Minute), Level: "INFO", Message: "Server started on port 8080"},
			{Timestamp: time.Now().Add(-2 * time.Minute), Level: "INFO", Message: "New connection from 192.168.1.100"},
			{Timestamp: time.Now().Add(-1 * time.Minute), Level: "INFO", Message: "Data received: 1024 bytes"},
			{Timestamp: time.Now().Add(-30 * time.Second), Level: "INFO", Message: "New connection from 10.0.0.50"},
		}

		for _, log := range sampleLogs {
			timestamp := log.Timestamp.Format("15:04:05")
			var levelStyle lipgloss.Style
			switch log.Level {
			case "INFO":
				levelStyle = InfoStyle
			case "WARN":
				levelStyle = WarningStyle
			case "ERROR":
				levelStyle = ErrorStyle
			default:
				levelStyle = MutedStyle
			}

			logLine := fmt.Sprintf("[%s] %s %s",
				timestamp,
				levelStyle.Render(log.Level),
				log.Message)
			logs.WriteString("  " + logLine)
			logs.WriteString("\n")
		}
	} else {
		logs.WriteString(MutedStyle.Render("  No logs available"))
		logs.WriteString("\n")
	}

	return logs.String()
}