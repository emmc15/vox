#!/usr/bin/env python3
"""
MCP Client Test Script for Vox STT Server

This script demonstrates how to interact with the Vox MCP server
for speech-to-text transcription via microphone capture.
"""

import json
import subprocess
import sys

def send_request(proc, request):
    """Send a JSON-RPC request to the MCP server"""
    request_json = json.dumps(request) + '\n'
    proc.stdin.write(request_json.encode('utf-8'))
    proc.stdin.flush()

def read_response(proc):
    """Read a JSON-RPC response from the MCP server"""
    while True:
        line = proc.stdout.readline()
        if not line:
            return None
        try:
            return json.loads(line.decode('utf-8'))
        except json.JSONDecodeError:
            # Skip non-JSON lines (server logs)
            print(f"[Server log] {line.decode('utf-8').strip()}", file=sys.stderr)
            continue

def main():
    # Start the MCP server
    print("Starting Vox MCP server...")
    proc = subprocess.Popen(
        ['./build/vox', '--mode', 'mcp'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )

    try:
        # 1. Initialize
        print("\n1. Initializing MCP connection...")
        init_request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {}
                },
                "clientInfo": {
                    "name": "vox-test-client",
                    "version": "1.0.0"
                }
            }
        }
        send_request(proc, init_request)
        response = read_response(proc)
        print(f"Initialize response: {json.dumps(response, indent=2)}")

        # 2. List tools
        print("\n2. Listing available tools...")
        list_tools_request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list"
        }
        send_request(proc, list_tools_request)
        response = read_response(proc)
        print(f"Available tools: {json.dumps(response, indent=2)}")

        # 3. List models
        print("\n3. Listing available models...")
        list_models_request = {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "list_models"
            }
        }
        send_request(proc, list_models_request)
        response = read_response(proc)
        print(f"Models: {json.dumps(response, indent=2)}")

        # 4. Transcribe audio (captures from microphone)
        print("\n4. Transcribing audio from microphone (speak now)...")
        transcribe_request = {
            "jsonrpc": "2.0",
            "id": 4,
            "method": "tools/call",
            "params": {
                "name": "transcribe_audio",
                "arguments": {}
            }
        }
        send_request(proc, transcribe_request)
        response = read_response(proc)
        print(f"Transcription result: {json.dumps(response, indent=2)}")

        # 5. Ping test
        print("\n5. Testing ping...")
        ping_request = {
            "jsonrpc": "2.0",
            "id": 5,
            "method": "ping"
        }
        send_request(proc, ping_request)
        response = read_response(proc)
        print(f"Ping response: {json.dumps(response, indent=2)}")

        print("\n✓ All tests completed successfully!")

    except Exception as e:
        print(f"\n✗ Error: {e}", file=sys.stderr)
        return 1
    finally:
        # Clean shutdown
        print("\nShutting down server...")
        proc.terminate()
        proc.wait(timeout=5)

    return 0

if __name__ == '__main__':
    sys.exit(main())
