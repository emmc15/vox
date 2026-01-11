package tts

import "context"

// Engine defines the interface for text-to-speech engines
type Engine interface {
	// Initialize sets up the TTS engine with the given config
	Initialize(config Config) error

	// Synthesize converts text to audio, streaming chunks via callback
	Synthesize(ctx context.Context, req SynthesizeRequest, callback AudioCallback) error

	// ListVoices returns available voices
	ListVoices() []Voice

	// Close releases resources
	Close() error

	// IsInitialized returns true if engine is ready
	IsInitialized() bool
}

// Config holds TTS engine configuration
type Config struct {
	ModelPath string
	// Voice-specific settings
	DefaultVoice string
}

// SynthesizeRequest contains text-to-speech parameters
type SynthesizeRequest struct {
	Text  string
	Voice string
	Speed float32 // 1.0 = normal, 0.5 = half speed, 2.0 = double
}

// AudioChunk represents a chunk of synthesized audio
type AudioChunk struct {
	Data       []byte
	SampleRate int
	Channels   int
}

// AudioCallback is called for each audio chunk during synthesis
type AudioCallback func(chunk AudioChunk) error

// Voice represents an available TTS voice
type Voice struct {
	ID       string
	Name     string
	Language string
	Gender   string
}

// DefaultConfig returns default TTS configuration
func DefaultConfig(modelPath string) Config {
	return Config{
		ModelPath:    modelPath,
		DefaultVoice: "default",
	}
}
