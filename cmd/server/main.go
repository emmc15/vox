package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/emmett/vox/internal/app"
	"github.com/emmett/vox/internal/models"
	grpcserver "github.com/emmett/vox/internal/server/grpc"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

var (
	port        = flag.Int("port", 50051, "gRPC server port")
	modelName   = flag.String("model", "", "STT model name (default: vosk-model-small-en-us-0.15)")
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Vox gRPC Server v%s\n", Version)
		fmt.Printf("  Commit:  %s\n", GitCommit)
		fmt.Printf("  Branch:  %s\n", GitBranch)
		fmt.Printf("  Built:   %s\n", BuildTime)
		os.Exit(0)
	}

	fmt.Printf("Vox gRPC Server v%s (commit: %s)\n", Version, GitCommit)

	// Resolve model
	mgr := app.NewModelManager()
	selectedModel, err := mgr.SelectModel(*modelName, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error selecting model: %v\n", err)
		os.Exit(1)
	}

	selectedModel, err = mgr.EnsureModel(selectedModel, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get model path
	modelPath, err := models.GetModelPath(selectedModel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting model path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Using model: %s\n", selectedModel)

	// Create and start server
	cfg := grpcserver.Config{
		Port:      *port,
		ModelPath: modelPath,
	}

	server, err := grpcserver.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating server: %v\n", err)
		os.Exit(1)
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		server.Stop()
		os.Exit(0)
	}()

	// Start server
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
