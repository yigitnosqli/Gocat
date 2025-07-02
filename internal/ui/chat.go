package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatMessage represents a chat message
type ChatMessage struct {
	Timestamp time.Time
	Sender    string
	Message   string
	IsLocal   bool
}

// ChatState represents the state of the chat mode
type ChatState struct {
	messages    []ChatMessage
	inputBuffer string
	connected   bool
	remoteHost  string
	scrollPos   int
}

// updateChat handles chat mode input
func (m Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.switchToMode(ModeMenu)
		return m, nil

	case "?":
		m.switchToMode(ModeHelp)
		return m, nil

	case "enter":
		// Send message
		if strings.TrimSpace(m.input) != "" {
			// Add message to chat (simulated)
			m.messages = append(m.messages, "You: "+m.input)
			m.input = ""
			m.setSuccess("Message sent")
		}
		return m, nil

	case "backspace":
		// Remove last character
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil

	case "up":
		// Scroll up in chat history
		return m, nil

	case "down":
		// Scroll down in chat history
		return m, nil

	default:
		// Add character to input
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}

	return m, nil
}

// viewChat renders the chat interface
func (m Model) viewChat() string {
	var content strings.Builder

	// Add GoCat banner for chat
	banner := m.renderChatBanner()
	content.WriteString(banner)
	content.WriteString("\n\n")

	// Title with connection status
	title := HeaderStyle.Render("💬 Chat Mode")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Connection info
	connInfo := m.renderChatConnectionInfo()
	content.WriteString(connInfo)
	content.WriteString("\n\n")

	// Chat messages area
	messagesArea := m.renderChatMessages()
	content.WriteString(messagesArea)
	content.WriteString("\n\n")

	// Input area
	inputArea := m.renderChatInput()
	content.WriteString(inputArea)
	content.WriteString("\n\n")

	// Chat controls
	controls := m.renderChatControls()
	content.WriteString(controls)

	return content.String()
}

// renderChatConnectionInfo renders connection information
func (m Model) renderChatConnectionInfo() string {
	var info strings.Builder

	if m.connected {
		status := StatusConnected()
		remote := InfoStyle.Render("Connected to: localhost:8080")
		connInfo := lipgloss.JoinHorizontal(lipgloss.Left, status, "  ", remote)
		info.WriteString(connInfo)
	} else {
		status := StatusDisconnected()
		message := ErrorStyle.Render("Not connected to any remote host")
		connInfo := lipgloss.JoinHorizontal(lipgloss.Left, status, "  ", message)
		info.WriteString(connInfo)
		info.WriteString("\n")
		info.WriteString(MutedStyle.Render("  Use Connect mode first to establish a connection"))
	}

	return info.String()
}

// renderChatMessages renders the chat message history
func (m Model) renderChatMessages() string {
	var messages strings.Builder

	messagesTitle := InfoStyle.Render("Messages:")
	messages.WriteString(messagesTitle)
	messages.WriteString("\n")

	// Create a box for messages
	messageBox := BoxStyle.Width(60).Height(10)

	if len(m.messages) == 0 {
		// No messages yet
		emptyMsg := MutedStyle.Render("No messages yet. Start typing to send a message!")
		messageContent := lipgloss.NewStyle().Padding(2).Render(emptyMsg)
		messages.WriteString(messageBox.Render(messageContent))
	} else {
		// Sample chat messages
		sampleMessages := []ChatMessage{
			{Timestamp: time.Now().Add(-10 * time.Minute), Sender: "Remote", Message: "Hello! Connection established.", IsLocal: false},
			{Timestamp: time.Now().Add(-9 * time.Minute), Sender: "You", Message: "Hi there! Great to connect.", IsLocal: true},
			{Timestamp: time.Now().Add(-8 * time.Minute), Sender: "Remote", Message: "How can I help you today?", IsLocal: false},
			{Timestamp: time.Now().Add(-7 * time.Minute), Sender: "You", Message: "Just testing the chat functionality.", IsLocal: true},
			{Timestamp: time.Now().Add(-5 * time.Minute), Sender: "Remote", Message: "Looks like it's working perfectly!", IsLocal: false},
		}

		var messageContent strings.Builder
		for _, msg := range sampleMessages {
			timestamp := msg.Timestamp.Format("15:04")
			var msgStyle lipgloss.Style
			var prefix string

			if msg.IsLocal {
				msgStyle = SuccessStyle
				prefix = "→"
			} else {
				msgStyle = InfoStyle
				prefix = "←"
			}

			msgLine := fmt.Sprintf("%s [%s] %s: %s",
				prefix, timestamp, msgStyle.Render(msg.Sender), msg.Message)
			messageContent.WriteString(msgLine)
			messageContent.WriteString("\n")
		}

		messages.WriteString(messageBox.Render(messageContent.String()))
	}

	return messages.String()
}

// renderChatInput renders the message input area
func (m Model) renderChatInput() string {
	var input strings.Builder

	inputLabel := InfoStyle.Render("Message:")
	input.WriteString(inputLabel)
	input.WriteString("\n")

	// Input box with current text
	inputText := m.input
	if inputText == "" {
		inputText = MutedStyle.Render("Type your message here...")
	}

	// Add cursor
	if len(m.input) > 0 {
		inputText = m.input + "█"
	}

	inputBox := BoxStyle.Width(60).Render(inputText)
	input.WriteString(inputBox)

	return input.String()
}

// renderChatControls renders chat control buttons and help
func (m Model) renderChatControls() string {
	var controls strings.Builder

	// Control buttons
	sendBtn := SuccessStyle.Render("[Enter] Send Message")
	clearBtn := WarningStyle.Render("[Ctrl+L] Clear Chat")
	scrollBtn := InfoStyle.Render("[↑↓] Scroll History")
	helpBtn := InfoStyle.Render("[?] Help")
	backBtn := MutedStyle.Render("[Esc] Back to Menu")

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Left,
		sendBtn, "  ",
		clearBtn, "  ",
		scrollBtn, "  ",
		helpBtn, "  ",
		backBtn,
	)
	controls.WriteString(buttonRow)
	controls.WriteString("\n\n")

	// Chat statistics
	stats := m.renderChatStats()
	controls.WriteString(stats)

	return controls.String()
}

// renderChatStats renders chat statistics
func (m Model) renderChatStats() string {
	var stats strings.Builder

	statsTitle := MutedStyle.Render("Session Stats:")
	stats.WriteString(statsTitle)
	stats.WriteString("\n")

	// Sample statistics
	messageCount := MutedStyle.Render("  Messages sent: 12")
	bytesTransferred := MutedStyle.Render("  Bytes transferred: 2.4 KB")
	sessionTime := MutedStyle.Render("  Session time: 15m 32s")

	stats.WriteString(messageCount)
	stats.WriteString("\n")
	stats.WriteString(bytesTransferred)
	stats.WriteString("\n")
	stats.WriteString(sessionTime)

	return stats.String()
}

// renderChatBanner renders the GoCat banner for chat mode
func (m Model) renderChatBanner() string {
	logo := `
  ██████╗  ██████╗  ██████╗ █████╗ ████████╗
 ██╔════╝ ██╔═══██╗██╔════╝██╔══██╗╚══██╔══╝
 ██║  ███╗██║   ██║██║     ███████║   ██║   
 ██║   ██║██║   ██║██║     ██╔══██║   ██║   
 ╚██████╔╝╚██████╔╝╚██████╗██║  ██║   ██║   
  ╚═════╝  ╚═════╝  ╚═════╝╚═╝  ╚═╝   ╚═╝   
`
	return TitleStyle.Render(logo)
}
