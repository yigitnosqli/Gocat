package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExecuteGoCatCommand executes a gocat command and returns the output
func ExecuteGoCatCommand(ctx context.Context, command string, args []string) (string, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	
	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// FormatToolResult formats the result for MCP response
func FormatToolResult(result interface{}, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	// Format as text content
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("%v", result),
			},
		},
	}, nil
}

// ParsePortRange parses port specification
func ParsePortRange(ports string) string {
	// Clean up port specification
	ports = strings.TrimSpace(ports)
	
	// Validate format
	if strings.Contains(ports, ",") || strings.Contains(ports, "-") {
		return ports
	}
	
	// Single port
	return ports
}

// BuildGoCatArgs builds command line arguments for gocat
func BuildGoCatArgs(command string, args map[string]interface{}) []string {
	result := []string{command}
	
	switch command {
	case "scan":
		if host, ok := args["host"].(string); ok {
			result = append(result, host)
		}
		if ports, ok := args["ports"].(string); ok {
			result = append(result, "--ports", ports)
		}
		if protocol, ok := args["protocol"].(string); ok && protocol == "udp" {
			result = append(result, "--udp")
		}
		if concurrency, ok := args["concurrency"].(float64); ok {
			result = append(result, "--concurrency", fmt.Sprintf("%.0f", concurrency))
		}
		if timeout, ok := args["timeout"].(float64); ok {
			result = append(result, "--scan-timeout", fmt.Sprintf("%.0fs", timeout))
		}
		
	case "connect":
		if host, ok := args["host"].(string); ok {
			result = append(result, host)
		}
		if port, ok := args["port"].(float64); ok {
			result = append(result, fmt.Sprintf("%.0f", port))
		}
		if protocol, ok := args["protocol"].(string); ok {
			if protocol == "udp" {
				result = append(result, "--udp")
			} else if protocol == "sctp" {
				result = append(result, "--sctp")
			}
		}
		if ssl, ok := args["ssl"].(bool); ok && ssl {
			result = append(result, "--ssl")
		}
		if timeout, ok := args["timeout"].(float64); ok {
			result = append(result, "--wait", fmt.Sprintf("%.0fs", timeout))
		}
		if retry, ok := args["retry"].(float64); ok {
			result = append(result, "--retry", fmt.Sprintf("%.0f", retry))
		}
		
	case "listen":
		if port, ok := args["port"].(float64); ok {
			result = append(result, fmt.Sprintf("%.0f", port))
		}
		if protocol, ok := args["protocol"].(string); ok && protocol == "udp" {
			result = append(result, "--udp")
		}
		if ssl, ok := args["ssl"].(bool); ok && ssl {
			result = append(result, "--ssl")
		}
		if keepOpen, ok := args["keep_open"].(bool); ok && keepOpen {
			result = append(result, "--keep-open")
		}
	}
	
	return result
}

// ValidateArguments validates tool arguments
func ValidateArguments(toolName string, args map[string]interface{}) error {
	required := getRequiredArgs(toolName)
	
	for _, arg := range required {
		if _, exists := args[arg]; !exists {
			return fmt.Errorf("missing required argument: %s", arg)
		}
	}
	
	return nil
}

// getRequiredArgs returns required arguments for a tool
func getRequiredArgs(toolName string) []string {
	requirements := map[string][]string{
		"scan_ports":         {"host", "ports"},
		"scan_network":       {"network"},
		"connect":            {"host", "port"},
		"listen":             {"port"},
		"send_file":          {"file", "host", "port"},
		"receive_file":       {"port"},
		"start_proxy":        {"listen_port", "backends"},
		"convert_protocol":   {"from", "to"},
		"start_broker":       {"port"},
		"websocket_server":   {"port"},
		"websocket_connect":  {"url"},
		"unix_socket_listen": {"path"},
		"unix_socket_connect": {"path"},
		"create_ssh_tunnel":  {"ssh_server"},
		"dns_lookup":         {"hostname"},
		"dns_tunnel":         {"domain", "mode"},
	}
	
	if req, exists := requirements[toolName]; exists {
		return req
	}
	
	return []string{}
}

// getGoCatExecutable returns the path to gocat executable (internal)
func getGoCatExecutable() string {
	// Try to find gocat in PATH
	path, err := exec.LookPath("gocat")
	if err == nil {
		return path
	}
	
	// Try GetGoCatPath from autoconfig
	if path, err := GetGoCatPath(); err == nil {
		return path
	}
	
	// Fallback to relative path
	return "./gocat"
}

// FormatDuration formats duration for command line
func FormatDuration(seconds float64) string {
	if seconds < 1 {
		return fmt.Sprintf("%.0fms", seconds*1000)
	}
	return fmt.Sprintf("%.0fs", seconds)
}

// FormatSize formats size for command line
func FormatSize(bytes float64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", bytes/KB)
	default:
		return fmt.Sprintf("%.0fB", bytes)
	}
}
