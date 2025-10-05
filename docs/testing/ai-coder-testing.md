# AI Coder Testing Guide

This guide covers testing the AI Coder functionality in Brummer, including TTY-dependent tests and CI/CD integration.

## Overview

The AI Coder testing suite is organized into several categories:

1. **Unit Tests** - Fast tests without TTY requirements
2. **TTY Integration Tests** - Tests requiring real terminal allocation
3. **Headless Integration Tests** - Tests that run without TTY
4. **End-to-End Workflows** - Complete user scenario tests

## Test Organization

```
test/
├── integration_test.go              # General integration tests (no TTY)
├── tty/                            # TTY-specific tests
│   ├── tty_testutil.go            # TTY testing utilities
│   ├── ai_coder_lifecycle_test.go # Session management tests
│   ├── ai_coder_interaction_test.go # User interaction tests
│   ├── ai_coder_integration_test.go # End-to-end workflows
│   └── test_runner.go             # Test runner program
├── testutil/
│   ├── helpers.go                 # General test utilities
│   └── fixtures/                  # Test fixtures
└── ai_coder_system_test.go        # System-level placeholder tests
```

## Running Tests

### Local Development

#### Quick Test Run
```bash
# Run all AI coder tests
./test-tty-tests.sh

# Run only TTY tests
./test-tty-tests.sh tty

# Run only headless tests
./test-tty-tests.sh headless

# Run with verbose output
./test-tty-tests.sh -v

# Run specific test pattern
./test-tty-tests.sh -p TestAICoderSessionCreation
```

#### Manual Test Execution
```bash
# Build brummer first
make build

# Run TTY tests with Go directly
go test -tags=tty,integration -v ./test/tty/...

# Run headless integration tests
go test -tags=integration -v ./test/...

# Run with specific pattern
go test -tags=tty,integration -run TestAICoderSessionCreation -v ./test/tty/...
```

### CI/CD Integration

#### GitHub Actions Example
```yaml
name: AI Coder Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'

    - name: Build Brummer
      run: make build

    - name: Run Headless Tests
      run: go test -tags=integration -v ./test/...

    - name: Run TTY Tests
      run: |
        # Install script for TTY allocation
        sudo apt-get update
        sudo apt-get install -y expect

        # Run TTY tests
        BRUMMER_BINARY=./brum ./test-tty-tests.sh tty

    - name: Run All Tests
      run: BRUMMER_BINARY=./brum ./test-tty-tests.sh
```

#### Docker-based Testing
```dockerfile
FROM golang:1.21

# Install TTY support packages
RUN apt-get update && apt-get install -y \
    script \
    expect \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

# Build brummer
RUN make build

# Run tests
CMD ["./test-tty-tests.sh"]
```

## Test Categories

### Unit Tests
- **Location**: `internal/tui/*_test.go`, `internal/aicoder/*_test.go`
- **Build Tags**: None
- **TTY Required**: No
- **Purpose**: Test individual components in isolation
- **Execution Time**: < 1 second

```bash
go test ./internal/tui/... -v
go test ./internal/aicoder/... -v
```

### TTY Integration Tests
- **Location**: `test/tty/`
- **Build Tags**: `tty,integration`
- **TTY Required**: Yes
- **Purpose**: Test real terminal interactions
- **Execution Time**: 5-30 seconds

```bash
go test -tags=tty,integration -v ./test/tty/...
```

### Headless Integration Tests
- **Location**: `test/integration_test.go`
- **Build Tags**: `integration`
- **TTY Required**: No
- **Purpose**: Test component integration without TTY
- **Execution Time**: 2-10 seconds

```bash
go test -tags=integration -v ./test/...
```

## Test Utilities

### TTYTest Utility
The `TTYTest` struct provides TTY allocation and management:

```go
tt := NewTTYTest(t)
defer tt.Cleanup()

// Start brummer with TTY support
err := tt.StartTTY("--debug")

// Send input to terminal
err := tt.SendInput("echo hello\n")

// Navigate UI
err := tt.NavigateToAICoders()
err := tt.FocusTerminal()

// Wait for output
err := tt.WaitForOutput("hello", 5*time.Second)
```

### Test Fixtures
Test fixtures are located in `test/fixtures/` and include:
- Sample package.json files
- Mock scripts for testing
- Configuration files

## Writing New Tests

### Adding TTY Tests

1. Create test file in `test/tty/`
2. Add build tags: `//go:build tty && integration`
3. Use `TTYTest` utility for TTY allocation
4. Follow naming convention: `TestAICoder*`

Example:
```go
//go:build tty && integration
// +build tty,integration

package tty

func TestAICoderNewFeature(t *testing.T) {
    tt := NewTTYTest(t)
    defer tt.Cleanup()

    err := tt.StartTTY("--debug")
    if err != nil {
        t.Fatalf("failed to start brummer: %v", err)
    }

    // Test implementation
    tt.NavigateToAICoders()
    // ... test logic
}
```

### Adding Unit Tests

1. Create test file in appropriate package
2. Use mocks for external dependencies
3. Test individual functions and methods

Example:
```go
func TestAICoderController_NewFeature(t *testing.T) {
    // Setup mocks
    mockManager := &MockAICoderManager{}

    // Test implementation
    controller := NewAICoderController(...)
    result := controller.NewFeature()

    // Assertions
    assert.Equal(t, expected, result)
}
```

## Troubleshooting

### Common Issues

#### TTY Allocation Errors
```
no such device or address
```
**Solution**: Ensure tests run with proper TTY allocation. Use the test runner script instead of direct `go test`.

#### Test Timeouts
```
timeout waiting for output
```
**Solution**: Increase timeout or check if brummer is starting correctly.

#### Binary Not Found
```
brummer binary not found
```
**Solution**: Build brummer first with `make build` or set `BRUMMER_BINARY` environment variable.

#### Tests Not Found
```
no test files
```
**Solution**: Ensure build tags are specified correctly: `-tags=tty,integration`

### Debugging Test Failures

1. **Enable Verbose Output**:
   ```bash
   ./test-tty-tests.sh -v
   ```

2. **Run Individual Tests**:
   ```bash
   go test -tags=tty,integration -run TestSpecificTest -v ./test/tty/
   ```

3. **Check Test Logs**:
   ```bash
   # Look for captured output in test failure messages
   # Check brummer logs in test output
   ```

4. **Manual Testing**:
   ```bash
   # Run brummer manually to reproduce issues
   ./brum --debug
   ```

## Performance Considerations

- **TTY tests** are slower due to terminal allocation overhead
- **Parallel execution** is limited for TTY tests
- **Test isolation** ensures cleanup between tests
- **Resource usage** is monitored to prevent test leaks

## Best Practices

1. **Use descriptive test names** that explain the scenario
2. **Clean up resources** with `defer tt.Cleanup()`
3. **Use appropriate timeouts** for different operations
4. **Test both happy path and error conditions**
5. **Use fixtures** for consistent test data
6. **Document complex test scenarios** with comments
7. **Run tests locally** before submitting PRs

## Continuous Integration

### Prerequisites for CI

1. **TTY Support**: Install `script` or `expect` package
2. **Go Environment**: Use appropriate Go version
3. **Build Dependencies**: Ensure all build tools are available

### CI Pipeline Stages

1. **Build Stage**: Compile brummer binary
2. **Unit Test Stage**: Run fast unit tests
3. **Integration Test Stage**: Run headless tests
4. **TTY Test Stage**: Run TTY-dependent tests
5. **Cleanup Stage**: Remove temporary files

### Example CI Configuration

See the GitHub Actions example above for a complete CI setup that handles all test categories.