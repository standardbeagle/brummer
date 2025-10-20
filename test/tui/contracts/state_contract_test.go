package contracts

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/standardbeagle/brummer/test/tui/fixtures"
)

// TestStateImmutability verifies that state objects remain unchanged after mutations
func TestStateImmutability(t *testing.T) {
	tests := []struct {
		name  string
		state func() interface{}
	}{
		{
			name: "ErrorsState",
			state: func() interface{} {
				return fixtures.CreateTestErrorsState(0)
			},
		},
		{
			name: "LogsState",
			state: func() interface{} {
				return fixtures.CreateTestLogsState(0)
			},
		},
		{
			name: "WebState",
			state: func() interface{} {
				return fixtures.CreateTestWebState(0)
			},
		},
		{
			name: "URLsState",
			state: func() interface{} {
				return fixtures.CreateTestURLsState(0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.state()

			// Try to modify based on state type
			switch s := original.(type) {
			case *state.ErrorsState:
				error := fixtures.CreateTestErrorsState(1).GroupedErrors[0]
				modified := s.AddGroupedError(error)

				// ASSERT: Original unchanged
				assert.Equal(t, 0, len(s.GroupedErrors), "Original state should be unchanged")

				// ASSERT: Modified has new data
				assert.Equal(t, 1, len(modified.GroupedErrors), "Modified state should have new data")

				// ASSERT: Different memory addresses
				assert.NotEqual(t, s, modified, "Original and modified should be different instances")

			case *state.LogsState:
				log := fixtures.CreateTestLogsWithCount(1, 0, 0, 0)[0]
				modified := s.AddLogEntry(log)

				// ASSERT: Original unchanged
				assert.Equal(t, 0, len(s.Entries), "Original state should be unchanged")

				// ASSERT: Modified has new data
				assert.Equal(t, 1, len(modified.Entries), "Modified state should have new data")

				// ASSERT: Different memory addresses
				assert.NotEqual(t, s, modified, "Original and modified should be different instances")

			case *state.WebState:
				event := fixtures.CreateTestWebState(1).Events[0]
				modified := s.AddProxyEvent(event)

				// ASSERT: Original unchanged
				assert.Equal(t, 0, len(s.Events), "Original state should be unchanged")

				// ASSERT: Modified has new data
				assert.Equal(t, 1, len(modified.Events), "Modified state should have new data")

				// ASSERT: Different memory addresses
				assert.NotEqual(t, s, modified, "Original and modified should be different instances")

			case *state.URLsState:
				url := fixtures.CreateTestURLsState(1).URLs[0]
				modified := s.AddURL(url)

				// ASSERT: Original unchanged
				assert.Equal(t, 0, len(s.URLs), "Original state should be unchanged")

				// ASSERT: Modified has new data
				assert.Equal(t, 1, len(modified.URLs), "Modified state should have new data")

				// ASSERT: Different memory addresses
				assert.NotEqual(t, s, modified, "Original and modified should be different instances")
			}
		})
	}
}

// TestNoMutexes verifies that state objects contain no mutex fields
func TestNoMutexes(t *testing.T) {
	tests := []struct {
		name  string
		state interface{}
	}{
		{
			name:  "ErrorsState",
			state: &state.ErrorsState{},
		},
		{
			name:  "LogsState",
			state: &state.LogsState{},
		},
		{
			name:  "WebState",
			state: &state.WebState{},
		},
		{
			name:  "URLsState",
			state: &state.URLsState{},
		},
		{
			name:  "ProcessesState",
			state: &state.ProcessesState{},
		},
		{
			name:  "ProxyState",
			state: &state.ProxyState{},
		},
		{
			name:  "SettingsState",
			state: &state.SettingsState{},
		},
		{
			name:  "CommandWindowState",
			state: &state.CommandWindowState{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateType := reflect.TypeOf(tt.state).Elem()

			for i := 0; i < stateType.NumField(); i++ {
				field := stateType.Field(i)
				fieldType := field.Type.String()

				// ASSERT: No mutex types
				assert.NotContains(t, fieldType, "sync.Mutex",
					"Field %s should not contain sync.Mutex", field.Name)
				assert.NotContains(t, fieldType, "sync.RWMutex",
					"Field %s should not contain sync.RWMutex", field.Name)
			}
		})
	}
}

// TestVersionIncrement verifies that version is incremented on every mutation
func TestVersionIncrement(t *testing.T) {
	tests := []struct {
		name  string
		state func() interface{}
	}{
		{
			name: "ErrorsState",
			state: func() interface{} {
				return fixtures.CreateTestErrorsState(0)
			},
		},
		{
			name: "LogsState",
			state: func() interface{} {
				return fixtures.CreateTestLogsState(0)
			},
		},
		{
			name: "WebState",
			state: func() interface{} {
				return fixtures.CreateTestWebState(0)
			},
		},
		{
			name: "URLsState",
			state: func() interface{} {
				return fixtures.CreateTestURLsState(0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.state()

			// Get original version
			var originalVersion uint64
			switch s := original.(type) {
			case *state.ErrorsState:
				originalVersion = s.GetVersion()
				error := fixtures.CreateTestErrorsState(1).GroupedErrors[0]
				modified := s.AddGroupedError(error)
				assert.Equal(t, originalVersion+1, modified.GetVersion(), "Version should increment")
			case *state.LogsState:
				originalVersion = s.GetVersion()
				log := fixtures.CreateTestLogsWithCount(1, 0, 0, 0)[0]
				modified := s.AddLogEntry(log)
				assert.Equal(t, originalVersion+1, modified.GetVersion(), "Version should increment")
			case *state.WebState:
				originalVersion = s.GetVersion()
				event := fixtures.CreateTestWebState(1).Events[0]
				modified := s.AddProxyEvent(event)
				assert.Equal(t, originalVersion+1, modified.GetVersion(), "Version should increment")
			case *state.URLsState:
				originalVersion = s.GetVersion()
				url := fixtures.CreateTestURLsState(1).URLs[0]
				modified := s.AddURL(url)
				assert.Equal(t, originalVersion+1, modified.GetVersion(), "Version should increment")
			}
		})
	}
}

// TestCopyOnWrite verifies that multiple updates create different instances
func TestCopyOnWrite(t *testing.T) {
	original := fixtures.CreateTestErrorsState(0)

	error1 := fixtures.CreateTestErrorsState(1).GroupedErrors[0]
	error2 := fixtures.CreateTestErrorsState(1).GroupedErrors[0]

	// Multiple updates
	updated1 := original.AddGroupedError(error1)
	updated2 := original.AddGroupedError(error2)

	// ASSERT: All different instances
	assert.NotEqual(t, original, updated1, "Original and updated1 should be different")
	assert.NotEqual(t, original, updated2, "Original and updated2 should be different")
	assert.NotEqual(t, updated1, updated2, "updated1 and updated2 should be different")

	// ASSERT: Original still unchanged
	assert.Equal(t, 0, len(original.GroupedErrors), "Original should remain unchanged")
}

// TestConcurrentReads verifies concurrent reads are safe
func TestConcurrentReads(t *testing.T) {
	state := fixtures.CreateTestErrorsState(10)

	// Launch 100 concurrent readers
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Read operations
			_ = state.GetGroupedErrors()
			_ = state.GetSelectedIndex()
			_ = state.GetFilterProcess()
		}()
	}

	// ASSERT: No race conditions (run with -race flag)
	wg.Wait()
}

// TestStateValidation verifies state validation works
func TestStateValidation(t *testing.T) {
	state := fixtures.CreateTestErrorsState(0)

	// ASSERT: Valid initial state
	issues := state.Validate()
	assert.Empty(t, issues, "Initial state should be valid")

	// Create invalid state (manually)
	invalid := &state.ErrorsState{
		SelectedIndex: 10, // Out of bounds
		GroupedErrors: []state.GroupedError{}, // Empty
		MaxRetention: 100,
		Version:      1,
		LastUpdated:  time.Now(),
	}

	// ASSERT: Validation catches issue
	issues = invalid.Validate()
	assert.NotEmpty(t, issues, "Invalid state should have validation issues")
	assert.Contains(t, issues[0], "index out of bounds", "Should detect index out of bounds")
}

// TestRetentionLimit verifies FIFO retention limit enforcement
func TestRetentionLimit(t *testing.T) {
	state := fixtures.CreateTestErrorsState(0)
	state = state.SetMaxRetention(10)

	// Add 20 errors
	for i := 0; i < 20; i++ {
		error := state.GroupedError{
			ID:           fmt.Sprintf("error-%d", i),
			ProcessName:  "test-process",
			ProcessPID:   1000,
			FirstLine:    fmt.Sprintf("Error %d", i),
			FullContent:  []string{fmt.Sprintf("Error %d", i)},
			ErrorType:    state.ErrorTypeGeneric,
			Timestamp:    time.Now(),
			Version:      uint64(i + 1),
		}
		state = state.AddGroupedError(error)
	}

	// ASSERT: Only last 10 retained
	assert.Equal(t, 10, len(state.GroupedErrors), "Should retain only max limit")

	// ASSERT: Oldest discarded (FIFO)
	assert.Equal(t, "error-10", state.GroupedErrors[0].ID, "Should discard oldest first")
	assert.Equal(t, "error-19", state.GroupedErrors[9].ID, "Should keep newest")
}