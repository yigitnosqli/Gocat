package readline

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

// Editor represents a simple readline editor
type Editor struct {
	reader *bufio.Reader
	history []string
	prompt string
}

// NewEditor creates a new readline editor
func NewEditor() *Editor {
	return &Editor{
		reader: bufio.NewReader(os.Stdin),
		history: make([]string, 0),
		prompt: ">> ",
	}
}

// SetPrompt sets the prompt string
func (e *Editor) SetPrompt(prompt string) {
	e.prompt = prompt
}

// Readline reads a line from stdin with the prompt
func (e *Editor) Readline() (string, error) {
	color.Yellow(e.prompt)
	line, err := e.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", io.EOF
		}
		return "", fmt.Errorf("failed to read line: %v", err)
	}

	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")

	if line != "" {
		e.AddHistoryEntry(line)
	}

	return line, nil
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