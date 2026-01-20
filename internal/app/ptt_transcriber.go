package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/emmett/vox/internal/audio"
	"github.com/emmett/vox/internal/input"
	"github.com/emmett/vox/internal/models"
	"github.com/emmett/vox/internal/output"
	"github.com/emmett/vox/internal/stt"
)

// PTTConfig holds configuration for push-to-talk mode
type PTTConfig struct {
	TranscriberConfig
	Hotkey string
}

// PTTTranscriber handles push-to-talk transcription
type PTTTranscriber struct {
	config      PTTConfig
	capturer    audio.Capturer
	audioConfig audio.CaptureConfig
	engine      stt.Engine
	hotkeyMgr   *input.HotkeyManager
	statusOut   *output.ConsoleOutput

	recording       bool
	audioBuffer     []byte
	transcriptCount int
	wg              sync.WaitGroup
}

// NewPTTTranscriber creates a new PTTTranscriber
func NewPTTTranscriber(config PTTConfig) *PTTTranscriber {
	return &PTTTranscriber{config: config}
}

// Run starts the push-to-talk transcription loop
func (p *PTTTranscriber) Run() error {
	mgr := NewModelManager()

	// Select and ensure model
	selectedModel, err := mgr.SelectModel(p.config.ModelName, false)
	if err != nil {
		return fmt.Errorf("failed to select model: %w", err)
	}
	selectedModel, err = mgr.EnsureModel(selectedModel, p.config.AutoDownload)
	if err != nil {
		return err
	}

	fmt.Printf("Using model: %s\n", selectedModel)

	modelPath, err := models.GetModelPath(selectedModel)
	if err != nil {
		return fmt.Errorf("failed to get model path: %w", err)
	}

	// Select audio device
	deviceMgr := NewDeviceManager()
	selectedDevice, err := deviceMgr.SelectDevice(p.config.AudioDevice)
	if err != nil {
		return err
	}

	// Initialize STT engine
	fmt.Println("Initializing speech recognition engine...")
	p.engine = stt.NewVoskEngine()
	sttConfig := stt.DefaultConfig(modelPath)
	if err := p.engine.Initialize(sttConfig); err != nil {
		return fmt.Errorf("failed to initialize STT engine: %w", err)
	}
	defer p.engine.Close()

	// Set up audio config (stored for recreating capturer each session)
	p.audioConfig = getAudioConfigForModel(selectedModel)
	p.audioConfig.DeviceID = selectedDevice.ID

	// Status output
	p.statusOut = output.DefaultConsoleOutput()

	// Set up context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nExiting...")
		cancel()
	}()

	// Recording state channel
	toggleChan := make(chan bool, 10)

	// Set up hotkey manager
	p.hotkeyMgr = input.NewHotkeyManager(func(recording bool) {
		toggleChan <- recording
	})

	if err := p.hotkeyMgr.Start(ctx, p.config.Hotkey); err != nil {
		return fmt.Errorf("failed to start hotkey listener: %w", err)
	}
	defer p.hotkeyMgr.Stop()

	fmt.Printf("\nPush-to-talk mode. Press %s to toggle recording.\n", p.config.Hotkey)
	fmt.Println("Press Ctrl+C to exit.")
	fmt.Println("\nWaiting...")

	// Main loop
	var capturerRunning bool
	for {
		select {
		case <-ctx.Done():
			p.recording = false
			p.wg.Wait()
			if capturerRunning {
				p.capturer.Stop()
			}
			return nil

		case recording := <-toggleChan:
			if recording {
				// Start recording
				fmt.Println("\n[Recording]")
				p.audioBuffer = nil
				p.engine.Reset()

				// Create fresh capturer for this recording session
				var err error
				p.capturer, err = audio.NewCapturer(p.audioConfig)
				if err != nil {
					p.statusOut.Error(fmt.Sprintf("Failed to create capturer: %v", err))
					continue
				}

				if err := p.capturer.Start(ctx); err != nil {
					p.statusOut.Error(fmt.Sprintf("Failed to start capture: %v", err))
					continue
				}
				capturerRunning = true
				p.recording = true

				// Start goroutine to collect samples
				p.wg.Add(1)
				go p.collectSamples(ctx)

			} else {
				// Stop recording and transcribe
				p.recording = false
				p.wg.Wait() // wait for collectSamples to exit
				if capturerRunning {
					p.capturer.Stop()
					capturerRunning = false
				}

				fmt.Println("[Stopped - transcribing...]")

				// Transcribe collected audio
				if len(p.audioBuffer) > 0 {
					result, err := p.transcribe(ctx)
					if err != nil {
						p.statusOut.Error(fmt.Sprintf("Transcription error: %v", err))
					} else if result != "" {
						p.transcriptCount++
						fmt.Printf("[%d] \"%s\"\n", p.transcriptCount, result)
					} else {
						fmt.Println("[No speech detected]")
					}
				} else {
					fmt.Println("[No audio recorded]")
				}

				fmt.Println("\nWaiting...")
			}
		}
	}
}

// collectSamples collects audio samples while recording
func (p *PTTTranscriber) collectSamples(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case sample, ok := <-p.capturer.Samples():
			if !ok || !p.recording {
				return
			}
			p.audioBuffer = append(p.audioBuffer, sample.Data...)
		case err, ok := <-p.capturer.Errors():
			if !ok {
				return
			}
			p.statusOut.Error(fmt.Sprintf("Capture error: %v", err))
		}
	}
}

// transcribe processes the collected audio buffer
func (p *PTTTranscriber) transcribe(ctx context.Context) (string, error) {
	// Process audio in chunks
	chunkSize := 480 * 2 // 30ms at 16kHz, 16-bit mono
	for i := 0; i < len(p.audioBuffer); i += chunkSize {
		end := i + chunkSize
		if end > len(p.audioBuffer) {
			end = len(p.audioBuffer)
		}
		chunk := p.audioBuffer[i:end]

		_, err := p.engine.ProcessAudio(ctx, chunk)
		if err != nil {
			return "", err
		}
	}

	// Get final result
	result, err := p.engine.FinalResult()
	if err != nil {
		return "", err
	}

	return result.Text, nil
}
