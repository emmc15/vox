package stt

import "context"

// Result represents a speech recognition result
type Result struct {
	// Text is the recognized text
	Text string

	// Partial indicates if this is a partial result (still processing)
	// or a final result (sentence/phrase complete)
	Partial bool

	// Confidence is the recognition confidence (0.0 to 1.0)
	Confidence float64

	// Alternatives contains alternative recognition results
	Alternatives []Alternative
}

// Alternative represents an alternative recognition result
type Alternative struct {
	Text       string
	Confidence float64
}

// Config holds configuration for the STT engine
type Config struct {
	// ModelPath is the path to the STT model directory
	ModelPath string

	// SampleRate is the audio sample rate in Hz
	SampleRate int

	// MaxAlternatives is the maximum number of alternative results to return
	MaxAlternatives int

	// ShowWords enables word-level timestamps
	ShowWords bool
}

// Engine is the interface for speech-to-text engines
type Engine interface {
	// Initialize initializes the engine with the given configuration
	Initialize(config Config) error

	// ProcessAudio processes audio data and returns recognition results
	// Audio data should be 16-bit PCM
	ProcessAudio(ctx context.Context, audioData []byte) (*Result, error)

	// FinalResult returns the final result and resets the recognizer
	FinalResult() (*Result, error)

	// Reset resets the recognizer state
	Reset() error

	// Close releases resources
	Close() error

	// IsInitialized returns true if the engine is initialized
	IsInitialized() bool
}

// DefaultConfig returns a default STT configuration
func DefaultConfig(modelPath string) Config {
	return Config{
		ModelPath:       modelPath,
		SampleRate:      16000,
		MaxAlternatives: 0,
		ShowWords:       false,
	}
}
