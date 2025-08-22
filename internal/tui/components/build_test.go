package components_test

import (
	"testing"
	
	"github.com/standardbeagle/brummer/internal/tui/components"
)

// TestComponentSystemBuild verifies that all components compile correctly
func TestComponentSystemBuild(t *testing.T) {
	// This test simply ensures all components can be instantiated
	// It's a build-time check more than a runtime test
	
	t.Run("Core Components", func(t *testing.T) {
		// Base component
		_ = components.BaseComponent{}
		
		// View switcher
		switcher := components.NewViewSwitcher()
		if switcher == nil {
			t.Error("Failed to create ViewSwitcher")
		}
		
		// Notification component
		notif := components.NewNotificationComponent(nil)
		if notif == nil {
			t.Error("Failed to create NotificationComponent")
		}
		
		// Layout component
		layout := components.NewLayoutComponent()
		if layout == nil {
			t.Error("Failed to create LayoutComponent")
		}
	})
	
	t.Run("View Components", func(t *testing.T) {
		// Process view
		processView := components.NewProcessViewComponent(nil)
		if processView == nil {
			t.Error("Failed to create ProcessViewComponent")
		}
		
		// Logs view
		logsView := components.NewLogsViewComponent(nil)
		if logsView == nil {
			t.Error("Failed to create LogsViewComponent")
		}
		
		// Errors view
		errorsView := components.NewErrorsViewComponent()
		if errorsView == nil {
			t.Error("Failed to create ErrorsViewComponent")
		}
		
		// URLs view
		urlsView := components.NewURLsViewComponent()
		if urlsView == nil {
			t.Error("Failed to create URLsViewComponent")
		}
		
		// Web view
		webView := components.NewWebViewComponent()
		if webView == nil {
			t.Error("Failed to create WebViewComponent")
		}
		
		// AI Coders view
		aiView := components.NewAICodersViewComponent()
		if aiView == nil {
			t.Error("Failed to create AICodersViewComponent")
		}
		
		// Settings view
		settingsView := components.NewSettingsViewComponent()
		if settingsView == nil {
			t.Error("Failed to create SettingsViewComponent")
		}
		
		// Script selector
		scriptSelector := components.NewScriptSelectorComponent(nil, nil)
		if scriptSelector == nil {
			t.Error("Failed to create ScriptSelectorComponent")
		}
	})
	
	t.Run("UI Components", func(t *testing.T) {
		// Command input
		cmdInput := components.NewCommandInputComponent()
		if cmdInput == nil {
			t.Error("Failed to create CommandInputComponent")
		}
	})
	
	t.Run("Message Types", func(t *testing.T) {
		// Test that message types are defined
		_ = components.ProcessStartMsg{}
		_ = components.ProcessStopMsg{}
		_ = components.LogsUpdateMsg{}
		_ = components.ErrorOccurredMsg{}
		_ = components.URLDetectedMsg{}
		_ = components.SystemNotificationMsg{}
		_ = components.ViewChangeMsg{}
		_ = components.CommandExecuteMsg{}
	})
}

// TestComponentIntegration verifies component integration
func TestComponentIntegration(t *testing.T) {
	t.Run("ViewSwitcher Integration", func(t *testing.T) {
		switcher := components.NewViewSwitcher()
		
		// Add some views
		processView := components.NewProcessViewComponent(nil)
		logsView := components.NewLogsViewComponent(nil)
		
		switcher.AddView("processes", processView)
		switcher.AddView("logs", logsView)
		
		// Test switching (SwitchTo doesn't return an error)
		switcher.SwitchTo("logs")
		
		// Test current view returns string
		current := switcher.CurrentView()
		if current != "logs" {
			t.Errorf("Expected current view to be 'logs', got '%s'", current)
		}
	})
	
	t.Run("Layout Integration", func(t *testing.T) {
		layout := components.NewLayoutComponent()
		
		// Set up components
		layout.SetMain(components.NewProcessViewComponent(nil))
		layout.SetHeader(components.NewNotificationComponent(nil))
		layout.SetFooter(components.NewCommandInputComponent())
		
		// Test different layout modes
		layout.SetMode(components.LayoutModeFull)
		layout.SetMode(components.LayoutModeHeaderFooter)
		layout.SetMode(components.LayoutModeComplete)
		
		// Render should not panic
		view := layout.View()
		if view == "" {
			// Empty view is OK for testing
		}
	})
}