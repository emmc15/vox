package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/emmett/diaz/internal/audio"
	"github.com/emmett/diaz/internal/models"
	"github.com/emmett/diaz/internal/output"
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
	listModels      = flag.Bool("list-models", false, "List all available models for download")
	listDownloaded  = flag.Bool("list-downloaded", false, "List all downloaded models")
	downloadModel   = flag.String("download-model", "", "Download a specific model by name")
	modelName       = flag.String("model", "", "Use a specific model (default: "+models.DefaultModelName+")")
	showVersion     = flag.Bool("version", false, "Show version information")
	autoDownload    = flag.Bool("auto-download", false, "Automatically download default model if not found (no prompt)")
)

func main() {
	flag.Parse()

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

	// Handle model management commands
	if *listModels {
		handleListModels()
		return
	}

	if *listDownloaded {
		handleListDownloaded()
		return
	}

	if *downloadModel != "" {
		handleDownloadModel(*downloadModel)
		return
	}

	// Run main application
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleListModels() {
	fmt.Println("Available models for download:")
	fmt.Println()

	for i, model := range models.AvailableModels {
		fmt.Printf("%d. %s\n", i+1, model.Name)
		fmt.Printf("   Language: %s\n", model.Language)
		fmt.Printf("   Size:     %s\n", model.Size)
		fmt.Printf("   Info:     %s\n", model.Description)

		// Check if already downloaded
		downloaded, _ := models.IsModelDownloaded(model.Name)
		if downloaded {
			fmt.Printf("   Status:   ✓ Downloaded\n")
		} else {
			fmt.Printf("   Status:   Not downloaded\n")
		}
		fmt.Println()
	}

	fmt.Println("To download a model, use:")
	fmt.Println("  diaz --download-model <model-name>")
}

func handleListDownloaded() {
	downloaded, err := models.ListDownloadedModels()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing models: %v\n", err)
		os.Exit(1)
	}

	if len(downloaded) == 0 {
		fmt.Println("No models downloaded yet.")
		fmt.Println()
		fmt.Println("Use 'diaz --list-models' to see available models")
		fmt.Println("Use 'diaz --download-model <name>' to download a model")
		return
	}

	fmt.Printf("Downloaded models (%d):\n", len(downloaded))
	fmt.Println()

	for i, modelName := range downloaded {
		fmt.Printf("%d. %s", i+1, modelName)
		if modelName == models.DefaultModelName {
			fmt.Printf(" [DEFAULT]")
		}
		fmt.Println()

		// Get model path and size
		modelPath, err := models.GetModelPath(modelName)
		if err == nil {
			// Try to get directory size (rough estimate)
			fmt.Printf("   Path: %s\n", modelPath)
		}
	}
	fmt.Println()
	fmt.Println("To use a model, run:")
	fmt.Println("  diaz --model <model-name>")
}

func handleDownloadModel(name string) {
	// Check if model exists in available list
	model := models.FindModel(name)
	if model == nil {
		fmt.Fprintf(os.Stderr, "Error: Unknown model '%s'\n", name)
		fmt.Println()
		fmt.Println("Use 'diaz --list-models' to see available models")
		os.Exit(1)
	}

	// Check if already downloaded
	downloaded, err := models.IsModelDownloaded(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking model: %v\n", err)
		os.Exit(1)
	}

	if downloaded {
		fmt.Printf("Model '%s' is already downloaded.\n", name)
		modelPath, _ := models.GetModelPath(name)
		fmt.Printf("Location: %s\n", modelPath)
		return
	}

	// Download the model
	fmt.Printf("Downloading model: %s (%s)\n", model.Name, model.Size)
	fmt.Printf("Description: %s\n", model.Description)
	fmt.Println()

	err = models.DownloadModel(name, func(downloaded, total int64) {
		percent := float64(downloaded) / float64(total) * 100
		fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError downloading model: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✓ Model '%s' downloaded successfully!\n", name)
}

func run() error {
	// Determine which model to use
	selectedModel := models.DefaultModelName
	if *modelName != "" {
		selectedModel = *modelName
	}

	// Check if model is downloaded
	downloaded, err := models.IsModelDownloaded(selectedModel)
	if err != nil {
		return fmt.Errorf("failed to check for model: %w", err)
	}

	if !downloaded {
		if *autoDownload {
			// Auto-download without prompting
			fmt.Printf("Model '%s' not found. Downloading automatically...\n", selectedModel)
			err = models.DownloadModel(selectedModel, func(downloaded, total int64) {
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
			})
			if err != nil {
				return fmt.Errorf("failed to download model: %w", err)
			}
			fmt.Println()
		} else {
			// Prompt user
			fmt.Printf("Model '%s' not found.\n", selectedModel)
			fmt.Println()
			fmt.Println("Available models:")
			for i, model := range models.AvailableModels {
				marker := ""
				if model.Name == selectedModel {
					marker = " (selected)"
				}
				fmt.Printf("  %d. %s (%s) - %s%s\n", i+1, model.Name, model.Size, model.Description, marker)
			}
			fmt.Println()
			fmt.Printf("Download '%s'? (y/n): ", selectedModel)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println()
				fmt.Println("You can download models using:")
				fmt.Println("  diaz --list-models          # List available models")
				fmt.Println("  diaz --download-model <name> # Download a specific model")
				return nil
			}

			// Download the model with progress
			err = models.DownloadModel(selectedModel, func(downloaded, total int64) {
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
			})
			if err != nil {
				return fmt.Errorf("failed to download model: %w", err)
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("Using model: %s\n", selectedModel)
	}

	// Get model path
	modelPath, err := models.GetModelPath(selectedModel)
	if err != nil {
		return fmt.Errorf("failed to get model path: %w", err)
	}

	// List available audio devices
	fmt.Println("\nDetecting audio devices...")
	devices, err := audio.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no audio capture devices found")
	}

	fmt.Printf("Found %d capture device(s):\n", len(devices))
	for _, device := range devices {
		fmt.Printf("  - %s\n", device.String())
	}
	fmt.Println()

	// Get default device
	defaultDevice, err := audio.GetDefaultDevice()
	if err != nil {
		return fmt.Errorf("failed to get default device: %w", err)
	}

	fmt.Printf("Using device: %s\n", defaultDevice.Name)
	fmt.Println()

	// Initialize STT engine
	fmt.Println("Initializing speech recognition engine...")
	engine := stt.NewVoskEngine()
	sttConfig := stt.DefaultConfig(modelPath)
	if err := engine.Initialize(sttConfig); err != nil {
		return fmt.Errorf("failed to initialize STT engine: %w", err)
	}
	defer engine.Close()

	// Initialize audio capture
	audioConfig := audio.DefaultConfig()
	capturer, err := audio.NewCapturer(audioConfig)
	if err != nil {
		return fmt.Errorf("failed to create capturer: %w", err)
	}

	// Initialize output
	out := output.DefaultConsoleOutput()
	out.Info("Speech recognition ready!")
	out.Info(fmt.Sprintf("Listening on %s (sample rate: %d Hz, channels: %d)",
		defaultDevice.Name, audioConfig.SampleRate, audioConfig.Channels))
	out.Info("Speak into your microphone. Press Ctrl+C to stop.")
	fmt.Println()
	fmt.Println("Transcription:")
	fmt.Println("=" + strings.Repeat("=", 70))

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
				fmt.Printf("\n[FINAL] %s", finalResult.Text)
				if finalResult.Confidence > 0 {
					fmt.Printf(" (confidence: %.2f)", finalResult.Confidence)
				}
				fmt.Println()
			}

			fmt.Println("\n" + strings.Repeat("=", 72))
			out.Info("Transcription stopped")
			out.Info(fmt.Sprintf("Total transcriptions: %d", transcriptionCount))
			return nil

		case sample, ok := <-capturer.Samples():
			if !ok {
				return nil
			}

			// Process audio through STT engine
			result, err := engine.ProcessAudio(ctx, sample.Data)
			if err != nil {
				out.Error(fmt.Sprintf("STT error: %v", err))
				continue
			}

			if result == nil {
				continue
			}

			// Handle partial results (real-time feedback)
			if result.Partial && result.Text != "" {
				if result.Text != lastPartialText {
					// Clear previous partial result and show new one
					fmt.Printf("\r%s", strings.Repeat(" ", 80))
					fmt.Printf("\r[partial] %s", result.Text)
					lastPartialText = result.Text
				}
			}

			// Handle final results (complete phrases/sentences)
			if !result.Partial && result.Text != "" {
				// Clear partial result line
				fmt.Printf("\r%s\r", strings.Repeat(" ", 80))

				// Print final transcription
				transcriptionCount++
				fmt.Printf("[%d] %s", transcriptionCount, result.Text)
				if result.Confidence > 0 {
					fmt.Printf(" (confidence: %.2f)", result.Confidence)
				}
				fmt.Println()

				lastPartialText = ""
			}

		case err, ok := <-capturer.Errors():
			if !ok {
				return nil
			}
			out.Error(fmt.Sprintf("Capture error: %v", err))
		}
	}
}
