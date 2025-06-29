package readline

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// Editor represents a readline editor with rlwrap-like features
type Editor struct {
	reader        *bufio.Reader
	history       []string
	prompt        string
	historyIndex  int
	currentLine   []rune
	cursorPos     int
	originalState *term.State
	fd            int
}

// NewEditor creates a new readline editor
func NewEditor() *Editor {
	fd := int(os.Stdin.Fd())
	originalState, _ := term.GetState(fd)

	return &Editor{
		reader:        bufio.NewReader(os.Stdin),
		history:       make([]string, 0),
		prompt:        ">> ",
		historyIndex:  -1,
		currentLine:   make([]rune, 0),
		cursorPos:     0,
		originalState: originalState,
		fd:            fd,
	}
}

// SetPrompt sets the prompt string
func (e *Editor) SetPrompt(prompt string) {
	e.prompt = prompt
}

// Readline reads a line from stdin with the prompt and rlwrap-like features
func (e *Editor) Readline() (string, error) {
	// Check if we're in a terminal (interactive mode) or not (test mode)
	if !term.IsTerminal(e.fd) {
		// Non-interactive mode (tests) - use simple line reading
		line, err := e.reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		// Remove trailing newline and carriage return
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			e.AddHistoryEntry(line)
		}
		return line, nil
	}

	// Interactive mode - use rlwrap-like features
	// Setup signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)

	// Reset editor state
	e.currentLine = make([]rune, 0)
	e.cursorPos = 0
	e.historyIndex = -1

	// Enable raw mode for character-by-character input
	if err := e.enableRawMode(); err != nil {
		return "", fmt.Errorf("failed to enable raw mode: %v", err)
	}
	defer e.disableRawMode()

	// Display prompt
	color.Yellow(e.prompt)

	for {
		select {
		case <-sigChan:
			// Handle Ctrl+C
			fmt.Print("\n")
			return "", fmt.Errorf("interrupted")
		default:
			// Read character
			ch, err := e.readChar()
			if err != nil {
				if err == io.EOF {
					return "", io.EOF
				}
				return "", err
			}

			switch ch {
			case 13: // Enter
				fmt.Print("\n")
				line := string(e.currentLine)
				if line != "" {
					e.AddHistoryEntry(line)
				}
				return line, nil

			case 3: // Ctrl+C
				fmt.Print("\n")
				return "", fmt.Errorf("interrupted")

			case 4: // Ctrl+D (EOF)
				if len(e.currentLine) == 0 {
					return "", io.EOF
				}

			case 18: // Ctrl+R (reverse search)
				e.reverseSearch()

			case 127, 8: // Backspace
				e.backspace()

			case 27: // Escape sequence (arrow keys)
				if e.handleEscapeSequence() {
					continue
				}

			default:
				if unicode.IsPrint(rune(ch)) {
					e.insertChar(rune(ch))
				}
			}
		}
	}
}

// AddHistoryEntry adds a command to history
func (e *Editor) AddHistoryEntry(entry string) {
	e.history = append(e.history, entry)
}

// GetHistory returns the command history
func (e *Editor) GetHistory() []string {
	historyCopy := make([]string, len(e.history))
	copy(historyCopy, e.history)
	return historyCopy
}

// enableRawMode enables raw terminal mode for character-by-character input
func (e *Editor) enableRawMode() error {
	_, err := term.MakeRaw(e.fd)
	return err
}

// disableRawMode restores original terminal mode
func (e *Editor) disableRawMode() {
	if e.originalState != nil {
		if err := term.Restore(e.fd, e.originalState); err != nil {
			// Log error but don't fail, as this is cleanup code
			fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal state: %v\n", err)
		}
	}
}

// readChar reads a single character from stdin
func (e *Editor) readChar() (byte, error) {
	buf := make([]byte, 1)
	_, err := os.Stdin.Read(buf)
	return buf[0], err
}

// insertChar inserts a character at the current cursor position
func (e *Editor) insertChar(ch rune) {
	if e.cursorPos == len(e.currentLine) {
		e.currentLine = append(e.currentLine, ch)
	} else {
		e.currentLine = append(e.currentLine[:e.cursorPos+1], e.currentLine[e.cursorPos:]...)
		e.currentLine[e.cursorPos] = ch
	}
	e.cursorPos++
	e.refreshLine()
}

// backspace removes the character before the cursor
func (e *Editor) backspace() {
	if e.cursorPos > 0 {
		e.currentLine = append(e.currentLine[:e.cursorPos-1], e.currentLine[e.cursorPos:]...)
		e.cursorPos--
		e.refreshLine()
	}
}

// refreshLine redraws the current line
func (e *Editor) refreshLine() {
	// Clear current line
	fmt.Print("\r\033[K")
	// Redraw prompt and line
	color.Yellow(e.prompt)
	fmt.Print(string(e.currentLine))
	// Position cursor
	if e.cursorPos < len(e.currentLine) {
		fmt.Printf("\033[%dD", len(e.currentLine)-e.cursorPos)
	}
}

// handleEscapeSequence handles arrow keys and other escape sequences
func (e *Editor) handleEscapeSequence() bool {
	// Read the next character
	ch1, err := e.readChar()
	if err != nil {
		return false
	}

	if ch1 == '[' {
		// Read the third character
		ch2, err := e.readChar()
		if err != nil {
			return false
		}

		switch ch2 {
		case 'A': // Up arrow
			e.historyUp()
			return true
		case 'B': // Down arrow
			e.historyDown()
			return true
		case 'C': // Right arrow
			e.cursorRight()
			return true
		case 'D': // Left arrow
			e.cursorLeft()
			return true
		}
	}
	return false
}

// historyUp navigates to previous history entry
func (e *Editor) historyUp() {
	if len(e.history) == 0 {
		return
	}

	if e.historyIndex == -1 {
		e.historyIndex = len(e.history) - 1
	} else if e.historyIndex > 0 {
		e.historyIndex--
	}

	e.currentLine = []rune(e.history[e.historyIndex])
	e.cursorPos = len(e.currentLine)
	e.refreshLine()
}

// historyDown navigates to next history entry
func (e *Editor) historyDown() {
	if len(e.history) == 0 || e.historyIndex == -1 {
		return
	}

	if e.historyIndex < len(e.history)-1 {
		e.historyIndex++
		e.currentLine = []rune(e.history[e.historyIndex])
	} else {
		e.historyIndex = -1
		e.currentLine = make([]rune, 0)
	}

	e.cursorPos = len(e.currentLine)
	e.refreshLine()
}

// cursorLeft moves cursor to the left
func (e *Editor) cursorLeft() {
	if e.cursorPos > 0 {
		e.cursorPos--
		fmt.Print("\033[D")
	}
}

// cursorRight moves cursor to the right
func (e *Editor) cursorRight() {
	if e.cursorPos < len(e.currentLine) {
		e.cursorPos++
		fmt.Print("\033[C")
	}
}

// reverseSearch implements Ctrl+R functionality for searching history
func (e *Editor) reverseSearch() {
	if len(e.history) == 0 {
		return
	}

	searchTerm := make([]rune, 0)
	matchIndex := -1

	for {
		// Display search prompt
		fmt.Print("\r\033[K")
		color.Cyan("(reverse-i-search)`%s': ", string(searchTerm))

		// Find matching history entry
		if len(searchTerm) > 0 {
			for i := len(e.history) - 1; i >= 0; i-- {
				if strings.Contains(e.history[i], string(searchTerm)) {
					matchIndex = i
					break
				}
			}
		}

		// Display matched line
		if matchIndex >= 0 {
			fmt.Print(e.history[matchIndex])
		}

		// Read character
		ch, err := e.readChar()
		if err != nil {
			break
		}

		switch ch {
		case 13: // Enter - accept current match
			if matchIndex >= 0 {
				e.currentLine = []rune(e.history[matchIndex])
				e.cursorPos = len(e.currentLine)
			}
			fmt.Print("\n")
			return

		case 3, 7: // Ctrl+C or Ctrl+G - cancel search
			fmt.Print("\r\033[K")
			color.Yellow(e.prompt)
			fmt.Print(string(e.currentLine))
			return

		case 18: // Ctrl+R - find next match
			if matchIndex > 0 {
				for i := matchIndex - 1; i >= 0; i-- {
					if strings.Contains(e.history[i], string(searchTerm)) {
						matchIndex = i
						break
					}
				}
			}

		case 127, 8: // Backspace
			if len(searchTerm) > 0 {
				searchTerm = searchTerm[:len(searchTerm)-1]
				matchIndex = -1
			}

		default:
			if unicode.IsPrint(rune(ch)) {
				searchTerm = append(searchTerm, rune(ch))
				matchIndex = -1
			}
		}
	}

	// Restore prompt
	fmt.Print("\r\033[K")
	color.Yellow(e.prompt)
	fmt.Print(string(e.currentLine))
}
