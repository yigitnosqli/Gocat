package ui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// App represents the main application with graceful shutdown support
type App struct {
	program *tea.Program
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	running bool
}

// NewApp creates a new application instance
func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Shutdown gracefully shuts down the application
func (app *App) Shutdown(timeout time.Duration) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.running {
		return nil
	}

	// Cancel context to signal shutdown
	app.cancel()

	// Create timeout context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Quit the program
	if app.program != nil {
		app.program.Quit()
	}

	// Wait for shutdown or timeout
	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	case <-time.After(100 * time.Millisecond): // Give some time for cleanup
		app.running = false
		return nil
	}
}

// Context returns the application context
func (app *App) Context() context.Context {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.ctx
}

// IsRunning returns whether the application is running
func (app *App) IsRunning() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.running
}

// RunTUI starts the terminal user interface
func RunTUI() error {
	// Create the model
	m := NewModel()

	// Create the program
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// RunTUIWithGracefulShutdown starts the TUI with graceful shutdown support
func RunTUIWithGracefulShutdown() error {
	app := NewApp()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start shutdown handler in a goroutine
	go func() {
		<-sigChan
		app.Shutdown(5 * time.Second)
	}()

	// Ensure cleanup of signal handling
	defer func() {
		signal.Stop(sigChan)
		close(sigChan)
	}()

	// Create the model
	m := NewModel()

	// Create the program
	app.program = tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	app.mu.Lock()
	app.running = true
	app.mu.Unlock()

	// Run the program
	if _, err := app.program.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// RunTUIWithArgs starts the TUI with command line arguments
func RunTUIWithArgs(args []string) error {
	// Parse arguments and set initial mode if needed
	m := NewModel()

	// Handle command line arguments to set initial mode
	if len(args) > 0 {
		switch args[0] {
		case "connect":
			m.mode = ModeConnect
		case "listen":
			m.mode = ModeListen
		case "chat":
			m.mode = ModeChat
		case "broker":
			m.mode = ModeBroker
		case "scan":
			m.mode = ModeScan
		case "help":
			m.mode = ModeHelp
		default:
			m.mode = ModeMenu
		}
	}

	// Create the program
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// CheckTerminalSupport checks if the terminal supports the required features
func CheckTerminalSupport() error {
	// Check if we're in a terminal
	if !isTerminal() {
		return fmt.Errorf("not running in a terminal")
	}

	// Check terminal size
	width, height := getTerminalSize()
	if width < 80 || height < 24 {
		return fmt.Errorf("terminal too small (minimum 80x24, current %dx%d)", width, height)
	}

	return nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// getTerminalSize returns the terminal dimensions
func getTerminalSize() (width, height int) {
	// This is a simplified version
	// In a real implementation, you'd use syscalls or libraries
	// to get the actual terminal size
	return 120, 40 // Default reasonable size
}

// ShowVersion displays version information in TUI style
func ShowVersion(version, commit, date string) {
	fmt.Println(TitleStyle.Render("🐱 GoCat - Network Swiss Army Knife"))
	fmt.Println()
	fmt.Println(InfoStyle.Render("Version: ") + SuccessStyle.Render(version))
	fmt.Println(InfoStyle.Render("Commit:  ") + MutedStyle.Render(commit))
	fmt.Println(InfoStyle.Render("Built:   ") + MutedStyle.Render(date))
	fmt.Println()
	fmt.Println(MutedStyle.Render("Built with ❤️  using Bubble Tea and Lip Gloss"))
}

// ShowQuickHelp displays quick help information
func ShowQuickHelp() {
	fmt.Println(HeaderStyle.Render("GoCat Quick Help"))
	fmt.Println()
	fmt.Println(InfoStyle.Render("Usage:"))
	fmt.Println("  gocat [command]")
	fmt.Println()
	fmt.Println(InfoStyle.Render("Available Commands:"))
	fmt.Println("  connect    Connect to a remote host")
	fmt.Println("  listen     Listen for incoming connections")
	fmt.Println("  chat       Start chat mode")
	fmt.Println("  broker     Start network broker")
	fmt.Println("  scan       Scan network ports")
	fmt.Println("  tui        Start interactive TUI (default)")
	fmt.Println("  help       Show help information")
	fmt.Println("  version    Show version information")
	fmt.Println()
	fmt.Println(InfoStyle.Render("Flags:"))
	fmt.Println("  -h, --help     Show help")
	fmt.Println("  -v, --version  Show version")
	fmt.Println()
	fmt.Println(MutedStyle.Render("Run 'gocat tui' or just 'gocat' to start the interactive interface."))
}
