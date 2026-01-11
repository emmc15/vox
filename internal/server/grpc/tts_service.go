package grpc

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	voxpb "github.com/emmett/vox/api/proto"
	"github.com/emmett/vox/internal/tts"
)

// TTSService implements the gRPC TTS service
type TTSService struct {
	voxpb.UnimplementedTTSServer
	engine tts.Engine
	mu     sync.Mutex
}

// NewTTSService creates a new TTS service
func NewTTSService(engine tts.Engine) *TTSService {
	return &TTSService{engine: engine}
}

// Synthesize handles text-to-speech synthesis with streaming audio output
func (s *TTSService) Synthesize(req *voxpb.SynthesizeRequest, stream grpc.ServerStreamingServer[voxpb.AudioChunk]) error {
	ctx := stream.Context()

	ttsReq := tts.SynthesizeRequest{
		Text:  req.Text,
		Voice: req.Voice,
		Speed: req.Speed,
	}

	// Stream audio chunks as they're generated
	return s.engine.Synthesize(ctx, ttsReq, func(chunk tts.AudioChunk) error {
		return stream.Send(&voxpb.AudioChunk{
			Data:       chunk.Data,
			SampleRate: int32(chunk.SampleRate),
			Channels:   int32(chunk.Channels),
		})
	})
}

// ListVoices returns available TTS voices
func (s *TTSService) ListVoices(ctx context.Context, req *voxpb.ListVoicesRequest) (*voxpb.ListVoicesResponse, error) {
	voices := s.engine.ListVoices()
	result := &voxpb.ListVoicesResponse{
		Voices:       make([]*voxpb.Voice, len(voices)),
		DefaultVoice: "en_US-lessac-medium",
	}

	for i, v := range voices {
		result.Voices[i] = &voxpb.Voice{
			Id:       v.ID,
			Name:     v.Name,
			Language: v.Language,
			Gender:   v.Gender,
		}
	}

	return result, nil
}
