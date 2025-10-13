# TUI Integration Testing Guide

## Overview

This guide provides comprehensive documentation for the TUI Integration Testing Harness, a sophisticated testing framework designed specifically for validating Terminal User Interface (TUI) functionality in agent-based environments with limited shell capabilities.

The testing harness addresses critical challenges in TUI testing:

- **False Positive Prevention**: Eliminates TTY allocation failures that incorrectly indicate TUI brokenness
- **Real Code Path Testing**: Validates actual TUI application behavior through production services
- **Production Integrity**: Prevents test workarounds from compromising application functionality
- **Agent Compatibility**: Optimized for limited bash shell environments in AI agent systems

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    TUI Testing Harness                      │
├─────────────────────────────────────────────────────────────┤
│  TestHarness                                                │
│  ├── PTYLayer (Pseudo-terminal simulation)                 │
│  ├── InputSimulator (Keyboard input generation)            │
│  ├── StateValidator (State capture and comparison)         │
│  ├── TestReporter (Results and reporting)                  │
│  └── Configuration (Test environment setup)                │
├─────────────────────────────────────────────────────────────┤
│  Integration Points                                         │
│  ├── Brummer InputService                                   │
│  ├── Brummer NavigationService                              │
│  ├── BubbleTea Framework                                   │
│  └── Process Management                                     │
├─────────────────────────────────────────────────────────────┤
│  Test Categories                                            │
│  ├── Contract Tests (Interface compliance)                 │
│  ├── Integration Tests (End-to-end functionality)          │
│  ├── PTY Tests (Pseudo-terminal behavior)                 │
│  ├── Keyboard Tests (Input handling validation)            │
│  ├── View Tests (Navigation and state changes)             │
│  └── Performance Tests (Timing and resource usage)         │
└─────────────────────────────────────────────────────────────┘
```

### PTY Layer

The **PTY (Pseudo-terminal) Layer** is the core innovation that enables realistic TUI testing without requiring actual TTY allocation:

```go
type PTYLayer struct {
    mu              sync.RWMutex
    pty             *os.File        // PTY master file descriptor
    tty             *os.File        // PTY slave file descriptor
    allocated       bool            // PTY allocation status
    size            PTYSize         // Terminal dimensions
    ctx             context.Context // Operation context
    inputBuffer     []byte          // Input history for debugging
    outputBuffer    []byte          // Output capture for validation
}
```

**Key Features:**
- **Terminal Emulation**: Simulates real terminal behavior without `/dev/tty`
- **Input Simulation**: Converts `tea.KeyMsg` to terminal input sequences
- **Output Capture**: Captures TUI output for state validation
- **Size Management**: Controls terminal dimensions for responsive testing
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Quick Start

### Prerequisites

- Go 1.21+ recommended
- PTY support (Unix systems: native, Windows: limited)
- BubbleTea framework
- Brummer application binary

### Setup

1. **Clone and prepare the project:**
   ```bash
   cd /path/to/brummer
   git checkout your-feature-branch
   ```

2. **Run the test environment setup:**
   ```bash
   ./test/scripts/setup_test_env.sh
   ```

3. **Build the application:**
   ```bash
   make build
   ```

4. **Run quick validation:**
   ```bash
   ./test/quick_test.sh
   ```

### Running Tests

#### Quick Tests (Development)
```bash
# Fast subset for development feedback
./test/quick_test.sh

# Verbose development testing
./test/dev_test.sh
```

#### Comprehensive Testing
```bash
# Run all tests with default settings
./test/scripts/run_tui_tests.sh

# Run specific test categories
./test/scripts/run_tui_tests.sh Contract Integration

# Verbose with parallel execution
./test/scripts/run_tui_tests.sh -v -p All
```

#### Validation
```bash
# Full functionality validation
./test/scripts/validate_tui.sh

# Specific validation types
./test/scripts/validate_tui.sh Keyboard Navigation Performance

# Strict mode (warnings become failures)
./test/scripts/validate_tui.sh -s All
```

#### Pre-commit Testing
```bash
# Automatic testing before commits (configured via git hooks)
git commit -m "feat: add new feature"
# Tests run automatically, commit blocked on failures
```

## Test Categories

### 1. Contract Tests

Validate interface compliance and behavior contracts:

```go
func TestTestHarness_Contract(t *testing.T) {
    // Test lifecycle management
    harness := NewTestHarness(config)

    // Test initialization
    err := harness.Initialize()
    assert.NoError(t, err)
    assert.True(t, harness.IsInitialized())

    // Test component access
    inputSim := harness.GetInputSimulator()
    assert.NotNil(t, inputSim)

    // Test cleanup
    err = harness.Stop()
    assert.NoError(t, err)
}
```

**Coverage:**
- TestHarness interface methods
- InputSimulator contract compliance
- StateValidator behavior
- TestReporter functionality

### 2. Integration Tests

End-to-end testing of real TUI functionality:

```go
func TestKeyboardInputPTY_Simulation(t *testing.T) {
    // Setup test environment
    config := DefaultTestHarnessConfig()
    config.PTYLayer = ptyLayer

    harness := NewTestHarness(config)
    defer harness.Stop()

    // Define test scenario
    testCase := TestCase{
        ID: "keyboard_basic_input",
        Actions: []TestAction{
            {Type: "KeyPress", Key: "h"},
            {Type: "KeyPress", Key: "e"},
            {Type: "KeyPress", Key: "l"},
            {Type: "KeyPress", Key: "l"},
            {Type: "KeyPress", Key: "o"},
        },
        ExpectedStates: []ExpectedState{
            {View: "processes", ContentContains: "hello"},
        },
    }

    // Execute test
    result := harness.RunTest(testCase)

    // Validate results
    assert.Equal(t, TestStatusPass, result.Status)
}
```

**Coverage:**
- PTY layer integration
- Keyboard input through PTY
- View switching functionality
- Command palette operations
- Full-stack TUI workflows

### 3. PTY Layer Tests

Validate pseudo-terminal behavior:

```go
func TestPTYLayer_AllocationAndBasicFunctionality(t *testing.T) {
    config := DefaultPTYConfig()
    ptyLayer, err := NewPTYLayer(config)
    require.NoError(t, err)
    defer ptyLayer.Cleanup()

    // Test allocation
    err = ptyLayer.Allocate()
    assert.NoError(t, err)
    assert.True(t, ptyLayer.IsAllocated())

    // Test input/output
    testData := []byte("test input")
    err = ptyLayer.WriteInput(testData)
    assert.NoError(t, err)

    output, err := ptyLayer.ReadOutput()
    assert.NoError(t, err)

    // Test cleanup
    err = ptyLayer.Cleanup()
    assert.NoError(t, err)
    assert.False(t, ptyLayer.IsAllocated())
}
```

**Coverage:**
- PTY allocation and management
- Input/output operations
- Size management
- Cleanup and resource management
- Error handling

### 4. Performance Tests

Validate timing and resource usage:

```go
func TestPerformance_InputResponse(t *testing.T) {
    harness := setupPerformanceTestHarness(t)
    defer harness.Stop()

    startTime := time.Now()

    // Simulate rapid input
    for i := 0; i < 100; i++ {
        key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
        err := harness.GetInputSimulator().SendKey(key)
        require.NoError(t, err)
    }

    responseTime := time.Since(startTime)

    // Validate performance requirements
    assert.Less(t, responseTime, 5*time.Second)
    assert.Greater(t, harness.GetMetrics().InputEvents, 90)
}
```

**Coverage:**
- Input response timing (<100ms per keystroke)
- View switching performance (<200ms)
- Memory usage patterns (<100MB)
- Startup time (<5 seconds)
- PTY layer efficiency

## Configuration

### Test Harness Configuration

```go
type TestHarnessConfig struct {
    // PTY Configuration
    PTYLayer       *PTYLayer
    PTYSize        PTYSize
   PTYTimeout      time.Duration

    // Testing Configuration
    Verbose        bool
    StrictMode     bool
    CaptureLogs    bool
    KeepTestData   bool

    // Performance Settings
    MaxInputResponse time.Duration
    MaxViewSwitch    time.Duration
    MemoryLimit      int64

    // Integration Settings
    BrummerBinary    string
    TestWorkspace    string
    EnableParallel   bool
}
```

### Environment Variables

```bash
# Core Configuration
BRUMMER_BINARY="./brum"           # Path to brummer binary
TEST_TIMEOUT="120s"               # Default test timeout
VERBOSE="false"                   # Enable verbose output

# Test Environment
TEST_RESULTS_DIR="./test/results"  # Results output directory
TEST_LOGS_DIR="./test/logs"        # Log files directory
TEST_DATA_DIR="./test/data"        # Test data directory

# PTY Configuration
PTY_AVAILABLE="true"               # PTY layer availability
PTY_DEFAULT_ROWS="24"              # Default terminal rows
PTY_DEFAULT_COLS="80"              # Default terminal columns

# Performance Settings
MAX_STARTUP_TIME_MS="5000"         # Maximum startup time
MAX_INPUT_RESPONSE_MS="100"        # Maximum input response time
MEMORY_LIMIT_MB="100"              # Memory usage limit
```

### Configuration Files

#### test/config/test_env.toml
```toml
[environment]
test_mode = true
verbose_logging = false
pty_required = true
timeout_seconds = 30

[testing]
enable_debug = false
capture_logs = true
keep_test_data = false
parallel_tests = false

[pty]
default_rows = 24
default_cols = 80
enable_terminal_emulation = false
buffer_size = 4096

[performance]
max_startup_time_ms = 5000
max_input_response_ms = 100
max_view_switch_ms = 200
memory_limit_mb = 100
```

## Writing Tests

### Test Structure

```go
func TestYourFeature(t *testing.T) {
    // Setup test environment
    config := DefaultTestHarnessConfig()
    config.PTYLayer = setupPTYLayer(t)
    config.Verbose = testing.Verbose()

    harness := NewTestHarness(config)
    defer harness.Stop()

    // Initialize
    err := harness.Initialize()
    require.NoError(t, err)

    // Define test case
    testCase := TestCase{
        ID:          "your_test_case_id",
        Description: "Description of what you're testing",
        Actions: []TestAction{
            {
                Type: "KeyPress",
                Key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}},
                Delay: 50 * time.Millisecond,
            },
            {
                Type: "KeyPress",
                Key:  tea.KeyMsg{Type: tea.KeyEnter},
                Delay: 100 * time.Millisecond,
            },
        },
        ExpectedStates: []ExpectedState{
            {
                View:            "expected_view",
                ContentContains: []string{"expected text"},
                CursorPosition:  CursorPosition{Row: 5, Col: 10},
            },
        },
        Timeout: 10 * time.Second,
    }

    // Execute test
    result := harness.RunTest(testCase)

    // Validate results
    assert.Equal(t, TestStatusPass, result.Status)
    if result.Status == TestStatusFail {
        t.Logf("Test failed: %s", result.Error)
        t.Logf("Expected state: %+v", testCase.ExpectedStates[0])
        t.Logf("Actual state: %+v", result.ActualState)
    }
}
```

### Action Types

#### KeyPress Actions
```go
TestAction{
    Type: "KeyPress",
    Key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}},
    Delay: 50 * time.Millisecond,
}
```

#### View Switch Actions
```go
TestAction{
    Type:  "ViewSwitch",
    View:  "logs",
    Delay: 100 * time.Millisecond,
}
```

#### Command Actions
```go
TestAction{
    Type:    "Command",
    Command: "quit",
    Delay:   200 * time.Millisecond,
}
```

#### Wait Actions
```go
TestAction{
    Type:  "Wait",
    Delay: 500 * time.Millisecond,
}
```

### Expected States

```go
ExpectedState{
    View:            "processes",           // Expected current view
    ContentContains: []string{"nginx"},     // Text that should be present
    ContentNotContains: []string{"error"}, // Text that should NOT be present
    CursorPosition:  CursorPosition{Row: 5, Col: 10}, // Expected cursor position
    ProcessCount:    &Range{Min: 1, Max: 10},       // Expected process count
    UIElements:      []string{"header", "footer"},  // Expected UI elements
    CustomValidators: []StateValidatorFunc{customValidationFunc},
}
```

### Custom Validators

```go
func customValidationFunc(snapshot StateSnapshot) error {
    // Custom validation logic
    model, ok := snapshot.Model.(YourModelType)
    if !ok {
        return fmt.Errorf("unexpected model type: %T", snapshot.Model)
    }

    if model.SomeProperty != "expected_value" {
        return fmt.Errorf("property validation failed: got %s, want expected_value", model.SomeProperty)
    }

    return nil
}
```

## Troubleshooting

### Common Issues

#### 1. PTY Layer Not Available
```
Error: PTY layer test failed
```

**Solutions:**
- **Unix Systems**: Ensure `/dev/ptmx` is accessible
- **Windows**: PTY support is limited, consider WSL or skip PTY tests
- **Docker**: Run with `--device /dev/ptmx` for PTY access

```bash
# Check PTY availability
ls -la /dev/ptmx
go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality
```

#### 2. Test Timeout Issues
```
Error: timeout running tests
```

**Solutions:**
- Increase timeout: `TEST_TIMEOUT=300s ./test/scripts/run_tui_tests.sh`
- Check system performance and resource usage
- Run tests individually to identify slow tests

```bash
# Run with extended timeout
./test/scripts/run_tui_tests.sh -t 300s

# Identify slow tests
./test/scripts/run_tui_tests.sh -v | grep "took.*ms"
```

#### 3. Binary Not Found
```
Error: Brummer binary not found
```

**Solutions:**
- Build the binary: `make build`
- Specify binary path: `BRUMMER_BINARY=/path/to/brum ./test/scripts/run_tui_tests.sh`
- Check binary permissions: `chmod +x brum`

#### 4. Integration Test Failures
```
Error: Integration test failed
```

**Solutions:**
- Check test logs: `cat test/logs/integration_*.log`
- Validate test environment: `./test/scripts/setup_test_env.sh`
- Run tests with verbose output: `./test/scripts/run_tui_tests.sh -v`

#### 5. Memory Issues
```
Error: out of memory
```

**Solutions:**
- Increase memory limits in configuration
- Run tests sequentially (disable parallel)
- Check for memory leaks in test code

```bash
# Run sequentially
./test/scripts/run_tui_tests.sh --no-parallel

# Increase memory limit
MEMORY_LIMIT_MB=200 ./test/scripts/run_tui_tests.sh
```

### Debug Mode

Enable comprehensive debugging:

```bash
# Enable verbose logging
VERBOSE=true ./test/scripts/run_tui_tests.sh

# Keep test artifacts for inspection
KEEP_LOGS=true ./test/scripts/run_tui_tests.sh

# Run specific failing test
go test -tags=tty,integration ./test/tui/integration -v -run TestFailingTestName -timeout 300s
```

### Log Analysis

```bash
# Find failed tests
grep -r "FAIL" test/results/

# Check performance issues
grep -r "slow\|timeout" test/logs/

# Analyze PTY issues
grep -r "pty\|terminal" test/logs/

# Check memory usage
grep -r "memory\|alloc" test/logs/
```

## Best Practices

### Test Design

1. **Test Real Scenarios**: Focus on user workflows rather than implementation details
2. **Validate State Changes**: Ensure tests verify actual state changes, not just method calls
3. **Use Timeouts Appropriately**: Set reasonable timeouts based on expected performance
4. **Clean Up Resources**: Always clean up PTY layers and temporary files
5. **Avoid Sleep**: Use context and event-based synchronization instead of fixed delays

### Performance Considerations

1. **Parallel Execution**: Enable parallel testing for faster feedback
2. **Resource Limits**: Set appropriate memory and time limits
3. **Test Isolation**: Ensure tests don't interfere with each other
4. **Batch Operations**: Group related operations to reduce overhead

### CI/CD Integration

1. **Fast Feedback**: Run quick tests first, then comprehensive tests
2. **Fail Fast**: Configure strict mode to catch issues early
3. **Artifact Collection**: Save test reports and logs for analysis
4. **Environment Consistency**: Use the same test environment across stages

#### GitHub Actions Example
```yaml
name: TUI Tests
on: [push, pull_request]

jobs:
  tui-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'

    - name: Setup Test Environment
      run: ./test/scripts/setup_test_env.sh -c

    - name: Build Application
      run: make build

    - name: Run TUI Tests
      run: ./test/scripts/ci_test.sh

    - name: Upload Test Results
      uses: actions/upload-artifact@v3
      with:
        name: test-results
        path: test/results/
```

### Maintenance

1. **Regular Updates**: Keep test fixtures and expected states current
2. **Performance Monitoring**: Track test execution times and resource usage
3. **Test Coverage**: Monitor and maintain comprehensive test coverage
4. **Documentation**: Keep test documentation synchronized with code changes

## Advanced Usage

### Custom Test Scenarios

Create complex test scenarios:

```go
func TestComplexWorkflow(t *testing.T) {
    config := DefaultTestHarnessConfig()
    config.EnableParallel = false  // Disable for complex tests

    harness := NewTestHarness(config)
    defer harness.Stop()

    // Multi-step workflow
    workflow := []TestAction{
        // Start application
        {Type: "Start"},

        // Navigate to logs view
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}, Delay: 100},

        // Search for specific log entry
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}, Delay: 100},
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}, Delay: 50},
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}, Delay: 50},
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}, Delay: 50},
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyEnter}, Delay: 100},

        // Validate search results
        {Type: "ValidateState", ExpectedState: ExpectedState{
            ContentContains: []string{"error"},
        }},

        // Return to processes view
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, Delay: 100},

        // Quit application
        {Type: "KeyPress", Key: tea.KeyMsg{Type: tea.KeyCtrlC}, Delay: 100},
    }

    testCase := TestCase{
        ID:          "complex_workflow",
        Description: "Multi-step navigation and search workflow",
        Actions:     workflow,
        Timeout:     30 * time.Second,
    }

    result := harness.RunTest(testCase)
    assert.Equal(t, TestStatusPass, result.Status)
}
```

### Performance Benchmarks

Create performance benchmarks:

```go
func BenchmarkInputSimulation(b *testing.B) {
    config := DefaultTestHarnessConfig()
    harness := NewTestHarness(config)
    defer harness.Stop()

    key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            err := harness.GetInputSimulator().SendKey(key)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

### Custom Test Reporters

Implement custom reporting:

```go
type CustomTestReporter struct {
    *TestResultReporter
    customMetrics map[string]interface{}
}

func (ctr *CustomTestReporter) GenerateCustomReport(results []TestResult) error {
    // Generate custom report format
    report := map[string]interface{}{
        "custom_metrics": ctr.customMetrics,
        "test_results":   results,
        "generated_at":   time.Now(),
    }

    // Save to custom format
    return ctr.SaveCustomReport(report, "custom_report.json")
}
```

## API Reference

### Core Interfaces

#### TestHarness Interface
```go
type TestHarness interface {
    Initialize() error
    Start() error
    Stop() error
    RunTest(testCase TestCase) TestResult
    GetInputSimulator() InputSimulator
    GetStateValidator() StateValidator
    GetTestReporter() TestReporter
    IsInitialized() bool
    GetMetrics() TestMetrics
}
```

#### InputSimulator Interface
```go
type InputSimulator interface {
    SendKey(key tea.KeyMsg) error
    SendKeys(keys []tea.KeyMsg) error
    SendTextInput(text string) error
    SendKeySequence(sequence []TestAction) error
    SetDelay(delay time.Duration)
    GetInputHistory() []tea.KeyMsg
}
```

#### StateValidator Interface
```go
type StateValidator interface {
    CaptureState(model interface{}) (StateSnapshot, error)
    ValidateState(snapshot StateSnapshot, expected ExpectedState) error
    CompareStates(actual, expected StateSnapshot) StateComparison
    AddCustomValidator(name string, validator StateValidatorFunc)
    GetValidationHistory() []ValidationResult
}
```

### Configuration Structures

```go
type TestHarnessConfig struct {
    PTYLayer       *PTYLayer
    PTYSize        PTYSize
    PTYTimeout     time.Duration
    Verbose        bool
    StrictMode     bool
    CaptureLogs    bool
    KeepTestData   bool
    MaxInputResponse time.Duration
    MaxViewSwitch    time.Duration
    MemoryLimit      int64
    BrummerBinary    string
    TestWorkspace    string
    EnableParallel   bool
}

type PTYConfig struct {
    Size        PTYSize
    Timeout     time.Duration
    EnableDebug bool
}

type PTYSize struct {
    Rows uint16
    Cols uint16
}
```

### Test Data Structures

```go
type TestCase struct {
    ID             string
    Description    string
    Actions        []TestAction
    ExpectedStates []ExpectedState
    Timeout        time.Duration
    SetupFunc      func() error
    CleanupFunc    func() error
}

type TestAction struct {
    Type        string
    Key         tea.KeyMsg
    Command     string
    View        string
    Delay       time.Duration
    CustomData  interface{}
}

type ExpectedState struct {
    View              string
    ContentContains   []string
    ContentNotContains []string
    CursorPosition    CursorPosition
    ProcessCount      *Range
    UIElements        []string
    CustomValidators  []StateValidatorFunc
}

type TestResult struct {
    TestID        string
    Status        TestStatus
    Duration      time.Duration
    Error         string
    ExpectedState ExpectedState
    ActualState   ActualState
    Metrics       TestMetrics
    Timestamp     time.Time
}
```

## Contributing

### Adding New Tests

1. **Create Test File**: Add test files in appropriate directories
   - Contract tests: `test/tui/harness/*_contract_test.go`
   - Integration tests: `test/tui/integration/*_test.go`
   - PTY tests: `test/tui/pty/*_test.go`

2. **Follow Naming Conventions**:
   ```go
   func TestComponentName_SpecificFeature(t *testing.T) {
       // Test implementation
   }
   ```

3. **Add Test Categories**: Update test execution scripts to include new tests
4. **Update Documentation**: Document new test scenarios and validation rules

### Extending the Framework

1. **New Action Types**: Extend `TestAction` type and corresponding handlers
2. **Custom Validators**: Implement `StateValidatorFunc` for domain-specific validation
3. **New Reporters**: Implement custom test reporters for different output formats
4. **Performance Metrics**: Add new performance tracking capabilities

### Code Style

Follow the established Go coding standards:
- Use `gofmt` and `golangci-lint`
- Write clear, descriptive function names
- Add comprehensive comments for public APIs
- Include table-driven tests for multiple scenarios

## Support

For issues, questions, or contributions:

1. **Check the logs**: Review test logs and error messages
2. **Consult troubleshooting**: See the troubleshooting section above
3. **Review test examples**: Examine existing test implementations
4. **Check configuration**: Verify test environment setup

---

**Note**: This testing harness is specifically designed for agent-based environments with limited shell capabilities. For traditional development environments, additional testing tools and approaches may be beneficial.