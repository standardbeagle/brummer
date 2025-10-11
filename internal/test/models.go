package test

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// TestFramework represents a testing framework integration
type TestFramework interface {
	Name() string
	Detect() bool
	RunTests(pattern string, options TestOptions) (*TestRun, error)
	ParseOutput(output string) (*TestResults, error)
	GetCoverage(results *TestResults) (*TestCoverage, error)
}

// TestOptions contains options for test execution
type TestOptions struct {
	WatchMode bool
	Coverage  bool
	Parallel  bool
	Verbose   bool
	Timeout   time.Duration
	Filter    string
	EnvVars   map[string]string
}

// TestRun represents a single test execution
type TestRun struct {
	ID        string        `json:"id"`
	Framework string        `json:"framework"`
	Pattern   string        `json:"pattern"`
	StartTime time.Time     `json:"start_time"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Status    TestRunStatus `json:"status"`
	Results   *TestResults  `json:"results,omitempty"`
	Duration  time.Duration `json:"duration"`
	Options   TestOptions   `json:"options"`
	mutex     sync.RWMutex  `json:"-"`
}

// TestRunStatus represents the status of a test run
type TestRunStatus string

const (
	TestRunStatusPending   TestRunStatus = "pending"
	TestRunStatusRunning   TestRunStatus = "running"
	TestRunStatusCompleted TestRunStatus = "completed"
	TestRunStatusFailed    TestRunStatus = "failed"
	TestRunStatusCancelled TestRunStatus = "cancelled"
)

// TestResults contains the results of a test run
type TestResults struct {
	TotalTests   int              `json:"total_tests"`
	PassedTests  int              `json:"passed_tests"`
	FailedTests  int              `json:"failed_tests"`
	SkippedTests int              `json:"skipped_tests"`
	TestCases    []TestCase       `json:"test_cases"`
	Coverage     *TestCoverage    `json:"coverage,omitempty"`
	Performance  *TestPerformance `json:"performance,omitempty"`
	mutex        sync.RWMutex     `json:"-"`
}

// TestCase represents a single test case result
type TestCase struct {
	Name        string           `json:"name"`
	FilePath    string           `json:"file_path"`
	Status      TestCaseStatus   `json:"status"`
	Duration    time.Duration    `json:"duration"`
	StartTime   time.Time        `json:"start_time"`
	EndTime     time.Time        `json:"end_time"`
	FailureInfo *TestFailureInfo `json:"failure_info,omitempty"`
	SkipReason  string           `json:"skip_reason,omitempty"`
}

// TestCaseStatus represents the status of a test case
type TestCaseStatus string

const (
	TestCaseStatusPassed  TestCaseStatus = "passed"
	TestCaseStatusFailed  TestCaseStatus = "failed"
	TestCaseStatusSkipped TestCaseStatus = "skipped"
	TestCaseStatusPending TestCaseStatus = "pending"
	TestCaseStatusRunning TestCaseStatus = "running"
)

// TestFailureInfo contains failure information for a test case
type TestFailureInfo struct {
	Message    string   `json:"message"`
	StackTrace []string `json:"stack_trace"`
	Line       int      `json:"line,omitempty"`
	Column     int      `json:"column,omitempty"`
}

// TestCoverage contains coverage information
type TestCoverage struct {
	OverallPct   float64            `json:"overall_pct"`
	LineCoverage int                `json:"line_coverage"`
	TotalLines   int                `json:"total_lines"`
	Files        []TestCoverageFile `json:"files"`
	mutex        sync.RWMutex       `json:"-"`
}

// TestCoverageFile contains coverage information for a single file
type TestCoverageFile struct {
	FilePath     string  `json:"file_path"`
	LinesCovered int     `json:"lines_covered"`
	TotalLines   int     `json:"total_lines"`
	CoveragePct  float64 `json:"coverage_pct"`
}

// TestPerformance contains performance information
type TestPerformance struct {
	TotalDuration   time.Duration          `json:"total_duration"`
	AverageDuration time.Duration          `json:"average_duration"`
	SlowTests       []TestPerformanceEntry `json:"slow_tests"`
	SlowThreshold   time.Duration          `json:"slow_threshold"`
	mutex           sync.RWMutex           `json:"-"`
}

// TestPerformanceEntry contains performance information for a test case
type TestPerformanceEntry struct {
	TestName      string        `json:"test_name"`
	FilePath      string        `json:"file_path"`
	Duration      time.Duration `json:"duration"`
	IsSlow        bool          `json:"is_slow"`
	SlowThreshold time.Duration `json:"slow_threshold"`
}

// TestManager manages test runs and framework integrations
type TestManager struct {
	frameworks map[string]TestFramework
	activeRuns map[string]*TestRun
	results    map[string]*TestResults
	mutex      sync.RWMutex
}

// NewTestManager creates a new test manager
func NewTestManager() *TestManager {
	return &TestManager{
		frameworks: make(map[string]TestFramework),
		activeRuns: make(map[string]*TestRun),
		results:    make(map[string]*TestResults),
	}
}

// RegisterFramework registers a test framework
func (tm *TestManager) RegisterFramework(framework TestFramework) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.frameworks[framework.Name()] = framework
}

// GetFrameworks returns all registered frameworks
func (tm *TestManager) GetFrameworks() []TestFramework {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	frameworks := make([]TestFramework, 0, len(tm.frameworks))
	for _, f := range tm.frameworks {
		frameworks = append(frameworks, f)
	}
	return frameworks
}

// DetectFrameworks detects available test frameworks in the current directory
func (tm *TestManager) DetectFrameworks() []TestFramework {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	detected := make([]TestFramework, 0)
	for _, f := range tm.frameworks {
		if f.Detect() {
			detected = append(detected, f)
		}
	}
	return detected
}

// StartTestRun starts a new test run
func (tm *TestManager) StartTestRun(frameworkName, pattern string, options TestOptions) (*TestRun, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	framework, exists := tm.frameworks[frameworkName]
	if !exists {
		return nil, fmt.Errorf("framework %s not registered", frameworkName)
	}

	if !framework.Detect() {
		return nil, fmt.Errorf("framework %s not detected in current directory", frameworkName)
	}

	run := &TestRun{
		ID:        generateRunID(),
		Framework: frameworkName,
		Pattern:   pattern,
		StartTime: time.Now(),
		Status:    TestRunStatusPending,
		Options:   options,
	}

	tm.activeRuns[run.ID] = run
	return run, nil
}

// GetRun returns a test run by ID
func (tm *TestManager) GetRun(runID string) (*TestRun, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	run, exists := tm.activeRuns[runID]
	return run, exists
}

// GetActiveRuns returns all active test runs
func (tm *TestManager) GetActiveRuns() []*TestRun {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	runs := make([]*TestRun, 0, len(tm.activeRuns))
	for _, run := range tm.activeRuns {
		runs = append(runs, run)
	}
	return runs
}

// UpdateRunStatus updates the status of a test run
func (tm *TestManager) UpdateRunStatus(runID string, status TestRunStatus, results *TestResults) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	run, exists := tm.activeRuns[runID]
	if !exists {
		return
	}

	run.mutex.Lock()
	run.Status = status
	if results != nil {
		run.Results = results
	}

	if status == TestRunStatusCompleted || status == TestRunStatusFailed || status == TestRunStatusCancelled {
		now := time.Now()
		run.EndTime = &now
		run.Duration = now.Sub(run.StartTime)
	}
	run.mutex.Unlock()

	if results != nil {
		tm.results[runID] = results
	}
}

// ClearResults clears test results
func (tm *TestManager) ClearResults(runID string) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if runID == "" {
		// Clear all results
		tm.results = make(map[string]*TestResults)
	} else {
		delete(tm.results, runID)
	}
}

// generateRunID generates a unique run ID
func generateRunID() string {
	return fmt.Sprintf("test-run-%d-%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a random string of length n
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
