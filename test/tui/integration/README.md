# TUI Integration Tests - Keyboard Input PTY

This directory contains integration tests for keyboard input simulation through PTY (Pseudo-Terminal) for TUI testing.

## Overview

The keyboard input PTY tests validate that keyboard input works correctly through the PTY layer for TUI testing. These tests are designed to:

- Verify keyboard input simulation through PTY
- Test tea.KeyMsg generation via PTY
- Validate input timing and delays
- Test complex keyboard sequences
- Handle error cases for keyboard input failures

## Test Structure

### Main Test Files

- `test_keyboard_input_pty_test.go` - Comprehensive keyboard input PTY integration tests

### Key Test Categories

1. **Basic Simulation Tests** (`TestKeyboardInputPTY_Simulation`)
   - Single key input simulation
   - Multiple key input simulation
   - Special key input (Escape, Enter, Tab, Backspace)

2. **tea.KeyMsg Generation Tests** (`TestKeyboardInputPTY_TeaKeyMsgGeneration`)
   - Generate basic KeyMsg objects
   - Generate control KeyMsg objects
   - Test KeyMsg processing in TUI model

3. **Timing and Delay Tests** (`TestKeyboardInputPTY_InputTimingAndDelays`)
   - Single key timing
   - Key sequence with delays
   - Rapid input handling
   - Input debouncing

4. **Complex Sequence Tests** (`TestKeyboardInputPTY_ComplexSequences`)
   - Command input sequences
   - Navigation sequences
   - Text editing sequences
   - Multi-key combinations
   - Escape sequence handling

5. **Error Handling Tests** (`TestKeyboardInputPTY_ErrorHandling`)
   - Operations on unallocated PTY
   - Invalid key sequences
   - Context cancellation
   - Resource exhaustion

6. **Implementation Status Tests** (`TestKeyboardInputPTY_ImplementationRequired`)
   - Verifies that PTY layer is not yet implemented
   - Designed to fail until implementation is provided

7. **Integration Tests** (`TestKeyboardInputPTY_IntegrationWithBrummer`)
   - Tests with actual Brummer binary
   - Basic TUI navigation
   - Command input
   - Output validation

## Running the Tests

### Prerequisites

1. Build the Brummer binary:
   ```bash
   make build
   ```

2. Set the binary environment variable (optional):
   ```bash
   export BRUMMER_BINARY=./brum
   ```

### Running Tests

#### Run All Keyboard Input PTY Tests
```bash
BRUMMER_BINARY=./brum go test -tags=integration,tty -v ./test/tui/integration/
```

#### Run Specific Test Categories
```bash
# Run basic simulation tests
BRUMMER_BINARY=./brum go test -tags=integration,tty -run TestKeyboardInputPTY_Simulation -v ./test/tui/integration/

# Run timing tests
BRUMMER_BINARY=./brum go test -tags=integration,tty -run TestKeyboardInputPTY_InputTimingAndDelays -v ./test/tui/integration/

# Run implementation status check
BRUMMER_BINARY=./brum go test -tags=integration,tty -run TestKeyboardInputPTY_ImplementationRequired -v ./test/tui/integration/
```

#### Run in Short Mode (Skip Integration Tests)
```bash
go test -short -tags=integration,tty -run TestKeyboardInputPTY -v ./test/tui/integration/
```

## Current Status

**🚨 IMPORTANT**: These tests are currently designed to **FAIL** because the PTY keyboard input layer is not yet implemented.

### Expected Behavior

- Tests should pass because they expect `NotImplementedError` exceptions
- When PTY layer is implemented, tests should be updated to work with real implementation
- The `TestKeyboardInputPTY_ImplementationRequired` test specifically verifies that implementation is missing

### Test Output Example

```
⌨️  TESTING: Keyboard Input Simulation through PTY
=============================================
Test 1: PTY Allocation for Keyboard Input
❌ PTY allocation failed (expected): method not implemented: AllocatePTY
   This will fail until PTY keyboard input layer is implemented
```

## Implementation Requirements

When implementing the PTY keyboard input layer, these methods need to be implemented:

1. `AllocatePTY(cols, rows int) error` - Allocate PTY with specified dimensions
2. `SimulateKeyPress(key string) error` - Simulate single key press
3. `SimulateKeySequence(keys []string, delay time.Duration) error` - Simulate key sequence with delays
4. `GenerateKeyMsg(key string) (tea.KeyMsg, error)` - Generate tea.KeyMsg from key input
5. `GetOutput() []byte` - Get captured output from PTY
6. `Cleanup()` - Clean up PTY resources

## Error Types

The tests use custom error types for specific failure scenarios:

- `NotImplementedError` - Method not yet implemented
- `InvalidInputError` - Invalid input parameters
- `NotAllocatedError` - Operation on unallocated PTY

## Test Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/hinshun/vt10x` - Terminal emulator
- `github.com/stretchr/testify` - Testing assertions
- `github.com/standardbeagle/brummer/internal/*` - Brummer internal packages

## Troubleshooting

### Test Skips

- Tests skip if `BRUMMER_BINARY` is not set and `./brum` doesn't exist
- Tests skip in short mode with `-short` flag
- Tests skip if PTY allocation is not implemented

### Build Failures

- Ensure all dependencies are installed: `go mod tidy`
- Ensure build tags are correct: `go test -tags=integration,tty`
- Check for import errors or type mismatches

### Test Failures

- Check if Brummer binary is built and accessible
- Verify TTY support on the test system
- Check for proper test setup and cleanup

## Future Improvements

When PTY layer is implemented:

1. Update tests to use real PTY functionality
2. Add performance benchmarks for keyboard input
3. Add tests for Unicode/UTF-8 character input
4. Add tests for international keyboard layouts
5. Add tests for accessibility features
6. Add visual regression tests for TUI responses