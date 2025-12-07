package audio

import (
	"context"
	"time"
)

// CaptureConfig holds configuration for audio capture
type CaptureConfig struct {
	// SampleRate is the number of samples per second (Hz)
	// Common values: 16000 (recommended for STT), 44100, 48000
	SampleRate uint32

	// Channels is the number of audio channels
	// 1 = mono (recommended for STT), 2 = stereo
	Channels uint32

	// BitDepth is the number of bits per sample
	// Common values: 16, 24, 32
	BitDepth uint32

	// BufferFrames is the number of frames per buffer
	// Smaller = lower latency, higher CPU usage
	BufferFrames uint32

	// DeviceID is the audio device identifier
	// Empty string = use default device
	DeviceID string
}

// DefaultConfig returns a default configuration optimized for speech recognition
func DefaultConfig() CaptureConfig {
	return CaptureConfig{
		SampleRate:   16000, // 16kHz is optimal for most STT engines
		Channels:     1,     // Mono
		BitDepth:     16,    // 16-bit
		BufferFrames: 480,   // 30ms at 16kHz
		DeviceID:     "",    // Default device
	}
}

// AudioSample represents a chunk of captured audio data
type AudioSample struct {
	Data      []byte    // Raw audio data
	Timestamp time.Time // When the sample was captured
	Frames    uint32    // Number of audio frames in this sample
}

// Capturer is the interface for audio capture implementations
type Capturer interface {
	// Start begins audio capture
	Start(ctx context.Context) error

	// Stop stops audio capture
	Stop() error

	// Samples returns a channel that receives audio samples
	Samples() <-chan AudioSample

	// Errors returns a channel that receives capture errors
	Errors() <-chan error

	// IsRunning returns true if capture is currently active
	IsRunning() bool
}

// NewCapturer creates a new audio capturer with the given configuration
func NewCapturer(config CaptureConfig) (Capturer, error) {
	return NewMalgoCapturer(config)
}
