package mcp

import (
	"context"
	"fmt"

	"github.com/emmett/vox/internal/stt"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	ServerName      string
	ServerVersion   string
	ModelPath       string
	DefaultModel    string
	VADThreshold    float64
	VADSilenceDelay float64
	VADEnabled      bool
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
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model":             map[string]string{"type": "string"},
				"vad_enabled":       map[string]interface{}{"type": []string{"boolean", "null"}},
				"vad_threshold":     map[string]string{"type": "number"},
				"vad_silence_delay": map[string]string{"type": "number"},
			},
		},
	}, s.handleTranscribeAudio)

	sdk.AddTool(s.mcpServer, &sdk.Tool{
		Name:        "list_models",
		Description: "List available Vosk models",
	}, s.handleListModels)
}
