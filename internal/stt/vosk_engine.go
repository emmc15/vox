package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	vosk "github.com/alphacep/vosk-api/go"
)

// VoskEngine implements the Engine interface using Vosk
type VoskEngine struct {
	model       *vosk.VoskModel
	recognizer  *vosk.VoskRecognizer
	config      Config
	mu          sync.Mutex
	initialized bool
}

// VoskResult represents the JSON result from Vosk
type VoskResult struct {
	Text   string `json:"text"`
	Result []struct {
		Conf  float64 `json:"conf"`
		End   float64 `json:"end"`
		Start float64 `json:"start"`
		Word  string  `json:"word"`
	} `json:"result,omitempty"`
	Partial string `json:"partial,omitempty"`
}

// NewVoskEngine creates a new Vosk STT engine
func NewVoskEngine() *VoskEngine {
	return &VoskEngine{}
}

// Initialize initializes the Vosk engine
func (v *VoskEngine) Initialize(config Config) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.initialized {
		return fmt.Errorf("engine already initialized")
	}

	// Set log level (0 = errors only, higher = more verbose)
	vosk.SetLogLevel(-1) // Suppress logs

	// Load the model
	model, err := vosk.NewModel(config.ModelPath)
	if err != nil {
		return fmt.Errorf("failed to load model from %s: %w", config.ModelPath, err)
	}
	if model == nil {
		return fmt.Errorf("failed to load model from %s: model returned nil", config.ModelPath)
	}
	v.model = model

	// Create recognizer
	recognizer, err := vosk.NewRecognizer(model, float64(config.SampleRate))
	if err != nil {
		model.Free()
		return fmt.Errorf("failed to create recognizer: %w", err)
	}
	v.recognizer = recognizer

	// Configure recognizer
	if config.MaxAlternatives > 0 {
		v.recognizer.SetMaxAlternatives(config.MaxAlternatives)
	}
	// Always enable word results to get confidence scores
	v.recognizer.SetWords(1)

	v.config = config
	v.initialized = true

	return nil
}

// ProcessAudio processes audio data and returns recognition results
func (v *VoskEngine) ProcessAudio(ctx context.Context, audioData []byte) (*Result, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.initialized {
		return nil, fmt.Errorf("engine not initialized")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Accept waveform data
	state := v.recognizer.AcceptWaveform(audioData)

	var result Result

	if state > 0 {
		// Final result available
		resultJSON := v.recognizer.Result()
		var voskResult VoskResult
		if err := json.Unmarshal([]byte(resultJSON), &voskResult); err != nil {
			return nil, fmt.Errorf("failed to parse result: %w", err)
		}

		result.Text = voskResult.Text
		result.Partial = false
		result.Confidence = calculateAverageConfidence(voskResult)
	} else {
		// Partial result
		partialJSON := v.recognizer.PartialResult()
		var voskResult VoskResult
		if err := json.Unmarshal([]byte(partialJSON), &voskResult); err != nil {
			return nil, fmt.Errorf("failed to parse partial result: %w", err)
		}

		result.Text = voskResult.Partial
		result.Partial = true
		result.Confidence = 0.0
	}

	return &result, nil
}

// FinalResult returns the final result and resets the recognizer
func (v *VoskEngine) FinalResult() (*Result, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.initialized {
		return nil, fmt.Errorf("engine not initialized")
	}

	// Get final result
	resultJSON := v.recognizer.FinalResult()
	var voskResult VoskResult
	if err := json.Unmarshal([]byte(resultJSON), &voskResult); err != nil {
		return nil, fmt.Errorf("failed to parse final result: %w", err)
	}

	result := Result{
		Text:       voskResult.Text,
		Partial:    false,
		Confidence: calculateAverageConfidence(voskResult),
	}

	return &result, nil
}

// Reset resets the recognizer state
func (v *VoskEngine) Reset() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.initialized {
		return fmt.Errorf("engine not initialized")
	}

	// Vosk automatically resets after FinalResult
	// We can also create a new recognizer if needed
	return nil
}

// Close releases resources
func (v *VoskEngine) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.initialized {
		return nil
	}

	// Free recognizer
	if v.recognizer != nil {
		v.recognizer.Free()
		v.recognizer = nil
	}

	// Free model
	if v.model != nil {
		v.model.Free()
		v.model = nil
	}

	v.initialized = false
	return nil
}

// IsInitialized returns true if the engine is initialized
func (v *VoskEngine) IsInitialized() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.initialized
}

// calculateAverageConfidence calculates the average confidence from word results
func calculateAverageConfidence(result VoskResult) float64 {
	if len(result.Result) == 0 {
		return 0.0
	}

	var sum float64
	for _, word := range result.Result {
		sum += word.Conf
	}

	return sum / float64(len(result.Result))
}
