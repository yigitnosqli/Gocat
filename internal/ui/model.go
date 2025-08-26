package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppMode represents different modes of the application
type AppMode int

const (
	ModeMenu AppMode = iota
	ModeConnect
	ModeListen
	ModeChat
	ModeBroker
	ModeScan
	ModeHelp
)

// BrokerState represents the state of the broker mode
type BrokerState struct {
	connections []BrokerConnection
	isListening bool
	port        string
}

// Model represents the main application state
type Model struct {
	mode           AppMode
	width          int
	height         int
	menuItems      []string
	selected       int
	input          string
	cursor         int
	messages       []string
	chatState      ChatState
	brokerState    BrokerState
	scanState      ScanState
	listenState    ListenState
	status         string
	connected      bool
	listening      bool
	showHelp       bool
	errorMsg       string
	successMsg     string
	connectionInfo string
	lastActivity   time.Time
}

// NewModel creates a new model with default values
func NewModel() Model {
	return Model{
		mode: ModeMenu,
		menuItems: []string{
			"🔗 Connect",
			"👂 Listen",
			"💬 Chat",
			"🔄 Broker",
			"🔍 Scan",
			"❓ Help",
			"🚪 Exit",
		},
		selected:     0,
		messages:     []string{},
		status:       "Ready",
		lastActivity: time.Now(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case ModeMenu:
			return m.updateMenu(msg)
		case ModeConnect:
			return m.updateConnect(msg)
		case ModeListen:
			return m.updateListen(msg)
		case ModeChat:
			return m.updateChat(msg)
		case ModeBroker:
			return m.updateBroker(msg)
		case ModeScan:
			return m.updateScan(msg)
		case ModeHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

// View renders the current view
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	switch m.mode {
	case ModeMenu:
		content = m.viewMenu()
	case ModeConnect:
		content = m.viewConnect()
	case ModeListen:
		content = m.viewListen()
	case ModeChat:
		content = m.viewChat()
	case ModeBroker:
		content = m.viewBroker()
	case ModeScan:
		content = m.viewScan()
	case ModeHelp:
		content = m.viewHelp()
	default:
		content = m.viewMenu()
	}

	// Add header and footer
	header := m.renderHeader()
	footer := m.renderFooter()

	// Calculate content height
	contentHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 2

	// Wrap content in a box
	contentBox := AdaptiveBoxStyle(m.width, contentHeight).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, contentBox, footer)
}

// renderHeader renders the application header
func (m Model) renderHeader() string {
	title := "GoCat - Network Swiss Army Knife"
	status := m.renderStatus()

	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		TitleStyle.Render(title),
		strings.Repeat(" ", m.width-lipgloss.Width(title)-lipgloss.Width(status)-4),
		status,
	)

	return AdaptiveHeaderStyle(m.width).Render(headerContent)
}

// renderFooter renders the application footer
func (m Model) renderFooter() string {
	var help string
	switch m.mode {
	case ModeMenu:
		help = "↑/↓: Navigate • Enter: Select • q: Quit"
	case ModeConnect, ModeListen, ModeChat, ModeBroker, ModeScan:
		help = "Esc: Back to Menu • q: Quit • ?: Help"
	case ModeHelp:
		help = "Esc: Back • q: Quit"
	}

	return StatusBarStyle.Width(m.width).Render(help)
}

// renderStatus renders the current status
func (m Model) renderStatus() string {
	if m.errorMsg != "" {
		return ErrorStyle.Render("✗ " + m.errorMsg)
	}
	if m.successMsg != "" {
		return SuccessStyle.Render("✓ " + m.successMsg)
	}

	if m.connected {
		return StatusConnected()
	}
	if m.listening {
		return StatusListening()
	}
	return MutedStyle.Render("● " + m.status)
}

// Helper methods for clearing messages
func (m *Model) clearMessages() {
	m.errorMsg = ""
	m.successMsg = ""
}

func (m *Model) setSuccess(msg string) {
	m.clearMessages()
	m.successMsg = msg
}

// Mode switching helpers
func (m *Model) switchToMode(mode AppMode) {
	m.mode = mode
	m.clearMessages()
	m.input = ""
	m.cursor = 0
}
