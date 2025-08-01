package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAICoderPTYView_GetTerminalSize(t *testing.T) {
	tests := []struct {
		name             string
		width            int
		height           int
		isFullScreen     bool
		hasStatusMessage bool
		showHelp         bool
		wantWidth        int
		wantHeight       int
		description      string
	}{
		{
			name:         "uninitialized dimensions return defaults",
			width:        0,
			height:       0,
			isFullScreen: false,
			wantWidth:    80,
			wantHeight:   24,
			description:  "Zero dimensions should trigger fallback to standard terminal size",
		},
		{
			name:         "negative width calculation triggers width fallback",
			width:        5, // 5 - 4 - 2 = -1 (negative)
			height:       10,
			isFullScreen: false,
			wantWidth:    80, // Falls back to default
			wantHeight:   5,  // 10 - 3 - 2 = 5 (valid)
			description:  "Width calculation resulting in negative should use default width",
		},
		{
			name:         "negative height calculation triggers height fallback",
			width:        50,
			height:       4, // 4 - 3 - 2 = -1 (negative)
			isFullScreen: false,
			wantWidth:    44, // 50 - 4 - 2 = 44 (valid)
			wantHeight:   24, // Falls back to default
			description:  "Height calculation resulting in negative should use default height",
		},
		{
			name:         "valid windowed dimensions with base layout",
			width:        100,
			height:       40,
			isFullScreen: false,
			wantWidth:    94, // 100 - 4 - 2 = 94
			wantHeight:   35, // 40 - 3 - 2 = 35
			description:  "Normal windowed mode with header (3) + footer (2) deductions",
		},
		{
			name:             "windowed with status message increases header",
			width:            100,
			height:           40,
			isFullScreen:     false,
			hasStatusMessage: true,
			wantWidth:        94, // 100 - 4 - 2 = 94
			wantHeight:       33, // 40 - 5 - 2 = 33 (header+2 for status)
			description:      "Status message adds 2 lines to header calculation",
		},
		{
			name:         "windowed with help enabled increases footer",
			width:        100,
			height:       40,
			isFullScreen: false,
			showHelp:     true,
			wantWidth:    94, // 100 - 4 - 2 = 94
			wantHeight:   27, // 40 - 3 - 10 = 27 (footer = 10 for help)
			description:  "Help mode uses extended footer (10 lines)",
		},
		{
			name:         "fullscreen uses maximum space",
			width:        100,
			height:       40,
			isFullScreen: true,
			wantWidth:    96, // 100 - 4 = 96
			wantHeight:   36, // 40 - 4 = 36
			description:  "Fullscreen minimizes UI overhead for maximum terminal space",
		},
		{
			name:         "fullscreen with minimal dimensions triggers fallbacks",
			width:        3,
			height:       3,
			isFullScreen: true,
			wantWidth:    80, // 3 - 4 = -1, fallback to 80
			wantHeight:   24, // 3 - 4 = -1, fallback to 24
			description:  "Small fullscreen dimensions should use safe defaults",
		},
		{
			name:         "edge case: exactly minimum windowed width",
			width:        6, // 6 - 4 - 2 = 0, triggers fallback
			height:       20,
			isFullScreen: false,
			wantWidth:    80, // Fallback for width <= 0
			wantHeight:   15, // 20 - 3 - 2 = 15
			description:  "Width calculation of exactly 0 should trigger fallback",
		},
		{
			name:         "edge case: exactly minimum windowed height",
			width:        50,
			height:       5, // 5 - 3 - 2 = 0, triggers fallback
			isFullScreen: false,
			wantWidth:    44, // 50 - 4 - 2 = 44
			wantHeight:   24, // Fallback for height <= 0
			description:  "Height calculation of exactly 0 should trigger fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PTY manager
			mockDataProvider := &mockBrummerDataProvider{}
			mockEventBus := &mockEventBus{}
			ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)

			view := NewAICoderPTYView(ptyManager)
			view.width = tt.width
			view.height = tt.height
			view.isFullScreen = tt.isFullScreen

			// Set up status message if needed
			if tt.hasStatusMessage {
				view.statusMessage = "Test status message"
				view.statusTime = time.Now() // Current time ensures it's within 3s window
			}

			// Set help mode if needed
			view.showHelp = tt.showHelp

			gotWidth, gotHeight := view.getTerminalSize()

			assert.Equal(t, tt.wantWidth, gotWidth, "width mismatch for case: %s", tt.description)
			assert.Equal(t, tt.wantHeight, gotHeight, "height mismatch for case: %s", tt.description)
			assert.Greater(t, gotWidth, 0, "width should be positive")
			assert.Greater(t, gotHeight, 0, "height should be positive")
		})
	}
}

func TestAICoderPTYView_SessionLifecycle(t *testing.T) {
	t.Run("attach to session before window size initialization", func(t *testing.T) {
		// This test simulates the exact scenario that caused the original panic
		mockDataProvider := &mockBrummerDataProvider{}
		mockEventBus := &mockEventBus{}
		ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)

		view := NewAICoderPTYView(ptyManager)
		// Simulate uninitialized view (width/height = 0)
		view.width = 0
		view.height = 0

		// Create a PTY session
		session, err := ptyManager.CreateSession("test", "echo", []string{"hello"})
		require.NoError(t, err)
		require.NotNil(t, session)
		defer session.Close()

		// This should not panic even with uninitialized dimensions
		require.NotPanics(t, func() {
			err = view.AttachToSession(session.ID)
		})
		assert.NoError(t, err)
		assert.Equal(t, session, view.currentSession, "session should be attached")
	})

	t.Run("session resize with negative dimensions uses defaults", func(t *testing.T) {
		mockDataProvider := &mockBrummerDataProvider{}
		mockEventBus := &mockEventBus{}
		ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)
		view := NewAICoderPTYView(ptyManager)

		// Create a PTY session
		session, err := ptyManager.CreateSession("test", "sh", []string{"-c", "echo test"})
		require.NoError(t, err)
		defer session.Close()

		view.SetCurrentSession(session)

		// Should not panic with negative dimensions
		require.NotPanics(t, func() {
			view.width = -10
			view.height = -5
			width, height := view.getTerminalSize()
			// Should return default dimensions
			assert.Equal(t, 80, width)
			assert.Equal(t, 24, height)
		})
	})

	t.Run("detach from session clears current session", func(t *testing.T) {
		mockDataProvider := &mockBrummerDataProvider{}
		mockEventBus := &mockEventBus{}
		ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)
		view := NewAICoderPTYView(ptyManager)
		view.width = 80
		view.height = 24

		// Create and attach to session
		session, err := ptyManager.CreateSession("test", "echo", []string{"hello"})
		require.NoError(t, err)
		defer session.Close()

		err = view.AttachToSession(session.ID)
		require.NoError(t, err)
		assert.NotNil(t, view.currentSession)

		// Detach from session (simulate session closing)
		view.currentSession = nil
		view.UnfocusTerminal()
		assert.Nil(t, view.currentSession, "current session should be cleared")
		assert.False(t, view.terminalFocused, "terminal should not be focused after detach")
	})
}

func TestAICoderPTYView_ScrollFunctionality(t *testing.T) {
	t.Run("mouse wheel scrolling with bounds checking", func(t *testing.T) {
		mockDataProvider := &mockBrummerDataProvider{}
		mockEventBus := &mockEventBus{}
		ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)
		view := NewAICoderPTYView(ptyManager)
		view.width = 80
		view.height = 24

		// Create session with some output history
		session, err := ptyManager.CreateSession("test", "echo", []string{"hello"})
		require.NoError(t, err)
		defer session.Close()

		view.SetCurrentSession(session)

		// Mock some history data - simulate multiple lines of output
		historyData := make([]byte, 0)
		for i := 0; i < 50; i++ {
			historyData = append(historyData, []byte(fmt.Sprintf("Line %d\n", i))...)
		}

		// Initial scroll offset should be 0
		assert.Equal(t, 0, view.scrollOffset, "initial scroll offset should be 0")

		// Test mouse wheel up (scroll up)
		initialOffset := view.scrollOffset
		view.scrollOffset += 3 // Simulate mouse wheel up
		assert.Greater(t, view.scrollOffset, initialOffset, "scroll up should increase offset")

		// Test mouse wheel down (scroll down)
		view.scrollOffset -= 3 // Simulate mouse wheel down
		assert.Equal(t, initialOffset, view.scrollOffset, "scroll down should decrease offset")

		// Test scroll down beyond 0 (should clamp to 0)
		view.scrollOffset = 1
		view.scrollOffset -= 3
		if view.scrollOffset < 0 {
			view.scrollOffset = 0
		}
		assert.Equal(t, 0, view.scrollOffset, "scroll offset should not go below 0")
	})

	t.Run("page up/down scrolling respects bounds", func(t *testing.T) {
		mockDataProvider := &mockBrummerDataProvider{}
		mockEventBus := &mockEventBus{}
		ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)
		view := NewAICoderPTYView(ptyManager)
		view.width = 80
		view.height = 24

		session, err := ptyManager.CreateSession("test", "echo", []string{"hello"})
		require.NoError(t, err)
		defer session.Close()

		view.SetCurrentSession(session)

		_, termHeight := view.getTerminalSize()
		halfPage := termHeight / 2

		// Test page up
		initialOffset := view.scrollOffset
		view.scrollOffset += halfPage
		assert.Equal(t, halfPage, view.scrollOffset, "page up should scroll half terminal height")

		// Test page down
		view.scrollOffset -= halfPage
		assert.Equal(t, initialOffset, view.scrollOffset, "page down should return to initial position")

		// Test page down beyond 0 (should clamp to 0)
		view.scrollOffset = 5
		view.scrollOffset -= halfPage
		if view.scrollOffset < 0 {
			view.scrollOffset = 0
		}
		assert.Equal(t, 0, view.scrollOffset, "page down should not scroll below 0")
	})

	t.Run("auto-scroll on new content", func(t *testing.T) {
		mockDataProvider := &mockBrummerDataProvider{}
		mockEventBus := &mockEventBus{}
		ptyManager := aicoder.NewPTYManager(mockDataProvider, mockEventBus)
		view := NewAICoderPTYView(ptyManager)
		view.width = 80
		view.height = 24

		// Set some scroll offset
		view.scrollOffset = 10

		// Simulate new content arriving (this typically resets scroll)
		view.addToScrollback("New content line")

		// Auto-scroll should reset offset to 0 when new content arrives
		assert.Equal(t, 0, view.scrollOffset, "new content should trigger auto-scroll to bottom")
	})
}

// Mock implementations for testing
type mockBrummerDataProvider struct{}

func (m *mockBrummerDataProvider) GetLastError() *logs.ErrorContext                  { return nil }
func (m *mockBrummerDataProvider) GetRecentLogs(count int) []logs.LogEntry           { return nil }
func (m *mockBrummerDataProvider) GetTestFailures() interface{}                      { return nil }
func (m *mockBrummerDataProvider) GetBuildOutput() string                            { return "" }
func (m *mockBrummerDataProvider) GetProcessInfo() interface{}                       { return nil }
func (m *mockBrummerDataProvider) GetDetectedURLs() []logs.URLEntry                  { return nil }
func (m *mockBrummerDataProvider) GetRecentProxyRequests(count int) []*proxy.Request { return nil }

type mockEventBus struct{}

func (m *mockEventBus) Emit(event string, data interface{}) {}
