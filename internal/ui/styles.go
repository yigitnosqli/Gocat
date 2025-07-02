package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#7D56F4") // Purple
	SecondaryColor = lipgloss.Color("#04B575") // Green
	AccentColor    = lipgloss.Color("#FF6B6B") // Red
	WarningColor   = lipgloss.Color("#FFD93D") // Yellow
	InfoColor      = lipgloss.Color("#6BCF7F") // Light Green
	ErrorColor     = lipgloss.Color("#FF4757") // Dark Red

	// Neutral colors
	TextColor      = lipgloss.Color("#FAFAFA") // Light
	MutedColor     = lipgloss.Color("#8B949E") // Gray
	BorderColor    = lipgloss.Color("#30363D") // Dark Gray
	BackgroundColor = lipgloss.Color("#0D1117") // Dark Background
)

// Common styles
var (
	// Title style
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		PaddingLeft(2).
		PaddingRight(2)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(TextColor).
		Background(PrimaryColor).
		Padding(0, 2).
		MarginBottom(1)

	// Box style
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderColor).
		Padding(1, 2).
		Margin(1)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
		Foreground(InfoColor).
		Bold(true)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true)

	// Info style
	InfoStyle = lipgloss.NewStyle().
		Foreground(InfoColor)

	// Muted style
	MutedStyle = lipgloss.NewStyle().
		Foreground(MutedColor)

	// Highlight style
	HighlightStyle = lipgloss.NewStyle().
		Background(PrimaryColor).
		Foreground(TextColor).
		Bold(true)

	// Button style
	ButtonStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(PrimaryColor).
		Padding(0, 2).
		Margin(0, 1).
		Border(lipgloss.RoundedBorder())

	// Active button style
	ActiveButtonStyle = lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Background(TextColor).
		Padding(0, 2).
		Margin(0, 1).
		Border(lipgloss.RoundedBorder()).
		Bold(true)

	// Status bar style
	StatusBarStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(BorderColor).
		Padding(0, 1)

	// Help style
	HelpStyle = lipgloss.NewStyle().
		Foreground(MutedColor).
		MarginTop(1)
)

// Adaptive styles for different terminal sizes
func AdaptiveBoxStyle(width, height int) lipgloss.Style {
	return BoxStyle.Width(width - 4).Height(height - 4)
}

func AdaptiveHeaderStyle(width int) lipgloss.Style {
	return HeaderStyle.Width(width - 2)
}



// Status indicators
func StatusConnected() string {
	return SuccessStyle.Render("● Connected")
}

func StatusDisconnected() string {
	return ErrorStyle.Render("● Disconnected")
}

func StatusListening() string {
	return InfoStyle.Render("● Listening")
}

func StatusConnecting() string {
	return WarningStyle.Render("● Connecting...")
}