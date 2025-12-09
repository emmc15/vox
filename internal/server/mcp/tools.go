package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/emmett/diaz/internal/models"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type TranscribeArgs struct {
	Audio           string  `json:"audio" jsonschema:"required,description=Base64-encoded audio data (16kHz mono 16-bit PCM)"`
	Model           string  `json:"model,omitempty" jsonschema:"description=Model name to use for transcription"`
	VadEnabled      *bool   `json:"vad_enabled,omitempty" jsonschema:"description=Enable Voice Activity Detection (default: true)"`
	VadThreshold    float64 `json:"vad_threshold,omitempty" jsonschema:"description=VAD energy threshold 0.001-0.1 (default: 0.01)"`
	VadSilenceDelay float64 `json:"vad_silence_delay,omitempty" jsonschema:"description=Seconds to wait after speech ends (default: 5.0)"`
}

type ListModelsArgs struct{}

func (s *Server) handleTranscribeAudio(ctx context.Context, req *sdk.CallToolRequest, args TranscribeArgs) (*sdk.CallToolResult, any, error) {
	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(args.Audio)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid base64 audio: %w", err)
	}

	// TODO: Process VAD parameters from args

	// Process audio through STT engine
	_, err = s.sttEngine.ProcessAudio(ctx, audioData)
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
			&sdk.TextContent{Text: fmt.Sprintf("Confidence: %.2f, Duration: N/A", finalResult.Confidence)},
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
