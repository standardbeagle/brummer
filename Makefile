# Brummer Makefile

# Variables
BINARY_NAME=brum
INSTALL_DIR=/usr/local/bin
USER_INSTALL_DIR=$(HOME)/.local/bin
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
.DEFAULT_GOAL := build

# Build the binary
.PHONY: build
build:
	@echo "🔨 Building Brummer..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) cmd/brum/main.go
	@echo "✅ Build complete: ./$(BINARY_NAME)"

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Install system-wide (requires sudo)
.PHONY: install
install: build
	@echo "📦 Installing to $(INSTALL_DIR)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod 755 $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Installed to $(INSTALL_DIR)/$(BINARY_NAME)"

# Install for current user
.PHONY: install-user
install-user: build
	@echo "📦 Installing to $(USER_INSTALL_DIR)..."
	@mkdir -p $(USER_INSTALL_DIR)
	@cp $(BINARY_NAME) $(USER_INSTALL_DIR)/
	@chmod 755 $(USER_INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Installed to $(USER_INSTALL_DIR)/$(BINARY_NAME)"
	@echo "💡 Make sure $(USER_INSTALL_DIR) is in your PATH"

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
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 cmd/brum/main.go
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 cmd/brum/main.go
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 cmd/brum/main.go
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 cmd/brum/main.go
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe cmd/brum/main.go
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
	@echo "  make test           - Run tests"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make build-all      - Build for multiple platforms"
	@echo "  make pack-extension - Package browser extension"
	@echo "  make help           - Show this help"
	@echo ""
	@echo "Quick start:"
	@echo "  $$ make install-user"
	@echo "  $$ brum"