package main

import (
	"bufio"
	"context"
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

func main() {
	fmt.Printf("Diaz v%s (commit: %s, branch: %s, built: %s)\n",
		Version, GitCommit, GitBranch, BuildTime)
	fmt.Println("Speech-to-Text Application")
	fmt.Println()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Check for models
	modelName := models.DefaultModelName
	downloaded, err := models.IsModelDownloaded(modelName)
	if err != nil {
		return fmt.Errorf("failed to check for model: %w", err)
	}

	if !downloaded {
		fmt.Printf("Model '%s' not found.\n", modelName)
		fmt.Println("Available models:")
		for i, model := range models.AvailableModels {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, model.Name, model.Size, model.Description)
		}
		fmt.Println()
		fmt.Printf("Download default model '%s'? (y/n): ", modelName)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cannot proceed without a model. Exiting.")
			return nil
		}

		// Download the model with progress
		err = models.DownloadModel(modelName, func(downloaded, total int64) {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
		})
		if err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		fmt.Println()
	} else {
		fmt.Printf("Using model: %s\n", modelName)
	}

	// Get model path
	modelPath, err := models.GetModelPath(modelName)
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
