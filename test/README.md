# Brummer Regression Test Suite

This directory contains a comprehensive regression testing framework for Brummer that validates all core functionality in both TUI and no-TUI modes.

## Overview

The test suite ensures that:
- ✅ **MCP Server** works correctly in both modes
- ✅ **Proxy Server** handles URL detection and request forwarding
- ✅ **Logging System** captures and displays output properly
- ✅ **Process Management** starts, monitors, and stops subprocesses
- ✅ **Integration** between all components functions seamlessly

## Test Categories

### 🔌 MCP Tests (`mcp_tests.go`)
- MCP server startup in both TUI and no-TUI modes
- MCP server URL display in system messages
- JSON-RPC request handling
- Session tracking and connection types
- Debug mode functionality

### 🌐 Proxy Tests (`proxy_tests.go`)
- Proxy server startup and URL detection
- HTTP request handling and forwarding
- Proxy disable flag (`--no-proxy`)
- Integration with process output parsing

### 📝 Logging Tests (`logging_tests.go`)
- System message logging in both modes
- Process output capture and display
- Error detection and highlighting
- Log timestamps and formatting
- Log filtering capabilities (TUI)

### ⚙️ Process Tests (`process_tests.go`)
- Process startup and management
- Multiple concurrent processes
- Process exit handling and cleanup
- Process ID generation
- Script detection from package.json

### 🔗 Integration Tests (`integration_tests.go`)
- Full stack functionality (all components together)
- MCP + Proxy integration
- Process + Logging integration
- Debug mode comprehensive testing
- Cleanup and shutdown procedures

## Running Tests

### Quick Start
```bash
# Run all regression tests
make test-regression

# Run with verbose output
make test-regression-verbose

# Run specific component tests
make test-mcp
make test-proxy
make test-logging
make test-processes
make test-integration
```

### Manual Execution
```bash
# Build and run all tests
./test/run_tests.sh

# Run with verbose output
./test/run_tests.sh --verbose

# Use existing binary (skip build)
./test/run_tests.sh --skip-build

# Run specific tests
./test/run_tests.sh --filter MCP
```

### Advanced Usage
```bash
# Use custom binary
./test/run_tests.sh --binary /path/to/custom/brum

# Run only integration tests
./test/run_tests.sh --filter Integration

# Verbose output with custom binary
./test/run_tests.sh --verbose --binary ./my-brum
```

## Test Structure

### Test Workspace
Each test run creates a temporary workspace (`test_workspace/`) containing:
- `package.json` with test scripts
- Sample configuration files
- Temporary process outputs

### Test Results
Tests report:
- ✅ **Pass/Fail status** for each test
- ⏱️ **Execution duration**
- 📝 **Detailed output** and error messages
- 📊 **Component-wise statistics**
- 🎯 **Mode coverage** (TUI vs no-TUI)

### Sample Output
```
🧪 Starting Brummer Regression Test Suite
==========================================

📋 Running MCP Server Tests...
  ✅ PASS [NoTUI] MCP - MCP Server Startup (1.2s)
  ✅ PASS [TUI] MCP - MCP Server Startup (3.1s)
  ✅ PASS [NoTUI] MCP - MCP URL Display (0.8s)
  ✅ PASS [NoTUI] MCP - JSON-RPC Requests (2.3s)

📋 Running Proxy Server Tests...
  ✅ PASS [NoTUI] Proxy - Proxy Server Startup (1.5s)
  ✅ PASS [TUI] Proxy - Proxy Server Startup (3.2s)
  ✅ PASS [NoTUI] Proxy - URL Detection (1.8s)

==================================================
📊 REGRESSION TEST SUMMARY
==================================================
✅ MCP: 8/8 passed (0 failed)
✅ Proxy: 5/5 passed (0 failed)
✅ Logging: 6/6 passed (0 failed)
✅ Processes: 6/6 passed (0 failed)
✅ Integration: 6/6 passed (0 failed)
--------------------------------------------------
OVERALL: 31/31 tests passed - ✅ ALL TESTS PASSED
Mode Coverage: 18 TUI tests, 13 NoTUI tests
```

## Test Development

### Adding New Tests
1. Add test function to appropriate `*_tests.go` file
2. Register test in the component's test runner
3. Follow the `TestResult` structure for consistency

### Test Function Template
```go
func (ts *TestSuite) testNewFeature() TestResult {
    start := time.Now()
    result := TestResult{
        Name:      "New Feature",
        Mode:      "NoTUI", // or "TUI"
        Component: "ComponentName",
        Passed:    false,
    }

    // Test implementation here
    cmd := exec.Command("timeout", "5s", ts.BinaryPath, "args...")
    output, err := cmd.CombinedOutput()
    
    result.Duration = time.Since(start)
    
    // Validation logic
    if /* success condition */ {
        result.Passed = true
        result.Details = []string{"Success details"}
    } else {
        result.Error = "Error description"
        result.Details = []string{"Debug information"}
    }

    return result
}
```

### Best Practices
- ⏱️ Use timeouts to prevent hanging tests
- 📝 Provide detailed error messages and debug info
- 🎯 Test both TUI and no-TUI modes where applicable
- 🧹 Clean up resources after tests
- 📊 Include meaningful assertions and validations

## Troubleshooting

### Common Issues
- **Port conflicts**: Tests use dynamic port allocation
- **Timing issues**: Increase timeouts for slower systems
- **Binary not found**: Ensure `make build` runs successfully
- **Permission errors**: Check file permissions on test scripts

### Debug Mode
Run tests with `--verbose` flag to see:
- Full command outputs
- Detailed test execution logs
- Component-specific debug information
- Timing and performance metrics

### Manual Testing
For debugging specific issues:
```bash
# Test MCP server manually
./brum -d test_workspace --no-tui --debug

# Test TUI mode manually
./brum -d test_workspace --debug

# Test specific script
./brum -d test_workspace --no-tui test
```

## CI/CD Integration

The test suite is designed for automated testing:
- Exit codes: 0 = success, 1 = failure
- Machine-readable output available
- Configurable timeouts and retries
- Minimal external dependencies

### GitHub Actions Example
```yaml
- name: Run Regression Tests
  run: |
    make build
    make test-regression
```

This comprehensive test suite ensures Brummer's reliability across all supported modes and prevents regressions during development.
