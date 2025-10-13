// Package harness provides the core testing framework for TUI integration testing.
// It coordinates PTY layer, input simulation, state validation, and test reporting.
package harness

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// TestHarnessStatus represents the current status of the test harness
type TestHarnessStatus int

const (
	TestHarnessStatusUninitialized TestHarnessStatus = iota
	TestHarnessStatusInitializing
	TestHarnessStatusReady
	TestHarnessStatusRunning
	TestHarnessStatusCompleted
	TestHarnessStatusFailed
	TestHarnessStatusStopped
)

// String returns the string representation of TestHarnessStatus
func (s TestHarnessStatus) String() string {
	switch s {
	case TestHarnessStatusUninitialized:
		return "uninitialized"
	case TestHarnessStatusInitializing:
		return "initializing"
	case TestHarnessStatusReady:
		return "ready"
	case TestHarnessStatusRunning:
		return "running"
	case TestHarnessStatusCompleted:
		return "completed"
	case TestHarnessStatusFailed:
		return "failed"
	case TestHarnessStatusStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// InputSimulator interface for input simulation
type InputSimulator interface {
	GenerateKeySequence(action string) ([]tea.KeyMsg, error)
	SimulateKeyPress(key tea.KeyMsg) error
	SimulateKeySequence(sequence []tea.KeyMsg) error
	SetKeyMappings(mappings []KeyMapping) error
	SetTimingDelay(delay time.Duration) error
	GetCurrentSequence() []tea.KeyMsg
	ClearSequence() error
}

// IStateValidator interface for state validation
type IStateValidator interface {
	CaptureState(model interface{}) (StateSnapshot, error)
	ValidateTransition(from, to StateSnapshot) ValidationResult
	CompareStates(expected interface{}, actual StateSnapshot) ComparisonResult
}

// ITestReporter interface for test reporting
type ITestReporter interface {
	AddTestResult(result TestResult) error
	GetTestResults() []TestResult
	GetTestResult(testID string) (TestResult, error)
	GenerateReport(results []TestResult) (Report, error)
}

// TestHarness is the main testing framework coordinator
type TestHarness struct {
	mu           sync.RWMutex
	id           string
	config       TestHarnessConfiguration
	status       TestHarnessStatus
	ptyLayer     *pty.PTYLayer
	simulator    InputSimulator
	validator    IStateValidator
	reporter     ITestReporter
	ctx          context.Context
	cancel       context.CancelFunc
	startTime    time.Time
	endTime      time.Time
	results      []TestResult
	currentTest  *TestCase
	teacModel    tea.Model
	teacProgram  *tea.Program
}

// NewTestHarness creates a new test harness instance
func NewTestHarness() *TestHarness {
	return &TestHarness{
		id:      generateTestHarnessID(),
		status:  TestHarnessStatusUninitialized,
		results: make([]TestResult, 0),
	}
}

// generateTestHarnessID generates a unique test harness ID
func generateTestHarnessID() string {
	return fmt.Sprintf("test-harness-%d", time.Now().UnixNano())
}

// Initialize sets up the test harness with the given configuration
func (th *TestHarness) Initialize(config TestHarnessConfiguration) error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.status != TestHarnessStatusUninitialized {
		return fmt.Errorf("test harness already initialized")
	}

	th.status = TestHarnessStatusInitializing
	th.config = config

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	th.ctx = ctx
	th.cancel = cancel

	// Initialize PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.Timeout = config.TestTimeout
	ptyConfig.Size = pty.PTYSize{
		Rows: config.TerminalRows,
		Cols: config.TerminalCols,
	}

	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	if err != nil {
		th.status = TestHarnessStatusFailed
		cancel()
		return fmt.Errorf("failed to create PTY layer: %w", err)
	}

	th.ptyLayer = ptyLayer

	// Initialize components
	if err := th.initializeComponents(); err != nil {
		th.status = TestHarnessStatusFailed
		th.cleanup()
		cancel()
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	th.status = TestHarnessStatusReady
	return nil
}

// initializeComponents sets up the harness components
func (th *TestHarness) initializeComponents() error {
	// Initialize state validator (doesn't require PTY allocation)
	validator := NewStateValidator(StateValidatorConfig{
		EnableDetailedValidation: th.config.StateValidation,
	})
	th.validator = validator

	// Initialize test reporter
	reporter := NewTestReporter(TestReporterConfig{
		ReportLevel: th.config.ReportLevel,
	})
	th.reporter = reporter

	// Note: Input simulator will be created after PTY allocation in Start()
	return nil
}

// Start begins the test harness and prepares it for test execution
func (th *TestHarness) Start() error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.status != TestHarnessStatusReady {
		return fmt.Errorf("test harness not ready, current status: %s", th.status.String())
	}

	// Allocate PTY
	if err := th.ptyLayer.Allocate(); err != nil {
		th.status = TestHarnessStatusFailed
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	// Now create input simulator (PTY is allocated)
	simulator, err := NewInputSimulator(InputSimulatorConfig{
		KeyInputDelay: th.config.KeyInputDelay,
		PTYLayer:      th.ptyLayer,
	})
	if err != nil {
		th.status = TestHarnessStatusFailed
		return fmt.Errorf("failed to create input simulator: %w", err)
	}
	th.simulator = simulator

	th.status = TestHarnessStatusRunning
	th.startTime = time.Now()

	return nil
}

// Stop stops the test harness and cleans up resources
func (th *TestHarness) Stop() error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.status == TestHarnessStatusUninitialized {
		return nil
	}

	// Stop any running tea program
	if th.teacProgram != nil {
		th.teacProgram.Quit()
		th.teacProgram = nil
	}

	// Cleanup PTY layer
	if th.ptyLayer != nil {
		if err := th.ptyLayer.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup PTY layer: %w", err)
		}
	}

	// Cancel context
	if th.cancel != nil {
		th.cancel()
	}

	th.endTime = time.Now()
	th.status = TestHarnessStatusStopped

	return nil
}

// GetStatus returns the current test harness status
func (th *TestHarness) GetStatus() TestHarnessStatus {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.status
}

// RunTest executes a single test case
func (th *TestHarness) RunTest(testCase TestCase) TestResult {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.status != TestHarnessStatusRunning {
		return TestResult{
			TestID: testCase.ID,
			Status: TestStatusError,
			Error:  fmt.Sprintf("test harness not running, current status: %s", th.status.String()),
		}
	}

	// Set current test
	th.currentTest = &testCase

	// Create test result
	result := TestResult{
		TestID:      testCase.ID,
		Action:      testCase.Name,
		Status:      TestStatusRunning,
		StartTime:   time.Now(),
		Description: testCase.Description,
	}

	// Add test result to results
	th.results = append(th.results, result)

	// Run the test
	testResult := th.runTestCase(testCase)

	// Update result in list
	th.results[len(th.results)-1] = testResult

	// Clear current test
	th.currentTest = nil

	return testResult
}

// runTestCase executes a single test case
func (th *TestHarness) runTestCase(testCase TestCase) TestResult {
	startTime := time.Now()

	// Setup TUI model for testing
	model, err := th.setupTUIModel()
	if err != nil {
		return TestResult{
			TestID:    testCase.ID,
			Action:    testCase.Name,
			Status:    TestStatusError,
			Error:     fmt.Sprintf("failed to setup TUI model: %v", err),
			StartTime: startTime,
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}

	// Initialize model with PTY
	options := []tea.ProgramOption{
		tea.WithInput(th.ptyLayer),
		tea.WithOutput(th.ptyLayer),
		tea.WithoutSignalHandler(),
	}

	program := tea.NewProgram(model, options...)
	th.teacProgram = program

	// Run test actions
	for i, action := range testCase.Actions {
		if err := th.executeAction(action); err != nil {
			return TestResult{
				TestID:    testCase.ID,
				Action:    testCase.Name,
				Status:    TestStatusError,
				Error:     fmt.Sprintf("failed to execute action %d: %v", i, err),
				StartTime: startTime,
				Duration:  time.Since(startTime),
				Timestamp: time.Now(),
			}
		}

		// Validate expected state if any
		if i < len(testCase.ExpectedStates) {
			expectedState := testCase.ExpectedStates[i]
			if err := th.validateExpectedState(expectedState, model); err != nil {
				return TestResult{
					TestID:    testCase.ID,
					Action:    testCase.Name,
					Status:    TestStatusFail,
					Error:     fmt.Sprintf("state validation failed: %v", err),
					StartTime: startTime,
					Duration:  time.Since(startTime),
					Timestamp: time.Now(),
				}
			}
		}
	}

	// Stop the program
	if program != nil {
		program.Quit()
		program.ReleaseTerminal()
		th.teacProgram = nil
	}

	// Test passed
	return TestResult{
		TestID:    testCase.ID,
		Action:    testCase.Name,
		Status:    TestStatusPass,
		StartTime: startTime,
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
		Metrics: TestMetrics{
			InputEvents:     len(testCase.Actions),
			StateValidations: len(testCase.ExpectedStates),
		},
	}
}

// setupTUIModel creates and initializes a TUI model for testing
func (th *TestHarness) setupTUIModel() (tea.Model, error) {
	// Create a mock TUI model for testing
	model := &MockTUIModel{}

	// Initialize the model with initial state
	cmd := model.Init()
	if cmd != nil {
		// Handle any initial commands if needed
	}

	return model, nil
}

// executeAction executes a single test action
func (th *TestHarness) executeAction(action TestAction) error {
	switch action.Type {
	case ActionTypeKeyPress:
		return th.executeKeyPress(action)
	case ActionTypeKeySequence:
		return th.executeKeySequence(action)
	case ActionTypeWait:
		return th.executeWait(action)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeKeyPress handles key press actions
func (th *TestHarness) executeKeyPress(action TestAction) error {
	params, ok := action.Parameters.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid parameters type for key press")
	}

	keyMsg, ok := params["key"].(tea.KeyMsg)
	if !ok {
		return fmt.Errorf("invalid key parameter")
	}

	return th.simulator.SimulateKeyPress(keyMsg)
}

// executeKeySequence handles key sequence actions
func (th *TestHarness) executeKeySequence(action TestAction) error {
	params, ok := action.Parameters.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid parameters type for key sequence")
	}

	keySequence, ok := params["sequence"].([]tea.KeyMsg)
	if !ok {
		return fmt.Errorf("invalid key sequence parameter")
	}

	return th.simulator.SimulateKeySequence(keySequence)
}

// executeWait handles wait actions
func (th *TestHarness) executeWait(action TestAction) error {
	params, ok := action.Parameters.(map[string]interface{})
	if ok {
		if duration, ok := params["duration"].(time.Duration); ok {
			if duration > 0 {
				time.Sleep(duration)
			}
			return nil
		}
	}

	// Use action delay if duration not specified
	if action.Delay > 0 {
		time.Sleep(action.Delay)
	}

	return nil
}

// validateExpectedState validates the expected state after an action
func (th *TestHarness) validateExpectedState(expected ExpectedState, model tea.Model) error {
	// Capture current state
	snapshot, err := th.validator.CaptureState(model)
	if err != nil {
		return fmt.Errorf("failed to capture state: %w", err)
	}

	// Compare with expected state
	comparison := th.validator.CompareStates(expected.ModelState, snapshot)
	if !comparison.Match {
		return fmt.Errorf("state mismatch: %s", comparison.Message)
	}

	return nil
}

// RunTestSuite executes a test suite
func (th *TestHarness) RunTestSuite(testSuite TestSuite) TestSuiteResult {
	startTime := time.Now()

	result := TestSuiteResult{
		Name:      testSuite.Name,
		StartTime: startTime,
		Results:   make([]TestResult, 0, len(testSuite.TestCases)),
	}

	// Update configuration with suite settings
	if testSuite.Configuration.TestTimeout > 0 {
		th.config.TestTimeout = testSuite.Configuration.TestTimeout
	}

	for _, testCase := range testSuite.TestCases {
		testResult := th.RunTest(testCase)
		result.Results = append(result.Results, testResult)

		// Stop suite if test failed and configuration requires it
		if testResult.Status == TestStatusError && testSuite.Configuration.StopOnError {
			break
		}
	}

	// Calculate suite metrics
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	result.TotalTests = len(testSuite.TestCases)
	result.PassedTests = 0
	result.FailedTests = 0
	result.ErrorTests = 0

	for _, testResult := range result.Results {
		switch testResult.Status {
		case TestStatusPass:
			result.PassedTests++
		case TestStatusFail:
			result.FailedTests++
		case TestStatusError:
			result.ErrorTests++
		}
	}

	result.Status = TestSuiteStatusPassed
	if result.FailedTests > 0 || result.ErrorTests > 0 {
		result.Status = TestSuiteStatusFailed
	}

	return result
}

// GetInputSimulator returns the input simulator
func (th *TestHarness) GetInputSimulator() InputSimulator {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.simulator
}

// GetStateValidator returns the state validator
func (th *TestHarness) GetStateValidator() IStateValidator {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.validator
}

// GetTestReporter returns the test reporter
func (th *TestHarness) GetTestReporter() ITestReporter {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.reporter
}

// GetResults returns all test results
func (th *TestHarness) GetResults() []TestResult {
	th.mu.RLock()
	defer th.mu.RUnlock()

	results := make([]TestResult, len(th.results))
	copy(results, th.results)
	return results
}

// GetID returns the test harness ID
func (th *TestHarness) GetID() string {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.id
}

// GetConfig returns the test harness configuration
func (th *TestHarness) GetConfig() TestHarnessConfiguration {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.config
}

// GetCurrentTest returns the currently running test
func (th *TestHarness) GetCurrentTest() *TestCase {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.currentTest
}

// GetDuration returns the total duration of the test harness
func (th *TestHarness) GetDuration() time.Duration {
	th.mu.RLock()
	defer th.mu.RUnlock()

	if th.startTime.IsZero() {
		return 0
	}

	endTime := th.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	return endTime.Sub(th.startTime)
}

// cleanup performs internal cleanup
func (th *TestHarness) cleanup() {
	if th.ptyLayer != nil {
		th.ptyLayer.Cleanup()
		th.ptyLayer = nil
	}

	if th.cancel != nil {
		th.cancel()
	}
}

// MockTUIModel provides a simple mock TUI model for testing
type MockTUIModel struct {
	viewType string
	content  string
}

// Init initializes the mock model
func (m *MockTUIModel) Init() tea.Cmd {
	return nil
}

// Update handles updates in the mock model
func (m *MockTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			if len(msg.Runes) > 0 {
				switch msg.Runes[0] {
				case '1':
					m.viewType = "processes"
				case '2':
					m.viewType = "logs"
				case '3':
					m.viewType = "errors"
				case '4':
					m.viewType = "aicoders"
				case '5':
					m.viewType = "settings"
				}
			}
		}
	}
	return m, nil
}

// View renders the mock model
func (m *MockTUIModel) View() string {
	if m.viewType == "" {
		m.viewType = "processes" // Default view
	}
	return fmt.Sprintf("Mock TUI - Current View: %s\nContent: %s", m.viewType, m.content)
}