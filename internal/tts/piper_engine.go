package tts

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// PiperEngine implements the Engine interface using Piper TTS CLI
type PiperEngine struct {
	config      Config
	mu          sync.Mutex
	initialized bool
	voices      []Voice
}

// NewPiperEngine creates a new Piper TTS engine
func NewPiperEngine() *PiperEngine {
	return &PiperEngine{
		voices: []Voice{
			{ID: "en_US-lessac-medium", Name: "Lessac", Language: "en-US", Gender: "female"},
			{ID: "en_US-ryan-medium", Name: "Ryan", Language: "en-US", Gender: "male"},
		},
	}
}

// Initialize sets up the Piper engine
func (p *PiperEngine) Initialize(config Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("engine already initialized")
	}

	p.config = config
	p.initialized = true

	return nil
}

// Synthesize converts text to audio using piper CLI
func (p *PiperEngine) Synthesize(ctx context.Context, req SynthesizeRequest, callback AudioCallback) error {
	p.mu.Lock()
	if !p.initialized {
		p.mu.Unlock()
		return fmt.Errorf("engine not initialized")
	}
	modelPath := p.config.ModelPath
	p.mu.Unlock()

	if modelPath == "" {
		return fmt.Errorf("TTS model path not configured")
	}

	// Build piper command
	args := []string{
		"--model", modelPath,
		"--output_raw",
	}

	// Add speed if not default
	if req.Speed > 0 && req.Speed != 1.0 {
		args = append(args, "--length_scale", fmt.Sprintf("%.2f", 1.0/req.Speed))
	}

	cmd := exec.CommandContext(ctx, "piper", args...)

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start piper process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start piper: %w", err)
	}

	// Write text to stdin and close
	go func() {
		io.WriteString(stdin, req.Text)
		stdin.Close()
	}()

	// Read audio output in chunks
	reader := bufio.NewReader(stdout)
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		default:
		}

		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			cmd.Wait()
			return fmt.Errorf("error reading piper output: %w", err)
		}

		if n > 0 {
			chunk := AudioChunk{
				Data:       append([]byte(nil), buf[:n]...),
				SampleRate: 22050, // Piper default
				Channels:   1,
			}
			if err := callback(chunk); err != nil {
				cmd.Process.Kill()
				return err
			}
		}
	}

	return cmd.Wait()
}

// ListVoices returns available voices
func (p *PiperEngine) ListVoices() []Voice {
	return p.voices
}

// Close releases resources
func (p *PiperEngine) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil
	}

	p.initialized = false
	return nil
}

// IsInitialized returns true if engine is ready
func (p *PiperEngine) IsInitialized() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.initialized
}
