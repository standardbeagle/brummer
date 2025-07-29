package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAICommandIntegration tests the full flow of the /ai command
// This would have caught the panic from uninitialized dimensions
func TestAICommandIntegration(t *testing.T) {
	// Create all required components
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)
	logStore := logs.NewStore(10000)
	proxyServer := proxy.NewServer(20888, eventBus)
	
	// Create model with minimal window size (simulates startup conditions)
	model := NewModelWithView(
		processMgr,
		logStore,
		eventBus,
		nil, // MCP server
		proxyServer,
		7777,
		ViewScriptSelector,
		false, // debug mode
	)
	
	// Simulate window not yet resized (0x0) - this was the bug condition
	model.width = 0
	model.height = 0
	
	// Initialize update channel
	model.updateChan = make(chan tea.Msg, 100)
	
	// Initialize the model (would normally happen in BubbleTea)
	initCmd := model.Init()
	if initCmd != nil {
		// Execute init command if any
		go func() {
			msg := initCmd()
			if msg != nil {
				model.updateChan <- msg
			}
		}()
	}
	
	// Simulate the /ai command
	t.Run("AI command with zero dimensions", func(t *testing.T) {
		// This should not panic even with 0x0 dimensions
		require.NotPanics(t, func() {
			model.handleSlashCommand("/ai claude test task")
		})
		
		// Give time for async operations
		time.Sleep(100 * time.Millisecond)
		
		// Verify we switched to AI coder view
		assert.Equal(t, ViewAICoders, model.currentView)
	})
	
	// Now test with proper dimensions
	t.Run("AI command with proper dimensions", func(t *testing.T) {
		// Simulate window resize (what normally happens before user types)
		windowMsg := tea.WindowSizeMsg{
			Width:  120,
			Height: 40,
		}
		
		// Update the model with window size
		newModel, cmd := model.Update(windowMsg)
		model = newModel.(*Model)
		
		// Execute any commands
		if cmd != nil {
			go func() {
				msg := cmd()
				if msg != nil {
					model.updateChan <- msg
				}
			}()
		}
		
		// Now dimensions should be set
		assert.Equal(t, 120, model.width)
		assert.Equal(t, 40, model.height)
		
		// Try the command again with proper dimensions
		model.handleSlashCommand("/ai claude another task")
		
		// Give time for async operations
		time.Sleep(100 * time.Millisecond)
		
		// Should still be in AI coder view
		assert.Equal(t, ViewAICoders, model.currentView)
	})
}

// TestAICommandPTYSessionLifecycle tests the complete PTY session lifecycle
func TestAICommandPTYSessionLifecycle(t *testing.T) {
	// Create components
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)
	logStore := logs.NewStore(10000)
	proxyServer := proxy.NewServer(20888, eventBus)
	
	// Track PTY events
	var ptyEvents []string
	eventBus.Subscribe(events.EventType("pty_session_created"), func(event events.Event) {
		ptyEvents = append(ptyEvents, "created")
	})
	eventBus.Subscribe(events.EventType("pty_session_closed"), func(event events.Event) {
		ptyEvents = append(ptyEvents, "closed")
	})
	
	// Create model
	model := NewModelWithView(
		processMgr,
		logStore,
		eventBus,
		nil,
		proxyServer,
		7777,
		ViewScriptSelector,
		false,
	)
	
	// Set reasonable dimensions
	model.width = 100
	model.height = 30
	model.updateChan = make(chan tea.Msg, 100)
	
	// Initialize
	if cmd := model.Init(); cmd != nil {
		go func() {
			if msg := cmd(); msg != nil {
				model.updateChan <- msg
			}
		}()
	}
	
	t.Run("Create interactive session", func(t *testing.T) {
		// Clear events
		ptyEvents = ptyEvents[:0]
		
		// Create interactive session (no task)
		model.handleSlashCommand("/ai mock")
		
		// Wait for async operations
		time.Sleep(200 * time.Millisecond)
		
		// Should have created a session
		assert.Contains(t, ptyEvents, "created", "PTY session should be created")
		
		// Verify AI coder manager has the session
		if model.aiCoderManager != nil {
			sessions := model.aiCoderManager.GetPTYManager().ListSessions()
			assert.NotEmpty(t, sessions, "Should have at least one PTY session")
		}
	})
	
	t.Run("Create task session", func(t *testing.T) {
		// Clear events
		ptyEvents = ptyEvents[:0]
		
		// Create task session
		model.handleSlashCommand("/ai mock implement feature X")
		
		// Wait for async operations
		time.Sleep(200 * time.Millisecond)
		
		// Should have created another session
		assert.Contains(t, ptyEvents, "created", "Task PTY session should be created")
	})
	
	t.Run("Handle missing provider", func(t *testing.T) {
		// Try with non-existent provider
		model.handleSlashCommand("/ai nonexistent some task")
		
		// Should log an error but not panic
		time.Sleep(100 * time.Millisecond)
		
		// Check logs for error
		logs := logStore.GetAll()
		hasError := false
		for _, log := range logs {
			if log.IsError && contains(log.Content, "nonexistent") {
				hasError = true
				break
			}
		}
		assert.True(t, hasError, "Should log error for unknown provider")
	})
}

// TestPTYViewWindowResize tests that PTY view handles window resize correctly
func TestPTYViewWindowResize(t *testing.T) {
	// Create minimal PTY view
	mockDataProvider := &mockBrummerDataProvider{}
	mockEventBus := &mockEventBus{}
	ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)
	
	view := NewAICoderPTYView(ptyManager)
	
	// Create a session
	session, err := ptyManager.CreateSession("test", "echo", []string{"hello"})
	require.NoError(t, err)
	
	// Attach with zero dimensions (bug scenario)
	view.width = 0
	view.height = 0
	
	require.NotPanics(t, func() {
		err = view.AttachToSession(session.ID)
	})
	assert.NoError(t, err)
	
	// Now resize
	windowMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedView, _ := view.Update(windowMsg)
	view = updatedView
	
	// Verify dimensions updated
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
	
	// Verify terminal was resized
	termWidth, termHeight := view.getTerminalSize()
	assert.Greater(t, termWidth, 0)
	assert.Greater(t, termHeight, 0)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}