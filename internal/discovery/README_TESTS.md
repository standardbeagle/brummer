# Discovery System Tests

This directory contains comprehensive tests for the Brummer instance discovery system. These tests are designed to be rock-solid - they will pass when discovery is working correctly and fail when there are issues.

## Test Files

### 1. `discovery_comprehensive_test.go`
Comprehensive tests covering all aspects of the discovery system:
- **Directory Creation**: Tests that discovery creates required directories
- **File Validation**: Tests instance file format and validation
- **File Watcher**: Tests that all file changes are detected reliably
- **Concurrent Operations**: Tests thread safety with many simultaneous operations
- **Stale Cleanup**: Tests removal of dead instances
- **Error Handling**: Tests graceful handling of corrupted files

### 2. `discovery_integration_test.go` (in mcp package)
Integration tests for discovery with connection manager:
- **Discovery to Connection**: Tests the full flow from file discovery to MCP connection
- **State Transitions**: Tests connection state changes (discovered → connecting → active)
- **Multiple Instances**: Tests hub discovering and connecting to multiple instances
- **Instance Removal**: Tests handling when instance files disappear
- **Rapid Changes**: Tests system stability under rapid add/remove cycles

### 3. `diagnostics.go` & `diagnostics_test.go`
Diagnostic tools for troubleshooting discovery issues:
- **Diagnostic Report**: Comprehensive system state analysis
- **Issue Diagnosis**: Specific actionable diagnosis for common problems
- **Setup Verification**: Pre-flight checks for discovery system
- **Human-Readable Output**: Clear reports for debugging

### 4. `debug_example_test.go`
Examples showing how to debug discovery issues:
- **Step-by-step debugging process**
- **Event tracking and timeline analysis**
- **Production debugging patterns**
- **Troubleshooting workflows**

## Running the Tests

### Run All Discovery Tests
```bash
go test -v ./internal/discovery
```

### Run Specific Test Categories
```bash
# Directory and file handling
go test -v ./internal/discovery -run TestDiscoveryDirectoryCreation
go test -v ./internal/discovery -run TestInstanceFileValidation

# Concurrency and reliability
go test -v ./internal/discovery -run TestFileWatcherReliability
go test -v ./internal/discovery -run TestDiscoveryConcurrentOperations

# Integration with hub
go test -v ./internal/mcp -run TestConnectionManagerDiscoveryIntegration
go test -v ./internal/mcp -run TestMultipleInstanceDiscovery

# Diagnostics
go test -v ./internal/discovery -run TestDiagnostic
```

### Run with Race Detection
```bash
go test -race -v ./internal/discovery
```

## Common Discovery Issues and How These Tests Help

### Issue: "Hub isn't finding instances"

**Tests that verify this:**
- `TestHubDiscoveryIntegration` - Simulates full hub discovery flow
- `TestFileWatcherReliability` - Ensures file changes are detected
- `TestInstanceFileValidation` - Checks file format is correct

**Diagnostic tools:**
```go
// Generate diagnostic report
report, _ := GenerateDiagnosticReport(instancesDir)
PrintDiagnosticReport(os.Stdout, report)

// Get specific diagnosis
diagnosis, _ := DiagnoseDiscoveryIssue(instancesDir)
fmt.Println(diagnosis)
```

### Issue: "Instance registered but not connecting"

**Tests that verify this:**
- `TestDiscoveryToConnectionStateFlow` - Tracks state transitions
- `TestConnectionManagerDiscoveryIntegration` - Tests connection flow

**Debug approach:**
1. Check if instance file exists and is valid
2. Verify connection manager received discovery callback
3. Check connection state transitions
4. Look for connection errors in logs

### Issue: "Instances disappearing randomly"

**Tests that verify this:**
- `TestStaleInstanceCleanup` - Tests cleanup logic
- `TestInstanceFileDisappearance` - Tests removal detection
- `TestRapidInstanceChurn` - Tests rapid add/remove cycles

**Common causes:**
- Process no longer running (PID check fails)
- LastPing timeout exceeded (5 minutes default)
- File permissions changed
- Disk full or I/O errors

## Test Design Philosophy

These tests follow several key principles:

1. **Deterministic**: Tests use actual process PIDs and controlled timing
2. **Comprehensive**: Cover both happy paths and error conditions
3. **Diagnostic**: Provide clear information when failures occur
4. **Realistic**: Simulate real-world scenarios (concurrent ops, file corruption)
5. **Fast**: Most tests complete in milliseconds

## Adding New Tests

When adding new discovery tests:

1. Use the `TestHelper` for consistent setup/teardown
2. Always clean up resources (use `defer`)
3. Test both success and failure cases
4. Include diagnostic output for failures
5. Run with `-race` flag to catch concurrency issues

## Debugging Failed Tests

If a discovery test fails:

1. **Check the diagnostic output** - Tests print detailed state information
2. **Run with verbose flag** - `go test -v` for more details
3. **Use diagnostic tools** - Generate a diagnostic report
4. **Check event timeline** - Many tests track event sequences
5. **Verify environment** - Check permissions, disk space, etc.

## Performance Considerations

The tests are designed to be fast while thorough:
- File operations use atomic writes
- Concurrent tests use controlled goroutine counts
- Timeouts are aggressive but reasonable (50-500ms typical)
- No sleeps except where testing timing-sensitive behavior