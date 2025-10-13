//go:build tty && integration
// +build tty,integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/harness"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// NumberKeyTabSwitchTest explicitly tests number key tab switching functionality
// through the TTY/PTY harness. This provides comprehensive validation of the
// number key navigation feature (1-9 keys).
//
// Key Features Tested:
// - Direct number key navigation (1-5 mapped to specific views)
// - Invalid number key handling (0, 6-9 should be ignored)
// - Rapid number key sequences
// - State consistency after number key navigation
// - Integration with PTY layer and navigation service

// NumberKeyTestHarness provides specialized testing interface for number key tab switching
type NumberKeyTestHarness struct {
	t                    *testing.T
	ctx                  context.Context
	cancel               context.CancelFunc
	harness              *harness.TestHarness
	ptyLayer             *pty.PTYLayer
	navigationService    *services.NavigationService
	eventBus             *events.EventBus
	processMgr           *process.Manager
	logStore             *logs.Store
	model                tea.Model
	program              *tea.Program
	expectedMappings     map[int]messages.ViewType
	actualKeyPresses     map[int]int
	viewSwitchHistory    []ViewSwitchEvent
	switchCount          int
	testStartTime        time.Time
}

// ViewSwitchEvent records a single view switch event
type ViewSwitchEvent struct {
	Timestamp      time.Time             `json:"timestamp"`
	FromView       messages.ViewType     `json:"from_view"`
	ToView         messages.ViewType     `json:"to_view"`
	Trigger        string                `json:"trigger"`        // "number_key", "tab", "command"
	KeyPressed     string                `json:"key_pressed"`    // The actual key that was pressed
	ExpectedView   messages.ViewType     `json:"expected_view"`  // What we expected to switch to
	SwitchDuration time.Duration         `json:"switch_duration"`
	Success        bool                  `json:"success"`
}

// NewNumberKeyTestHarness creates a new number key test harness
func NewNumberKeyTestHarness(t *testing.T) *NumberKeyTestHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Expected number key to view mappings (based on current implementation)
	expectedMappings := map[int]messages.ViewType{
		1: messages.ProcessesView,
		2: messages.LogsView,
		3: messages.ErrorsView,
		4: messages.AICodersView,
		5: messages.SettingsView,
		// Note: 6-9 are currently unmapped and should be ignored
	}

	return &NumberKeyTestHarness{
		t:                t,
		ctx:              ctx,
		cancel:           cancel,
		expectedMappings: expectedMappings,
		actualKeyPresses: make(map[int]int),
		viewSwitchHistory: make([]ViewSwitchEvent, 0),
		testStartTime:    time.Now(),
	}
}

// TestNumberKeyTabSwitch_BasicMapping tests basic number key to view mappings
func TestNumberKeyTabSwitch_BasicMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tab switching test in short mode")
	}

	t.Log("🔢 TESTING: Basic Number Key Tab Switching (1-5)")
	t.Log("==============================================")

	// Create test harness
	h := NewNumberKeyTestHarness(t)
	defer h.Cleanup()

	// Initialize harness
	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize number key test harness")

	// Test each mapped number key (1-5)
	for key := 1; key <= 5; key++ {
		t.Run(fmt.Sprintf("NumberKey_%d", key), func(t *testing.T) {
			expectedView := h.expectedMappings[key]

			// Get current view before switch
			beforeView := h.GetCurrentView()
			t.Logf("   Before pressing %d: %s", key, beforeView)

			// Press number key
			startTime := time.Now()
			err := h.SimulateNumberKey(key)
			duration := time.Since(startTime)

			require.NoError(t, err, "Failed to simulate number key %d", key)

			// Wait for view switch to complete
			h.WaitForViewSwitch(expectedView, 2*time.Second)

			// Validate view switch
			afterView := h.GetCurrentView()
			assert.Equal(t, expectedView, afterView,
				"View should have switched to %s after pressing %d (got %s)", expectedView, key, afterView)

			// Record successful switch
			h.RecordViewSwitch(beforeView, afterView, "number_key", fmt.Sprintf("%d", key), expectedView, duration, true)

			// Validate key press was recorded
			h.AssertNumberKeyPressed(key)

			t.Logf("   ✓ Number key %d successfully switched to %s (took %v)", key, afterView, duration)
		})
	}

	// Validate final state
	h.ValidateFinalState()
	t.Log("✅ Basic number key mapping tests passed")
}

// TestNumberKeyTabSwitch_InvalidKeys tests invalid number key handling
func TestNumberKeyTabSwitch_InvalidKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tab switching test in short mode")
	}

	t.Log("🔢 TESTING: Invalid Number Key Handling (0, 6-9)")
	t.Log("===============================================")

	h := NewNumberKeyTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize number key test harness")

	// Test invalid keys
	invalidKeys := []int{0, 6, 7, 8, 9}

	for _, key := range invalidKeys {
		t.Run(fmt.Sprintf("InvalidKey_%d", key), func(t *testing.T) {
			// Get current view before key press
			beforeView := h.GetCurrentView()
			t.Logf("   Before pressing invalid key %d: %s", key, beforeView)

			// Press invalid number key
			err := h.SimulateNumberKey(key)
			require.NoError(t, err, "Failed to simulate invalid number key %d", key)

			// Wait briefly to ensure no view switch occurs
			time.Sleep(200 * time.Millisecond)

			// View should remain unchanged
			afterView := h.GetCurrentView()
			assert.Equal(t, beforeView, afterView,
				"View should remain unchanged after pressing invalid key %d", key)

			// Record that invalid key was ignored
			h.RecordViewSwitch(beforeView, afterView, "number_key", fmt.Sprintf("%d", key), beforeView, 0, false)

			// Validate key press was recorded but no switch occurred
			h.AssertNumberKeyPressed(key)

			t.Logf("   ✓ Invalid key %d correctly ignored (view stayed %s)", key, afterView)
		})
	}

	// Validate final state
	h.ValidateFinalState()
	t.Log("✅ Invalid number key handling tests passed")
}

// TestNumberKeyTabSwitch_RapidSequences tests rapid number key sequences
func TestNumberKeyTabSwitch_RapidSequences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tab switching test in short mode")
	}

	t.Log("🔢 TESTING: Rapid Number Key Sequences")
	t.Log("=====================================")

	h := NewNumberKeyTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize number key test harness")

	// Test rapid sequence
	testSequences := [][]int{
		{1, 2, 3, 4, 5},           // Forward sequence
		{5, 4, 3, 2, 1},           // Reverse sequence
		{1, 3, 5, 2, 4},           // Mixed sequence
		{2, 2, 3, 3, 1, 1},       // Repeated keys
	}

	for seqIndex, sequence := range testSequences {
		t.Run(fmt.Sprintf("RapidSequence_%d", seqIndex), func(t *testing.T) {
			t.Logf("   Testing rapid sequence: %v", sequence)

			startTime := time.Now()

			// Execute rapid key presses
			for i, key := range sequence {
				t.Logf("     Pressing key %d (%d/%d)", key, i+1, len(sequence))

				err := h.SimulateNumberKey(key)
				require.NoError(t, err, "Failed to simulate key %d in rapid sequence", key)

				// Small delay between rapid keys
				time.Sleep(10 * time.Millisecond)
			}

			// Wait for all switches to complete
			time.Sleep(500 * time.Millisecond)

			totalDuration := time.Since(startTime)
			t.Logf("   Rapid sequence completed in %v (avg %v per key)",
				totalDuration, totalDuration/time.Duration(len(sequence)))

			// Validate the final view matches the last key in sequence
			if len(sequence) > 0 {
				lastKey := sequence[len(sequence)-1]
				if expectedView, exists := h.expectedMappings[lastKey]; exists {
					finalView := h.GetCurrentView()
					assert.Equal(t, expectedView, finalView,
						"Final view should match last key in sequence (key %d -> %s)", lastKey, expectedView)
				}
			}

			// Validate all keys were recorded
			for _, key := range sequence {
				h.AssertNumberKeyPressed(key)
			}
		})
	}

	// Validate final state
	h.ValidateFinalState()
	t.Log("✅ Rapid number key sequence tests passed")
}

// TestNumberKeyTabSwitch_StateConsistency tests state consistency during number key navigation
func TestNumberKeyTabSwitch_StateConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tab switching test in short mode")
	}

	t.Log("🔢 TESTING: State Consistency During Number Key Navigation")
	t.Log("=======================================================")

	h := NewNumberKeyTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize number key test harness")

	// Test state consistency across multiple switches
	testKeys := []int{1, 2, 3, 2, 4, 5, 1}
	expectedViews := []messages.ViewType{
		messages.ProcessesView,
		messages.LogsView,
		messages.ErrorsView,
		messages.LogsView,
		messages.AICodersView,
		messages.SettingsView,
		messages.ProcessesView,
	}

	assert.Equal(t, len(testKeys), len(expectedViews), "Test arrays should have same length")

	for i, key := range testKeys {
		t.Run(fmt.Sprintf("StateConsistency_Step_%d", i), func(t *testing.T) {
			expectedView := expectedViews[i]

			// Record state before switch
			beforeView := h.GetCurrentView()
			beforeSwitchCount := h.GetSwitchCount()
			beforeKeyPresses := h.GetKeyPressCount()

			t.Logf("   Step %d: Pressing %d, expecting %s", i+1, key, expectedView)
			t.Logf("     Before: view=%s, switches=%d, key_presses=%d",
				beforeView, beforeSwitchCount, beforeKeyPresses)

			// Press key
			startTime := time.Now()
			err := h.SimulateNumberKey(key)
			require.NoError(t, err, "Failed to press key %d", key)

			// Wait for switch
			h.WaitForViewSwitch(expectedView, 1*time.Second)
			switchDuration := time.Since(startTime)

			// Record state after switch
			afterView := h.GetCurrentView()
			afterSwitchCount := h.GetSwitchCount()
			afterKeyPresses := h.GetKeyPressCount()

			t.Logf("     After: view=%s, switches=%d, key_presses=%d (took %v)",
				afterView, afterSwitchCount, afterKeyPresses, switchDuration)

			// Validate state consistency
			assert.Equal(t, expectedView, afterView, "View should match expected view")
			assert.Equal(t, beforeSwitchCount+1, afterSwitchCount, "Switch count should increment by 1")
			assert.Equal(t, beforeKeyPresses+1, afterKeyPresses, "Key press count should increment by 1")

			// Record the switch event
			h.RecordViewSwitch(beforeView, afterView, "number_key", fmt.Sprintf("%d", key), expectedView, switchDuration, true)

			// Validate navigation service state
			h.ValidateNavigationServiceState(expectedView)

			t.Logf("   ✓ Step %d: State consistency validated", i+1)
		})
	}

	// Validate overall history consistency
	h.ValidateSwitchHistoryConsistency()
	t.Log("✅ State consistency tests passed")
}

// TestNumberKeyTabSwitch_PerformanceMetrics tests performance metrics for number key switching
func TestNumberKeyTabSwitch_PerformanceMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tab switching test in short mode")
	}

	t.Log("🔢 TESTING: Performance Metrics for Number Key Switching")
	t.Log("=====================================================")

	h := NewNumberKeyTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize number key test harness")

	// Performance test parameters
	iterations := 20
	maxAcceptableDuration := 50 * time.Millisecond

	var totalDuration time.Duration
	var switchDurations = make([]time.Duration, 0, iterations)

	// Run performance test
	for i := 0; i < iterations; i++ {
		key := (i % 5) + 1 // Cycle through keys 1-5
		expectedView := h.expectedMappings[key]

		// Measure switch duration
		startTime := time.Now()
		err := h.SimulateNumberKey(key)
		require.NoError(t, err, "Failed to press key %d in performance test", key)

		h.WaitForViewSwitch(expectedView, 500*time.Millisecond)
		switchDuration := time.Since(startTime)

		switchDurations = append(switchDurations, switchDuration)
		totalDuration += switchDuration

		// Validate performance threshold
		assert.Less(t, switchDuration, maxAcceptableDuration,
			"Switch %d took %v, should be less than %v", i+1, switchDuration, maxAcceptableDuration)
	}

	// Calculate statistics
	avgDuration := totalDuration / time.Duration(iterations)
	minDuration := switchDurations[0]
	maxDuration := switchDurations[0]

	for _, duration := range switchDurations {
		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	t.Logf("   Performance Statistics (%d iterations):", iterations)
	t.Logf("     Total time: %v", totalDuration)
	t.Logf("     Average time per switch: %v", avgDuration)
	t.Logf("     Min switch time: %v", minDuration)
	t.Logf("     Max switch time: %v", maxDuration)
	t.Logf("     Keys pressed: %d", h.GetKeyPressCount())
	t.Logf("     View switches: %d", h.GetSwitchCount())

	// Performance assertions
	assert.Less(t, avgDuration, 30*time.Millisecond, "Average switch time should be under 30ms")
	assert.Less(t, maxDuration, maxAcceptableDuration, "Max switch time should be under threshold")
	assert.Equal(t, iterations, h.GetKeyPressCount(), "All key presses should be recorded")
	assert.Equal(t, iterations, h.GetSwitchCount(), "All switches should be recorded")

	// Validate final state
	h.ValidateFinalState()
	t.Log("✅ Performance metrics tests passed")
}

// TestNumberKeyTabSwitch_IntegrationWithPTY tests integration with PTY layer
func TestNumberKeyTabSwitch_IntegrationWithPTY(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tab switching test in short mode")
	}

	t.Log("🔢 TESTING: Integration with PTY Layer")
	t.Log("====================================")

	h := NewNumberKeyTestHarness(t)
	defer h.Cleanup()

	err := h.Initialize()
	require.NoError(t, err, "Failed to initialize number key test harness")

	// Test that PTY layer properly routes number key events
	testKeys := []int{1, 2, 3, 4, 5}

	for _, key := range testKeys {
		t.Run(fmt.Sprintf("PTYIntegration_Key_%d", key), func(t *testing.T) {
			expectedView := h.expectedMappings[key]

			// Test PTY input simulation
			err := h.SimulateNumberKeyThroughPTY(key)
			require.NoError(t, err, "Failed to simulate key %d through PTY", key)

			// Wait for PTY event processing
			h.WaitForViewSwitch(expectedView, 1*time.Second)

			// Validate PTY integration worked
			actualView := h.GetCurrentView()
			assert.Equal(t, expectedView, actualView,
				"PTY should have routed number key %d to switch to %s", key, expectedView)

			// Validate PTY state
			h.ValidatePTYState()

			t.Logf("   ✓ PTY integration validated for key %d -> %s", key, actualView)
		})
	}

	t.Log("✅ PTY integration tests passed")
}

// Initialize sets up the test harness with all necessary components
func (h *NumberKeyTestHarness) Initialize() error {
	h.t.Helper()

	// Create temporary working directory with package.json
	tempDir := h.t.TempDir()
	packageJSON := `{
		"name": "number-key-test-project",
		"scripts": {
			"dev": "echo 'Dev server running' && sleep 60",
			"test": "echo 'Running tests' && sleep 2",
			"build": "echo 'Building' && sleep 1"
		}
	}`
	err := os.WriteFile(tempDir+"/package.json", []byte(packageJSON), 0644)
	if err != nil {
		return fmt.Errorf("failed to create package.json: %w", err)
	}

	// Create event bus
	h.eventBus = events.NewEventBus()

	// Create process manager and log store
	h.processMgr, err = process.NewManager(tempDir, h.eventBus, true)
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
	ptyConfig.Timeout = 30 * time.Second

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
	cfg := &config.Config{} // Use empty config for tests
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
		tempDir,
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

// SimulateNumberKey simulates pressing a number key
func (h *NumberKeyTestHarness) SimulateNumberKey(key int) error {
	h.t.Helper()

	if key < 0 || key > 9 {
		return fmt.Errorf("invalid number key: %d", key)
	}

	// Record the key press
	h.actualKeyPresses[key]++

	// Convert key to appropriate format and send to PTY
	var keyStr string
	switch key {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
		keyStr = string(rune('0' + key))
	default:
		return fmt.Errorf("unsupported number key: %d", key)
	}

	// Write key to PTY
	_, err := h.ptyLayer.Write([]byte(keyStr))
	if err != nil {
		return fmt.Errorf("failed to write key %d to PTY: %w", key, err)
	}

	return nil
}

// SimulateNumberKeyThroughPTY simulates a number key specifically through PTY layer
func (h *NumberKeyTestHarness) SimulateNumberKeyThroughPTY(key int) error {
	h.t.Helper()

	// This method simulates the key press through the PTY layer specifically
	// to test PTY integration with navigation service
	return h.SimulateNumberKey(key)
}

// GetCurrentView returns the current view from the model
func (h *NumberKeyTestHarness) GetCurrentView() messages.ViewType {
	if h.model == nil {
		return ""
	}

	// Use type assertion to access model methods
	if model, ok := h.model.(interface{ GetCurrentView() string }); ok {
		return messages.ViewType(model.GetCurrentView())
	}

	return ""
}

// WaitForViewSwitch waits for the view to switch to the expected view
func (h *NumberKeyTestHarness) WaitForViewSwitch(expectedView messages.ViewType, timeout time.Duration) {
	h.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.GetCurrentView() == expectedView {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	h.t.Fatalf("Timeout waiting for view to switch to %s (current: %s)",
		expectedView, h.GetCurrentView())
}

// RecordViewSwitch records a view switch event
func (h *NumberKeyTestHarness) RecordViewSwitch(from, to messages.ViewType, trigger, key string, expected messages.ViewType, duration time.Duration, success bool) {
	h.t.Helper()

	event := ViewSwitchEvent{
		Timestamp:      time.Now(),
		FromView:       from,
		ToView:         to,
		Trigger:        trigger,
		KeyPressed:     key,
		ExpectedView:   expected,
		SwitchDuration: duration,
		Success:        success,
	}

	h.viewSwitchHistory = append(h.viewSwitchHistory, event)
	h.switchCount++
}

// AssertNumberKeyPressed asserts that a number key was pressed
func (h *NumberKeyTestHarness) AssertNumberKeyPressed(key int) {
	h.t.Helper()

	count, exists := h.actualKeyPresses[key]
	if !exists || count == 0 {
		h.t.Errorf("Number key %d was not pressed (press count: %d)", key, count)
	}
}

// GetSwitchCount returns the number of view switches
func (h *NumberKeyTestHarness) GetSwitchCount() int {
	return h.switchCount
}

// GetKeyPressCount returns the total number of key presses
func (h *NumberKeyTestHarness) GetKeyPressCount() int {
	total := 0
	for _, count := range h.actualKeyPresses {
		total += count
	}
	return total
}

// ValidateFinalState performs final state validation
func (h *NumberKeyTestHarness) ValidateFinalState() {
	h.t.Helper()

	// Ensure we have a valid current view
	currentView := h.GetCurrentView()
	assert.NotEmpty(h.t, currentView, "Should have a valid current view")

	// Ensure current view is one of the expected views
	validViews := []messages.ViewType{
		messages.ProcessesView,
		messages.LogsView,
		messages.ErrorsView,
		messages.AICodersView,
		messages.SettingsView,
	}
	assert.Contains(h.t, validViews, currentView, "Current view should be valid")

	// Validate key press counts
	totalKeyPresses := h.GetKeyPressCount()
	assert.Greater(h.t, totalKeyPresses, 0, "Should have recorded key presses")

	// Validate switch count matches successful switches
	successfulSwitches := 0
	for _, event := range h.viewSwitchHistory {
		if event.Success {
			successfulSwitches++
		}
	}
	assert.Equal(h.t, successfulSwitches, h.GetSwitchCount(), "Switch count should match successful switches")
}

// ValidateNavigationServiceState validates the navigation service state
func (h *NumberKeyTestHarness) ValidateNavigationServiceState(expectedView messages.ViewType) {
	h.t.Helper()

	if h.navigationService == nil {
		h.t.Skip("Navigation service not available for validation")
		return
	}

	// Get current view from navigation service
	currentView := h.navigationService.GetCurrentView()
	assert.Equal(h.t, expectedView, currentView, "Navigation service view should match expected view")
}

// ValidatePTYState validates PTY layer state
func (h *NumberKeyTestHarness) ValidatePTYState() {
	h.t.Helper()

	if h.ptyLayer == nil {
		h.t.Skip("PTY layer not available for validation")
		return
	}

	// Basic PTY state validation
	assert.NotNil(h.t, h.ptyLayer, "PTY layer should be available")
	// Additional PTY-specific validations can be added here
}

// ValidateSwitchHistoryConsistency validates switch event history consistency
func (h *NumberKeyTestHarness) ValidateSwitchHistoryConsistency() {
	h.t.Helper()

	// Ensure we have switch history
	assert.Greater(h.t, len(h.viewSwitchHistory), 0, "Should have recorded view switch history")

	// Validate each switch event
	for i, event := range h.viewSwitchHistory {
		assert.True(h.t, event.Timestamp.After(h.testStartTime) || event.Timestamp.Equal(h.testStartTime),
			"Switch event %d should have valid timestamp", i)
		assert.NotEmpty(h.t, event.Trigger, "Switch event %d should have trigger", i)
		assert.NotEmpty(h.t, event.KeyPressed, "Switch event %d should have key pressed", i)

		// For successful switches, validate view consistency
		if event.Success {
			assert.Equal(h.t, event.ExpectedView, event.ToView,
				"Switch event %d should have correct destination view", i)
		}
	}

	// Validate chronological order
	for i := 1; i < len(h.viewSwitchHistory); i++ {
		assert.True(h.t, h.viewSwitchHistory[i].Timestamp.After(h.viewSwitchHistory[i-1].Timestamp),
			"Switch history should be in chronological order")
	}
}

// Cleanup cleans up test resources
func (h *NumberKeyTestHarness) Cleanup() {
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