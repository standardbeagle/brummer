package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestSuite represents a collection of tests
type TestSuite struct {
	Name        string
	Package     string
	Description string
	Critical    bool // Critical tests must pass
}

// Test suites in dependency order
var testSuites = []TestSuite{
	{
		Name:        "Events",
		Package:     "./pkg/events",
		Description: "Event bus and communication system",
		Critical:    true,
	},
	{
		Name:        "Config",
		Package:     "./internal/config",
		Description: "Configuration management",
		Critical:    true,
	},
	{
		Name:        "Logs",
		Package:     "./internal/logs",
		Description: "Log storage and processing",
		Critical:    true,
	},
	{
		Name:        "Process",
		Package:     "./internal/process",
		Description: "Process management and script execution",
		Critical:    true,
	},
	{
		Name:        "Discovery",
		Package:     "./internal/discovery",
		Description: "Instance discovery and registration",
		Critical:    true,
	},
	{
		Name:        "Proxy",
		Package:     "./internal/proxy",
		Description: "HTTP proxy server (reverse and full modes)",
		Critical:    false,
	},
	{
		Name:        "MCP Core",
		Package:     "./internal/mcp",
		Description: "MCP server implementation and connection management",
		Critical:    true,
	},
	{
		Name:        "TUI",
		Package:     "./internal/tui",
		Description: "Terminal user interface",
		Critical:    false,
	},
	{
		Name:        "Integration",
		Package:     "./test",
		Description: "Integration tests",
		Critical:    false,
	},
	{
		Name:        "Main",
		Package:     "./cmd/brum",
		Description: "Main application entry point",
		Critical:    true,
	},
}

func main() {
	fmt.Println("=== Brummer Test Suite Runner ===")
	fmt.Println()

	var args []string
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "short":
			args = []string{"-short"}
		case "verbose":
			args = []string{"-v"}
		case "race":
			args = []string{"-race"}
		case "cover":
			args = []string{"-cover", "-coverprofile=coverage.out"}
		case "bench":
			args = []string{"-bench=."}
		case "critical":
			// Run only critical tests
			runCriticalTests()
			return
		case "help":
			printHelp()
			return
		default:
			fmt.Printf("Unknown option: %s\n", os.Args[1])
			printHelp()
			return
		}
	}

	// Run all test suites
	results := runAllTests(args)
	
	// Print summary
	printSummary(results)
	
	// Exit with error if any critical tests failed
	for _, result := range results {
		if result.Suite.Critical && !result.Passed {
			fmt.Printf("\n❌ Critical test suite '%s' failed\n", result.Suite.Name)
			os.Exit(1)
		}
	}
	
	fmt.Println("\n✅ All tests completed successfully")
}

func printHelp() {
	fmt.Println("Usage: go run test_runner.go [option]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  short     Run tests in short mode (skip slow tests)")
	fmt.Println("  verbose   Run tests with verbose output")
	fmt.Println("  race      Run tests with race detection")
	fmt.Println("  cover     Run tests with coverage analysis")
	fmt.Println("  bench     Run benchmarks")
	fmt.Println("  critical  Run only critical tests")
	fmt.Println("  help      Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run test_runner.go            # Run all tests")
	fmt.Println("  go run test_runner.go short      # Quick test run")
	fmt.Println("  go run test_runner.go cover      # Generate coverage report")
}

type TestResult struct {
	Suite    TestSuite
	Passed   bool
	Duration time.Duration
	Output   string
	Error    error
}

func runAllTests(args []string) []TestResult {
	var results []TestResult
	
	for i, suite := range testSuites {
		fmt.Printf("[%d/%d] Running %s tests...\n", i+1, len(testSuites), suite.Name)
		fmt.Printf("  Package: %s\n", suite.Package)
		fmt.Printf("  Description: %s\n", suite.Description)
		
		result := runTestSuite(suite, args)
		results = append(results, result)
		
		if result.Passed {
			fmt.Printf("  ✅ PASSED (%v)\n", result.Duration)
		} else {
			fmt.Printf("  ❌ FAILED (%v)\n", result.Duration)
			if result.Error != nil {
				fmt.Printf("  Error: %v\n", result.Error)
			}
			if result.Output != "" {
				fmt.Printf("  Output:\n%s\n", indentOutput(result.Output))
			}
		}
		fmt.Println()
	}
	
	return results
}

func runCriticalTests() {
	fmt.Println("Running critical tests only...")
	fmt.Println()
	
	var criticalSuites []TestSuite
	for _, suite := range testSuites {
		if suite.Critical {
			criticalSuites = append(criticalSuites, suite)
		}
	}
	
	var failed []string
	for i, suite := range criticalSuites {
		fmt.Printf("[%d/%d] Running %s tests...\n", i+1, len(criticalSuites), suite.Name)
		
		result := runTestSuite(suite, []string{"-short"})
		
		if result.Passed {
			fmt.Printf("  ✅ PASSED (%v)\n", result.Duration)
		} else {
			fmt.Printf("  ❌ FAILED (%v)\n", result.Duration)
			failed = append(failed, suite.Name)
		}
		fmt.Println()
	}
	
	if len(failed) > 0 {
		fmt.Printf("❌ Failed critical tests: %s\n", strings.Join(failed, ", "))
		os.Exit(1)
	} else {
		fmt.Println("✅ All critical tests passed")
	}
}

func runTestSuite(suite TestSuite, args []string) TestResult {
	start := time.Now()
	
	// Check if package has tests
	if !hasTestFiles(suite.Package) {
		return TestResult{
			Suite:    suite,
			Passed:   true, // No tests is considered passing
			Duration: time.Since(start),
			Output:   "No test files found",
		}
	}
	
	// Build command
	cmdArgs := []string{"test"}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, suite.Package)
	
	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1") // Enable for race detection
	
	output, err := cmd.CombinedOutput()
	
	return TestResult{
		Suite:    suite,
		Passed:   err == nil,
		Duration: time.Since(start),
		Output:   string(output),
		Error:    err,
	}
}

func hasTestFiles(packagePath string) bool {
	// Remove ./ prefix if present
	if strings.HasPrefix(packagePath, "./") {
		packagePath = packagePath[2:]
	}
	
	files, err := filepath.Glob(filepath.Join(packagePath, "*_test.go"))
	if err != nil {
		return false
	}
	return len(files) > 0
}

func printSummary(results []TestResult) {
	fmt.Println("=== Test Summary ===")
	fmt.Println()
	
	var passed, failed, skipped int
	var totalDuration time.Duration
	
	for _, result := range results {
		totalDuration += result.Duration
		
		status := ""
		if result.Output == "No test files found" {
			status = "⏭️  SKIPPED"
			skipped++
		} else if result.Passed {
			status = "✅ PASSED"
			passed++
		} else {
			status = "❌ FAILED"
			failed++
		}
		
		critical := ""
		if result.Suite.Critical {
			critical = " (critical)"
		}
		
		fmt.Printf("%-20s %s %8v%s\n", 
			result.Suite.Name, 
			status, 
			result.Duration,
			critical,
		)
	}
	
	fmt.Println()
	fmt.Printf("Total: %d tests, %d passed, %d failed, %d skipped\n", 
		len(results), passed, failed, skipped)
	fmt.Printf("Total time: %v\n", totalDuration)
	
	if failed > 0 {
		fmt.Printf("\n❌ %d test suite(s) failed\n", failed)
	}
}

func indentOutput(output string) string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		lines = append(lines, "    "+scanner.Text())
	}
	return strings.Join(lines, "\n")
}