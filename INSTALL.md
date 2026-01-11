# Installation Guide

## Prerequisites

### System Requirements
- Linux (x86_64) / macOS / Windows
- Go 1.21 or higher
- C compiler (gcc/clang)
- Make

### Audio System
- **Linux**: ALSA or PulseAudio
- **macOS**: CoreAudio (built-in)
- **Windows**: WASAPI (built-in)

## Installing Vosk API Library

Vox uses the Vosk speech recognition engine, which requires the native Vosk library to be installed.

### Option 1: Automated Installation (Linux x86_64)

Run the provided installation script:

```bash
chmod +x scripts/install-vosk-lib.sh
./scripts/install-vosk-lib.sh
```

This will:
- Download the Vosk library
- Install it to `/usr/local/lib/libvosk.so`
- Install the header to `/usr/local/include/vosk_api.h`
- Update the library cache

### Option 2: Manual Installation

#### Linux

1. Download Vosk library from releases:
   ```bash
   wget https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-0.3.45-py3-none-linux_x86_64.whl
   ```

2. Extract the wheel file:
   ```bash
   unzip vosk-0.3.45-py3-none-linux_x86_64.whl
   ```

3. Install the library and header:
   ```bash
   sudo cp vosk/libvosk.so /usr/local/lib/
   sudo wget https://raw.githubusercontent.com/alphacep/vosk-api/master/src/vosk_api.h \
       -O /usr/local/include/vosk_api.h
   sudo ldconfig
   ```

#### macOS

1. Install using Homebrew (if available):
   ```bash
   # Vosk is not in Homebrew yet, use manual method
   ```

2. Or download and install manually:
   ```bash
   # Download macOS library from releases
   wget https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-0.3.45-py3-none-macosx_10_9_x86_64.whl
   unzip vosk-0.3.45-py3-none-macosx_10_9_x86_64.whl
   sudo cp vosk/libvosk.dylib /usr/local/lib/
   sudo wget https://raw.githubusercontent.com/alphacep/vosk-api/master/src/vosk_api.h \
       -O /usr/local/include/vosk_api.h
   ```

#### Windows

1. Download the Windows library:
   ```powershell
   # Download from releases
   wget https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-0.3.45-py3-none-win_amd64.whl
   ```

2. Extract and copy:
   ```powershell
   # Extract vosk.dll and vosk_api.h
   # Copy to appropriate locations in your PATH
   ```

### Option 3: Build from Source

If pre-built binaries aren't available for your platform:

```bash
git clone https://github.com/alphacep/vosk-api
cd vosk-api/src
make
sudo make install
```

## Verifying Installation

Check that the library is installed:

```bash
# Linux/macOS
ldconfig -p | grep vosk
ls -l /usr/local/lib/libvosk.so*
ls -l /usr/local/include/vosk_api.h

# Or
pkg-config --libs --cflags vosk
```

## Building Vox

Once Vosk is installed:

```bash
# Build for your platform
make build

# Or build for all platforms
make build-all
```

## Downloading Speech Models

On first run, Vox will prompt you to download a speech recognition model:

```bash
./build/vox
```

The default model is `vosk-model-small-en-us-0.15` (40MB), which provides:
- Fast performance
- Good accuracy for clear speech
- Low memory usage

Other available models:
- `vosk-model-en-us-0.22-lgraph` (128MB) - Balanced
- `vosk-model-en-us-0.22` (1.8GB) - High accuracy

## Troubleshooting

### Library not found error

If you get `cannot find -lvosk`:

1. Verify the library is installed:
   ```bash
   ls /usr/local/lib/libvosk.so*
   ```

2. Update library cache:
   ```bash
   sudo ldconfig
   ```

3. Add to library path:
   ```bash
   export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
   ```

### Header not found error

If you get `vosk_api.h: No such file or directory`:

1. Verify header exists:
   ```bash
   ls /usr/local/include/vosk_api.h
   ```

2. Set CGO flags:
   ```bash
   export CGO_CFLAGS="-I/usr/local/include"
   export CGO_LDFLAGS="-L/usr/local/lib -lvosk"
   ```

### Audio device errors

**Linux**: Ensure ALSA or PulseAudio is running
```bash
aplay -l  # List audio devices
```

**Permissions**: Add your user to the audio group
```bash
sudo usermod -a -G audio $USER
```

## Next Steps

After installation:

1. Run Vox: `./build/vox`
2. Download a model when prompted
3. Start speaking into your microphone
4. See real-time transcriptions!

For more information, see README.md
