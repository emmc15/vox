package audio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
)

// MalgoCapturer implements the Capturer interface using malgo
type MalgoCapturer struct {
	config       CaptureConfig
	device       *malgo.Device
	malgoContext *malgo.AllocatedContext
	samples      chan AudioSample
	errors       chan error
	running      bool
	mu           sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewMalgoCapturer creates a new malgo-based audio capturer
func NewMalgoCapturer(config CaptureConfig) (*MalgoCapturer, error) {
	return &MalgoCapturer{
		config:   config,
		samples:  make(chan AudioSample, 10), // Buffer up to 10 samples
		errors:   make(chan error, 10),
		stopChan: make(chan struct{}),
	}, nil
}

// Start begins audio capture
func (m *MalgoCapturer) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("capturer is already running")
	}
	m.running = true
	m.mu.Unlock()

	// Initialize malgo context
	malgoCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		return fmt.Errorf("failed to initialize malgo context: %w", err)
	}
	m.malgoContext = malgoCtx

	// Configure device
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16 // 16-bit signed integer
	deviceConfig.Capture.Channels = m.config.Channels
	deviceConfig.SampleRate = m.config.SampleRate
	deviceConfig.PeriodSizeInFrames = m.config.BufferFrames

	// Data callback - called when audio data is available
	var dataCallback malgo.DeviceCallbacks
	dataCallback.Data = func(pOutputSample, pInputSamples []byte, framecount uint32) {
		// Copy the input samples to avoid data races
		dataCopy := make([]byte, len(pInputSamples))
		copy(dataCopy, pInputSamples)

		sample := AudioSample{
			Data:      dataCopy,
			Timestamp: time.Now(),
			Frames:    framecount,
		}

		// Non-blocking send to samples channel
		select {
		case m.samples <- sample:
		default:
			// Channel is full, log or handle overflow
			select {
			case m.errors <- fmt.Errorf("sample buffer overflow, dropping frames"):
			default:
			}
		}
	}

	// Initialize device
	device, err := malgo.InitDevice(m.malgoContext.Context, deviceConfig, dataCallback)
	if err != nil {
		m.malgoContext.Uninit()
		m.malgoContext.Free()
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		return fmt.Errorf("failed to initialize device: %w", err)
	}
	m.device = device

	// Start the device
	err = device.Start()
	if err != nil {
		device.Uninit()
		m.malgoContext.Uninit()
		m.malgoContext.Free()
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		return fmt.Errorf("failed to start device: %w", err)
	}

	// Start context monitoring goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		select {
		case <-ctx.Done():
			m.Stop()
		case <-m.stopChan:
			return
		}
	}()

	return nil
}

// Stop stops audio capture
func (m *MalgoCapturer) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	// Signal stop
	close(m.stopChan)

	// Stop the device
	if m.device != nil {
		err := m.device.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop device: %w", err)
		}
		m.device.Uninit()
	}

	// Uninitialize malgo context
	if m.malgoContext != nil {
		m.malgoContext.Uninit()
		m.malgoContext.Free()
	}

	// Wait for goroutines
	m.wg.Wait()

	// Close channels
	close(m.samples)
	close(m.errors)

	return nil
}

// Samples returns a channel that receives audio samples
func (m *MalgoCapturer) Samples() <-chan AudioSample {
	return m.samples
}

// Errors returns a channel that receives capture errors
func (m *MalgoCapturer) Errors() <-chan error {
	return m.errors
}

// IsRunning returns true if capture is currently active
func (m *MalgoCapturer) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetDeviceInfo returns information about the capture device
func (m *MalgoCapturer) GetDeviceInfo() (malgo.DeviceInfo, error) {
	if m.malgoContext == nil {
		return malgo.DeviceInfo{}, fmt.Errorf("malgo context not initialized")
	}

	infos, err := m.malgoContext.Devices(malgo.Capture)
	if err != nil {
		return malgo.DeviceInfo{}, fmt.Errorf("failed to get devices: %w", err)
	}

	if len(infos) == 0 {
		return malgo.DeviceInfo{}, fmt.Errorf("no capture devices found")
	}

	// Return the first (default) device info
	return infos[0], nil
}
