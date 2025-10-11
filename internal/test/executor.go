package test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TestExecutor handles actual test execution for different frameworks
type TestExecutor struct {
	frameworks map[string]TestFramework
}

// NewTestExecutor creates a new test executor
func NewTestExecutor() *TestExecutor {
	te := &TestExecutor{
		frameworks: make(map[string]TestFramework),
	}

	// Register built-in frameworks
	te.RegisterFramework(&JestFramework{})
	te.RegisterFramework(&VitestFramework{})
	te.RegisterFramework(&GoTestFramework{})
	te.RegisterFramework(&PytestFramework{})

	return te
}

// RegisterFramework registers a test framework
func (te *TestExecutor) RegisterFramework(framework TestFramework) {
	te.frameworks[framework.Name()] = framework
}

// ExecuteTests runs tests and returns real results
func (te *TestExecutor) ExecuteTests(ctx context.Context, frameworkName, pattern string, options TestOptions) (*TestResults, error) {
	// Auto-detect framework if not specified
	if frameworkName == "" {
		detected := te.DetectAvailableFrameworks()
		if len(detected) == 0 {
			return nil, fmt.Errorf("no test frameworks detected in current directory")
		}
		frameworkName = detected[0].Name()
	}

	framework, exists := te.frameworks[frameworkName]
	if !exists {
		return nil, fmt.Errorf("framework %s not registered", frameworkName)
	}

	if !framework.Detect() {
		return nil, fmt.Errorf("framework %s not detected in current directory", frameworkName)
	}

	// Build the command
	cmd, err := te.buildTestCommand(framework, pattern, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build test command: %w", err)
	}

	// Execute the command
	return te.executeCommand(ctx, cmd, framework, options)
}

// DetectAvailableFrameworks returns all detected frameworks in current directory
func (te *TestExecutor) DetectAvailableFrameworks() []TestFramework {
	detected := make([]TestFramework, 0)
	for _, f := range te.frameworks {
		if f.Detect() {
			detected = append(detected, f)
		}
	}
	return detected
}

// buildTestCommand builds the actual command to run tests
func (te *TestExecutor) buildTestCommand(framework TestFramework, pattern string, options TestOptions) (*exec.Cmd, error) {
	var args []string

	switch framework.Name() {
	case "jest":
		args = []string{"jest", "--json", "--outputFile", "/tmp/jest-results.json"}
		if options.Coverage {
			args = append(args, "--coverage")
		}
		if pattern != "" {
			args = append(args, "--testNamePattern", pattern)
		}
		if options.Verbose {
			args = append(args, "--verbose")
		}
		return exec.Command("npx", args...), nil

	case "vitest":
		args = []string{"vitest", "run", "--reporter=json"}
		if options.Coverage {
			args = append(args, "--coverage")
		}
		if pattern != "" {
			args = append(args, pattern)
		}
		if options.Verbose {
			args = append(args, "--verbose")
		}
		return exec.Command("npx", args...), nil

	case "go":
		args = []string{"go", "test", "-v", "-json"}
		if options.Coverage {
			args = append(args, "-cover")
		}
		if pattern != "" {
			args = append(args, "-run", pattern)
		}
		return exec.Command("go", args...), nil

	case "pytest":
		args = []string{"python3", "-m", "pytest", "--json-report", "--json-report-file=/tmp/pytest-results.json"}
		if options.Coverage {
			args = append(args, "--cov")
		}
		if pattern != "" {
			args = append(args, "-k", pattern)
		}
		if options.Verbose {
			args = append(args, "-v")
		}
		return exec.Command(args[0], args[1:]...), nil

	case "node":
		// Custom Node.js test runner
		return exec.Command("node", "sample.test.js"), nil

	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework.Name())
	}
}

// executeCommand runs the command and parses results
func (te *TestExecutor) executeCommand(ctx context.Context, cmd *exec.Cmd, framework TestFramework, options TestOptions) (*TestResults, error) {
	results := &TestResults{
		TestCases: make([]TestCase, 0),
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start test command: %w", err)
	}

	// Create scanners to read output line by line
	scanner := bufio.NewScanner(stdout)
	errScanner := bufio.NewScanner(stderr)

	// Process output in real-time
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		done <- err
	}()

	// Parse output as it comes
	for scanner.Scan() {
		line := scanner.Text()
		te.processOutputLine(line, results, framework)
	}

	// Also process error output
	for errScanner.Scan() {
		line := errScanner.Text()
		te.processOutputLine(line, results, framework)
	}

	// Wait for command to complete
	if err := <-done; err != nil {
		// Try to parse any partial results we got
		if framework.Name() == "go" {
			// For Go tests, we can often get partial results even on failure
			te.parseGoTestResults(results)
		}

		return results, fmt.Errorf("test command failed: %w", err)
	}

	// Parse final results based on framework
	switch framework.Name() {
	case "jest", "vitest":
		return te.parseNodeTestResults(results, framework)
	case "go":
		return te.parseGoTestResults(results)
	case "pytest":
		return te.parsePythonTestResults(results)
	case "node":
		// For our custom Node.js runner, results are already processed
		return results, nil
	default:
		return results, nil
	}
}

// processOutputLine processes individual lines of test output
func (te *TestExecutor) processOutputLine(line string, results *TestResults, framework TestFramework) {
	// Log the line for debugging
	if framework.Name() == "go" {
		te.processGoOutputLine(line, results)
	} else if framework.Name() == "node" {
		te.processNodeOutputLine(line, results)
	}
	// For other frameworks, we'll parse after completion
}

// processGoOutputLine processes Go test output line by line
func (te *TestExecutor) processGoOutputLine(line string, results *TestResults) {
	// Look for JSON test output lines
	if strings.HasPrefix(line, "{") {
		var goTestEvent map[string]interface{}
		if err := json.Unmarshal([]byte(line), &goTestEvent); err == nil {
			if action, ok := goTestEvent["Action"].(string); ok {
				switch action {
				case "run":
					if name, ok := goTestEvent["Test"].(string); ok {
						tc := TestCase{
							Name:      name,
							Status:    TestCaseStatusRunning,
							StartTime: time.Now(),
						}
						results.TestCases = append(results.TestCases, tc)
					}
				case "pass":
					if name, ok := goTestEvent["Test"].(string); ok {
						te.updateTestCaseStatus(results, name, TestCaseStatusPassed)
					}
				case "fail":
					if name, ok := goTestEvent["Test"].(string); ok {
						te.updateTestCaseStatus(results, name, TestCaseStatusFailed)
						if output, ok := goTestEvent["Output"].(string); ok {
							te.updateTestCaseError(results, name, output)
						}
					}
				case "skip":
					if name, ok := goTestEvent["Test"].(string); ok {
						te.updateTestCaseStatus(results, name, TestCaseStatusSkipped)
					}
				}
			}
		}
	}
}

// processNodeOutputLine processes Node.js test output line by line
func (te *TestExecutor) processNodeOutputLine(line string, results *TestResults) {
	if strings.Contains(line, "Running") {
		// Extract test name
		parts := strings.Split(line, "Running")
		if len(parts) > 1 {
			name := strings.TrimSpace(parts[1])
			tc := TestCase{
				Name:      name,
				Status:    TestCaseStatusRunning,
				StartTime: time.Now(),
			}
			results.TestCases = append(results.TestCases, tc)
		}
	} else if strings.HasPrefix(line, "✓") {
		// Test passed
		te.updateTestCaseStatus(results, line, TestCaseStatusPassed)
	} else if strings.HasPrefix(line, "✗") {
		// Test failed
		te.updateTestCaseStatus(results, line, TestCaseStatusFailed)
	}
}

// updateTestCaseStatus updates the status of a test case
func (te *TestExecutor) updateTestCaseStatus(results *TestResults, identifier string, status TestCaseStatus) {
	for i := range results.TestCases {
		if strings.Contains(results.TestCases[i].Name, identifier) ||
			strings.Contains(identifier, results.TestCases[i].Name) {
			results.TestCases[i].Status = status
			results.TestCases[i].EndTime = time.Now()

			// Calculate duration
			if !results.TestCases[i].StartTime.IsZero() {
				results.TestCases[i].Duration = results.TestCases[i].EndTime.Sub(results.TestCases[i].StartTime)
			}

			// Update counters
			switch status {
			case TestCaseStatusPassed:
				results.PassedTests++
			case TestCaseStatusFailed:
				results.FailedTests++
			case TestCaseStatusSkipped:
				results.SkippedTests++
			}
			break
		}
	}
}

// updateTestCaseError updates error information for a test case
func (te *TestExecutor) updateTestCaseError(results *TestResults, identifier string, errorMsg string) {
	for i := range results.TestCases {
		if strings.Contains(results.TestCases[i].Name, identifier) {
			results.TestCases[i].FailureInfo = &TestFailureInfo{
				Message: errorMsg,
			}
			break
		}
	}
}

// parseGoTestResults finalizes Go test results
func (te *TestExecutor) parseGoTestResults(results *TestResults) (*TestResults, error) {
	results.TotalTests = len(results.TestCases)
	// Counters should already be updated during processing
	return results, nil
}

// parseNodeTestResults parses Node.js test results
func (te *TestExecutor) parseNodeTestResults(results *TestResults, framework TestFramework) (*TestResults, error) {
	// For our simple Node.js runner, results are processed line by line
	results.TotalTests = len(results.TestCases)
	return results, nil
}

// parsePythonTestResults parses Python test results
func (te *TestExecutor) parsePythonTestResults(results *TestResults) (*TestResults, error) {
	// For now, return the results as-is
	// TODO: Implement proper pytest JSON parsing
	results.TotalTests = len(results.TestCases)
	return results, nil
}
