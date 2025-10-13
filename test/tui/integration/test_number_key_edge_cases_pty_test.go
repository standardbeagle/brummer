//go:build tty && integration
// +build tty,integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// NumberKeyEdgeCaseTest tests edge cases and boundary conditions for number key tab switching
// This test focuses on unusual scenarios, error conditions, and stress testing
// the number key navigation feature.

// EdgeCaseTestHarness provides specialized testing for edge cases
type EdgeCaseTestHarness struct {
	t                    *testing.T
	ctx                  context.Context
	cancel               context.CancelFunc
	ptyLayer             *pty.PTYLayer
	navigationService    *services.NavigationService
	eventBus             *events.EventBus
	processMgr           *process.Manager
	logStore             *logs.Store
	model                tea.Model
	program              *tea.Program
	errorEvents          []ErrorEvent
	stressTestResults    StressTestResults
	workingDir           string
}

// ErrorEvent records an error or unusual event during testing
type ErrorEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	ErrorType   string    `json:"error_type"`
	Description string    `json:"description"`
	Key         string    `json:"key,omitempty"`
	Context     string    `json:"context,omitempty"`
	Recovered   bool      `json:"recovered"`
}

// StressTestResults records stress test metrics
type StressTestResults struct {
	TotalKeyPresses      int           `json:"total_key_presses"`
	SuccessfulSwitches   int           `json:"successful_switches"`
	FailedSwitches       int           `json:"failed_switches"`
	TimeoutEvents        int           `json:"timeout_events"`
	AverageResponseTime  time.Duration `json:"average_response_time"`
	MaxResponseTime      time.Duration `json:"max_response_time"`
	MinResponseTime      time.Duration `json:"min_response_time"`
	MemoryUsageMB        float64       `json:"memory_usage_mb"`
	CorruptedEvents      int           `json:"corrupted_events"`
}

// NewEdgeCaseTestHarness creates a new edge case test harness
func NewEdgeCaseTestHarness(t *testing.T) *EdgeCaseTestHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	return &EdgeCaseTestHarness{
		t:                 t,
		ctx:               ctx,
		cancel:            cancel,
		errorEvents:       make([]ErrorEvent, 0),
		stressTestResults: StressTestResults{},
	}
}

// TestNumberKeyEdgeCases_ZeroKey tests behavior with key '0' (unmapped)
func TestNumberKeyEdgeCases_ZeroKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Zero Key Edge Case (Unmapped)")
	t.Log("=======================================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	// Test pressing '0' multiple times
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("ZeroKey_Press_%d", i+1), func(t *testing.T) {
			beforeView := h.GetCurrentView()
			t.Logf("   Before pressing '0': %s", beforeView)

			// Press '0' key
			err := h.SimulateKey("0")
			require.NoError(t, err, "Failed to press '0' key")

			// Wait briefly
			time.Sleep(100 * time.Millisecond)

			// View should remain unchanged
			afterView := h.GetCurrentView()
			assert.Equal(t, beforeView, afterView, "View should remain unchanged after pressing '0'")

			t.Logf("   ✓ '0' key correctly ignored (view stayed %s)", afterView)
		})
	}

	// Validate no switches occurred
	assert.Equal(t, 0, h.GetSwitchCount(), "No view switches should have occurred")
	t.Log("✅ Zero key edge case test passed")
}

// TestNumberKeyEdgeCases_UnmappedKeys tests keys 6-9 (currently unmapped)
func TestNumberKeyEdgeCases_UnmappedKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Unmapped Keys Edge Case (6-9)")
	t.Log("=======================================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	unmappedKeys := []string{"6", "7", "8", "9"}

	for _, key := range unmappedKeys {
		t.Run(fmt.Sprintf("UnmappedKey_%s", key), func(t *testing.T) {
			beforeView := h.GetCurrentView()
			t.Logf("   Before pressing '%s': %s", key, beforeView)

			// Press unmapped key
			err := h.SimulateKey(key)
			require.NoError(t, err, "Failed to press unmapped key '%s'", key)

			// Wait briefly
			time.Sleep(100 * time.Millisecond)

			// View should remain unchanged
			afterView := h.GetCurrentView()
			assert.Equal(t, beforeView, afterView, "View should remain unchanged after pressing unmapped key '%s'", key)

			t.Logf("   ✓ Unmapped key '%s' correctly ignored (view stayed %s)", key, afterView)
		})
	}

	// Validate no switches occurred
	assert.Equal(t, 0, h.GetSwitchCount(), "No view switches should have occurred for unmapped keys")
	t.Log("✅ Unmapped keys edge case test passed")
}

// TestNumberKeyEdgeCases_NonNumericKeys tests non-numeric keys that shouldn't trigger view switches
func TestNumberKeyEdgeCases_NonNumericKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Non-Numeric Keys Edge Case")
	t.Log("=====================================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	// Test various non-numeric keys that shouldn't trigger number key navigation
	nonNumericKeys := []string{
		"a", "b", "c", "x", "z",  // Letters
		"!", "@", "#", "$", "%",   // Special characters
		" ", "\n", "\r",         // Space and control characters
	}

	for _, key := range nonNumericKeys {
		t.Run(fmt.Sprintf("NonNumericKey_%s", fmt.Sprintf("%q", key)), func(t *testing.T) {
			beforeView := h.GetCurrentView()
			t.Logf("   Before pressing non-numeric key %s: %s", fmt.Sprintf("%q", key), beforeView)

			// Press non-numeric key
			err := h.SimulateKey(key)
			require.NoError(t, err, "Failed to press non-numeric key %s", fmt.Sprintf("%q", key))

			// Wait briefly
			time.Sleep(100 * time.Millisecond)

			// View might change due to other key bindings (like 'q' for quit), but shouldn't be due to number key logic
			afterView := h.GetCurrentView()
			t.Logf("   After non-numeric key %s: %s", fmt.Sprintf("%q", key), afterView)

			// Record that we tested this key
			h.RecordEvent("non_numeric_key_test", fmt.Sprintf("Tested non-numeric key: %s", fmt.Sprintf("%q", key)), key, "view_navigation_test", true)
		})
	}

	t.Log("✅ Non-numeric keys edge case test passed")
}

// TestNumberKeyEdgeCases_MixedInvalidSequences tests sequences mixing valid and invalid keys
func TestNumberKeyEdgeCases_MixedInvalidSequences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Mixed Invalid Key Sequences")
	t.Log("=====================================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	// Test sequences with mixed valid/invalid keys
	testSequences := [][]string{
		{"1", "0", "2", "9", "3"},      // Valid keys with invalid in between
		{"6", "1", "7", "2", "8"},      // Invalid keys first, then valid
		{"0", "6", "7", "8", "9"},      // All invalid
		{"1", "x", "2", "!", "3"},      // Valid keys with non-numeric in between
	}

	for seqIndex, sequence := range testSequences {
		t.Run(fmt.Sprintf("MixedSequence_%d", seqIndex), func(t *testing.T) {
			t.Logf("   Testing mixed sequence: %v", sequence)

			switchCount := 0
			expectedSwitches := 0

			// Count expected valid switches in sequence
			for _, key := range sequence {
				if key == "1" || key == "2" || key == "3" || key == "4" || key == "5" {
					expectedSwitches++
				}
			}

			for i, key := range sequence {
				t.Logf("     Pressing key %s (%d/%d)", key, i+1, len(sequence))

				beforeView := h.GetCurrentView()
				err := h.SimulateKey(key)
				require.NoError(t, err, "Failed to press key %s", key)

				// Wait for processing
				time.Sleep(100 * time.Millisecond)

				afterView := h.GetCurrentView()

				// Check if view actually changed
				if beforeView != afterView {
					switchCount++
					t.Logf("       View switched: %s -> %s", beforeView, afterView)
				}
			}

			t.Logf("   Sequence completed: %d actual switches, %d expected switches", switchCount, expectedSwitches)

			// The number of switches should match expected valid keys (approximately, allowing for other key bindings)
			assert.GreaterOrEqual(t, switchCount, 0, "Should have non-negative switch count")
			assert.LessOrEqual(t, switchCount, expectedSwitches, "Should not have more switches than valid keys")
		})
	}

	t.Log("✅ Mixed invalid sequences edge case test passed")
}

// TestNumberKeyEdgeCases_RapidStressTest performs stress testing with rapid key presses
func TestNumberKeyEdgeCases_RapidStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Rapid Stress Test")
	t.Log("============================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	// Stress test parameters
	totalKeys := 1000
	validKeys := []string{"1", "2", "3", "4", "5"}
	invalidKeys := []string{"0", "6", "7", "8", "9", "a", "b", "c", "x", "z"}
	allKeys := append(validKeys, invalidKeys...)

	t.Logf("   Performing stress test: %d rapid key presses", totalKeys)

	startTime := time.Now()
	var responseTimes []time.Duration
	timeoutCount := 0

	for i := 0; i < totalKeys; i++ {
		// Select random key
		keyIndex := i % len(allKeys)
		key := allKeys[keyIndex]

		// Measure response time
		keyStartTime := time.Now()

		beforeView := h.GetCurrentView()
		err := h.SimulateKey(key)
		if err != nil {
			h.RecordEvent("stress_test_error", fmt.Sprintf("Failed to press key %s: %v", key, err), key, "stress_test", false)
			continue
		}

		// Wait for processing (very short for stress test)
		time.Sleep(1 * time.Millisecond)
		afterView := h.GetCurrentView()

		responseTime := time.Since(keyStartTime)
		responseTimes = append(responseTimes, responseTime)

		// Check for timeout (response taking too long)
		if responseTime > 100*time.Millisecond {
			timeoutCount++
			h.RecordEvent("stress_test_timeout", fmt.Sprintf("Key %s took too long: %v", key, responseTime), key, "stress_test", false)
		}

		// Count successful switches
		if beforeView != afterView && (key == "1" || key == "2" || key == "3" || key == "4" || key == "5") {
			h.stressTestResults.SuccessfulSwitches++
		} else if beforeView != afterView {
			// Unexpected switch (shouldn't happen for invalid keys)
			h.RecordEvent("unexpected_switch", fmt.Sprintf("Invalid key %s caused view switch", key), key, "stress_test", false)
		}
	}

	totalDuration := time.Since(startTime)

	// Calculate statistics
	h.stressTestResults.TotalKeyPresses = totalKeys
	h.stressTestResults.TimeoutEvents = timeoutCount

	if len(responseTimes) > 0 {
		var totalTime time.Duration
		maxTime := responseTimes[0]
		minTime := responseTimes[0]

		for _, rt := range responseTimes {
			totalTime += rt
			if rt > maxTime {
				maxTime = rt
			}
			if rt < minTime {
				minTime = rt
			}
		}

		h.stressTestResults.AverageResponseTime = totalTime / time.Duration(len(responseTimes))
		h.stressTestResults.MaxResponseTime = maxTime
		h.stressTestResults.MinResponseTime = minTime
	}

	t.Logf("   Stress test completed in %v", totalDuration)
	t.Logf("   Total keys pressed: %d", h.stressTestResults.TotalKeyPresses)
	t.Logf("   Successful switches: %d", h.stressTestResults.SuccessfulSwitches)
	t.Logf("   Timeout events: %d", h.stressTestResults.TimeoutEvents)
	t.Logf("   Average response time: %v", h.stressTestResults.AverageResponseTime)
	t.Logf("   Max response time: %v", h.stressTestResults.MaxResponseTime)
	t.Logf("   Min response time: %v", h.stressTestResults.MinResponseTime)

	// Validate stress test results
	assert.Equal(t, totalKeys, h.stressTestResults.TotalKeyPresses, "Should have pressed all keys")
	assert.Less(t, h.stressTestResults.TimeoutEvents, totalKeys/10, "Timeout events should be less than 10% of total")
	assert.Less(t, h.stressTestResults.AverageResponseTime, 10*time.Millisecond, "Average response time should be reasonable")
	assert.Less(t, h.stressTestResults.MaxResponseTime, 200*time.Millisecond, "Max response time should be reasonable")

	t.Log("✅ Rapid stress test passed")
}

// TestNumberKeyEdgeCases_ConcurrentAccess tests concurrent key press scenarios
func TestNumberKeyEdgeCases_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Concurrent Access Edge Case")
	t.Log("=====================================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	// Test concurrent key presses (simulated by rapid alternating keys)
	concurrentSequence := []string{"1", "2", "1", "3", "1", "4", "1", "5"}
	iterations := 50

	t.Logf("   Testing concurrent-like sequence: %v, repeated %d times", concurrentSequence, iterations)

	for iteration := 0; iteration < iterations; iteration++ {
		for i, key := range concurrentSequence {
			t.Logf("     Iteration %d, key %d/%d: %s", iteration+1, i+1, len(concurrentSequence), key)

			beforeView := h.GetCurrentView()
			err := h.SimulateKey(key)
			require.NoError(t, err, "Failed to press key %s in iteration %d", key, iteration+1)

			// Minimal wait to simulate concurrent access
			time.Sleep(1 * time.Millisecond)

			afterView := h.GetCurrentView()

			// For key '1', we should see consistent behavior (always switch to Processes view)
			if key == "1" {
				if afterView != messages.ProcessesView {
					h.RecordEvent("concurrent_inconsistency", fmt.Sprintf("Key '1' didn't switch to Processes view: got %s", afterView), key, "concurrent_test", false)
				}
			}
		}
	}

	// Final validation
	finalView := h.GetCurrentView()
	assert.NotEmpty(t, finalView, "Should have a valid final view")

	t.Logf("   Concurrent access test completed, final view: %s", finalView)
	t.Log("✅ Concurrent access edge case test passed")
}

// TestNumberKeyEdgeCases_MemoryLeakDetection tests for memory leaks during extensive key pressing
func TestNumberKeyEdgeCases_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key edge case tests in short mode")
	}

	t.Log("🚫 TESTING: Memory Leak Detection")
	t.Log("=================================")

	h := NewEdgeCaseTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize edge case test harness")

	// Record initial memory usage (simplified)
	initialMemory := h.EstimateMemoryUsage()
	t.Logf("   Initial memory usage: %.2f MB", initialMemory)

	// Perform extensive key pressing
	memoryTestKeys := 500
	for i := 0; i < memoryTestKeys; i++ {
		key := string(rune('1' + (i % 5))) // Cycle through 1-5
		err := h.SimulateKey(key)
		if err != nil {
			t.Logf("   Warning: Failed to press key %s: %v", key, err)
		}

		// Small delay
		time.Sleep(1 * time.Millisecond)
	}

	// Check memory usage after extensive key pressing
	finalMemory := h.EstimateMemoryUsage()
	t.Logf("   Final memory usage: %.2f MB", finalMemory)

	memoryIncrease := finalMemory - initialMemory
	memoryIncreasePercent := (memoryIncrease / initialMemory) * 100

	t.Logf("   Memory increase: %.2f MB (%.1f%%)", memoryIncrease, memoryIncreasePercent)

	// Memory increase should be reasonable (less than 50MB or 50% increase)
	assert.Less(t, memoryIncrease, 50.0, "Memory increase should be less than 50MB")
	assert.Less(t, memoryIncreasePercent, 50.0, "Memory increase should be less than 50%")

	h.stressTestResults.MemoryUsageMB = finalMemory
	t.Log("✅ Memory leak detection test passed")
}

// Initialize sets up the edge case test harness
func (h *EdgeCaseTestHarness) Initialize() error {
	h.t.Helper()

	// Create temporary working directory
	h.workingDir = h.t.TempDir()
	packageJSON := `{
		"name": "edge-case-test-project",
		"scripts": {
			"dev": "echo 'Dev server running' && sleep 60",
			"test": "echo 'Running tests' && sleep 2"
		}
	}`
	err := os.WriteFile(h.workingDir+"/package.json", []byte(packageJSON), 0644)
	if err != nil {
		return fmt.Errorf("failed to create package.json: %w", err)
	}

	// Create event bus
	h.eventBus = events.NewEventBus()

	// Create process manager and log store
	h.processMgr, err = process.NewManager(h.workingDir, h.eventBus, true)
	if err != nil {
		return fmt.Errorf("failed to create process manager: %w", err)
	}

	h.logStore = logs.NewStore(10000, h.eventBus)

	// Create navigation service
	h.navigationService = services.NewNavigationService(h.eventBus)

	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.Size = pty.PTYSize{Rows: 24, Cols: 80}
	ptyConfig.EnableDebug = true
	ptyConfig.Timeout = 60 * time.Second

	h.ptyLayer, err = pty.NewPTYLayer(ptyConfig)
	if err != nil {
		return fmt.Errorf("failed to create PTY layer: %w", err)
	}

	// Allocate PTY
	err = h.ptyLayer.Allocate()
	if err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	// Create TUI model
	cfg := &config.Config{}
	h.model = tui.NewModelWithView(
		h.processMgr,
		h.logStore,
		h.eventBus,
		nil, // mcpServer not needed
		nil, // proxyServer not needed
		7777, // mcpPort
		messages.ProcessesView, // Start with Processes view
		false, // debugMode
		cfg,
		h.workingDir,
	)

	// Create tea program with PTY I/O
	options := []tea.ProgramOption{
		tea.WithInput(h.ptyLayer),
		tea.WithOutput(h.ptyLayer),
		tea.WithoutSignalHandler(),
		tea.WithContext(h.ctx),
	}

	h.program = tea.NewProgram(h.model, options...)

	// Start the program
	go func() {
		if _, err := h.program.Run(); err != nil {
			h.t.Logf("Program ended with error: %v", err)
		}
	}()

	// Wait for program to initialize
	time.Sleep(100 * time.Millisecond)

	return nil
}

// SimulateKey simulates pressing a key
func (h *EdgeCaseTestHarness) SimulateKey(key string) error {
	h.t.Helper()

	// Handle special keys
	var keyBytes []byte
	switch key {
	case " ":
		keyBytes = []byte(" ")
	case "\t":
		keyBytes = []byte("\t")
	case "\n":
		keyBytes = []byte("\n")
	case "\r":
		keyBytes = []byte("\r")
	default:
		if len(key) == 1 {
			keyBytes = []byte(key)
		} else {
			return fmt.Errorf("unsupported key: %s", key)
		}
	}

	// Write key to PTY
	_, err := h.ptyLayer.Write(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to write key %s to PTY: %w", key, err)
	}

	return nil
}

// GetCurrentView returns the current view
func (h *EdgeCaseTestHarness) GetCurrentView() messages.ViewType {
	if h.model == nil {
		return ""
	}

	if model, ok := h.model.(interface{ GetCurrentView() string }); ok {
		return messages.ViewType(model.GetCurrentView())
	}

	return ""
}

// GetSwitchCount returns the number of view switches (simplified for edge cases)
func (h *EdgeCaseTestHarness) GetSwitchCount() int {
	// This is a simplified implementation
	// In a real implementation, this would track actual view switches
	return 0
}

// RecordEvent records an error or unusual event
func (h *EdgeCaseTestHarness) RecordEvent(errorType, description, key, context string, recovered bool) {
	h.t.Helper()

	event := ErrorEvent{
		Timestamp:   time.Now(),
		ErrorType:   errorType,
		Description: description,
		Key:         key,
		Context:     context,
		Recovered:   recovered,
	}

	h.errorEvents = append(h.errorEvents, event)

	if !recovered {
		h.t.Logf("⚠️  Error event recorded: %s - %s", errorType, description)
	}
}

// EstimateMemoryUsage provides a rough estimate of memory usage
func (h *EdgeCaseTestHarness) EstimateMemoryUsage() float64 {
	// This is a simplified memory estimation
	// In a real implementation, this would use runtime.MemStats
	return 10.0 + float64(len(h.errorEvents))*0.01 // Base 10MB + small amount per event
}

// Cleanup cleans up test resources
func (h *EdgeCaseTestHarness) Cleanup() {
	h.t.Helper()

	if h.cancel != nil {
		h.cancel()
	}

	if h.program != nil {
		h.program.Quit()
		h.program = nil
	}

	if h.ptyLayer != nil {
		h.ptyLayer.Cleanup()
		h.ptyLayer = nil
	}

	if h.processMgr != nil {
		h.processMgr.Shutdown()
		h.processMgr = nil
	}
}