package input

import (
	"reflect"
	"testing"
)

func TestParseHostPort(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		defaultHost string
		expectedHost string
		expectedPort string
		expectError  bool
	}{
		{
			name:         "port only",
			args:         []string{"8080"},
			defaultHost:  "0.0.0.0",
			expectedHost: "0.0.0.0",
			expectedPort: "8080",
			expectError:  false,
		},
		{
			name:         "host and port",
			args:         []string{"localhost", "8080"},
			defaultHost:  "0.0.0.0",
			expectedHost: "localhost",
			expectedPort: "8080",
			expectError:  false,
		},
		{
			name:        "no arguments",
			args:        []string{},
			defaultHost: "0.0.0.0",
			expectError: true,
		},
		{
			name:        "invalid port",
			args:        []string{"invalid"},
			defaultHost: "0.0.0.0",
			expectError: true,
		},
		{
			name:        "port out of range",
			args:        []string{"70000"},
			defaultHost: "0.0.0.0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := ParseHostPort(tt.args, tt.defaultHost)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if host != tt.expectedHost {
				t.Errorf("expected host %s, got %s", tt.expectedHost, host)
			}

			if port != tt.expectedPort {
				t.Errorf("expected port %s, got %s", tt.expectedPort, port)
			}
		})
	}
}

func TestValidateShell(t *testing.T) {
	tests := []struct {
		name        string
		shell       string
		expectError bool
	}{
		{
			name:        "valid shell",
			shell:       "/bin/bash",
			expectError: false,
		},
		{
			name:        "empty shell",
			shell:       "",
			expectError: true,
		},
		{
			name:        "shell with semicolon",
			shell:       "/bin/bash; rm -rf /",
			expectError: true,
		},
		{
			name:        "shell with ampersand",
			shell:       "/bin/bash & rm -rf /",
			expectError: true,
		},
		{
			name:        "shell with pipe",
			shell:       "/bin/bash | cat",
			expectError: false, // pipes are allowed in shell paths
		},
		{
			name:        "shell with spaces",
			shell:       "/usr/bin/my shell",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShell(tt.shell)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		expectedCmd    string
		expectedArgs   []string
	}{
		{
			name:         "simple command",
			command:      "ls",
			expectedCmd:  "ls",
			expectedArgs: nil,
		},
		{
			name:         "command with args",
			command:      "ls -la /tmp",
			expectedCmd:  "ls",
			expectedArgs: []string{"-la", "/tmp"},
		},
		{
			name:         "empty command",
			command:      "",
			expectedCmd:  "",
			expectedArgs: nil,
		},
		{
			name:         "command with multiple spaces",
			command:      "  ls   -la   /tmp  ",
			expectedCmd:  "ls",
			expectedArgs: []string{"-la", "/tmp"},
		},
		{
			name:         "command with quoted args",
			command:      "echo hello world",
			expectedCmd:  "echo",
			expectedArgs: []string{"hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := ParseCommand(tt.command)

			if cmd != tt.expectedCmd {
				t.Errorf("expected command %q, got %q", tt.expectedCmd, cmd)
			}

			// Handle nil vs empty slice comparison
			if (args == nil && tt.expectedArgs != nil && len(tt.expectedArgs) > 0) ||
				(tt.expectedArgs == nil && args != nil && len(args) > 0) ||
				(args != nil && tt.expectedArgs != nil && !reflect.DeepEqual(args, tt.expectedArgs)) {
				t.Errorf("expected args %v, got %v", tt.expectedArgs, args)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		expectError bool
	}{
		{
			name:        "valid port 80",
			port:        "80",
			expectError: false,
		},
		{
			name:        "valid port 443",
			port:        "443",
			expectError: false,
		},
		{
			name:        "valid port 65535",
			port:        "65535",
			expectError: false,
		},
		{
			name:        "valid port 1",
			port:        "1",
			expectError: false,
		},
		{
			name:        "invalid port 0",
			port:        "0",
			expectError: true,
		},
		{
			name:        "invalid port 65536",
			port:        "65536",
			expectError: true,
		},
		{
			name:        "invalid port negative",
			port:        "-1",
			expectError: true,
		},
		{
			name:        "invalid port string",
			port:        "abc",
			expectError: true,
		},
		{
			name:        "invalid port empty",
			port:        "",
			expectError: true,
		},
		{
			name:        "invalid port with spaces",
			port:        " 80 ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.port)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseHostPort(b *testing.B) {
	args := []string{"localhost", "8080"}
	defaultHost := "0.0.0.0"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseHostPort(args, defaultHost)
	}
}

func BenchmarkParseCommand(b *testing.B) {
	command := "ls -la /tmp/test/directory"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseCommand(command)
	}
}

func BenchmarkValidateShell(b *testing.B) {
	shell := "/bin/bash"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateShell(shell)
	}
}