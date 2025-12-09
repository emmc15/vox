package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/emmett/diaz/internal/audio"
	"github.com/emmett/diaz/internal/models"
	"github.com/emmett/diaz/internal/output"
	"github.com/emmett/diaz/internal/stt"
)

// TranscriberConfig holds configuration for the transcription session
type TranscriberConfig struct {
	ModelName       string
	OutputFormat    string
	OutputFile      string
	EnableVAD       bool
	VADThreshold    float64
	VADSilenceDelay float64
	AudioDevice     string
	AutoDownload    bool
}

// Transcriber orchestrates the transcription process
type Transcriber struct {
	config TranscriberConfig
}

// NewTranscriber creates a new Transcriber instance
func NewTranscriber(config TranscriberConfig) *Transcriber {
	return &Transcriber{config: config}
}

// Run starts the transcription session
func (t *Transcriber) Run() error {
	mgr := NewModelManager()

	// Determine which model to use
	selectedModel, err := mgr.SelectModel(t.config.ModelName, false)
	if err != nil {
		return fmt.Errorf("failed to select model: %w", err)
	}

	// Ensure model is downloaded
	selectedModel, err = mgr.EnsureModel(selectedModel, t.config.AutoDownload)
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
	deviceMgr := NewDeviceManager()
	selectedDevice, err := deviceMgr.SelectDevice(t.config.AudioDevice)
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
	if t.config.OutputFile != "" {
		var fileErr error
		outFile, fileErr = os.Create(t.config.OutputFile)
		if fileErr != nil {
			return fmt.Errorf("failed to create output file: %w", fileErr)
		}
		defer outFile.Close()
		writer = outFile
	}

	// Create formatter based on format flag
	switch strings.ToLower(t.config.OutputFormat) {
	case "json":
		formatter = output.NewJSONFormatter(writer)
	case "text":
		formatter = output.NewPlainTextFormatter(writer)
	case "console":
		// For console mode, we'll use the console output directly
		formatter = nil
	default:
		return fmt.Errorf("unknown output format: %s (valid: console, json, text)", t.config.OutputFormat)
	}
	if formatter != nil {
		defer formatter.Close()
	}

	// Console output for status messages (always to stderr when using file output)
	statusOut := output.DefaultConsoleOutput()
	if t.config.OutputFile != "" {
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
	if t.config.EnableVAD {
		vadConfig := audio.DefaultVADConfig()
		vadConfig.EnergyThreshold = t.config.VADThreshold
		// Convert silence delay (seconds) to frames
		// Assuming 30ms per frame (16kHz, ~480 samples per frame)
		framesPerSecond := 33.33 // ~30ms per frame
		vadConfig.SilenceFrames = int(t.config.VADSilenceDelay * framesPerSecond)
		vad = audio.NewVAD(vadConfig)
		statusOut.Info(fmt.Sprintf("Voice Activity Detection enabled (threshold: %.4f, silence delay: %.1fs)", t.config.VADThreshold, t.config.VADSilenceDelay))
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

// getAudioConfigForModel selects the appropriate audio configuration based on model size
func getAudioConfigForModel(modelName string) audio.CaptureConfig {
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
