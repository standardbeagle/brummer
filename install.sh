#!/bin/bash

# Brummer Installation Script
# Installs Brummer TUI package script manager

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Installation configuration
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="brum"
REPO_URL="https://github.com/standardbeagle/brummer"

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

# Header
echo -e "${YELLOW}"
echo "🐝 Brummer Installation Script"
echo "================================"
echo -e "${NC}"

# Check if running as root for system-wide install
if [[ $EUID -eq 0 ]]; then
   print_info "Running as root - will install system-wide to $INSTALL_DIR"
else
   print_warning "Not running as root - will install to user directory"
   INSTALL_DIR="$HOME/.local/bin"
   mkdir -p "$INSTALL_DIR"
fi

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        echo "Visit https://golang.org/dl/ for installation instructions."
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Found Go version: $GO_VERSION"
}

# Check if already installed
check_existing() {
    if command -v $BINARY_NAME &> /dev/null; then
        CURRENT_VERSION=$($BINARY_NAME --version 2>/dev/null || echo "unknown")
        print_warning "Brummer is already installed (version: $CURRENT_VERSION)"
        read -p "Do you want to reinstall/update? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Installation cancelled"
            exit 0
        fi
    fi
}

# Install from source
install_from_source() {
    print_info "Installing from source..."
    
    # Build the binary
    print_info "Building Brummer..."
    go build -o "$BINARY_NAME" cmd/brum/main.go
    
    if [[ ! -f "$BINARY_NAME" ]]; then
        print_error "Build failed - binary not created"
        exit 1
    fi
    
    # Make it executable
    chmod +x "$BINARY_NAME"
    
    # Move to install directory
    print_info "Installing to $INSTALL_DIR/$BINARY_NAME..."
    if [[ $EUID -eq 0 ]] || [[ "$INSTALL_DIR" == "$HOME/.local/bin" ]]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        print_warning "Need sudo access to install to $INSTALL_DIR"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    fi
}

# Install from GitHub releases (future feature)
install_from_release() {
    print_info "Checking for latest release..."
    
    # Detect OS and architecture
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64)
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
    
    print_warning "Binary releases not yet available. Falling back to source installation."
    install_from_source
}

# Setup shell completion (optional)
setup_completion() {
    print_info "Setting up shell completion..."
    
    # Detect shell
    SHELL_NAME=$(basename "$SHELL")
    
    case $SHELL_NAME in
        bash)
            COMPLETION_DIR="$HOME/.bash_completion.d"
            mkdir -p "$COMPLETION_DIR"
            cat > "$COMPLETION_DIR/brum" << 'EOF'
# Brummer bash completion
_brum() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="--help --version --port --project-dir --no-mcp"

    case "${prev}" in
        --port)
            COMPREPLY=( $(compgen -W "7777 8080 3000" -- ${cur}) )
            return 0
            ;;
        --project-dir)
            COMPREPLY=( $(compgen -d -- ${cur}) )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}
complete -F _brum brum
EOF
            print_success "Bash completion installed to $COMPLETION_DIR/brum"
            print_info "Run 'source $COMPLETION_DIR/brum' or restart your shell"
            ;;
        zsh)
            print_info "Zsh completion setup skipped (add manually if needed)"
            ;;
        fish)
            print_info "Fish completion setup skipped (add manually if needed)"
            ;;
        *)
            print_info "Unknown shell: $SHELL_NAME - skipping completion setup"
            ;;
    esac
}

# Verify installation
verify_installation() {
    print_info "Verifying installation..."
    
    # Check if binary is in PATH
    if ! command -v $BINARY_NAME &> /dev/null; then
        print_warning "Brummer installed but not in PATH"
        print_info "Add $INSTALL_DIR to your PATH:"
        echo "export PATH=\"$INSTALL_DIR:\$PATH\""
        echo "Then run: source ~/.bashrc (or restart your terminal)"
    else
        INSTALLED_PATH=$(which $BINARY_NAME)
        print_success "Brummer installed successfully at: $INSTALLED_PATH"
        
        # Show version
        if $BINARY_NAME --version &> /dev/null; then
            VERSION=$($BINARY_NAME --version)
            print_info "Version: $VERSION"
        fi
    fi
}

# Main installation flow
main() {
    # Check prerequisites
    check_go
    check_existing
    
    # Determine installation method
    if [[ -f "go.mod" ]] && [[ -f "cmd/brummer/main.go" ]]; then
        print_info "Found source files in current directory"
        install_from_source
    else
        print_warning "Source files not found in current directory"
        read -p "Clone from GitHub and install? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            TEMP_DIR=$(mktemp -d)
            print_info "Cloning repository to $TEMP_DIR..."
            git clone "$REPO_URL" "$TEMP_DIR/brummer"
            cd "$TEMP_DIR/brummer"
            install_from_source
            cd - > /dev/null
            rm -rf "$TEMP_DIR"
        else
            print_error "Cannot proceed without source files"
            exit 1
        fi
    fi
    
    # Optional: Setup completion
    read -p "Setup shell completion? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        setup_completion
    fi
    
    # Verify installation
    verify_installation
    
    # Show next steps
    echo
    echo -e "${GREEN}Installation complete!${NC}"
    echo
    echo "Next steps:"
    echo "1. Run 'brum' in a project directory with package.json"
    echo "2. Press '?' in the TUI for help"
    echo "3. Install the browser extension for enhanced debugging"
    echo
    echo "For more information: https://github.com/standardbeagle/brummer"
}

# Run main installation
main

# Optionally start Brummer
echo
read -p "Start Brummer now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Starting Brummer..."
    exec $BINARY_NAME
fi