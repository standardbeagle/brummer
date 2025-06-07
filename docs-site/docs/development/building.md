---
sidebar_position: 2
---

# Building Brummer

This guide covers building Brummer from source for development or custom deployments.

## Prerequisites

### Required Tools

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Git** - For cloning the repository
- **Make** (optional) - For using the Makefile

### Recommended Tools

- **golangci-lint** - For code linting
- **goreleaser** - For building releases
- **Node.js 18+** - For browser extension development

## Getting the Source

Clone the repository:

```bash
git clone https://github.com/yourusername/brummer.git
cd brummer
```

## Building the TUI

### Quick Build

```bash
go build -o brummer cmd/brummer/main.go
```

### Production Build

With optimizations and version information:

```bash
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse HEAD)

go build -ldflags "\
  -X main.Version=$VERSION \
  -X main.BuildTime=$BUILD_TIME \
  -X main.GitCommit=$GIT_COMMIT \
  -s -w" \
  -o brummer cmd/brummer/main.go
```

### Using Make

```bash
# Development build
make build

# Production build
make build-prod

# Build for all platforms
make build-all
```

### Cross-Compilation

Build for different platforms:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o brummer-linux-amd64 cmd/brummer/main.go

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o brummer-darwin-amd64 cmd/brummer/main.go

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o brummer-darwin-arm64 cmd/brummer/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o brummer.exe cmd/brummer/main.go
```

## Building the Browser Extension

### Development Build

```bash
cd browser-extension
./build.sh dev
```

This creates an unpacked extension in `browser-extension/build/dev/`.

### Production Build

```bash
cd browser-extension
./build.sh prod
```

This creates:
- `browser-extension/build/chrome/` - Chrome extension
- `browser-extension/build/firefox/` - Firefox extension
- `browser-extension/build/*.zip` - Packaged extensions

### Manual Build

1. **Install dependencies**:
   ```bash
   cd browser-extension
   npm install
   ```

2. **Build for Chrome**:
   ```bash
   npm run build:chrome
   ```

3. **Build for Firefox**:
   ```bash
   npm run build:firefox
   ```

## Development Workflow

### 1. Running in Development Mode

```bash
# Run with hot reload
go run cmd/brummer/main.go --dev

# With specific log level
go run cmd/brummer/main.go --log-level=debug

# With MCP server disabled
go run cmd/brummer/main.go --no-mcp
```

### 2. Testing Changes

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/process/...

# Run with race detection
go test -race ./...
```

### 3. Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Auto-fix issues
golangci-lint run --fix
```

## Build Configuration

### Build Tags

Use build tags for conditional compilation:

```bash
# Build without MCP support
go build -tags nomcp -o brummer cmd/brummer/main.go

# Build with experimental features
go build -tags experimental -o brummer cmd/brummer/main.go
```

### Environment Variables

Configure build behavior:

```bash
# Disable CGO (for static builds)
CGO_ENABLED=0 go build -o brummer cmd/brummer/main.go

# Custom Go flags
GOFLAGS="-mod=readonly" go build -o brummer cmd/brummer/main.go
```

## Debugging Builds

### 1. Debug Symbols

Include debug symbols:

```bash
go build -gcflags="all=-N -l" -o brummer cmd/brummer/main.go
```

### 2. Race Detection

Build with race detector:

```bash
go build -race -o brummer cmd/brummer/main.go
```

### 3. Profiling Support

Enable profiling:

```bash
go build -tags profile -o brummer cmd/brummer/main.go
```

## Release Process

### 1. Using GoReleaser

Create `.goreleaser.yml`:

```yaml
before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - replacements:
      darwin: Darwin
      linux: Linux
      windows: Windows
      amd64: x86_64

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
```

Run release:

```bash
# Dry run
goreleaser release --snapshot --rm-dist

# Actual release
goreleaser release
```

### 2. Manual Release

```bash
# Tag the release
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# Build releases
make release
```

## Docker Build

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /app
COPY . .
RUN make build-prod

# Runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/brummer /usr/local/bin/
ENTRYPOINT ["brummer"]
```

### Building

```bash
# Build image
docker build -t brummer:latest .

# Multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 -t brummer:latest .
```

## Troubleshooting Builds

### Common Issues

#### 1. Module Dependencies

```bash
# Clear module cache
go clean -modcache

# Download dependencies
go mod download

# Verify dependencies
go mod verify
```

#### 2. Build Cache

```bash
# Clear build cache
go clean -cache

# Clear test cache
go clean -testcache
```

#### 3. Version Information

Ensure git is available:

```bash
# Check git status
git status

# Fetch tags
git fetch --tags
```

### Platform-Specific Issues

#### macOS

- **Code signing**: Use `codesign` for distribution
- **Notarization**: Required for macOS Catalina+

#### Windows

- **Antivirus**: May flag unsigned executables
- **Terminal**: Some features require Windows Terminal

#### Linux

- **Dependencies**: Ensure glibc compatibility
- **Permissions**: May need execution permissions

## Performance Optimization

### 1. Binary Size

Reduce binary size:

```bash
# Strip debug info
go build -ldflags="-s -w" -o brummer cmd/brummer/main.go

# Use UPX (optional)
upx --best brummer
```

### 2. Build Time

Speed up builds:

```bash
# Parallel builds
go build -p 4 -o brummer cmd/brummer/main.go

# Cache builds
export GOCACHE=/path/to/cache
```

### 3. Runtime Performance

```bash
# Enable optimizations
go build -gcflags="-m" -o brummer cmd/brummer/main.go
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21', '1.22']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
      - run: go build ./...
      - run: go test ./...
```

## Next Steps

- Set up your [Development Environment](./contributing#development-setup)
- Learn about [Contributing](./contributing)
- Understand the [Architecture](./architecture)