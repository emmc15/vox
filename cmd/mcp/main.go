package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/emmett/diaz/internal/app"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

var (
	modelName       = flag.String("model", "", "Use a specific model (default: vosk-model-small-en-us-0.15)")
	enableVAD       = flag.Bool("vad", true, "Enable Voice Activity Detection")
	vadThreshold    = flag.Float64("vad-threshold", 0.001, "VAD energy threshold (0.001-0.1, lower=more sensitive)")
	vadSilenceDelay = flag.Float64("vad-silence-delay", 5.0, "Delay in seconds after last speech before returning to silence")
	showVersion     = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Diaz MCP v%s\n", Version)
		fmt.Printf("  Commit:  %s\n", GitCommit)
		fmt.Printf("  Branch:  %s\n", GitBranch)
		fmt.Printf("  Built:   %s\n", BuildTime)
		os.Exit(0)
	}

	handler := app.NewMCPHandler(*modelName, Version, GitCommit, *vadThreshold, *vadSilenceDelay, *enableVAD)
	if err := handler.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
