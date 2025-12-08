package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/emmett/diaz/internal/audio"
	"github.com/emmett/diaz/internal/models"
	"github.com/emmett/diaz/internal/stt"
)

// TranscriptionService handles audio transcription with VAD
type TranscriptionService struct {
	modelPath    string
	defaultModel string
	mu           sync.Mutex
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(modelPath, defaultModel string) (*TranscriptionService, error) {
	// If no model path specified, try to get default model
	if modelPath == "" && defaultModel != "" {
		path, err := models.GetModelPath(defaultModel)
		if err != nil {
			return nil, fmt.Errorf("failed to get model path for %s: %w", defaultModel, err)
		}
		modelPath = path
	}

	return &TranscriptionService{
		modelPath:    modelPath,
		defaultModel: defaultModel,
	}, nil
}

// TranscribeAudio transcribes audio data with VAD support
func (ts *TranscriptionService) TranscribeAudio(ctx context.Context, params TranscribeAudioParams) (*TranscribeAudioResult, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	startTime := time.Now()

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(params.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	// Get model path
	modelPath := ts.modelPath
	if params.Model != "" {
		path, err := models.GetModelPath(params.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to get model path: %w", err)
		}
		modelPath = path
	}

	// Initialize STT engine
	engine := stt.NewVoskEngine()
	sttConfig := stt.DefaultConfig(modelPath)
	if err := engine.Initialize(sttConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize STT engine: %w", err)
	}
	defer engine.Close()

	// Initialize VAD if enabled
	var vad *audio.VAD
	if params.VADEnabled {
		vadConfig := audio.DefaultVADConfig()
		vadConfig.EnergyThreshold = params.VADThreshold
		// Convert silence delay to frames (~33 frames per second)
		vadConfig.SilenceFrames = int(params.VADSilenceDelay * 33.33)
		vad = audio.NewVAD(vadConfig)
	}

	// Process audio in chunks (480 samples = 30ms at 16kHz)
	const frameSize = 480 * 2 // 480 samples * 2 bytes per sample (16-bit)
	var transcription string
	var confidence float64
	var speechDetected bool

	for offset := 0; offset < len(audioData); offset += frameSize {
		// Get chunk
		end := offset + frameSize
		if end > len(audioData) {
			end = len(audioData)
		}
		chunk := audioData[offset:end]

		// If chunk is too small, pad with zeros
		if len(chunk) < frameSize {
			padded := make([]byte, frameSize)
			copy(padded, chunk)
			chunk = padded
		}

		// Process with VAD if enabled
		if vad != nil {
			isSpeaking, _, speechEnded := vad.ProcessFrame(chunk)

			// Mark that we detected speech at some point
			if isSpeaking {
				speechDetected = true
			}

			// If speech ended, finalize and return
			if speechEnded && speechDetected {
				result, err := engine.FinalResult()
				if err == nil && result.Text != "" {
					transcription = result.Text
					confidence = result.Confidence
					break
				}
			}

			// Skip processing during silence
			if !isSpeaking && !speechDetected {
				continue
			}
		}

		// Process audio through STT
		result, err := engine.ProcessAudio(ctx, chunk)
		if err != nil {
			continue
		}

		if result != nil && !result.Partial && result.Text != "" {
			transcription = result.Text
			confidence = result.Confidence
		}
	}

	// Get final result if we haven't already
	if transcription == "" {
		result, err := engine.FinalResult()
		if err != nil {
			return nil, fmt.Errorf("failed to get final result: %w", err)
		}
		transcription = result.Text
		confidence = result.Confidence
	}

	duration := time.Since(startTime).Seconds()

	return &TranscribeAudioResult{
		Text:       transcription,
		Confidence: confidence,
		Timestamp:  time.Now(),
		Duration:   duration,
	}, nil
}

// Close closes the transcription service
func (ts *TranscriptionService) Close() error {
	// Cleanup if needed
	return nil
}
