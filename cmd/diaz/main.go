package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emmett/diaz/internal/audio"
	"github.com/emmett/diaz/internal/output"
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
	// List available audio devices
	fmt.Println("Detecting audio devices...")
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

	// Initialize audio capture
	config := audio.DefaultConfig()
	capturer, err := audio.NewCapturer(config)
	if err != nil {
		return fmt.Errorf("failed to create capturer: %w", err)
	}

	// Initialize output
	out := output.DefaultConsoleOutput()
	out.Info("Initializing audio capture...")

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nStopping...")
		cancel()
	}()

	// Start capturing
	out.Info(fmt.Sprintf("Starting capture (sample rate: %d Hz, channels: %d, bit depth: %d)",
		config.SampleRate, config.Channels, config.BitDepth))
	out.Info("Press Ctrl+C to stop")
	fmt.Println()

	err = capturer.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start capture: %w", err)
	}
	defer capturer.Stop()

	// Process audio samples
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var totalSamples uint64
	var totalBytes uint64
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			out.Info("Capture stopped")
			duration := time.Since(startTime)
			out.Info(fmt.Sprintf("Captured %d samples (%d bytes) in %s",
				totalSamples, totalBytes, duration.Round(time.Second)))
			return nil

		case sample, ok := <-capturer.Samples():
			if !ok {
				return nil
			}

			totalSamples++
			totalBytes += uint64(len(sample.Data))

			// Calculate audio level (RMS)
			level := calculateRMS(sample.Data)
			out.WriteAudioLevel(level)

		case err, ok := <-capturer.Errors():
			if !ok {
				return nil
			}
			out.Error(fmt.Sprintf("Capture error: %v", err))

		case <-ticker.C:
			// Periodic update (if needed)
		}
	}
}

// calculateRMS calculates the Root Mean Square (RMS) of audio samples
// This gives us the audio level/volume
func calculateRMS(data []byte) float64 {
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

	rms := math.Sqrt(sum / float64(sampleCount))
	return rms
}
