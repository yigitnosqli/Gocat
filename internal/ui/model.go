package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ibrahmsql/gocat/internal/readline"
)

// AppMode represents the current application mode
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

// BrokerState represents the broker mode state
type BrokerState struct {
	port        string
	protocol    string
	connections []BrokerConnection
	isListening bool
	stats       BrokerStats
}

// BrokerStats represents broker statistics
type BrokerStats struct {
	TotalConnections int64
	ActiveConnections int
	BytesTransferred int64
	Uptime          time.Duration
}

// Model represents the main application model
type Model struct {
	// Current application mode
	mode AppMode

	// Common state
	input       string
	errorMsg    string
	successMsg  string
	connected   bool
	connectionInfo string
	quitting    bool

	// UI dimensions
	width  int
	height int

	// Readline editor for advanced input handling
	readlineEditor *readline.Editor
	readlineMode   bool
	historyEnabled bool

	// Mode-specific states (using existing structs)
	connectState *ConnectState
	listenState  *ListenState
	chatState    *ChatState
	scanState    *ScanState
	brokerState  *BrokerState
	
	// Additional state fields
	listening    bool
	lastActivity time.Time
	messages     []string
	selected     int
	menuItems    []string
}

// NewModel creates a new model instance
func NewModel() Model {
	// Initialize readline editor with advanced features
	readlineEditor := readline.NewEditor()
	readlineEditor.SetPrompt("gocat> ")
	readlineEditor.SetMaxHistory(1000)
	readlineEditor.SetHistoryFile(".gocat_history")
	readlineEditor.SetIgnoreCase(true)
	readlineEditor.SetWordBreakChars(" \t\n\r\f\v")
	
	// Set up completions for common commands
	completions := []string{
		"connect", "listen", "chat", "broker", "scan", "help", "quit", "exit",
		"clear", "history", "tcp", "udp", "localhost", "127.0.0.1",
	}
	readlineEditor.SetCompletions(completions)
	
	return Model{
		mode:         ModeMenu,
		input:        "",
		errorMsg:     "",
		successMsg:   "",
		connected:    false,
		connectionInfo: "",
		quitting:     false,
		width:        80,
		height:       24,
		readlineEditor: readlineEditor,
		readlineMode:   false,
		historyEnabled: true,
		connectState: &ConnectState{
			host:          "",
			port:          "80",
			protocol:      "tcp",
			focused:       0,
			protocols:     []string{"tcp", "udp"},
			protocolIndex: 0,
		},
		listenState: &ListenState{
			port:        "8080",
			protocol:    "tcp",
			connections: make([]Connection, 0),
			logMessages: make([]LogMessage, 0),
		},
		chatState: &ChatState{
			messages:    make([]ChatMessage, 0),
			inputBuffer: "",
			connected:   false,
			remoteHost:  "",
			scrollPos:   0,
		},
		scanState: &ScanState{
			targetHost: "",
			portRange:  "1-1000",
			results:    make([]ScanResult, 0),
			scanning:   false,
			progress:   0,
			totalPorts: 0,
			scanType:   "tcp",
		},
		brokerState: &BrokerState{
			port:        "8080",
			protocol:    "tcp",
			connections: make([]BrokerConnection, 0),
			isListening: false,
			stats:       BrokerStats{},
		},
		listening: false,
		lastActivity: time.Now(),
		messages: make([]string, 0),
		selected: 0,
		menuItems: []string{"Connect", "Listen", "Chat", "Broker", "Scan", "Help", "Quit"},
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
		// Handle readline mode if enabled
		if m.readlineMode && m.readlineEditor != nil {
			switch msg.String() {
			case "ctrl+c":
				m.readlineMode = false
				m.errorMsg = ""
				m.successMsg = ""
				return m, nil
			case "enter":
				// Process readline input
				if m.input != "" {
					m.addToHistory(m.input)
					m.readlineMode = false
					// Process the command based on current mode
					return m.processCommand(m.input)
				}
				m.readlineMode = false
				return m, nil
			default:
				// For now, just disable readline mode on other keys
				m.readlineMode = false
				return m, nil
			}
		}

		// Global key bindings
		switch msg.String() {
		case "ctrl+c", "q":
			if m.mode == ModeMenu {
				m.quitting = true
				return m, tea.Quit
			}
			// For other modes, let mode-specific handlers deal with it
		case "esc":
			m.mode = ModeMenu
			m.errorMsg = ""
			m.successMsg = ""
			m.updateCompletionsForMode()
			return m, nil
		case "ctrl+l":
			// Clear screen
			m.errorMsg = ""
			m.successMsg = ""
			return m, nil
		case "ctrl+h":
			// Show history
			history := m.getHistory()
			if len(history) > 0 {
				lastFive := history
				if len(history) > 5 {
					lastFive = history[len(history)-5:]
				}
				m.setSuccess("Recent commands: " + fmt.Sprintf("%v", lastFive))
			} else {
				m.setSuccess("No command history")
			}
			return m, nil
		}

		// Mode-specific updates
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
	if m.quitting {
		return "Goodbye!\n"
	}

	switch m.mode {
	case ModeMenu:
		return m.viewMenu()
	case ModeConnect:
		return m.viewConnect()
	case ModeListen:
		return m.viewListen()
	case ModeChat:
		return m.viewChat()
	case ModeBroker:
		return m.viewBroker()
	case ModeScan:
		return m.viewScan()
	case ModeHelp:
		return m.viewHelp()
	default:
		return "Unknown mode\n"
	}
}

// switchToMode switches to a different application mode
func (m *Model) switchToMode(mode AppMode) {
	m.mode = mode
	m.errorMsg = ""
	m.successMsg = ""
}

// setError sets an error message
func (m *Model) setError(msg string) {
	m.errorMsg = msg
	m.successMsg = ""
}

// setSuccess sets a success message
func (m *Model) setSuccess(msg string) {
	m.successMsg = msg
	m.errorMsg = ""
}

// addToHistory adds a command to readline history
func (m *Model) addToHistory(command string) {
	if m.historyEnabled && m.readlineEditor != nil {
		m.readlineEditor.AddHistoryEntry(command)
	}
}

// getHistory returns the command history
func (m *Model) getHistory() []string {
	if m.readlineEditor != nil {
		return m.readlineEditor.GetHistory()
	}
	return []string{}
}

// updateCompletionsForMode updates completions based on current mode
func (m *Model) updateCompletionsForMode() {
	if m.readlineEditor == nil {
		return
	}

	var completions []string
	switch m.mode {
	case ModeConnect:
		completions = []string{
			"tcp", "udp", "localhost", "127.0.0.1", "0.0.0.0",
			"80", "443", "22", "21", "25", "53", "110", "143", "993", "995",
			"connect", "disconnect", "status", "help", "back", "quit",
		}
	case ModeListen:
		completions = []string{
			"tcp", "udp", "start", "stop", "status", "clear",
			"8080", "3000", "5000", "8000", "9000",
			"help", "back", "quit",
		}
	case ModeChat:
		completions = []string{
			"send", "clear", "history", "connect", "disconnect",
			"help", "back", "quit",
		}
	case ModeBroker:
		completions = []string{
			"start", "stop", "status", "clear", "connections",
			"tcp", "udp", "help", "back", "quit",
		}
	case ModeScan:
		completions = []string{
			"tcp", "udp", "syn", "connect", "start", "stop", "clear",
			"1-1000", "1-65535", "80,443,22", "localhost", "127.0.0.1",
			"help", "back", "quit",
		}
	default:
		completions = []string{
			"connect", "listen", "chat", "broker", "scan", "help", "quit", "exit",
			"clear", "history",
		}
	}

	m.readlineEditor.SetCompletions(completions)
}





// processCommand processes a command based on the current mode
func (m Model) processCommand(command string) (tea.Model, tea.Cmd) {
	// Clear the input after processing
	m.input = ""
	
	// Handle global commands first
	switch command {
	case "quit", "exit":
		m.quitting = true
		return m, tea.Quit
	case "clear":
		m.errorMsg = ""
		m.successMsg = ""
		return m, nil
	case "history":
		history := m.getHistory()
		if len(history) > 0 {
			lastFive := history
			if len(history) > 5 {
				lastFive = history[len(history)-5:]
			}
			m.setSuccess(fmt.Sprintf("Recent commands: %v", lastFive))
		} else {
			m.setSuccess("No command history")
		}
		return m, nil
	case "help":
		m.mode = ModeHelp
		m.updateCompletionsForMode()
		return m, nil
	case "back", "menu":
		m.mode = ModeMenu
		m.updateCompletionsForMode()
		return m, nil
	}
	
	// Handle mode-specific commands
	switch m.mode {
	case ModeMenu:
		switch command {
		case "connect", "1":
			m.mode = ModeConnect
			m.updateCompletionsForMode()
		case "listen", "2":
			m.mode = ModeListen
			m.updateCompletionsForMode()
		case "chat", "3":
			m.mode = ModeChat
			m.updateCompletionsForMode()
		case "broker", "4":
			m.mode = ModeBroker
			m.updateCompletionsForMode()
		case "scan", "5":
			m.mode = ModeScan
			m.updateCompletionsForMode()
		default:
			m.setError("Unknown command: " + command)
		}
	default:
		// For other modes, set the input and let the mode-specific handlers process it
		m.input = command
		// Process through normal key handling
		return m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	
	return m, nil
}
