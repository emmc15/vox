package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/emmett/vox/internal/app"
	"github.com/emmett/vox/internal/config"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

var (
	configFile      = flag.String("config", "", "Path to configuration file (default: ~/.voxrc or /etc/vox/config.yaml)")
	listModels      = flag.Bool("list-models", false, "List all available models for download")
	listDownloaded  = flag.Bool("list-downloaded", false, "List all downloaded models")
	downloadModel   = flag.String("download-model", "", "Download a specific model by name")
	modelName       = flag.String("model", "", "Use a specific model (default: vosk-model-small-en-us-0.15)")
	selectModel     = flag.Bool("select-model", false, "Interactively select a model to use")
	setDefault      = flag.String("set-default", "", "Set a model as the default")
	outputFormat    = flag.String("format", "console", "Output format: console, json, text")
	outputFile      = flag.String("output", "", "Output file (default: stdout)")
	enableVAD       = flag.Bool("vad", true, "Enable Voice Activity Detection for better pause handling")
	vadThreshold    = flag.Float64("vad-threshold", 0.01, "VAD energy threshold (0.001-0.1, lower=more sensitive)")
	vadSilenceDelay = flag.Float64("vad-silence-delay", 2.5, "Delay in seconds after last speech before returning to silence")
	audioDevice     = flag.String("device", "", "Audio input device name (use --list-devices to see available devices)")
	listDevices     = flag.Bool("list-devices", false, "List all available audio input devices")
	showVersion     = flag.Bool("version", false, "Show version information")
	autoDownload    = flag.Bool("auto-download", false, "Automatically download default model if not found (no prompt)")
)

func main() {
	flag.Parse()

	cfg, err := config.LoadWithFallback(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	applyConfigDefaults(cfg)

	if *showVersion {
		fmt.Printf("Vox CLI v%s\n", Version)
		fmt.Printf("  Commit:  %s\n", GitCommit)
		fmt.Printf("  Branch:  %s\n", GitBranch)
		fmt.Printf("  Built:   %s\n", BuildTime)
		os.Exit(0)
	}

	fmt.Printf("Vox CLI v%s (commit: %s, branch: %s, built: %s)\n",
		Version, GitCommit, GitBranch, BuildTime)
	fmt.Println("Speech-to-Text Application")
	fmt.Println()

	if *listDevices {
		dm := app.NewDeviceManager()
		if err := dm.ListDevices(); err != nil {
			os.Exit(1)
		}
		return
	}

	mgr := app.NewModelManager()

	if *listModels {
		if err := mgr.ListModels(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *listDownloaded {
		if err := mgr.ListDownloaded(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *downloadModel != "" {
		if err := mgr.Download(*downloadModel); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *setDefault != "" {
		if err := mgr.SetDefault(*setDefault); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func applyConfigDefaults(cfg *config.Config) {
	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

	if !flagsSet["model"] && cfg.Model.Default != "" {
		*modelName = cfg.Model.Default
	}
	if !flagsSet["format"] && cfg.Output.Format != "" {
		*outputFormat = cfg.Output.Format
	}
	if !flagsSet["output"] && cfg.Output.File != "" {
		*outputFile = cfg.Output.File
	}
	if !flagsSet["vad"] {
		*enableVAD = cfg.VAD.Enabled
	}
	if !flagsSet["vad-threshold"] && cfg.VAD.Threshold > 0 {
		*vadThreshold = cfg.VAD.Threshold
	}
	if !flagsSet["vad-silence-delay"] && cfg.VAD.SilenceDelay > 0 {
		*vadSilenceDelay = cfg.VAD.SilenceDelay
	}
	if !flagsSet["device"] && cfg.Audio.Device != "" {
		*audioDevice = cfg.Audio.Device
	}
}

func run() error {
	mgr := app.NewModelManager()
	selectedModel := *modelName
	if *selectModel {
		var err error
		selectedModel, err = mgr.SelectInteractive()
		if err != nil {
			return fmt.Errorf("failed to select model: %w", err)
		}
	}

	config := app.TranscriberConfig{
		ModelName:       selectedModel,
		OutputFormat:    *outputFormat,
		OutputFile:      *outputFile,
		EnableVAD:       *enableVAD,
		VADThreshold:    *vadThreshold,
		VADSilenceDelay: *vadSilenceDelay,
		AudioDevice:     *audioDevice,
		AutoDownload:    *autoDownload,
	}

	transcriber := app.NewTranscriber(config)
	return transcriber.Run()
}
