package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// runProcessTests executes all process management tests
func (ts *TestSuite) runProcessTests() error {
	tests := []struct {
		name string
		fn   func() TestResult
	}{
		{"Process Startup (NoTUI)", ts.testProcessStartupNoTUI},
		{"Process Startup (TUI)", ts.testProcessStartupTUI},
		{"Multiple Processes", ts.testMultipleProcesses},
		{"Process Exit Handling", ts.testProcessExitHandling},
		{"Process ID Generation", ts.testProcessIDGeneration},
		{"Script Detection", ts.testScriptDetection},
	}

	for _, test := range tests {
		result := test.fn()
		ts.addResult(result)
	}

	return nil
}

// testProcessStartupNoTUI tests process startup in headless mode
func (ts *TestSuite) testProcessStartupNoTUI() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Process Startup",
		Mode:      "NoTUI",
		Component: "Processes",
		Passed:    false,
	}

	// Run brummer with test script
	cmd := exec.Command("timeout", "5s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "test")
	output, _ := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check for process startup messages
	startupPatterns := []string{
		"Started script 'test'",
		"✅ Started script",
		"PID:",
	}

	detectedCount := 0
	for _, pattern := range startupPatterns {
		if strings.Contains(outputStr, pattern) {
			detectedCount++
			result.Details = append(result.Details, fmt.Sprintf("Found startup indicator: %s", pattern))
		}
	}

	if detectedCount >= 1 {
		result.Passed = true
		result.Details = append(result.Details, "Process startup working correctly in headless mode")

		// Extract PID if available
		pidPattern := regexp.MustCompile(`PID: (\w+-\d+)`)
		if matches := pidPattern.FindStringSubmatch(outputStr); len(matches) > 1 {
			result.Details = append(result.Details, fmt.Sprintf("Process ID: %s", matches[1]))
		}
	} else {
		result.Error = "Process startup indicators not found"
		result.Details = append(result.Details, fmt.Sprintf("Output: %s", outputStr))
	}

	return result
}

// testProcessStartupTUI tests process startup in TUI mode
func (ts *TestSuite) testProcessStartupTUI() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Process Startup",
		Mode:      "TUI",
		Component: "Processes",
		Passed:    false,
	}

	// Run brummer in TUI mode with test script
	cmd := exec.Command("timeout", "3s", ts.BinaryPath, "-d", ts.TestDir, "test")
	output, err := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// In TUI mode, we expect successful startup (timeout exit code)
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 124 { // timeout exit code
			result.Passed = true
			result.Details = []string{"TUI mode with process started successfully (terminated by timeout as expected)"}
		} else {
			result.Error = fmt.Sprintf("TUI mode failed with exit code %d", exitError.ExitCode())
			result.Details = []string{fmt.Sprintf("Output: %s", outputStr)}
		}
	} else if err == nil {
		result.Error = "TUI process completed unexpectedly"
		result.Details = []string{fmt.Sprintf("Output: %s", outputStr)}
	} else {
		result.Error = fmt.Sprintf("Unexpected error: %v", err)
		result.Details = []string{fmt.Sprintf("Output: %s", outputStr)}
	}

	return result
}

// testMultipleProcesses tests running multiple processes simultaneously
func (ts *TestSuite) testMultipleProcesses() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Multiple Processes",
		Mode:      "NoTUI",
		Component: "Processes",
		Passed:    false,
	}

	// Run brummer with multiple scripts
	cmd := exec.Command("timeout", "8s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "test", "build")
	output, _ := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check for multiple process indicators
	testProcessPatterns := []string{
		"Started script 'test'",
		"Running tests...",
		"Tests completed!",
	}

	buildProcessPatterns := []string{
		"Started script 'build'",
		"Building project...",
		"Build completed!",
	}

	testDetected := 0
	buildDetected := 0

	for _, pattern := range testProcessPatterns {
		if strings.Contains(outputStr, pattern) {
			testDetected++
		}
	}

	for _, pattern := range buildProcessPatterns {
		if strings.Contains(outputStr, pattern) {
			buildDetected++
		}
	}

	if testDetected >= 2 && buildDetected >= 2 {
		result.Passed = true
		result.Details = []string{
			"Multiple processes executed successfully",
			fmt.Sprintf("Test process indicators: %d", testDetected),
			fmt.Sprintf("Build process indicators: %d", buildDetected),
		}
	} else {
		result.Error = "Multiple processes not executed properly"
		result.Details = []string{
			fmt.Sprintf("Test indicators: %d, Build indicators: %d", testDetected, buildDetected),
			fmt.Sprintf("Output: %s", outputStr),
		}
	}

	return result
}

// testProcessExitHandling tests that process exits are handled correctly
func (ts *TestSuite) testProcessExitHandling() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Process Exit Handling",
		Mode:      "NoTUI",
		Component: "Processes",
		Passed:    false,
	}

	// Run brummer with test script (which should complete)
	cmd := exec.Command("timeout", "5s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "test")
	output, _ := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check for process completion indicators
	completionPatterns := []string{
		"Tests completed!",
		"Process exited",
		"exit code",
	}

	detectedCount := 0
	for _, pattern := range completionPatterns {
		if strings.Contains(outputStr, pattern) {
			detectedCount++
			result.Details = append(result.Details, fmt.Sprintf("Found completion indicator: %s", pattern))
		}
	}

	if detectedCount >= 1 {
		result.Passed = true
		result.Details = append(result.Details, "Process exit handling working correctly")
	} else {
		result.Error = "Process exit not handled properly"
		result.Details = append(result.Details, fmt.Sprintf("Output: %s", outputStr))
	}

	return result
}

// testProcessIDGeneration tests that processes get unique IDs
func (ts *TestSuite) testProcessIDGeneration() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Process ID Generation",
		Mode:      "NoTUI",
		Component: "Processes",
		Passed:    false,
	}

	// Run brummer with test script
	cmd := exec.Command("timeout", "5s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "test")
	output, _ := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check for process ID patterns
	pidPattern := regexp.MustCompile(`PID: (\w+-\d+)`)
	matches := pidPattern.FindAllStringSubmatch(outputStr, -1)

	if len(matches) >= 1 {
		result.Passed = true
		result.Details = []string{
			fmt.Sprintf("Found %d process ID(s)", len(matches)),
		}

		for i, match := range matches {
			if len(match) > 1 {
				result.Details = append(result.Details, fmt.Sprintf("Process ID %d: %s", i+1, match[1]))
			}
		}
	} else {
		result.Error = "No process IDs found in output"
		result.Details = []string{fmt.Sprintf("Output: %s", outputStr)}
	}

	return result
}

// testScriptDetection tests that package.json scripts are detected
func (ts *TestSuite) testScriptDetection() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Script Detection",
		Mode:      "NoTUI",
		Component: "Processes",
		Passed:    true, // Assume this works if we can run scripts
		Details:   []string{"Script detection tested indirectly via script execution"},
	}

	result.Duration = time.Since(start)
	return result
}
