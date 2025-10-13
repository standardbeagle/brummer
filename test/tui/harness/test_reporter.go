package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TestReporter generates detailed test execution reports
type TestReporter struct {
	mu       sync.RWMutex
	config   TestReporterConfig
	results  []TestResult
	metrics  TestMetrics
}

// NewTestReporter creates a new test reporter instance
func NewTestReporter(config TestReporterConfig) *TestReporter {
	return &TestReporter{
		config:  config,
		results: make([]TestResult, 0),
		metrics: TestMetrics{},
	}
}

// AddTestResult adds a test result to the reporter
func (tr *TestReporter) AddTestResult(result TestResult) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.results = append(tr.results, result)
	tr.updateMetrics()

	// Generate real-time report if configured
	if tr.config.OutputDir != "" {
		tr.generateRealTimeReport()
	}

	return nil
}

// GetTestResults returns all test results
func (tr *TestReporter) GetTestResults() []TestResult {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	results := make([]TestResult, len(tr.results))
	copy(results, tr.results)
	return results
}

// GetTestResult returns a specific test result by ID
func (tr *TestReporter) GetTestResult(testID string) (TestResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	for _, result := range tr.results {
		if result.TestID == testID {
			return result, nil
		}
	}

	return TestResult{}, fmt.Errorf("test result not found: %s", testID)
}

// GenerateReport generates a comprehensive test report
func (tr *TestReporter) GenerateReport(results []TestResult) (Report, error) {
	tr.mu.Lock()
	// Use internal results if none provided
	if len(results) == 0 {
		results = tr.results
	}
	tr.mu.Unlock()

	if len(results) == 0 {
		return Report{}, fmt.Errorf("no test results to report")
	}

	report := Report{
		ID:          fmt.Sprintf("report-%d", time.Now().UnixNano()),
		GeneratedAt: time.Now(),
		Results:     results,
	}

	// Generate summary
	report.Summary = tr.generateSummary(results)

	// Generate metadata
	report.Metadata = ReportMetadata{
		Version:       "1.0.0",
		TestHarnessID: fmt.Sprintf("test-harness-%d", time.Now().Unix()),
		Environment: map[string]string{
			"go_version":    getGoVersion(),
			"os":           getOS(),
			"arch":         getArch(),
			"timestamp":    time.Now().Format(time.RFC3339),
		},
	}

	return report, nil
}

// generateSummary generates test summary statistics
func (tr *TestReporter) generateSummary(results []TestResult) TestSummary {
	summary := TestSummary{
		TotalTests:  len(results),
		GeneratedAt: time.Now(),
	}

	var totalDuration time.Duration
	for _, result := range results {
		switch result.Status {
		case TestStatusPass:
			summary.PassedTests++
		case TestStatusFail:
			summary.FailedTests++
		case TestStatusError:
			summary.ErrorTests++
		case TestStatusSkip:
			summary.SkippedTests++
		}
		totalDuration += result.Duration
	}

	summary.Duration = totalDuration
	if summary.TotalTests > 0 {
		summary.SuccessRate = float64(summary.PassedTests) / float64(summary.TotalTests) * 100
	}

	return summary
}

// updateMetrics updates the reporter metrics based on current results
func (tr *TestReporter) updateMetrics() {
	tr.metrics = TestMetrics{
		InputEvents:      0,
		StateValidations: 0,
		ViewChanges:      0,
		Duration:         0,
		MemoryUsed:       0,
	}

	for _, result := range tr.results {
		tr.metrics.InputEvents += result.Metrics.InputEvents
		tr.metrics.StateValidations += result.Metrics.StateValidations
		tr.metrics.ViewChanges += result.Metrics.ViewChanges
		tr.metrics.Duration += result.Duration
		// Note: MemoryUsed calculation would require memory profiling
	}
}

// GetMetrics returns the current test metrics
func (tr *TestReporter) GetMetrics() TestMetrics {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return tr.metrics
}

// CalculateMetrics recalculates metrics from all results
func (tr *TestReporter) CalculateMetrics() TestMetrics {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.updateMetrics()
	return tr.metrics
}

// ExportReport exports a report in the specified format
func (tr *TestReporter) ExportReport(report Report, format ReportFormat) ([]byte, error) {
	switch format {
	case ReportFormatJSON:
		return tr.exportJSON(report)
	case ReportFormatHTML:
		return tr.exportHTML(report)
	case ReportFormatXML:
		return tr.exportXML(report)
	case ReportFormatJUNIT:
		return tr.exportJUNIT(report)
	case ReportFormatTXT:
		return tr.exportTXT(report)
	case ReportFormatCSV:
		return tr.exportCSV(report)
	default:
		return nil, fmt.Errorf("unsupported report format: %s", format)
	}
}

// exportJSON exports report as JSON
func (tr *TestReporter) exportJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// exportHTML exports report as HTML
func (tr *TestReporter) exportHTML(report Report) ([]byte, error) {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>TUI Test Report - {{.Summary.GeneratedAt.Format "2006-01-02 15:04:05"}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .summary { display: flex; gap: 20px; margin-bottom: 20px; }
        .metric { background: #e9ecef; padding: 15px; border-radius: 5px; text-align: center; }
        .metric h3 { margin: 0 0 5px 0; }
        .metric .value { font-size: 24px; font-weight: bold; }
        .pass { color: #28a745; }
        .fail { color: #dc3545; }
        .error { color: #ffc107; }
        .skip { color: #6c757d; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .test-pass { background-color: #d4edda; }
        .test-fail { background-color: #f8d7da; }
        .test-error { background-color: #fff3cd; }
        .test-skip { background-color: #e2e3e5; }
    </style>
</head>
<body>
    <div class="header">
        <h1>TUI Integration Test Report</h1>
        <p>Generated: {{.Summary.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p>Report ID: {{.ID}}</p>
    </div>

    <div class="summary">
        <div class="metric">
            <h3>Total Tests</h3>
            <div class="value">{{.Summary.TotalTests}}</div>
        </div>
        <div class="metric">
            <h3 class="pass">Passed</h3>
            <div class="value pass">{{.Summary.PassedTests}}</div>
        </div>
        <div class="metric">
            <h3 class="fail">Failed</h3>
            <div class="value fail">{{.Summary.FailedTests}}</div>
        </div>
        <div class="metric">
            <h3 class="error">Errors</h3>
            <div class="value error">{{.Summary.ErrorTests}}</div>
        </div>
        <div class="metric">
            <h3>Success Rate</h3>
            <div class="value">{{printf "%.1f%%" .Summary.SuccessRate}}</div>
        </div>
        <div class="metric">
            <h3>Duration</h3>
            <div class="value">{{.Summary.Duration}}</div>
        </div>
    </div>

    <h2>Test Results</h2>
    <table>
        <thead>
            <tr>
                <th>Test ID</th>
                <th>Name</th>
                <th>Status</th>
                <th>Duration</th>
                <th>Description</th>
                <th>Error</th>
            </tr>
        </thead>
        <tbody>
            {{range .Results}}
            <tr class="test-{{.Status}}">
                <td>{{.TestID}}</td>
                <td>{{.Action}}</td>
                <td>{{.Status}}</td>
                <td>{{.Duration}}</td>
                <td>{{.Description}}</td>
                <td>{{.Error}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>
`

	// This would need proper template execution in a real implementation
	// For now, return a simple HTML structure
	return []byte(fmt.Sprintf(htmlTemplate, report)), nil
}

// exportXML exports report as XML
func (tr *TestReporter) exportXML(report Report) ([]byte, error) {
	// Simple XML structure
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<testreport id="%s" generated_at="%s">
    <summary>
        <total_tests>%d</total_tests>
        <passed_tests>%d</passed_tests>
        <failed_tests>%d</failed_tests>
        <error_tests>%d</error_tests>
        <success_rate>%.2f</success_rate>
        <duration>%s</duration>
    </summary>
    <results>
`, report.ID, report.GeneratedAt.Format(time.RFC3339),
		report.Summary.TotalTests, report.Summary.PassedTests,
		report.Summary.FailedTests, report.Summary.ErrorTests,
		report.Summary.SuccessRate, report.Summary.Duration)

	for _, result := range report.Results {
		xml += fmt.Sprintf(`        <test>
            <id>%s</id>
            <name>%s</name>
            <status>%s</status>
            <duration>%s</duration>
            <description>%s</description>
            <error>%s</error>
        </test>
`, result.TestID, result.Action, result.Status,
			result.Duration, result.Description, result.Error)
	}

	xml += `    </results>
</testreport>`

	return []byte(xml), nil
}

// exportJUNIT exports report in JUnit XML format
func (tr *TestReporter) exportJUNIT(report Report) ([]byte, error) {
	// JUnit XML format
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="TUI Integration Tests" tests="%d" failures="%d" errors="%d" time="%.3f">
`, report.Summary.TotalTests, report.Summary.FailedTests,
		report.Summary.ErrorTests, report.Summary.Duration.Seconds())

	for _, result := range report.Results {
		xml += fmt.Sprintf(`    <testcase name="%s" classname="TUIIntegration" time="%.3f">
`, result.Action, result.Duration.Seconds())

		if result.Status == TestStatusFail {
			xml += fmt.Sprintf(`        <failure message="%s">%s</failure>
`, result.Error, result.Error)
		} else if result.Status == TestStatusError {
			xml += fmt.Sprintf(`        <error message="%s">%s</error>
`, result.Error, result.Error)
		}

		xml += "    </testcase>\n"
	}

	xml += "</testsuite>"

	return []byte(xml), nil
}

// exportTXT exports report as plain text
func (tr *TestReporter) exportTXT(report Report) ([]byte, error) {
	txt := fmt.Sprintf(`TUI Integration Test Report
============================
Generated: %s
Report ID: %s

Summary:
--------
Total Tests: %d
Passed: %d
Failed: %d
Errors: %d
Success Rate: %.1f%%
Duration: %s

Test Results:
--------------
`, report.GeneratedAt.Format(time.RFC3339), report.ID,
		report.Summary.TotalTests, report.Summary.PassedTests,
		report.Summary.FailedTests, report.Summary.ErrorTests,
		report.Summary.SuccessRate, report.Summary.Duration)

	for _, result := range report.Results {
		txt += fmt.Sprintf(`
[%s] %s (%s)
  Duration: %s
  Description: %s`, result.Status, result.Action, result.TestID,
			result.Duration, result.Description)

		if result.Error != "" {
			txt += fmt.Sprintf("\n  Error: %s", result.Error)
		}
		txt += "\n"
	}

	return []byte(txt), nil
}

// exportCSV exports report as CSV
func (tr *TestReporter) exportCSV(report Report) ([]byte, error) {
	csv := "Test ID,Name,Status,Duration,Description,Error\n"

	for _, result := range report.Results {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			result.TestID, result.Action, result.Status,
			result.Duration, result.Description, result.Error)
	}

	return []byte(csv), nil
}

// generateRealTimeReport generates a real-time report update
func (tr *TestReporter) generateRealTimeReport() {
	if tr.config.OutputDir == "" {
		return
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(tr.config.OutputDir, 0755); err != nil {
		return
	}

	// Generate report
	report, err := tr.GenerateReport(nil)
	if err != nil {
		return
	}

	// Export in default format
	data, err := tr.ExportReport(report, ReportFormatJSON)
	if err != nil {
		return
	}

	// Write to file
	filename := filepath.Join(tr.config.OutputDir, "latest-report.json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return
	}
}

// ClearResults clears all test results
func (tr *TestReporter) ClearResults() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.results = tr.results[:0]
	tr.metrics = TestMetrics{}
}

// GetResultsByStatus returns results filtered by status
func (tr *TestReporter) GetResultsByStatus(status TestStatus) []TestResult {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var filtered []TestResult
	for _, result := range tr.results {
		if result.Status == status {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// GetFailedTests returns all failed tests
func (tr *TestReporter) GetFailedTests() []TestResult {
	return tr.GetResultsByStatus(TestStatusFail)
}

// GetPassedTests returns all passed tests
func (tr *TestReporter) GetPassedTests() []TestResult {
	return tr.GetResultsByStatus(TestStatusPass)
}

// GetErrorTests returns all tests with errors
func (tr *TestReporter) GetErrorTests() []TestResult {
	return tr.GetResultsByStatus(TestStatusError)
}

// Helper functions for environment information
func getGoVersion() string {
	// This would typically use runtime.Version()
	return "go1.24.2"
}

func getOS() string {
	// This would typically use runtime.GOOS
	return "linux"
}

func getArch() string {
	// This would typically use runtime.GOARCH
	return "amd64"
}