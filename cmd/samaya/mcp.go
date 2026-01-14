package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kairos/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpPort int

var mcpCmd = &cobra.Command{
	Use:   "mcp [command]",
	Short: "MCP server for AI integration",
	Long: `Kairos MCP Server - A versatile Model Context Protocol server for work tracking.

This enables AI assistants (Claude, Cursor, etc.) to access your work data with
four powerful capabilities:

  think       - Reasoning and analysis about work patterns
  evolve      - Self-improvement suggestions based on your data
  consciousness - Self-awareness about your current work state
  persist     - Long-term memory storage and retrieval

Examples:
  kairos mcp start           # Start server on default port 8765
  kairos mcp start -p 9000   # Start on custom port
  kairos mcp tools           # List available tools
  kairos mcp query think     # Query a tool directly
  kairos mcp register        # Print MCP config for AI clients
`,
	Aliases: []string{"server"},
}

var mcpStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the MCP server",
	Long: `Start the Kairos MCP server.
The server will run until interrupted (Ctrl+C).

AI assistants can connect to: http://localhost:8765/mcp
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if mcpPort == 0 {
			mcpPort = 8765
		}

		fmt.Printf("Kairos MCP Server v1.0\n")
		fmt.Printf("========================\n")
		fmt.Printf("Port: http://localhost:%d/mcp\n", mcpPort)
		fmt.Printf("Press Ctrl+C to stop\n\n")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\nShutting down MCP server...")
			cancel()
		}()

		return mcp.RunServer(db, ollamaService, mcpPort)
	},
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	Long: `List all available MCP tools with their descriptions and parameters.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(db, ollamaService, 0)

		fmt.Println("Available MCP Tools:")
		fmt.Println("====================\n")

		tools := server.ToolRegistry.List()
		for _, tool := range tools {
			fmt.Printf("  %s\n", tool.Name)
			fmt.Printf("    Description: %s\n", tool.Description)
			if params, ok := tool.Parameters["properties"].(map[string]interface{}); ok {
				fmt.Printf("    Parameters:\n")
				for name, param := range params {
					if p, ok := param.(map[string]interface{}); ok {
						enum := ""
						if e, ok := p["enum"]; ok {
							enum = fmt.Sprintf(" (one of: %v)", e)
						}
						fmt.Printf("      - %s: %s%s\n", name, p["description"], enum)
					}
				}
			}
			fmt.Println()
		}
		return nil
	},
}

var mcpQueryCmd = &cobra.Command{
	Use:   "query <tool> [args]",
	Short: "Query a tool directly",
	Long: `Query an MCP tool directly from the command line.
Useful for testing or quick lookups.

Examples:
  kairos mcp query consciousness aspect=current
  kairos mcp query think question="Should I take a break?" analysis_type=productivity
  kairos mcp query persist action=list
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toolName := args[0]

		server := mcp.NewServer(db, ollamaService, 0)
		tool, exists := server.ToolRegistry.Get(toolName)
		if !exists {
			return fmt.Errorf("unknown tool: %s", toolName)
		}

		// Parse remaining args as key=value pairs
		toolArgs := make(map[string]interface{})
		for _, arg := range args[1:] {
			parts := splitOnce(arg, "=")
			if len(parts) == 2 {
				toolArgs[parts[0]] = parseValue(parts[1])
			}
		}

		// Execute tool
		handlers := server.hooks[toolName]
		if len(handlers) == 0 {
			return fmt.Errorf("no handler for tool: %s", toolName)
		}

		ctx := context.Background()
		result, err := handlers[0](ctx, toolArgs)
		if err != nil {
			return err
		}

		// Pretty print
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	},
}

var mcpRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Print MCP configuration for AI clients",
	Long: `Print the MCP server configuration in JSON format.
Use this output to configure AI assistants like Claude or Cursor.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := mcpPort
		if port == 0 {
			port = 8765
		}

		config := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"kairos": map[string]interface{}{
					"url": fmt.Sprintf("http://localhost:%d/mcp", port),
					"transport": "http",
				},
			},
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		fmt.Println(string(data))
		return nil
	},
}

var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check MCP server status",
	Long: `Check if the MCP server is running and responsive.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := mcpPort
		if port == 0 {
			port = 8765
		}

		// Try to hit the health endpoint
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("http://localhost:%d/health", port), nil)
		if err != nil {
			fmt.Printf("MCP Server: Not running (port %d)\n", port)
			return nil
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("MCP Server: Not running (port %d)\n", port)
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			fmt.Printf("MCP Server: Running on port %d\n", port)
		} else {
			fmt.Printf("MCP Server: Error (status %d)\n", resp.StatusCode)
		}
		return nil
	},
}

func init() {
	mcpCmd.AddCommand(mcpStartCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	mcpCmd.AddCommand(mcpQueryCmd)
	mcpCmd.AddCommand(mcpRegisterCmd)
	mcpCmd.AddCommand(mcpStatusCmd)

	mcpStartCmd.Flags().IntVarP(&mcpPort, "port", "p", 8765, "Port for MCP server")
	mcpRegisterCmd.Flags().IntVarP(&mcpPort, "port", "p", 8765, "Port for MCP server")

	rootCmd.AddCommand(mcpCmd)
}

// Helper functions
func splitOnce(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func parseValue(s string) interface{} {
	// Try bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	// Try number
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
