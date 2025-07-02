package security

import (
	"net"
	"testing"
	"time"
)

func TestInputValidator_ValidateHostname(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name     string
		hostname string
		wantErr  bool
	}{
		{"valid hostname", "example.com", false},
		{"valid IP v4", "192.168.1.1", false},
		{"valid IP v6", "2001:db8::1", false},
		{"localhost", "localhost", false},
		{"empty hostname", "", true},
		{"too long hostname", string(make([]byte, 300)), true},
		{"invalid characters", "host_name", true},
		{"starts with dash", "-example.com", true},
		{"ends with dash", "example.com-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateHostname(tt.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostname() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidator_ValidatePort(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		port    string
		want    int
		wantErr bool
	}{
		{"valid port 80", "80", 80, false},
		{"valid port 443", "443", 443, false},
		{"valid port 65535", "65535", 65535, false},
		{"empty port", "", 0, true},
		{"invalid port string", "abc", 0, true},
		{"port zero", "0", 0, true},
		{"port too high", "65536", 0, true},
		{"negative port", "-1", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidatePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInputValidator_ValidatePortRange(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name      string
		portRange string
		want      []int
		wantErr   bool
	}{
		{"single port", "80", []int{80}, false},
		{"valid range", "80-82", []int{80, 81, 82}, false},
		{"same start and end", "80-80", []int{80}, false},
		{"empty range", "", nil, true},
		{"invalid format", "80-82-84", nil, true},
		{"start greater than end", "82-80", nil, true},
		{"too large range", "1-20000", nil, true},
		{"invalid start port", "abc-80", nil, true},
		{"invalid end port", "80-abc", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.ValidatePortRange(tt.portRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equalIntSlices(got, tt.want) {
				t.Errorf("ValidatePortRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInputValidator_ValidateProtocol(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name     string
		protocol string
		wantErr  bool
	}{
		{"tcp lowercase", "tcp", false},
		{"TCP uppercase", "TCP", false},
		{"udp lowercase", "udp", false},
		{"UDP uppercase", "UDP", false},
		{"invalid protocol", "http", true},
		{"empty protocol", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateProtocol(tt.protocol)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProtocol() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidator_SanitizeCommand(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		command string
		want    string
		wantErr bool
	}{
		{"safe command", "echo hello", "echo hello", false},
		{"empty command", "", "", true},
		{"too long command", string(make([]byte, 1001)), "", true},
		{"command with semicolon", "echo hello; rm -rf /", "", true},
		{"command with pipe", "cat file | rm", "", true},
		{"command with rm", "rm file.txt", "", true},
		{"command with backtick", "echo `whoami`", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.SanitizeCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	// Test allowing requests under limit
	for i := 0; i < 3; i++ {
		if !rl.Allow("test-client") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Test blocking request over limit
	if rl.Allow("test-client") {
		t.Error("Request should be blocked (over limit)")
	}

	// Test different client
	if !rl.Allow("other-client") {
		t.Error("Request from different client should be allowed")
	}

	// Wait for window to reset
	time.Sleep(time.Second + 100*time.Millisecond)

	// Test allowing requests after window reset
	if !rl.Allow("test-client") {
		t.Error("Request should be allowed after window reset")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"valid length 16", 16, false},
		{"valid length 32", 32, false},
		{"zero length", 0, true},
		{"negative length", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateSecureToken(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecureToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != tt.length*2 { // hex encoding doubles the length
					t.Errorf("GenerateSecureToken() length = %v, want %v", len(got), tt.length*2)
				}
				// Test that multiple calls produce different tokens
				got2, _ := GenerateSecureToken(tt.length)
				if got == got2 {
					t.Error("GenerateSecureToken() should produce different tokens")
				}
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"empty strings", "", "", true},
		{"one empty", "hello", "", false},
		{"different lengths", "hello", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SecureCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("SecureCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"private 10.x", "10.0.0.1", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"localhost", "127.0.0.1", true},
		{"link-local", "169.254.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"public IP 2", "1.1.1.1", false},
		{"IPv6 localhost", "::1", true},
		{"IPv6 private", "fc00::1", true},
		{"IPv6 link-local", "fe80::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Invalid IP address: %s", tt.ip)
			}
			if got := IsPrivateIP(ip); got != tt.want {
				t.Errorf("IsPrivateIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare int slices
func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
