package harness

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// ActionType represents the type of test action
type ActionType string

const (
	ActionTypeKeyPress    ActionType = "key_press"
	ActionTypeKeySequence ActionType = "key_sequence"
	ActionTypeWait       ActionType = "wait"
	ActionTypeStateCheck ActionType = "state_check"
)

// TestAction represents a single action in a test case
type TestAction struct {
	Type        ActionType    `json:"type"`
	Parameters  interface{}   `json:"parameters"`
	Description  string        `json:"description"`
	Delay       time.Duration `json:"delay"`
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPass   TestStatus = "pass"
	TestStatusFail   TestStatus = "fail"
	TestStatusError  TestStatus = "error"
	TestStatusSkip   TestStatus = "skip"
	TestStatusRunning TestStatus = "running"
)

// TestMetrics contains metrics for a test execution
type TestMetrics struct {
	InputEvents      int           `json:"input_events"`
	StateValidations int           `json:"state_validations"`
	ViewChanges      int           `json:"view_changes"`
	Duration         time.Duration `json:"duration"`
	MemoryUsed       int64         `json:"memory_used"`
}

// TestResult represents the result of a test execution
type TestResult struct {
	TestID       string        `json:"test_id"`
	Action       string        `json:"action"`
	ExpectedState ExpectedState `json:"expected_state"`
	ActualState   ActualState   `json:"actual_state"`
	Status       TestStatus    `json:"status"`
	Duration     time.Duration `json:"duration"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	StartTime    time.Time     `json:"start_time"`
	Metrics      TestMetrics   `json:"metrics"`
	Description  string        `json:"description"`
}

// ExpectedState defines the expected TUI state after input
type ExpectedState struct {
	TriggerAction string                 `json:"trigger_action"`
	ViewType      interface{}            `json:"view_type"`
	SelectedProcess string               `json:"selected_process"`
	ModelState    map[string]interface{} `json:"model_state"`
	Timeout       time.Duration          `json:"timeout"`
}

// ActualState represents the captured actual state
type ActualState struct {
	ViewType      interface{}            `json:"view_type"`
	SelectedProcess string               `json:"selected_process"`
	ModelState    map[string]interface{} `json:"model_state"`
	Timestamp     time.Time              `json:"timestamp"`
	Content       string                 `json:"content"`
	CursorRow     int                    `json:"cursor_row"`
	CursorCol     int                    `json:"cursor_col"`
}

// TestCase represents a complete test case
type TestCase struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Actions         []TestAction   `json:"actions"`
	ExpectedStates  []ExpectedState `json:"expected_states"`
	Preconditions   []string       `json:"preconditions"`
	Timeout         time.Duration  `json:"timeout"`
	SetupActions    []TestAction   `json:"setup_actions"`
	TeardownActions []TestAction   `json:"teardown_actions"`
}

// TestSuite represents a collection of test cases
type TestSuite struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	TestCases     []TestCase `json:"test_cases"`
	Configuration TestSuiteConfiguration `json:"configuration"`
}

// TestSuiteConfiguration contains configuration for test suite execution
type TestSuiteConfiguration struct {
	TestTimeout   time.Duration `json:"test_timeout"`
	StateValidation bool        `json:"state_validation"`
	StopOnError   bool          `json:"stop_on_error"`
	Parallel      bool          `json:"parallel"`
	MaxConcurrent int           `json:"max_concurrent"`
}

// TestSuiteStatus represents the status of a test suite
type TestSuiteStatus string

const (
	TestSuiteStatusPassed TestSuiteStatus = "passed"
	TestSuiteStatusFailed TestSuiteStatus = "failed"
	TestSuiteStatusError  TestSuiteStatus = "error"
	TestSuiteStatusSkipped TestSuiteStatus = "skipped"
)

// TestSuiteResult represents the result of a test suite execution
type TestSuiteResult struct {
	Name         string              `json:"name"`
	Status       TestSuiteStatus     `json:"status"`
	StartTime    time.Time           `json:"start_time"`
	EndTime      time.Time           `json:"end_time"`
	Duration     time.Duration       `json:"duration"`
	TotalTests   int                 `json:"total_tests"`
	PassedTests  int                 `json:"passed_tests"`
	FailedTests  int                 `json:"failed_tests"`
	ErrorTests   int                 `json:"error_tests"`
	SkippedTests int                 `json:"skipped_tests"`
	Results      []TestResult        `json:"results"`
	Metrics      TestSuiteMetrics    `json:"metrics"`
}

// TestSuiteMetrics contains metrics for a test suite execution
type TestSuiteMetrics struct {
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	TotalInputEvents int         `json:"total_input_events"`
	TotalStateValidations int     `json:"total_state_validations"`
}

// ReportLevel represents the level of detail in test reports
type ReportLevel string

const (
	ReportLevelSummary    ReportLevel = "summary"
	ReportLevelBasic      ReportLevel = "basic"
	ReportLevelDetailed   ReportLevel = "detailed"
	ReportLevelComprehensive ReportLevel = "comprehensive"
)

// TestScope represents the scope of tests to execute
type TestScope string

const (
	TestScopeAll         TestScope = "all"
	TestScopeUnit        TestScope = "unit"
	TestScopeIntegration TestScope = "integration"
	TestScopeE2E         TestScope = "e2e"
	TestScopeSmoke       TestScope = "smoke"
	TestScopeRegression  TestScope = "regression"
)

// ReportFormat represents the format of test reports
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatXML  ReportFormat = "xml"
	ReportFormatJUNIT ReportFormat = "junit"
	ReportFormatTXT  ReportFormat = "txt"
	ReportFormatCSV  ReportFormat = "csv"
)

// Report represents a generated test report
type Report struct {
	ID          string       `json:"id"`
	GeneratedAt time.Time    `json:"generated_at"`
	Summary     TestSummary  `json:"summary"`
	Results     []TestResult `json:"results"`
	Metadata    ReportMetadata `json:"metadata"`
}

// TestSummary contains summary information about test execution
type TestSummary struct {
	TotalTests    int           `json:"total_tests"`
	PassedTests   int           `json:"passed_tests"`
	FailedTests   int           `json:"failed_tests"`
	ErrorTests    int           `json:"error_tests"`
	SkippedTests  int           `json:"skipped_tests"`
	Duration      time.Duration `json:"duration"`
	SuccessRate   float64       `json:"success_rate"`
	GeneratedAt   time.Time     `json:"generated_at"`
}

// ReportMetadata contains metadata about the generated report
type ReportMetadata struct {
	Version       string            `json:"version"`
	Environment   map[string]string `json:"environment"`
	TestHarnessID string            `json:"test_harness_id"`
	Configuration TestHarnessConfiguration `json:"configuration"`
}

// KeyMapping maps test actions to keyboard input sequences
type KeyMapping struct {
	Action      string      `json:"action"`
	Keys        []tea.KeyMsg `json:"keys"`
	Description string      `json:"description"`
	Preconditions []string   `json:"preconditions"`
}

// InputSimulatorConfig contains configuration for input simulation
type InputSimulatorConfig struct {
	KeyInputDelay time.Duration `json:"key_input_delay"`
	PTYLayer      pty.PTYLayer  `json:"-"`
	KeyMappings   []KeyMapping  `json:"key_mappings"`
}

// StateValidatorConfig contains configuration for state validation
type StateValidatorConfig struct {
	EnableDetailedValidation bool `json:"enable_detailed_validation"`
	PTYLayer               pty.PTYLayer `json:"-"`
	ValidationRules        []ValidationRule `json:"validation_rules"`
}

// TestReporterConfig contains configuration for test reporting
type TestReporterConfig struct {
	ReportLevel    ReportLevel `json:"report_level"`
	OutputFormats  []ReportFormat `json:"output_formats"`
	OutputDir      string      `json:"output_dir"`
	IncludeScreenshots bool    `json:"include_screenshots"`
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details"`
}

// ComparisonResult represents the result of a state comparison
type ComparisonResult struct {
	Match   bool   `json:"match"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details"`
	Diff    []StateDiff `json:"diff"`
}

// StateDiff represents a difference between expected and actual state
type StateDiff struct {
	Path     string      `json:"path"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Type     DiffType    `json:"type"`
}

// DiffType represents the type of state difference
type DiffType string

const (
	DiffTypeAdded     DiffType = "added"
	DiffTypeRemoved   DiffType = "removed"
	DiffTypeModified  DiffType = "modified"
	DiffTypeType      DiffType = "type"
)

// StateSnapshot represents a captured state of the TUI
type StateSnapshot struct {
	Timestamp    time.Time              `json:"timestamp"`
	Model        interface{}            `json:"model"`
	Content      string                 `json:"content"`
	Cursor       CursorPosition         `json:"cursor"`
	Size         TerminalSize           `json:"size"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CursorPosition represents the cursor position in the terminal
type CursorPosition struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// TerminalSize represents the terminal size
type TerminalSize struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// ValidationFunction represents a validation function
type ValidationFunction func(state StateSnapshot) ValidationResult

// KeyMappingValidationFunction represents a validation function for key mappings
type KeyMappingValidationFunction func(context KeyMappingValidationContext) ValidationResult

// ValidationRule represents a rule for state validation
type ValidationRule struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description  string              `json:"description"`
	Validator   ValidationFunction  `json:"-"`
	KeyMappingValidator KeyMappingValidationFunction `json:"-"`
	Parameters  interface{}         `json:"parameters"`
	Enabled     bool                `json:"enabled"`
	IsKeyMappingRule bool             `json:"is_key_mapping_rule"`
}