package grpc

import (
	"context"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"

	voxpb "github.com/emmett/vox/api/proto"
	"github.com/emmett/vox/internal/stt"
)

// STTService implements the gRPC STT service
type STTService struct {
	voxpb.UnimplementedSTTServer
	engine stt.Engine
	mu     sync.Mutex
}

// NewSTTService creates a new STT service
func NewSTTService(engine stt.Engine) *STTService {
	return &STTService{engine: engine}
}

// Transcribe handles bidirectional streaming transcription
func (s *STTService) Transcribe(stream grpc.BidiStreamingServer[voxpb.AudioChunk, voxpb.TranscriptResult]) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			finalResult, err := s.engine.FinalResult()
			s.mu.Unlock()
			if err == nil && finalResult.Text != "" {
				stream.Send(&voxpb.TranscriptResult{
					Text:        finalResult.Text,
					IsFinal:     true,
					Confidence:  float32(finalResult.Confidence),
					TimestampMs: time.Now().UnixMilli(),
				})
			}
			return ctx.Err()

		default:
			chunk, err := stream.Recv()
			if err == io.EOF {
				s.mu.Lock()
				finalResult, err := s.engine.FinalResult()
				s.mu.Unlock()
				if err == nil && finalResult.Text != "" {
					stream.Send(&voxpb.TranscriptResult{
						Text:        finalResult.Text,
						IsFinal:     true,
						Confidence:  float32(finalResult.Confidence),
						TimestampMs: time.Now().UnixMilli(),
					})
				}
				return nil
			}
			if err != nil {
				return err
			}

			// Process audio chunk
			s.mu.Lock()
			result, err := s.engine.ProcessAudio(ctx, chunk.Data)
			s.mu.Unlock()
			if err != nil {
				return err
			}

			if result != nil && result.Text != "" {
				stream.Send(&voxpb.TranscriptResult{
					Text:        result.Text,
					IsFinal:     !result.Partial,
					Confidence:  float32(result.Confidence),
					TimestampMs: time.Now().UnixMilli(),
				})
			}
		}
	}
}

// ListModels returns available STT models
func (s *STTService) ListModels(ctx context.Context, req *voxpb.ListModelsRequest) (*voxpb.ListModelsResponse, error) {
	// TODO: Implement actual model listing from models package
	return &voxpb.ListModelsResponse{
		Models: []*voxpb.Model{
			{
				Name:        "vosk-model-small-en-us-0.15",
				Description: "Small English model for general transcription",
				SizeBytes:   40000000,
				Downloaded:  true,
			},
		},
		DefaultModel: "vosk-model-small-en-us-0.15",
	}, nil
}
