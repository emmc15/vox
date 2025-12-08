package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/emmett/diaz/internal/models"
)

// Server represents an MCP server
type Server struct {
	reader       io.Reader
	writer       io.Writer
	scanner      *bufio.Scanner
	capabilities Capabilities
	serverInfo   ServerInfo
	initialized  bool
	mu           sync.RWMutex

	// STT engine pool
	transcriber *TranscriptionService

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds server configuration
type Config struct {
	ServerName    string
	ServerVersion string
	ModelPath     string
	DefaultModel  string
}

// NewServer creates a new MCP server
func NewServer(cfg Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize transcription service
	transcriber, err := NewTranscriptionService(cfg.ModelPath, cfg.DefaultModel)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create transcription service: %w", err)
	}

	s := &Server{
		reader:      os.Stdin,
		writer:      os.Stdout,
		scanner:     bufio.NewScanner(os.Stdin),
		transcriber: transcriber,
		ctx:         ctx,
		cancel:      cancel,
		serverInfo: ServerInfo{
			Name:    cfg.ServerName,
			Version: cfg.ServerVersion,
		},
		capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
	}

	return s, nil
}

// Start starts the MCP server
func (s *Server) Start() error {
	log.Println("MCP server starting...")

	// Read and process requests line by line (stdio transport)
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse request
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, ParseError, "failed to parse request", err)
			continue
		}

		// Handle request
		s.handleRequest(&req)
	}

	if err := s.scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.cancel()
	if s.transcriber != nil {
		return s.transcriber.Close()
	}
	return nil
}

// handleRequest handles an MCP request
func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleListTools(req)
	case "tools/call":
		s.handleCallTool(req)
	case "ping":
		s.handlePing(req)
	default:
		s.sendError(req.ID, MethodNotFound, fmt.Sprintf("unknown method: %s", req.Method), nil)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *Request) {
	var params InitializeParams
	if err := s.parseParams(req.Params, &params); err != nil {
		s.sendError(req.ID, InvalidParams, "invalid initialize params", err)
		return
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    s.capabilities,
		ServerInfo:      s.serverInfo,
	}

	s.sendResponse(req.ID, result)
}

// handleListTools handles the tools/list request
func (s *Server) handleListTools(req *Request) {
	if !s.isInitialized() {
		s.sendError(req.ID, InvalidRequest, "server not initialized", nil)
		return
	}

	tools := s.getTools()
	result := ListToolsResult{
		Tools: tools,
	}

	s.sendResponse(req.ID, result)
}

// handleCallTool handles the tools/call request
func (s *Server) handleCallTool(req *Request) {
	if !s.isInitialized() {
		s.sendError(req.ID, InvalidRequest, "server not initialized", nil)
		return
	}

	var params CallToolParams
	if err := s.parseParams(req.Params, &params); err != nil {
		s.sendError(req.ID, InvalidParams, "invalid tool call params", err)
		return
	}

	// Route to appropriate tool handler
	switch params.Name {
	case "transcribe_audio":
		s.handleTranscribeAudio(req.ID, params.Arguments)
	case "list_models":
		s.handleListModels(req.ID)
	default:
		s.sendError(req.ID, MethodNotFound, fmt.Sprintf("unknown tool: %s", params.Name), nil)
	}
}

// handleTranscribeAudio handles the transcribe_audio tool
func (s *Server) handleTranscribeAudio(id interface{}, args map[string]interface{}) {
	// Parse arguments
	var params TranscribeAudioParams
	if err := s.parseParams(args, &params); err != nil {
		s.sendError(id, InvalidParams, "invalid transcribe_audio params", err)
		return
	}

	// Set defaults
	if params.VADThreshold == 0 {
		params.VADThreshold = 0.01
	}
	if params.VADSilenceDelay == 0 {
		params.VADSilenceDelay = 5.0
	}

	// Call transcription service
	result, err := s.transcriber.TranscribeAudio(s.ctx, params)
	if err != nil {
		s.sendError(id, InternalError, "transcription failed", err)
		return
	}

	// Format response
	content := []Content{
		{
			Type: "text",
			Text: result.Text,
		},
	}

	// Add metadata as additional content
	metadata := fmt.Sprintf("Confidence: %.2f, Duration: %.2fs", result.Confidence, result.Duration)
	content = append(content, Content{
		Type: "text",
		Text: metadata,
	})

	toolResult := CallToolResult{
		Content: content,
		IsError: false,
	}

	s.sendResponse(id, toolResult)
}

// handleListModels handles the list_models tool
func (s *Server) handleListModels(id interface{}) {
	var modelList []string
	for _, model := range models.AvailableModels {
		modelList = append(modelList, fmt.Sprintf("%s (%s)", model.Name, model.Size))
	}

	text := "Available models:\n" + joinStrings(modelList, "\n")

	toolResult := CallToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: text,
			},
		},
		IsError: false,
	}

	s.sendResponse(id, toolResult)
}

// handlePing handles ping request
func (s *Server) handlePing(req *Request) {
	s.sendResponse(req.ID, map[string]string{"status": "ok"})
}

// Helper methods

func (s *Server) isInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

func (s *Server) parseParams(params interface{}, target interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (s *Server) sendResponse(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeResponse(resp)
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.writeResponse(resp)
}

func (s *Server) writeResponse(resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Write response followed by newline (stdio transport)
	_, err = s.writer.Write(append(data, '\n'))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (s *Server) getTools() []Tool {
	return []Tool{
		{
			Name:        "transcribe_audio",
			Description: "Transcribe audio to text using Vosk STT. Accepts base64-encoded audio data and returns transcription when silence is detected.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"audio": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded audio data (16kHz, mono, 16-bit PCM)",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Model name to use (optional, uses default if not specified)",
					},
					"vad_enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable Voice Activity Detection (default: true)",
					},
					"vad_threshold": map[string]interface{}{
						"type":        "number",
						"description": "VAD energy threshold 0.001-0.1 (default: 0.01)",
					},
					"vad_silence_delay": map[string]interface{}{
						"type":        "number",
						"description": "Seconds to wait after speech before finalizing (default: 5.0)",
					},
				},
				"required": []string{"audio"},
			},
		},
		{
			Name:        "list_models",
			Description: "List all available Vosk speech recognition models",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
