# TUI Testing Harness API Contract

**Version**: 1.0.0
**Date**: 2025-01-12
**Purpose**: Defines the API contract for TUI Integration Testing Harness

## Overview

This document defines the API contract for the TUI Testing Harness, specifying interfaces for automated testing of Brummer's terminal interface functionality without requiring TTY allocation.

## Core Interfaces

### TestHarness Interface

```go
type TestHarness interface {
    // Lifecycle management
    Initialize(config TestHarnessConfiguration) error
    Start() error
    Stop() error
    GetStatus() TestHarnessStatus

    // Test execution
    RunTest(testCase TestCase) TestResult
    RunTestSuite(testSuite TestSuite) TestSuiteResult

    // Component access
    GetInputSimulator() InputSimulator
    GetStateValidator() StateValidator
    GetTestReporter() TestReporter
}
```

### InputSimulator Interface

```go
type InputSimulator interface {
    // Input generation
    GenerateKeySequence(action string) ([]tea.KeyMsg, error)
    SimulateKeyPress(key tea.KeyMsg) error
    SimulateKeySequence(sequence []tea.KeyMsg) error

    // Configuration
    SetKeyMappings(mappings []KeyMapping) error
    SetTimingDelay(delay time.Duration) error

    // State management
    GetCurrentSequence() []tea.KeyMsg
    ClearSequence() error
}
```

### StateValidator Interface

```go
type StateValidator interface {
    // State capture and validation
    CaptureState(model interface{}) (StateSnapshot, error)
    ValidateTransition(from, to StateSnapshot) ValidationResult
    CompareStates(expected, actual StateSnapshot) ComparisonResult

    // Expected state management
    AddExpectedState(state ExpectedState) error
    ClearExpectedStates() error

    // Validation rules
    AddValidationRule(rule ValidationRule) error
    ValidateState(state StateSnapshot) []ValidationResult
}
```

### TestReporter Interface

```go
type TestReporter interface {
    // Result management
    AddTestResult(result TestResult) error
    GetTestResults() []TestResult
    GetTestResult(testID string) (TestResult, error)

    // Report generation
    GenerateReport() (Report, error)
    GenerateSummary() TestSummary
    ExportReport(format ReportFormat) ([]byte, error)

    // Metrics
    CalculateMetrics() TestMetrics
    GetMetrics() TestMetrics
}
```

## Data Structures

### TestHarnessConfiguration

```go
type TestHarnessConfiguration struct {
    TestTimeout     time.Duration `json:"test_timeout"`
    KeyInputDelay   time.Duration `json:"key_input_delay"`
    StateValidation bool          `json:"state_validation"`
    ReportLevel     ReportLevel   `json:"report_level"`
    TestScope       TestScope     `json:"test_scope"`
}
```

### TestCase

```go
type TestCase struct {
    ID           string        `json:"id"`
    Name         string        `json:"name"`
    Description  string        `json:"description"`
    Actions      []TestAction  `json:"actions"`
    ExpectedStates []ExpectedState `json:"expected_states"`
    Preconditions []string      `json:"preconditions"`
    Timeout      time.Duration `json:"timeout"`
}
```

### TestAction

```go
type TestAction struct {
    Type        ActionType    `json:"type"`
    Parameters  interface{}   `json:"parameters"`
    Description  string        `json:"description"`
    Delay       time.Duration `json:"delay"`
}
```

### TestResult

```go
type TestResult struct {
    TestID       string        `json:"test_id"`
    Action       string        `json:"action"`
    ExpectedState ExpectedState `json:"expected_state"`
    ActualState   ActualState   `json:"actual_state"`
    Status       TestStatus    `json:"status"`
    Duration     time.Duration `json:"duration"`
    Error        string        `json:"error,omitempty"`
    Timestamp    time.Time     `json:"timestamp"`
    Metrics      TestMetrics   `json:"metrics"`
}
```

### Validation Rules

```go
type ValidationRule struct {
    ID          string              `json:"id"`
    Name        string              `json:"name"`
    Description  string              `json:"description"`
    Validator   ValidationFunction  `json:"-"`
    Parameters  interface{}         `json:"parameters"`
}

type ValidationFunction func(state StateSnapshot) ValidationResult
```

## API Endpoints (for external integration)

### Test Execution

#### POST /api/v1/tests/run
Execute a single test case.

**Request**:
```json
{
  "test_case": {
    "id": "view-switching-1",
    "name": "Switch to Processes View",
    "actions": [
      {
        "type": "key_press",
        "parameters": {"key": "1"},
        "description": "Press number 1 key"
      }
    ],
    "expected_states": [
      {
        "trigger_action": "key_press",
        "view_type": "processes",
        "timeout": "1s"
      }
    ]
  }
}
```

**Response**:
```json
{
  "test_result": {
    "test_id": "view-switching-1",
    "status": "pass",
    "duration": "250ms",
    "actual_state": {
      "view_type": "processes",
      "selected_process": "",
      "timestamp": "2025-01-12T10:30:00Z"
    }
  }
}
```

#### POST /api/v1/tests/suite
Execute a test suite.

**Request**:
```json
{
  "test_suite": {
    "name": "Basic TUI Functionality",
    "test_cases": ["view-switching-1", "command-palette-1", "keyboard-navigation-1"],
    "configuration": {
      "test_timeout": "30s",
      "state_validation": true
    }
  }
}
```

#### GET /api/v1/results/{test_id}
Retrieve test results.

**Response**:
```json
{
  "test_result": {
    "test_id": "view-switching-1",
    "status": "pass",
    "duration": "250ms",
    "metrics": {
      "input_events": 1,
      "state_validations": 1,
      "view_changes": 1
    }
  }
}
```

### Reporting

#### GET /api/v1/reports/summary
Get test execution summary.

**Response**:
```json
{
  "summary": {
    "total_tests": 10,
    "passed": 8,
    "failed": 2,
    "total_duration": "5.2s",
    "last_execution": "2025-01-12T10:30:00Z"
  }
}
```

#### GET /api/v1/reports/{format}
Generate test report in specified format.

**Parameters**:
- `format`: `json` | `html` | `xml` | `junit`

**Response**: Report data in requested format

## Error Handling

### Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| 1001 | Test harness not initialized | 400 |
| 1002 | Invalid test configuration | 400 |
| 1003 | Test execution timeout | 408 |
| 1004 | State validation failed | 422 |
| 1005 | Input simulation error | 500 |
| 1006 | Report generation failed | 500 |

### Error Response Format

```json
{
  "error": {
    "code": 1003,
    "message": "Test execution timeout",
    "details": {
      "test_id": "view-switching-1",
      "timeout": "30s",
      "elapsed": "30.1s"
    }
  }
}
```

## Integration Points

### Pre-commit Hook Integration

```bash
#!/bin/bash
# pre-commit-tui-tests.sh

# Run TUI integration tests
./test/scripts/run_tui_tests.sh

# Check exit code
if [ $? -ne 0 ]; then
    echo "❌ TUI tests failed - commit blocked"
    exit 1
fi

echo "✅ TUI tests passed"
exit 0
```

### Makefile Integration

```makefile
# TUI testing targets
.PHONY: test-tui test-tui-integration validate-tui

test-tui:
	@echo "Running TUI integration tests..."
	@./test/scripts/run_tui_tests.sh

test-tui-integration:
	@echo "Running comprehensive TUI tests..."
	@BRUMMER_BINARY=./brum go test -tags=integration ./test/tui -v

validate-tui:
	@echo "Validating TUI functionality..."
	@./test/scripts/validate_tui.sh
```

## Performance Requirements

- **Test Execution**: Individual tests must complete within 30 seconds
- **State Validation**: State comparison must complete within 100ms
- **Report Generation**: Report generation must complete within 5 seconds
- **Memory Usage**: Testing harness must not exceed 100MB memory usage
- **Concurrent Tests**: Support up to 5 concurrent test executions

## Security Considerations

- Test harness should not access production data
- Input simulation should be limited to test scope
- Test reports should not contain sensitive application state
- Pre-commit hooks should be executable only by authorized users

This API contract provides the foundation for implementing a comprehensive TUI testing harness that meets all functional requirements while maintaining compatibility with existing Brummer architecture and development workflows.