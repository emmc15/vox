package grpc

import (
	"context"
	"sync"

	"github.com/emmett/vox/internal/tts"
)

// TTSService implements the gRPC TTS service
type TTSService struct {
	engine tts.Engine
	mu     sync.Mutex
}

// NewTTSService creates a new TTS service
func NewTTSService(engine tts.Engine) *TTSService {
	return &TTSService{engine: engine}
}

// SynthesizeRequest for TTS
type SynthesizeRequest struct {
	Text  string
	Voice string
	Speed float32
}

// SynthesizeStream is the streaming interface for audio output
type SynthesizeStream interface {
	Send(*AudioChunk) error
	Context() context.Context
}

// Synthesize handles text-to-speech synthesis with streaming audio output
func (s *TTSService) Synthesize(req *SynthesizeRequest, stream SynthesizeStream) error {
	ctx := stream.Context()

	ttsReq := tts.SynthesizeRequest{
		Text:  req.Text,
		Voice: req.Voice,
		Speed: req.Speed,
	}

	// Stream audio chunks as they're generated
	return s.engine.Synthesize(ctx, ttsReq, func(chunk tts.AudioChunk) error {
		return stream.Send(&AudioChunk{
			Data:       chunk.Data,
			SampleRate: int32(chunk.SampleRate),
			Channels:   int32(chunk.Channels),
		})
	})
}

// Voice represents a TTS voice
type Voice struct {
	ID       string
	Name     string
	Language string
	Gender   string
}

// ListVoicesResponse contains available voices
type ListVoicesResponse struct {
	Voices       []Voice
	DefaultVoice string
}

// ListVoices returns available TTS voices
func (s *TTSService) ListVoices() *ListVoicesResponse {
	voices := s.engine.ListVoices()
	result := &ListVoicesResponse{
		Voices:       make([]Voice, len(voices)),
		DefaultVoice: "en_US-lessac-medium",
	}

	for i, v := range voices {
		result.Voices[i] = Voice{
			ID:       v.ID,
			Name:     v.Name,
			Language: v.Language,
			Gender:   v.Gender,
		}
	}

	return result
}
