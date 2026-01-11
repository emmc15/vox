package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	voxpb "github.com/emmett/vox/api/proto"
	"github.com/emmett/vox/internal/stt"
	"github.com/emmett/vox/internal/tts"
)

// Server wraps the gRPC server and services
type Server struct {
	grpcServer *grpc.Server
	sttEngine  stt.Engine
	ttsEngine  tts.Engine
	port       int
}

// Config holds server configuration
type Config struct {
	Port         int
	STTModelPath string
	TTSModelPath string
}

// NewServer creates a new gRPC server
func NewServer(cfg Config) (*Server, error) {
	// Initialize STT engine
	sttEngine := stt.NewVoskEngine()
	sttCfg := stt.DefaultConfig(cfg.STTModelPath)
	if err := sttEngine.Initialize(sttCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize STT engine: %w", err)
	}

	// Initialize TTS engine
	ttsEngine := tts.NewPiperEngine()
	ttsCfg := tts.DefaultConfig(cfg.TTSModelPath)
	if err := ttsEngine.Initialize(ttsCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize TTS engine: %w", err)
	}

	s := &Server{
		grpcServer: grpc.NewServer(),
		sttEngine:  sttEngine,
		ttsEngine:  ttsEngine,
		port:       cfg.Port,
	}

	// Register services
	sttService := NewSTTService(sttEngine)
	voxpb.RegisterSTTServer(s.grpcServer, sttService)

	ttsService := NewTTSService(ttsEngine)
	voxpb.RegisterTTSServer(s.grpcServer, ttsService)

	// Enable reflection for grpcurl
	reflection.Register(s.grpcServer)

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	fmt.Printf("gRPC server listening on :%d\n", s.port)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
	s.sttEngine.Close()
	s.ttsEngine.Close()
}
