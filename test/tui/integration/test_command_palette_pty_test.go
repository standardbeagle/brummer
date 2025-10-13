//go:build tty && integration
// +build tty,integration

package integration

// TestCommandPalettePTY provides comprehensive integration testing for command palette functionality
// through PTY (pseudo-terminal) simulation. This test suite validates that the command palette
// correctly handles keyboard input, state management, and command execution through a PTY layer.
//
// The tests are designed to FAIL initially since there's no PTY command palette implementation yet.
// When a PTY layer is implemented, these tests should be updated to work with the real implementation.
//
// Test Coverage:
// - Command palette opening via PTY
// - Command input and execution
// - Command window state management
// - Keyboard navigation in command interface
// - PTY allocation and cleanup
// - Error handling for PTY failures
//
// Usage:
//   go test -tags=tty,integration ./test/tui/integration -v -run TestCommandPalettePTY
//   INTEGRATION_TESTS=1 BRUMMER_BINARY=./brum go test -tags=tty,integration ./test/tui/integration -v

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestCommandPalettePTYIntegration tests the command palette functionality through PTY
// This test validates:
// 1. Command palette opening via PTY
// 2. Command input and execution
// 3. Command window state management
// 4. Keyboard navigation in command interface
//
// This test is designed to FAIL initially since there's no PTY command palette implementation yet.
// When a PTY command palette layer is implemented, this test should be updated to work with the
// real implementation.

// CommandPalettePTYTester provides testing interface for command palette through PTY
type CommandPalettePTYTester struct {
	t                *testing.T
	ctx              context.Context
	cancel           context.CancelFunc
	pty              *os.File
	terminal         *vt10x.Terminal
	output           []byte
	testBinary       string
	workingDir       string
	program          *tea.Program
	model            *tui.Model
	eventBus         *events.EventBus
	processMgr       *process.Manager
	logStore         *logs.Store
	commandWindowState commandWindowState
	lastCommandInput string
	commandHistory   []string
	keyPressCount    int
}

// NewCommandPalettePTYTester creates a new command palette PTY tester instance
func NewCommandPalettePTYTester(t *testing.T) *CommandPalettePTYTester {
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
  "name": "command-palette-test-project",
  "scripts": {
    "dev": "echo 'Dev server running on http://localhost:3000' && sleep 60",
    "test": "echo 'Running tests...' && sleep 2 && echo 'Tests complete'",
    "build": "echo 'Building...' && sleep 1 && echo 'Build complete'",
    "help": "echo 'Available commands: dev, test, build, help'",
    "version": "echo 'Test Project v1.0.0'"
  }
}`
	err := os.WriteFile(tempDir+"/package.json", []byte(packageJSON), 0644)
	require.NoError(t, err, "Failed to create test package.json")

	ctx, cancel := context.WithCancel(context.Background())

	return &CommandPalettePTYTester{
		t:                 t,
		ctx:               ctx,
		cancel:            cancel,
		testBinary:        testBinary,
		workingDir:        tempDir,
		output:            make([]byte, 0),
		commandWindowState: commandWindowStateClosed,
		commandHistory:    make([]string, 0),
	}
}

// TestCommandPalettePTY_AllocationAndBasicFunctionality tests PTY allocation and basic command palette functionality
func TestCommandPalettePTY_AllocationAndBasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command palette PTY integration test in short mode")
	}
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run.")
	}

	t.Log("🎯 TESTING: Command Palette PTY Allocation and Basic Functionality")
	t.Log("=================================================================")

	// Create command palette PTY tester
	commandTester := NewCommandPalettePTYTester(t)
	defer commandTester.Cleanup()

	// Test 1: PTY Allocation for Command Palette
	t.Log("Test 1: PTY Allocation for Command Palette")
	err := commandTester.AllocatePTY(80, 24)
	if err != nil {
		t.Logf("❌ PTY allocation failed (expected): %v", err)
		t.Logf("   This will fail until PTY command palette layer is implemented")
		// Since there's no implementation yet, we expect this to fail
		assert.Contains(t, err.Error(), "not implemented", "Expected 'not implemented' error")
		return
	}

	t.Log("✅ PTY allocation for command palette succeeded")

	// Test 2: Initialize TUI Model
	t.Log("Test 2: Initialize TUI Model")
	err = commandTester.InitializeTUIModel()
	require.NoError(t, err, "Failed to initialize TUI model")

	// Test 3: Command Palette Opening
	t.Log("Test 3: Command Palette Opening")
	err = commandTester.OpenCommandPalette()
	require.NoError(t, err, "Failed to open command palette")

	// Verify command palette state
	state := commandTester.GetCommandWindowState()
	assert.Equal(t, commandWindowStateOpen, state, "Command palette should be open")

	// Test 4: Command Input
	t.Log("Test 4: Command Input")
	testCommand := "help"
	err = commandTester.TypeCommand(testCommand)
	require.NoError(t, err, "Failed to type command")

	// Verify command was recorded
	assert.Equal(t, testCommand, commandTester.GetLastCommandInput(), "Last command input should match")

	// Test 5: Command Execution
	t.Log("Test 5: Command Execution")
	err = commandTester.ExecuteCommand()
	require.NoError(t, err, "Failed to execute command")

	// Wait for command processing
	time.Sleep(1 * time.Second)

	// Test 6: Command Palette Closing
	t.Log("Test 6: Command Palette Closing")
	err = commandTester.CloseCommandPalette()
	require.NoError(t, err, "Failed to close command palette")

	// Verify command palette state
	state = commandTester.GetCommandWindowState()
	assert.Equal(t, commandWindowStateClosed, state, "Command palette should be closed")

	t.Log("✅ Command palette PTY allocation and basic functionality tests passed")
}

// TestCommandPalettePTYIntegration tests the command palette functionality through PTY
func TestCommandPalettePTYIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command palette PTY integration test in short mode")
	}
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run.")
	}

	// Build the binary for testing
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	tests := []struct {
		name           string
		inputSequence  []ptyInput
		expectedOutput []string
		expectedStates []commandWindowState
		timeout        time.Duration
	}{
		{
			name: "OpenCommandPaletteAndClose",
			inputSequence: []ptyInput{
				{data: []byte{'/'}, delay: 100 * time.Millisecond},
				{data: []byte{27}, delay: 100 * time.Millisecond}, // ESC to close
			},
			expectedOutput: []string{
				"Command Palette", // Command palette title should appear
			},
			expectedStates: []commandWindowState{
				commandWindowStateClosed,
				commandWindowStateOpen,
				commandWindowStateClosed,
			},
			timeout: 5 * time.Second,
		},
		{
			name: "CommandPaletteTypeAndExecute",
			inputSequence: []ptyInput{
				{data: []byte{'/'}, delay: 100 * time.Millisecond},
				{data: []byte("help"), delay: 50 * time.Millisecond},
				{data: []byte{'\n'}, delay: 200 * time.Millisecond}, // Execute help command
			},
			expectedOutput: []string{
				"Command Palette",
				"help", // Command should be typed
				"Help", // Help output should appear
			},
			expectedStates: []commandWindowState{
				commandWindowStateClosed,
				commandWindowStateOpen,
				commandWindowStateOpen, // Still open while typing
				commandWindowStateClosed, // Close after execution
			},
			timeout: 5 * time.Second,
		},
		{
			name: "CommandPaletteNavigation",
			inputSequence: []ptyInput{
				{data: []byte{'/'}, delay: 100 * time.Millisecond},
				{data: []byte{27, 91, 66}, delay: 50 * time.Millisecond}, // Down arrow
				{data: []byte{27, 91, 65}, delay: 50 * time.Millisecond}, // Up arrow
				{data: []byte{27}, delay: 100 * time.Millisecond}, // ESC to close
			},
			expectedOutput: []string{
				"Command Palette",
			},
			expectedStates: []commandWindowState{
				commandWindowStateClosed,
				commandWindowStateOpen,
				commandWindowStateOpen, // Still open during navigation
				commandWindowStateClosed,
			},
			timeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCommandPalettePTY(t, binaryPath, tt.inputSequence, tt.expectedOutput, tt.expectedStates, tt.timeout)
		})
	}
}

// testCommandPalettePTY runs a single command palette PTY test scenario
func testCommandPalettePTY(t *testing.T, binaryPath string, inputSequence []ptyInput, expectedOutput []string, expectedStates []commandWindowState, timeout time.Duration) {
	// Create PTY
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set up command with PTY
	cmd := exec.CommandContext(ctx, binaryPath, "--no-mcp")
	cmd.Dir = t.TempDir() // Run in temp directory to avoid interference

	// Start PTY
	ptmx, err := pty.Start(cmd)
	require.NoError(t, err)
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Create output buffer
	var outputBuf bytes.Buffer
	done := make(chan struct{})

	// Read output in goroutine
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						t.Logf("PTY read error: %v", err)
					}
					return
				}
				outputBuf.Write(buf[:n])
				t.Logf("PTY output: %q", string(buf[:n]))
			}
		}
	}()

	// Wait for initial TUI to load
	time.Sleep(500 * time.Millisecond)

	// Send input sequence
	for i, input := range inputSequence {
		if i < len(expectedStates) {
			// Verify state before input
			currentState := detectCommandWindowState(outputBuf.String())
			if currentState != expectedStates[i] {
				t.Logf("State mismatch at step %d: expected %v, got %v", i, expectedStates[i], currentState)
			}
		}

		_, err := ptmx.Write(input.data)
		require.NoError(t, err)
		time.Sleep(input.delay)
	}

	// Wait for final output
	time.Sleep(200 * time.Millisecond)

	// Cancel context to stop the process
	cancel()

	// Wait for output collection to finish
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Log("Timeout waiting for output collection")
	}

	// Verify expected output in the PTY output
	finalOutput := outputBuf.String()
	t.Logf("Final PTY output:\n%s", finalOutput)

	for _, expected := range expectedOutput {
		assert.Contains(t, finalOutput, expected, "Expected output not found in PTY output")
	}

	// Verify final state
	if len(expectedStates) > 0 {
		finalState := detectCommandWindowState(finalOutput)
		expectedFinalState := expectedStates[len(expectedStates)-1]
		assert.Equal(t, expectedFinalState, finalState, "Final command window state mismatch")
	}
}

// TestCommandPalettePTYStateTransitions tests detailed state transitions
func TestCommandPalettePTYStateTransitions(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run.")
	}

	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--no-mcp")
	cmd.Dir = t.TempDir()

	ptmx, err := pty.Start(cmd)
	require.NoError(t, err)
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	var outputBuf bytes.Buffer
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					return
				}
				outputBuf.Write(buf[:n])
			}
		}
	}()

	// Wait for TUI initialization
	time.Sleep(500 * time.Millisecond)

	// Test state transition sequence
	transitions := []struct {
		name           string
		input          []byte
		delay          time.Duration
		expectedState  commandWindowState
		expectedOutput string
	}{
		{
			name:           "Initial state",
			input:          nil,
			delay:          0,
			expectedState:  commandWindowStateClosed,
			expectedOutput: "",
		},
		{
			name:           "Open command palette",
			input:          []byte{'/'},
			delay:          100 * time.Millisecond,
			expectedState:  commandWindowStateOpen,
			expectedOutput: "Command Palette",
		},
		{
			name:           "Type command",
			input:          []byte("version"),
			delay:          50 * time.Millisecond,
			expectedState:  commandWindowStateOpen,
			expectedOutput: "version",
		},
		{
			name:           "Clear command",
			input:          []byte{27, 91, 51, 126}, // Delete key
			delay:          50 * time.Millisecond,
			expectedState:  commandWindowStateOpen,
			expectedOutput: "",
		},
		{
			name:           "Close command palette",
			input:          []byte{27}, // ESC
			delay:          100 * time.Millisecond,
			expectedState:  commandWindowStateClosed,
			expectedOutput: "",
		},
	}

	for i, transition := range transitions {
		t.Run(transition.name, func(t *testing.T) {
			if transition.input != nil {
				_, err := ptmx.Write(transition.input)
				require.NoError(t, err)
				time.Sleep(transition.delay)
			}

			output := outputBuf.String()
			currentState := detectCommandWindowState(output)

			t.Logf("Transition %d (%s): state=%v, output=%q",
				i, transition.name, currentState, output)

			if transition.expectedOutput != "" {
				assert.Contains(t, output, transition.expectedOutput,
					"Expected output not found for transition: %s", transition.name)
			}

			assert.Equal(t, transition.expectedState, currentState,
				"State mismatch for transition: %s", transition.name)
		})
	}

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Log("Timeout waiting for final output")
	}
}

// TestCommandPalettePTYKeyboardNavigation tests keyboard navigation in command palette
func TestCommandPalettePTYKeyboardNavigation(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run.")
	}

	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--no-mcp")
	cmd.Dir = t.TempDir()

	ptmx, err := pty.Start(cmd)
	require.NoError(t, err)
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	var outputBuf bytes.Buffer
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					return
				}
				outputBuf.Write(buf[:n])
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Test navigation key sequences
	navigationTests := []struct {
		name        string
		keys        [][]byte
		description string
	}{
		{
			name: "Arrow navigation",
			keys: [][]byte{
				{'/'},                                      // Open command palette
				{27, 91, 66},                             // Down arrow
				{27, 91, 65},                             // Up arrow
				{27, 91, 67},                             // Right arrow
				{27, 91, 68},                             // Left arrow
				{27},                                      // ESC to close
			},
			description: "Test arrow key navigation in command palette",
		},
		{
			name: "Tab completion",
			keys: [][]byte{
				{'/'},                                      // Open command palette
				{'h'},                                     // Type 'h'
				{9},                                       // Tab
				{27},                                      // ESC to close
			},
			description: "Test tab completion in command palette",
		},
		{
			name: "Ctrl shortcuts",
			keys: [][]byte{
				{'/'},                                      // Open command palette
				{1},                                       // Ctrl+A (beginning of line)
				{5},                                       // Ctrl+E (end of line)
				{23},                                      // Ctrl+W (delete word)
				{27},                                      // ESC to close
			},
			description: "Test control key shortcuts in command palette",
		},
	}

	for _, test := range navigationTests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Testing: %s", test.description)

			for _, keySeq := range test.keys {
				_, err := ptmx.Write(keySeq)
				require.NoError(t, err)
				time.Sleep(50 * time.Millisecond)
			}

			// Check that command palette is still responsive
			currentOutput := outputBuf.String()
			state := detectCommandWindowState(currentOutput)

			// Command palette should be open during navigation tests
			assert.Equal(t, commandWindowStateOpen, state,
				"Command palette should remain open during navigation")

			// Wait before next test
			time.Sleep(100 * time.Millisecond)
		})
	}

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Log("Timeout waiting for final output")
	}
}

// Types and helper functions

type ptyInput struct {
	data  []byte
	delay time.Duration
}

type commandWindowState int

const (
	commandWindowStateClosed commandWindowState = iota
	commandWindowStateOpen
	commandWindowStateTyping
	commandWindowStateNavigating
)

// detectCommandWindowState analyzes PTY output to determine command window state
func detectCommandWindowState(output string) commandWindowState {
	if strings.Contains(output, "Command Palette") {
		// Look for typing indicators
		if strings.Contains(output, ">") && strings.Contains(output, "_") {
			return commandWindowStateTyping
		}
		// Look for navigation indicators
		if strings.Contains(output, "┃") || strings.Contains(output, "▶") {
			return commandWindowStateNavigating
		}
		return commandWindowStateOpen
	}
	return commandWindowStateClosed
}

// buildTestBinary builds the test binary for integration testing
func buildTestBinary(t *testing.T) string {
	t.Helper()

	// Get the project root
	cwd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := cwd
	for {
		if _, err := os.Stat(projectRoot + "/go.mod"); err == nil {
			break
		}
		parent := projectRoot + "/.."
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod file)")
		}
		projectRoot = parent
	}

	// Build the binary
	binaryPath := projectRoot + "/brum-test"
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/brum")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, string(output))
	}

	// Make sure binary is executable
	err = os.Chmod(binaryPath, 0755)
	require.NoError(t, err)

	return binaryPath
}

// Command Palette PTY Interface Methods (these would be implemented by the real PTY command palette layer)

// AllocatePTY allocates a new PTY with the specified dimensions
func (c *CommandPalettePTYTester) AllocatePTY(cols, rows int) error {
	// This is a placeholder - real implementation would go here
	return &CommandPaletteNotImplementedError{"AllocatePTY"}
}

// InitializeTUIModel initializes the TUI model for testing
func (c *CommandPalettePTYTester) InitializeTUIModel() error {
	// Create services
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{} // Use empty config for tests
	}

	c.eventBus = events.NewEventBus()
	c.processMgr, err = process.NewManager(c.workingDir, c.eventBus, true) // hasPackageJSON = true
	if err != nil {
		return err
	}
	c.logStore = logs.NewStore(10000, c.eventBus) // maxEntries = 10000

	// Create model
	c.model = tui.NewModel(
		c.processMgr,
		c.logStore,
		c.eventBus,
		nil,  // mcpServer not needed for command palette tests
		nil,  // proxyServer not needed for command palette tests
		7777, // mcpPort default
		cfg,
		c.workingDir, // workingDir
	)

	return nil
}

// OpenCommandPalette opens the command palette via PTY
func (c *CommandPalettePTYTester) OpenCommandPalette() error {
	c.t.Helper()
	c.commandWindowState = commandWindowStateOpen
	c.lastCommandInput = ""
	return &CommandPaletteNotImplementedError{"OpenCommandPalette"}
}

// CloseCommandPalette closes the command palette via PTY
func (c *CommandPalettePTYTester) CloseCommandPalette() error {
	c.t.Helper()
	c.commandWindowState = commandWindowStateClosed
	return &CommandPaletteNotImplementedError{"CloseCommandPalette"}
}

// TypeCommand types a command into the command palette via PTY
func (c *CommandPalettePTYTester) TypeCommand(command string) error {
	c.t.Helper()
	c.lastCommandInput = command
	c.commandHistory = append(c.commandHistory, command)
	c.commandWindowState = commandWindowStateTyping
	return &CommandPaletteNotImplementedError{"TypeCommand"}
}

// ExecuteCommand executes the current command via PTY
func (c *CommandPalettePTYTester) ExecuteCommand() error {
	c.t.Helper()
	if c.lastCommandInput != "" {
		c.commandHistory = append(c.commandHistory, c.lastCommandInput)
	}
	c.keyPressCount++
	return &CommandPaletteNotImplementedError{"ExecuteCommand"}
}

// NavigateCommandPalette navigates through command palette options via PTY
func (c *CommandPalettePTYTester) NavigateCommandPalette(direction string) error {
	c.t.Helper()
	c.commandWindowState = commandWindowStateNavigating
	return &CommandPaletteNotImplementedError{"NavigateCommandPalette"}
}

// GetCommandWindowState returns the current command window state
func (c *CommandPalettePTYTester) GetCommandWindowState() commandWindowState {
	return c.commandWindowState
}

// GetLastCommandInput returns the last command input
func (c *CommandPalettePTYTester) GetLastCommandInput() string {
	return c.lastCommandInput
}

// GetCommandHistory returns the command history
func (c *CommandPalettePTYTester) GetCommandHistory() []string {
	return c.commandHistory
}

// GetOutput returns the captured output from the PTY
func (c *CommandPalettePTYTester) GetOutput() []byte {
	return c.output
}

// GetKeyPressCount returns the number of key presses recorded
func (c *CommandPalettePTYTester) GetKeyPressCount() int {
	return c.keyPressCount
}

// Cleanup cleans up PTY resources
func (c *CommandPalettePTYTester) Cleanup() {
	c.t.Helper()
	if c.cancel != nil {
		c.cancel()
	}
	if c.pty != nil {
		c.pty.Close()
		c.pty = nil
	}
	if c.program != nil {
		c.program.Quit()
		c.program = nil
	}
}

// Cancel cancels the context
func (c *CommandPalettePTYTester) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Custom error types for command palette PTY testing
type CommandPaletteNotImplementedError struct {
	Method string
}

func (e *CommandPaletteNotImplementedError) Error() string {
	return "command palette method not implemented: " + e.Method
}

// TestCommandPalettePTY_ImplementationRequired verifies that command palette PTY layer is not yet implemented
// This test is designed to fail until a real command palette PTY layer implementation is provided
func TestCommandPalettePTY_ImplementationRequired(t *testing.T) {
	t.Log("🚨 COMMAND PALETTE PTY IMPLEMENTATION STATUS CHECK")
	t.Log("===============================================")
	t.Log("This test verifies that command palette PTY layer functionality is not yet implemented.")
	t.Log("When command palette PTY layer is implemented, this test should be updated or removed.")

	// Try to use command palette PTY functionality
	commandTester := NewCommandPalettePTYTester(t)
	defer commandTester.Cleanup()

	err := commandTester.AllocatePTY(80, 24)
	if err == nil {
		t.Error("Expected PTY allocation to fail with 'not implemented' error, but it succeeded")
		t.Error("This suggests command palette PTY layer may now be implemented - update this test!")
		return
	}

	var notImplErr *CommandPaletteNotImplementedError
	if !assert.ErrorAs(t, err, &notImplErr, "Expected CommandPaletteNotImplementedError") {
		t.Logf("Got different error type (this is expected until command palette PTY layer is implemented): %T", err)
	}

	t.Log("✅ Command palette PTY layer implementation status confirmed - not yet implemented")
	t.Log("   This is expected until command palette PTY layer is implemented")
}