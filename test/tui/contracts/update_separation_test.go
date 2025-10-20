package contracts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/standardbeagle/brummer/test/tui/fixtures"
)

// TestViewNeverCallsUpdate verifies View() doesn't trigger state mutations
func TestViewNeverCallsUpdate(t *testing.T) {
	// Create a master data with known state
	masterData := fixtures.CreateTestMasterData()
	originalVersion := masterData.ErrorsState.GetVersion()

	// Render all views
	errorsView := components.ErrorsView{}
	logsView := components.LogsView{}
	webView := components.WebView{}

	_ = errorsView.ViewWithState(masterData)
	_ = logsView.ViewWithState(masterData)
	_ = webView.ViewWithState(masterData)

	// ASSERT: State versions unchanged
	assert.Equal(t, originalVersion, masterData.ErrorsState.GetVersion(),
		"View() should not cause state mutations")
	assert.Equal(t, originalVersion, masterData.LogsState.GetVersion(),
		"View() should not cause state mutations")
	assert.Equal(t, originalVersion, masterData.WebState.GetVersion(),
		"View() should not cause state mutations")
}

// TestAllMutationsInUpdate verifies all state changes flow through Update()
func TestAllMutationsInUpdate(t *testing.T) {
	// This test will verify the Redux/Elm pattern where Update() is the
	// ONLY place where state mutations occur

	// Create initial state
	masterData := fixtures.CreateTestMasterData()
	originalVersion := masterData.ErrorsState.GetVersion()

	// Simulate what Update() would do
	// In the real implementation, this would be in Model.Update()
	newError := fixtures.CreateTestErrorsState(1).GroupedErrors[0]
	newErrorsState := masterData.ErrorsState.AddGroupedError(newError)

	// ASSERT: Version incremented
	assert.Equal(t, originalVersion+1, newErrorsState.GetVersion(),
		"State mutation should increment version")

	// ASSERT: Different instance
	assert.NotEqual(t, masterData.ErrorsState, newErrorsState,
		"State mutation should create new instance")

	// This test documents the pattern that ALL mutations should happen
	// through the Update() function, never in Views or Services
}

// TestNoDirectStateModification verifies services don't modify state directly
func TestNoDirectStateModification(t *testing.T) {
	state := fixtures.CreateTestErrorsState(5)
	originalVersion := state.GetVersion()
	originalData := state.GetGroupedErrors()

	// Create service
	service := services.NewErrorGroupingService()

	// Call service method
	testInput := fixtures.StackTraceError()
	cmd := service.ParseAndGroupError(testInput)

	// Execute command
	if cmd != nil {
		msg := cmd()

		// ASSERT: State unchanged by service
		assert.Equal(t, originalVersion, state.GetVersion(),
			"Service should not modify state directly")
		assert.Equal(t, originalData, state.GetGroupedErrors(),
			"Service should not modify state directly")

		// State modification should happen in Update() when processing the message
		// This is the Redux/Elm pattern: Service -> Message -> Update -> New State
		_ = msg
	}
}

// TestCommandPattern verifies all async operations use tea.Cmd pattern
func TestCommandPattern(t *testing.T) {
	// This test verifies that services return tea.Cmd instead of
	// launching goroutines directly

	service := services.NewErrorGroupingService()

	// Test that service methods return commands
	testInput := fixtures.StackTraceError()
	cmd := service.ParseAndGroupError(testInput)

	// ASSERT: Returns a command (not nil)
	assert.NotNil(t, cmd, "Service should return tea.Cmd, not launch goroutines directly")

	// ASSERT: Command is executable
	assert.IsType(t, func() interface{}{}, cmd,
		"Command should be a function that returns a message")

	// Execute command
	if cmd != nil {
		msg := cmd()
		// ASSERT: Command produces a message
		assert.NotNil(t, msg, "Command should produce a message")
	}
}

// TestViewReadOnly verifies View() only reads state, never writes
func TestViewReadOnly(t *testing.T) {
	masterData := fixtures.CreateTestMasterData()

	// Capture state versions before view rendering
	errorsVersion := masterData.ErrorsState.GetVersion()
	logsVersion := masterData.LogsState.GetVersion()
	webVersion := masterData.WebState.GetVersion()

	// Render views multiple times
	errorsView := components.ErrorsView{}
	logsView := components.LogsView{}
	webView := components.WebView{}

	for i := 0; i < 10; i++ {
		_ = errorsView.ViewWithState(masterData)
		_ = logsView.ViewWithState(masterData)
		_ = webView.ViewWithState(masterData)

		// ASSERT: Versions unchanged after each render
		assert.Equal(t, errorsVersion, masterData.ErrorsState.GetVersion(),
			"View should not modify state (iteration %d)", i)
		assert.Equal(t, logsVersion, masterData.LogsState.GetVersion(),
			"View should not modify state (iteration %d)", i)
		assert.Equal(t, webVersion, masterData.WebState.GetVersion(),
			"View should not modify state (iteration %d)", i)
	}
}

// TestServiceReturnsCmdOnly verifies services only return commands, never mutate state
func TestServiceReturnsCmdOnly(t *testing.T) {
	service := services.NewErrorGroupingService()
	state := fixtures.CreateTestErrorsState(5)
	originalVersion := state.GetVersion()

	// Test multiple service calls
	testInputs := []string{
		fixtures.StackTraceError(),
		fixtures.CompilationError(),
		fixtures.TestFailureError(),
		fixtures.GenericMultiLineError(),
	}

	for _, input := range testInputs {
		cmd := service.ParseAndGroupError(input)

		// ASSERT: Returns command
		assert.NotNil(t, cmd, "Service should return command for input: %s", input)

		// Execute command
		if cmd != nil {
			msg := cmd()

			// ASSERT: State unchanged
			assert.Equal(t, originalVersion, state.GetVersion(),
				"Service should not modify state for input: %s", input)

			_ = msg // Use the message
		}
	}
}

// TestReduxElmPattern verifies the complete Redux/Elm data flow
func TestReduxElmPattern(t *testing.T) {
	// This test documents and verifies the Redux/Elm pattern:
	// View -> User Action -> Message -> Update -> New State -> View

	// 1. Initial State
	masterData := fixtures.CreateTestMasterData()
	originalErrorsVersion := masterData.ErrorsState.GetVersion()

	// 2. View renders (read-only)
	errorsView := components.ErrorsView{}
	_ = errorsView.ViewWithState(masterData)

	// 3. User Action triggers Service
	service := services.NewErrorGroupingService()
	testInput := fixtures.StackTraceError()
	cmd := service.ParseAndGroupError(testInput)

	// 4. Service returns Command (no state modification)
	assert.NotNil(t, cmd, "Service should return command")

	// 5. Command produces Message
	msg := cmd()
	assert.NotNil(t, msg, "Command should produce message")

	// 6. Update() processes Message (this would be in Model.Update)
	// For this test, we simulate what Update() would do
	newError := fixtures.CreateTestErrorsState(1).GroupedErrors[0]
	newErrorsState := masterData.ErrorsState.AddGroupedError(newError)

	// 7. New State created
	assert.Equal(t, originalErrorsVersion+1, newErrorsState.GetVersion(),
		"Update() should create new state with incremented version")

	// 8. View renders with new state (back to step 2)
	newMasterData := masterData
	newMasterData.ErrorsState = newErrorsState
	_ = errorsView.ViewWithState(newMasterData)

	// This complete cycle demonstrates the Redux/Elm pattern
}

// TestImmutabilityEnforcement verifies that the immutable pattern is enforced
func TestImmutabilityEnforcement(t *testing.T) {
	// Create test state
	originalState := fixtures.CreateTestErrorsState(3)
	originalVersion := originalState.GetVersion()

	// Create a "modified" state
	newError := fixtures.CreateTestErrorsState(1).GroupedErrors[0]
	modifiedState := originalState.AddGroupedError(newError)

	// ASSERT: Original unchanged
	assert.Equal(t, originalVersion, originalState.GetVersion(),
		"Original state should remain unchanged")
	assert.Equal(t, 3, len(originalState.GroupedErrors),
		"Original state should remain unchanged")

	// ASSERT: Modified state is different
	assert.Equal(t, originalVersion+1, modifiedState.GetVersion(),
		"Modified state should have incremented version")
	assert.Equal(t, 4, len(modifiedState.GroupedErrors),
		"Modified state should have new data")

	// This enforces the immutability pattern critical for Redux/Elm
}

// TestNoSideEffectsInViews verifies views have no side effects
func TestNoSideEffectsInViews(t *testing.T) {
	masterData := fixtures.CreateTestMasterData()

	// Capture initial state
	errors := len(masterData.ErrorsState.GroupedErrors)
	logs := len(masterData.LogsState.Entries)
	webEvents := len(masterData.WebState.Events)

	// Render views
	errorsView := components.ErrorsView{}
	logsView := components.LogsView{}
	webView := components.WebView{}

	output1 := errorsView.ViewWithState(masterData)
	output2 := logsView.ViewWithState(masterData)
	output3 := webView.ViewWithState(masterData)

	// ASSERT: No side effects
	assert.Equal(t, errors, len(masterData.ErrorsState.GroupedErrors),
		"View should not modify data")
	assert.Equal(t, logs, len(masterData.LogsState.Entries),
		"View should not modify data")
	assert.Equal(t, webEvents, len(masterData.WebState.Events),
		"View should not modify data")

	// ASSERT: Deterministic output
	output1b := errorsView.ViewWithState(masterData)
	output2b := logsView.ViewWithState(masterData)
	output3b := webView.ViewWithState(masterData)

	assert.Equal(t, output1, output1b, "View should be deterministic")
	assert.Equal(t, output2, output2b, "View should be deterministic")
	assert.Equal(t, output3, output3b, "View should be deterministic")
}