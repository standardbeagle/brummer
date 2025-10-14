package services

import (
	"time"
)

// ServiceStatus represents the status of a service for ServiceBridge integration
type ServiceStatus struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

// TestRunOptions contains options for test execution
type TestRunOptions struct {
	// Test selection
	Framework string   `json:"framework,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Files     []string `json:"files,omitempty"`

	// Execution options
	WatchMode bool          `json:"watch_mode,omitempty"`
	Coverage  bool          `json:"coverage,omitempty"`
	Parallel  bool          `json:"parallel,omitempty"`
	Verbose   bool          `json:"verbose,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`

	// Environment
	EnvVars map[string]string `json:"env_vars,omitempty"`

	// Output options
	OutputFormat string `json:"output_format,omitempty"` // "json", "junit", "text"
	SaveResults  bool   `json:"save_results,omitempty"`
}

// TestSummary provides a condensed summary of test results for AI agents
type TestSummary struct {
	TotalTests int           `json:"total_tests"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Skipped    int           `json:"skipped"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`

	// AI-optimized fields
	SuccessRate float64 `json:"success_rate"`
	IsPassing   bool    `json:"is_passing"`
	HasFailures bool    `json:"has_failures"`

	// Quick failure summary (if any)
	FailureSummary string `json:"failure_summary,omitempty"`
	FailureCount   int    `json:"failure_count,omitempty"`
}

// TestFailure provides detailed failure information for debugging
type TestFailure struct {
	TestName   string        `json:"test_name"`
	FilePath   string        `json:"file_path"`
	Line       int           `json:"line"`
	Message    string        `json:"message"`
	StackTrace []string      `json:"stack_trace,omitempty"`
	Expected   string        `json:"expected,omitempty"`
	Actual     string        `json:"actual,omitempty"`
	Diff       string        `json:"diff,omitempty"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
}

// TestCoverage provides coverage information
type TestCoverage struct {
	OverallPct   float64            `json:"overall_pct"`
	LineCoverage int                `json:"line_coverage"`
	TotalLines   int                `json:"total_lines"`
	Files        []TestCoverageFile `json:"files"`
	LastUpdated  time.Time          `json:"last_updated"`
}

// TestCoverageFile contains coverage information for a single file
type TestCoverageFile struct {
	FilePath     string  `json:"file_path"`
	LinesCovered int     `json:"lines_covered"`
	TotalLines   int     `json:"total_lines"`
	CoveragePct  float64 `json:"coverage_pct"`
	MissedLines  []int   `json:"missed_lines,omitempty"`
}

// TestStatus represents the current status of test operations
type TestStatus struct {
	IsRunning           bool       `json:"is_running"`
	IsWatching          bool       `json:"is_watching"`
	ActiveRuns          int        `json:"active_runs"`
	LastRunTime         *time.Time `json:"last_run_time,omitempty"`
	LastRunStatus       string     `json:"last_run_status,omitempty"`
	AvailableFrameworks []string   `json:"available_frameworks"`
}

// WatchStatus represents the status of watch mode
type WatchStatus struct {
	IsActive     bool       `json:"is_active"`
	StartTime    time.Time  `json:"start_time"`
	LastTrigger  *time.Time `json:"last_trigger,omitempty"`
	Pattern      string     `json:"pattern,omitempty"`
	TriggerCount int        `json:"trigger_count"`
}

// TestAnalysis provides insights and recommendations
type TestAnalysis struct {
	AnalysisType    string                 `json:"analysis_type"`
	Summary         string                 `json:"summary"`
	Recommendations []string               `json:"recommendations"`
	Metrics         map[string]interface{} `json:"metrics"`
	Issues          []TestIssue            `json:"issues,omitempty"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// TestIssue represents an issue found during test analysis
type TestIssue struct {
	Type        string `json:"type"`     // "performance", "coverage", "flaky", "quality"
	Severity    string `json:"severity"` // "low", "medium", "high", "critical"
	Description string `json:"description"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// GeneratedTests represents generated test scaffolding
type GeneratedTests struct {
	TargetFile    string    `json:"target_file"`
	TestFile      string    `json:"test_file"`
	Framework     string    `json:"framework"`
	GeneratedCode string    `json:"generated_code"`
	Imports       []string  `json:"imports"`
	Functions     []string  `json:"functions"`
	GeneratedAt   time.Time `json:"generated_at"`
	Instructions  string    `json:"instructions"`
}

// DebugContext provides comprehensive debugging information
type DebugContext struct {
	TestID        string                 `json:"test_id"`
	TestName      string                 `json:"test_name"`
	FilePath      string                 `json:"file_path"`
	Error         string                 `json:"error"`
	StackTrace    []string               `json:"stack_trace"`
	RelatedFiles  []string               `json:"related_files"`
	RecentChanges []FileChange           `json:"recent_changes"`
	Environment   map[string]string      `json:"environment"`
	Logs          []string               `json:"logs"`
	Context       map[string]interface{} `json:"context"`
}

// FileChange represents a file change relevant to test debugging
type FileChange struct {
	FilePath    string    `json:"file_path"`
	ChangeType  string    `json:"change_type"` // "modified", "added", "deleted"
	Timestamp   time.Time `json:"timestamp"`
	ContentDiff string    `json:"content_diff,omitempty"`
}
