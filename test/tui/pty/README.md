# PTY Layer Integration Tests

This package provides integration tests for the PTY (pseudo-terminal) layer functionality that enables TUI testing without TTY allocation.

## Overview

The PTY layer integration tests validate the core PTY functionality that would enable automated TUI testing without requiring actual TTY allocation. This follows the same pattern as the existing `/test/tty/` package but focuses specifically on the PTY layer abstractions.

## Test Coverage

### Primary Test Functions

1. **TestPTYLayer_AllocationAndManagement**
   - Tests PTY allocation with specified dimensions
   - Validates PTY size management (set/get operations)
   - Checks PTY state validation
   - Verifies PTY file descriptor management

2. **TestPTYLayer_TTYBridgeFunctionality**
   - Tests PTY-to-TTY bridge initialization
   - Validates terminal emulator integration
   - Tests input/output bridge functionality
   - Verifies control sequence handling

3. **TestPTYLayer_Cleanup**
   - Tests normal PTY cleanup procedures
   - Validates multiple cleanup calls (should be safe)
   - Tests context cancellation handling
   - Verifies resource management

4. **TestPTYLayer_ErrorHandling**
   - Tests double allocation error handling
   - Validates invalid size parameter handling
   - Tests operations on unallocated PTY
   - Verifies binary not found error handling

5. **TestPTYLayer_IntegrationWithBrummer**
   - Tests starting Brummer with PTY integration
   - Validates basic TUI interaction patterns
   - Tests output capture and analysis
   - Verifies multiple command processing

6. **TestPTYLayer_ImplementationRequired**
   - Verifies that PTY layer is not yet implemented
   - This test will fail when real implementation is provided

## Current Status

⚠️ **No PTY Layer Implementation Exists Yet**

The current tests use placeholder methods that return `NotImplementedError`. When a real PTY layer is implemented:

1. The placeholder methods should be replaced with real implementations
2. The `TestPTYLayer_ImplementationRequired` test should be updated or removed
3. Integration tests should validate real PTY behavior with Brummer
4. Performance benchmarks should be added

## Running Tests

```bash
# Run all PTY integration tests
BRUMMER_BINARY=./brum go test -tags="tty integration" -v ./test/tui/pty

# Run specific test
BRUMMER_BINARY=./brum go test -tags="tty integration" -v ./test/tui/pty -run TestPTYLayer_AllocationAndManagement

# Run in short mode (skips integration tests)
go test -short -v ./test/tui/pty
```

## Environment Variables

- `BRUMMER_BINARY`: Path to the brummer binary (defaults to `./brum`)
  Required for integration tests with the actual application.

## Expected Test Behavior

### Before PTY Implementation
- `TestPTYLayer_AllocationAndManagement` passes by expecting `NotImplementedError`
- `TestPTYLayer_ImplementationRequired` passes by confirming no implementation
- Other tests skip due to lack of PTY allocation

### After PTY Implementation
- All tests should pass with real PTY functionality
- `TestPTYLayer_ImplementationRequired` should be updated or removed
- Tests should validate real PTY behavior and integration

## Dependencies

- `github.com/creack/pty` - PTY allocation and management
- `github.com/hinshun/vt10x` - Terminal emulator integration
- `github.com/stretchr/testify` - Test assertions and helpers

## Interface Definition

The PTY layer should implement the following interface methods (currently placeholders):

```go
// Core PTY operations
AllocatePTY(cols, rows int) error
SetSize(cols, rows int) error
GetSize() (*pty.Winsize, error)
IsAllocated() bool
GetFileDescriptor() *os.File

// TTY bridge functionality
InitializeTTYBridge() error
GetTerminal() *vt10x.Terminal
SendInput(input string) error
GetOutput() []byte

// Process management
StartTestProcess() error
Cleanup()
Cancel()
```

## Integration with Testing Framework

The PTY layer tests integrate with the existing TUI testing framework and follow the same patterns as:

- `/test/tty/` - Real PTY integration tests
- `/test/tui/harness/` - Test harness contract tests
- `/test/tui/integration/` - TUI integration tests

## Next Steps

1. **Implement PTY Layer**: Create the real PTY layer implementation
2. **Update Tests**: Replace placeholder methods with real implementations
3. **Add Performance Tests**: Add benchmarks for PTY operations
4. **Integration**: Ensure PTY layer works with existing TUI components
5. **Documentation**: Update this README with implementation details

## Architecture Notes

The PTY layer is designed to provide:
- **Abstracted PTY operations**: Hide complexity of PTY allocation/management
- **TTY bridge functionality**: Bridge PTY to terminal emulator for testing
- **Resource management**: Proper cleanup and error handling
- **Testing integration**: Seamless integration with existing test framework
- **Non-blocking operation**: Support for concurrent testing scenarios

This enables automated TUI testing without requiring actual TTY allocation, solving the critical limitation mentioned in the project documentation.