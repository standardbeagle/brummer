# PTY Input Integration for Brummer TUI

This document describes the PTY (pseudo-terminal) input integration system that enables realistic keyboard input simulation for TUI testing environments.

## Overview

The PTY input integration connects the PTY layer with the existing InputService to enable:

- **Realistic keyboard input simulation** through actual PTY layer
- **Complete InputService integration** maintaining all production functionality
- **Comprehensive validation** of keyboard routing paths
- **Backward compatibility** with existing InputService usage
- **Testing environment support** for TUI applications without requiring /dev/tty

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Test Harness  │───▶│  PTYInputAdapter │───▶│   PTY Layer     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │  InputService    │◀───│  Keyboard Input │
                       └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   TUI Components │
                       └──────────────────┘
```

## Components

### 1. PTYInputAdapter (`pty_input_adapter.go`)

 Bridges PTY layer with InputService for realistic input simulation.

**Key Features:**
- Connects PTY layer to InputService
- Handles concurrent input processing
- Provides statistics and monitoring
- Supports configurable timing and buffering

**Usage:**
```go
// Create PTY layer
ptyLayer, _ := pty.NewPTYLayer(pty.DefaultPTYConfig())
ptyLayer.Allocate()

// Create InputService with PTY support
inputService := NewInputServiceWithPTY(ptyLayer)
inputService.Start(ctx)

// Create and start adapter
adapter, _ := NewPTYInputAdapter(ptyLayer, inputService, DefaultPTYInputAdapterConfig())
adapter.Start()
defer adapter.Stop()

// Simulate keyboard input
key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
adapter.SimulateKeyPress(key)
```

### 2. Enhanced InputService (`input_service.go`)

 Extended InputService with PTY support while maintaining full backward compatibility.

**New PTY Features:**
- `NewInputServiceWithPTY(ptyLayer)` - Constructor with PTY support
- `EnablePTY(ptyLayer)` - Enable PTY mode dynamically
- `DisablePTY()` - Disable PTY mode
- `RouteKeyMessagePTY(key, context)` - PTY-aware routing
- `SimulatePTYKeyPress(key)` - Simulate through PTY layer
- `ValidatePTYInput(keys)` - Validate PTY input path
- `GetPTYStatistics()` - PTY usage statistics

**Backward Compatibility:**
- All existing methods work exactly as before
- PTY features are optional and disabled by default
- No breaking changes to existing API

**Usage:**
```go
// Traditional usage (unchanged)
inputService := NewInputService()
inputService.Start(ctx)

// PTY-enabled usage
inputService := NewInputServiceWithPTY(ptyLayer)
inputService.Start(ctx)

// Enable PTY dynamically
inputService.EnablePTY(ptyLayer)

// Route keys through PTY path
handled := inputService.RouteKeyMessagePTY(key, "test_context")
```

### 3. KeyboardRoutingValidator (`keyboard_routing_validation.go`)

 Comprehensive validation system for keyboard routing through PTY->InputService path.

**Validation Features:**
- Tests complete PTY->InputService routing
- Validates timing and performance
- Checks event flow
- Comprehensive test case coverage
- Statistics and reporting

**Usage:**
```go
// Create validator
validator, _ := NewKeyboardRoutingValidator(ptyLayer, inputService, DefaultKeyboardRoutingValidatorConfig())
validator.Start()
defer validator.Stop()

// Validate PTY->InputService path
err := validator.ValidatePTYToInputServicePath()

// Run comprehensive validation
results, err := validator.ValidateKeyboardRouting()

// Get statistics
stats := validator.GetValidationStatistics()
```

## Configuration

### PTYInputAdapterConfig

```go
type PTYInputAdapterConfig struct {
    EventBufferSize   int           // Buffer size for key events (default: 1000)
    ErrorBufferSize   int           // Buffer size for errors (default: 100)
    ReadTimeout       time.Duration // PTY read timeout (default: 100ms)
    ShutdownTimeout   time.Duration // Shutdown timeout (default: 5s)
    ActivityTimeout   time.Duration // Inactivity timeout (default: 30s)
    EnableStatistics  bool          // Enable statistics collection (default: true)
    EnableDebug       bool          // Enable debug logging (default: false)
    BufferInput       bool          // Buffer input events (default: true)
    ForwardUnknownKeys bool         // Forward unhandled keys to PTY (default: false)
    MonitorPTYOutput  bool          // Monitor PTY output for input (default: false)
    PTYReadInterval   time.Duration // PTY read polling interval (default: 10ms)
}
```

### KeyboardRoutingValidatorConfig

```go
type KeyboardRoutingValidatorConfig struct {
    TestTimeout         time.Duration // Test case timeout (default: 30s)
    MaxTestCases       int           // Maximum test cases (default: 100)
    EnableStatistics    bool          // Enable statistics (default: true)
    EnableDebug         bool          // Enable debug logging (default: false)
    StrictValidation    bool          // Enable strict validation (default: true)
    RequirePTYEnabled   bool          // Require PTY to be enabled (default: true)
    CheckEventFlow      bool          // Check event flow (default: true)
    ValidateTiming      bool          // Validate timing requirements (default: true)
    MaxKeyProcessingTime time.Duration // Max processing time per key (default: 100ms)
    MaxEventLatency     time.Duration // Max event latency (default: 50ms)
}
```

## Usage Examples

### Basic PTY Integration

```go
func BasicPTYIntegration() error {
    // Create PTY layer
    ptyConfig := pty.DefaultPTYConfig()
    ptyLayer, err := pty.NewPTYLayer(ptyConfig)
    if err != nil {
        return err
    }
    defer ptyLayer.Cleanup()

    // Allocate PTY
    if err := ptyLayer.Allocate(); err != nil {
        return err
    }

    // Create InputService with PTY support
    inputService := NewInputServiceWithPTY(ptyLayer)
    if err := inputService.Start(ctx); err != nil {
        return err
    }
    defer inputService.Shutdown(ctx)

    // Simulate keyboard input
    testKeys := CreateTestKeySequence()
    for _, key := range testKeys {
        if err := inputService.SimulatePTYKeyPress(key); err != nil {
            return err
        }
    }

    // Verify statistics
    stats := inputService.GetPTYStatistics()
    fmt.Printf("Processed %d PTY keys\n", stats.PTYKeyCount)

    return nil
}
```

### Advanced Testing with Validation

```go
func AdvancedPTYTesting() error {
    // Setup components
    ptyLayer, _ := pty.NewPTYLayer(pty.DefaultPTYConfig())
    ptyLayer.Allocate()
    defer ptyLayer.Cleanup()

    inputService := NewInputServiceWithPTY(ptyLayer)
    inputService.Start(ctx)
    defer inputService.Shutdown(ctx)

    // Register test handlers
    inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
        // Handle global keys
        return nil
    })

    inputService.SubscribeWithContextKeys(ctx, "test", func(keyMsg tea.KeyMsg) tea.Cmd {
        // Handle context-specific keys
        return nil
    })

    // Create validator
    validator, _ := NewKeyboardRoutingValidator(ptyLayer, inputService, DefaultKeyboardRoutingValidatorConfig())
    validator.Start()
    defer validator.Stop()

    // Run validation
    results, err := validator.ValidateKeyboardRouting()
    if err != nil {
        return err
    }

    // Analyze results
    passed := 0
    failed := 0
    for _, result := range results {
        if result.Success {
            passed++
        } else {
            failed++
            fmt.Printf("Test failed: %s - %v\n", result.TestCase.Name, result.Error)
        }
    }

    fmt.Printf("Validation completed: %d passed, %d failed\n", passed, failed)
    return nil
}
```

### Integration with Test Harness

```go
func TestHarnessIntegration() error {
    // Create PTY layer for testing
    ptyLayer, _ := pty.NewPTYLayer(pty.DefaultPTYConfig())
    ptyLayer.Allocate()
    defer ptyLayer.Cleanup()

    // Create InputService
    inputService := NewInputServiceWithPTY(ptyLayer)
    inputService.Start(ctx)
    defer inputService.Shutdown(ctx)

    // Create PTY adapter
    adapter, _ := NewPTYInputAdapter(ptyLayer, inputService, DefaultPTYInputAdapterConfig())
    adapter.Start()
    defer adapter.Stop()

    // Simulate user interaction sequence
    interactions := []string{
        "help",        // Press 'h' for help
        "navigate:up", // Press up arrow
        "navigate:down", // Press down arrow
        "quit",        // Press 'q' to quit
    }

    for _, interaction := range interactions {
        keys, err := parseInteraction(interaction)
        if err != nil {
            return err
        }

        for _, key := range keys {
            if err := adapter.SimulateKeyPress(key); err != nil {
                return err
            }
            time.Sleep(50 * time.Millisecond) // Simulate realistic typing
        }
    }

    // Verify results
    stats := inputService.GetPTYStatistics()
    fmt.Printf("Simulated %d interactions, processed %d keys\n",
        len(interactions), stats.PTYKeyCount)

    return nil
}
```

## Testing

The PTY integration includes comprehensive tests:

1. **Unit Tests**: Test individual components in isolation
2. **Integration Tests**: Test component interactions
3. **Validation Tests**: Test keyboard routing validation
4. **Backward Compatibility Tests**: Ensure existing functionality works
5. **Concurrency Tests**: Test thread-safe operation
6. **Performance Tests**: Validate timing requirements

### Running Tests

```bash
# Run all PTY integration tests
go test -v ./internal/tui/services -run PTY

# Run specific test categories
go test -v ./internal/tui/services -run TestPTYInputIntegration
go test -v ./internal/tui/services -run TestKeyboardRoutingValidation
go test -v ./internal/tui/services -run TestInputServiceBackwardCompatibility

# Run with race detection
go test -race -v ./internal/tui/services -run PTY
```

## Migration Guide

### For Existing Code

Existing InputService usage continues to work without changes:

```go
// This continues to work exactly as before
inputService := NewInputService()
inputService.Start(ctx)

// All existing methods work unchanged
inputService.SubscribeToGlobalKeys(ctx, handler)
inputService.RouteKeyMessage(key, context)
```

### To Enable PTY Support

```go
// Option 1: Use PTY-enabled constructor
inputService := NewInputServiceWithPTY(ptyLayer)

// Option 2: Enable PTY dynamically
inputService := NewInputService()
inputService.EnablePTY(ptyLayer)

// Option 3: Use PTY-aware routing
handled := inputService.RouteKeyMessagePTY(key, "test_context")
```

## Performance Considerations

- **PTY overhead**: Minimal impact on existing code
- **Memory usage**: Small additional footprint for PTY integration
- **Latency**: PTY routing adds ~1-5ms overhead per key
- **Concurrency**: Uses lock-free patterns for optimal performance

## Troubleshooting

### Common Issues

1. **"PTY layer must be allocated"**
   - Ensure `ptyLayer.Allocate()` is called before using with InputService

2. **"PTY mode is not enabled"**
   - Call `inputService.EnablePTY(ptyLayer)` or use `NewInputServiceWithPTY()`

3. **High latency in key processing**
   - Check PTY layer configuration
   - Reduce `PTYReadInterval` in adapter config
   - Ensure handlers are not blocking

4. **Memory leaks in tests**
   - Always call `adapter.Stop()` and `inputService.Shutdown()`
   - Use `defer` statements for cleanup

### Debug Mode

Enable debug logging for troubleshooting:

```go
// Enable debug in PTY layer
ptyConfig := pty.DefaultPTYConfig()
ptyConfig.EnableDebug = true

// Enable debug in adapter
adapterConfig := DefaultPTYInputAdapterConfig()
adapterConfig.EnableDebug = true

// Enable debug in validator
validatorConfig := DefaultKeyboardRoutingValidatorConfig()
validatorConfig.EnableDebug = true
```

## Future Enhancements

- **Enhanced terminal emulation**: Full VT100/ANSI escape sequence support
- **Multi-PTY support**: Multiple PTY instances for complex testing
- **Recording/Playback**: Record and replay user interactions
- **Performance profiling**: Built-in performance analysis tools
- **Visual debugging**: GUI tools for PTY interaction visualization