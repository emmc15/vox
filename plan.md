# Refactoring Plan: cmd/diaz/main.go → internal/app

## Goal
Reduce main.go from 971 lines to ~150 lines by extracting business logic into reusable abstractions.

## Current Status
- **main.go**: 626 lines (reduced from 971)
- **Completed**: Step 1 - ModelManager, Step 2 - DeviceManager

## Steps

### ✅ Step 1: ModelManager (DONE)
**File**: `internal/app/model_manager.go`

Extracted methods:
- `ListModels()` - list available models
- `ListDownloaded()` - list downloaded models
- `Download(name)` - download specific model
- `SetDefault(name)` - set default model
- `SelectInteractive()` - interactive model selection
- `EnsureModel(name, autoDownload)` - ensure model is downloaded
- `SelectModel(modelName, selectInteractive)` - determine which model to use

**Impact**: Removed ~258 lines from main.go

---

### ✅ Step 2: DeviceManager (DONE)
**File**: `internal/app/device_manager.go`

Extracted methods:
- `ListDevices()` - list audio devices (from `handleListDevices()`)
- `SelectDevice(deviceName)` - select device by name or use default
- Device validation logic

**Impact**: Removed ~87 lines from main.go

---

### ✅ Step 3: Transcriber
**File**: `internal/app/transcriber.go`

Extract main transcription loop logic:
- `Run(config)` - main transcription orchestration
- VAD processing logic
- STT engine integration
- Output formatting
- Signal handling

**Estimated impact**: Remove ~300 lines from main.go

---

### ⏭️ Step 4: MCP Server Handler (optional)
**File**: `internal/app/mcp_handler.go`

Extract:
- `runMCPServer()` function
- MCP configuration logic

**Estimated impact**: Remove ~100 lines from main.go

---

### ⏭️ Step 5: Audio Config Helper (optional)
**File**: `internal/app/audio_config.go`

Extract:
- `getAudioConfigForModel(modelName)` helper

**Estimated impact**: Remove ~20 lines from main.go

---

## Final Target Structure

```
cmd/diaz/main.go (~150 lines)
├── Flag definitions
├── Config loading
├── Command routing
└── Version info

internal/app/
├── model_manager.go (✅ done)
├── device_manager.go
├── transcriber.go
├── mcp_handler.go
└── audio_config.go
```

## Next Action
Start with **Step 3: Transcriber**
