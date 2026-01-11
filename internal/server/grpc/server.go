package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/emmett/vox/internal/stt"
)

// Server wraps the gRPC server and services
type Server struct {
	grpcServer *grpc.Server
	sttEngine  stt.Engine
	port       int
}

// Config holds server configuration
type Config struct {
	Port      int
	ModelPath string
}

// NewServer creates a new gRPC server
func NewServer(cfg Config) (*Server, error) {
	// Initialize STT engine
	engine := stt.NewVoskEngine()
	sttCfg := stt.DefaultConfig(cfg.ModelPath)
	if err := engine.Initialize(sttCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize STT engine: %w", err)
	}

	s := &Server{
		grpcServer: grpc.NewServer(),
		sttEngine:  engine,
		port:       cfg.Port,
	}

	// Register services
	sttService := NewSTTService(engine)
	RegisterSTTServer(s.grpcServer, sttService)

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
}

// RegisterSTTServer is a placeholder until proto is generated
func RegisterSTTServer(s *grpc.Server, srv *STTService) {
	// Will be replaced by generated code: voxpb.RegisterSTTServer(s, srv)
}
