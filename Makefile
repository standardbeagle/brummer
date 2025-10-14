# Brummer Makefile

# Variables
BINARY_NAME=brum
INSTALL_DIR=/usr/local/bin
USER_INSTALL_DIR=$(HOME)/.local/bin
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Load .env file if it exists
ifneq ($(wildcard .env),)
    include .env
    export
endif

# Detect if running in WSL
ifneq ($(wildcard /proc/sys/fs/binfmt_misc/WSLInterop),)
    IS_WSL := true
    # Use WINDOWS_USER from .env if set, otherwise try to detect
    ifdef WINDOWS_USER
        WIN_USER := $(WINDOWS_USER)
    else
        WIN_USER := $(shell wslpath -w ~ 2>/dev/null | sed 's/.*\\\([^\\]*\)$$/\1/' || echo "")
    endif
    ifneq ($(WIN_USER),)
        WIN_INSTALL_DIR := /mnt/c/Users/$(WIN_USER)/.local/bin
    endif
endif

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    DETECTED_OS := linux
endif
ifeq ($(UNAME_S),Darwin)
    DETECTED_OS := darwin
endif
ifeq ($(OS),Windows_NT)
    DETECTED_OS := windows
    BINARY_NAME := $(BINARY_NAME).exe
endif

# Detect architecture
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
    DETECTED_ARCH := amd64
endif
ifeq ($(UNAME_M),aarch64)
    DETECTED_ARCH := arm64
endif
ifeq ($(UNAME_M),arm64)
    DETECTED_ARCH := arm64
endif
# Default to amd64 if unable to detect
DETECTED_ARCH ?= amd64

# Default target
.DEFAULT_GOAL := build

# Build the binary
.PHONY: build
build:
	@echo "🔨 Building Brummer..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/brum
	@echo "✅ Build complete: ./$(BINARY_NAME)"

# Run unit tests (excluding integration tests and problematic packages)
.PHONY: test
test:
	@echo "🧪 Running unit tests..."
	@go test -timeout 60s \
		./internal/config \
		./internal/logs \
		./internal/parser \
		./internal/discovery \
		./pkg/events \
		./pkg/filters \
		./pkg/ports

# Run unit tests with verbose output
.PHONY: test-verbose
test-verbose:
	@echo "🧪 Running unit tests (verbose)..."
	@go test -timeout 60s -v \
		./internal/config \
		./internal/logs \
		./internal/parser \
		./internal/discovery \
		./pkg/events \
		./pkg/filters \
		./pkg/ports

# Run fast unit tests (working packages only)
.PHONY: test-fast
test-fast:
	@echo "🧪 Running fast unit tests..."
	@go test \
		./internal/config \
		./internal/discovery \
		./pkg/events

# Run all unit tests including slower ones
.PHONY: test-unit-all
test-unit-all:
	@echo "🧪 Running all unit tests..."
	@go test -timeout 2m \
		./internal/config \
		./internal/logs \
		./internal/parser \
		./internal/process \
		./internal/proxy \
		./internal/tui \
		./internal/discovery \
		./pkg/...

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "🧪 Running tests with race detection..."
	@CGO_ENABLED=1 go test -race -timeout 10m \
		./internal/config \
		./internal/logs \
		./internal/discovery \
		./internal/tui \
		./internal/process \
		./internal/proxy \
		./pkg/events

# Run comprehensive race detection across all packages
.PHONY: test-race-all
test-race-all:
	@echo "🧪 Running comprehensive race detection..."
	@CGO_ENABLED=1 go test -race -timeout=10m ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test -cover -coverprofile=coverage.out \
		./internal/config \
		./internal/logs \
		./internal/discovery \
		./pkg/events
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report saved to coverage.html"

# Run only MCP tests (separate due to complexity)
.PHONY: test-mcp-unit
test-mcp-unit:
	@echo "🧪 Running MCP unit tests..."
	@go test -timeout 2m ./internal/mcp

# Run integration tests (including hub tests)
.PHONY: test-integration-unit
test-integration-unit:
	@echo "🧪 Running integration tests..."
	@go test -timeout 5m ./test

# Run regression test suite
.PHONY: test-regression
test-regression: build
	@echo "🧪 Running regression test suite..."
	@./test/run_tests.sh

# Run regression tests with verbose output
.PHONY: test-regression-verbose
test-regression-verbose: build
	@echo "🧪 Running regression tests (verbose)..."
	@./test/run_tests.sh --verbose

# Run regression tests without building (use existing binary)
.PHONY: test-regression-quick
test-regression-quick:
	@echo "🧪 Running regression tests (quick)..."
	@./test/run_tests.sh --skip-build

# Run only MCP regression tests
.PHONY: test-mcp
test-mcp: build
	@echo "🧪 Running MCP regression tests..."
	@./test/run_tests.sh --filter MCP

# Run only Proxy regression tests
.PHONY: test-proxy
test-proxy: build
	@echo "🧪 Running Proxy regression tests..."
	@./test/run_tests.sh --filter Proxy

# Run only Logging regression tests
.PHONY: test-logging
test-logging: build
	@echo "🧪 Running Logging regression tests..."
	@./test/run_tests.sh --filter Logging

# Run only Process regression tests
.PHONY: test-processes
test-processes: build
	@echo "🧪 Running Process regression tests..."
	@./test/run_tests.sh --filter Processes

# Run only Integration regression tests
.PHONY: test-integration
test-integration: build
	@echo "🧪 Running Integration regression tests..."
	@./test/run_tests.sh --filter Integration

# Run Expect-based TUI tests
.PHONY: test-expect
test-expect: build
	@echo "🎭 Running Expect-based TUI tests..."
	@if ! command -v expect >/dev/null 2>&1; then \
		echo "❌ expect not found. Install with:"; \
		echo "   sudo apt-get install expect  # Ubuntu/Debian"; \
		echo "   brew install expect           # macOS"; \
		exit 1; \
	fi
	@export BRUMMER_BINARY=./$(BINARY_NAME)
	@./test/tty/expect_basic_tui.exp
	@./test/tty/expect_navigation.exp
	@./test/tty/expect_commands.exp
	@./test/tty/expect_ai_coders.exp
	@echo "✅ Expect tests completed"

# Run Tmux-based TUI tests (alternative to expect)
.PHONY: test-tmux
test-tmux: build
	@echo "🖥️  Running Tmux-based TUI tests..."
	@if ! command -v tmux >/dev/null 2>&1; then \
		echo "❌ tmux not found. Install with:"; \
		echo "   sudo apt-get install tmux  # Ubuntu/Debian"; \
		echo "   brew install tmux           # macOS"; \
		exit 1; \
	fi
	@export BRUMMER_BINARY=./$(BINARY_NAME)
	@./test/tty/tmux_tty_test.sh
	@echo "✅ Tmux tests completed"

# Run System Bash TTY tests (basic validation)
.PHONY: test-tty
test-tty: build
	@echo "💻 Running System Bash TTY tests..."
	@export BRUMMER_BINARY=./$(BINARY_NAME)
	@./test/tty/system_bash_test.sh
	@echo "✅ System bash tests completed"

# Run TUI integration tests with new harness
.PHONY: test-tui
test-tui: build
	@echo "🖥️  Running TUI integration tests..."
	@./test/scripts/run_tui_tests.sh

# Run TUI integration tests (comprehensive)
.PHONY: test-tui-integration
test-tui-integration: build
	@echo "🖥️  Running comprehensive TUI tests..."
	@BRUMMER_BINARY=./$(BINARY_NAME) go test -tags=integration ./test/tui -v

# Validate TUI functionality
.PHONY: validate-tui
validate-tui: build
	@echo "✅ Validating TUI functionality..."
	@./test/scripts/validate_tui.sh

# Setup TUI test environment
.PHONY: setup-tui-test-env
setup-tui-test-env:
	@echo "🔧 Setting up TUI test environment..."
	@./test/scripts/setup_test_env.sh

# Run all TUI tests (expect + alternatives + new harness)
.PHONY: test-tui-all
test-tui-all: build
	@echo "🖥️  Running all TUI tests..."
	@make setup-tui-test-env
	@make test-tui || echo "⚠️  TUI harness tests failed, trying alternatives..."
	@make test-expect || echo "⚠️  Expect tests failed, trying alternatives..."
	@make test-tmux || echo "⚠️  Tmux tests failed, trying system bash..."
	@make test-tty || echo "⚠️  All TUI tests failed - check environment"
	@echo "✅ TUI testing completed"

# Run all tests (unit + regression + TUI)
.PHONY: test-all
test-all: test test-regression test-tui-all
	@echo "✅ All tests completed"

# Install system-wide (requires sudo)
.PHONY: install
install: build
	@echo "📦 Installing to $(INSTALL_DIR)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod 755 $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Installed to $(INSTALL_DIR)/$(BINARY_NAME)"

# Install for current user
.PHONY: install-user
install-user: build-all
	@echo "📦 Installing to $(USER_INSTALL_DIR)..."
	@mkdir -p $(USER_INSTALL_DIR)
	@cp dist/$(BINARY_NAME)-$(DETECTED_OS)-$(DETECTED_ARCH) $(USER_INSTALL_DIR)/$(BINARY_NAME)
ifeq ($(DETECTED_OS),windows)
	@echo "✅ Installed to $(USER_INSTALL_DIR)/$(BINARY_NAME)"
else
	@chmod 755 $(USER_INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Installed to $(USER_INSTALL_DIR)/$(BINARY_NAME)"
endif
ifeq ($(IS_WSL),true)
    ifdef WIN_INSTALL_DIR
	@echo "📦 Installing Windows binary to: $(WIN_INSTALL_DIR)..."
	@mkdir -p $(WIN_INSTALL_DIR)
	@if [ -f "dist/$(BINARY_NAME)-windows-$(DETECTED_ARCH).exe" ]; then \
		cp dist/$(BINARY_NAME)-windows-$(DETECTED_ARCH).exe $(WIN_INSTALL_DIR)/$(BINARY_NAME).exe; \
		echo "✅ Installed to $(WIN_INSTALL_DIR)/$(BINARY_NAME).exe"; \
	else \
		echo "⚠️  Windows binary not found: dist/$(BINARY_NAME)-windows-$(DETECTED_ARCH).exe"; \
		echo "💡 Run 'make build-all' first to create Windows binaries"; \
	fi
	@echo "💡 Make sure both $(USER_INSTALL_DIR) and $(WIN_INSTALL_DIR) are in your PATH"
    else
	@echo "💡 Make sure $(USER_INSTALL_DIR) is in your PATH"
	@echo "⚠️  Could not detect Windows user directory for dual installation"
	@echo "💡 Set WINDOWS_USER=YOUR_USERNAME in .env file to enable Windows installation"
    endif
else
	@echo "💡 Make sure $(USER_INSTALL_DIR) is in your PATH"
endif


# Uninstall
.PHONY: uninstall
uninstall:
	@echo "🗑️  Uninstalling Brummer..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@rm -f $(USER_INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Uninstalled"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf test/test_workspace
	@rm -rf dist
	@go clean
	@echo "✅ Clean complete"

# Run the application
.PHONY: run
run: build
	@echo "🐝 Starting Brummer..."
	@./$(BINARY_NAME)

# Development mode with hot reload
.PHONY: dev
dev:
	@echo "🔄 Starting in development mode..."
	@command -v air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air

# Format code
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Formatting complete"

# Lint code
.PHONY: lint
lint:
	@echo "🔍 Linting code..."
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run
	@echo "✅ Linting complete"

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "🏗️  Building for multiple platforms..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/brum
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/brum
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/brum
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/brum
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/brum
	@GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-arm64.exe ./cmd/brum
	@echo "✅ Multi-platform build complete. Binaries in ./dist/"

# Package browser extension
.PHONY: pack-extension
pack-extension:
	@echo "📦 Packaging browser extension..."
	@cd browser-extension && bash build.sh
	@echo "✅ Extension packaged"

# Show help
.PHONY: help
help:
	@echo "🐝 Brummer Makefile Commands:"
	@echo ""
	@echo "  make build          - Build the binary"
	@echo "  make install        - Install system-wide (requires sudo)"
	@echo "  make install-user   - Install for current user"
	@echo "  make uninstall      - Remove installed binary"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run            - Build and run"
	@echo "  make dev            - Run in development mode with hot reload"
	@echo "  make test           - Run unit tests (reliable packages)"
	@echo "  make test-verbose   - Run unit tests with verbose output"
	@echo "  make test-fast      - Run fast unit tests only"
	@echo "  make test-unit-all  - Run all unit tests (including slower ones)"
	@echo "  make test-race      - Run tests with race detection"
	@echo "  make test-race-all  - Run comprehensive race detection"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-mcp-unit  - Run MCP unit tests"
	@echo "  make test-integration-unit - Run integration tests"
	@echo "  make test-regression - Run regression test suite"
	@echo "  make test-regression-verbose - Run regression tests (verbose)"
	@echo "  make test-regression-quick - Run regression tests (skip build)"
	@echo "  make test-mcp       - Run MCP regression tests"
	@echo "  make test-proxy     - Run Proxy regression tests"
	@echo "  make test-logging   - Run Logging regression tests"
	@echo "  make test-processes - Run Process regression tests"
	@echo "  make test-integration - Run Integration regression tests"
	@echo "  make test-tui       - Run TUI integration tests with new harness"
	@echo "  make test-tui-integration - Run comprehensive TUI integration tests"
	@echo "  make validate-tui   - Validate TUI functionality"
	@echo "  make setup-tui-test-env - Setup TUI test environment"
	@echo "  make test-all       - Run all tests (unit + regression + TUI)"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make build-all      - Build for multiple platforms"
	@echo "  make pack-extension - Package browser extension"
	@echo "  make help           - Show this help"
	@echo ""
	@echo "Quick start:"
	@echo "  $$ make install-user"
	@echo "  $$ brum"