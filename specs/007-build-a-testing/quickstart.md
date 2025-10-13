# TUI Integration Testing Harness - Quick Start Guide

**Version**: 1.0.0
**Date**: 2025-01-12
**Purpose**: Get started with TUI Integration Testing Harness quickly

## Overview

The TUI Integration Testing Harness provides automated testing capabilities for Brummer's terminal interface without requiring TTY allocation. This guide will help you get started with writing and running TUI tests.

## Prerequisites

- Go 1.21+ installed
- Brummer source code
- Basic understanding of BubbleTea framework
- Familiarity with Go testing patterns

## Quick Setup

### 1. Build the Test Harness

```bash
# Build Brummer with testing support
make build

# Or build directly
go build -o brum ./cmd/brum/main.go
```

### 2. Verify Test Environment

```bash
# Run basic TUI test validation
./test/scripts/validate_tui.sh

# Expected output:
# ✅ TUI testing environment validated
# ✅ Input simulation working
# ✅ State validation working
# ✅ Test reporting working
```

## Writing Your First TUI Test

### Basic Test Structure

Create a test file in `test/tui/integration/`:

```go
package integration

import (
    "testing"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/tui"
    "github.com/standardbeagle/brummer/test/tui/harness"
)

func TestViewSwitching(t *testing.T) {
    // Create test harness
    th := harness.NewTestHarness()

    // Configure test
    config := harness.TestHarnessConfiguration{
        TestTimeout:     10 * time.Second,
        KeyInputDelay:   100 * time.Millisecond,
        StateValidation: true,
        ReportLevel:     harness.DetailedReport,
    }

    err := th.Initialize(config)
    if err != nil {
        t.Fatalf("Failed to initialize test harness: %v", err)
    }

    // Define test case
    testCase := harness.TestCase{
        ID:          "switch-to-processes",
        Name:        "Switch to Processes View",
        Description: "Test switching to processes view using number key 1",
        Actions: []harness.TestAction{
            {
                Type: harness.KeyPressAction,
                Parameters: map[string]interface{}{
                    "key": tea.KeyMsg{
                        Type: tea.KeyRunes,
                        Runes: []rune{'1'},
                    },
                },
                Description: "Press number 1 key",
                Delay:      100 * time.Millisecond,
            },
        },
        ExpectedStates: []harness.ExpectedState{
            {
                TriggerAction: "key_press",
                ViewType:      tui.ViewProcesses,
                Timeout:       1 * time.Second,
            },
        },
    }

    // Run test
    result := th.RunTest(testCase)

    // Validate result
    if result.Status != harness.TestStatusPass {
        t.Errorf("Test failed: %s", result.Error)
    }

    // Cleanup
    err = th.Stop()
    if err != nil {
        t.Logf("Warning: Failed to stop test harness: %v", err)
    }
}
```

### Running Individual Tests

```bash
# Run specific TUI test
go test -tags=integration ./test/tui/integration -run TestViewSwitching -v

# Run all TUI integration tests
go test -tags=integration ./test/tui/integration -v

# Run with timeout
go test -tags=integration ./test/tui/integration -timeout 30s
```

## Common Test Patterns

### 1. Keyboard Input Testing

```go
func TestKeyboardInput(t *testing.T) {
    th := harness.NewTestHarness()
    config := harness.DefaultTestConfig()
    th.Initialize(config)
    defer th.Stop()

    // Test different keyboard inputs
    testCases := []struct {
        name     string
        key      tea.KeyMsg
        expected tui.ViewType
    }{
        {"Press 1", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, tui.ViewProcesses},
        {"Press 2", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}, tui.ViewLogs},
        {"Press 3", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}, tui.ViewErrors},
        {"Press Tab", tea.KeyMsg{Type: tea.KeyTab}, tui.ViewLogs}, // Assuming tab cycles to next view
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            testCase := harness.TestCase{
                ID:    "keyboard-" + tc.name,
                Name:  "Test " + tc.name,
                Actions: []harness.TestAction{
                    {
                        Type:       harness.KeyPressAction,
                        Parameters: map[string]interface{}{"key": tc.key},
                    },
                },
                ExpectedStates: []harness.ExpectedState{
                    {
                        TriggerAction: "key_press",
                        ViewType:      tc.expected,
                        Timeout:       1 * time.Second,
                    },
                },
            }

            result := th.RunTest(testCase)
            if result.Status != harness.TestStatusPass {
                t.Errorf("Failed to switch view with %s: %s", tc.name, result.Error)
            }
        })
    }
}
```

### 2. Command Palette Testing

```go
func TestCommandPalette(t *testing.T) {
    th := harness.NewTestHarness()
    config := harness.DefaultTestConfig()
    th.Initialize(config)
    defer th.Stop()

    // Test opening command palette
    testCase := harness.TestCase{
        ID:   "open-command-palette",
        Name: "Open Command Palette",
        Actions: []harness.TestAction{
            {
                Type: harness.KeyPressAction,
                Parameters: map[string]interface{}{
                    "key": tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}},
                },
            },
        },
        ExpectedStates: []harness.ExpectedState{
            {
                TriggerAction: "key_press",
                ModelState: map[string]interface{}{
                    "command_window_visible": true,
                },
                Timeout: 1 * time.Second,
            },
        },
    }

    result := th.RunTest(testCase)
    if result.Status != harness.TestStatusPass {
        t.Errorf("Failed to open command palette: %s", result.Error)
    }
}
```

### 3. Complex Interaction Testing

```go
func TestComplexWorkflow(t *testing.T) {
    th := harness.NewTestHarness()
    config := harness.DefaultTestConfig()
    th.Initialize(config)
    defer th.Stop()

    // Test multi-step workflow
    testCase := harness.TestCase{
        ID:   "complex-workflow",
        Name: "Complex TUI Workflow",
        Actions: []harness.TestAction{
            // Switch to processes view
            {
                Type: harness.KeyPressAction,
                Parameters: map[string]interface{}{
                    "key": tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}},
                },
            },
            // Open command palette
            {
                Type: harness.KeyPressAction,
                Parameters: map[string]interface{}{
                    "key": tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}},
                },
                Delay: 200 * time.Millisecond,
            },
            // Type command
            {
                Type: harness.KeyPressAction,
                Parameters: map[string]interface{}{
                    "key": tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}},
                },
                Delay: 100 * time.Millisecond,
            },
            // Execute command
            {
                Type: harness.KeyPressAction,
                Parameters: map[string]interface{}{
                    "key": tea.KeyMsg{Type: tea.KeyEnter},
                },
                Delay: 100 * time.Millisecond,
            },
        },
        ExpectedStates: []harness.ExpectedState{
            {
                TriggerAction: "key_press",
                ViewType:      tui.ViewProcesses,
                Timeout:       1 * time.Second,
            },
            {
                TriggerAction: "key_press",
                ModelState: map[string]interface{}{
                    "command_window_visible": true,
                },
                Timeout: 1 * time.Second,
            },
        },
    }

    result := th.RunTest(testCase)
    if result.Status != harness.TestStatusPass {
        t.Errorf("Complex workflow failed: %s", result.Error)
    }

    // Check detailed metrics
    if result.Metrics.InputEvents != 4 {
        t.Errorf("Expected 4 input events, got %d", result.Metrics.InputEvents)
    }
}
```

## Running Tests

### Command Line Usage

```bash
# Run all TUI tests
make test-tui

# Run integration tests
make test-tui-integration

# Run with verbose output
go test -tags=integration ./test/tui -v

# Run specific test file
go test -tags=integration ./test/tui/integration/keyboard_test.go -v

# Run with coverage
go test -tags=integration ./test/tui -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Scripts

```bash
# Run comprehensive test suite
./test/scripts/run_tui_tests.sh

# Validate TUI functionality
./test/scripts/validate_tui.sh

# Setup test environment
./test/scripts/setup_test_env.sh
```

## Understanding Test Results

### Test Output Format

```
=== RUN   TestViewSwitching
=== PAUSE TestViewSwitching
    test_harness.go:45: Switching to Processes view
    test_harness.go:67: View changed to: processes
=== CONT  TestViewSwitching
--- PASS: TestViewSwitching (0.25s)
PASS
ok      github.com/standardbeagle/brummer/test/tui/integration    0.256s
```

### Test Report Format

```json
{
  "test_id": "switch-to-processes",
  "status": "pass",
  "duration": "250ms",
  "actual_state": {
    "view_type": "processes",
    "selected_process": "",
    "timestamp": "2025-01-12T10:30:00Z"
  },
  "metrics": {
    "input_events": 1,
    "state_validations": 1,
    "view_changes": 1
  }
}
```

## Troubleshooting

### Common Issues

1. **Test timeout errors**
   ```bash
   # Increase timeout in test configuration
   config.TestTimeout = 30 * time.Second
   ```

2. **State validation failures**
   ```bash
   # Check expected state definitions
   # Verify model state structure
   # Use detailed logging to debug
   ```

3. **Input simulation not working**
   ```bash
   # Verify key message format
   # Check keyboard mapping configuration
   # Test with simple key sequences first
   ```

### Debug Mode

```bash
# Run tests with debug logging
DEBUG_TUI_TESTS=true go test -tags=integration ./test/tui -v

# Generate detailed test reports
go test -tags=integration ./test/tui -reporter=json
```

## Best Practices

1. **Keep tests focused**: Test one specific behavior per test
2. **Use descriptive names**: Make test names self-explanatory
3. **Clean up resources**: Always stop test harness in defer
4. **Validate edge cases**: Test error conditions and boundary cases
5. **Use table-driven tests**: For multiple similar test cases
6. **Document complex workflows**: Add comments for multi-step tests

## Integration with Development Workflow

### Pre-commit Hooks

The testing harness includes pre-commit hooks that automatically run TUI tests:

```bash
# Pre-commit hook will run:
# 1. Basic TUI functionality tests
# 2. Keyboard input validation
# 3. View switching verification
# 4. Command palette testing
```

### Continuous Integration

```yaml
# .github/workflows/tui-tests.yml
name: TUI Tests
on: [push, pull_request]
jobs:
  tui-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: make test-tui
      - run: make test-tui-integration
```

This quick start guide provides everything you need to get started with TUI integration testing. For more advanced usage and detailed API documentation, refer to the other specification documents.