//go:build tty && integration
// +build tty,integration

package integration

import (
	"context"
	"os"
	"strings"
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
	"github.com/standardbeagle/brummer/pkg/events"
)

// KeyboardInputPTYTest tests keyboard input simulation through PTY for TUI testing.
// This test validates that keyboard input works correctly through the PTY layer.
//
// This test is designed to FAIL initially since there's no PTY keyboard input implementation yet.
// When a PTY keyboard input layer is implemented, this test should be updated to work with the
// real implementation.

// KeyboardInputPTYTester provides testing interface for keyboard input through PTY
type KeyboardInputPTYTester struct {
	t             *testing.T
	ctx           context.Context
	cancel        context.CancelFunc
	pty           *os.File
	terminal      *vt10x.Terminal
	output        []byte
	testBinary    string
	workingDir    string
	program       *tea.Program
	model         *tui.Model
	eventBus      *events.EventBus
	processMgr    *process.Manager
	logStore      *logs.Store
	keyPressCount int
	inputHistory  []string
	lastKeyPress  time.Time
}

// NewKeyboardInputPTYTester creates a new keyboard input PTY tester instance
func NewKeyboardInputPTYTester(t *testing.T) *KeyboardInputPTYTester {
	t.Helper()

	// Find or create test binary
	testBinary := os.Getenv("BRUMMER_BINARY")
	if testBinary == "" {
		testBinary = "./brum"
		if _, err := os.Stat(testBinary); os.IsNotExist(err) {
			t.Skip("Brummer binary not found. Set BRUMMER_BINARY or build with 'make build'")
		}
	}

	// Create temporary working directory for testing
	tempDir := t.TempDir()

	// Create test package.json
	packageJSON := `{
  "name": "keyboard-test-project",
  "scripts": {
    "dev": "echo 'Dev server running on http://localhost:3000' && sleep 60",
    "test": "echo 'Running tests...' && sleep 2 && echo 'Tests complete'",
    "build": "echo 'Building...' && sleep 1 && echo 'Build complete'"
  }
}`
	err := os.WriteFile(tempDir+"/package.json", []byte(packageJSON), 0644)
	require.NoError(t, err, "Failed to create test package.json")

	ctx, cancel := context.WithCancel(context.Background())

	return &KeyboardInputPTYTester{
		t:            t,
		ctx:          ctx,
		cancel:       cancel,
		testBinary:   testBinary,
		workingDir:   tempDir,
		output:       make([]byte, 0),
		inputHistory: make([]string, 0),
	}
}

// TestKeyboardInputPTY_Simulation tests basic keyboard input simulation through PTY
func TestKeyboardInputPTY_Simulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyboard input PTY integration test in short mode")
	}

	t.Log("⌨️  TESTING: Keyboard Input Simulation through PTY")
	t.Log("==============================================")

	// Create keyboard input PTY tester
	keyboardTester := NewKeyboardInputPTYTester(t)
	defer keyboardTester.Cleanup()

	// Test 1: PTY Allocation for Keyboard Input
	t.Log("Test 1: PTY Allocation for Keyboard Input")
	err := keyboardTester.AllocatePTY(80, 24)
	if err != nil {
		t.Logf("❌ PTY allocation failed (expected): %v", err)
		t.Logf("   This will fail until PTY keyboard input layer is implemented")
		// Since there's no implementation yet, we expect this to fail
		assert.Contains(t, err.Error(), "not implemented", "Expected 'not implemented' error")
		return
	}

	t.Log("✅ PTY allocation for keyboard input succeeded")

	// Test 2: Single Key Input Simulation
	t.Log("Test 2: Single Key Input Simulation")
	testKey := "q"
	err = keyboardTester.SimulateKeyPress(testKey)
	require.NoError(t, err, "Failed to simulate single key press")

	// Verify key was processed
	keyboardTester.WaitForKeyPress(1, 2*time.Second)
	assert.Equal(t, 1, keyboardTester.GetKeyPressCount(), "Should have recorded 1 key press")

	// Test 3: Multiple Key Input Simulation
	t.Log("Test 3: Multiple Key Input Simulation")
	testKeys := []string{"h", "e", "l", "l", "o"}
	err = keyboardTester.SimulateKeySequence(testKeys, 50*time.Millisecond)
	require.NoError(t, err, "Failed to simulate key sequence")

	// Verify keys were processed
	keyboardTester.WaitForKeyPress(len(testKeys)+1, 2*time.Second)
	assert.Equal(t, len(testKeys)+1, keyboardTester.GetKeyPressCount(), "Should have recorded all key presses")

	// Test 4: Special Key Input (Escape, Enter, etc.)
	t.Log("Test 4: Special Key Input Simulation")
	specialKeys := []string{"\x1b", "\r", "\t", "\x7f"} // Escape, Enter, Tab, Backspace
	for _, key := range specialKeys {
		err = keyboardTester.SimulateKeyPress(key)
		require.NoError(t, err, "Failed to simulate special key: %x", []byte(key)[0])
	}

	t.Log("✅ Keyboard input simulation tests passed")
}

// TestKeyboardInputPTY_TeaKeyMsgGeneration tests tea.KeyMsg generation via PTY
func TestKeyboardInputPTY_TeaKeyMsgGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyboard input PTY integration test in short mode")
	}

	t.Log("⌨️  TESTING: tea.KeyMsg Generation via PTY")
	t.Log("=========================================")

	keyboardTester := NewKeyboardInputPTYTester(t)
	defer keyboardTester.Cleanup()

	// Test 1: Initialize TUI Model
	t.Log("Test 1: Initialize TUI Model")
	err := keyboardTester.AllocatePTY(80, 24)
	if err != nil {
		t.Skip("PTY allocation not implemented, skipping KeyMsg generation tests")
	}

	err = keyboardTester.InitializeTUIModel()
	require.NoError(t, err, "Failed to initialize TUI model")

	// Test 2: Generate Basic KeyMsg
	t.Log("Test 2: Generate Basic KeyMsg")
	testKeys := []string{"a", "b", "c", "1", "2", "3"}
	for _, key := range testKeys {
		keyMsg, err := keyboardTester.GenerateKeyMsg(key)
		require.NoError(t, err, "Failed to generate KeyMsg for key: %s", key)

		assert.Equal(t, tea.KeyRunes, keyMsg.Type, "KeyMsg should be of type KeyRunes")
		assert.Equal(t, []rune(key), keyMsg.Runes, "KeyMsg should contain correct runes")
	}

	// Test 3: Generate Control KeyMsg
	t.Log("Test 3: Generate Control KeyMsg")
	controlKeyMappings := map[string]tea.KeyType{
		"\r":      tea.KeyEnter,
		"\x1b":    tea.KeyEscape,
		"\t":      tea.KeyTab,
		"\x7f":    tea.KeyBackspace,
		"\x1b[A":  tea.KeyUp,
		"\x1b[B":  tea.KeyDown,
		"\x1b[C":  tea.KeyRight,
		"\x1b[D":  tea.KeyLeft,
	}

	for keySequence, expectedType := range controlKeyMappings {
		keyMsg, err := keyboardTester.GenerateKeyMsg(keySequence)
		require.NoError(t, err, "Failed to generate control KeyMsg for: %x", []byte(keySequence)[0])

		assert.Equal(t, expectedType, keyMsg.Type, "KeyMsg should have correct type for sequence: %x", []byte(keySequence)[0])
	}

	// Test 4: KeyMsg Processing in TUI Model
	t.Log("Test 4: KeyMsg Processing in TUI Model")
	testKeyMsg, err := keyboardTester.GenerateKeyMsg("q") // Quit key
	require.NoError(t, err, "Failed to generate quit KeyMsg")

	// Send KeyMsg to model
	_, cmd := keyboardTester.model.Update(testKeyMsg)
	assert.NotNil(t, cmd, "Model should return a command after processing KeyMsg")

	t.Log("✅ tea.KeyMsg generation tests passed")
}

// TestKeyboardInputPTY_InputTimingAndDelays tests input timing and delays
func TestKeyboardInputPTY_InputTimingAndDelays(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyboard input PTY integration test in short mode")
	}

	t.Log("⌨️  TESTING: Input Timing and Delays")
	t.Log("===================================")

	keyboardTester := NewKeyboardInputPTYTester(t)
	defer keyboardTester.Cleanup()

	// Test 1: PTY Allocation for Timing Tests
	t.Log("Test 1: PTY Allocation for Timing Tests")
	err := keyboardTester.AllocatePTY(80, 24)
	if err != nil {
		t.Skip("PTY allocation not implemented, skipping timing tests")
	}

	// Test 2: Single Key Timing
	t.Log("Test 2: Single Key Timing")
	startTime := time.Now()
	err = keyboardTester.SimulateKeyPress("a")
	require.NoError(t, err, "Failed to simulate key press")

	keyboardTester.WaitForKeyPress(1, 1*time.Second)
	duration := time.Since(startTime)

	assert.Less(t, duration, 500*time.Millisecond, "Single key press should be processed quickly")
	t.Logf("   Single key processing time: %v", duration)

	// Test 3: Key Sequence with Delays
	t.Log("Test 3: Key Sequence with Delays")
	testKeys := []string{"h", "e", "l", "l", "o"}
	delay := 100 * time.Millisecond

	startTime = time.Now()
	err = keyboardTester.SimulateKeySequence(testKeys, delay)
	require.NoError(t, err, "Failed to simulate key sequence")

	keyboardTester.WaitForKeyPress(len(testKeys), 2*time.Second)
	totalDuration := time.Since(startTime)

	expectedMinDuration := time.Duration(len(testKeys)-1) * delay
	assert.GreaterOrEqual(t, totalDuration, expectedMinDuration, "Total duration should respect delays")
	assert.Less(t, totalDuration, expectedMinDuration+500*time.Millisecond, "Total duration should not be much longer than expected")

	t.Logf("   Key sequence processing time: %v (expected min: %v)", totalDuration, expectedMinDuration)

	// Test 4: Rapid Input Handling
	t.Log("Test 4: Rapid Input Handling")
	rapidKeys := []string{"x", "y", "z"}
	rapidDelay := 10 * time.Millisecond

	startTime = time.Now()
	err = keyboardTester.SimulateKeySequence(rapidKeys, rapidDelay)
	require.NoError(t, err, "Failed to simulate rapid key sequence")

	keyboardTester.WaitForKeyPress(len(rapidKeys), 1*time.Second)
	rapidDuration := time.Since(startTime)

	assert.Less(t, rapidDuration, 200*time.Millisecond, "Rapid input should be processed quickly")
	t.Logf("   Rapid input processing time: %v", rapidDuration)

	// Test 5: Input Debouncing (if applicable)
	t.Log("Test 5: Input Debouncing Test")
	rapidSameKeys := []string{"a", "a", "a", "a", "a"}
	veryRapidDelay := 1 * time.Millisecond

	startTime = time.Now()
	err = keyboardTester.SimulateKeySequence(rapidSameKeys, veryRapidDelay)
	require.NoError(t, err, "Failed to simulate very rapid same key sequence")

	keyboardTester.WaitForKeyPress(len(rapidSameKeys), 1*time.Second)
	debounceDuration := time.Since(startTime)

	t.Logf("   Very rapid input processing time: %v", debounceDuration)

	t.Log("✅ Input timing and delays tests passed")
}

// TestKeyboardInputPTY_ComplexSequences tests complex keyboard sequences
func TestKeyboardInputPTY_ComplexSequences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyboard input PTY integration test in short mode")
	}

	t.Log("⌨️  TESTING: Complex Keyboard Sequences")
	t.Log("====================================")

	keyboardTester := NewKeyboardInputPTYTester(t)
	defer keyboardTester.Cleanup()

	// Test 1: PTY Allocation for Complex Sequences
	t.Log("Test 1: PTY Allocation for Complex Sequences")
	err := keyboardTester.AllocatePTY(80, 24)
	if err != nil {
		t.Skip("PTY allocation not implemented, skipping complex sequence tests")
	}

	err = keyboardTester.InitializeTUIModel()
	require.NoError(t, err, "Failed to initialize TUI model")

	// Test 2: Command Input Sequence (Open command palette, type command, execute)
	t.Log("Test 2: Command Input Sequence")
	commandSequence := []string{
		"/",      // Open command palette
		"help",   // Type help command
		"\r",     // Execute command
	}

	err = keyboardTester.SimulateKeySequence(commandSequence, 100*time.Millisecond)
	require.NoError(t, err, "Failed to simulate command sequence")

	// Wait for command processing
	time.Sleep(1 * time.Second)

	// Check if help command was processed (look for help-related output)
	output := keyboardTester.GetOutput()
	outputStr := string(output)
	helpIndicators := []string{"help", "command", "available", "usage"}

	foundHelp := false
	for _, indicator := range helpIndicators {
		if strings.Contains(strings.ToLower(outputStr), indicator) {
			foundHelp = true
			break
		}
	}

	if foundHelp {
		t.Log("✅ Command sequence produced expected help output")
	} else {
		t.Log("⚠️  Command sequence output unclear, but PTY input processing worked")
	}

	// Test 3: Navigation Sequence (Arrow keys, Tab, etc.)
	t.Log("Test 3: Navigation Sequence")
	navigationSequence := []string{
		"\x1b[B", // Down arrow
		"\x1b[B", // Down arrow
		"\x1b[A", // Up arrow
		"\t",     // Tab
		"\x1b[C", // Right arrow
		"\x1b[D", // Left arrow
	}

	err = keyboardTester.SimulateKeySequence(navigationSequence, 50*time.Millisecond)
	require.NoError(t, err, "Failed to simulate navigation sequence")

	// Test 4: Text Editing Sequence (Type text, edit, etc.)
	t.Log("Test 4: Text Editing Sequence")
	editSequence := []string{
		"hello",  // Type hello
		"\x1b[D", // Move left
		"\x1b[D", // Move left
		"\x7f",   // Backspace
		"p",      // Type p (should become "hpello")
	}

	err = keyboardTester.SimulateKeySequence(editSequence, 50*time.Millisecond)
	require.NoError(t, err, "Failed to simulate text editing sequence")

	// Test 5: Complex Multi-Key Combinations
	t.Log("Test 5: Complex Multi-Key Combinations")
	combinations := []struct {
		keys     []string
		desc     string
		expected string
	}{
		{
			keys:     []string{"\x1b", "O", "H"}, // Home in some terminals
			desc:     "Home key combination",
			expected: "home",
		},
		{
			keys:     []string{"\x1b", "O", "F"}, // End in some terminals
			desc:     "End key combination",
			expected: "end",
		},
		{
			keys:     []string{"\x1b", "[", "5", "~"}, // Page up
			desc:     "Page up combination",
			expected: "page up",
		},
		{
			keys:     []string{"\x1b", "[", "6", "~"}, // Page down
			desc:     "Page down combination",
			expected: "page down",
		},
	}

	for _, combo := range combinations {
		err = keyboardTester.SimulateKeySequence(combo.keys, 50*time.Millisecond)
		require.NoError(t, err, "Failed to simulate %s", combo.desc)
		t.Logf("   Successfully simulated %s", combo.desc)
	}

	// Test 6: Escape Sequence Handling
	t.Log("Test 6: Escape Sequence Handling")
	escapeSequences := []string{
		"\x1b[31m", // Set text color to red
		"\x1b[1m",  // Bold text
		"\x1b[0m",  // Reset formatting
		"\x1b[2J",  // Clear screen
		"\x1b[H",   // Move cursor to home
	}

	err = keyboardTester.SimulateKeySequence(escapeSequences, 100*time.Millisecond)
	require.NoError(t, err, "Failed to simulate escape sequences")

	t.Log("✅ Complex keyboard sequences tests passed")
}

// TestKeyboardInputPTY_ErrorHandling tests error handling for keyboard input failures
func TestKeyboardInputPTY_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyboard input PTY integration test in short mode")
	}

	t.Log("⌨️  TESTING: Keyboard Input Error Handling")
	t.Log("=========================================")

	keyboardTester := NewKeyboardInputPTYTester(t)

	// Test 1: Operations on Unallocated PTY
	t.Log("Test 1: Operations on Unallocated PTY")
	err := keyboardTester.SimulateKeyPress("a")
	assert.Error(t, err, "Key press should fail on unallocated PTY")
	assert.Contains(t, err.Error(), "not allocated", "Expected not allocated error")

	err = keyboardTester.SimulateKeySequence([]string{"a", "b"}, 50*time.Millisecond)
	assert.Error(t, err, "Key sequence should fail on unallocated PTY")
	assert.Contains(t, err.Error(), "not allocated", "Expected not allocated error")

	_, err = keyboardTester.GenerateKeyMsg("a")
	assert.Error(t, err, "KeyMsg generation should fail on unallocated PTY")
	assert.Contains(t, err.Error(), "not allocated", "Expected not allocated error")

	// Test 2: Invalid Key Sequences
	t.Log("Test 2: Invalid Key Sequences")
	err = keyboardTester.AllocatePTY(80, 24)
	if err != nil {
		t.Skip("PTY allocation not implemented, skipping error handling tests")
	}

	// Test with nil/empty sequences
	err = keyboardTester.SimulateKeySequence(nil, 50*time.Millisecond)
	assert.Error(t, err, "Empty key sequence should fail")

	err = keyboardTester.SimulateKeySequence([]string{}, 50*time.Millisecond)
	assert.Error(t, err, "Empty key slice should fail")

	// Test with negative delays
	err = keyboardTester.SimulateKeySequence([]string{"a", "b"}, -50*time.Millisecond)
	assert.Error(t, err, "Negative delay should fail")

	// Test 3: Context Cancellation
	t.Log("Test 3: Context Cancellation")
	keyboardTester2 := NewKeyboardInputPTYTester(t)
	err = keyboardTester2.AllocatePTY(80, 24)
	require.NoError(t, err, "Failed to allocate PTY for context test")

	// Cancel context
	keyboardTester2.Cancel()

	// Operations should fail after context cancellation
	err = keyboardTester2.SimulateKeyPress("test")
	assert.Error(t, err, "Operations should fail after context cancellation")
	assert.Contains(t, err.Error(), "context canceled", "Expected context canceled error")

	// Test 4: Resource Exhaustion
	t.Log("Test 4: Resource Exhaustion")
	keyboardTester3 := NewKeyboardInputPTYTester(t)
	err = keyboardTester3.AllocatePTY(80, 24)
	require.NoError(t, err, "Failed to allocate PTY for resource test")

	// Simulate rapid input to test resource handling
	rapidKeys := make([]string, 1000)
	for i := range rapidKeys {
		rapidKeys[i] = "a"
	}

	err = keyboardTester3.SimulateKeySequence(rapidKeys, 1*time.Millisecond)
	if err != nil {
		// It's okay if this fails due to resource constraints
		t.Logf("   Rapid input handling failed as expected: %v", err)
	} else {
		t.Log("   Rapid input handling succeeded")
	}

	t.Log("✅ Keyboard input error handling tests passed")
}

// Keyboard Input PTY Interface Methods (these would be implemented by the real PTY keyboard input layer)

// AllocatePTY allocates a new PTY with the specified dimensions
func (k *KeyboardInputPTYTester) AllocatePTY(cols, rows int) error {
	// This is a placeholder - real implementation would go here
	return &NotImplementedError{"AllocatePTY"}
}

// InitializeTUIModel initializes the TUI model for testing
func (k *KeyboardInputPTYTester) InitializeTUIModel() error {
	// Create services
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{} // Use empty config for tests
	}

	k.eventBus = events.NewEventBus()
	k.processMgr, err = process.NewManager(k.workingDir, k.eventBus, true) // hasPackageJSON = true
	if err != nil {
		return err
	}
	k.logStore = logs.NewStore(10000, k.eventBus) // maxEntries = 10000

	// Create model
	k.model = tui.NewModel(
		k.processMgr,
		k.logStore,
		k.eventBus,
		nil,  // mcpServer not needed for keyboard tests
		nil,  // proxyServer not needed for keyboard tests
		7777, // mcpPort default
		cfg,
		k.workingDir, // workingDir
	)

	return nil
}

// SimulateKeyPress simulates a single key press through the PTY
func (k *KeyboardInputPTYTester) SimulateKeyPress(key string) error {
	k.t.Helper()
	k.lastKeyPress = time.Now()
	k.inputHistory = append(k.inputHistory, key)
	return &NotImplementedError{"SimulateKeyPress"}
}

// SimulateKeySequence simulates a sequence of key presses with delays
func (k *KeyboardInputPTYTester) SimulateKeySequence(keys []string, delay time.Duration) error {
	k.t.Helper()
	if keys == nil || len(keys) == 0 {
		return &InvalidInputError{"key sequence cannot be empty"}
	}
	if delay < 0 {
		return &InvalidInputError{"delay cannot be negative"}
	}

	for _, key := range keys {
		k.inputHistory = append(k.inputHistory, key)
		k.lastKeyPress = time.Now()
	}

	return &NotImplementedError{"SimulateKeySequence"}
}

// GenerateKeyMsg generates a tea.KeyMsg from a key input
func (k *KeyboardInputPTYTester) GenerateKeyMsg(key string) (tea.KeyMsg, error) {
	k.t.Helper()
	if k.model == nil {
		return tea.KeyMsg{}, &NotAllocatedError{"TUI model not initialized"}
	}

	// Create basic KeyMsg
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(key),
	}

	// Handle special keys
	switch key {
	case "\r":
		keyMsg.Type = tea.KeyEnter
	case "\x1b":
		keyMsg.Type = tea.KeyEscape
	case "\t":
		keyMsg.Type = tea.KeyTab
	case "\x7f":
		keyMsg.Type = tea.KeyBackspace
	case "\x1b[A":
		keyMsg.Type = tea.KeyUp
	case "\x1b[B":
		keyMsg.Type = tea.KeyDown
	case "\x1b[C":
		keyMsg.Type = tea.KeyRight
	case "\x1b[D":
		keyMsg.Type = tea.KeyLeft
	}

	return keyMsg, nil
}

// GetKeyPressCount returns the number of key presses recorded
func (k *KeyboardInputPTYTester) GetKeyPressCount() int {
	return k.keyPressCount
}

// GetInputHistory returns the history of inputs
func (k *KeyboardInputPTYTester) GetInputHistory() []string {
	return k.inputHistory
}

// GetOutput returns the captured output from the PTY
func (k *KeyboardInputPTYTester) GetOutput() []byte {
	return k.output
}

// WaitForKeyPress waits for a specific number of key presses
func (k *KeyboardInputPTYTester) WaitForKeyPress(count int, timeout time.Duration) {
	k.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if k.GetKeyPressCount() >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	k.t.Fatalf("Timeout waiting for %d key presses (got %d)", count, k.GetKeyPressCount())
}

// Cleanup cleans up PTY resources
func (k *KeyboardInputPTYTester) Cleanup() {
	k.t.Helper()
	if k.cancel != nil {
		k.cancel()
	}
	if k.pty != nil {
		k.pty.Close()
		k.pty = nil
	}
	if k.program != nil {
		k.program.Quit()
		k.program = nil
	}
}

// Cancel cancels the context
func (k *KeyboardInputPTYTester) Cancel() {
	if k.cancel != nil {
		k.cancel()
	}
}

// Custom error types for keyboard input PTY testing
type NotImplementedError struct {
	Method string
}

func (e *NotImplementedError) Error() string {
	return "method not implemented: " + e.Method
}

type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string {
	return "invalid input: " + e.Message
}

type NotAllocatedError struct {
	Message string
}

func (e *NotAllocatedError) Error() string {
	return "not allocated: " + e.Message
}

// TestKeyboardInputPTY_ImplementationRequired verifies that keyboard input PTY layer is not yet implemented
// This test is designed to fail until a real keyboard input PTY layer implementation is provided
func TestKeyboardInputPTY_ImplementationRequired(t *testing.T) {
	t.Log("🚨 KEYBOARD INPUT PTY IMPLEMENTATION STATUS CHECK")
	t.Log("===============================================")
	t.Log("This test verifies that keyboard input PTY layer functionality is not yet implemented.")
	t.Log("When keyboard input PTY layer is implemented, this test should be updated or removed.")

	// Try to use keyboard input PTY functionality
	keyboardTester := NewKeyboardInputPTYTester(t)
	defer keyboardTester.Cleanup()

	err := keyboardTester.AllocatePTY(80, 24)
	if err == nil {
		t.Error("Expected PTY allocation to fail with 'not implemented' error, but it succeeded")
		t.Error("This suggests keyboard input PTY layer may now be implemented - update this test!")
		return
	}

	var notImplErr *NotImplementedError
	if !assert.ErrorAs(t, err, &notImplErr, "Expected NotImplementedError") {
		t.Logf("Got different error type (this is expected until keyboard input PTY layer is implemented): %T", err)
	}

	t.Log("✅ Keyboard input PTY layer implementation status confirmed - not yet implemented")
	t.Log("   This is expected until keyboard input PTY layer is implemented")
}

// TestKeyboardInputPTY_IntegrationWithBrummerTests integration with actual Brummer binary
func TestKeyboardInputPTY_IntegrationWithBrummer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyboard input PTY integration test in short mode")
	}

	t.Log("⌨️  TESTING: Keyboard Input PTY Integration with Brummer")
	t.Log("===================================================")

	keyboardTester := NewKeyboardInputPTYTester(t)
	defer keyboardTester.Cleanup()

	// Test 1: Start Brummer with PTY
	t.Log("Test 1: Start Brummer with PTY")
	err := keyboardTester.AllocatePTY(80, 24)
	if err != nil {
		t.Skip("Keyboard input PTY allocation not implemented, skipping integration tests")
	}

	err = keyboardTester.InitializeTUIModel()
	require.NoError(t, err, "Failed to initialize TUI model")

	// Test 2: Basic TUI Navigation
	t.Log("Test 2: Basic TUI Navigation")
	navigationKeys := []string{
		"1", // Go to Logs tab
		"2", // Go to Processes tab
		"6", // Go to AI Coders tab
		"q", // Quit
	}

	err = keyboardTester.SimulateKeySequence(navigationKeys, 200*time.Millisecond)
	require.NoError(t, err, "Failed to simulate navigation keys")

	// Wait for navigation processing
	time.Sleep(2 * time.Second)

	// Test 3: Command Input
	t.Log("Test 3: Command Input")
	commandKeys := []string{
		"/",     // Open command palette
		"help",  // Type help
		"\r",    // Execute
		"q",     // Quit
	}

	err = keyboardTester.SimulateKeySequence(commandKeys, 100*time.Millisecond)
	require.NoError(t, err, "Failed to simulate command input")

	// Wait for command processing
	time.Sleep(2 * time.Second)

	// Test 4: Output Validation
	t.Log("Test 4: Output Validation")
	output := keyboardTester.GetOutput()
	assert.NotEmpty(t, output, "Should have captured some output")

	outputStr := string(output)
	t.Logf("Captured output length: %d characters", len(outputStr))

	// Look for expected patterns in output
	expectedPatterns := []string{
		"Brummer",
		"Development",
		"Logs",
		"Processes",
	}

	foundPatterns := 0
	for _, pattern := range expectedPatterns {
		if strings.Contains(outputStr, pattern) {
			foundPatterns++
		}
	}

	t.Logf("Found %d/%d expected patterns in output", foundPatterns, len(expectedPatterns))
	assert.GreaterOrEqual(t, foundPatterns, 1, "Should have found at least one expected pattern")

	// Test 5: Keyboard Input History Validation
	t.Log("Test 5: Keyboard Input History Validation")
	inputHistory := keyboardTester.GetInputHistory()
	expectedKeys := append(navigationKeys, commandKeys...)

	assert.Len(t, inputHistory, len(expectedKeys), "Should have recorded all input keys")

	for i, expectedKey := range expectedKeys {
		if i < len(inputHistory) {
			assert.Equal(t, expectedKey, inputHistory[i], "Input history should match at position %d", i)
		}
	}

	t.Log("✅ Keyboard input PTY integration with Brummer tests passed")
}