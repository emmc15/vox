package grpc

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/emmett/vox/internal/stt"
)

// STTService implements the gRPC STT service
type STTService struct {
	engine stt.Engine
	mu     sync.Mutex
}

// NewSTTService creates a new STT service
func NewSTTService(engine stt.Engine) *STTService {
	return &STTService{engine: engine}
}

// AudioChunk represents incoming audio data
type AudioChunk struct {
	Data       []byte
	SampleRate int32
	Channels   int32
}

// TranscriptResult represents a transcription result
type TranscriptResult struct {
	Text        string
	IsFinal     bool
	Confidence  float32
	TimestampMs int64
}

// TranscribeStream is the streaming interface for bidirectional transcription
type TranscribeStream interface {
	Send(*TranscriptResult) error
	Recv() (*AudioChunk, error)
	Context() context.Context
}

// Transcribe handles bidirectional streaming transcription
// This will be updated to use generated proto types once protoc runs
func (s *STTService) Transcribe(stream TranscribeStream) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			// Send final result before closing
			s.mu.Lock()
			finalResult, err := s.engine.FinalResult()
			s.mu.Unlock()
			if err == nil && finalResult.Text != "" {
				stream.Send(&TranscriptResult{
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
				// Client closed stream, send final result
				s.mu.Lock()
				finalResult, err := s.engine.FinalResult()
				s.mu.Unlock()
				if err == nil && finalResult.Text != "" {
					stream.Send(&TranscriptResult{
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
				stream.Send(&TranscriptResult{
					Text:        result.Text,
					IsFinal:     !result.Partial,
					Confidence:  float32(result.Confidence),
					TimestampMs: time.Now().UnixMilli(),
				})
			}
		}
	}
}
