package readline

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
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
	completions   []string
	completionFn  func(string) []string
	historyFile   string
	maxHistory    int
	filters       []Filter
	colorPrompt   bool
	promptColor   *color.Color
	ignoreCase    bool
	wordBreakChars string
}

// Filter represents an input/output filter similar to rlwrap filters
type Filter interface {
	ProcessInput(input string) string
	ProcessOutput(output string) string
	ProcessPrompt(prompt string) string
}

// CompletionResult represents a completion match
type CompletionResult struct {
	Text        string
	Description string
	Type        string
}

// NewEditor creates a new readline editor with rlwrap-like features
func NewEditor() *Editor {
	fd := int(os.Stdin.Fd())
	originalState, _ := term.GetState(fd)

	return &Editor{
		reader:         bufio.NewReader(os.Stdin),
		history:        make([]string, 0),
		prompt:         ">> ",
		historyIndex:   -1,
		currentLine:    make([]rune, 0),
		cursorPos:      0,
		originalState:  originalState,
		fd:             fd,
		completions:    make([]string, 0),
		maxHistory:     1000,
		filters:        make([]Filter, 0),
		colorPrompt:    true,
		promptColor:    color.New(color.FgYellow, color.Bold),
		wordBreakChars: " \t\n\r\f\v",
	}
}

// SetPrompt sets the prompt string with optional color
func (e *Editor) SetPrompt(prompt string) {
	e.prompt = prompt
}

// SetPromptColor enables/disables colored prompt
func (e *Editor) SetPromptColor(enabled bool, c *color.Color) {
	e.colorPrompt = enabled
	if c != nil {
		e.promptColor = c
	}
}

// SetHistoryFile sets the history file path for persistent history
func (e *Editor) SetHistoryFile(path string) {
	e.historyFile = path
	e.loadHistory()
}

// SetMaxHistory sets the maximum number of history entries
func (e *Editor) SetMaxHistory(max int) {
	e.maxHistory = max
}

// AddFilter adds an input/output filter
func (e *Editor) AddFilter(filter Filter) {
	e.filters = append(e.filters, filter)
}

// SetCompletionFunction sets a custom completion function
func (e *Editor) SetCompletionFunction(fn func(string) []string) {
	e.completionFn = fn
}

// AddCompletion adds a static completion word
func (e *Editor) AddCompletion(word string) {
	e.completions = append(e.completions, word)
}

// SetCompletions sets the list of completion words
func (e *Editor) SetCompletions(words []string) {
	e.completions = make([]string, len(words))
	copy(e.completions, words)
}

// Readline reads a line from stdin with rlwrap-like features
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
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
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

	// Display prompt with filters and colors
	prompt := e.processPrompt(e.prompt)
	e.displayPrompt(prompt)

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
				processedLine := e.processInput(line)
				if processedLine != "" {
					e.AddHistoryEntry(processedLine)
				}
				return processedLine, nil

			case 3: // Ctrl+C
				fmt.Print("\n")
				return "", fmt.Errorf("interrupted")

			case 4: // Ctrl+D (EOF)
				if len(e.currentLine) == 0 {
					return "", io.EOF
				}

			case 9: // Tab (completion)
				e.handleCompletion()

			case 18: // Ctrl+R (reverse search)
				e.reverseSearch()

			case 12: // Ctrl+L (clear screen)
				e.clearScreen()

			case 1: // Ctrl+A (beginning of line)
				e.cursorPos = 0
				e.refreshLine()

			case 5: // Ctrl+E (end of line)
				e.cursorPos = len(e.currentLine)
				e.refreshLine()

			case 11: // Ctrl+K (kill to end of line)
				e.currentLine = e.currentLine[:e.cursorPos]
				e.refreshLine()

			case 21: // Ctrl+U (kill to beginning of line)
				e.currentLine = e.currentLine[e.cursorPos:]
				e.cursorPos = 0
				e.refreshLine()

			case 23: // Ctrl+W (kill word backward)
				e.killWordBackward()

			case 127, 8: // Backspace
				e.backspace()

			case 27: // Escape sequence (arrow keys, etc.)
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

// AddHistoryEntry adds a command to history with deduplication
func (e *Editor) AddHistoryEntry(entry string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}

	// Remove duplicate if it exists
	for i, h := range e.history {
		if h == entry {
			e.history = append(e.history[:i], e.history[i+1:]...)
			break
		}
	}

	// Add to end
	e.history = append(e.history, entry)

	// Trim history if too long
	if len(e.history) > e.maxHistory {
		e.history = e.history[len(e.history)-e.maxHistory:]
	}

	// Save to file if configured
	if e.historyFile != "" {
		e.saveHistory()
	}
}

// GetHistory returns a copy of the command history
func (e *Editor) GetHistory() []string {
	historyCopy := make([]string, len(e.history))
	copy(historyCopy, e.history)
	return historyCopy
}

// ClearHistory clears the command history
func (e *Editor) ClearHistory() {
	e.history = make([]string, 0)
	if e.historyFile != "" {
		e.saveHistory()
	}
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

// readChar reads a single character from stdin with timeout
func (e *Editor) readChar() (byte, error) {
	buf := make([]byte, 1)
	_, err := os.Stdin.Read(buf)
	return buf[0], err
}

// displayPrompt displays the prompt with optional coloring
func (e *Editor) displayPrompt(prompt string) {
	if e.colorPrompt && e.promptColor != nil {
		e.promptColor.Print(prompt)
	} else {
		fmt.Print(prompt)
	}
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

// killWordBackward removes the word before the cursor
func (e *Editor) killWordBackward() {
	if e.cursorPos == 0 {
		return
	}

	// Find the start of the current word
	start := e.cursorPos - 1
	for start > 0 && strings.ContainsRune(e.wordBreakChars, e.currentLine[start]) {
		start--
	}
	for start > 0 && !strings.ContainsRune(e.wordBreakChars, e.currentLine[start-1]) {
		start--
	}

	// Remove the word
	e.currentLine = append(e.currentLine[:start], e.currentLine[e.cursorPos:]...)
	e.cursorPos = start
	e.refreshLine()
}

// refreshLine redraws the current line with proper cursor positioning
func (e *Editor) refreshLine() {
	// Clear current line
	fmt.Print("\r\033[K")
	// Redraw prompt and line
	prompt := e.processPrompt(e.prompt)
	e.displayPrompt(prompt)
	fmt.Print(string(e.currentLine))
	// Position cursor
	if e.cursorPos < len(e.currentLine) {
		fmt.Printf("\033[%dD", len(e.currentLine)-e.cursorPos)
	}
}

// clearScreen clears the terminal screen
func (e *Editor) clearScreen() {
	fmt.Print("\033[2J\033[H")
	prompt := e.processPrompt(e.prompt)
	e.displayPrompt(prompt)
	fmt.Print(string(e.currentLine))
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

	switch ch1 {
	case '[':
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
		case 'H': // Home
			e.cursorPos = 0
			e.refreshLine()
			return true
		case 'F': // End
			e.cursorPos = len(e.currentLine)
			e.refreshLine()
			return true
		case '3': // Delete key
			// Read the ~ character
			if ch3, err := e.readChar(); err == nil && ch3 == '~' {
				e.deleteChar()
			}
			return true
		case '1':
			// Home key: ESC[1~
			if ch3, err := e.readChar(); err == nil && ch3 == '~' {
				e.cursorPos = 0
				e.refreshLine()
			}
			return true
		case '4':
			// End key: ESC[4~
			if ch3, err := e.readChar(); err == nil && ch3 == '~' {
				e.cursorPos = len(e.currentLine)
				e.refreshLine()
			}
			return true
		}
	case 'O':
		// Read the third character
		ch2, err := e.readChar()
		if err != nil {
			return false
		}

		switch ch2 {
		case 'H': // Home
			e.cursorPos = 0
			e.refreshLine()
			return true
		case 'F': // End
			e.cursorPos = len(e.currentLine)
			e.refreshLine()
			return true
		}
	case 'b': // Alt+B (word backward)
		e.wordBackward()
		return true
	case 'f': // Alt+F (word forward)
		e.wordForward()
		return true
	}
	return false
}

// deleteChar deletes the character at the cursor position
func (e *Editor) deleteChar() {
	if e.cursorPos < len(e.currentLine) {
		e.currentLine = append(e.currentLine[:e.cursorPos], e.currentLine[e.cursorPos+1:]...)
		e.refreshLine()
	}
}

// wordBackward moves cursor to the beginning of the previous word
func (e *Editor) wordBackward() {
	if e.cursorPos == 0 {
		return
	}

	// Skip whitespace
	for e.cursorPos > 0 && strings.ContainsRune(e.wordBreakChars, e.currentLine[e.cursorPos-1]) {
		e.cursorPos--
	}
	// Skip word characters
	for e.cursorPos > 0 && !strings.ContainsRune(e.wordBreakChars, e.currentLine[e.cursorPos-1]) {
		e.cursorPos--
	}
	e.refreshLine()
}

// wordForward moves cursor to the beginning of the next word
func (e *Editor) wordForward() {
	if e.cursorPos >= len(e.currentLine) {
		return
	}

	// Skip word characters
	for e.cursorPos < len(e.currentLine) && !strings.ContainsRune(e.wordBreakChars, e.currentLine[e.cursorPos]) {
		e.cursorPos++
	}
	// Skip whitespace
	for e.cursorPos < len(e.currentLine) && strings.ContainsRune(e.wordBreakChars, e.currentLine[e.cursorPos]) {
		e.cursorPos++
	}
	e.refreshLine()
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

// cursorLeft moves cursor left
func (e *Editor) cursorLeft() {
	if e.cursorPos > 0 {
		e.cursorPos--
		e.refreshLine()
	}
}

// cursorRight moves cursor right
func (e *Editor) cursorRight() {
	if e.cursorPos < len(e.currentLine) {
		e.cursorPos++
		e.refreshLine()
	}
}

// handleCompletion handles tab completion
func (e *Editor) handleCompletion() {
	currentWord := e.getCurrentWord()
	if currentWord == "" {
		return
	}

	matches := e.getCompletions(currentWord)
	if len(matches) == 0 {
		return
	}

	if len(matches) == 1 {
		// Single match - complete it
		e.completeWord(matches[0])
	} else {
		// Multiple matches - show them
		e.showCompletions(matches)
	}
}

// getCurrentWord gets the current word being typed
func (e *Editor) getCurrentWord() string {
	if e.cursorPos == 0 {
		return ""
	}

	// Find word boundaries
	start := e.cursorPos - 1
	for start > 0 && !strings.ContainsRune(e.wordBreakChars, e.currentLine[start-1]) {
		start--
	}

	return string(e.currentLine[start:e.cursorPos])
}

// getCompletions gets completion matches for a word
func (e *Editor) getCompletions(word string) []string {
	var matches []string

	// Use custom completion function if available
	if e.completionFn != nil {
		matches = append(matches, e.completionFn(word)...)
	}

	// Add static completions
	for _, completion := range e.completions {
		if e.matchesCompletion(word, completion) {
			matches = append(matches, completion)
		}
	}

	// Add history-based completions
	for _, histEntry := range e.history {
		words := strings.Fields(histEntry)
		for _, histWord := range words {
			if e.matchesCompletion(word, histWord) {
				// Avoid duplicates
				found := false
				for _, match := range matches {
					if match == histWord {
						found = true
						break
					}
				}
				if !found {
					matches = append(matches, histWord)
				}
			}
		}
	}

	// Sort matches
	sort.Strings(matches)
	return matches
}

// matchesCompletion checks if a word matches a completion
func (e *Editor) matchesCompletion(word, completion string) bool {
	if e.ignoreCase {
		return strings.HasPrefix(strings.ToLower(completion), strings.ToLower(word))
	}
	return strings.HasPrefix(completion, word)
}

// completeWord completes the current word
func (e *Editor) completeWord(completion string) {
	currentWord := e.getCurrentWord()
	if currentWord == "" {
		return
	}

	// Replace current word with completion
	start := e.cursorPos - len([]rune(currentWord))
	e.currentLine = append(e.currentLine[:start], append([]rune(completion), e.currentLine[e.cursorPos:]...)...)
	e.cursorPos = start + len([]rune(completion))
	e.refreshLine()
}

// showCompletions displays available completions
func (e *Editor) showCompletions(matches []string) {
	fmt.Print("\n")
	for i, match := range matches {
		fmt.Printf("%s", match)
		if i < len(matches)-1 {
			fmt.Print("  ")
		}
		if (i+1)%8 == 0 { // 8 completions per line
			fmt.Print("\n")
		}
	}
	if len(matches)%8 != 0 {
		fmt.Print("\n")
	}
	e.refreshLine()
}

// reverseSearch implements Ctrl+R reverse history search
func (e *Editor) reverseSearch() {
	fmt.Print("\n(reverse-i-search): ")
	searchTerm := ""
	searchResults := []string{}

	for {
		ch, err := e.readChar()
		if err != nil {
			break
		}

		switch ch {
		case 13: // Enter - select current result
			if len(searchResults) > 0 {
				e.currentLine = []rune(searchResults[0])
				e.cursorPos = len(e.currentLine)
			}
			fmt.Print("\n")
			e.refreshLine()
			return

		case 27: // Escape - cancel search
			fmt.Print("\n")
			e.refreshLine()
			return

		case 127, 8: // Backspace
			if len(searchTerm) > 0 {
				searchTerm = searchTerm[:len(searchTerm)-1]
			}

		default:
			if unicode.IsPrint(rune(ch)) {
				searchTerm += string(rune(ch))
			}
		}

		// Update search results
		searchResults = e.searchHistory(searchTerm)

		// Display current search
		fmt.Print("\r\033[K")
		fmt.Printf("(reverse-i-search): %s", searchTerm)
		if len(searchResults) > 0 {
			fmt.Printf(" [%s]", searchResults[0])
		}
	}
}

// searchHistory searches through history for matching entries
func (e *Editor) searchHistory(term string) []string {
	var results []string
	if term == "" {
		return results
	}

	// Search backwards through history
	for i := len(e.history) - 1; i >= 0; i-- {
		entry := e.history[i]
		if e.ignoreCase {
			if strings.Contains(strings.ToLower(entry), strings.ToLower(term)) {
				results = append(results, entry)
			}
		} else {
			if strings.Contains(entry, term) {
				results = append(results, entry)
			}
		}
	}

	return results
}

// processInput processes input through filters
func (e *Editor) processInput(input string) string {
	result := input
	for _, filter := range e.filters {
		result = filter.ProcessInput(result)
	}
	return result
}

// processPrompt processes prompt through filters
func (e *Editor) processPrompt(prompt string) string {
	result := prompt
	for _, filter := range e.filters {
		result = filter.ProcessPrompt(result)
	}
	return result
}

// loadHistory loads history from file
func (e *Editor) loadHistory() {
	if e.historyFile == "" {
		return
	}

	file, err := os.Open(e.historyFile)
	if err != nil {
		return // File doesn't exist yet
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			e.history = append(e.history, line)
		}
	}

	// Trim to max history
	if len(e.history) > e.maxHistory {
		e.history = e.history[len(e.history)-e.maxHistory:]
	}
}

// saveHistory saves history to file
func (e *Editor) saveHistory() {
	if e.historyFile == "" {
		return
	}

	file, err := os.Create(e.historyFile)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range e.history {
		fmt.Fprintln(writer, entry)
	}
	writer.Flush()
}

// Simple password filter that hides password prompts
type PasswordFilter struct {
	patterns []*regexp.Regexp
}

// NewPasswordFilter creates a new password filter
func NewPasswordFilter(patterns []string) *PasswordFilter {
	pf := &PasswordFilter{
		patterns: make([]*regexp.Regexp, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			pf.patterns = append(pf.patterns, re)
		}
	}

	return pf
}

func (pf *PasswordFilter) ProcessInput(input string) string {
	// Don't add password inputs to history
	for _, pattern := range pf.patterns {
		if pattern.MatchString(input) {
			return "" // Don't save to history
		}
	}
	return input
}

func (pf *PasswordFilter) ProcessOutput(output string) string {
	return output // Don't modify output
}

func (pf *PasswordFilter) ProcessPrompt(prompt string) string {
	return prompt // Don't modify prompt
}

// Utility functions for rlwrap-like behavior

// SetIgnoreCase sets case sensitivity for completions and search
func (e *Editor) SetIgnoreCase(ignore bool) {
	e.ignoreCase = ignore
}

// SetWordBreakChars sets the characters that break words
func (e *Editor) SetWordBreakChars(chars string) {
	e.wordBreakChars = chars
}

// GetTerminalSize returns the current terminal size
func GetTerminalSize() (width, height int, err error) {
	fd := int(os.Stdout.Fd())
	width, height, err = term.GetSize(fd)
	return
}

// IsTerminal checks if the given file descriptor is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// FormatDuration formats a duration for display
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// ParsePortRange parses a port range string like "80-443" or "22,80,443"
func ParsePortRange(portRange string) ([]int, error) {
	var ports []int

	// Handle comma-separated ports
	if strings.Contains(portRange, ",") {
		parts := strings.Split(portRange, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if port, err := strconv.Atoi(part); err == nil {
				if port > 0 && port <= 65535 {
					ports = append(ports, port)
				}
			}
		}
		return ports, nil
	}

	// Handle range like "80-443"
	if strings.Contains(portRange, "-") {
		parts := strings.Split(portRange, "-")
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && start <= end && start > 0 && end <= 65535 {
				for port := start; port <= end; port++ {
					ports = append(ports, port)
				}
				return ports, nil
			}
		}
	}

	// Handle single port
	if port, err := strconv.Atoi(strings.TrimSpace(portRange)); err == nil {
		if port > 0 && port <= 65535 {
			ports = append(ports, port)
			return ports, nil
		}
	}

	return nil, fmt.Errorf("invalid port range: %s", portRange)
}
