package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsViewComponent_NewComponent(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 24)

	if len(component.settings) == 0 {
		t.Error("Expected default settings to be loaded")
	}

	// Verify expected categories exist
	categories := make(map[string]bool)
	for _, setting := range component.settings {
		categories[setting.Category] = true
	}

	expectedCategories := []string{"MCP Server", "Proxy Server", "Process Manager", "Logs", "UI"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("Expected category '%s' to be present", cat)
		}
	}

	view := component.View()
	if !strings.Contains(view, "Configuration Settings") {
		t.Error("View should contain 'Configuration Settings' header")
	}
}

func TestSettingsViewComponent_CategoryGrouping(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 24)

	view := component.View()

	// Verify categories are displayed
	if !strings.Contains(view, "MCP Server") {
		t.Error("View should contain 'MCP Server' category")
	}
	if !strings.Contains(view, "Proxy Server") {
		t.Error("View should contain 'Proxy Server' category")
	}
	if !strings.Contains(view, "Process Manager") {
		t.Error("View should contain 'Process Manager' category")
	}
	if !strings.Contains(view, "Logs") {
		t.Error("View should contain 'Logs' category")
	}
	if !strings.Contains(view, "UI") {
		t.Error("View should contain 'UI' category")
	}
}

func TestSettingsViewComponent_SettingDisplay(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 24)

	view := component.View()

	// Verify specific settings are displayed
	expectedSettings := []string{
		"mcp.port",
		"mcp.enabled",
		"proxy.enabled",
		"proxy.port",
		"processes.max",
		"logs.max_entries",
		"ui.refresh_rate",
	}

	for _, setting := range expectedSettings {
		if !strings.Contains(view, setting) {
			t.Errorf("View should contain setting '%s'", setting)
		}
	}

	// Verify read-only notice
	if !strings.Contains(view, "read-only") {
		t.Error("View should contain read-only notice")
	}
}

func TestSettingsViewComponent_SetSize(t *testing.T) {
	component := NewSettingsViewComponent()

	component.SetSize(100, 50)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 50 {
		t.Errorf("Expected height 50, got %d", component.Height())
	}
	if component.viewport.Width != 100 {
		t.Errorf("Expected viewport width 100, got %d", component.viewport.Width)
	}
	if component.viewport.Height != 50 {
		t.Errorf("Expected viewport height 50, got %d", component.viewport.Height)
	}
}

func TestSettingsViewComponent_Visibility(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 24)

	// Default is visible
	if !component.IsVisible() {
		t.Error("Component should be visible by default")
	}

	// Test hiding
	component.SetVisible(false)
	if component.IsVisible() {
		t.Error("Component should be hidden")
	}

	// View should be empty when hidden
	view := component.View()
	if view != "" {
		t.Error("Hidden component should return empty view")
	}
}

func TestSettingsViewComponent_KeyboardScrolling(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 10) // Small height to require scrolling

	// Simulate down arrow key
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	component.Update(downKey)

	// Simulate up arrow key
	upKey := tea.KeyMsg{Type: tea.KeyUp}
	component.Update(upKey)

	// Should not crash and should maintain valid state
	view := component.View()
	if view == "" {
		t.Error("View should not be empty after scrolling")
	}
}

func TestSettingsViewComponent_WindowSizeMsg(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 24)

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 30}
	component.Update(msg)

	if component.Width() != 120 {
		t.Errorf("Expected width 120 after resize, got %d", component.Width())
	}
	if component.Height() != 30 {
		t.Errorf("Expected height 30 after resize, got %d", component.Height())
	}

	// Verify viewport content is regenerated
	view := component.View()
	if view == "" {
		t.Error("View should not be empty after resize")
	}
}

func TestSettingsViewComponent_DescriptionDisplay(t *testing.T) {
	component := NewSettingsViewComponent()
	component.SetSize(80, 24)

	view := component.View()

	// Verify descriptions are displayed
	if !strings.Contains(view, "Port for MCP server") {
		t.Error("View should contain MCP port description")
	}
	if !strings.Contains(view, "Maximum concurrent processes") {
		t.Error("View should contain process max description")
	}
}
