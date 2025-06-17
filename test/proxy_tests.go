package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// runProxyTests executes all proxy-related tests
func (ts *TestSuite) runProxyTests() error {
	tests := []struct {
		name string
		fn   func() TestResult
	}{
		{"Proxy Server Startup (NoTUI)", ts.testProxyStartupNoTUI},
		{"Proxy Server Startup (TUI)", ts.testProxyStartupTUI},
		{"Proxy URL Detection", ts.testProxyURLDetection},
		{"Proxy Request Handling", ts.testProxyRequestHandling},
		{"Proxy Disable Flag", ts.testProxyDisableFlag},
	}

	for _, test := range tests {
		result := test.fn()
		ts.addResult(result)
	}

	return nil
}

// testProxyStartupNoTUI tests proxy server startup in headless mode
func (ts *TestSuite) testProxyStartupNoTUI() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Proxy Server Startup",
		Mode:      "NoTUI",
		Component: "Proxy",
		Passed:    false,
	}

	// Run brummer with a script that outputs URLs
	cmd := exec.Command("timeout", "5s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "dev")
	output, err := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check for proxy server startup messages
	proxyPattern := regexp.MustCompile(`🌐 Started HTTP proxy server on port \d+`)
	if proxyPattern.MatchString(outputStr) {
		result.Passed = true
		result.Details = []string{"Proxy server started successfully in headless mode"}

		// Extract proxy information
		if matches := proxyPattern.FindString(outputStr); matches != "" {
			result.Details = append(result.Details, fmt.Sprintf("Found: %s", matches))
		}
	} else {
		result.Error = "Proxy server startup message not found in output"
		if err != nil {
			result.Error += fmt.Sprintf(" (error: %v)", err)
		}
		result.Details = []string{fmt.Sprintf("Output: %s", outputStr)}
	}

	return result
}

// testProxyStartupTUI tests proxy server startup in TUI mode
func (ts *TestSuite) testProxyStartupTUI() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Proxy Server Startup",
		Mode:      "TUI",
		Component: "Proxy",
		Passed:    false,
	}

	// Run brummer in TUI mode with a script that should trigger proxy
	cmd := exec.Command("timeout", "3s", ts.BinaryPath, "-d", ts.TestDir, "dev")
	output, err := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// In TUI mode, we expect the process to start successfully and then timeout
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 124 { // timeout exit code
			result.Passed = true
			result.Details = []string{"TUI mode with proxy started successfully (terminated by timeout as expected)"}
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

// testProxyURLDetection tests that proxy detects URLs from process output
func (ts *TestSuite) testProxyURLDetection() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "URL Detection",
		Mode:      "NoTUI",
		Component: "Proxy",
		Passed:    false,
	}

	// Run brummer with dev script that outputs a URL
	cmd := exec.Command("timeout", "5s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "dev")
	output, _ := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check for URL detection and proxy creation
	urlDetectionPatterns := []string{
		"http://localhost:3000",
		"proxy server on port",
		"Started HTTP proxy",
	}

	detectedCount := 0
	for _, pattern := range urlDetectionPatterns {
		if strings.Contains(outputStr, pattern) {
			detectedCount++
			result.Details = append(result.Details, fmt.Sprintf("Detected: %s", pattern))
		}
	}

	if detectedCount >= 2 { // At least URL and proxy startup
		result.Passed = true
		result.Details = append(result.Details, "URL detection and proxy creation working")
	} else {
		result.Error = "URL detection or proxy creation not working properly"
		result.Details = append(result.Details, fmt.Sprintf("Output: %s", outputStr))
	}

	return result
}

// testProxyRequestHandling tests that proxy can handle HTTP requests
func (ts *TestSuite) testProxyRequestHandling() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Request Handling",
		Mode:      "NoTUI",
		Component: "Proxy",
		Passed:    false,
	}

	// Start brummer with dev script in background
	cmd := exec.Command(ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "dev")
	cmd.Start()
	defer cmd.Process.Kill()

	// Wait for servers to start
	time.Sleep(3 * time.Second)

	// Try to find proxy port from common ranges
	proxyPorts := []int{20000, 20001, 20002, 20003, 20004, 20005}
	var lastErr error

	for _, port := range proxyPorts {
		url := fmt.Sprintf("http://localhost:%d", port)
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		// If we get any response (even error), proxy is working
		result.Passed = true
		result.Details = []string{
			fmt.Sprintf("Successfully connected to proxy on port %d", port),
			fmt.Sprintf("Response status: %s", resp.Status),
		}
		break
	}

	if !result.Passed {
		result.Error = fmt.Sprintf("Failed to connect to proxy server: %v", lastErr)
		result.Details = []string{"Tried ports: 20000-20005"}
	}

	result.Duration = time.Since(start)
	return result
}

// testProxyDisableFlag tests the --no-proxy flag
func (ts *TestSuite) testProxyDisableFlag() TestResult {
	start := time.Now()
	result := TestResult{
		Name:      "Disable Flag",
		Mode:      "NoTUI",
		Component: "Proxy",
		Passed:    false,
	}

	// Run brummer with --no-proxy flag
	cmd := exec.Command("timeout", "5s", ts.BinaryPath, "-d", ts.TestDir, "--no-tui", "--no-proxy", "dev")
	output, _ := cmd.CombinedOutput()

	result.Duration = time.Since(start)
	outputStr := string(output)

	// Check that proxy server is NOT started
	proxyPattern := regexp.MustCompile(`🌐 Started HTTP proxy server`)
	if !proxyPattern.MatchString(outputStr) {
		result.Passed = true
		result.Details = []string{"Proxy server correctly disabled with --no-proxy flag"}

		// Verify the dev script still runs
		if strings.Contains(outputStr, "Dev server running") {
			result.Details = append(result.Details, "Dev script executed successfully without proxy")
		}
	} else {
		result.Error = "Proxy server started despite --no-proxy flag"
		result.Details = []string{fmt.Sprintf("Output: %s", outputStr)}
	}

	return result
}
