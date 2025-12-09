package mcp

import (
	"context"
	"fmt"

	"github.com/emmett/diaz/internal/stt"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	ServerName    string
	ServerVersion string
	ModelPath     string
	DefaultModel  string
}

type Server struct {
	config    Config
	mcpServer *sdk.Server
	sttEngine stt.Engine
}

func NewServer(cfg Config) (*Server, error) {
	s := &Server{
		config: cfg,
	}

	// Initialize STT engine
	s.sttEngine = stt.NewVoskEngine()
	sttConfig := stt.DefaultConfig(cfg.ModelPath)
	if err := s.sttEngine.Initialize(sttConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize STT engine: %w", err)
	}

	// Create MCP server
	s.mcpServer = sdk.NewServer(&sdk.Implementation{
		Name:    cfg.ServerName,
		Version: cfg.ServerVersion,
	}, nil)

	// Register tools
	s.registerTools()

	return s, nil
}

func (s *Server) Start() error {
	return s.mcpServer.Run(context.Background(), &sdk.StdioTransport{})
}

func (s *Server) Stop() error {
	if s.sttEngine != nil {
		s.sttEngine.Close()
	}
	return nil
}

func (s *Server) registerTools() {
	sdk.AddTool(s.mcpServer, &sdk.Tool{
		Name:        "transcribe_audio",
		Description: "Transcribe audio with Voice Activity Detection support",
	}, s.handleTranscribeAudio)

	sdk.AddTool(s.mcpServer, &sdk.Tool{
		Name:        "list_models",
		Description: "List available Vosk models",
	}, s.handleListModels)
}
