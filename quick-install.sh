#!/usr/bin/env bash

# Brummer Quick Installation Script
# Downloads and installs the latest Brummer binary for your platform

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="standardbeagle/brummer"
BINARY_NAME="brum"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|cygwin*|msys*)
            OS="windows"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    
    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    print_info "Detected platform: $PLATFORM"
}

# Get the download URL for the latest release
get_download_url() {
    if [ "$VERSION" = "latest" ]; then
        # Get latest release tag
        print_info "Fetching latest release information..."
        RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
        
        # Check if we have curl or wget
        if command -v curl >/dev/null 2>&1; then
            RELEASE_JSON=$(curl -s "$RELEASE_URL")
        elif command -v wget >/dev/null 2>&1; then
            RELEASE_JSON=$(wget -qO- "$RELEASE_URL")
        else
            print_error "Neither curl nor wget found. Please install one of them."
            exit 1
        fi
        
        # Extract version tag
        VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name": *"[^"]*"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
        
        if [ -z "$VERSION" ]; then
            print_error "Failed to fetch latest release version"
            exit 1
        fi
    fi
    
    print_info "Installing version: $VERSION"
    
    # Construct download URL
    BINARY_SUFFIX=""
    if [ "$OS" = "windows" ]; then
        BINARY_SUFFIX=".exe"
    fi
    
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}${BINARY_SUFFIX}"
    print_info "Download URL: $DOWNLOAD_URL"
}

# Download the binary
download_binary() {
    print_info "Downloading Brummer..."
    
    TEMP_DIR=$(mktemp -d)
    TEMP_BINARY="${TEMP_DIR}/${BINARY_NAME}${BINARY_SUFFIX}"
    
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$TEMP_BINARY" "$DOWNLOAD_URL" || {
            print_error "Failed to download binary"
            rm -rf "$TEMP_DIR"
            exit 1
        }
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$TEMP_BINARY" "$DOWNLOAD_URL" || {
            print_error "Failed to download binary"
            rm -rf "$TEMP_DIR"
            exit 1
        }
    fi
    
    print_success "Download complete"
}

# Install the binary
install_binary() {
    print_info "Installing to $INSTALL_DIR..."
    
    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Make binary executable
    chmod +x "$TEMP_BINARY"
    
    # Move to install directory
    mv "$TEMP_BINARY" "${INSTALL_DIR}/${BINARY_NAME}${BINARY_SUFFIX}"
    
    # Clean up
    rm -rf "$TEMP_DIR"
    
    print_success "Installation complete!"
}

# Verify installation
verify_installation() {
    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_warning "$INSTALL_DIR is not in your PATH"
        print_info "Add it to your PATH by adding this line to your shell profile:"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\""
        echo
    fi
    
    # Try to run the binary
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        print_success "Brummer is ready to use!"
        print_info "Run 'brum' to get started"
    else
        print_info "Run '${INSTALL_DIR}/${BINARY_NAME}' to get started"
    fi
}

# Main installation flow
main() {
    echo -e "${YELLOW}"
    echo "🐝 Brummer Quick Installer"
    echo "=========================="
    echo -e "${NC}"
    
    # Check if already installed
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        CURRENT_VERSION=$("$BINARY_NAME" --version 2>/dev/null || echo "unknown")
        print_warning "Brummer is already installed (version: $CURRENT_VERSION)"
        read -p "Do you want to reinstall/update? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Installation cancelled"
            exit 0
        fi
    fi
    
    # Detect platform
    detect_platform
    
    # Get download URL
    get_download_url
    
    # Download binary
    download_binary
    
    # Install binary
    install_binary
    
    # Verify installation
    verify_installation
    
    echo
    print_success "Installation complete! 🎉"
    echo
    echo "Next steps:"
    echo "1. Run 'brum' in a project directory with package.json"
    echo "2. Press '?' in the TUI for help"
    echo "3. Visit https://github.com/${REPO} for documentation"
}

# Handle errors
trap 'print_error "Installation failed"; exit 1' ERR

# Run main installation
main