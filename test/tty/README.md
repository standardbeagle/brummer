# AI Coder TTY Tests

This directory contains TTY-dependent integration tests for the AI Coder functionality in Brummer.

## Overview

These tests require real terminal allocation and test the complete AI Coder user experience, including:
- Session creation and management
- Terminal interaction and key bindings
- Multi-session workflows
- Error handling and recovery
- End-to-end user scenarios

## Test Structure

```
test/tty/
├── README.md                      # This file
├── tty_testutil.go               # TTY testing utilities and framework
├── ai_coder_lifecycle_test.go    # Session lifecycle management tests
├── ai_coder_interaction_test.go  # User interaction and terminal behavior tests
├── ai_coder_integration_test.go  # End-to-end workflow tests
└── test_runner.go                # Go test runner program
```

## Running Tests

### Quick Start
```bash
# From project root
./test-tty-tests.sh tty

# Or with Go directly
go test -tags=tty,integration -v ./test/tty/...
```

### Individual Test Files
```bash
# Run lifecycle tests
go test -tags=tty,integration -run TestAICoderSession -v ./test/tty/

# Run interaction tests
go test -tags=tty,integration -run TestAICoderTerminal -v ./test/tty/

# Run integration tests
go test -tags=tty,integration -run TestAICoderEndToEnd -v ./test/tty/
```

### With Custom Options
```bash
# With longer timeout
go test -tags=tty,integration -timeout 120s -v ./test/tty/...

# Run specific test
go test -tags=tty,integration -run TestAICoderSessionCreation -v ./test/tty/

# With verbose output
go test -tags=tty,integration -v ./test/tty/... 2>&1 | tee test.log
```

## Test Categories

### 1. Session Lifecycle Tests (`ai_coder_lifecycle_test.go`)
- `TestAICoderSessionCreation` - Creating and managing sessions
- `TestAICoderSessionNavigation` - Switching between sessions
- `TestAICoderTerminalProvider` - Terminal provider functionality
- `TestAICoderErrorHandling` - Error scenarios and recovery
- `TestAICoderShutdown` - Graceful shutdown with active sessions

### 2. User Interaction Tests (`ai_coder_interaction_test.go`)
- `TestAICoderTerminalInteraction` - Focus/unfocus, key bindings
- `TestAICoderDataInjection` - Ctrl+E/L/T/B data injection
- `TestAICoderSessionNavigation` - Session switching navigation
- `TestAICoderScrolling` - Viewport scrolling functionality

### 3. Integration Tests (`ai_coder_integration_test.go`)
- `TestAICoderEndToEndWorkflow` - Complete user workflows
- `TestAICoderMultiSessionWorkflow` - Managing multiple sessions
- `TestAICoderDebugWorkflow` - Debug mode functionality
- `TestAICoderTerminalProviderWorkflow` - Terminal provider usage
- `TestAICoderErrorRecoveryWorkflow` - Error handling scenarios

## Test Utilities

The `tty_testutil.go` file provides the `TTYTest` struct for TTY allocation:

```go
tt := NewTTYTest(t)
defer tt.Cleanup()

// Start brummer with TTY support
err := tt.StartTTY("--debug")

// UI Navigation
err := tt.NavigateToAICoders()
err := tt.FocusTerminal()
err := tt.UnfocusTerminal()

// Input simulation
err := tt.SendInput("echo hello\n")
err := tt.TypeInTerminal("some text")

// Output verification
err := tt.WaitForOutput("hello", 5*time.Second)
tt.AssertContains("expected text")
```

## Key Bindings Tested

| Key Combination | Function | Test Coverage |
|-----------------|----------|---------------|
| Tab x5          | Navigate to AI Coders | ✅ |
| Enter           | Focus terminal | ✅ |
| Ctrl+Q          | Unfocus terminal | ✅ |
| Ctrl+N          | Next session | ✅ |
| Ctrl+P          | Previous session | ✅ |
| Ctrl+D          | Detach from session | ✅ |
| F11             | Toggle full screen | ✅ |
| F12             | Toggle debug mode | ✅ |
| Ctrl+H          | Toggle help | ✅ |
| Ctrl+E/L/T/B    | Data injection | ✅ |
| PgUp/PgDn       | Scroll viewport | ✅ |
| ESC             | Exit full screen | ✅ |

## Test Fixtures

Tests use fixtures from `test/fixtures/`:
- Sample package.json files
- Mock scripts
- Configuration templates

## Prerequisites

### System Requirements
- Linux/macOS/Windows with TTY support
- Go 1.21 or later
- `script` command (usually pre-installed)

### Build Requirements
```bash
# Build brummer first
make build

# Or build manually
go build -o brum ./cmd/brum/main.go
```

### Environment Variables
- `BRUMMER_BINARY` - Path to brummer binary (optional)
- `FORCE_TTY` - Force TTY allocation (set by test runner)

## Troubleshooting

### TTY Allocation Issues
If tests fail with "no such device or address", ensure:
1. Not running in a limited environment
2. `script` command is available
3. Terminal supports TTY allocation

### Binary Not Found
```bash
# Build brummer
make build

# Or set environment variable
export BRUMMER_BINARY=/path/to/brum
```

### Test Timeouts
```bash
# Increase timeout
go test -tags=tty,integration -timeout 120s -v ./test/tty/...
```

### Debugging Test Failures
```bash
# Run with verbose output
./test-tty-tests.sh -v

# Run individual test
go test -tags=tty,integration -run TestSpecificTest -v ./test/tty/

# Check captured output
go test -tags=tty,integration -v ./test/tty/... 2>&1 | grep -A10 -B10 "FAIL"
```

## Best Practices

1. **Always use `defer tt.Cleanup()`** for proper resource cleanup
2. **Use appropriate timeouts** for different operations
3. **Add sleep statements** after UI operations to allow for rendering
4. **Test both happy path and error conditions**
5. **Use descriptive test names** that explain the scenario
6. **Log test progress** with `t.Logf()` for debugging

## CI/CD Integration

These tests are designed to work in CI/CD environments:

```yaml
# GitHub Actions example
- name: Run TTY Tests
  run: |
    sudo apt-get update && sudo apt-get install -y script
    BRUMMER_BINARY=./brum ./test-tty-tests.sh tty
```

See `../docs/testing/ai-coder-testing.md` for complete CI/CD setup instructions.

## Contributing

When adding new tests:

1. **Use build tags**: `//go:build tty && integration`
2. **Follow naming convention**: `TestAICoder*`
3. **Use TTYTest utility** for consistent test setup
4. **Test complete workflows** rather than individual functions
5. **Add appropriate cleanup** to prevent resource leaks
6. **Document complex scenarios** with comments
7. **Update this README** when adding new test categories