//go:build tty && integration
// +build tty,integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/harness"
)

// TestNumberKeySimple_BasicMapping tests basic number key to view mapping
func TestNumberKeySimple_BasicMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tests in short mode")
	}

	t.Log("🧪 Testing Number Key Tab Switching - Basic Mapping")
	t.Log("================================================")

	// Create test harness configuration
	config := harness.DefaultTestConfig()
	config.TestTimeout = 10 * time.Second
	config.KeyInputDelay = 10 * time.Millisecond
	config.TerminalRows = 24
	config.TerminalCols = 80
	config.StateValidation = true

	// Create test harness
	testHarness := harness.NewTestHarness()
	defer func() {
		err := testHarness.Stop()
		if err != nil {
			t.Logf("Warning: Failed to stop test harness: %v", err)
		}
	}()

	// Initialize harness
	err := testHarness.Initialize(config)
	require.NoError(t, err, "Failed to initialize test harness")

	// Start harness
	err = testHarness.Start()
	require.NoError(t, err, "Failed to start test harness")

	t.Log("✅ Test harness initialized and started successfully")

	// Define expected number key mappings
	expectedMappings := map[int]string{
		1: "processes",
		2: "logs",
		3: "errors",
		4: "aicoders",
		5: "settings",
	}

	// Test each mapped number key
	for key, expectedView := range expectedMappings {
		t.Run("Key"+string(rune('0'+key)), func(t *testing.T) {
			t.Helper()

			t.Logf("Testing key %d -> %s", key, expectedView)

			// Create a test case for number key press
			testCase := harness.TestCase{
				ID:          fmt.Sprintf("number-key-%d-test", key),
				Name:        fmt.Sprintf("Press number key %d", key),
				Description: fmt.Sprintf("Test that pressing %d switches to %s view", key, expectedView),
				Actions: []harness.TestAction{
					{
						Type:       harness.ActionTypeKeyPress,
						Parameters: map[string]interface{}{
							"key": createKeyMsg(rune('0' + key)),
						},
						Description: fmt.Sprintf("Press number key %d", key),
					},
					{
						Type:       harness.ActionTypeWait,
						Parameters: map[string]interface{}{
							"duration": 100 * time.Millisecond,
						},
						Description: "Wait for view change",
					},
				},
				ExpectedStates: []harness.ExpectedState{
					{
						TriggerAction: fmt.Sprintf("key:%d", key),
						ViewType:      expectedView,
						Timeout:       1 * time.Second,
						ModelState: map[string]interface{}{
							"view_type": expectedView,
							"active":    true,
						},
					},
				},
				Timeout: 5 * time.Second,
			}

			// Run the test
			result := testHarness.RunTest(testCase)

			// Assert test passed
			assert.Equal(t, harness.TestStatusPass, result.Status,
				fmt.Sprintf("Number key %d test should pass", key))
			assert.Equal(t, testCase.ID, result.TestID, "Test ID should match")

			if result.Status != harness.TestStatusPass {
				t.Logf("❌ Test failed for key %d: %s", key, result.Error)
			} else {
				t.Logf("✅ Key %d test passed", key)
			}
		})
	}

	t.Log("🎯 Basic number key mapping tests completed")
}

// TestNumberKeySimple_InvalidKeys tests that invalid number keys are ignored
func TestNumberKeySimple_InvalidKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping number key tests in short mode")
	}

	t.Log("🧪 Testing Number Key Tab Switching - Invalid Keys")
	t.Log("===============================================")

	// Create test harness configuration
	config := harness.DefaultTestConfig()
	config.TestTimeout = 10 * time.Second
	config.KeyInputDelay = 10 * time.Millisecond

	// Create test harness
	testHarness := harness.NewTestHarness()
	defer func() {
		err := testHarness.Stop()
		if err != nil {
			t.Logf("Warning: Failed to stop test harness: %v", err)
		}
	}()

	// Initialize and start harness
	err := testHarness.Initialize(config)
	require.NoError(t, err)
	err = testHarness.Start()
	require.NoError(t, err)

	// Test invalid number keys (0, 6, 7, 8, 9)
	invalidKeys := []int{0, 6, 7, 8, 9}

	for _, key := range invalidKeys {
		t.Run("InvalidKey"+string(rune('0'+key)), func(t *testing.T) {
			t.Helper()

			t.Logf("Testing invalid key %d should be ignored", key)

			// Create test case for invalid number key
			testCase := harness.TestCase{
				ID:          fmt.Sprintf("invalid-key-%d-test", key),
				Name:        fmt.Sprintf("Press invalid number key %d", key),
				Description: fmt.Sprintf("Test that pressing %d is ignored", key),
				Actions: []harness.TestAction{
					{
						Type:       harness.ActionTypeKeyPress,
						Parameters: map[string]interface{}{
							"key": createKeyMsg(rune('0' + key)),
						},
						Description: fmt.Sprintf("Press invalid number key %d", key),
					},
					{
						Type:       harness.ActionTypeWait,
						Parameters: map[string]interface{}{
							"duration": 100 * time.Millisecond,
						},
						Description: "Wait to ensure no view change",
					},
				},
				ExpectedStates: []harness.ExpectedState{
					{
						TriggerAction: fmt.Sprintf("key:%d", key),
						ModelState: map[string]interface{}{
							"view_unchanged": true,
						},
						Timeout: 1 * time.Second,
					},
				},
				Timeout: 5 * time.Second,
			}

			// Run the test
			result := testHarness.RunTest(testCase)

			// For invalid keys, we expect the test to pass (key should be ignored without error)
			assert.Equal(t, harness.TestStatusPass, result.Status,
				fmt.Sprintf("Invalid key %d should be handled gracefully", key))

			t.Logf("✅ Invalid key %d handled correctly", key)
		})
	}

	t.Log("🎯 Invalid number key tests completed")
}

// createKeyMsg creates a tea.KeyMsg for a rune
func createKeyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{
		Type: tea.KeyRunes,
		Runes: []rune{r},
	}
}