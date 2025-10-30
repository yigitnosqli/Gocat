package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	name        string
	version     string
	tools       map[string]*Tool
	resources   map[string]*Resource
	prompts     map[string]*Prompt
	mu          sync.RWMutex
	handlers    map[string]RequestHandler
	ctx         context.Context
	cancel      context.CancelFunc
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     ToolHandler
}

// Resource represents an MCP resource
type Resource struct {
	URI         string            `json:"uri"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	MimeType    string            `json:"mimeType"`
	Handler     ResourceHandler
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Arguments   []PromptArgument `json:"arguments"`
	Handler     PromptHandler
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// RequestHandler handles MCP requests
type RequestHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// ToolHandler handles tool calls
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (interface{}, error)

// ResourceHandler handles resource reads
type ResourceHandler func(ctx context.Context) (interface{}, error)

// PromptHandler handles prompt generation
type PromptHandler func(ctx context.Context, arguments map[string]string) (string, error)

// MCPRequest represents an MCP protocol request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewMCPServer creates a new MCP server
func NewMCPServer(name, version string) *MCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	
	server := &MCPServer{
		name:      name,
		version:   version,
		tools:     make(map[string]*Tool),
		resources: make(map[string]*Resource),
		prompts:   make(map[string]*Prompt),
		handlers:  make(map[string]RequestHandler),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register standard MCP methods
	server.registerStandardHandlers()

	return server
}

// RegisterTool registers a tool
func (s *MCPServer) RegisterTool(tool *Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
	logger.Debug("Registered MCP tool: %s", tool.Name)
}

// RegisterResource registers a resource
func (s *MCPServer) RegisterResource(resource *Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[resource.URI] = resource
	logger.Debug("Registered MCP resource: %s", resource.URI)
}

// RegisterPrompt registers a prompt
func (s *MCPServer) RegisterPrompt(prompt *Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts[prompt.Name] = prompt
	logger.Debug("Registered MCP prompt: %s", prompt.Name)
}

// Start starts the MCP server on stdio
func (s *MCPServer) Start(reader io.Reader, writer io.Writer) error {
	logger.Info("Starting MCP server: %s v%s", s.name, s.version)

	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(writer)

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			var req MCPRequest
			if err := decoder.Decode(&req); err != nil {
				if err == io.EOF {
					return nil
				}
				logger.Error("Failed to decode request: %v", err)
				continue
			}

			response := s.handleRequest(req)
			if err := encoder.Encode(response); err != nil {
				logger.Error("Failed to encode response: %v", err)
				continue
			}
		}
	}
}

// handleRequest handles an MCP request
func (s *MCPServer) handleRequest(req MCPRequest) MCPResponse {
	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	result, err := handler(ctx, req.Params)
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// registerStandardHandlers registers standard MCP protocol handlers
func (s *MCPServer) registerStandardHandlers() {
	s.handlers["initialize"] = s.handleInitialize
	s.handlers["tools/list"] = s.handleToolsList
	s.handlers["tools/call"] = s.handleToolsCall
	s.handlers["resources/list"] = s.handleResourcesList
	s.handlers["resources/read"] = s.handleResourcesRead
	s.handlers["prompts/list"] = s.handlePromptsList
	s.handlers["prompts/get"] = s.handlePromptsGet
}

// handleInitialize handles initialization
func (s *MCPServer) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
			"prompts":   map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.name,
			"version": s.version,
		},
	}, nil
}

// handleToolsList handles tools list request
func (s *MCPServer) handleToolsList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall handles tool call request
func (s *MCPServer) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	s.mu.RLock()
	tool, exists := s.tools[callParams.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", callParams.Name)
	}

	result, err := tool.Handler(ctx, callParams.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("%v", result),
			},
		},
	}, nil
}

// handleResourcesList handles resources list request
func (s *MCPServer) handleResourcesList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]map[string]interface{}, 0, len(s.resources))
	for _, resource := range s.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         resource.URI,
			"name":        resource.Name,
			"description": resource.Description,
			"mimeType":    resource.MimeType,
		})
	}

	return map[string]interface{}{
		"resources": resources,
	}, nil
}

// handleResourcesRead handles resource read request
func (s *MCPServer) handleResourcesRead(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var readParams struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(params, &readParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	s.mu.RLock()
	resource, exists := s.resources[readParams.URI]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("resource not found: %s", readParams.URI)
	}

	content, err := resource.Handler(ctx)
	if err != nil {
		return nil, fmt.Errorf("resource read failed: %w", err)
	}

	return map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      resource.URI,
				"mimeType": resource.MimeType,
				"text":     fmt.Sprintf("%v", content),
			},
		},
	}, nil
}

// handlePromptsList handles prompts list request
func (s *MCPServer) handlePromptsList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]map[string]interface{}, 0, len(s.prompts))
	for _, prompt := range s.prompts {
		prompts = append(prompts, map[string]interface{}{
			"name":        prompt.Name,
			"description": prompt.Description,
			"arguments":   prompt.Arguments,
		})
	}

	return map[string]interface{}{
		"prompts": prompts,
	}, nil
}

// handlePromptsGet handles prompt get request
func (s *MCPServer) handlePromptsGet(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var getParams struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}

	if err := json.Unmarshal(params, &getParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	s.mu.RLock()
	prompt, exists := s.prompts[getParams.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("prompt not found: %s", getParams.Name)
	}

	text, err := prompt.Handler(ctx, getParams.Arguments)
	if err != nil {
		return nil, fmt.Errorf("prompt generation failed: %w", err)
	}

	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": text,
				},
			},
		},
	}, nil
}

// Stop stops the MCP server
func (s *MCPServer) Stop() {
	logger.Info("Stopping MCP server")
	s.cancel()
}
