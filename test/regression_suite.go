package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TestResult represents the result of a single test
type TestResult struct {
	Name      string
	Mode      string // "TUI" or "NoTUI"
	Component string // "MCP", "Proxy", "Logging", "Processes"
	Passed    bool
	Duration  time.Duration
	Error     string
	Details   []string
}

// TestSuite manages and runs all regression tests
type TestSuite struct {
	Results    []TestResult
	BinaryPath string
	TestDir    string
	Verbose    bool
}

// NewTestSuite creates a new test suite
func NewTestSuite(binaryPath string, verbose bool) *TestSuite {
	testDir := filepath.Join(filepath.Dir(binaryPath), "test_workspace")
	return &TestSuite{
		Results:    make([]TestResult, 0),
		BinaryPath: binaryPath,
		TestDir:    testDir,
		Verbose:    verbose,
	}
}

// RunAll executes the complete regression test suite
func (ts *TestSuite) RunAll() error {
	fmt.Println("🧪 Starting Brummer Regression Test Suite")
	fmt.Println("==========================================")

	// Setup test workspace
	if err := ts.setupTestWorkspace(); err != nil {
		return fmt.Errorf("failed to setup test workspace: %w", err)
	}
	defer ts.cleanupTestWorkspace()

	// Test categories to run
	testCategories := []struct {
		name string
		fn   func() error
	}{
		{"MCP Server Tests", ts.runMCPTests},
		{"Proxy Server Tests", ts.runProxyTests},
		{"Logging System Tests", ts.runLoggingTests},
		{"Process Management Tests", ts.runProcessTests},
		{"Integration Tests", ts.runIntegrationTests},
	}

	// Run all test categories
	for _, category := range testCategories {
		fmt.Printf("\n📋 Running %s...\n", category.name)
		if err := category.fn(); err != nil {
			fmt.Printf("❌ %s failed: %v\n", category.name, err)
		}
	}

	// Print summary
	ts.printSummary()

	// Return error if any tests failed
	if ts.hasFailures() {
		return fmt.Errorf("regression tests failed")
	}

	return nil
}

// addResult adds a test result to the suite
func (ts *TestSuite) addResult(result TestResult) {
	ts.Results = append(ts.Results, result)

	status := "✅ PASS"
	if !result.Passed {
		status = "❌ FAIL"
	}

	fmt.Printf("  %s [%s] %s - %s (%v)\n",
		status, result.Mode, result.Component, result.Name, result.Duration)

	if !result.Passed && result.Error != "" {
		fmt.Printf("    Error: %s\n", result.Error)
	}

	if ts.Verbose && len(result.Details) > 0 {
		for _, detail := range result.Details {
			fmt.Printf("    📝 %s\n", detail)
		}
	}
}

// hasFailures returns true if any tests failed
func (ts *TestSuite) hasFailures() bool {
	for _, result := range ts.Results {
		if !result.Passed {
			return true
		}
	}
	return false
}

// printSummary prints the final test summary
func (ts *TestSuite) printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 REGRESSION TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 50))

	// Count results by category and mode
	stats := make(map[string]map[string]int)
	for _, result := range ts.Results {
		if stats[result.Component] == nil {
			stats[result.Component] = make(map[string]int)
		}
		if result.Passed {
			stats[result.Component]["pass"]++
		} else {
			stats[result.Component]["fail"]++
		}
	}

	// Print component stats
	totalPass := 0
	totalFail := 0

	for component, counts := range stats {
		pass := counts["pass"]
		fail := counts["fail"]
		total := pass + fail

		status := "✅"
		if fail > 0 {
			status = "❌"
		}

		fmt.Printf("%s %s: %d/%d passed (%d failed)\n",
			status, component, pass, total, fail)

		totalPass += pass
		totalFail += fail
	}

	// Print overall stats
	fmt.Println(strings.Repeat("-", 50))
	overallStatus := "✅ ALL TESTS PASSED"
	if totalFail > 0 {
		overallStatus = fmt.Sprintf("❌ %d TESTS FAILED", totalFail)
	}

	fmt.Printf("OVERALL: %d/%d tests passed - %s\n",
		totalPass, totalPass+totalFail, overallStatus)

	// Print mode breakdown
	tuiTests := 0
	noTuiTests := 0
	for _, result := range ts.Results {
		if result.Mode == "TUI" {
			tuiTests++
		} else if result.Mode == "NoTUI" {
			noTuiTests++
		}
	}

	fmt.Printf("Mode Coverage: %d TUI tests, %d NoTUI tests\n", tuiTests, noTuiTests)

	if totalFail > 0 {
		fmt.Println("\n❌ FAILED TESTS:")
		for _, result := range ts.Results {
			if !result.Passed {
				fmt.Printf("  - [%s] %s: %s - %s\n",
					result.Mode, result.Component, result.Name, result.Error)
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run regression_suite.go <path-to-brum-binary> [--verbose]")
		os.Exit(1)
	}

	binaryPath := os.Args[1]
	verbose := len(os.Args) > 2 && os.Args[2] == "--verbose"

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Printf("❌ Binary not found: %s\n", binaryPath)
		os.Exit(1)
	}

	// Create and run test suite
	suite := NewTestSuite(binaryPath, verbose)
	if err := suite.RunAll(); err != nil {
		fmt.Printf("❌ Test suite failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🎉 All regression tests completed successfully!")
}

// setupTestWorkspace creates a test workspace with sample files
func (ts *TestSuite) setupTestWorkspace() error {
	// Create test directory
	if err := os.MkdirAll(ts.TestDir, 0755); err != nil {
		return err
	}

	// Create package.json for testing
	packageJSON := `{
  "name": "brummer-test-project",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'Running tests...' && sleep 2 && echo 'Tests completed!'",
    "build": "echo 'Building project...' && sleep 1 && echo 'Build completed!'",
    "dev": "echo 'Starting dev server...' && sleep 1 && echo 'Dev server running on http://localhost:3000'",
    "long-running": "echo 'Long running process...' && sleep 10 && echo 'Long process done'"
  }
}`

	packagePath := filepath.Join(ts.TestDir, "package.json")
	if err := os.WriteFile(packagePath, []byte(packageJSON), 0644); err != nil {
		return err
	}

	return nil
}

// cleanupTestWorkspace removes the test workspace
func (ts *TestSuite) cleanupTestWorkspace() {
	os.RemoveAll(ts.TestDir)
}
