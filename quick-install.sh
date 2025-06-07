#!/bin/bash
# Quick install script for Brummer
# Usage: curl -sSL https://raw.githubusercontent.com/beagle/brummer/main/quick-install.sh | bash

set -e

echo "🐝 Installing Brummer..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    echo "Visit https://golang.org/dl/ for installation instructions."
    exit 1
fi

# Clone and build
TEMP_DIR=$(mktemp -d)
git clone https://github.com/beagle/brummer "$TEMP_DIR/brummer" || {
    echo "Error: Failed to clone repository"
    exit 1
}

cd "$TEMP_DIR/brummer"
go build -o brum cmd/brummer/main.go

# Install
if [[ -w "/usr/local/bin" ]]; then
    mv brum /usr/local/bin/
else
    mkdir -p "$HOME/.local/bin"
    mv brum "$HOME/.local/bin/"
    echo "Installed to $HOME/.local/bin/brum"
    echo "Add $HOME/.local/bin to your PATH if not already added"
fi

# Cleanup
cd - > /dev/null
rm -rf "$TEMP_DIR"

echo "✅ Brummer installed successfully!"
echo "Run 'brum' to start"