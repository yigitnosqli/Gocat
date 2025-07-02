package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	var content strings.Builder

	// Add title
	title := TitleStyle.Render("🐱 GoCat")
	content.WriteString(lipgloss.NewStyle().Width(m.width-6).Align(lipgloss.Center).Render(title))
	content.WriteString("\n\n")

	// Add description
	desc := MutedStyle.Render("A powerful network utility tool with beautiful terminal interface")
	content.WriteString(lipgloss.NewStyle().Width(m.width-6).Align(lipgloss.Center).Render(desc))
	content.WriteString("\n\n")

	// Add menu items
	for i, item := range m.menuItems {
		var style lipgloss.Style
		if i == m.selected {
			style = ActiveButtonStyle
		} else {
			style = ButtonStyle
		}

		menuItem := style.Render(item)
		content.WriteString(lipgloss.NewStyle().Width(m.width-6).Align(lipgloss.Center).Render(menuItem))
		content.WriteString("\n")
	}

	// Add some spacing
	content.WriteString("\n")

	// Add feature highlights
	highlights := []string{
		"✨ Beautiful terminal UI with colors",
		"🚀 Fast and efficient networking", 
		"🔧 Multiple connection modes",
		"📡 Real-time communication",
		"🔍 Network scanning capabilities",
	}

	for _, highlight := range highlights {
		content.WriteString(InfoStyle.Render("  " + highlight))
		content.WriteString("\n")
	}

	return content.String()
}