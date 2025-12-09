package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/emmett/diaz/internal/app"
	"github.com/emmett/diaz/internal/config"
	"github.com/emmett/diaz/internal/models"
	"github.com/emmett/diaz/internal/server/mcp"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

// CLI flags
var (
	configFile      = flag.String("config", "", "Path to configuration file (default: ~/.diazrc or /etc/diaz/config.yaml)")
	mode            = flag.String("mode", "cli", "Operation mode: cli, mcp")
	listModels      = flag.Bool("list-models", false, "List all available models for download")
	listDownloaded  = flag.Bool("list-downloaded", false, "List all downloaded models")
	downloadModel   = flag.String("download-model", "", "Download a specific model by name")
	modelName       = flag.String("model", "", "Use a specific model (default: "+models.DefaultModelName+")")
	selectModel     = flag.Bool("select-model", false, "Interactively select a model to use")
	setDefault      = flag.String("set-default", "", "Set a model as the default")
	outputFormat    = flag.String("format", "json", "Output format: console, json, text")
	outputFile      = flag.String("output", "", "Output file (default: stdout)")
	enableVAD       = flag.Bool("vad", true, "Enable Voice Activity Detection for better pause handling")
	vadThreshold    = flag.Float64("vad-threshold", 0.01, "VAD energy threshold (0.001-0.1, lower=more sensitive)")
	vadSilenceDelay = flag.Float64("vad-silence-delay", 5.0, "Delay in seconds after last speech before returning to silence")
	audioDevice     = flag.String("device", "", "Audio input device name (use --list-devices to see available devices)")
	listDevices     = flag.Bool("list-devices", false, "List all available audio input devices")
	showVersion     = flag.Bool("version", false, "Show version information")
	autoDownload    = flag.Bool("auto-download", false, "Automatically download default model if not found (no prompt)")
)

func main() {
	flag.Parse()

	// Load configuration file
	cfg, err := config.LoadWithFallback(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// Apply config values as defaults (CLI flags override if explicitly set)
	applyConfigDefaults(cfg)

	// Handle version flag
	if *showVersion {
		fmt.Printf("Diaz v%s\n", Version)
		fmt.Printf("  Commit:  %s\n", GitCommit)
		fmt.Printf("  Branch:  %s\n", GitBranch)
		fmt.Printf("  Built:   %s\n", BuildTime)
		os.Exit(0)
	}

	fmt.Printf("Diaz v%s (commit: %s, branch: %s, built: %s)\n",
		Version, GitCommit, GitBranch, BuildTime)
	fmt.Println("Speech-to-Text Application")
	fmt.Println()

	// Handle MCP server mode
	if *mode == "mcp" {
		if err := runMCPServer(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle list devices flag
	if *listDevices {
		dm := app.NewDeviceManager()
		if err := dm.ListDevices(); err != nil {
			os.Exit(1)
		}
		return
	}

	// Handle model management commands
	mgr := app.NewModelManager()

	if *listModels {
		if err := mgr.ListModels(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *listDownloaded {
		if err := mgr.ListDownloaded(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *downloadModel != "" {
		if err := mgr.Download(*downloadModel); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *setDefault != "" {
		if err := mgr.SetDefault(*setDefault); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Run main application
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// applyConfigDefaults applies configuration values as defaults
// CLI flags override config file values if explicitly set
func applyConfigDefaults(cfg *config.Config) {
	// Check if flags were explicitly set by user
	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

	// Apply config defaults only if flag was not explicitly set
	if !flagsSet["model"] && cfg.Model.Default != "" {
		*modelName = cfg.Model.Default
	}

	if !flagsSet["format"] && cfg.Output.Format != "" {
		*outputFormat = cfg.Output.Format
	}

	if !flagsSet["output"] && cfg.Output.File != "" {
		*outputFile = cfg.Output.File
	}

	if !flagsSet["vad"] {
		*enableVAD = cfg.VAD.Enabled
	}

	if !flagsSet["vad-threshold"] && cfg.VAD.Threshold > 0 {
		*vadThreshold = cfg.VAD.Threshold
	}

	if !flagsSet["vad-silence-delay"] && cfg.VAD.SilenceDelay > 0 {
		*vadSilenceDelay = cfg.VAD.SilenceDelay
	}

	if !flagsSet["device"] && cfg.Audio.Device != "" {
		*audioDevice = cfg.Audio.Device
	}
}

func run() error {
	// Handle model selection if needed
	mgr := app.NewModelManager()
	selectedModel := *modelName
	if *selectModel {
		var err error
		selectedModel, err = mgr.SelectInteractive()
		if err != nil {
			return fmt.Errorf("failed to select model: %w", err)
		}
	}

	// Create transcriber configuration
	config := app.TranscriberConfig{
		ModelName:       selectedModel,
		OutputFormat:    *outputFormat,
		OutputFile:      *outputFile,
		EnableVAD:       *enableVAD,
		VADThreshold:    *vadThreshold,
		VADSilenceDelay: *vadSilenceDelay,
		AudioDevice:     *audioDevice,
		AutoDownload:    *autoDownload,
	}

	// Create and run transcriber
	transcriber := app.NewTranscriber(config)
	return transcriber.Run()
}

// runMCPServer starts the MCP server
func runMCPServer() error {
	fmt.Fprintf(os.Stderr, "Starting MCP server...\n")
	fmt.Fprintf(os.Stderr, "Protocol: Model Context Protocol (stdio transport)\n")
	fmt.Fprintf(os.Stderr, "Version: %s (commit: %s)\n\n", Version, GitCommit)

	// Get default model
	var modelPath string
	var selectedModel string

	if *modelName != "" {
		selectedModel = *modelName
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

	// Create MCP server
	serverConfig := mcp.Config{
		ServerName:    "diaz-mcp",
		ServerVersion: Version,
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
