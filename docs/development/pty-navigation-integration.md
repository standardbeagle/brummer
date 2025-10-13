# PTY-NavigationService Integration

This document describes the PTY-NavigationService integration that enables realistic TUI testing without requiring a real TTY allocation. This integration connects PTY view switching with the existing Brummer NavigationService for comprehensive testing scenarios.

## Overview

The PTY-NavigationService integration provides:

- **Realistic Terminal Simulation**: Uses pseudo-terminals to simulate real TUI interactions
- **Production Service Integration**: Uses the actual NavigationService used in production
- **Comprehensive Validation**: Validates view switching, state consistency, and performance
- **Testing Harness**: Enables automated testing of TUI navigation in CI/CD environments

## Architecture

The integration consists of several key components:

### Core Components

1. **PTYNavigationAdapter** (`/internal/tui/services/pty_navigation_adapter.go`)
   - Bridges PTY input simulation with NavigationService
   - Handles key press simulation and state tracking
   - Provides statistics and monitoring

2. **PTYNavigationStateTracker** (`/internal/tui/services/pty_navigation_state_tracker.go`)
   - Tracks navigation state transitions
   - Validates state consistency
   - Records performance metrics

3. **PTYViewSwitchingValidator** (`/internal/tui/services/pty_view_switching_validator.go`)
   - Provides comprehensive validation of view switching
   - Tests number keys, tab cycling, and key sequences
   - Validates performance and state consistency

4. **PTY Layer** (`/test/tui/pty/pty_layer.go`)
   - Manages pseudo-terminal allocation and terminal emulation
   - Handles input/output simulation
   - Provides realistic terminal behavior

### Integration Flow

```
Test Input → PTYNavigationAdapter → NavigationService → View Switch
     ↓                    ↓                      ↓
   PTY Layer    →   State Tracking   →   Validation
     ↓                    ↓                      ↓
  Terminal I/O    →   Statistics      →   Test Results
```

## Usage Examples

### Basic Integration Setup

```go
import (
    "github.com/standardbeagle/brummer/internal/tui/services"
    "github.com/standardbeagle/brummer/pkg/events"
    "github.com/standardbeagle/brummer/test/tui/pty"
)

// Create test environment
eventBus := events.NewEventBus()
navigationService := services.NewNavigationService(eventBus)

// Create PTY layer
ptyConfig := pty.DefaultPTYConfig()
ptyLayer, err := pty.NewPTYLayer(ptyConfig)
if err != nil {
    log.Fatal(err)
}

// Allocate PTY
err = ptyLayer.Allocate()
if err != nil {
    log.Fatal(err)
}
defer ptyLayer.Cleanup()

// Create navigation adapter
adapterConfig := services.DefaultPTYNavigationAdapterConfig()
adapter, err := services.NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
if err != nil {
    log.Fatal(err)
}

// Start adapter
err = adapter.Start()
if err != nil {
    log.Fatal(err)
}
defer adapter.Stop()
```

### Testing Number Key Navigation

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/tui/services"
)

// Create validator
validator, err := services.NewPTYViewSwitchingValidator(ptyLayer, adapter, validatorConfig)
if err != nil {
    log.Fatal(err)
}

// Test number key 2 -> LogsView
result := validator.ValidateNumberKey(2, messages.LogsView)
if result.Success {
    fmt.Printf("✅ Number key 2 -> %s (duration: %v)\n", result.ActualView, result.Duration)
} else {
    fmt.Printf("❌ Number key 2 failed: %s\n", result.Error)
}
```

### Testing Tab Cycling

```go
// Test 5 tab cycles
result := validator.ValidateTabCycling(5)
if result.Success {
    fmt.Printf("✅ Tab cycling: visited %d unique views\n", result.Details["unique_views_visited"])
} else {
    fmt.Printf("❌ Tab cycling failed: %s\n", result.Error)
}
```

### Testing Key Sequences

```go
// Test complex key sequence
keySequence := []tea.KeyMsg{
    {Type: tea.KeyRunes, Runes: []rune{'2'}}, // Logs
    {Type: tea.KeyTab},                        // Next
    {Type: tea.KeyRunes, Runes: []rune{'4'}}, // AICoders
}

result := validator.ValidateKeySequence(keySequence, messages.AICodersView, "number_key")
if result.Success {
    fmt.Printf("✅ Key sequence completed -> %s\n", result.ActualView)
}
```

## Configuration

### PTYNavigationAdapterConfig

```go
type PTYNavigationAdapterConfig struct {
    EventBufferSize   int           // Buffer size for events (default: 1000)
    ErrorBufferSize   int           // Buffer size for errors (default: 100)
    ReadTimeout       time.Duration // PTY read timeout (default: 100ms)
    ShutdownTimeout   time.Duration // Shutdown timeout (default: 5s)
    ActivityTimeout   time.Duration // Inactivity timeout (default: 30s)
    ViewSwitchTimeout time.Duration // View switch timeout (default: 1s)
    EnableStatistics  bool          // Enable statistics collection
    EnableDebug       bool          // Enable debug logging
    TrackViewHistory  bool          // Track view history
    ValidateSwitches  bool          // Validate view switches
    RecordTriggers    bool          // Record switch triggers
}
```

### PTYNavigationStateTrackerConfig

```go
type PTYNavigationStateTrackerConfig struct {
    MaxHistorySize      int           // Maximum history entries (default: 100)
    TrackDurations      bool          // Track transition durations
    TrackErrors         bool          // Track errors
    TrackKeySequences   bool          // Track key sequences
    ValidateTransitions bool          // Validate transitions
    ValidateViews       bool          // Validate views
    ValidateNumberKeys  bool          // Validate number keys
    ValidateTabCycling  bool          // Validate tab cycling
    MaxSwitchDuration   time.Duration // Maximum allowed switch time
    WarningThreshold    time.Duration // Performance warning threshold
    EnableDebug         bool          // Enable debug logging
}
```

### PTYViewSwitchingValidatorConfig

```go
type PTYViewSwitchingValidatorConfig struct {
    EnableValidation      bool          // Enable validation
    ValidateStateChanges  bool          // Validate state changes
    ValidateTriggers      bool          // Validate triggers
    ValidatePerformance   bool          // Validate performance
    ValidateHistory       bool          // Validate history
    DefaultTimeout        time.Duration // Default timeout (default: 5s)
    MaxValidationTime     time.Duration // Max validation time (default: 30s)
    MinSwitchTime         time.Duration // Minimum switch time (default: 1ms)
    MaxSwitchTime         time.Duration // Maximum switch time (default: 2s)
    PerformanceThreshold  time.Duration // Performance threshold (default: 1s)
    TestInvalidKeys       bool          // Test invalid keys
    TestRapidSwitching    bool          // Test rapid switching
    TestKeySequences      bool          // Test key sequences
    TestStateConsistency  bool          // Test state consistency
    MaxValidationHistory  int           // Max validation history (default: 100)
    TrackValidationStats  bool          // Track validation statistics
    EnableDebug           bool          // Enable debug logging
}
```

## Testing Scenarios

### 1. Number Key Testing

Validates that pressing number keys (1-5) switches to the correct views:

```go
// Expected mappings:
// 1 -> ProcessesView
// 2 -> LogsView
// 3 -> ErrorsView
// 4 -> AICodersView
// 5 -> SettingsView

for i := 1; i <= 5; i++ {
    result := validator.ValidateNumberKey(i, getExpectedViewForNumberKey(i))
    assert.True(t, result.Success)
}
```

### 2. Tab Cycling Testing

Validates that Tab key cycles through views in the correct order:

```go
// Expected cycling order:
// ProcessesView -> LogsView -> ErrorsView -> AICodersView -> SettingsView -> ProcessesView

result := validator.ValidateTabCycling(5)
assert.True(t, result.Success)
assert.Greater(t, result.Details["unique_views_visited"], 1)
```

### 3. Key Sequence Testing

Validates complex sequences of key presses:

```go
keySequence := []tea.KeyMsg{
    {Type: tea.KeyRunes, Runes: []rune{'2'}}, // Switch to Logs
    {Type: tea.KeyTab},                        // Cycle to next
    {Type: tea.KeyRunes, Runes: []rune{'4'}}, // Switch to AICoders
}

result := validator.ValidateKeySequence(keySequence, messages.AICodersView, "number_key")
assert.True(t, result.Success)
```

### 4. State Consistency Testing

Validates that all components maintain consistent state:

```go
result := validator.ValidateStateConsistency()
assert.True(t, result.Success)
```

### 5. Performance Testing

Validates that view switching meets performance requirements:

```go
// Rapid switching test
rapidKeys := 100
startTime := time.Now()

for i := 0; i < rapidKeys; i++ {
    key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('1' + (i % 5))}}
    adapter.SimulateKeyPress(key)
}

duration := time.Since(startTime)
avgTime := duration / time.Duration(rapidKeys)

assert.Less(t, avgTime, 10*time.Millisecond, "Average key time should be under 10ms")
```

## Integration with Existing Tests

### Updating Existing PTY Tests

The existing PTY view switching tests in `/test/tui/integration/test_view_switching_pty_test.go` have been updated to use the new integration:

```go
// Before: placeholder implementation
func (v *ViewSwitchingPTYTester) AllocatePTY(cols, rows int) error {
    return &NotImplementedError{"AllocatePTY"}
}

// After: real implementation with PTY-NavigationService integration
func (v *ViewSwitchingPTYTester) AllocatePTY(cols, rows int) error {
    // Create event bus, navigation service, PTY layer, and adapter
    // Returns nil on success
}
```

### New Integration Tests

New comprehensive integration tests have been added in `/test/tui/integration/test_pty_navigation_integration_test.go`:

- `TestPTYNavigationIntegration`: Full integration test
- `TestPTYNavigationAdapterIsolated`: Adapter isolation test
- `TestPTYNavigationStateTracker`: State tracker test
- `TestPTYNavigationIntegrationPerformance`: Performance test

## Statistics and Monitoring

### Adapter Statistics

```go
type PTYNavigationAdapterStats struct {
    Active            bool
    KeyPressCount     uint64
    ErrorCount        uint64
    SwitchCount       uint64
    TabPresses        uint64
    LastActivity      time.Time
    LastSwitchTime    time.Time
    ViewHistoryLength int
    NumberKeyPresses  map[int]int
    SwitchTriggers    map[string]int
}
```

### Validator Statistics

```go
type ValidatorStatistics struct {
    TotalValidations      int
    SuccessfulValidations int
    FailedValidations    int
    AverageValidationTime time.Duration
    MaxValidationTime     time.Duration
    MinValidationTime     time.Duration
    TestTypeCounts        map[ValidationTestType]int
    ErrorCounts           map[string]int
    LastValidation        time.Time
}
```

### State Tracker Statistics

```go
type NavigationStateStats struct {
    TotalTransitions    uint64
    SuccessfulTransitions uint64
    FailedTransitions   uint64
    AverageSwitchTime   time.Duration
    MaxSwitchTime       time.Duration
    MinSwitchTime       time.Duration
    ViewVisitCounts     map[messages.ViewType]uint64
    TriggerCounts       map[string]uint64
    ErrorCounts         map[string]uint64
    LastActivity        time.Time
    InitialState        messages.ViewType
    CurrentState        messages.ViewType
}
```

## Best Practices

### 1. Test Isolation

Ensure each test creates its own components to avoid interference:

```go
func TestViewSwitching(t *testing.T) {
    // Create fresh components for each test
    eventBus := events.NewEventBus()
    navigationService := services.NewNavigationService(eventBus)
    ptyLayer, _ := pty.NewPTYLayer(pty.DefaultPTYConfig())

    // Cleanup after test
    defer ptyLayer.Cleanup()
}
```

### 2. Timeout Management

Use appropriate timeouts for different test scenarios:

```go
// Quick operations: 100ms
adapter.WaitForViewSwitch(targetCount, 100*time.Millisecond)

// Complex sequences: 1s
validator.ValidateKeySequence(keys, expectedView, "test")

// Performance tests: 30s
validator.ValidateTabCycling(10)
```

### 3. Error Handling

Handle errors gracefully and provide useful debug information:

```go
result := validator.ValidateNumberKey(2, messages.LogsView)
if !result.Success {
    t.Logf("Validation failed: %s", result.Error)
    t.Logf("Details: %+v", result.Details)
}
```

### 4. Performance Considerations

- Use debug logging sparingly in production tests
- Limit history size for long-running tests
- Monitor memory usage for extended test sessions
- Use appropriate timeouts to avoid hanging tests

## Troubleshooting

### Common Issues

1. **PTY Allocation Fails**
   - Ensure PTY device is available
   - Check permissions on `/dev/ptmx`
   - Verify build tags include required constraints

2. **View Switch Not Detected**
   - Increase view switch timeout
   - Check adapter statistics
   - Verify NavigationService is running

3. **State Inconsistency**
   - Reset components between tests
   - Check event bus connectivity
   - Verify callback registration

4. **Performance Issues**
   - Reduce debug logging
   - Limit history size
   - Check for goroutine leaks

### Debug Information

Enable debug logging to troubleshoot issues:

```go
adapterConfig := services.DefaultPTYNavigationAdapterConfig()
adapterConfig.EnableDebug = true

validatorConfig := services.DefaultPTYViewSwitchingValidatorConfig()
validatorConfig.EnableDebug = true
validatorConfig.LogValidations = true
validatorConfig.LogValidationDetails = true
```

## Examples and Documentation

- **Basic Integration**: See `/internal/tui/services/pty_navigation_integration_example.go`
- **Comprehensive Tests**: See `/test/tui/integration/test_pty_navigation_integration_test.go`
- **Updated PTY Tests**: See `/test/tui/integration/test_view_switching_pty_test.go`

## Future Enhancements

Potential areas for future improvement:

1. **Enhanced Terminal Emulation**: More sophisticated terminal emulation for complex interactions
2. **Visual Testing**: Integration with visual regression testing frameworks
3. **Cross-Platform Support**: Enhanced support for different operating systems
4. **Performance Optimization**: Further optimizations for high-frequency testing
5. **Additional Validation**: More sophisticated validation scenarios and edge cases

## Conclusion

The PTY-NavigationService integration provides a robust foundation for TUI testing that enables:

- **Realistic Testing**: Simulates real user interactions through PTY
- **Production Integration**: Uses actual production NavigationService
- **Comprehensive Validation**: Validates all aspects of view switching
- **Automated Testing**: Enables CI/CD integration without TTY requirements

This integration significantly improves the reliability and comprehensiveness of TUI testing while maintaining compatibility with existing test infrastructure.