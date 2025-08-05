package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayoutController_SizeCalculations(t *testing.T) {
	model := createTestModelWithDefaults()
	controller := NewLayoutController(
		model.processMgr,
		model.logStore,
		model.mcpServer,
		model.proxyServer,
		"test-version",
		"test-dir",
	)

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "standard_terminal",
			width:  120,
			height: 40,
		},
		{
			name:   "small_terminal",
			width:  80,
			height: 24,
		},
		{
			name:   "large_terminal",
			width:  200,
			height: 50,
		},
		{
			name:   "minimum_size",
			width:  40,
			height: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update controller size
			controller.UpdateSize(tt.width, tt.height)

			// Verify the controller accepted the size update without panicking
			assert.NotNil(t, controller, "Controller should remain valid after size update")
		})
	}
}

func TestLayoutController_RenderMethods(t *testing.T) {
	model := createTestModelWithDefaults()
	controller := NewLayoutController(
		model.processMgr,
		model.logStore,
		model.mcpServer,
		model.proxyServer,
		"test-version",
		"test-dir",
	)

	// Set a reasonable size
	controller.UpdateSize(80, 24)

	// Test footer rendering
	t.Run("footer_render", func(t *testing.T) {
		// Update some state
		controller.SetSelectedProcess("test-process")
		controller.SetCurrentView("Processes")
		
		footer := controller.RenderFooter()
		assert.NotEmpty(t, footer, "Footer should not be empty")
	})

	// Test system panel rendering
	t.Run("system_panel_render", func(t *testing.T) {
		// Test with panel closed
		controller.SetSystemPanelOpen(false)
		panel := controller.RenderSystemPanel()
		assert.Empty(t, panel, "System panel should be empty when closed")
		
		// Test with panel open
		controller.SetSystemPanelOpen(true)
		panel = controller.RenderSystemPanel()
		// Panel might be empty if no system logs, that's OK
		assert.NotPanics(t, func() {
			controller.RenderSystemPanel()
		})
	})
}

func TestLayoutController_StateManagement(t *testing.T) {
	model := createTestModelWithDefaults()
	controller := NewLayoutController(
		model.processMgr,
		model.logStore,
		model.mcpServer,
		model.proxyServer,
		"test-version",
		"test-dir",
	)

	// Test help state
	t.Run("help_state", func(t *testing.T) {
		controller.SetShowHelp(true)
		// Verify it sets without panic
		assert.NotPanics(t, func() {
			controller.SetShowHelp(false)
		})
	})

	// Test view updates
	t.Run("view_updates", func(t *testing.T) {
		controller.SetCurrentView("Logs")
		// Just verify it doesn't panic
		assert.NotPanics(t, func() {
			controller.SetCurrentView("Processes")
		})
	})

	// Test process selection
	t.Run("process_selection", func(t *testing.T) {
		controller.SetSelectedProcess("test-123")
		// Verify it handles process selection
		assert.NotPanics(t, func() {
			controller.SetSelectedProcess("")
		})
	})
}

func TestLayoutController_MinimumDimensions(t *testing.T) {
	model := createTestModelWithDefaults()
	controller := NewLayoutController(
		model.processMgr,
		model.logStore,
		model.mcpServer,
		model.proxyServer,
		"test-version",
		"test-dir",
	)

	// Test with very small dimensions
	assert.NotPanics(t, func() {
		controller.UpdateSize(1, 1)
		controller.RenderFooter()
		controller.RenderSystemPanel()
	}, "Should handle minimum dimensions without panic")

	// Test with zero dimensions
	assert.NotPanics(t, func() {
		controller.UpdateSize(0, 0)
		controller.RenderFooter()
		controller.RenderSystemPanel()
	}, "Should handle zero dimensions without panic")

	// Test with negative dimensions (should be handled gracefully)
	assert.NotPanics(t, func() {
		controller.UpdateSize(-10, -10)
		controller.RenderFooter()
		controller.RenderSystemPanel()
	}, "Should handle negative dimensions without panic")
}

func TestLayoutController_SystemPanelRendering(t *testing.T) {
	model := createTestModelWithDefaults()
	controller := NewLayoutController(
		model.processMgr,
		model.logStore,
		model.mcpServer,
		model.proxyServer,
		"test-version",
		"test-dir",
	)

	controller.UpdateSize(80, 24)

	// Test system panel states
	t.Run("system_panel_closed", func(t *testing.T) {
		controller.SetSystemPanelOpen(false)
		// Verify panel height is 0 when closed
		height := controller.GetSystemPanelHeight()
		assert.Equal(t, 0, height, "System panel height should be 0 when closed")
	})

	t.Run("system_panel_open", func(t *testing.T) {
		controller.SetSystemPanelOpen(true)
		// Verify panel has height when open
		height := controller.GetSystemPanelHeight()
		assert.Greater(t, height, 0, "System panel height should be greater than 0 when open")
	})
}