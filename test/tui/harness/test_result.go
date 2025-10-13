package harness

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TestResultManager manages test results
type TestResultManager struct {
	mu       sync.RWMutex
	results  []TestResult
	metrics  TestMetrics
	summary  TestSummary
}

// NewTestResultManager creates a new test result manager
func NewTestResultManager() *TestResultManager {
	return &TestResultManager{
		results: make([]TestResult, 0),
		metrics: TestMetrics{},
		summary: TestSummary{},
	}
}

// AddResult adds a test result
func (trm *TestResultManager) AddResult(result TestResult) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	// Validate result
	if err := trm.validateResult(result); err != nil {
		return fmt.Errorf("invalid test result: %w", err)
	}

	// Set timestamp if not set
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	// Add to results
	trm.results = append(trm.results, result)

	// Update metrics
	trm.updateMetrics()

	// Update summary
	trm.updateSummary()

	return nil
}

// GetResult returns a test result by ID
func (trm *TestResultManager) GetResult(testID string) (TestResult, error) {
	trm.mu.RLock()
	defer trm.mu.RUnlock()

	for _, result := range trm.results {
		if result.TestID == testID {
			return result, nil
		}
	}

	return TestResult{}, fmt.Errorf("test result not found: %s", testID)
}

// GetResults returns all test results
func (trm *TestResultManager) GetResults() []TestResult {
	trm.mu.RLock()
	defer trm.mu.RUnlock()

	results := make([]TestResult, len(trm.results))
	copy(results, trm.results)
	return results
}

// GetResultsByStatus returns results filtered by status
func (trm *TestResultManager) GetResultsByStatus(status TestStatus) []TestResult {
	trm.mu.RLock()
	defer trm.mu.RUnlock()

	var filtered []TestResult
	for _, result := range trm.results {
		if result.Status == status {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// GetMetrics returns the current metrics
func (trm *TestResultManager) GetMetrics() TestMetrics {
	trm.mu.RLock()
	defer trm.mu.RUnlock()

	return trm.metrics
}

// GetSummary returns the current summary
func (trm *TestResultManager) GetSummary() TestSummary {
	trm.mu.RLock()
	defer trm.mu.RUnlock()

	return trm.summary
}

// ClearResults clears all test results
func (trm *TestResultManager) ClearResults() {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	trm.results = trm.results[:0]
	trm.metrics = TestMetrics{}
	trm.summary = TestSummary{}
}

// validateResult validates a test result
func (trm *TestResultManager) validateResult(result TestResult) error {
	if result.TestID == "" {
		return fmt.Errorf("test ID cannot be empty")
	}

	if result.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}

	// Validate status
	switch result.Status {
	case TestStatusPass, TestStatusFail, TestStatusError, TestStatusSkip, TestStatusRunning:
		// Valid statuses
	default:
		return fmt.Errorf("invalid test status: %s", result.Status)
	}

	// Validate duration
	if result.Duration < 0 {
		return fmt.Errorf("duration cannot be negative")
	}

	return nil
}

// updateMetrics updates the metrics based on current results
func (trm *TestResultManager) updateMetrics() {
	trm.metrics = TestMetrics{
		InputEvents:      0,
		StateValidations: 0,
		ViewChanges:      0,
		Duration:         0,
		MemoryUsed:       0,
	}

	var totalDuration time.Duration
	for _, result := range trm.results {
		trm.metrics.InputEvents += result.Metrics.InputEvents
		trm.metrics.StateValidations += result.Metrics.StateValidations
		trm.metrics.ViewChanges += result.Metrics.ViewChanges
		totalDuration += result.Duration
	}

	trm.metrics.Duration = totalDuration
}

// updateSummary updates the summary based on current results
func (trm *TestResultManager) updateSummary() {
	trm.summary = TestSummary{
		TotalTests:  len(trm.results),
		GeneratedAt: time.Now(),
		Duration:   trm.metrics.Duration,
	}

	for _, result := range trm.results {
		switch result.Status {
		case TestStatusPass:
			trm.summary.PassedTests++
		case TestStatusFail:
			trm.summary.FailedTests++
		case TestStatusError:
			trm.summary.ErrorTests++
		case TestStatusSkip:
			trm.summary.SkippedTests++
		}
	}

	if trm.summary.TotalTests > 0 {
		trm.summary.SuccessRate = float64(trm.summary.PassedTests) / float64(trm.summary.TotalTests) * 100
	}
}

// TestResultBuilder helps build test results
type TestResultBuilder struct {
	result TestResult
}

// NewTestResultBuilder creates a new test result builder
func NewTestResultBuilder() *TestResultBuilder {
	return &TestResultBuilder{
		result: TestResult{
			StartTime: time.Now(),
			Metrics:  TestMetrics{},
		},
	}
}

// WithTestID sets the test ID
func (trb *TestResultBuilder) WithTestID(testID string) *TestResultBuilder {
	trb.result.TestID = testID
	return trb
}

// WithAction sets the action
func (trb *TestResultBuilder) WithAction(action string) *TestResultBuilder {
	trb.result.Action = action
	return trb
}

// WithStatus sets the status
func (trb *TestResultBuilder) WithStatus(status TestStatus) *TestResultBuilder {
	trb.result.Status = status
	return trb
}

// WithError sets the error message
func (trb *TestResultBuilder) WithError(err error) *TestResultBuilder {
	if err != nil {
		trb.result.Error = err.Error()
	}
	return trb
}

// WithDescription sets the description
func (trb *TestResultBuilder) WithDescription(description string) *TestResultBuilder {
	trb.result.Description = description
	return trb
}

// WithExpectedState sets the expected state
func (trb *TestResultBuilder) WithExpectedState(state ExpectedState) *TestResultBuilder {
	trb.result.ExpectedState = state
	return trb
}

// WithActualState sets the actual state
func (trb *TestResultBuilder) WithActualState(state ActualState) *TestResultBuilder {
	trb.result.ActualState = state
	return trb
}

// WithMetrics sets the metrics
func (trb *TestResultBuilder) WithMetrics(metrics TestMetrics) *TestResultBuilder {
	trb.result.Metrics = metrics
	return trb
}

// WithStartTime sets the start time
func (trb *TestResultBuilder) WithStartTime(startTime time.Time) *TestResultBuilder {
	trb.result.StartTime = startTime
	return trb
}

// WithTimestamp sets the timestamp
func (trb *TestResultBuilder) WithTimestamp(timestamp time.Time) *TestResultBuilder {
	trb.result.Timestamp = timestamp
	return trb
}

// Build builds the test result
func (trb *TestResultBuilder) Build() TestResult {
	// Auto-calculate duration if not set
	if trb.result.Duration == 0 && !trb.result.StartTime.IsZero() {
		trb.result.Duration = time.Since(trb.result.StartTime)
	}

	// Set timestamp if not set
	if trb.result.Timestamp.IsZero() {
		trb.result.Timestamp = time.Now()
	}

	return trb.result
}

// TestResultSerializer serializes and deserializes test results
type TestResultSerializer struct{}

// NewTestResultSerializer creates a new test result serializer
func NewTestResultSerializer() *TestResultSerializer {
	return &TestResultSerializer{}
}

// Serialize serializes a test result to JSON
func (trs *TestResultSerializer) Serialize(result TestResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

// SerializeArray serializes multiple test results to JSON
func (trs *TestResultSerializer) SerializeArray(results []TestResult) ([]byte, error) {
	return json.MarshalIndent(results, "", "  ")
}

// Deserialize deserializes JSON to a test result
func (trs *TestResultSerializer) Deserialize(data []byte) (TestResult, error) {
	var result TestResult
	err := json.Unmarshal(data, &result)
	return result, err
}

// DeserializeArray deserializes JSON to multiple test results
func (trs *TestResultSerializer) DeserializeArray(data []byte) ([]TestResult, error) {
	var results []TestResult
	err := json.Unmarshal(data, &results)
	return results, err
}

// TestResultReporter generates reports from test results
type TestResultReporter struct {
	serializer *TestResultSerializer
}

// NewTestResultReporter creates a new test result reporter
func NewTestResultReporter() *TestResultReporter {
	return &TestResultReporter{
		serializer: NewTestResultSerializer(),
	}
}

// GenerateTextReport generates a text report
func (trr *TestResultReporter) GenerateTextReport(results []TestResult) (string, error) {
	if len(results) == 0 {
		return "No test results to report", nil
	}

	report := "Test Results Report\n"
	report += "==================\n\n"

	// Summary
	summary := trr.generateSummary(results)
	report += fmt.Sprintf("Total Tests: %d\n", summary.TotalTests)
	report += fmt.Sprintf("Passed: %d\n", summary.PassedTests)
	report += fmt.Sprintf("Failed: %d\n", summary.FailedTests)
	report += fmt.Sprintf("Errors: %d\n", summary.ErrorTests)
	report += fmt.Sprintf("Success Rate: %.1f%%\n", summary.SuccessRate)
	report += fmt.Sprintf("Total Duration: %s\n\n", summary.Duration)

	// Individual results
	report += "Individual Results:\n"
	report += "------------------\n"

	for i, result := range results {
		report += fmt.Sprintf("%d. %s (%s)\n", i+1, result.Action, result.Status)
		report += fmt.Sprintf("   Test ID: %s\n", result.TestID)
		report += fmt.Sprintf("   Duration: %s\n", result.Duration)
		if result.Description != "" {
			report += fmt.Sprintf("   Description: %s\n", result.Description)
		}
		if result.Error != "" {
			report += fmt.Sprintf("   Error: %s\n", result.Error)
		}
		report += "\n"
	}

	return report, nil
}

// generateSummary generates a summary from results
func (trr *TestResultReporter) generateSummary(results []TestResult) TestSummary {
	summary := TestSummary{
		TotalTests: len(results),
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

// SaveToFile saves results to a file
func (trr *TestResultReporter) SaveToFile(results []TestResult, filename string) error {
	data, err := trr.serializer.SerializeArray(results)
	if err != nil {
		return fmt.Errorf("failed to serialize results: %w", err)
	}

	return writeToFile(filename, data)
}

// LoadFromFile loads results from a file
func (trr *TestResultReporter) LoadFromFile(filename string) ([]TestResult, error) {
	data, err := readFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return trr.serializer.DeserializeArray(data)
}

// writeToFile writes data to a file
func writeToFile(filename string, data []byte) error {
	// This would use os.WriteFile in a real implementation
	return fmt.Errorf("file I/O not implemented in this example")
}

// readFromFile reads data from a file
func readFromFile(filename string) ([]byte, error) {
	// This would use os.ReadFile in a real implementation
	return nil, fmt.Errorf("file I/O not implemented in this example")
}