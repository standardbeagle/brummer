# PTY Integration Guide

This guide covers the PTY (Pseudo-Terminal) integration for BubbleTea applications in Brummer, enabling realistic terminal simulation without requiring `/dev/tty`.

## Overview

The PTY integration system provides:

- **Terminal Emulation**: Full terminal capabilities without needing a real TTY
- **Automatic Detection**: Intelligently switches between PTY and standard TTY modes
- **Size Control**: Dynamic terminal resizing and cursor management
- **Graceful Termination**: Proper cleanup and signal handling
- **Performance Monitoring**: Built-in statistics and debugging tools

## Architecture

The PTY integration consists of several key components:

### Core Components

1. **PTYIntegration** (`/internal/tui/pty_integration.go`)
   - Main PTY wrapper for BubbleTea programs
   - Manages PTY allocation and program lifecycle
   - Provides input/output simulation

2. **PTYTerminalEmulator** (`/internal/tui/pty_terminal_emulator.go`)
   - Advanced terminal emulation with ANSI escape sequence support
   - Scrollback buffer and cursor tracking
   - Performance monitoring and history management

3. **PTYTerminationHandler** (`/internal/tui/pty_termination.go`)
   - Comprehensive shutdown and cleanup management
   - Signal handling with graceful and forced termination
   - Resource tracking and error recovery

4. **PTYMainWrapper** (`/internal/tui/pty_main_wrapper.go`)
   - High-level wrapper that coordinates all PTY components
   - Automatic TTY/PTY detection and fallback
   - Configuration and performance monitoring

5. **PTY Layer** (`/test/tui/pty/pty_layer.go`)
   - Low-level PTY allocation and I/O management
   - Cross-platform compatibility
   - Terminal size control

## Usage

### Basic Usage

The PTY integration automatically detects when PTY should be used:

```bash
# Auto-detect: will use PTY if no TTY is available
brum

# Force PTY usage
brum --use-pty

# Disable PTY (force standard TTY)
brum --no-pty
```

### Configuration

PTY integration can be configured through the main wrapper:

```go
config := tui.DefaultPTYMainWrapperConfig()

// Enable debugging
config.EnableDebug = true
config.LogStartupSteps = true
config.LogPerformance = true

// Terminal settings
config.PTYIntegration.InitialCols = 120
config.PTYIntegration.InitialRows = 30

// Performance settings
config.StartupTimeout = 30 * time.Second
config.ShutdownTimeout = 10 * time.Second
```

### Programmatic Usage

```go
// Create a model
model := myBubbleTeaModel()

// Use auto-detection
result := tui.AutoRunWithPTY(model, tea.WithAltScreen())

// Use custom configuration
config := tui.DefaultPTYMainWrapperConfig()
result := tui.RunWithPTYAndOptions(model, config, tea.WithAltScreen())

// Check results
if result.Error != nil {
    log.Printf("PTY program failed: %v", result.Error)
} else {
    log.Printf("Program ran for %v (PTY: %v)",
        result.Duration, result.WasPTYUsed)
}
```

## Advanced Features

### Terminal Emulation

The terminal emulator provides full ANSI escape sequence support:

```go
// Get terminal emulator from wrapper
emulator := wrapper.GetTerminalEmulator()

// Process ANSI input
input := []byte("\x1b[31mHello\x1b[0m\n") // Red "Hello"
err := emulator.ProcessInput(input)

// Get terminal state
state := emulator.GetTerminalState()
cursor := emulator.GetCursorPosition()

// Get performance stats
stats := emulator.GetPerformanceStats()
```

### Signal Handling

The termination handler manages shutdown processes:

```go
// Get termination handler
handler := wrapper.GetTerminationHandler()

// Add custom callbacks
handler.AddTerminationCallback("cleanup", 50, func(reason tui.ShutdownReason) error {
    // Custom cleanup logic
    return nil
})

// Track resources for cleanup
handler.TrackResource("database", dbConn, func() error {
    return dbConn.Close()
})

// Manual shutdown
err := handler.Shutdown(tui.ReasonUserRequest)
```

### Input Simulation

Direct keyboard input simulation through PTY:

```go
// Get PTY integration
ptyInt := wrapper.GetPTYIntegration()

// Simulate key presses
err := ptyInt.SimulateInput("hello world")

// Simulate specific keys
err := ptyInt.SimulateKeyPress(tea.KeyMsg{
    Type:  tea.KeyEnter,
    Runes: []rune{},
})

// Get captured output
output := ptyInt.GetOutput()
```

## Testing

### Integration Tests

The PTY integration includes comprehensive test coverage:

```bash
# Run PTY integration tests
make test-pty

# Run specific test suites
go test -tags=tty ./test/tui/pty/...

# Run with debugging
BRUM_DEBUG=1 go test -tags=tty ./test/tui/integration/...
```

### Manual Testing

Test PTY functionality manually:

```bash
# Test in non-TTY environment
brum --use-pty 2>/dev/null &
PID=$!

# Test input simulation
echo "Sending input to PID $PID"
# Use integration tools to send input

# Check results
kill $PID
```

## Configuration Options

### PTY Integration Configuration

```go
type PTYIntegrationConfig struct {
    InitialCols        uint16        // Terminal width (default: 80)
    InitialRows        uint16        // Terminal height (default: 24)
    AllocationTimeout   time.Duration // PTY allocation timeout (default: 10s)
    InputDelay         time.Duration // Delay between keystrokes (default: 50ms)
    ShutdownTimeout    time.Duration // Shutdown timeout (default: 5s)
    EnableRenderer     bool          // Enable output renderer (default: true)
    EnableInputController bool       // Enable input simulation (default: true)
    EnableBridge       bool          // Enable message bridge (default: true)
    EnableDebug        bool          // Enable debug logging (default: false)
}
```

### Terminal Emulator Configuration

```go
type TerminalEmulatorConfig struct {
    Rows                int           // Terminal rows (default: 24)
    Cols                int           // Terminal columns (default: 80)
    MaxScrollbackLines   int           // Scrollback buffer size (default: 1000)
    EnableScrollback     bool          // Enable scrollback (default: true)
    DefaultCursorShape  CursorShape  // Default cursor appearance (default: Block)
    CursorBlinkInterval time.Duration // Cursor blink rate (default: 500ms)
    ShowCursor          bool          // Show cursor (default: true)
    EnableANSIColors    bool          // Enable color support (default: true)
    EnableUTF8          bool          // Enable UTF-8 support (default: true)
    TabSize            int           // Tab size in spaces (default: 4)
}
```

### Termination Configuration

```go
type TerminationConfig struct {
    GracefulShutdownTimeout time.Duration // Graceful shutdown timeout (default: 10s)
    ForcedShutdownTimeout   time.Duration // Forced shutdown timeout (default: 3s)
    ResourceCleanupTimeout  time.Duration // Resource cleanup timeout (default: 5s)
    HandleSignals          bool          // Handle system signals (default: true)
    EnableGracefulShutdown bool          // Enable graceful shutdown (default: true)
    EnableForcedShutdown   bool          // Enable forced shutdown (default: true)
    TrackResources         bool          // Track resources for cleanup (default: true)
    RetryOnError           bool          // Retry on errors (default: true)
    MaxRetryAttempts       int           // Maximum retry attempts (default: 3)
    MaxErrorHistory        int           // Error history size (default: 50)
}
```

## Debugging

### Enable Debug Mode

```bash
# Enable PTY debugging
brum --debug --use-pty

# Enable MCP debugging with PTY
brum --mcp-debug --use-pty

# Enable comprehensive debugging
BRUM_DEBUG=1 MCP_DEBUG=1 brum --use-pty
```

### Performance Monitoring

Performance statistics are automatically collected:

```go
result := wrapper.Run()

// Access performance stats
stats := result.PerformanceStats
fmt.Printf("Performance: %+v\n", stats)

// Terminal emulator stats
if emulator := wrapper.GetTerminalEmulator() {
    terminalStats := emulator.GetPerformanceStats()
    fmt.Printf("Terminal: %+v\n", terminalStats)
}

// Termination handler stats
if handler := wrapper.GetTerminationHandler() {
    terminationStats := handler.GetShutdownStats()
    fmt.Printf("Termination: %+v\n", terminationStats)
}
```

### Error Handling

Comprehensive error tracking and reporting:

```go
// Get last error
if err := wrapper.GetLastError(); err != nil {
    log.Printf("PTY Error: %v", err)
}

// Get error history
handler := wrapper.GetTerminationHandler()
errors := handler.GetErrorHistory()
for _, err := range errors {
    log.Printf("Error at %v: %s (%s)",
        err.Timestamp, err.Error, err.Context)
}
```

## Troubleshooting

### Common Issues

#### PTY Allocation Fails
```
Error: failed to allocate PTY: operation not permitted
```

**Solution**: Ensure proper permissions or run with `--no-pty`

#### Terminal Size Issues
```
Error: invalid terminal size: 0x0
```

**Solution**: Set minimum terminal size in configuration

#### Signal Handling
```
Warning: signal handler error for SIGINT: operation not permitted
```

**Solution**: Use `--no-pty` in restricted environments

### Platform-Specific Notes

#### Windows
PTY support is limited on Windows. The system will:
- Auto-disable PTY on Windows by default
- Fall back to standard BubbleTea mode
- Provide appropriate warnings

#### WSL2/WSL1
- PTY detection works automatically
- No special configuration needed
- Standard TTY detection may be unreliable

#### Docker/Containers
- Use `--use-pty` when TTY is not available
- Ensure proper container permissions
- May need `--privileged` flag for PTY allocation

## Migration Guide

### From Standard BubbleTea

**Before:**
```go
program := tea.NewProgram(model, tea.WithAltScreen())
finalModel := program.Run()
```

**After (Auto-detection):**
```go
result := tui.AutoRunWithPTY(model, tea.WithAltScreen())
finalModel := result.Model
```

**After (Explicit PTY):**
```go
config := tui.DefaultPTYMainWrapperConfig()
result := tui.RunWithPTYAndOptions(model, config, tea.WithAltScreen())
finalModel := result.Model
```

### From Custom TUI

**Before:**
```go
func runTUI() {
    program := tea.NewProgram(model, tea.WithAltScreen())
    program.Run()
}
```

**After:**
```go
func runTUI() {
    config := tui.DefaultPTYMainWrapperConfig()
    result := tui.RunWithPTYAndOptions(model, config, tea.WithAltScreen())
    if result.Error != nil {
        log.Printf("TUI failed: %v", result.Error)
    }
}
```

## Examples

### Example 1: Basic PTY Application

```go
package main

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/tui"
)

type model struct{}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, nil
}

func (m model) View() string {
    return "Hello PTY World!\n"
}

func main() {
    model := model{}

    // Auto-detect and run with PTY support
    result := tui.AutoRunWithPTY(model, tea.WithAltScreen())

    if result.Error != nil {
        fmt.Printf("Error: %v\n", result.Error)
        return
    }

    fmt.Printf("Ran for %v (PTY: %v)\n",
        result.Duration, result.WasPTYUsed)
}
```

### Example 2: Custom PTY Configuration

```go
package main

import (
    "log"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/tui"
)

type model struct{}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m model) View() string { return "Custom PTY\n" }

func main() {
    // Create custom configuration
    config := tui.DefaultPTYMainWrapperConfig()
    config.PTYIntegration.InitialCols = 100
    config.PTYIntegration.InitialRows = 30
    config.EnableDebug = true
    config.LogStartupSteps = true
    config.EnableTerminalEmulation = true

    // Add custom startup callback
    wrapper := tui.NewPTYMainWrapper(config)
    wrapper.AddStartupCallback(func() error {
        log.Println("Custom PTY startup")
        return nil
    })

    wrapper.AddShutdownCallback(func() error {
        log.Println("Custom PTY shutdown")
        return nil
    })

    // Set model and options
    if err := wrapper.SetModel(model{}); err != nil {
        log.Fatalf("Failed to set model: %v", err)
    }

    if err := wrapper.SetProgramOptions(tea.WithAltScreen()); err != nil {
        log.Fatalf("Failed to set options: %v", err)
    }

    // Run with custom configuration
    result := wrapper.Run()

    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
    }

    log.Printf("Result: %+v", result)
}
```

### Example 3: Advanced PTY with Terminal Emulation

```go
package main

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/tui"
)

type model struct {
    emulator *tui.PTYTerminalEmulator
    wrapper  *tui.PTYMainWrapper
}

func (m model) Init() tea.Cmd {
    // Start terminal emulator if available
    if m.emulator != nil {
        // Monitor terminal events
        m.emulator.AddEventCallback(tui.EventContentChange, func(event tui.TerminalEvent) {
            fmt.Printf("Terminal content changed: %+v\n", event)
        })
    }

    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, nil
}

func (m model) View() string {
    if m.emulator != nil {
        return m.emulator.Render()
    }
    return "No terminal emulator available\n"
}

func main() {
    config := tui.DefaultPTYMainWrapperConfig()
    config.EnableTerminalEmulation = true

    wrapper := tui.NewPTYMainWrapper(config)

    // Set model
    if err := wrapper.SetProgramOptions(tea.WithAltScreen()); err != nil {
        fmt.Printf("Failed to set options: %v\n", err)
        return
    }

    // Initialize first to get components
    if err := wrapper.Initialize(); err != nil {
        fmt.Printf("Failed to initialize: %v\n", err)
        return
    }

    // Get terminal emulator
    emulator := wrapper.GetTerminalEmulator()

    // Create model with emulator reference
    model := model{
        emulator: emulator,
        wrapper:  wrapper,
    }

    if err := wrapper.SetModel(model); err != nil {
        fmt.Printf("Failed to set model: %v\n", err)
        return
    }

    // Run with terminal emulation
    result := wrapper.Run()

    if result.Error != nil {
        fmt.Printf("Error: %v\n", result.Error)
        return
    }

    fmt.Printf("Terminal emulation: %+v\n", result.PerformanceStats)
}
```

## Integration with Existing Components

### AI Coder Integration

The PTY integration works seamlessly with AI Coder features:

```go
// AI Coder sessions automatically use PTY when available
// No configuration needed - automatic detection handles it

// The AI coder component will receive proper terminal I/O through PTY
// when running in non-TTY environments
```

### MCP Server Integration

PTY integration doesn't interfere with MCP server functionality:

```go
// MCP server runs independently of PTY
// PTY is only used for the TUI interface
// All MCP operations continue to work normally
```

### Process Management

Process management integrates with PTY for terminal I/O:

```go
// Process logs are captured through PTY when available
// Terminal output from managed processes is properly displayed
// Keyboard input to control processes works through PTY simulation
```

## Best Practices

### Performance Considerations

1. **Buffer Management**: Monitor scrollback buffer size
2. **Input Delay**: Adjust input delay for realistic typing simulation
3. **Resource Cleanup**: Always properly shutdown PTY resources
4. **Memory Usage**: Monitor memory usage with large output

### Security Considerations

1. **Input Validation**: Validate all simulated input
2. **Resource Limits**: Set appropriate buffer limits
3. **Cleanup**: Ensure proper resource cleanup on exit
4. **Permissions**: Handle PTY allocation permissions gracefully

### Testing Considerations

1. **Environment Detection**: Test both PTY and TTY modes
2. **Integration Tests**: Use PTY integration for comprehensive testing
3. **Performance Tests**: Monitor performance impact
4. **Error Scenarios**: Test error handling and recovery

## API Reference

### PTYMainWrapper

#### Constructor
```go
func NewPTYMainWrapper(config PTYMainWrapperConfig) *PTYMainWrapper
```

#### Methods
```go
func (wrapper *PTYMainWrapper) SetModel(model tea.Model) error
func (wrapper *PTYMainWrapper) SetProgramOptions(options ...tea.ProgramOption) error
func (wrapper *PTYMainWrapper) Run() PTYMainWrapperResult
func (wrapper *PTYMainWrapper) ShouldUsePTY() bool
func (wrapper *PTYMainWrapper) Initialize() error
func (wrapper *PTYMainWrapper) Shutdown() error
func (wrapper *PTYMainWrapper) Cleanup() error
```

#### Static Methods
```go
func RunWithPTY(model tea.Model, config PTYMainWrapperConfig) PTYMainWrapperResult
func RunWithPTYAndOptions(model tea.Model, config PTYMainWrapperConfig, options ...tea.ProgramOption) PTYMainWrapperResult
func AutoRunWithPTY(model tea.Model, options ...tea.ProgramOption) PTYMainWrapperResult
```

### PTYIntegration

#### Constructor
```go
func NewPTYIntegration(config PTYIntegrationConfig) (*PTYIntegration, error)
```

#### Methods
```go
func (pi *PTYIntegration) Allocate() error
func (pi *PTYIntegration) SetModel(model tea.Model) error
func (pi *PTYIntegration) SetProgramOptions(options ...tea.ProgramOption) error
func (pi *PTYIntegration) Start() error
func (pi *PTYIntegration) Run() (tea.Model, error)
func (pi *PTYIntegration) Stop() error
func (pi *PTYIntegration) IsRunning() bool
func (pi *PTYIntegration) GetModel() tea.Model
func (pi *PTYIntegration) Resize(cols, rows uint16) error
func (pi *PTYIntegration) SimulateInput(input string) error
func (pi *PTYIntegration) SimulateKeyPress(key tea.KeyMsg) error
func (pi *PTYIntegration) GetOutput() string
func (pi *PTYIntegration) ClearBuffers() error
func (pi *PTYIntegration) Cleanup() error
```

### PTYTerminalEmulator

#### Constructor
```go
func NewPTYTerminalEmulator(ptyLayer *pty.PTYLayer, config TerminalEmulatorConfig) *PTYTerminalEmulator
```

#### Methods
```go
func (te *PTYTerminalEmulator) ProcessInput(input []byte) error
func (te *PTYTerminalEmulator) Resize(rows, cols int) error
func (te *PTYTerminalEmulator) Render() string
func (te *PTYTerminalEmulator) GetCursorPosition() CursorPosition
func (te *PTYTerminalEmulator) GetTerminalState() TerminalState
func (te *PTYTerminalEmulator) GetScrollbackBuffer() []string
func (te *PTYTerminalEmulator) GetInputHistory() []string
func (te *PTYTerminalEmulator) GetOutputHistory() []string
func (te *PTYTerminalEmulator) GetPerformanceStats() map[string]interface{}
func (te *PTYTerminalEmulator) AddEventCallback(eventType EventType, callback func(TerminalEvent))
func (te *PTYTerminalEmulator) ClearScreen()
func (te *PTYTerminalEmulator) Reset()
```

### PTYTerminationHandler

#### Constructor
```go
func NewPTYTerminationHandler(ptyIntegration *PTYIntegration, config TerminationConfig) *PTYTerminationHandler
```

#### Methods
```go
func (th *PTYTerminationHandler) SetProgram(program *tea.Program)
func (th *PTYTerminationHandler) Start() error
func (th *PTYTerminationHandler) Shutdown(reason ShutdownReason) error
func (th *PTYTerminationHandler) ForceShutdown(reason ShutdownReason) error
func (th *PTYTerminationHandler) GetState() TerminationState
func (th *PTYTerminationHandler) GetShutdownReason() ShutdownReason
func (th *PTYTerminationHandler) GetLastError() error
func (th *PTYTerminationHandler) GetErrorHistory() []TerminationError
func (th *PTYTerminationHandler) GetTrackedResources() []TrackedResource
func (th *PTYTerminationHandler) AddTerminationCallback(name string, priority int, handler func(ShutdownReason) error)
func (th *PTYTerminationHandler) AddCleanupCallback(name string, priority int, handler func() error)
func (th *PTYTerminationHandler) AddSignalHandler(sig syscall.Signal, handler SignalHandler)
func (th *PTYTerminationHandler) TrackResource(name string, resource interface{}, cleanup func() error)
func (th *PTYTerminationHandler) UntrackResource(name string)
func (th *PTYTerminationHandler) IsShuttingDown() bool
func (th *PTYTerminationHandler) IsTerminated() bool
```

## Conclusion

The PTY integration system provides comprehensive terminal emulation capabilities for BubbleTea applications in Brummer. It enables:

- **Seamless Operation**: Automatic detection between PTY and TTY modes
- **Full Terminal Features**: ANSI escape sequences, cursor control, scrollback
- **Robust Error Handling**: Graceful degradation and comprehensive cleanup
- **Performance Monitoring**: Built-in statistics and debugging tools
- **Easy Integration**: Simple API that works with existing BubbleTea code

The system is designed to work transparently in most scenarios while providing extensive configuration options for advanced use cases.