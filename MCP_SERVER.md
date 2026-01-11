# Vox MCP Server

Vox implements the Model Context Protocol (MCP) for speech-to-text transcription, allowing AI assistants and other clients to use Vox for voice transcription.

## Overview

The MCP server runs in stdio transport mode, communicating via JSON-RPC 2.0 messages over stdin/stdout. It provides tools for:

- **Audio transcription** with Voice Activity Detection (VAD)
- **Model management** and listing
- Automatic silence detection and finalization

## Quick Start

### Start the MCP Server

```bash
# Make sure you have a model downloaded first
./build/vox --download-model vosk-model-small-en-us-0.15

# Start the MCP server
./build/vox --mode mcp

# Or with a specific model
./build/vox --mode mcp --model vosk-model-en-us-0.22-lgraph
```

### Test with Example Client

```bash
# Make the test script executable
chmod +x examples/mcp_client_test.py

# Run the test client
python3 examples/mcp_client_test.py
```

## MCP Protocol

### Initialization

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "clientInfo": {
      "name": "my-client",
      "version": "1.0.0"
    }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {
        "listChanged": false
      }
    },
    "serverInfo": {
      "name": "vox-mcp",
      "version": "0.1.0"
    }
  }
}
```

### List Tools

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

Response includes available tools:
- `transcribe_audio` - Transcribe audio with VAD support
- `list_models` - List available Vosk models

### Transcribe Audio

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "transcribe_audio",
    "arguments": {
      "audio": "<base64-encoded-audio-data>",
      "vad_enabled": true,
      "vad_threshold": 0.01,
      "vad_silence_delay": 5.0
    }
  }
}
```

**Parameters:**
- `audio` (required): Base64-encoded audio data (16kHz, mono, 16-bit PCM)
- `model` (optional): Model name to use for transcription
- `vad_enabled` (optional): Enable Voice Activity Detection (default: true)
- `vad_threshold` (optional): VAD energy threshold 0.001-0.1 (default: 0.01)
- `vad_silence_delay` (optional): Seconds to wait after speech ends (default: 5.0)

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "hello world"
      },
      {
        "type": "text",
        "text": "Confidence: 0.95, Duration: 1.23s"
      }
    ],
    "isError": false
  }
}
```

### List Models

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "list_models"
  }
}
```

## Audio Format Requirements

The MCP server expects audio in the following format:

- **Sample Rate**: 16kHz (16000 Hz)
- **Channels**: Mono (1 channel)
- **Bit Depth**: 16-bit signed PCM
- **Encoding**: Base64-encoded raw PCM data

### Converting Audio Files

```bash
# Using ffmpeg to convert to required format
ffmpeg -i input.wav -ar 16000 -ac 1 -f s16le output.raw

# Convert to base64
base64 output.raw > output.base64
```

### Python Example

```python
import base64
import wave

def load_wav_as_base64(filename):
    with wave.open(filename, 'rb') as wav:
        # Ensure correct format
        assert wav.getframerate() == 16000, "Must be 16kHz"
        assert wav.getnchannels() == 1, "Must be mono"
        assert wav.getsampwidth() == 2, "Must be 16-bit"

        # Read and encode
        audio_data = wav.readframes(wav.getnframes())
        return base64.b64encode(audio_data).decode('utf-8')
```

## Voice Activity Detection (VAD)

The MCP server uses VAD to automatically detect when speech starts and ends:

1. **Audio starts** → VAD monitors energy levels
2. **Speech detected** → Transcription begins
3. **Speech ends** → Wait for `vad_silence_delay` seconds
4. **Silence confirmed** → Return transcription result

### VAD Parameters

- `vad_threshold`: Lower values are more sensitive (detect quieter speech)
  - Quiet environment: 0.005
  - Normal environment: 0.01 (default)
  - Noisy environment: 0.02-0.05

- `vad_silence_delay`: How long to wait before finalizing
  - Quick responses: 2.0 seconds
  - Normal speech: 5.0 seconds (default)
  - Long pauses: 10.0 seconds

## Integration Examples

### Claude Desktop Integration

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "vox-stt": {
      "command": "/path/to/vox",
      "args": ["--mode", "mcp", "--model", "vosk-model-small-en-us-0.15"]
    }
  }
}
```

### Custom Python Client

```python
import subprocess
import json
import base64

class VoxMCPClient:
    def __init__(self, vox_path='./build/vox', model=None):
        args = [vox_path, '--mode', 'mcp']
        if model:
            args.extend(['--model', model])

        self.proc = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        self.request_id = 0

    def send_request(self, method, params=None):
        self.request_id += 1
        request = {
            'jsonrpc': '2.0',
            'id': self.request_id,
            'method': method,
            'params': params or {}
        }

        msg = json.dumps(request) + '\n'
        self.proc.stdin.write(msg.encode())
        self.proc.stdin.flush()

        response = self.proc.stdout.readline()
        return json.loads(response)

    def transcribe(self, audio_data, vad_enabled=True, threshold=0.01, silence_delay=5.0):
        audio_b64 = base64.b64encode(audio_data).decode('utf-8')

        return self.send_request('tools/call', {
            'name': 'transcribe_audio',
            'arguments': {
                'audio': audio_b64,
                'vad_enabled': vad_enabled,
                'vad_threshold': threshold,
                'vad_silence_delay': silence_delay
            }
        })

    def close(self):
        self.proc.terminate()
        self.proc.wait()
```

## Error Handling

The server returns standard JSON-RPC error codes:

- `-32700`: Parse error (invalid JSON)
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32602`: Invalid parameters
- `-32603`: Internal error (transcription failed)

Example error response:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": -32602,
    "message": "invalid transcribe_audio params",
    "data": "missing required field: audio"
  }
}
```

## Performance Notes

- **Small model** (40MB): ~10-20ms latency, suitable for real-time
- **Medium model** (128MB): ~50-100ms latency, good balance
- **Large model** (1.8GB): ~200-500ms latency, highest accuracy

The MCP server processes audio synchronously (one request at a time) to ensure optimal STT engine performance.

## Troubleshooting

### "Model not found" error

Download the model first:
```bash
./build/vox --download-model vosk-model-small-en-us-0.15
```

### Connection timeout

Ensure the server is outputting to stderr (not stdout) for logs:
- Server logs go to stderr
- MCP protocol messages go to stdout

### No transcription returned

Check:
1. Audio format is correct (16kHz, mono, 16-bit PCM)
2. Audio contains actual speech
3. VAD threshold is appropriate for your audio
4. Sufficient silence delay for your use case

## See Also

- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Vox README](README.md)
- [Configuration Guide](config.yaml.example)
