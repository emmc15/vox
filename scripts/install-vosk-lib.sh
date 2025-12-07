#!/bin/bash
# Install Vosk API library for Linux

set -e

echo "Installing Vosk API library..."
echo ""

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
    echo "Warning: This script is designed for x86_64. Your architecture is $ARCH"
    echo "You may need to build Vosk from source."
    exit 1
fi

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

cd "$TMP_DIR"

# Download pre-built library from official releases
# Using the Python wheel which contains the compiled library
echo "Downloading Vosk library..."
wget https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-0.3.45-py3-none-linux_x86_64.whl \
    -O vosk.whl

echo "Extracting library..."
unzip -q vosk.whl

# The library is in vosk/ directory
if [ -f "vosk/libvosk.so" ]; then
    echo "Installing to /usr/local/lib and /usr/local/include..."

    # Copy library
    sudo cp vosk/libvosk.so /usr/local/lib/

    # Download header file from source
    echo "Downloading vosk_api.h..."
    sudo wget https://raw.githubusercontent.com/alphacep/vosk-api/master/src/vosk_api.h \
        -O /usr/local/include/vosk_api.h

    # Update library cache
    echo "Updating library cache..."
    sudo ldconfig

    echo ""
    echo "✓ Vosk API installed successfully!"
    echo "  Library: /usr/local/lib/libvosk.so"
    echo "  Header:  /usr/local/include/vosk_api.h"
    echo ""
else
    echo "Error: Could not find libvosk.so in the downloaded package"
    exit 1
fi
