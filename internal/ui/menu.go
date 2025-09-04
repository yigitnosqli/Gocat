package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateMenu handles menu navigation
func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}

	case "down", "j":
		if m.selected < len(m.menuItems)-1 {
			m.selected++
		}

	case "enter", " ":
		switch m.selected {
		case 0: // Connect
			m.switchToMode(ModeConnect)
		case 1: // Listen
			m.switchToMode(ModeListen)
		case 2: // Chat
			m.switchToMode(ModeChat)
		case 3: // Broker
			m.switchToMode(ModeBroker)
		case 4: // Scan
			m.switchToMode(ModeScan)
		case 5: // Help
			m.switchToMode(ModeHelp)
		case 6: // Exit
			return m, tea.Quit
		}
	}

	return m, nil
}

// viewMenu renders the main menu
func (m Model) viewMenu() string {
	title := "🐱 GoCat"
	description := "Network Swiss Army Knife"
	
	menuItems := []string{
		"Connect",
		"Listen",
		"Chat",
		"Broker",
		"Scan",
		"Help",
		"Quit",
	}
	
	var menu strings.Builder
	
	// Title
	menu.WriteString(title + "\n")
	menu.WriteString(description + "\n\n")
	
	// Menu items
	for i, item := range menuItems {
		if i == m.selected {
			menu.WriteString("> " + item + "\n")
		} else {
			menu.WriteString("  " + item + "\n")
		}
	}
	
	// Feature highlights
	menu.WriteString("\n")
	features := []string{
		"🔗 TCP/UDP connections",
		"📡 Port scanning",
		"💬 Real-time chat",
		"🔄 Message broker",
	}
	
	for _, feature := range features {
		menu.WriteString("  " + feature + "\n")
	}
	
	return menu.String()
}
