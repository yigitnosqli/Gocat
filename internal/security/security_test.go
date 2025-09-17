package security

import (
	"crypto/tls"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ibrahmsql/gocat/internal/errors"
)

func TestInputValidator_ValidateHostname(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name     string
		hostname string
		wantErr  bool
		errCode  string
	}{
		{"valid hostname", "example.com", false, ""},
		{"valid IP v4", "192.168.1.1", false, ""},
		{"valid IP v6", "2001:db8::1", false, ""},
		{"localhost", "localhost", false, ""},
		{"empty hostname", "", true, "VAL001"},
		{"too long hostname", string(make([]byte, 300)), true, "VAL002"},
		{"suspicious pattern pipe", "example.com|whoami", true, "SEC006"},
		{"suspicious pattern semicolon", "example.com;rm", true, "SEC006"},
		{"suspicious pattern dollar", "example$test.com", true, "SEC006"},
		{"starts with dash", "-example.com", true, "VAL004"},
		{"ends with dash", "example.com-", true, "VAL004"},
		{"consecutive dots", "example..com", true, "VAL005"},
		{"consecutive hyphens", "example--test.com", true, "VAL005"},
		{"empty label", "example..com", true, "VAL005"},
		{"label too long", "a" + strings.Repeat("b", 63) + ".com", true, "VAL007"},
		{"invalid label format", "example_.com", true, "VAL008"},
		{"multicast IP", "224.0.0.1", true, "VAL003"},
		{"unspecified IP", "0.0.0.0", true, "VAL003"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateHostname(tt.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostname() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCode != "" {
				if gcErr, ok := err.(*errors.GoCatError); ok {
					if gcErr.Code() != tt.errCode {
						t.Errorf("ValidateHostname() error code = %v, want %v", gcErr.Code(), tt.errCode)
					}
				} else {
					t.Errorf("ValidateHostname() expected GoCatError, got %T", err)
				}
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
		errCode string
	}{
		{"valid port 80", "80", 80, false, ""},
		{"valid port 443", "443", 443, false, ""},
		{"valid port 65535", "65535", 65535, false, ""},
		{"empty port", "", 0, true, "VAL009"},
		{"non-numeric characters", "80abc", 0, true, "SEC007"},
		{"leading zeros", "0080", 0, true, "VAL010"},
		{"too long", "123456", 0, true, "VAL011"},
		{"invalid format", "abc", 0, true, "SEC007"},
		{"port zero", "0", 0, true, "VAL013"},
		{"port too high", "65536", 0, true, "VAL013"},
		{"negative port", "-1", 0, true, "VAL013"},
		{"injection attempt", "80; rm -rf /", 0, true, "SEC007"},
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
			if tt.wantErr && tt.errCode != "" {
				if gcErr, ok := err.(*errors.GoCatError); ok {
					if gcErr.Code() != tt.errCode {
						t.Errorf("ValidatePort() error code = %v, want %v", gcErr.Code(), tt.errCode)
					}
				} else {
					t.Errorf("ValidatePort() expected GoCatError, got %T", err)
				}
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

func TestInputValidator_ValidateCommand(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		command string
		want    string
		wantErr bool
		errCode string
	}{
		{"safe command", "echo hello", "echo hello", false, ""},
		{"empty command", "", "", true, "VAL014"},
		{"too long command", string(make([]byte, 1001)), "", true, "VAL015"},
		{"command with semicolon", "echo hello; rm -rf /", "", true, "SEC008"},
		{"command with pipe", "cat file | rm", "", true, "SEC008"},
		{"command with rm", "rm file.txt", "", true, "SEC010"},
		{"command with backtick", "echo `whoami`", "", true, "SEC008"},
		{"command with sudo", "sudo echo hello", "", true, "SEC010"},
		{"command with path traversal", "echo ../etc/passwd", "", true, "SEC009"},
		{"non-whitelisted command", "python script.py", "", true, "SEC010"},
		{"whitelisted command with long arg", "echo " + strings.Repeat("a", 101), "", true, "VAL017"},
		{"command with path traversal in arg", "echo ../file", "", true, "SEC009"},
		{"valid ls command", "ls", "ls", false, ""},
		{"valid cat command", "cat file.txt", "cat file.txt", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.ValidateCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateCommand() = %v, want %v", got, tt.want)
			}
			if tt.wantErr && tt.errCode != "" {
				if gcErr, ok := err.(*errors.GoCatError); ok {
					if gcErr.Code() != tt.errCode {
						t.Errorf("ValidateCommand() error code = %v, want %v", gcErr.Code(), tt.errCode)
					}
				} else {
					t.Errorf("ValidateCommand() expected GoCatError, got %T", err)
				}
			}
		})
	}
}

func TestInputValidator_ValidateFilePath(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errCode string
	}{
		{"valid relative path", "file.txt", false, ""},
		{"valid absolute path", "/home/user/file.txt", false, ""},
		{"empty path", "", true, "VAL018"},
		{"too long path", strings.Repeat("a", 5000), true, "VAL019"},
		{"path traversal", "../etc/passwd", true, "SEC012"},
		{"null byte", "file\x00.txt", true, "SEC013"},
		{"dangerous system path", "/etc/passwd", true, "SEC014"},
		{"windows system path", "C:\\Windows\\system32", true, "SEC014"},
		{"path with pipe", "file|whoami", true, "SEC015"},
		{"path with semicolon", "file;rm", true, "SEC015"},
		{"valid directory path", "/home/user/documents/", false, ""},
		{"proc path", "/proc/version", true, "SEC014"},
		{"dev path", "/dev/null", true, "SEC014"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCode != "" {
				if gcErr, ok := err.(*errors.GoCatError); ok {
					if gcErr.Code() != tt.errCode {
						t.Errorf("ValidateFilePath() error code = %v, want %v", gcErr.Code(), tt.errCode)
					}
				} else {
					t.Errorf("ValidateFilePath() expected GoCatError, got %T", err)
				}
			}
		})
	}
}

func TestInputValidator_SanitizeInput(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal text", "hello world", "hello world"},
		{"empty input", "", ""},
		{"null bytes", "hello\x00world", "helloworld"},
		{"control characters", "hello\x01\x02world", "helloworld"},
		{"multiple spaces", "hello    world", "hello world"},
		{"leading/trailing spaces", "  hello world  ", "hello world"},
		{"tabs and newlines", "hello\tworld\n", "hello\tworld"},
		{"too long input", strings.Repeat("a", 1500), strings.Repeat("a", 1000)},
		{"mixed whitespace", "hello \t\n world", "hello \t\n world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.SanitizeInput(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeInput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)
	defer rl.Close()

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

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)
	defer rl.Close()

	// Fill up the limit
	rl.Allow("test-client")
	rl.Allow("test-client")

	// Should be blocked
	if rl.Allow("test-client") {
		t.Error("Request should be blocked")
	}

	// Reset the client
	rl.Reset("test-client")

	// Should be allowed again
	if !rl.Allow("test-client") {
		t.Error("Request should be allowed after reset")
	}
}

func TestRateLimiter_GetStats(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)
	defer rl.Close()

	// Make some requests
	rl.Allow("test-client")
	rl.Allow("test-client")
	rl.Allow("test-client")

	stats := rl.GetStats("test-client")

	if stats.Identifier != "test-client" {
		t.Errorf("Stats identifier = %v, want %v", stats.Identifier, "test-client")
	}

	if stats.RequestCount != 3 {
		t.Errorf("Stats request count = %v, want %v", stats.RequestCount, 3)
	}

	if stats.MaxRequests != 5 {
		t.Errorf("Stats max requests = %v, want %v", stats.MaxRequests, 5)
	}

	if stats.RemainingQuota != 2 {
		t.Errorf("Stats remaining quota = %v, want %v", stats.RemainingQuota, 2)
	}

	if stats.IsBlocked {
		t.Error("Stats should not show blocked when under limit")
	}

	// Fill up the limit
	rl.Allow("test-client")
	rl.Allow("test-client")

	stats = rl.GetStats("test-client")
	if !stats.IsBlocked {
		t.Error("Stats should show blocked when at limit")
	}

	if stats.RemainingQuota != 0 {
		t.Errorf("Stats remaining quota = %v, want %v", stats.RemainingQuota, 0)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(3, 100*time.Millisecond)
	defer rl.Close()

	// Make requests
	rl.Allow("test-client")
	rl.Allow("test-client")

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Should be able to make new requests (old ones cleaned up)
	for i := 0; i < 3; i++ {
		if !rl.Allow("test-client") {
			t.Errorf("Request %d should be allowed after cleanup", i+1)
		}
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
func TestGetSecureTLSConfig(t *testing.T) {
	config := GetSecureTLSConfig()

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion to be TLS 1.2, got %d", config.MinVersion)
	}

	if config.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion to be TLS 1.3, got %d", config.MaxVersion)
	}

	if config.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false by default")
	}

	if !config.PreferServerCipherSuites {
		t.Error("Expected PreferServerCipherSuites to be true by default")
	}

	if len(config.CipherSuites) == 0 {
		t.Error("Expected cipher suites to be configured")
	}

	expectedNextProtos := []string{"h2", "http/1.1"}
	if len(config.NextProtos) != len(expectedNextProtos) {
		t.Errorf("Expected %d next protocols, got %d", len(expectedNextProtos), len(config.NextProtos))
	}
}

func TestValidateTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *tls.Config
		wantErr bool
		errCode string
	}{
		{
			name: "secure config",
			config: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: false,
			},
			wantErr: false,
		},
		{
			name: "insecure TLS version",
			config: &tls.Config{
				MinVersion:         tls.VersionTLS11,
				InsecureSkipVerify: false,
			},
			wantErr: true,
			errCode: "TLS001",
		},
		{
			name: "skip certificate verification",
			config: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
			wantErr: true,
			errCode: "TLS002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCode != "" {
				if gcErr, ok := err.(*errors.GoCatError); ok {
					if gcErr.Code() != tt.errCode {
						t.Errorf("ValidateTLSConfig() error code = %v, want %v", gcErr.Code(), tt.errCode)
					}
				} else {
					t.Errorf("ValidateTLSConfig() expected GoCatError, got %T", err)
				}
			}
		})
	}
}
