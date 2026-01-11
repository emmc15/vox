package mcp

import (
	"context"
	"fmt"

	"github.com/emmett/vox/internal/audio"
	"github.com/emmett/vox/internal/models"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type TranscribeArgs struct {
	Model           string  `json:"model,omitempty"`
	VadEnabled      *bool   `json:"vad_enabled,omitempty"`
	VadThreshold    float64 `json:"vad_threshold,omitempty"`
	VadSilenceDelay float64 `json:"vad_silence_delay,omitempty"`
}

type ListModelsArgs struct{}

func (s *Server) handleTranscribeAudio(ctx context.Context, req *sdk.CallToolRequest, args TranscribeArgs) (*sdk.CallToolResult, any, error) {
	// Start audio capture
	capturer, err := audio.NewCapturer(audio.DefaultConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create capturer: %w", err)
	}
	defer capturer.Stop()

	// Setup VAD
	vadConfig := audio.DefaultVADConfig()
	// Use args if provided, otherwise use server config defaults
	if args.VadThreshold > 0 {
		vadConfig.EnergyThreshold = args.VadThreshold
	} else if s.config.VADThreshold > 0 {
		vadConfig.EnergyThreshold = s.config.VADThreshold
	}
	vad := audio.NewVAD(vadConfig)

	silenceCount := 0
	speechStarted := false
	var audioBuffer []byte

	// Start capture
	if err := capturer.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start capture: %w", err)
	}

	// Stop capturing when we detect silence after speech
	for {
		select {
		case sample := <-capturer.Samples():
			audioBuffer = append(audioBuffer, sample.Data...)
			isSpeech, _, speechEnded := vad.ProcessFrame(sample.Data)
			if isSpeech {
				speechStarted = true
			}
			if speechEnded && speechStarted {
				silenceCount++
				if silenceCount >= 1 {
					goto transcribe
				}
			}
		case err := <-capturer.Errors():
			return nil, nil, fmt.Errorf("capture error: %w", err)
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

transcribe:
	capturer.Stop()

	// Process audio through STT engine
	_, err = s.sttEngine.ProcessAudio(ctx, audioBuffer)
	if err != nil {
		return nil, nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Get final result
	finalResult, err := s.sttEngine.FinalResult()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get final result: %w", err)
	}

	return &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: finalResult.Text},
			//&sdk.TextContent{Text: fmt.Sprintf("Confidence: %.2f, Duration: N/A", finalResult.Confidence)},
		},
	}, nil, nil
}

func (s *Server) handleListModels(ctx context.Context, req *sdk.CallToolRequest, args ListModelsArgs) (*sdk.CallToolResult, any, error) {
	downloaded, err := models.ListDownloadedModels()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list models: %w", err)
	}

	content := []sdk.Content{
		&sdk.TextContent{Text: fmt.Sprintf("Downloaded models (%d):", len(downloaded))},
	}

	for _, model := range downloaded {
		content = append(content, &sdk.TextContent{Text: fmt.Sprintf("- %s", model)})
	}

	return &sdk.CallToolResult{Content: content}, nil, nil
}
