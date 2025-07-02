package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ScanResult represents a port scan result
type ScanResult struct {
	Host    string
	Port    int
	Status  string
	Service string
	Banner  string
}

// ScanState represents the state of the scan mode
type ScanState struct {
	targetHost string
	portRange  string
	scanType   string
	results    []ScanResult
	scanning   bool
	progress   int
	totalPorts int
}

// updateScan handles scan mode input
func (m Model) updateScan(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		// Start/Stop scan
		if m.listening { // Using listening as scanning flag
			m.listening = false
			m.setSuccess("Scan stopped")
		} else {
			m.listening = true
			m.setSuccess("Starting port scan...")
		}
		return m, nil

	case "c":
		// Clear results
		m.messages = []string{}
		m.setSuccess("Scan results cleared")
		return m, nil

	case "e":
		// Export results
		m.setSuccess("Results exported to scan_results.txt")
		return m, nil
	}

	return m, nil
}

// viewScan renders the scan interface
func (m Model) viewScan() string {
	var content strings.Builder

	// Title
	title := HeaderStyle.Render("🔍 Network Scanner")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Scan configuration
	config := m.renderScanConfig()
	content.WriteString(config)
	content.WriteString("\n\n")

	// Scan controls and status
	controls := m.renderScanControls()
	content.WriteString(controls)
	content.WriteString("\n\n")

	// Scan progress
	if m.listening { // Using listening as scanning flag
		progress := m.renderScanProgress()
		content.WriteString(progress)
		content.WriteString("\n\n")
	}

	// Scan results
	results := m.renderScanResults()
	content.WriteString(results)

	return content.String()
}

// renderScanConfig renders scan configuration
func (m Model) renderScanConfig() string {
	var config strings.Builder

	// Target host
	hostLabel := InfoStyle.Render("Target Host:")
	hostValue := BoxStyle.Width(25).Render("192.168.1.1")
	hostRow := lipgloss.JoinHorizontal(lipgloss.Left, hostLabel, "  ", hostValue)
	config.WriteString(hostRow)
	config.WriteString("\n")

	// Port range
	portLabel := InfoStyle.Render("Port Range:")
	portValue := BoxStyle.Width(25).Render("1-1000")
	portRow := lipgloss.JoinHorizontal(lipgloss.Left, portLabel, "  ", portValue)
	config.WriteString(portRow)
	config.WriteString("\n")

	// Scan type
	typeLabel := InfoStyle.Render("Scan Type:")
	scanTypes := []string{"TCP Connect", "SYN Scan", "UDP Scan", "Service Scan"}
	typeButtons := make([]string, len(scanTypes))

	for i, scanType := range scanTypes {
		if i == 0 { // TCP Connect selected by default
			typeButtons[i] = ActiveButtonStyle.Render(scanType)
		} else {
			typeButtons[i] = ButtonStyle.Render(scanType)
		}
	}

	typeRow := lipgloss.JoinHorizontal(lipgloss.Left,
		typeLabel, "  ",
		lipgloss.JoinHorizontal(lipgloss.Left, typeButtons...),
	)
	config.WriteString(typeRow)

	return config.String()
}

// renderScanControls renders scan control buttons
func (m Model) renderScanControls() string {
	var controls strings.Builder

	// Status indicator
	var status string
	if m.listening { // Using listening as scanning flag
		status = WarningStyle.Render("● Scanning...")
	} else {
		status = MutedStyle.Render("● Ready to scan")
	}
	controls.WriteString(status)
	controls.WriteString("\n\n")

	// Control buttons
	var startStopBtn string
	if m.listening {
		startStopBtn = ErrorStyle.Render("[S] Stop Scan")
	} else {
		startStopBtn = SuccessStyle.Render("[S] Start Scan")
	}

	clearBtn := WarningStyle.Render("[C] Clear Results")
	exportBtn := InfoStyle.Render("[E] Export Results")
	helpBtn := InfoStyle.Render("[?] Help")
	backBtn := MutedStyle.Render("[Esc] Back")

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Left,
		startStopBtn, "  ",
		clearBtn, "  ",
		exportBtn, "  ",
		helpBtn, "  ",
		backBtn,
	)
	controls.WriteString(buttonRow)

	return controls.String()
}

// renderScanProgress renders scan progress bar
func (m Model) renderScanProgress() string {
	var progress strings.Builder

	progressTitle := InfoStyle.Render("Scan Progress:")
	progress.WriteString(progressTitle)
	progress.WriteString("\n")

	// Simulate progress (in real implementation, this would be actual progress)
	currentPort := 450
	totalPorts := 1000
	progressPercent := float64(currentPort) / float64(totalPorts) * 100

	// Progress bar
	barWidth := 50
	filledWidth := int(float64(barWidth) * progressPercent / 100)
	emptyWidth := barWidth - filledWidth

	progressBar := SuccessStyle.Render(strings.Repeat("█", filledWidth)) +
		MutedStyle.Render(strings.Repeat("░", emptyWidth))

	progressInfo := fmt.Sprintf("[%s] %.1f%% (%d/%d ports)",
		progressBar, progressPercent, currentPort, totalPorts)

	progress.WriteString("  " + progressInfo)
	progress.WriteString("\n")

	// Current scanning info
	currentInfo := MutedStyle.Render(fmt.Sprintf("  Currently scanning: 192.168.1.1:%d", currentPort))
	progress.WriteString(currentInfo)

	return progress.String()
}

// renderScanResults renders scan results
func (m Model) renderScanResults() string {
	var results strings.Builder

	resultsTitle := InfoStyle.Render("Scan Results:")
	results.WriteString(resultsTitle)
	results.WriteString("\n")

	if m.listening || len(m.messages) > 0 {
		// Sample scan results
		sampleResults := []ScanResult{
			{Host: "192.168.1.1", Port: 22, Status: "Open", Service: "SSH", Banner: "OpenSSH 8.9"},
			{Host: "192.168.1.1", Port: 80, Status: "Open", Service: "HTTP", Banner: "Apache/2.4.41"},
			{Host: "192.168.1.1", Port: 443, Status: "Open", Service: "HTTPS", Banner: "nginx/1.18.0"},
			{Host: "192.168.1.1", Port: 21, Status: "Filtered", Service: "FTP", Banner: ""},
			{Host: "192.168.1.1", Port: 23, Status: "Closed", Service: "Telnet", Banner: ""},
			{Host: "192.168.1.1", Port: 25, Status: "Open", Service: "SMTP", Banner: "Postfix 3.4.13"},
			{Host: "192.168.1.1", Port: 53, Status: "Open", Service: "DNS", Banner: "BIND 9.16.1"},
			{Host: "192.168.1.1", Port: 3389, Status: "Filtered", Service: "RDP", Banner: ""},
		}

		// Results header
		header := fmt.Sprintf("%-15s %-6s %-10s %-10s %s",
			"HOST", "PORT", "STATUS", "SERVICE", "BANNER")
		results.WriteString(MutedStyle.Render("  " + header))
		results.WriteString("\n")
		results.WriteString(MutedStyle.Render("  " + strings.Repeat("-", 70)))
		results.WriteString("\n")

		// Results data
		for _, result := range sampleResults {
			var statusStyle lipgloss.Style
			switch result.Status {
			case "Open":
				statusStyle = SuccessStyle
			case "Closed":
				statusStyle = ErrorStyle
			case "Filtered":
				statusStyle = WarningStyle
			default:
				statusStyle = MutedStyle
			}

			resultLine := fmt.Sprintf("%-15s %-6d %-10s %-10s %s",
				result.Host,
				result.Port,
				statusStyle.Render(result.Status),
				result.Service,
				result.Banner)

			results.WriteString("  " + resultLine)
			results.WriteString("\n")
		}

		// Summary
		results.WriteString("\n")
		summary := MutedStyle.Render("  Summary: 5 open, 1 closed, 2 filtered ports found")
		results.WriteString(summary)
	} else {
		results.WriteString(MutedStyle.Render("  No scan results yet. Start a scan to see results here."))
	}

	return results.String()
}
