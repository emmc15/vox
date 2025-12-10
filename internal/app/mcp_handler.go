package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/emmett/diaz/internal/models"
	"github.com/emmett/diaz/internal/server/mcp"
)

// MCPHandler handles MCP server operations
type MCPHandler struct {
	modelName string
	version   string
	gitCommit string
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(modelName, version, gitCommit string) *MCPHandler {
	return &MCPHandler{
		modelName: modelName,
		version:   version,
		gitCommit: gitCommit,
	}
}

// Run starts the MCP server
func (h *MCPHandler) Run() error {
	fmt.Fprintf(os.Stderr, "Starting MCP server...\n")
	fmt.Fprintf(os.Stderr, "Protocol: Model Context Protocol (stdio transport)\n")
	fmt.Fprintf(os.Stderr, "Version: %s (commit: %s)\n\n", h.version, h.gitCommit)

	// Get default model
	var modelPath string
	var selectedModel string

	if h.modelName != "" {
		selectedModel = h.modelName
	} else {
		var err error
		selectedModel, err = models.GetDefaultModel()
		if err != nil {
			return fmt.Errorf("failed to get default model: %w", err)
		}
	}

	// Check if model is downloaded
	downloaded, err := models.IsModelDownloaded(selectedModel)
	if err != nil {
		return fmt.Errorf("failed to check for model: %w", err)
	}

	if !downloaded {
		return fmt.Errorf("model '%s' not found. Please download it first using:\n  diaz --download-model %s", selectedModel, selectedModel)
	}

	// Get model path
	modelPath, err = models.GetModelPath(selectedModel)
	if err != nil {
		return fmt.Errorf("failed to get model path: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Using model: %s\n", selectedModel)
	fmt.Fprintf(os.Stderr, "Model path: %s\n\n", modelPath)

	// Get absolute path to diaz binary
	execPath, err := os.Executable()
	if err != nil {
		execPath = "./build/diaz"
	}

	// Print MCP client configuration
	type MCPServerConfig struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	type MCPClientConfig struct {
		MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	}

	clientConfig := MCPClientConfig{
		MCPServers: map[string]MCPServerConfig{
			"diaz-stt": {
				Command: execPath,
				Args:    []string{"--mode", "mcp", "--model", selectedModel},
			},
		},
	}

	configJSON, err := json.MarshalIndent(clientConfig, "", "  ")
	if err == nil {
		fmt.Fprintf(os.Stderr, "MCP Client Configuration:\n%s\n\n", string(configJSON))
	}

	// Print Claude Code add command
	type ClaudeCodeConfig struct {
		Type    string   `json:"type"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}

	claudeConfig := ClaudeCodeConfig{
		Type:    "stdio",
		Command: execPath,
		Args:    []string{"--mode", "mcp", "--model", selectedModel},
	}

	claudeJSON, err := json.Marshal(claudeConfig)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Add to Claude Code:\n")
		fmt.Fprintf(os.Stderr, "claude mcp add-json stt '%s'\n\n", string(claudeJSON))
	}

	// Create MCP server
	serverConfig := mcp.Config{
		ServerName:    "diaz-mcp",
		ServerVersion: h.version,
		ModelPath:     modelPath,
		DefaultModel:  selectedModel,
	}

	server, err := mcp.NewServer(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	fmt.Fprintf(os.Stderr, "MCP server ready. Listening on stdin/stdout...\n")
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop.\n\n")

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		fmt.Fprintf(os.Stderr, "\nShutting down MCP server...\n")
		if err := server.Stop(); err != nil {
			return fmt.Errorf("error stopping server: %w", err)
		}
		return nil
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}
