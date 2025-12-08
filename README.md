# diaz
make computer shit talk

A real-time speech-to-text application built in Go that captures microphone audio and transcribes it locally using Vosk. Designed to be self-contained with minimal dependencies.

## Current Status

**✅ WORKING** - Diaz is fully functional for real-time speech-to-text transcription!

### Implemented Features

- ✅ **Real-time Audio Capture** - Multi-platform audio input via malgo
- ✅ **Speech Recognition** - Offline transcription using Vosk
- ✅ **Model Management** - Download, cache, and switch between models
- ✅ **Interactive UI** - Real-time partial and final transcriptions
- ✅ **Multi-model Support** - Small (40MB), Medium (128MB), Large (1.8GB) models
- ✅ **Adaptive Buffering** - Automatic buffer sizing based on model complexity
- ✅ **Device Detection** - Auto-detect and list available microphones
- ✅ **CLI Tools** - Model selection, downloads, default configuration
- ✅ **Multiple Output Formats** - JSON, plain text, or interactive console output
- ✅ **Voice Activity Detection** - Energy-based VAD with configurable silence delay

## Quick Start

### Prerequisites

1. **Install Vosk Library** (one-time setup):
   ```bash
   make install-vosk
   ```

2. **Build the application**:
   ```bash
   make build
   ```

3. **Run and download a model** (on first run):
   ```bash
   ./build/diaz
   ```

### Basic Usage

```bash
# Start transcription with default model
./build/diaz

# Interactive model selection
./build/diaz --select-model

# Use a specific model
./build/diaz --model vosk-model-en-us-0.22-lgraph
```

## Available Models

| Model | Size | Speed | Accuracy | Use Case |
|-------|------|-------|----------|----------|
| `vosk-model-small-en-us-0.15` | 40MB | Fast | Good | Real-time, resource-constrained |
| `vosk-model-en-us-0.22-lgraph` | 128MB | Medium | Better | **Recommended** - Best balance |
| `vosk-model-en-us-0.22` | 1.8GB | Slow | Best | High accuracy, powerful hardware |

## CLI Commands

### Model Management
```bash
# List available models
./build/diaz --list-models

# List downloaded models
./build/diaz --list-downloaded

# Download a specific model
./build/diaz --download-model vosk-model-en-us-0.22-lgraph

# Set default model
./build/diaz --set-default vosk-model-en-us-0.22-lgraph
```

### Running Transcription
```bash
# Use default/configured model
./build/diaz

# Interactive model selection
./build/diaz --select-model

# Use specific model
./build/diaz --model vosk-model-small-en-us-0.15

# Auto-download if missing (no prompt)
./build/diaz --auto-download
```

### Output Formats
```bash
# Default console output (interactive)
./build/diaz

# JSON output to stdout
./build/diaz --format json

# JSON output to file
./build/diaz --format json --output transcription.json

# Plain text output to file
./build/diaz --format text --output transcription.txt
```

### Voice Activity Detection
```bash
# Enable VAD for automatic pause detection (enabled by default)
./build/diaz --vad

# Enable VAD with custom sensitivity (lower=more sensitive)
./build/diaz --vad --vad-threshold 0.005

# Set silence delay (seconds after speech before finalizing)
./build/diaz --vad --vad-silence-delay 10.0

# VAD with JSON output
./build/diaz --vad --format json --output transcription.json
```

### Utility
```bash
# Show version
./build/diaz --version

# Show help
./build/diaz --help
```

## Architecture

### Current Implementation

```
diaz/
├── cmd/diaz/
│   └── main.go                    # CLI interface, model selection
├── internal/
│   ├── audio/
│   │   ├── capture.go             # Audio config & interface
│   │   ├── malgo_capturer.go      # Malgo implementation
│   │   ├── device.go              # Device enumeration
│   │   └── buffer.go              # Ring buffer for streaming
│   ├── stt/
│   │   ├── engine.go              # STT engine interface
│   │   └── vosk_engine.go         # Vosk implementation
│   ├── models/
│   │   └── manager.go             # Model download/management
│   └── output/
│       └── console.go             # Console output formatting
├── models/                        # Downloaded models stored here
├── scripts/
│   └── install-vosk-lib.sh        # Vosk library installer
├── build/                         # Compiled binaries
├── Makefile                       # Build automation
└── INSTALL.md                     # Detailed installation guide
```

### Key Components

**Audio Capture** (`internal/audio`)
- Cross-platform audio via malgo (Linux/macOS/Windows)
- Adaptive buffer sizing (50-300 samples) based on model size
- Device enumeration and selection
- 16kHz mono capture optimized for STT

**Speech Recognition** (`internal/stt`)
- Vosk engine integration
- Real-time partial results
- Final results with confidence scores
- Thread-safe audio processing

**Model Management** (`internal/models`)
- HTTP download with progress tracking
- Model caching in `./models/` directory
- Default model configuration (persisted)
- Available models list from Vosk repository

**Console Output** (`internal/output`)
- Real-time partial transcription updates
- Final transcriptions with numbering
- Confidence scores
- Error reporting

## Development Roadmap

### ✅ Phase 1: Foundation (COMPLETED)
- [x] Project setup and structure
- [x] Audio capture with malgo
- [x] Console output
- [x] Makefile with build targets

### ✅ Phase 2: STT Integration (COMPLETED)
- [x] Vosk STT engine integration
- [x] Model management system
- [x] Real-time transcription pipeline
- [x] Adaptive buffering for model sizes

### 🚧 Phase 3: Enhancement (IN PROGRESS)
**Completed:**
- [x] Multiple output formats (JSON, plain text) with extensible interface
- [x] Voice Activity Detection (VAD) for better pause detection
- [x] VAD silence delay - configurable delay after speech before returning to silence mode
- [ ] Timestamp support in transcriptions

**Next Priority Items:**

- [ ] Configuration file support (~/.diazrc)
- [ ] Audio input device selection flag


### 📋 Phase 4: Advanced Features (PLANNED)
- [ ] Real-time streaming API (WebSocket/HTTP)
- [ ] Custom vocabulary/word lists
- [ ] Punctuation and capitalization improvements

### 🔧 Phase 5: Optimization (FUTURE)
- [ ] Static linking for all platforms
- [ ] Binary size optimization
- [ ] Performance profiling and tuning
- [ ] Memory usage optimization
- [ ] Cross-platform builds (macOS, Windows ARM)
- [ ] Docker container
- [ ] CI/CD pipeline

## Installation

See [INSTALL.md](INSTALL.md) for detailed installation instructions including:
- Installing the Vosk library
- Platform-specific setup
- Troubleshooting

Quick install:
```bash
# Install Vosk library (Linux x86_64)
make install-vosk

# Or check if already installed
make check-vosk

# Build the application
make build
```

## Build System

The Makefile provides comprehensive build targets:

```bash
# Build commands
make build              # Build for current platform
make build-all          # Build for all platforms
make quick              # Quick build without dep check

# Development
make dev                # Run with race detector
make test               # Run tests
make fmt                # Format code
make check              # Run all checks

# Vosk management
make install-vosk       # Install Vosk library
make check-vosk         # Verify Vosk installation

# Utility
make clean              # Remove build artifacts
make help               # Show all targets
```

## Technical Details

### Audio Processing
- **Sample Rate**: 16kHz (optimal for STT)
- **Channels**: Mono (1 channel)
- **Bit Depth**: 16-bit signed PCM
- **Buffer Size**: 30ms frames (480 samples @ 16kHz)
- **Sample Buffering**: 50-300 samples based on model size

### Model Buffer Configurations
- **Small models**: 50 samples (~1.5s buffer)
- **Medium models**: 150 samples (~4.5s buffer)
- **Large models**: 300 samples (~9s buffer)

### Dependencies
- **Runtime**:
  - Go 1.21+
  - Vosk library (libvosk.so)
  - Audio system (ALSA/PulseAudio/CoreAudio/WASAPI)

- **Build**:
  - CGO enabled
  - C compiler (gcc/clang)
  - Make

## Performance Notes

- **Small model**: ~10-20ms processing latency, minimal CPU
- **Medium model**: ~50-100ms processing latency, moderate CPU
- **Large model**: ~200-500ms processing latency, high CPU/memory

The application uses adaptive buffering to prevent sample drops with slower models.

## Troubleshooting

### "cannot find -lvosk" error
Run `make install-vosk` to install the Vosk library.

### "sample buffer overflow" errors
The application automatically selects buffer sizes based on model. If you still see errors:
- Use a smaller model
- Close other CPU-intensive applications
- Check system audio latency settings

### No audio devices found
- **Linux**: Ensure user is in the `audio` group
- **All platforms**: Verify microphone permissions

See [INSTALL.md](INSTALL.md) for detailed troubleshooting.

## Contributing

Contributions welcome! Priority areas:
- File output support
- Additional language models
- Voice Activity Detection
- Platform testing (macOS, Windows)

## License

MIT License (to be added)

## Acknowledgments

- [Vosk](https://alphacephei.com/vosk/) - Offline speech recognition toolkit
- [malgo](https://github.com/gen2brain/malgo) - Cross-platform audio I/O for Go
