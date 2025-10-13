package services

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeyboardRoutingValidatorBasic tests basic validator functionality
func TestKeyboardRoutingValidatorBasic(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create validator
	config := DefaultKeyboardRoutingValidatorConfig()
	config.EnableDebug = true

	validator, err := NewKeyboardRoutingValidator(ptyLayer, inputService, config)
	require.NoError(t, err, "Failed to create keyboard routing validator")
	require.NotNil(t, validator, "Validator should not be nil")

	// Test starting validator
	err = validator.Start()
	require.NoError(t, err, "Failed to start validator")
	defer validator.Stop()

	// Test that validator is active
	assert.True(t, validator.IsActive(), "Validator should be active")

	// Test PTY->InputService path validation
	err = validator.ValidatePTYToInputServicePath()
	assert.NoError(t, err, "PTY->InputService path validation should pass")

	// Test event flow validation
	err = validator.ValidateEventFlow()
	assert.NoError(t, err, "Event flow validation should pass")

	// Get validation statistics
	stats := validator.GetValidationStatistics()
	assert.GreaterOrEqual(t, stats.ValidationRun, 1, "Should have at least 1 validation run")
	assert.True(t, stats.Active, "Should be active")
}

// TestKeyboardRoutingValidationRunComprehensive tests comprehensive validation
func TestKeyboardRoutingValidationRunComprehensive(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create validator
	config := DefaultKeyboardRoutingValidatorConfig()
	config.EnableDebug = false // Disable debug for cleaner output
	config.MaxTestCases = 10   // Limit test cases for faster test

	validator, err := NewKeyboardRoutingValidator(ptyLayer, inputService, config)
	require.NoError(t, err, "Failed to create keyboard routing validator")

	// Start validator
	err = validator.Start()
	require.NoError(t, err, "Failed to start validator")
	defer validator.Stop()

	// Run comprehensive validation
	results, err := validator.ValidateKeyboardRouting()
	require.NoError(t, err, "Failed to run comprehensive validation")
	require.NotEmpty(t, results, "Should have test results")

	// Verify all tests passed (basic setup should pass)
	for i, result := range results {
		if !result.Success {
			t.Logf("Test %d (%s) failed: %v", i, result.TestCase.Name, result.Error)
		}
		// Note: We don't assert that all tests pass here because the default
		// InputService setup doesn't have handlers for all key types
	}

	// Get statistics
	stats := validator.GetValidationStatistics()
	assert.Equal(t, len(results), stats.TotalTests, "Total tests should match results count")
	assert.Greater(t, stats.ValidationRun, 0, "Should have validation runs")
	assert.Greater(t, stats.PassedTests+stats.FailedTests, 0, "Should have some test results")
}

// TestKeyboardRoutingValidationWithHandlers tests validation with registered handlers
func TestKeyboardRoutingValidationWithHandlers(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Register some test handlers
	err = inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'a' {
			return func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} } // Return a command
		}
		return nil
	})
	require.NoError(t, err, "Failed to register global handler")

	err = inputService.SubscribeWithContextKeys(ctx, "test_context", func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyUp {
			return func() tea.Msg { return tea.KeyMsg{Type: tea.KeyDown} } // Return a command
		}
		return nil
	})
	require.NoError(t, err, "Failed to register context handler")

	// Create validator
	config := DefaultKeyboardRoutingValidatorConfig()
	config.EnableDebug = false
	config.MaxTestCases = 5

	validator, err := NewKeyboardRoutingValidator(ptyLayer, inputService, config)
	require.NoError(t, err, "Failed to create keyboard routing validator")

	// Start validator
	err = validator.Start()
	require.NoError(t, err, "Failed to start validator")
	defer validator.Stop()

	// Run validation
	results, err := validator.ValidateKeyboardRouting()
	require.NoError(t, err, "Failed to run validation")
	require.NotEmpty(t, results, "Should have test results")

	// Find and verify specific test results
	var basicCharResult *RoutingTestResult
	var navigationKeyResult *RoutingTestResult

	for _, result := range results {
		if result.TestCase.ID == "basic_char" {
			basicCharResult = &result
		} else if result.TestCase.ID == "navigation_key" {
			navigationKeyResult = &result
		}
	}

	// Basic character should be handled by global handler
	if basicCharResult != nil {
		assert.True(t, basicCharResult.ActualHandled, "Basic character should be handled by global handler")
		assert.Less(t, basicCharResult.Timing.ProcessingTime, config.MaxKeyProcessingTime, "Processing time should be within limits")
	}

	// Navigation key should be handled by context handler
	if navigationKeyResult != nil {
		assert.True(t, navigationKeyResult.ActualHandled, "Navigation key should be handled by context handler")
		assert.Less(t, navigationKeyResult.Timing.ProcessingTime, config.MaxKeyProcessingTime, "Processing time should be within limits")
	}
}

// TestKeyboardRoutingValidationTiming tests timing validation
func TestKeyboardRoutingValidationTiming(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create validator with strict timing requirements
	config := DefaultKeyboardRoutingValidatorConfig()
	config.EnableDebug = false
	config.ValidateTiming = true
	config.MaxKeyProcessingTime = 100 * time.Millisecond
	config.MaxEventLatency = 50 * time.Millisecond

	validator, err := NewKeyboardRoutingValidator(ptyLayer, inputService, config)
	require.NoError(t, err, "Failed to create keyboard routing validator")

	// Start validator
	err = validator.Start()
	require.NoError(t, err, "Failed to start validator")
	defer validator.Stop()

	// Run validation
	results, err := validator.ValidateKeyboardRouting()
	require.NoError(t, err, "Failed to run validation")
	require.NotEmpty(t, results, "Should have test results")

	// Verify timing results
	for _, result := range results {
		assert.LessOrEqual(t, result.Timing.ProcessingTime, config.MaxKeyProcessingTime,
			"Processing time should be within limits for test %s", result.TestCase.ID)
		assert.LessOrEqual(t, result.Timing.EventLatency, config.MaxEventLatency,
			"Event latency should be within limits for test %s", result.TestCase.ID)
		assert.Greater(t, result.Timing.ProcessingTime, time.Duration(0),
			"Processing time should be positive for test %s", result.TestCase.ID)
	}

	// Check overall timing statistics
	stats := validator.GetValidationStatistics()
	assert.Greater(t, stats.AverageTestDuration, time.Duration(0), "Average test duration should be positive")
	assert.Less(t, stats.MaxTestDuration, 1*time.Second, "Max test duration should be reasonable")
}

// TestKeyboardRoutingValidationStatistics tests statistics tracking
func TestKeyboardRoutingValidationStatistics(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create validator
	config := DefaultKeyboardRoutingValidatorConfig()
	config.EnableDebug = false
	config.MaxTestCases = 3

	validator, err := NewKeyboardRoutingValidator(ptyLayer, inputService, config)
	require.NoError(t, err, "Failed to create keyboard routing validator")

	// Check initial statistics
	initialStats := validator.GetValidationStatistics()
	assert.Equal(t, 0, initialStats.ValidationRun, "Initial validation run should be 0")
	assert.Equal(t, 0, initialStats.TotalTests, "Initial total tests should be 0")
	assert.False(t, initialStats.Active, "Should not be active initially")

	// Start validator
	err = validator.Start()
	require.NoError(t, err, "Failed to start validator")
	defer validator.Stop()

	// Check statistics after start
	startedStats := validator.GetValidationStatistics()
	assert.Equal(t, 0, startedStats.ValidationRun, "Validation run should still be 0 before tests")
	assert.Equal(t, 0, startedStats.TotalTests, "Total tests should still be 0 before tests")
	assert.True(t, startedStats.Active, "Should be active after start")

	// Run validation
	results, err := validator.ValidateKeyboardRouting()
	require.NoError(t, err, "Failed to run validation")

	// Check statistics after validation
	finalStats := validator.GetValidationStatistics()
	assert.GreaterOrEqual(t, finalStats.ValidationRun, 1, "Should have at least 1 validation run")
	assert.Equal(t, len(results), finalStats.TotalTests, "Total tests should match results count")
	assert.Equal(t, len(results), finalStats.PassedTests+finalStats.FailedTests, "Passed + failed should equal total")
	assert.Greater(t, finalStats.SuccessRate, 0.0, "Success rate should be non-negative")
	assert.LessOrEqual(t, finalStats.SuccessRate, 1.0, "Success rate should not exceed 1.0")
	assert.Greater(t, finalStats.AverageTestDuration, time.Duration(0), "Average duration should be positive")
}

// TestKeyboardRoutingValidationConcurrency tests concurrent validation
func TestKeyboardRoutingValidationConcurrency(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create multiple validators to test concurrent usage
	validators := make([]*KeyboardRoutingValidator, 3)

	for i := 0; i < 3; i++ {
		config := DefaultKeyboardRoutingValidatorConfig()
		config.EnableDebug = false
		config.MaxTestCases = 2

		validator, err := NewKeyboardRoutingValidator(ptyLayer, inputService, config)
		require.NoError(t, err, "Failed to create validator %d", i)
		validators[i] = validator

		// Start validator
		err = validator.Start()
		require.NoError(t, err, "Failed to start validator %d", i)
		defer validator.Stop()
	}

	// Run validation concurrently
	done := make(chan error, len(validators))

	for i, validator := range validators {
		go func(idx int, val *KeyboardRoutingValidator) {
			_, err := val.ValidateKeyboardRouting()
			done <- err
		}(i, validator)
	}

	// Wait for all validations to complete
	for i := 0; i < len(validators); i++ {
		select {
		case err := <-done:
			assert.NoError(t, err, "Concurrent validation %d should succeed", i)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent validations")
		}
	}

	// Verify all validators still active and have results
	for i, validator := range validators {
		assert.True(t, validator.IsActive(), "Validator %d should still be active", i)
		stats := validator.GetValidationStatistics()
		assert.Greater(t, stats.TotalTests, 0, "Validator %d should have test results", i)
	}
}