package services

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/test"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestManagementService manages test execution, lifecycle, and results tracking
// Integrates with ServiceBridge to provide test-related events and UI updates
type TestManagementService struct {
	// Core components
	runnerService *test.TestRunnerService
	logStore      *logs.Store
	eventBus      *events.EventBus

	// State management (using atomic operations for lock-free design)
	isWatching  int32        // 0 = not watching, 1 = watching
	activeRuns  atomic.Value // map[string]*test.TestRun
	lastResults atomic.Value // *test.TestResults
	watchCancel context.CancelFunc

	// Configuration
	config TestConfig

	// Event subscription cleanup
	subscriptions []func()
}

// TestConfig contains configuration for test management
type TestConfig struct {
	MaxHistory       int           `toml:"max_history"`
	DefaultTimeout   time.Duration `toml:"default_timeout"`
	WatchDebounce    time.Duration `toml:"watch_debounce"`
	DefaultCoverage  bool          `toml:"default_coverage"`
	VerboseByDefault bool          `toml:"verbose_by_default"`
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		MaxHistory:       100,
		DefaultTimeout:   5 * time.Minute,
		WatchDebounce:    500 * time.Millisecond,
		DefaultCoverage:  false,
		VerboseByDefault: false,
	}
}

// NewTestManagementService creates a new test management service
func NewTestManagementService(
	logStore *logs.Store,
	eventBus *events.EventBus,
	config TestConfig,
) *TestManagementService {
	tms := &TestManagementService{
		runnerService: test.NewTestRunnerService(),
		logStore:      logStore,
		eventBus:      eventBus,
		config:        config,
		subscriptions: []func(){},
	}

	// Initialize atomic values
	tms.activeRuns.Store(make(map[string]*test.TestRun))
	tms.lastResults.Store((*test.TestResults)(nil))

	return tms
}

// GetRunnerService returns the underlying test runner service
func (tms *TestManagementService) GetRunnerService() *test.TestRunnerService {
	return tms.runnerService
}

// DetectFrameworks detects available test frameworks in the current directory
func (tms *TestManagementService) DetectFrameworks() []test.TestFramework {
	return tms.runnerService.DetectAvailableFrameworks()
}

// RunTests executes tests with the given options
func (tms *TestManagementService) RunTests(ctx context.Context, opts TestRunOptions) (*test.TestRun, error) {
	// Detect available frameworks
	detected := tms.DetectFrameworks()
	if len(detected) == 0 {
		return nil, fmt.Errorf("no test frameworks detected in current directory")
	}

	// Use the first detected framework unless specified
	frameworkName := opts.Framework
	if frameworkName == "" {
		frameworkName = detected[0].Name()
	}

	// Convert options
	testOptions := test.TestOptions{
		WatchMode: opts.WatchMode,
		Coverage:  opts.Coverage,
		Parallel:  opts.Parallel,
		Verbose:   opts.Verbose,
		Timeout:   opts.Timeout,
		Filter:    opts.Pattern,
		EnvVars:   opts.EnvVars,
	}

	// Start test run
	run, err := tms.runnerService.GetManager().StartTestRun(frameworkName, opts.Pattern, testOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to start test run: %w", err)
	}

	// Track active run
	tms.addActiveRun(run)

	// Emit test started event
	tms.emitTestStartedEvent(run, opts)

	// NO MORE GOROUTINES - Test execution will be command-driven
	// through BubbleTea Update() cycle instead of background goroutine
	// The test execution will be triggered by a command in the UI loop

	return run, nil
}

// RunTestsInWatchMode starts watch mode for continuous testing
func (tms *TestManagementService) RunTestsInWatchMode(ctx context.Context, opts TestRunOptions) error {
	if !atomic.CompareAndSwapInt32(&tms.isWatching, 0, 1) {
		return fmt.Errorf("watch mode is already active")
	}

	// Create cancellable context for watch mode
	watchCtx, cancel := context.WithCancel(ctx)
	tms.watchCancel = cancel

	// Start initial test run
	_, err := tms.RunTests(watchCtx, opts)
	if err != nil {
		atomic.StoreInt32(&tms.isWatching, 0)
		return fmt.Errorf("failed to start initial test run: %w", err)
	}

	// File system watching for continuous testing
	// IMPLEMENTATION: Add fsnotify integration to watch for file changes
	// and trigger relevant test runs based on changed files
	// This requires:
	// 1. Set up file watchers for test directories
	// 2. Debounce file change events
	// 3. Determine which tests to run based on changed files
	// 4. Trigger selective test execution

	// Emit watch mode started event
	tms.emitWatchModeStartedEvent(opts)

	return nil
}

// StopWatchMode stops the current watch mode
func (tms *TestManagementService) StopWatchMode() error {
	if !atomic.CompareAndSwapInt32(&tms.isWatching, 1, 0) {
		return fmt.Errorf("watch mode is not active")
	}

	if tms.watchCancel != nil {
		tms.watchCancel()
		tms.watchCancel = nil
	}

	// Emit watch mode stopped event
	tms.emitWatchModeStoppedEvent()

	return nil
}

// IsWatching returns true if watch mode is active
func (tms *TestManagementService) IsWatching() bool {
	return atomic.LoadInt32(&tms.isWatching) == 1
}

// GetActiveRuns returns all currently active test runs
func (tms *TestManagementService) GetActiveRuns() []*test.TestRun {
	runs := tms.getActiveRuns()
	result := make([]*test.TestRun, 0, len(runs))
	for _, run := range runs {
		result = append(result, run)
	}
	return result
}

// GetLastResults returns the most recent test results
func (tms *TestManagementService) GetLastResults() *test.TestResults {
	return tms.lastResults.Load().(*test.TestResults)
}

// ClearResults clears test result history
func (tms *TestManagementService) ClearResults(scope string) error {
	switch scope {
	case "all":
		tms.lastResults.Store((*test.TestResults)(nil))
		tms.runnerService.GetManager().ClearResults("")
	case "failures":
		// Failure-specific clearing
		// IMPLEMENTATION: Filter out failed tests and keep only passing ones
		tms.lastResults.Store((*test.TestResults)(nil))
	case "results":
		tms.lastResults.Store((*test.TestResults)(nil))
	default:
		return fmt.Errorf("invalid clear scope: %s", scope)
	}

	// Emit results cleared event
	tms.emitResultsClearedEvent(scope)

	return nil
}

// GetTestSummary returns a summary of recent test results
func (tms *TestManagementService) GetTestSummary() *TestSummary {
	results := tms.GetLastResults()
	if results == nil {
		return &TestSummary{
			TotalTests: 0,
			Passed:     0,
			Failed:     0,
			Skipped:    0,
			Duration:   0,
			Timestamp:  time.Now(),
		}
	}

	// Calculate duration from test cases or use a default
	var duration time.Duration
	if len(results.TestCases) > 0 {
		// Sum up durations from test cases
		for _, tc := range results.TestCases {
			duration += tc.Duration
		}
	}

	return &TestSummary{
		TotalTests: results.TotalTests,
		Passed:     results.PassedTests,
		Failed:     results.FailedTests,
		Skipped:    results.SkippedTests,
		Duration:   duration,
		Timestamp:  time.Now(),
	}
}

// Stop stops the test management service and cleans up resources
func (tms *TestManagementService) Stop() error {
	// Stop watch mode if active
	if tms.IsWatching() {
		tms.StopWatchMode()
	}

	// Clean up event subscriptions
	for _, unsub := range tms.subscriptions {
		if unsub != nil {
			unsub()
		}
	}
	tms.subscriptions = nil

	return nil
}

// Helper methods for lock-free state management

func (tms *TestManagementService) addActiveRun(run *test.TestRun) {
	runs := tms.getActiveRuns()
	newRuns := make(map[string]*test.TestRun)
	for k, v := range runs {
		newRuns[k] = v
	}
	newRuns[run.ID] = run
	tms.activeRuns.Store(newRuns)
}

func (tms *TestManagementService) removeActiveRun(runID string) {
	runs := tms.getActiveRuns()
	newRuns := make(map[string]*test.TestRun)
	for k, v := range runs {
		if k != runID {
			newRuns[k] = v
		}
	}
	tms.activeRuns.Store(newRuns)
}

func (tms *TestManagementService) getActiveRuns() map[string]*test.TestRun {
	return tms.activeRuns.Load().(map[string]*test.TestRun)
}

// Test execution methods

func (tms *TestManagementService) executeTestRun(ctx context.Context, run *test.TestRun, opts TestRunOptions) {
	defer tms.removeActiveRun(run.ID)

	// Update status to running
	tms.runnerService.GetManager().UpdateRunStatus(run.ID, test.TestRunStatusRunning, nil)

	// Actually execute the tests using the framework
	// This replaces the previous simulation with real test execution
	tms.executeTests(ctx, run, opts)
}

// executeTests performs actual test execution using the test framework
func (tms *TestManagementService) executeTests(ctx context.Context, run *test.TestRun, opts TestRunOptions) {
	// Get the test executor from runner service
	executor := test.NewTestExecutor()

	// Convert options to framework options
	frameworkOptions := test.TestOptions{
		WatchMode: opts.WatchMode,
		Verbose:   opts.Verbose,
		Coverage:  opts.Coverage,
		Parallel:  opts.Parallel,
		Timeout:   opts.Timeout,
		Filter:    "",
		EnvVars:   opts.EnvVars,
	}

	// Execute tests with timeout
	done := make(chan error, 1)
	go func() {
		done <- tms.performExecution(ctx, executor, run, opts, frameworkOptions)
	}()

	select {
	case err := <-done:
		if err != nil {
			tms.runnerService.GetManager().UpdateRunStatus(run.ID, test.TestRunStatusFailed, nil)
			tms.emitTestCompletedEvent(run, fmt.Sprintf("execution error: %v", err))
			tms.logStore.Add("test", "TestService", fmt.Sprintf("❌ Test execution failed: %v", err), true)
			return
		}
	case <-ctx.Done():
		tms.runnerService.GetManager().UpdateRunStatus(run.ID, test.TestRunStatusCancelled, nil)
		tms.emitTestCompletedEvent(run, "cancelled")
		return
	}
}

// performExecution performs the actual test execution
func (tms *TestManagementService) performExecution(ctx context.Context, executor *test.TestExecutor, run *test.TestRun, opts TestRunOptions, frameworkOptions test.TestOptions) error {
	// Detect or use specified framework
	frameworkName := opts.Framework
	if frameworkName == "" {
		detected := executor.DetectAvailableFrameworks()
		if len(detected) == 0 {
			return fmt.Errorf("no test frameworks detected")
		}
		frameworkName = detected[0].Name()
	}

	// Execute tests using the framework
	results, err := executor.ExecuteTests(ctx, frameworkName, run.Pattern, frameworkOptions)
	if err != nil {
		return err
	}

	// Update run status with results
	status := test.TestRunStatusCompleted
	if results.FailedTests > 0 {
		status = test.TestRunStatusFailed
	}

	tms.runnerService.GetManager().UpdateRunStatus(run.ID, status, results)
	tms.lastResults.Store(results)

	// Log test results
	tms.logTestResults(run, results)

	// Emit completion event
	tms.emitTestCompletedEvent(run, "completed")

	return nil
}

func (tms *TestManagementService) simulateTestExecution(ctx context.Context, run *test.TestRun, opts TestRunOptions) {
	// Simulate test execution time
	executionTime := time.Duration(1+time.Now().UnixNano()%5) * time.Second
	select {
	case <-time.After(executionTime):
		// Continue with execution
	case <-ctx.Done():
		// Test was cancelled
		tms.runnerService.GetManager().UpdateRunStatus(run.ID, test.TestRunStatusCancelled, nil)
		tms.emitTestCompletedEvent(run, "cancelled")
		return
	}

	// Generate realistic test results
	results := tms.generateTestResults(opts)

	// Update run status
	status := test.TestRunStatusCompleted
	if results.FailedTests > 0 {
		status = test.TestRunStatusFailed
	}

	tms.runnerService.GetManager().UpdateRunStatus(run.ID, status, results)
	tms.lastResults.Store(results)

	// Log test results
	tms.logTestResults(run, results)

	// Emit completion event
	tms.emitTestCompletedEvent(run, string(status))
}

func (tms *TestManagementService) generateTestResults(opts TestRunOptions) *test.TestResults {
	// Generate realistic test results for demonstration
	totalTests := int(10 + (time.Now().UnixNano() % 20))
	passedTests := int(totalTests - int(time.Now().UnixNano()%3))
	failedTests := int(totalTests - passedTests)
	skippedTests := int(totalTests % 4)

	results := &test.TestResults{
		TotalTests:   totalTests,
		PassedTests:  passedTests,
		FailedTests:  failedTests,
		SkippedTests: skippedTests,
		TestCases:    make([]test.TestCase, 0, totalTests),
	}

	// Generate test cases
	for i := 0; i < totalTests; i++ {
		var status test.TestCaseStatus
		var failureInfo *test.TestFailureInfo

		if i < passedTests {
			status = test.TestCaseStatusPassed
		} else if i < passedTests+failedTests {
			status = test.TestCaseStatusFailed
			failureInfo = &test.TestFailureInfo{
				Message: fmt.Sprintf("Test case %d failed with assertion error", i),
				Line:    10 + i%20,
			}
		} else {
			status = test.TestCaseStatusSkipped
		}

		testCase := test.TestCase{
			Name:     fmt.Sprintf("TestCase_%d", i),
			FilePath: fmt.Sprintf("test/example_test.go"),
			Status:   status,
			Duration: time.Duration(10+time.Now().UnixNano()%100) * time.Millisecond,
		}

		if failureInfo != nil {
			testCase.FailureInfo = failureInfo
		}

		results.TestCases = append(results.TestCases, testCase)
	}

	// Add coverage if requested
	if opts.Coverage {
		results.Coverage = &test.TestCoverage{
			OverallPct: 70.0 + float64(time.Now().UnixNano()%20),
			Files: []test.TestCoverageFile{
				{
					FilePath:     "src/example.go",
					LinesCovered: 80,
					TotalLines:   100,
					CoveragePct:  80.0,
				},
			},
		}
	}

	return results
}

// Event emission methods

func (tms *TestManagementService) emitTestStartedEvent(run *test.TestRun, opts TestRunOptions) {
	if tms.eventBus != nil {
		tms.eventBus.Publish(events.Event{
			Type:      "test.started",
			ProcessID: run.ID,
			Data: map[string]interface{}{
				"run_id":    run.ID,
				"framework": run.Framework,
				"pattern":   run.Pattern,
				"options":   opts,
			},
		})
	}
}

func (tms *TestManagementService) emitTestCompletedEvent(run *test.TestRun, status string) {
	if tms.eventBus != nil {
		results := run.Results
		var summary interface{}
		if results != nil {
			// Calculate duration from test cases
			var duration time.Duration
			for _, tc := range results.TestCases {
				duration += tc.Duration
			}

			summary = map[string]interface{}{
				"total":    results.TotalTests,
				"passed":   results.PassedTests,
				"failed":   results.FailedTests,
				"skipped":  results.SkippedTests,
				"duration": duration,
			}
		}

		tms.eventBus.Publish(events.Event{
			Type:      "test.completed",
			ProcessID: run.ID,
			Data: map[string]interface{}{
				"run_id":  run.ID,
				"status":  status,
				"results": results,
				"summary": summary,
			},
		})
	}
}

func (tms *TestManagementService) emitWatchModeStartedEvent(opts TestRunOptions) {
	if tms.eventBus != nil {
		tms.eventBus.Publish(events.Event{
			Type: "test.watch_started",
			Data: map[string]interface{}{
				"options": opts,
			},
		})
	}
}

func (tms *TestManagementService) emitWatchModeStoppedEvent() {
	if tms.eventBus != nil {
		tms.eventBus.Publish(events.Event{
			Type: "test.watch_stopped",
			Data: map[string]interface{}{},
		})
	}
}

func (tms *TestManagementService) emitResultsClearedEvent(scope string) {
	if tms.eventBus != nil {
		tms.eventBus.Publish(events.Event{
			Type: "test.results_cleared",
			Data: map[string]interface{}{
				"scope": scope,
			},
		})
	}
}

// Logging methods

func (tms *TestManagementService) logTestResults(run *test.TestRun, results *test.TestResults) {
	if tms.logStore == nil {
		return
	}

	// Calculate duration from test cases
	var duration time.Duration
	for _, tc := range results.TestCases {
		duration += tc.Duration
	}

	// Log summary
	summary := fmt.Sprintf("Test run %s completed: %d/%d tests passed (%.2fs)",
		run.ID, results.PassedTests, results.TotalTests, duration.Seconds())

	tms.logStore.Add("test", run.ID, summary, false)

	// Log failures with details
	for _, testCase := range results.TestCases {
		if testCase.Status == test.TestCaseStatusFailed && testCase.FailureInfo != nil {
			failureMsg := fmt.Sprintf("FAILED: %s - %s (line %d)",
				testCase.Name, testCase.FailureInfo.Message, testCase.FailureInfo.Line)
			tms.logStore.Add("test", run.ID, failureMsg, true)
		}
	}
}

// EventBus integration for ServiceBridge compatibility

// SubscribeToEvents subscribes the test management service to relevant system events
func (tms *TestManagementService) SubscribeToEvents() {
	if tms.eventBus == nil {
		return
	}

	// Subscribe to process events that might affect testing
	tms.eventBus.Subscribe(events.ProcessStarted, func(evt events.Event) {
		// IMPLEMENTATION: Auto-run tests when dev servers start
		// This would trigger relevant test suites based on running processes
	})

	tms.eventBus.Subscribe(events.ProcessExited, func(evt events.Event) {
		// IMPLEMENTATION: Auto-run tests when dev servers stop
		// Could trigger cleanup or final validation tests
	})
}

// ServiceBridge integration methods

// GetServiceName returns the service name for ServiceBridge integration
func (tms *TestManagementService) GetServiceName() string {
	return "test_management"
}

// GetServiceStatus returns the current status of the test management service
func (tms *TestManagementService) GetServiceStatus() ServiceStatus {
	activeRuns := tms.GetActiveRuns()
	isWatching := tms.IsWatching()

	status := "idle"
	details := fmt.Sprintf("%d frameworks available", len(tms.DetectFrameworks()))

	if len(activeRuns) > 0 {
		status = "running"
		details = fmt.Sprintf("%d active test runs", len(activeRuns))
	}

	if isWatching {
		status = "watching"
		details += " (watch mode active)"
	}

	return ServiceStatus{
		Service: tms.GetServiceName(),
		Status:  status,
		Details: details,
	}
}
