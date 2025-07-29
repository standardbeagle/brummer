package tui

import (
	"testing"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAICoderPTYView_GetTerminalSize(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		height       int
		isFullScreen bool
		wantWidth    int
		wantHeight   int
	}{
		{
			name:         "uninitialized dimensions should return defaults",
			width:        0,
			height:       0,
			isFullScreen: false,
			wantWidth:    80,
			wantHeight:   24,
		},
		{
			name:         "negative dimensions after calculation should return defaults",
			width:        5,  // Less than 6, would result in negative
			height:       7,  // Less than 8, would result in negative
			isFullScreen: false,
			wantWidth:    80,
			wantHeight:   24,
		},
		{
			name:         "valid windowed dimensions",
			width:        100,
			height:       40,
			isFullScreen: false,
			wantWidth:    94,  // 100 - 6
			wantHeight:   32,  // 40 - 8
		},
		{
			name:         "valid fullscreen dimensions",
			width:        100,
			height:       40,
			isFullScreen: true,
			wantWidth:    96,  // 100 - 4
			wantHeight:   36,  // 40 - 4
		},
		{
			name:         "fullscreen with small dimensions",
			width:        3,   // Less than 4, would result in negative
			height:       3,   // Less than 4, would result in negative
			isFullScreen: true,
			wantWidth:    80,
			wantHeight:   24,
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

			gotWidth, gotHeight := view.getTerminalSize()

			assert.Equal(t, tt.wantWidth, gotWidth, "width mismatch")
			assert.Equal(t, tt.wantHeight, gotHeight, "height mismatch")
			assert.Greater(t, gotWidth, 0, "width should be positive")
			assert.Greater(t, gotHeight, 0, "height should be positive")
		})
	}
}

func TestTerminalBuffer_Resize_NegativeDimensions(t *testing.T) {
	// This test would have caught the panic
	buffer := &aicoder.TerminalBuffer{
		Lines: make([]aicoder.TerminalLine, 10),
	}

	// Should not panic with negative dimensions
	require.NotPanics(t, func() {
		buffer.Resize(-10, -5)
	})

	// Should use default dimensions
	assert.Equal(t, 80, buffer.Width)
	assert.Equal(t, 24, buffer.Height)
}

func TestAICoderPTYView_AttachToSession_BeforeWindowSize(t *testing.T) {
	// This test simulates the exact scenario that caused the panic
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

	// This should not panic even with uninitialized dimensions
	require.NotPanics(t, func() {
		err = view.AttachToSession(session.ID)
	})
	assert.NoError(t, err)
}

// Mock implementations for testing
type mockBrummerDataProvider struct{}

func (m *mockBrummerDataProvider) GetLastError() *logs.ErrorContext                { return nil }
func (m *mockBrummerDataProvider) GetRecentLogs(count int) []logs.LogEntry         { return nil }
func (m *mockBrummerDataProvider) GetTestFailures() interface{}                    { return nil }
func (m *mockBrummerDataProvider) GetBuildOutput() string                          { return "" }
func (m *mockBrummerDataProvider) GetProcessInfo() interface{}                     { return nil }
func (m *mockBrummerDataProvider) GetDetectedURLs() []logs.URLEntry                { return nil }
func (m *mockBrummerDataProvider) GetRecentProxyRequests(count int) []*proxy.Request { return nil }

type mockEventBus struct{}

func (m *mockEventBus) Emit(event string, data interface{}) {}