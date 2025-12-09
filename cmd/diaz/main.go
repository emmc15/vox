package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/emmett/diaz/internal/app"
	"github.com/emmett/diaz/internal/audio"
	"github.com/emmett/diaz/internal/config"
	"github.com/emmett/diaz/internal/models"
	"github.com/emmett/diaz/internal/output"
	"github.com/emmett/diaz/internal/server/mcp"
	"github.com/emmett/diaz/internal/stt"
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


// getAudioConfigForModel selects the appropriate audio configuration based on model size
func getAudioConfigForModel(modelName string) audio.CaptureConfig {
	// Detect model size from name
	// Small models: "small" in name
	// Large models: "0.22" (1.8GB model) or explicit "large"
	// Medium models: "lgraph" or other medium-sized models

	modelLower := strings.ToLower(modelName)

	// Large model detection (1.8GB model or "large" in name)
	if strings.Contains(modelLower, "vosk-model-en-us-0.22") && !strings.Contains(modelLower, "lgraph") {
		fmt.Println("[INFO] Large model detected - using large buffer configuration")
		return audio.LargeModelConfig()
	}

	// Medium model detection
	if strings.Contains(modelLower, "lgraph") ||
		strings.Contains(modelLower, "medium") {
		fmt.Println("[INFO] Medium model detected - using medium buffer configuration")
		return audio.MediumModelConfig()
	}

	// Small/default model
	fmt.Println("[INFO] Small model detected - using default buffer configuration")
	return audio.DefaultConfig()
}

func run() error {
	mgr := app.NewModelManager()

	// Determine which model to use
	selectedModel, err := mgr.SelectModel(*modelName, *selectModel)
	if err != nil {
		return fmt.Errorf("failed to select model: %w", err)
	}

	// Ensure model is downloaded
	selectedModel, err = mgr.EnsureModel(selectedModel, *autoDownload)
	if err != nil {
		return err
	}

	fmt.Printf("Using model: %s\n", selectedModel)

	// Get model path
	modelPath, err := models.GetModelPath(selectedModel)
	if err != nil {
		return fmt.Errorf("failed to get model path: %w", err)
	}

	// Select audio device
	deviceMgr := app.NewDeviceManager()
	selectedDevice, err := deviceMgr.SelectDevice(*audioDevice)
	if err != nil {
		return err
	}

	// Initialize STT engine
	fmt.Println("Initializing speech recognition engine...")
	engine := stt.NewVoskEngine()
	sttConfig := stt.DefaultConfig(modelPath)
	if err := engine.Initialize(sttConfig); err != nil {
		return fmt.Errorf("failed to initialize STT engine: %w", err)
	}
	defer engine.Close()

	// Select audio configuration based on model size
	audioConfig := getAudioConfigForModel(selectedModel)

	// Set the selected device
	audioConfig.DeviceID = selectedDevice.ID

	fmt.Printf("Audio buffer: %d samples (%.1f seconds at 16kHz)\n",
		audioConfig.SampleBufferSize,
		float64(audioConfig.SampleBufferSize)*float64(audioConfig.BufferFrames)/float64(audioConfig.SampleRate))

	// Initialize audio capture
	capturer, err := audio.NewCapturer(audioConfig)
	if err != nil {
		return fmt.Errorf("failed to create capturer: %w", err)
	}

	// Initialize output formatter
	var formatter output.Formatter
	var outFile *os.File

	// Determine output writer
	writer := os.Stdout
	if *outputFile != "" {
		var fileErr error
		outFile, fileErr = os.Create(*outputFile)
		if fileErr != nil {
			return fmt.Errorf("failed to create output file: %w", fileErr)
		}
		defer outFile.Close()
		writer = outFile
	}

	// Create formatter based on format flag
	switch strings.ToLower(*outputFormat) {
	case "json":
		formatter = output.NewJSONFormatter(writer)
	case "text":
		formatter = output.NewPlainTextFormatter(writer)
	case "console":
		// For console mode, we'll use the console output directly
		formatter = nil
	default:
		return fmt.Errorf("unknown output format: %s (valid: console, json, text)", *outputFormat)
	}
	if formatter != nil {
		defer formatter.Close()
	}

	// Console output for status messages (always to stderr when using file output)
	statusOut := output.DefaultConsoleOutput()
	if *outputFile != "" {
		// Redirect status messages to stderr when output goes to file
		statusOut = output.NewConsoleOutput(output.ConsoleConfig{
			ShowTimestamp: true,
			Writer:        os.Stderr,
		})
	}

	statusOut.Info("Speech recognition ready!")
	statusOut.Info(fmt.Sprintf("Listening on %s (sample rate: %d Hz, channels: %d)",
		selectedDevice.Name, audioConfig.SampleRate, audioConfig.Channels))
	statusOut.Info("Speak into your microphone. Press Ctrl+C to stop.")

	// Only show transcription header in console mode
	if formatter == nil {
		fmt.Println()
		fmt.Println("Transcription:")
		fmt.Println("=" + strings.Repeat("=", 70))
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nStopping...")
		cancel()
	}()

	// Start capturing
	err = capturer.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start capture: %w", err)
	}
	defer capturer.Stop()

	// Initialize VAD if enabled
	var vad *audio.VAD
	if *enableVAD {
		vadConfig := audio.DefaultVADConfig()
		vadConfig.EnergyThreshold = *vadThreshold
		// Convert silence delay (seconds) to frames
		// Assuming 30ms per frame (16kHz, ~480 samples per frame)
		framesPerSecond := 33.33 // ~30ms per frame
		vadConfig.SilenceFrames = int(*vadSilenceDelay * framesPerSecond)
		vad = audio.NewVAD(vadConfig)
		statusOut.Info(fmt.Sprintf("Voice Activity Detection enabled (threshold: %.4f, silence delay: %.1fs)", *vadThreshold, *vadSilenceDelay))
	}

	// Track state
	var lastPartialText string
	var transcriptionCount int

	// Process audio samples
	for {
		select {
		case <-ctx.Done():
			// Get final result
			finalResult, err := engine.FinalResult()
			if err == nil && finalResult.Text != "" {
				if formatter != nil {
					// Write final result to formatter
					transcriptionCount++
					formatter.WriteResult(output.TranscriptionResult{
						Index:      transcriptionCount,
						Text:       finalResult.Text,
						Confidence: finalResult.Confidence,
						Timestamp:  time.Now(),
						Partial:    false,
					})
				} else {
					// Console output
					fmt.Printf("\n[FINAL] %s", finalResult.Text)
					if finalResult.Confidence > 0 {
						fmt.Printf(" (confidence: %.2f)", finalResult.Confidence)
					}
					fmt.Println()
				}
			}

			if formatter != nil {
				formatter.Flush()
			} else {
				fmt.Println("\n" + strings.Repeat("=", 72))
			}
			statusOut.Info("Transcription stopped")
			statusOut.Info(fmt.Sprintf("Total transcriptions: %d", transcriptionCount))
			return nil

		case sample, ok := <-capturer.Samples():
			if !ok {
				return nil
			}

			// Process VAD if enabled
			if vad != nil {
				isSpeaking, speechStarted, speechEnded := vad.ProcessFrame(sample.Data)

				// Debug: show energy levels
				energy := vad.GetEnergyLevel(sample.Data)
				if formatter == nil {
					fmt.Printf("\r[Energy: %.6f, Speaking: %v]", energy, isSpeaking)
				}

				// Handle speech start
				if speechStarted {
					if formatter != nil {
						formatter.WriteEvent("vad", "Speech detected")
					} else {
						fmt.Printf("\n[Speech detected]\n")
					}
				}

				// Handle speech end - finalize current utterance
				if speechEnded {
					if formatter != nil {
						formatter.WriteEvent("vad", "Silence detected - finalizing")
					} else {
						fmt.Printf("\n[Silence detected - finalizing]\n")
					}

					// Get final result for this utterance
					finalResult, err := engine.FinalResult()
					if err == nil && finalResult.Text != "" {
						transcriptionCount++

						if formatter != nil {
							formatter.WriteResult(output.TranscriptionResult{
								Index:      transcriptionCount,
								Text:       finalResult.Text,
								Confidence: finalResult.Confidence,
								Timestamp:  time.Now(),
								Partial:    false,
							})
						} else {
							// Clear partial text
							fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
							fmt.Printf("[%d] %s", transcriptionCount, finalResult.Text)
							if finalResult.Confidence > 0 {
								fmt.Printf(" (confidence: %.2f)", finalResult.Confidence)
							}
							fmt.Println()
						}
					}

					// Reset for next utterance
					engine.Reset()
					lastPartialText = ""
					continue
				}

				// Skip processing during silence
				if !isSpeaking {
					continue
				}
			}

			// Process audio through STT engine
			result, err := engine.ProcessAudio(ctx, sample.Data)
			if err != nil {
				statusOut.Error(fmt.Sprintf("STT error: %v", err))
				continue
			}

			if result == nil {
				continue
			}

			// Handle partial results (real-time feedback)
			if result.Partial && result.Text != "" {
				if result.Text != lastPartialText {
					if formatter != nil {
						// Write partial result to formatter
						formatter.WritePartial(result.Text)
					} else {
						// Console output: clear previous partial result and show new one
						fmt.Printf("\r%s", strings.Repeat(" ", 80))
						fmt.Printf("\r[partial] %s", result.Text)
					}
					lastPartialText = result.Text
				}
			}

			// Handle final results (complete phrases/sentences)
			if !result.Partial && result.Text != "" {
				transcriptionCount++

				if formatter != nil {
					// Write to formatter
					formatter.WriteResult(output.TranscriptionResult{
						Index:      transcriptionCount,
						Text:       result.Text,
						Confidence: result.Confidence,
						Timestamp:  time.Now(),
						Partial:    false,
					})
				} else {
					// Console output: clear partial result line
					fmt.Printf("\r%s\r", strings.Repeat(" ", 80))

					// Print final transcription
					fmt.Printf("[%d] %s", transcriptionCount, result.Text)
					if result.Confidence > 0 {
						fmt.Printf(" (confidence: %.2f)", result.Confidence)
					}
					fmt.Println()
				}

				lastPartialText = ""
			}

		case err, ok := <-capturer.Errors():
			if !ok {
				return nil
			}
			statusOut.Error(fmt.Sprintf("Capture error: %v", err))
		}
	}
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
