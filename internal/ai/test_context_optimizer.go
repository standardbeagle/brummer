package ai

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/test"
)

// TestContextOptimizer optimizes test context for AI agents
type TestContextOptimizer struct {
	testRunner  *test.TestRunnerService
	eventBus    EventBus
	cache       map[string]*CachedTestContext
	cacheTTL    time.Duration
}

// CachedTestContext represents cached test context data
type CachedTestContext struct {
	Context     *TestContextData `json:"context"`
	Timestamp   time.Time       `json:"timestamp"`
	RunID       string          `json:"run_id"`
}

// TestContextData contains optimized test context for AI agents
type TestContextData struct {
	// Compact summary for passing tests
	PassingTestsSummary *PassingTestsSummary `json:"passing_tests_summary,omitempty"`

	// Detailed context for failing tests
	FailingTests         []DetailedFailure    `json:"failing_tests,omitempty"`

	// Changed files with relevant test results
	ChangedFilesContext  []FileTestContext    `json:"changed_files_context,omitempty"`

	// Test coverage gaps
	CoverageGaps         []CoverageGap        `json:"coverage_gaps,omitempty"`

	// Performance issues
	PerformanceIssues    []PerformanceIssue   `json:"performance_issues,omitempty"`

	// Recommendations
	Recommendations      []TestRecommendation `json:"recommendations,omitempty"`
}

// PassingTestsSummary provides a compact summary of passing tests
type PassingTestsSummary struct {
	TotalPassed     int      `json:"total_passed"`
	PassRate        float64  `json:"pass_rate"`
	TestFrameworks  []string `json:"test_frameworks"`
	LastRunDuration string   `json:"last_run_duration"`
}

// DetailedFailure contains detailed information about a test failure
type DetailedFailure struct {
	TestName        string   `json:"test_name"`
	FilePath        string   `json:"file_path"`
	Line            int      `json:"line,omitempty"`
	Column          int      `json:"column,omitempty"`
	ErrorMessage    string   `json:"error_message"`
	StackTrace      []string `json:"stack_trace"`
	SourceContext   []string `json:"source_context"`
	RelatedFiles    []string `json:"related_files"`
	LastFailed      time.Time `json:"last_failed"`
	FailureCount    int      `json:"failure_count"`
	ErrorType       string   `json:"error_type"`
}

// FileTestContext provides test context for a specific file
type FileTestContext struct {
	FilePath         string            `json:"file_path"`
	TestStatus       string            `json:"test_status"`
	RelatedTests     []string          `json:"related_tests"`
	Coverage         float64           `json:"coverage"`
	LastTested       time.Time         `json:"last_tested"`
	RecentChanges    []FileChange      `json:"recent_changes"`
	ImpactScore      float64           `json:"impact_score"`
}

// CoverageGap represents a coverage gap
type CoverageGap struct {
	FilePath         string            `json:"file_path"`
	UncoveredLines   []int             `json:"uncovered_lines"`
	CoveragePct      float64           `json:"coverage_pct"`
	RiskLevel        string            `json:"risk_level"`
	SuggestedTests   []string          `json:"suggested_tests"`
}

// PerformanceIssue represents a performance problem
type PerformanceIssue struct {
	TestName         string            `json:"test_name"`
	FilePath         string            `json:"file_path"`
	Duration         time.Duration     `json:"duration"`
	SlowThreshold    time.Duration     `json:"slow_threshold"`
	ImpactScore      float64           `json:"impact_score"`
	Suggestions      []string          `json:"suggestions"`
}

// TestRecommendation provides actionable recommendations
type TestRecommendation struct {
	Type             string            `json:"type"`
	Priority         string            `json:"priority"`
	Description      string            `json:"description"`
	Action           string            `json:"action"`
	TargetFiles      []string          `json:"target_files"`
	EstimatedEffort  string            `json:"estimated_effort"`
}

// FileChange represents a recent change to a file
type FileChange struct {
	Timestamp    time.Time `json:"timestamp"`
	ChangeType   string    `json:"change_type"` // "added", "modified", "deleted"
	LinesAdded   int       `json:"lines_added"`
	LinesRemoved int       `json:"lines_removed"`
}

// NewTestContextOptimizer creates a new test context optimizer
func NewTestContextOptimizer(testRunner *test.TestRunnerService, eventBus EventBus) *TestContextOptimizer {
	return &TestContextOptimizer{
		testRunner: testRunner,
		eventBus:   eventBus,
		cache:      make(map[string]*CachedTestContext),
		cacheTTL:   5 * time.Minute,
	}
}

// GetOptimizedContext returns optimized test context for AI agents
func (tco *TestContextOptimizer) GetOptimizedContext(changedFiles []string, includePassing bool) (*TestContextData, error) {
	cacheKey := tco.generateCacheKey(changedFiles, includePassing)

	// Check cache first
	if cached, found := tco.cache[cacheKey]; found && time.Since(cached.Timestamp) < tco.cacheTTL {
		return cached.Context, nil
	}

	manager := tco.testRunner.GetManager()
	activeRuns := manager.GetActiveRuns()

	if len(activeRuns) == 0 {
		return &TestContextData{}, nil
	}

	// Get the most recent run with results
	var latestRun *test.TestRun
	for _, run := range activeRuns {
		if run.Results != nil {
			if latestRun == nil || run.StartTime.After(latestRun.StartTime) {
				latestRun = run
			}
		}
	}

	if latestRun == nil || latestRun.Results == nil {
		return &TestContextData{}, nil
	}

	context := &TestContextData{
		ChangedFilesContext: tco.buildFileContext(changedFiles, latestRun.Results),
		CoverageGaps:        tco.identifyCoverageGaps(latestRun.Results),
		PerformanceIssues:  tco.identifyPerformanceIssues(latestRun.Results),
		Recommendations:     tco.generateRecommendations(latestRun.Results, changedFiles),
	}

	// Add passing tests summary if requested
	if includePassing {
		context.PassingTestsSummary = tco.buildPassingSummary(latestRun.Results)
	}

	// Add detailed failure information if there are failures
	if latestRun.Results.FailedTests > 0 {
		context.FailingTests = tco.buildDetailedFailures(latestRun.Results)
	}

	// Cache the result
	tco.cache[cacheKey] = &CachedTestContext{
		Context:   context,
		Timestamp: time.Now(),
		RunID:     latestRun.ID,
	}

	return context, nil
}

// buildPassingSummary creates a compact summary of passing tests
func (tco *TestContextOptimizer) buildPassingSummary(results *test.TestResults) *PassingTestsSummary {
	passRate := 0.0
	if results.TotalTests > 0 {
		passRate = float64(results.PassedTests) / float64(results.TotalTests) * 100
	}

	return &PassingTestsSummary{
		TotalPassed:      results.PassedTests,
		PassRate:         passRate,
		TestFrameworks:   []string{"unknown"}, // Would be enhanced with actual framework detection
		LastRunDuration:  fmt.Sprintf("%.2fs", time.Duration(results.TotalTests)*time.Millisecond/time.Second),
	}
}

// buildDetailedFailures creates detailed failure information
func (tco *TestContextOptimizer) buildDetailedFailures(results *test.TestResults) []DetailedFailure {
	failures := make([]DetailedFailure, 0, results.FailedTests)

	for _, tc := range results.TestCases {
		if tc.Status == test.TestCaseStatusFailed && tc.FailureInfo != nil {
			failure := DetailedFailure{
				TestName:     tc.Name,
				FilePath:     tc.FilePath,
				Line:         tc.FailureInfo.Line,
				Column:       tc.FailureInfo.Column,
				ErrorMessage: tc.FailureInfo.Message,
				StackTrace:   tc.FailureInfo.StackTrace,
				LastFailed:   tc.EndTime,
				FailureCount: 1, // Would be enhanced with historical data
				ErrorType:    tco.categorizeError(tc.FailureInfo.Message),
			}

			// Add source context
			failure.SourceContext = tco.getSourceContext(tc.FilePath, tc.FailureInfo.Line)

			// Find related files
			failure.RelatedFiles = tco.findRelatedFiles(tc.FilePath)

			failures = append(failures, failure)
		}
	}

	return failures
}

// buildFileContext creates test context for changed files
func (tco *TestContextOptimizer) buildFileContext(changedFiles []string, results *test.TestResults) []FileTestContext {
	context := make([]FileTestContext, 0, len(changedFiles))

	for _, filePath := range changedFiles {
		fileContext := FileTestContext{
			FilePath:     filePath,
			RelatedTests: tco.findTestsForFile(filePath, results),
			ImpactScore:  tco.calculateImpactScore(filePath, results),
		}

		// Add coverage info if available
		if results.Coverage != nil {
			for _, coverageFile := range results.Coverage.Files {
				if coverageFile.FilePath == filePath {
					fileContext.Coverage = coverageFile.CoveragePct
					break
				}
			}
		}

		// Determine test status
		if len(fileContext.RelatedTests) == 0 {
			fileContext.TestStatus = "no_tests"
		} else if fileContext.Coverage > 80 {
			fileContext.TestStatus = "well_covered"
		} else if fileContext.Coverage > 0 {
			fileContext.TestStatus = "partially_covered"
		} else {
			fileContext.TestStatus = "uncovered"
		}

		context = append(context, fileContext)
	}

	return context
}

// identifyCoverageGaps identifies files with low test coverage
func (tco *TestContextOptimizer) identifyCoverageGaps(results *test.TestResults) []CoverageGap {
	gaps := []CoverageGap{}

	if results.Coverage == nil {
		return gaps
	}

	for _, coverageFile := range results.Coverage.Files {
		if coverageFile.CoveragePct < 60 { // Threshold for "low coverage"
			gap := CoverageGap{
				FilePath:       coverageFile.FilePath,
				UncoveredLines: []int{}, // Would be enhanced with actual uncovered lines
				CoveragePct:    coverageFile.CoveragePct,
				SuggestedTests: tco.suggestTestsForFile(coverageFile.FilePath),
			}

			// Determine risk level
			if coverageFile.CoveragePct < 20 {
				gap.RiskLevel = "high"
			} else if coverageFile.CoveragePct < 40 {
				gap.RiskLevel = "medium"
			} else {
				gap.RiskLevel = "low"
			}

			gaps = append(gaps, gap)
		}
	}

	return gaps
}

// identifyPerformanceIssues identifies slow tests
func (tco *TestContextOptimizer) identifyPerformanceIssues(results *test.TestResults) []PerformanceIssue {
	issues := []PerformanceIssue{}

	if results.Performance == nil {
		return issues
	}

	for _, slowTest := range results.Performance.SlowTests {
		issue := PerformanceIssue{
			TestName:      slowTest.TestName,
			FilePath:      slowTest.FilePath,
			Duration:      slowTest.Duration,
			SlowThreshold: slowTest.SlowThreshold,
			ImpactScore:   tco.calculatePerformanceImpact(slowTest),
			Suggestions:   tco.suggestPerformanceFixes(slowTest),
		}

		issues = append(issues, issue)
	}

	return issues
}

// generateRecommendations generates actionable recommendations
func (tco *TestContextOptimizer) generateRecommendations(results *test.TestResults, changedFiles []string) []TestRecommendation {
	recommendations := []TestRecommendation{}

	// Recommend tests for changed files without coverage
	for _, filePath := range changedFiles {
		hasTests := false
		if results.Coverage != nil {
			for _, coverageFile := range results.Coverage.Files {
				if coverageFile.FilePath == filePath && coverageFile.CoveragePct > 0 {
					hasTests = true
					break
				}
			}
		}

		if !hasTests {
			recommendations = append(recommendations, TestRecommendation{
				Type:            "add_tests",
				Priority:        "high",
				Description:     fmt.Sprintf("File %s has no test coverage", filepath.Base(filePath)),
				Action:          "Write unit tests for this file",
				TargetFiles:     []string{filePath},
				EstimatedEffort: "2-4 hours",
			})
		}
	}

	// Recommend fixing failing tests
	if results.FailedTests > 0 {
		recommendations = append(recommendations, TestRecommendation{
			Type:            "fix_failures",
			Priority:        "critical",
			Description:     fmt.Sprintf("%d tests are failing", results.FailedTests),
			Action:          "Investigate and fix failing tests",
			TargetFiles:     tco.getFilesWithFailures(results),
			EstimatedEffort: "1-3 hours",
		})
	}

	// Recommend performance improvements
	if results.Performance != nil && len(results.Performance.SlowTests) > 0 {
		recommendations = append(recommendations, TestRecommendation{
			Type:            "optimize_performance",
			Priority:        "medium",
			Description:     fmt.Sprintf("%d tests are slow", len(results.Performance.SlowTests)),
			Action:          "Optimize slow test execution",
			TargetFiles:     tco.getFilesWithSlowTests(results),
			EstimatedEffort: "1-2 hours",
		})
	}

	return recommendations
}

// Helper methods

func (tco *TestContextOptimizer) generateCacheKey(changedFiles []string, includePassing bool) string {
	return fmt.Sprintf("%s-%v", strings.Join(changedFiles, ","), includePassing)
}

func (tco *TestContextOptimizer) categorizeError(message string) string {
	message = strings.ToLower(message)

	if strings.Contains(message, "assert") || strings.Contains(message, "expect") {
		return "assertion"
	} else if strings.Contains(message, "timeout") {
		return "timeout"
	} else if strings.Contains(message, "import") || strings.Contains(message, "module") {
		return "import"
	} else if strings.Contains(message, "null") || strings.Contains(message, "undefined") {
		return "null_reference"
	} else if strings.Contains(message, "type") || strings.Contains(message, "cannot") {
		return "type_error"
	}

	return "unknown"
}

func (tco *TestContextOptimizer) getSourceContext(filePath string, line int) []string {
	// TODO: Implement source context extraction
	// This would read the source file and extract lines around the error
	return []string{
		fmt.Sprintf("// Context around line %d in %s", line, filePath),
		"// Source context extraction not implemented yet",
	}
}

func (tco *TestContextOptimizer) findRelatedFiles(filePath string) []string {
	// TODO: Implement related file detection
	// This would analyze imports, dependencies, and file relationships
	return []string{
		filepath.Dir(filePath) + "/related_file.js",
	}
}

func (tco *TestContextOptimizer) findTestsForFile(filePath string, results *test.TestResults) []string {
	tests := []string{}

	for _, tc := range results.TestCases {
		// Simple heuristic: if test file path is related to source file path
		testFileName := filepath.Base(tc.FilePath)
		sourceFileName := filepath.Base(filePath)

		if strings.Contains(testFileName, strings.TrimSuffix(sourceFileName, filepath.Ext(sourceFileName))) {
			tests = append(tests, tc.Name)
		}
	}

	return tests
}

func (tco *TestContextOptimizer) calculateImpactScore(filePath string, results *test.TestResults) float64 {
	// Simple impact calculation based on related test failures
	score := 0.0
	relatedTests := tco.findTestsForFile(filePath, results)

	for _, testName := range relatedTests {
		for _, tc := range results.TestCases {
			if tc.Name == testName && tc.Status == test.TestCaseStatusFailed {
				score += 10.0
			} else if tc.Name == testName {
				score += 1.0
			}
		}
	}

	return score
}

func (tco *TestContextOptimizer) suggestTestsForFile(filePath string) []string {
	// TODO: Implement intelligent test suggestions
	return []string{
		fmt.Sprintf("test_%s", filepath.Base(filePath)),
		fmt.Sprintf("%s.test", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))),
	}
}

func (tco *TestContextOptimizer) calculatePerformanceImpact(slowTest test.TestPerformanceEntry) float64 {
	// Calculate impact based on how much slower than threshold
	ratio := float64(slowTest.Duration) / float64(slowTest.SlowThreshold)
	return ratio * 5.0 // Arbitrary impact score
}

func (tco *TestContextOptimizer) suggestPerformanceFixes(slowTest test.TestPerformanceEntry) []string {
	suggestions := []string{
		"Consider mocking expensive operations",
		"Reduce test data size",
		"Optimize setup and teardown",
		"Use parallel test execution if possible",
	}

	return suggestions
}

func (tco *TestContextOptimizer) getFilesWithFailures(results *test.TestResults) []string {
	files := []string{}
	seen := make(map[string]bool)

	for _, tc := range results.TestCases {
		if tc.Status == test.TestCaseStatusFailed {
			if !seen[tc.FilePath] {
				files = append(files, tc.FilePath)
				seen[tc.FilePath] = true
			}
		}
	}

	return files
}

func (tco *TestContextOptimizer) getFilesWithSlowTests(results *test.TestResults) []string {
	files := []string{}
	seen := make(map[string]bool)

	if results.Performance != nil {
		for _, slowTest := range results.Performance.SlowTests {
			if !seen[slowTest.FilePath] {
				files = append(files, slowTest.FilePath)
				seen[slowTest.FilePath] = true
			}
		}
	}

	return files
}

// EventBus interface for test context optimizer
type EventBus interface {
	Publish(message interface{})
}

// PublishTestContextEvent publishes optimized test context events
func (tco *TestContextOptimizer) PublishTestContextEvent(changedFiles []string, context *TestContextData) {
	if tco.eventBus != nil {
		// Publish event for AI agents to consume
		event := map[string]interface{}{
			"type":         "test_context_optimized",
			"changed_files": changedFiles,
			"context":      context,
			"timestamp":    time.Now(),
		}

		tco.eventBus.Publish(event)
	}
}

// ClearCache clears the internal cache
func (tco *TestContextOptimizer) ClearCache() {
	tco.cache = make(map[string]*CachedTestContext)
}

// GetCacheStats returns cache statistics
func (tco *TestContextOptimizer) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size":   len(tco.cache),
		"cache_ttl":    tco.cacheTTL.String(),
		"cache_entries": tco.getCacheEntries(),
	}
}

func (tco *TestContextOptimizer) getCacheEntries() []map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(tco.cache))

	for key, cached := range tco.cache {
		entries = append(entries, map[string]interface{}{
			"key":       key,
			"timestamp": cached.Timestamp,
			"run_id":    cached.RunID,
			"age":       time.Since(cached.Timestamp).String(),
		})
	}

	return entries
}