package audio

import (
	"math"
)

// VADConfig holds configuration for Voice Activity Detection
type VADConfig struct {
	// EnergyThreshold is the minimum energy level to consider as speech
	// Typical values: 0.001 to 0.1 (lower = more sensitive)
	EnergyThreshold float64

	// SilenceFrames is the number of consecutive silent frames before considering pause
	// At 16kHz with 30ms frames: 10 frames = 300ms of silence
	SilenceFrames int

	// SpeechFrames is the number of consecutive speech frames before triggering speech start
	// At 16kHz with 30ms frames: 3 frames = 90ms of speech
	SpeechFrames int
}

// DefaultVADConfig returns a default VAD configuration
func DefaultVADConfig() VADConfig {
	return VADConfig{
		EnergyThreshold: 0.01,   // Moderate sensitivity
		SilenceFrames:   33 * 8, // 1s of silence
		SpeechFrames:    3,      // 90ms of speech
	}
}

// VAD (Voice Activity Detector) detects speech vs silence in audio
type VAD struct {
	config              VADConfig
	silenceFrameCount   int
	speechFrameCount    int
	isSpeaking          bool
	lastSpeechDetection bool
}

// NewVAD creates a new voice activity detector
func NewVAD(config VADConfig) *VAD {
	return &VAD{
		config:     config,
		isSpeaking: false,
	}
}

// ProcessFrame processes an audio frame and returns whether speech is active
// Returns: (isSpeechActive, speechStarted, speechEnded)
func (v *VAD) ProcessFrame(audioData []byte) (bool, bool, bool) {
	// Calculate energy (RMS) of the audio frame
	energy := calculateEnergy(audioData)

	// DEBUG: Log energy levels
	// log.Printf("[VAD] Energy: %.6f | Threshold: %.6f | Speech: %v", energy, v.config.EnergyThreshold, energy > v.config.EnergyThreshold)

	// Detect if current frame contains speech based on energy
	frameHasSpeech := energy > v.config.EnergyThreshold

	speechStarted := false
	speechEnded := false

	if frameHasSpeech {
		// Speech detected in frame
		v.speechFrameCount++
		v.silenceFrameCount = 0

		// Check if we've crossed the speech threshold
		if !v.isSpeaking && v.speechFrameCount >= v.config.SpeechFrames {
			v.isSpeaking = true
			speechStarted = true
		}
	} else {
		// Silence detected in frame
		v.silenceFrameCount++
		v.speechFrameCount = 0

		// Check if we've crossed the silence threshold
		if v.isSpeaking && v.silenceFrameCount >= v.config.SilenceFrames {
			v.isSpeaking = false
			speechEnded = true
		}
	}

	v.lastSpeechDetection = frameHasSpeech
	return v.isSpeaking, speechStarted, speechEnded
}

// IsSpeaking returns whether speech is currently active
func (v *VAD) IsSpeaking() bool {
	return v.isSpeaking
}

// Reset resets the VAD state
func (v *VAD) Reset() {
	v.silenceFrameCount = 0
	v.speechFrameCount = 0
	v.isSpeaking = false
	v.lastSpeechDetection = false
}

// calculateEnergy calculates the energy (RMS) of an audio buffer
func calculateEnergy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Assuming 16-bit signed integers (2 bytes per sample)
	var sum float64
	sampleCount := len(data) / 2

	for i := 0; i < sampleCount; i++ {
		// Read 16-bit sample (little-endian)
		sample := int16(data[i*2]) | int16(data[i*2+1])<<8
		normalized := float64(sample) / 32768.0 // Normalize to -1.0 to 1.0
		sum += normalized * normalized
	}

	if sampleCount == 0 {
		return 0
	}

	// Return RMS (Root Mean Square) energy
	return math.Sqrt(sum / float64(sampleCount))
}

// GetEnergyLevel returns the energy threshold for debugging/calibration
func (v *VAD) GetEnergyLevel(audioData []byte) float64 {
	return calculateEnergy(audioData)
}
