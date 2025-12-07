# diaz
make computer shit talk

## Project Overview
A Go application that captures microphone audio and performs real-time speech-to-text transcription. The goal is to produce a single, self-contained binary for each platform with minimal external dependencies.

## Architecture Plan

### Core Components

#### 1. Audio Capture Layer (`internal/audio`)
- **Responsibility**: Interface with system microphone and capture audio streams
- **Implementation Options**:
  - **Primary**: `malgo` (pure Go, cross-platform audio I/O)
  - **Alternative**: PortAudio via CGO (more stable but requires C dependencies)
- **Features**:
  - Auto-detect default microphone
  - Configurable sample rate (16kHz recommended for STT)
  - Ring buffer for continuous capture
  - Audio level monitoring/VAD (Voice Activity Detection)

#### 2. Speech-to-Text Engine (`internal/stt`)
- **Responsibility**: Convert audio buffers to text transcriptions
- **Implementation Options**:
  - **Option A - Vosk** (offline, embeddable models)
    - Pros: Lightweight, multiple languages, works offline
    - Cons: CGO dependency, model files needed
  - **Option B - Whisper.cpp bindings** (state-of-the-art accuracy)
    - Pros: Excellent accuracy, offline
    - Cons: Larger model files, CGO dependency
  - **Option C - Cloud APIs** (Google/AWS/Azure)
    - Pros: No local models, better accuracy
    - Cons: Requires internet, API costs, latency
- **Recommended**: Vosk for embedded use case with bundled models

#### 3. Model Management (`internal/models`)
- **Responsibility**: Download, cache, and load STT models
- **Features**:
  - Embed small models in binary using `go:embed`
  - Download larger models on first run
  - Model versioning and updates
  - Automatic model selection based on language

#### 4. Output Handler (`internal/output`)
- **Responsibility**: Process and display transcription results
- **Features**:
  - Real-time console output
  - JSON output mode
  - File logging
  - WebSocket streaming (future)

#### 5. CLI Interface (`cmd/diaz`)
- **Responsibility**: User-facing command-line interface
- **Features**:
  - Start/stop recording
  - Model selection
  - Output format configuration
  - Language selection

### Project Structure
```
diaz/
├── cmd/
│   └── diaz/
│       └── main.go              # Application entry point
├── internal/
│   ├── audio/
│   │   ├── capture.go           # Audio capture interface
│   │   ├── device.go            # Device enumeration
│   │   └── buffer.go            # Audio buffering
│   ├── stt/
│   │   ├── engine.go            # STT engine interface
│   │   ├── vosk.go              # Vosk implementation
│   │   └── processor.go         # Audio preprocessing
│   ├── models/
│   │   ├── manager.go           # Model download/cache
│   │   └── embedded.go          # Embedded models
│   └── output/
│       ├── console.go           # Console output
│       ├── file.go              # File output
│       └── formatter.go         # Output formatting
├── models/                      # Embedded models (via go:embed)
│   └── vosk-model-small-en/    # Small English model
├── build/                       # Build artifacts
├── scripts/                     # Build scripts
├── Makefile                     # Build automation
├── go.mod
├── go.sum
└── README.md
```

## Build Strategy for Self-Contained Binaries

### Challenges
1. **CGO Dependencies**: Audio and STT libraries often require C
2. **Model Files**: STT models can be 40MB-1.5GB
3. **Platform-Specific**: Different audio APIs per OS

### Solutions
1. **Static Linking**: Use CGO with static linking flags
2. **Embedded Resources**: Use `go:embed` for small models
3. **Cross-Compilation**: Docker-based builds for each platform
4. **UPX Compression**: Compress final binaries (optional)

### Build Targets
- `linux-amd64`: Linux x86_64 (static, musl libc)
- `linux-arm64`: Linux ARM64 (Raspberry Pi, etc.)
- `darwin-amd64`: macOS Intel
- `darwin-arm64`: macOS Apple Silicon
- `windows-amd64`: Windows x86_64

## Development Roadmap

### Phase 1: Foundation
- [x] Project setup and structure
- [ ] Basic audio capture with malgo
- [ ] Simple console output
- [ ] Makefile with build targets

### Phase 2: STT Integration
- [ ] Integrate Vosk STT engine
- [ ] Model management system
- [ ] Real-time transcription pipeline

### Phase 3: Enhancement
- [ ] Voice Activity Detection
- [ ] Multiple language support
- [ ] Configuration file support
- [ ] Improved error handling

### Phase 4: Optimization
- [ ] Static linking for all platforms
- [ ] Binary size optimization
- [ ] Performance tuning
- [ ] CI/CD pipeline

## Usage

```bash
# Basic usage (uses default or configured model)
diaz

# Interactive model selection
diaz --select-model

# Use a specific model
diaz --model vosk-model-en-us-0.22-lgraph

# List all available models for download
diaz --list-models

# List downloaded models
diaz --list-downloaded

# Download a specific model
diaz --download-model vosk-model-en-us-0.22-lgraph

# Set default model
diaz --set-default vosk-model-en-us-0.22-lgraph

# Auto-download default model if missing (no prompt)
diaz --auto-download

# Show version
diaz --version
```

## Dependencies
- Go 1.21+
- CGO enabled (for audio and STT)
- Platform-specific:
  - Linux: ALSA/PulseAudio
  - macOS: CoreAudio
  - Windows: WASAPI

## Build Requirements
- Docker (for cross-compilation)
- Make
- Go toolchain
- C compiler (gcc/clang)

## Notes
- Initial focus: English language support
- Model size vs accuracy tradeoff
- Offline-first approach
- Future: Add real-time streaming API
