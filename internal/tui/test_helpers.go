package tui

import (
	"fmt"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
)

// createTestModelWithDefaults creates a minimal Model for testing
func createTestModelWithDefaults() *Model {
	// Create minimal dependencies
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(100, eventBus)
	
	// Create a temporary directory with a minimal package.json for testing
	tempDir, err := os.MkdirTemp("", "brummer-test-*")
	if err != nil {
		panic(fmt.Sprintf("Failed to create temp directory: %v", err))
	}
	
	// Create minimal package.json
	packageJson := `{
		"name": "test-project",
		"version": "1.0.0",
		"scripts": {
			"test": "echo \"test script\""
		}
	}`
	
	err = os.WriteFile(fmt.Sprintf("%s/package.json", tempDir), []byte(packageJson), 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to create test package.json: %v", err))
	}
	
	processMgr, err := process.NewManager(tempDir, eventBus, true)
	if err != nil {
		panic(fmt.Sprintf("Failed to create test process manager: %v", err))
	}

	// Create config
	defaultProvider := "claude"
	cfg := &config.Config{
		AICoders: &config.AICoderConfig{
			DefaultProvider: &defaultProvider,
			Providers: map[string]*config.ProviderConfig{
				"claude":   {},
				"terminal": {},
			},
		},
	}

	// Use NewModel to create a properly initialized model
	return NewModel(processMgr, logStore, eventBus, nil, nil, 7777, cfg)
}

// TestMessage is a test implementation of tea.Msg
type TestMessage struct {
	Content string
}

// Removed createTestModelWithMockDependencies - use createTestModelWithDefaults instead

// createTestProcessManager creates a real process manager for testing
// This is only used when we need to test process-specific behavior
// For most tests, use createTestModelWithDefaults() which includes a real process manager
func createTestProcessManager(t *testing.T) *process.Manager {
	eventBus := events.NewEventBus()
	mgr, err := process.NewManager(t.TempDir(), eventBus, true)
	if err != nil {
		t.Fatalf("Failed to create test process manager: %v", err)
	}
	return mgr
}

// createTestLogStore creates a real log store for testing
// For most tests, use createTestModelWithDefaults() which includes a real log store
func createTestLogStore(eventBus *events.EventBus) *logs.Store {
	return logs.NewStore(100, eventBus)
}

// waitForMessage is a helper to wait for a specific message type on a channel
func waitForMessage[T any](t *testing.T, ch <-chan tea.Msg, timeout time.Duration) T {
	t.Helper()
	select {
	case msg := <-ch:
		if typedMsg, ok := msg.(T); ok {
			return typedMsg
		}
		var zero T
		t.Fatalf("Expected message of type %T but got %T", zero, msg)
		return zero
	case <-time.After(timeout):
		var zero T
		t.Fatalf("Timeout waiting for message of type %T", zero)
		return zero
	}
}
