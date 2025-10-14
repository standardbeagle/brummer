package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTabsComponent_NewComponent(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	if len(component.tabs) != 8 {
		t.Errorf("Expected 8 tabs, got %d", len(component.tabs))
	}

	view := component.View()
	if !strings.Contains(view, "Logs") {
		t.Error("View should contain 'Logs' tab")
	}
}

func TestTabsComponent_SetActiveTab(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	component.SetActiveTab("processes")

	if component.GetActiveTab() != "processes" {
		t.Errorf("Expected active tab 'processes', got '%s'", component.GetActiveTab())
	}
	if component.activeIndex != 1 {
		t.Errorf("Expected activeIndex 1, got %d", component.activeIndex)
	}
}

func TestTabsComponent_SetActiveTabByIndex(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	component.SetActiveTabByIndex(2)

	if component.GetActiveTab() != "urls" {
		t.Errorf("Expected active tab 'urls', got '%s'", component.GetActiveTab())
	}
	if component.activeIndex != 2 {
		t.Errorf("Expected activeIndex 2, got %d", component.activeIndex)
	}
}

func TestTabsComponent_TabBadges(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	component.SetTabBadge("errors", "5")

	view := component.View()
	if !strings.Contains(view, "[5]") {
		t.Error("View should contain badge '[5]'")
	}

	component.ClearTabBadge("errors")
	view = component.View()
	if strings.Contains(view, "[5]") {
		t.Error("Badge should be cleared from view")
	}
}

func TestTabsComponent_ShowShortcuts(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	// Shortcuts shown by default
	view := component.View()
	if !strings.Contains(view, "(1)") {
		t.Error("View should contain shortcut '(1)'")
	}

	// Hide shortcuts
	component.SetShowShortcuts(false)
	view = component.View()
	if strings.Contains(view, "(1)") {
		t.Error("View should not contain shortcuts when hidden")
	}
}

func TestTabsComponent_EnableDisableTabs(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	// All tabs enabled by default
	view := component.View()
	if !strings.Contains(view, "AI Coders") {
		t.Error("View should contain 'AI Coders' when enabled")
	}

	// Disable AI Coders tab
	component.DisableTab("aicoders")
	view = component.View()
	if strings.Contains(view, "AI Coders") {
		t.Error("View should not contain 'AI Coders' when disabled")
	}

	// Re-enable
	component.EnableTab("aicoders")
	view = component.View()
	if !strings.Contains(view, "AI Coders") {
		t.Error("View should contain 'AI Coders' when re-enabled")
	}
}

func TestTabsComponent_AllTabsPresent(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(200, 1) // Wide viewport

	view := component.View()

	expectedTabs := []string{"Logs", "Processes", "URLs", "Errors", "Web", "AI Coders", "MCP", "Settings"}
	for _, tab := range expectedTabs {
		if !strings.Contains(view, tab) {
			t.Errorf("View should contain tab '%s'", tab)
		}
	}
}

func TestTabsComponent_Icons(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(200, 1)

	view := component.View()

	// Check for presence of icons (emojis)
	expectedIcons := []string{"📋", "⚙️", "🔗", "⚠️", "🌐", "🤖", "🔌"}
	for _, icon := range expectedIcons {
		if !strings.Contains(view, icon) {
			t.Errorf("View should contain icon '%s'", icon)
		}
	}
}

func TestTabsComponent_SetSize(t *testing.T) {
	component := NewTabsComponent()

	component.SetSize(100, 1)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 1 {
		t.Errorf("Expected height 1, got %d", component.Height())
	}
}

func TestTabsComponent_Visibility(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

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

func TestTabsComponent_WindowSizeMsg(t *testing.T) {
	component := NewTabsComponent()
	component.SetSize(80, 1)

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 1}
	component.Update(msg)

	if component.Width() != 120 {
		t.Errorf("Expected width 120 after resize, got %d", component.Width())
	}

	// Verify view still renders
	view := component.View()
	if view == "" {
		t.Error("View should not be empty after resize")
	}
}
