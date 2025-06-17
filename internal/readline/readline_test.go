package readline

import (
	"bufio"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestNewEditor(t *testing.T) {
	editor := NewEditor()

	if editor == nil {
		t.Error("NewEditor() returned nil")
	}

	if editor.reader == nil {
		t.Error("NewEditor() created editor with nil reader")
	}

	if editor.history == nil {
		t.Error("NewEditor() created editor with nil history")
	}

	if len(editor.history) != 0 {
		t.Errorf("NewEditor() created editor with non-empty history, got %d entries", len(editor.history))
	}

	if editor.prompt != ">> " {
		t.Errorf("NewEditor() created editor with wrong default prompt, got %q", editor.prompt)
	}
}

func TestSetPrompt(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{
			name:   "simple prompt",
			prompt: "$ ",
		},
		{
			name:   "complex prompt",
			prompt: "[user@host]$ ",
		},
		{
			name:   "empty prompt",
			prompt: "",
		},
		{
			name:   "prompt with colors",
			prompt: "\033[32m$ \033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := NewEditor()
			editor.SetPrompt(tt.prompt)

			if editor.prompt != tt.prompt {
				t.Errorf("SetPrompt() = %q, want %q", editor.prompt, tt.prompt)
			}
		})
	}
}

func TestAddHistoryEntry(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
	}{
		{
			name:    "single entry",
			entries: []string{"ls"},
		},
		{
			name:    "multiple entries",
			entries: []string{"ls", "cd /tmp", "pwd"},
		},
		{
			name:    "empty entry",
			entries: []string{""},
		},
		{
			name:    "entries with spaces",
			entries: []string{"ls -la", "grep pattern file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := NewEditor()

			for _, entry := range tt.entries {
				editor.AddHistoryEntry(entry)
			}

			history := editor.GetHistory()
			if len(history) != len(tt.entries) {
				t.Errorf("AddHistoryEntry() resulted in %d entries, want %d", len(history), len(tt.entries))
			}

			if !reflect.DeepEqual(history, tt.entries) {
				t.Errorf("AddHistoryEntry() history = %v, want %v", history, tt.entries)
			}
		})
	}
}

func TestGetHistory(t *testing.T) {
	editor := NewEditor()

	// Test empty history
	history := editor.GetHistory()
	if len(history) != 0 {
		t.Errorf("GetHistory() on empty editor = %v, want empty slice", history)
	}

	// Add some entries
	entries := []string{"command1", "command2", "command3"}
	for _, entry := range entries {
		editor.AddHistoryEntry(entry)
	}

	// Test populated history
	history = editor.GetHistory()
	if !reflect.DeepEqual(history, entries) {
		t.Errorf("GetHistory() = %v, want %v", history, entries)
	}

	// Test that returned slice is a copy (modification shouldn't affect original)
	history[0] = "modified"
	originalHistory := editor.GetHistory()
	if originalHistory[0] == "modified" {
		t.Error("GetHistory() returned a reference to internal slice, should return a copy")
	}
}

func TestReadlineWithInput(t *testing.T) {
	// Disable color output for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple command",
			input:    "ls\n",
			expected: "ls",
		},
		{
			name:     "command with args",
			input:    "ls -la /tmp\n",
			expected: "ls -la /tmp",
		},
		{
			name:     "empty line",
			input:    "\n",
			expected: "",
		},
		{
			name:     "line with carriage return",
			input:    "test\r\n",
			expected: "test",
		},
		{
			name:     "line with spaces",
			input:    "  spaced command  \n",
			expected: "  spaced command  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := NewEditor()
			// Replace the reader with a string reader
			editor.reader = bufio.NewReader(strings.NewReader(tt.input))

			result, err := editor.Readline()
			if err != nil {
				t.Errorf("Readline() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Readline() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestReadlineEOF(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	editor := NewEditor()
	// Create a reader that immediately returns EOF
	editor.reader = bufio.NewReader(strings.NewReader(""))

	_, err := editor.Readline()
	if err != io.EOF {
		t.Errorf("Readline() on EOF should return io.EOF, got %v", err)
	}
}

func TestReadlineHistoryIntegration(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	editor := NewEditor()

	// Test that non-empty lines are added to history
	input := "test command\n"
	editor.reader = bufio.NewReader(strings.NewReader(input))

	result, err := editor.Readline()
	if err != nil {
		t.Errorf("Readline() error = %v", err)
	}

	if result != "test command" {
		t.Errorf("Readline() = %q, want %q", result, "test command")
	}

	history := editor.GetHistory()
	if len(history) != 1 {
		t.Errorf("History should have 1 entry, got %d", len(history))
	}

	if history[0] != "test command" {
		t.Errorf("History[0] = %q, want %q", history[0], "test command")
	}
}

func TestReadlineEmptyLineHistory(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	editor := NewEditor()

	// Test that empty lines are not added to history
	input := "\n"
	editor.reader = bufio.NewReader(strings.NewReader(input))

	result, err := editor.Readline()
	if err != nil {
		t.Errorf("Readline() error = %v", err)
	}

	if result != "" {
		t.Errorf("Readline() = %q, want empty string", result)
	}

	history := editor.GetHistory()
	if len(history) != 0 {
		t.Errorf("History should be empty for empty line, got %d entries", len(history))
	}
}

func TestMultipleReadlines(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	editor := NewEditor()
	commands := []string{"ls", "pwd", "cd /tmp", ""}
	input := strings.Join(commands, "\n") + "\n"
	editor.reader = bufio.NewReader(strings.NewReader(input))

	var results []string
	for i := 0; i < len(commands); i++ {
		result, err := editor.Readline()
		if err != nil {
			t.Errorf("Readline() %d error = %v", i, err)
			continue
		}
		results = append(results, result)
	}

	if !reflect.DeepEqual(results, commands) {
		t.Errorf("Multiple Readline() = %v, want %v", results, commands)
	}

	// Check history (empty lines should not be included)
	expectedHistory := []string{"ls", "pwd", "cd /tmp"}
	history := editor.GetHistory()
	if !reflect.DeepEqual(history, expectedHistory) {
		t.Errorf("History after multiple Readline() = %v, want %v", history, expectedHistory)
	}
}

// Benchmark tests
func BenchmarkNewEditor(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEditor()
	}
}

func BenchmarkAddHistoryEntry(b *testing.B) {
	editor := NewEditor()
	command := "test command"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		editor.AddHistoryEntry(command)
	}
}

func BenchmarkGetHistory(b *testing.B) {
	editor := NewEditor()
	// Add some history entries
	for i := 0; i < 100; i++ {
		editor.AddHistoryEntry("command")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = editor.GetHistory()
	}
}

func BenchmarkReadline(b *testing.B) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	command := "test command\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		editor := NewEditor()
		editor.reader = bufio.NewReader(strings.NewReader(command))
		_, _ = editor.Readline()
	}
}

func BenchmarkSetPrompt(b *testing.B) {
	editor := NewEditor()
	prompt := "$ "

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		editor.SetPrompt(prompt)
	}
}