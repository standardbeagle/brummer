package messages

import (
	"time"
)

// Test lifecycle events
// These messages represent state changes in the test runner

type TestRunStartedMsg struct {
	RunID       string
 Framework   string // "jest", "vitest", "go", "pytest", etc.
	TestPattern string // File pattern or test name filter
	StartTime   time.Time
}

type TestRunCompletedMsg struct {
	RunID       string
	TotalTests  int
	PassedTests int
	FailedTests int
	SkippedTests int
	Duration    time.Duration
	EndTime     time.Time
}

type TestCaseStartedMsg struct {
	RunID     string
	TestName  string
	FilePath  string
	StartTime time.Time
}

type TestCaseCompletedMsg struct {
	RunID       string
	TestName    string
	FilePath    string
	Status      string // "passed", "failed", "skipped"
	Duration    time.Duration
	FailureInfo *TestFailureInfo
	EndTime     time.Time
}

type TestFailureInfo struct {
	Message    string
	StackTrace []string
	Line       int
	Column     int
}

type TestResultsClearedMsg struct {
	RunID string // Empty string = all cleared
}

// Test watch mode events

type TestWatchModeStartedMsg struct {
	WatchPattern string
	StartTime    time.Time
}

type TestWatchModeStoppedMsg struct {
	EndTime time.Time
}

type TestFileChangedMsg struct {
	FilePath    string
	TriggeredRun bool
	ChangeTime  time.Time
}

// Test coverage events

type TestCoverageUpdatedMsg struct {
	RunID       string
	CoveragePct float64
	Files       []TestCoverageFile
	Timestamp   time.Time
}

type TestCoverageFile struct {
	FilePath     string
	LinesCovered int
	TotalLines   int
	CoveragePct  float64
}

// Test performance events

type TestPerformanceUpdatedMsg struct {
	RunID      string
	TestName   string
	Duration   time.Duration
	SlowTests  []TestPerformanceEntry
	Timestamp  time.Time
}

type TestPerformanceEntry struct {
	TestName    string
	FilePath    string
	Duration    time.Duration
	IsSlow      bool
	SlowThreshold time.Duration
}