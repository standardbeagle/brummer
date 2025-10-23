package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/pty"
)

// KeyboardRoutingValidator validates that keyboard input is properly routed
// through the PTY->InputService path and ensures realistic testing behavior
type KeyboardRoutingValidator struct {
	// Core components
	ptyLayer     pty.PTYLayer
	inputService *InputService
	adapter      *PTYInputAdapter

	// Validation state
	ctx           context.Context
	cancel        context.CancelFunc
	validationMu  sync.RWMutex
	active        bool
	validationRun int

	// Test tracking
	testResults    []RoutingTestResult
	currentTest    *RoutingTestCase
	testStartTime  time.Time

	// Configuration
	config KeyboardRoutingValidatorConfig
}

// KeyboardRoutingValidatorConfig contains configuration for the validator
type KeyboardRoutingValidatorConfig struct {
	// Test parameters
	TestTimeout        time.Duration
	MaxTestCases      int
	EnableStatistics  bool
	EnableDebug       bool

	// Validation parameters
	StrictValidation   bool
	RequirePTYEnabled  bool
	CheckEventFlow     bool
	ValidateTiming     bool

	// Timing thresholds
	MaxKeyProcessingTime time.Duration
	MaxEventLatency      time.Duration
}

// DefaultKeyboardRoutingValidatorConfig returns a default configuration
func DefaultKeyboardRoutingValidatorConfig() KeyboardRoutingValidatorConfig {
	return KeyboardRoutingValidatorConfig{
		TestTimeout:          30 * time.Second,
		MaxTestCases:        100,
		EnableStatistics:    true,
		EnableDebug:         false,
		StrictValidation:    true,
		RequirePTYEnabled:   true,
		CheckEventFlow:       true,
		ValidateTiming:       true,
		MaxKeyProcessingTime: 100 * time.Millisecond,
		MaxEventLatency:      50 * time.Millisecond,
	}
}

// RoutingTestCase represents a single keyboard routing test case
type RoutingTestCase struct {
	ID          string
	Name        string
	Description string
	InputKey    tea.KeyMsg
	Context     string
	Expected    RoutingExpectation
	Timing      TimingExpectation
	Metadata    map[string]interface{}
}

// RoutingExpectation defines expected behavior for routing
type RoutingExpectation struct {
	ShouldHandle    bool
	ExpectedHandler string
	ShouldTrigger   []string
	ShouldNotTrigger []string
}

// TimingExpectation defines timing requirements
type TimingExpectation struct {
	MaxProcessingTime time.Duration
	MaxEventLatency   time.Duration
}

// RoutingTestResult contains the results of a routing test
type RoutingTestResult struct {
	TestCase       RoutingTestCase
	Success        bool
	ActualHandled  bool
	ActualHandlers []string
	Triggered      []string
	NotTriggered   []string
	Timing         TimingResult
	Error          error
	Duration       time.Duration
	Timestamp      time.Time
}

// TimingResult contains timing information
type TimingResult struct {
	InputTime      time.Time
	ProcessTime    time.Time
	CompleteTime   time.Time
	ProcessingTime time.Duration
	EventLatency   time.Duration
}

// NewKeyboardRoutingValidator creates a new keyboard routing validator
func NewKeyboardRoutingValidator(ptyLayer pty.PTYLayer, inputService *InputService, config KeyboardRoutingValidatorConfig) (*KeyboardRoutingValidator, error) {
	if ptyLayer == nil {
		return nil, fmt.Errorf("PTY layer is required")
	}
	if inputService == nil {
		return nil, fmt.Errorf("input service is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	validator := &KeyboardRoutingValidator{
		ptyLayer:     ptyLayer,
		inputService: inputService,
		ctx:          ctx,
		cancel:       cancel,
		config:       config,
		testResults:  make([]RoutingTestResult, 0),
	}

	return validator, nil
}

// Start begins the validation process
func (v *KeyboardRoutingValidator) Start() error {
	v.validationMu.Lock()
	defer v.validationMu.Unlock()

	if v.active {
		return fmt.Errorf("keyboard routing validator is already active")
	}

	// Ensure PTY is enabled if required
	if v.config.RequirePTYEnabled {
		if !v.inputService.IsPTYEnabled() {
			if err := v.inputService.EnablePTY(v.ptyLayer); err != nil {
				return fmt.Errorf("failed to enable PTY: %w", err)
			}
		}
	}

	// Create PTY input adapter if not provided
	if v.adapter == nil {
		adapterConfig := DefaultPTYInputAdapterConfig()
		adapterConfig.EnableDebug = v.config.EnableDebug

		var err error
		v.adapter, err = NewPTYInputAdapter(v.ptyLayer, v.inputService, adapterConfig)
		if err != nil {
			return fmt.Errorf("failed to create PTY input adapter: %w", err)
		}

		if err := v.adapter.Start(); err != nil {
			return fmt.Errorf("failed to start PTY input adapter: %w", err)
		}
	}

	v.active = true
	v.validationRun++

	if v.config.EnableDebug {
		fmt.Printf("Keyboard routing validator started (run %d)\n", v.validationRun)
	}

	return nil
}

// Stop stops the validation process
func (v *KeyboardRoutingValidator) Stop() error {
	v.validationMu.Lock()
	defer v.validationMu.Unlock()

	if !v.active {
		return nil
	}

	v.cancel()

	if v.adapter != nil {
		if err := v.adapter.Stop(); err != nil && v.config.EnableDebug {
			fmt.Printf("Warning: failed to stop PTY input adapter: %v\n", err)
		}
	}

	v.active = false

	if v.config.EnableDebug {
		fmt.Printf("Keyboard routing validator stopped (run %d)\n", v.validationRun)
	}

	return nil
}

// IsActive returns whether the validator is currently active
func (v *KeyboardRoutingValidator) IsActive() bool {
	v.validationMu.RLock()
	defer v.validationMu.RUnlock()
	return v.active
}

// ValidateKeyboardRouting runs comprehensive keyboard routing validation
func (v *KeyboardRoutingValidator) ValidateKeyboardRouting() ([]RoutingTestResult, error) {
	if !v.IsActive() {
		return nil, fmt.Errorf("validator is not active")
	}

	// Clear previous results
	v.testResults = make([]RoutingTestResult, 0)

	// Run test cases
	testCases := v.generateTestCases()
	if len(testCases) > v.config.MaxTestCases {
		testCases = testCases[:v.config.MaxTestCases]
	}

	for _, testCase := range testCases {
		result := v.runTestCase(testCase)
		v.testResults = append(v.testResults, result)

		if v.config.EnableDebug {
			status := "PASS"
			if !result.Success {
				status = "FAIL"
			}
			fmt.Printf("Test %s: %s (%s)\n", testCase.ID, status, result.Duration)
		}
	}

	return v.testResults, nil
}

// ValidatePTYToInputServicePath specifically tests PTY->InputService routing
func (v *KeyboardRoutingValidator) ValidatePTYToInputServicePath() error {
	if !v.IsActive() {
		return fmt.Errorf("validator is not active")
	}

	// Test basic PTY input flow
	testKeys := CreateTestKeySequence()

	for i, key := range testKeys {
		startTime := time.Now()

		// Route key through PTY->InputService path
		handled, _ := v.inputService.RouteKeyMessagePTY(key, "validation_test")

		processingTime := time.Since(startTime)

		if v.config.ValidateTiming && processingTime > v.config.MaxKeyProcessingTime {
			return fmt.Errorf("key %d processing time %v exceeds threshold %v", i, processingTime, v.config.MaxKeyProcessingTime)
		}

		if v.config.EnableDebug {
			fmt.Printf("PTY->InputService test key %d: handled=%v, time=%v\n", i, handled, processingTime)
		}
	}

	// Verify statistics
	stats := v.inputService.GetPTYStatistics()
	if stats.PTYKeyCount < uint64(len(testKeys)) {
		return fmt.Errorf("expected at least %d PTY keys, got %d", len(testKeys), stats.PTYKeyCount)
	}

	if stats.TotalKeyCount < uint64(len(testKeys)) {
		return fmt.Errorf("expected at least %d total keys, got %d", len(testKeys), stats.TotalKeyCount)
	}

	return nil
}

// ValidateEventFlow validates that events flow correctly through the system
func (v *KeyboardRoutingValidator) ValidateEventFlow() error {
	if !v.IsActive() || !v.config.CheckEventFlow {
		return nil
	}

	// This would ideally involve subscribing to event bus and verifying
	// that input events are published correctly
	// For now, we'll do a basic validation

	// Test that input events are published
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}

	// Capture initial statistics
	initialStats := v.inputService.GetPTYStatistics()

	// Route the key
	v.inputService.RouteKeyMessagePTY(testKey, "event_flow_test")

	// Allow some time for event processing
	time.Sleep(10 * time.Millisecond)

	// Verify that statistics were updated
	finalStats := v.inputService.GetPTYStatistics()

	if finalStats.TotalKeyCount <= initialStats.TotalKeyCount {
		return fmt.Errorf("expected key count to increase after routing")
	}

	return nil
}

// generateTestCases generates comprehensive test cases for keyboard routing
func (v *KeyboardRoutingValidator) generateTestCases() []RoutingTestCase {
	testCases := []RoutingTestCase{
		{
			ID:          "basic_char",
			Name:        "Basic Character Input",
			Description: "Test basic character routing through PTY->InputService",
			InputKey:    tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			Context:     "basic_test",
			Expected: RoutingExpectation{
				ShouldHandle: false, // No handlers by default
			},
			Timing: TimingExpectation{
				MaxProcessingTime: v.config.MaxKeyProcessingTime,
				MaxEventLatency:   v.config.MaxEventLatency,
			},
		},
		{
			ID:          "control_key",
			Name:        "Control Key Input",
			Description: "Test control key (Ctrl+C) routing",
			InputKey:    tea.KeyMsg{Type: tea.KeyCtrlC},
			Context:     "control_test",
			Expected: RoutingExpectation{
				ShouldHandle: true, // Ctrl+C should be handled for quit
			},
			Timing: TimingExpectation{
				MaxProcessingTime: v.config.MaxKeyProcessingTime,
				MaxEventLatency:   v.config.MaxEventLatency,
			},
		},
		{
			ID:          "navigation_key",
			Name:        "Navigation Key Input",
			Description: "Test navigation key (Up arrow) routing",
			InputKey:    tea.KeyMsg{Type: tea.KeyUp},
			Context:     "navigation_test",
			Expected: RoutingExpectation{
				ShouldHandle: false, // No navigation handlers by default
			},
			Timing: TimingExpectation{
				MaxProcessingTime: v.config.MaxKeyProcessingTime,
				MaxEventLatency:   v.config.MaxEventLatency,
			},
		},
		{
			ID:          "special_key",
			Name:        "Special Key Input",
			Description: "Test special key (Tab) routing",
			InputKey:    tea.KeyMsg{Type: tea.KeyTab},
			Context:     "special_test",
			Expected: RoutingExpectation{
				ShouldHandle: false, // No tab handlers by default
			},
			Timing: TimingExpectation{
				MaxProcessingTime: v.config.MaxKeyProcessingTime,
				MaxEventLatency:   v.config.MaxEventLatency,
			},
		},
	}

	// Add more complex test cases if needed
	if v.config.StrictValidation {
		// Add strict validation test cases
		strictCases := []RoutingTestCase{
			{
				ID:          "multi_char_sequence",
				Name:        "Multi-Character Sequence",
				Description: "Test routing of character sequence",
				InputKey:    tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}},
				Context:     "sequence_test",
				Expected: RoutingExpectation{
					ShouldHandle: false,
				},
				Timing: TimingExpectation{
					MaxProcessingTime: v.config.MaxKeyProcessingTime * 2,
					MaxEventLatency:   v.config.MaxEventLatency,
				},
			},
		}
		testCases = append(testCases, strictCases...)
	}

	return testCases
}

// runTestCase executes a single test case
func (v *KeyboardRoutingValidator) runTestCase(testCase RoutingTestCase) RoutingTestResult {
	v.validationMu.Lock()
	v.currentTest = &testCase
	v.testStartTime = time.Now()
	v.validationMu.Unlock()

	result := RoutingTestResult{
		TestCase:  testCase,
		Success:   true,
		Timestamp: time.Now(),
	}

	// Record input time
	timing := TimingResult{
		InputTime: time.Now(),
	}

	// Execute the test routing
	startTime := time.Now()
	handled, _ := v.inputService.RouteKeyMessagePTY(testCase.InputKey, testCase.Context)
	timing.ProcessTime = time.Now()
	result.ActualHandled = handled
	timing.ProcessingTime = timing.ProcessTime.Sub(startTime)

	// Record completion time
	timing.CompleteTime = time.Now()
	timing.EventLatency = timing.CompleteTime.Sub(timing.InputTime)
	result.Timing = timing

	// Validate results
	if testCase.Expected.ShouldHandle != handled {
		result.Success = false
		result.Error = fmt.Errorf("expected handled=%v, got handled=%v", testCase.Expected.ShouldHandle, handled)
	}

	// Validate timing if required
	if v.config.ValidateTiming {
		if timing.ProcessingTime > testCase.Timing.MaxProcessingTime {
			result.Success = false
			if result.Error == nil {
				result.Error = fmt.Errorf("processing time %v exceeds limit %v", timing.ProcessingTime, testCase.Timing.MaxProcessingTime)
			} else {
				result.Error = fmt.Errorf("%v; processing time %v exceeds limit %v", result.Error, timing.ProcessingTime, testCase.Timing.MaxProcessingTime)
			}
		}

		if timing.EventLatency > testCase.Timing.MaxEventLatency {
			result.Success = false
			if result.Error == nil {
				result.Error = fmt.Errorf("event latency %v exceeds limit %v", timing.EventLatency, testCase.Timing.MaxEventLatency)
			} else {
				result.Error = fmt.Errorf("%v; event latency %v exceeds limit %v", result.Error, timing.EventLatency, testCase.Timing.MaxEventLatency)
			}
		}
	}

	result.Duration = time.Since(v.testStartTime)

	v.validationMu.Lock()
	v.currentTest = nil
	v.validationMu.Unlock()

	return result
}

// GetTestResults returns all test results from the last validation run
func (v *KeyboardRoutingValidator) GetTestResults() []RoutingTestResult {
	v.validationMu.RLock()
	defer v.validationMu.RUnlock()

	// Return a copy to prevent modification
	results := make([]RoutingTestResult, len(v.testResults))
	copy(results, v.testResults)
	return results
}

// GetValidationStatistics returns statistics about validation runs
func (v *KeyboardRoutingValidator) GetValidationStatistics() ValidationStatistics {
	v.validationMu.RLock()
	defer v.validationMu.RUnlock()

	stats := ValidationStatistics{
		ValidationRun:  v.validationRun,
		TotalTests:     len(v.testResults),
		Active:         v.active,
	}

	for _, result := range v.testResults {
		if result.Success {
			stats.PassedTests++
		} else {
			stats.FailedTests++
		}

		if result.Duration > stats.MaxTestDuration {
			stats.MaxTestDuration = result.Duration
		}

		if result.Duration < stats.MinTestDuration || stats.MinTestDuration == 0 {
			stats.MinTestDuration = result.Duration
		}

		stats.TotalTestDuration += result.Duration
	}

	if stats.TotalTests > 0 {
		stats.AverageTestDuration = stats.TotalTestDuration / time.Duration(stats.TotalTests)
		stats.SuccessRate = float64(stats.PassedTests) / float64(stats.TotalTests)
	}

	return stats
}

// ValidationStatistics contains statistics about validation runs
type ValidationStatistics struct {
	ValidationRun      int
	TotalTests         int
	PassedTests        int
	FailedTests        int
	SuccessRate        float64
	MinTestDuration    time.Duration
	MaxTestDuration    time.Duration
	AverageTestDuration time.Duration
	TotalTestDuration  time.Duration
	Active             bool
}

// GetCurrentTest returns the currently running test case
func (v *KeyboardRoutingValidator) GetCurrentTest() *RoutingTestCase {
	v.validationMu.RLock()
	defer v.validationMu.RUnlock()
	if v.currentTest == nil {
		return nil
	}
	// Return a copy
	testCopy := *v.currentTest
	return &testCopy
}