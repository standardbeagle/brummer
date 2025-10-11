package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JestFramework implements TestFramework for Jest
type JestFramework struct{}

func (j *JestFramework) Name() string { return "jest" }

func (j *JestFramework) Detect() bool {
	// Check for package.json with jest dependency
	if _, err := os.Stat("package.json"); err == nil {
		content, err := os.ReadFile("package.json")
		if err != nil {
			return false
		}

		var pkg struct {
			DevDependencies map[string]string `json:"devDependencies"`
			Dependencies    map[string]string `json:"dependencies"`
			Scripts         map[string]string `json:"scripts"`
		}

		if err := json.Unmarshal(content, &pkg); err != nil {
			return false
		}

		// Check for jest in dependencies or scripts
		if _, has := pkg.DevDependencies["jest"]; has {
			return true
		}
		if _, has := pkg.Dependencies["jest"]; has {
			return true
		}
		for _, script := range pkg.Scripts {
			if strings.Contains(script, "jest") {
				return true
			}
		}
	}

	// Check for jest.config.js or similar files
	configFiles := []string{"jest.config.js", "jest.config.json", "jest.config.ts", "jest.config.mjs"}
	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			return true
		}
	}

	return false
}

func (j *JestFramework) RunTests(pattern string, options TestOptions) (*TestRun, error) {
	// This would execute jest and capture output
	// For now, return a placeholder implementation
	args := []string{"--json", "--outputFile", "/tmp/jest-results.json"}
	if options.Coverage {
		args = append(args, "--coverage")
	}
	if pattern != "" {
		args = append(args, "--testNamePattern", pattern)
	}

	// TODO: Actually execute jest command and parse results
	return &TestRun{
		ID:        generateRunID(),
		Framework: "jest",
		Pattern:   pattern,
		StartTime: time.Now(),
		Status:    TestRunStatusRunning,
		Options:   options,
	}, nil
}

func (j *JestFramework) ParseOutput(output string) (*TestResults, error) {
	// Parse Jest JSON output format
	var jestResults struct {
		NumTotalTests   int `json:"numTotalTests"`
		NumPassedTests  int `json:"numPassedTests"`
		NumFailedTests  int `json:"numFailedTests"`
		NumPendingTests int `json:"numPendingTests"`
		TestResults     []struct {
			Name     string `json:"name"`
			Status   string `json:"status"`
			Message  string `json:"message"`
			Location struct {
				Line   int `json:"line"`
				Column int `json:"column"`
			} `json:"location"`
			Duration int `json:"duration"`
		} `json:"testResults"`
		CoverageMap map[string]interface{} `json:"coverageMap"`
	}

	if err := json.Unmarshal([]byte(output), &jestResults); err != nil {
		return nil, fmt.Errorf("failed to parse jest output: %w", err)
	}

	results := &TestResults{
		TotalTests:   jestResults.NumTotalTests,
		PassedTests:  jestResults.NumPassedTests,
		FailedTests:  jestResults.NumFailedTests,
		SkippedTests: jestResults.NumPendingTests,
		TestCases:    make([]TestCase, 0, len(jestResults.TestResults)),
	}

	for _, tr := range jestResults.TestResults {
		tc := TestCase{
			Name:     tr.Name,
			Status:   parseJestStatus(tr.Status),
			Duration: time.Duration(tr.Duration) * time.Millisecond,
		}

		if tr.Status == "failed" {
			tc.FailureInfo = &TestFailureInfo{
				Message: tr.Message,
				Line:    tr.Location.Line,
				Column:  tr.Location.Column,
			}
		}

		results.TestCases = append(results.TestCases, tc)
	}

	return results, nil
}

func (j *JestFramework) GetCoverage(results *TestResults) (*TestCoverage, error) {
	// Parse Jest coverage output
	// TODO: Implement coverage parsing from jest --coverage output
	return &TestCoverage{
		OverallPct: 75.0,
		Files: []TestCoverageFile{
			{FilePath: "src/app.js", LinesCovered: 120, TotalLines: 150, CoveragePct: 80.0},
		},
	}, nil
}

func parseJestStatus(status string) TestCaseStatus {
	switch status {
	case "passed":
		return TestCaseStatusPassed
	case "failed":
		return TestCaseStatusFailed
	case "pending":
		return TestCaseStatusSkipped
	default:
		return TestCaseStatusPending
	}
}

// VitestFramework implements TestFramework for Vitest
type VitestFramework struct{}

func (v *VitestFramework) Name() string { return "vitest" }

func (v *VitestFramework) Detect() bool {
	// Check for vite.config.js with vitest or vitest.config files
	configFiles := []string{"vitest.config.js", "vitest.config.ts", "vite.config.js", "vite.config.ts"}
	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			if strings.Contains(string(content), "vitest") {
				return true
			}
		}
	}

	// Check package.json for vitest dependency
	if _, err := os.Stat("package.json"); err == nil {
		content, err := os.ReadFile("package.json")
		if err != nil {
			return false
		}

		var pkg struct {
			DevDependencies map[string]string `json:"devDependencies"`
			Dependencies    map[string]string `json:"dependencies"`
		}

		if err := json.Unmarshal(content, &pkg); err == nil {
			if _, has := pkg.DevDependencies["vitest"]; has {
				return true
			}
			if _, has := pkg.Dependencies["vitest"]; has {
				return true
			}
		}
	}

	return false
}

func (v *VitestFramework) RunTests(pattern string, options TestOptions) (*TestRun, error) {
	args := []string{"run", "--reporter=json"}
	if options.Coverage {
		args = append(args, "--coverage")
	}
	if pattern != "" {
		args = append(args, pattern)
	}

	return &TestRun{
		ID:        generateRunID(),
		Framework: "vitest",
		Pattern:   pattern,
		StartTime: time.Now(),
		Status:    TestRunStatusRunning,
		Options:   options,
	}, nil
}

func (v *VitestFramework) ParseOutput(output string) (*TestResults, error) {
	// Vitest JSON output is similar to Jest
	// For now, delegate to Jest parser
	jestParser := &JestFramework{}
	return jestParser.ParseOutput(output)
}

func (v *VitestFramework) GetCoverage(results *TestResults) (*TestCoverage, error) {
	jestParser := &JestFramework{}
	return jestParser.GetCoverage(results)
}

// GoTestFramework implements TestFramework for Go's built-in testing
type GoTestFramework struct{}

func (g *GoTestFramework) Name() string { return "go" }

func (g *GoTestFramework) Detect() bool {
	// Check for Go files and go.mod
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		return false
	}

	// Look for *_test.go files
	testFiles, err := filepath.Glob("*_test.go")
	if err != nil {
		return false
	}

	return len(testFiles) > 0
}

func (g *GoTestFramework) RunTests(pattern string, options TestOptions) (*TestRun, error) {
	args := []string{"test", "-v", "-json"}
	if options.Coverage {
		args = append(args, "-cover")
	}
	if pattern != "" {
		args = append(args, "-run", pattern)
	}

	return &TestRun{
		ID:        generateRunID(),
		Framework: "go",
		Pattern:   pattern,
		StartTime: time.Now(),
		Status:    TestRunStatusRunning,
		Options:   options,
	}, nil
}

func (g *GoTestFramework) ParseOutput(output string) (*TestResults, error) {
	results := &TestResults{
		TestCases: make([]TestCase, 0),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var goTestEvent struct {
			Action  string  `json:"Action"`
			Test    string  `json:"Test"`
			Package string  `json:"Package"`
			Time    float64 `json:"Time"`
			Output  string  `json:"Output"`
		}

		if err := json.Unmarshal([]byte(line), &goTestEvent); err != nil {
			continue
		}

		switch goTestEvent.Action {
		case "run":
			// Test started
			tc := TestCase{
				Name:      goTestEvent.Test,
				FilePath:  goTestEvent.Package,
				Status:    TestCaseStatusRunning,
				StartTime: time.Now(),
			}
			results.TestCases = append(results.TestCases, tc)

		case "pass":
			// Test passed
			for i := range results.TestCases {
				if results.TestCases[i].Name == goTestEvent.Test {
					results.TestCases[i].Status = TestCaseStatusPassed
					results.TestCases[i].Duration = time.Duration(goTestEvent.Time * float64(time.Second))
					results.TestCases[i].EndTime = time.Now()
					results.PassedTests++
					results.TotalTests++
					break
				}
			}

		case "fail":
			// Test failed
			for i := range results.TestCases {
				if results.TestCases[i].Name == goTestEvent.Test {
					results.TestCases[i].Status = TestCaseStatusFailed
					results.TestCases[i].Duration = time.Duration(goTestEvent.Time * float64(time.Second))
					results.TestCases[i].EndTime = time.Now()
					results.TestCases[i].FailureInfo = &TestFailureInfo{
						Message: goTestEvent.Output,
					}
					results.FailedTests++
					results.TotalTests++
					break
				}
			}

		case "skip":
			// Test skipped
			for i := range results.TestCases {
				if results.TestCases[i].Name == goTestEvent.Test {
					results.TestCases[i].Status = TestCaseStatusSkipped
					results.TestCases[i].EndTime = time.Now()
					results.SkippedTests++
					break
				}
			}
		}
	}

	return results, nil
}

func (g *GoTestFramework) GetCoverage(results *TestResults) (*TestCoverage, error) {
	// Parse go test -cover output
	return &TestCoverage{
		OverallPct: 0.0, // Would be parsed from coverage output
	}, nil
}

// PytestFramework implements TestFramework for pytest
type PytestFramework struct{}

func (p *PytestFramework) Name() string { return "pytest" }

func (p *PytestFramework) Detect() bool {
	// Check for Python test files
	patterns := []string{"test_*.py", "*_test.py"}
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err == nil && len(files) > 0 {
			return true
		}
	}

	// Check for pytest.ini, pyproject.toml, or setup.cfg with pytest config
	configFiles := []string{"pytest.ini", "pyproject.toml", "setup.cfg", "tox.ini"}
	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			if strings.Contains(string(content), "pytest") {
				return true
			}
		}
	}

	return false
}

func (p *PytestFramework) RunTests(pattern string, options TestOptions) (*TestRun, error) {
	args := []string{"--json-report", "--json-report-file=/tmp/pytest-results.json"}
	if options.Coverage {
		args = append(args, "--cov")
	}
	if pattern != "" {
		args = append(args, "-k", pattern)
	}

	return &TestRun{
		ID:        generateRunID(),
		Framework: "pytest",
		Pattern:   pattern,
		StartTime: time.Now(),
		Status:    TestRunStatusRunning,
		Options:   options,
	}, nil
}

func (p *PytestFramework) ParseOutput(output string) (*TestResults, error) {
	// Parse pytest JSON output
	var pytestResults struct {
		Summary struct {
			Total   int `json:"total"`
			Passed  int `json:"passed"`
			Failed  int `json:"failed"`
			Skipped int `json:"skipped"`
		} `json:"summary"`
		Tests []struct {
			NodeID   string  `json:"nodeid"`
			Outcome  string  `json:"outcome"`
			Duration float64 `json:"duration"`
			Call     struct {
				Longrepr string `json:"longrepr"`
			} `json:"call"`
		} `json:"tests"`
	}

	if err := json.Unmarshal([]byte(output), &pytestResults); err != nil {
		return nil, fmt.Errorf("failed to parse pytest output: %w", err)
	}

	results := &TestResults{
		TotalTests:   pytestResults.Summary.Total,
		PassedTests:  pytestResults.Summary.Passed,
		FailedTests:  pytestResults.Summary.Failed,
		SkippedTests: pytestResults.Summary.Skipped,
		TestCases:    make([]TestCase, 0, len(pytestResults.Tests)),
	}

	for _, test := range pytestResults.Tests {
		tc := TestCase{
			Name:     extractTestName(test.NodeID),
			FilePath: extractFilePath(test.NodeID),
			Status:   parsePytestOutcome(test.Outcome),
			Duration: time.Duration(test.Duration * float64(time.Second)),
		}

		if test.Outcome == "failed" && test.Call.Longrepr != "" {
			tc.FailureInfo = &TestFailureInfo{
				Message: test.Call.Longrepr,
			}
		}

		results.TestCases = append(results.TestCases, tc)
	}

	return results, nil
}

func (p *PytestFramework) GetCoverage(results *TestResults) (*TestCoverage, error) {
	// Parse pytest-cov output
	return &TestCoverage{
		OverallPct: 0.0, // Would be parsed from coverage output
	}, nil
}

func parsePytestOutcome(outcome string) TestCaseStatus {
	switch outcome {
	case "passed":
		return TestCaseStatusPassed
	case "failed":
		return TestCaseStatusFailed
	case "skipped":
		return TestCaseStatusSkipped
	default:
		return TestCaseStatusPending
	}
}

func extractTestName(nodeID string) string {
	// Extract test name from pytest node ID
	// Format: file.py::ClassName::test_method
	parts := strings.Split(nodeID, "::")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return nodeID
}

func extractFilePath(nodeID string) string {
	// Extract file path from pytest node ID
	parts := strings.Split(nodeID, "::")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// TestRunnerService manages test execution and framework detection
type TestRunnerService struct {
	manager    *TestManager
	frameworks map[string]TestFramework
}

// NewTestRunnerService creates a new test runner service
func NewTestRunnerService() *TestRunnerService {
	service := &TestRunnerService{
		manager:    NewTestManager(),
		frameworks: make(map[string]TestFramework),
	}

	// Register built-in frameworks
	service.RegisterFramework(&JestFramework{})
	service.RegisterFramework(&VitestFramework{})
	service.RegisterFramework(&GoTestFramework{})
	service.RegisterFramework(&PytestFramework{})

	return service
}

// RegisterFramework registers a test framework
func (trs *TestRunnerService) RegisterFramework(framework TestFramework) {
	trs.frameworks[framework.Name()] = framework
	trs.manager.RegisterFramework(framework)
}

// DetectAvailableFrameworks returns all detected frameworks in current directory
func (trs *TestRunnerService) DetectAvailableFrameworks() []TestFramework {
	return trs.manager.DetectFrameworks()
}

// RunTests executes tests using the best available framework
func (trs *TestRunnerService) RunTests(pattern string, options TestOptions) (*TestRun, error) {
	detected := trs.DetectAvailableFrameworks()
	if len(detected) == 0 {
		return nil, fmt.Errorf("no test frameworks detected")
	}

	// Use the first detected framework
	framework := detected[0]
	return trs.manager.StartTestRun(framework.Name(), pattern, options)
}

// RunTestsWithFramework executes tests using a specific framework
func (trs *TestRunnerService) RunTestsWithFramework(frameworkName, pattern string, options TestOptions) (*TestRun, error) {
	return trs.manager.StartTestRun(frameworkName, pattern, options)
}

// GetManager returns the underlying test manager
func (trs *TestRunnerService) GetManager() *TestManager {
	return trs.manager
}
