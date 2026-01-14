package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Server represents a reusable MCP server
type Server struct {
	port       int
	httpServer *http.Server
	toolRegistry *ToolRegistry
	running    bool
	mu         sync.Mutex
	onShutdown []func() // Callbacks on shutdown
	hooks      map[string][]func(args map[string]interface{}) (interface{}, error)
}

// ToolRegistry holds all available MCP tools
type ToolRegistry struct {
	tools map[string]Tool
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Handler is the function type for tool handlers
type Handler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// NewServer creates a new reusable MCP server
func NewServer(port int) *Server {
	return &Server{
		port:         port,
		toolRegistry: NewToolRegistry(),
		hooks:        make(map[string][]func(args map[string]interface{}) (interface{}, error)),
	}
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(name, description string, parameters map[string]interface{}) {
	r.tools[name] = Tool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
	}
}

// Get returns a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools
func (r *ToolRegistry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// AddHandler adds a handler for a tool
func (s *Server) AddHandler(name, description string, parameters map[string]interface{}, handler Handler) {
	s.toolRegistry.Register(name, description, parameters)
	s.hooks[name] = append(s.hooks[name], handler)
}

// OnShutdown adds a callback to run on shutdown
func (s *Server) OnShutdown(callback func()) {
	s.onShutdown = append(s.onShutdown, callback)
}

// Start begins the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/health", s.healthHandler)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		fmt.Printf("MCP Server running on http://localhost:%d/mcp\n", s.port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP server error: %v\n", err)
		}
	}()

	// Wait for shutdown
	<-ctx.Done()
	return s.Stop()
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.running = false

	// Run shutdown callbacks
	for _, cb := range s.onShutdown {
		cb()
	}

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	response := s.handleRequest(r.Context(), req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleRequest(ctx context.Context, req MCPRequest) MCPResponse {
	switch req.Method {
	case "initialize":
		return MCPResponse{
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": s.toolRegistry.List(),
				},
				"serverInfo": map[string]string{
					"name":    "kairos-mcp",
					"version": "1.0.0",
				},
			},
		}

	case "tools/list":
		return MCPResponse{
			Result: map[string]interface{}{
				"tools": s.toolRegistry.List(),
			},
		}

	case "tools/call":
		toolName, ok := req.Params["name"].(string)
		if !ok {
			return MCPResponse{Error: "missing tool name"}
		}

		toolArgs, _ := req.Params["arguments"].(map[string]interface{})
		if toolArgs == nil {
			toolArgs = make(map[string]interface{})
		}

		_, exists := s.toolRegistry.Get(toolName)
		if !exists {
			return MCPResponse{Error: fmt.Sprintf("unknown tool: %s", toolName)}
		}

		// Execute all handlers for this tool
		handlers := s.hooks[toolName]
		if len(handlers) == 0 {
			return MCPResponse{Error: fmt.Sprintf("no handler for tool: %s", toolName)}
		}

		var finalResult interface{}
		for _, handler := range handlers {
			result, err := handler(ctx, toolArgs)
			if err != nil {
				return MCPResponse{Error: err.Error()}
			}
			finalResult = result
		}

		return MCPResponse{
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type":  "text",
						"text": finalResult,
					},
				},
			},
		}

	default:
		return MCPResponse{Error: fmt.Sprintf("unknown method: %s", req.Method)}
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"server": "kairos-mcp",
	})
}

// MCPRequest represents an MCP request
type MCPRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// RunServer is a convenience function to run the server
func RunServer(port int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	server := NewServer(port)
	return server.Start(ctx)
}

// ToolParameters returns standard parameters for a tool
func ToolParameters(fields map[string]map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": fields,
	}
}

// StringParam creates a string parameter
func StringParam(description string, enum []string) map[string]interface{} {
	p := map[string]interface{}{
		"type":        "string",
		"description": description,
	}
	if enum != nil {
		p["enum"] = enum
	}
	return p
}

// IntParam creates an integer parameter
func IntParam(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

// BoolParam creates a boolean parameter
func BoolParam(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

// ArrayParam creates an array parameter
func ArrayParam(description string, items map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items":       items,
	}
}
