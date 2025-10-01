package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMCPViewComponent_NewComponent(t *testing.T) {
	component := NewMCPViewComponent()
	component.SetSize(80, 24)

	if len(component.status.AvailableTools) == 0 {
		t.Error("Expected default tools to be loaded")
	}

	view := component.View()
	if !strings.Contains(view, "MCP Server Status") {
		t.Error("View should contain 'MCP Server Status' header")
	}
}

func TestMCPViewComponent_StatusDisplay(t *testing.T) {
	component := NewMCPViewComponent()
	component.SetSize(80, 24)

	view := component.View()

	// Verify status fields are displayed
	if !strings.Contains(view, "Status:") {
		t.Error("View should contain 'Status:' label")
	}
	if !strings.Contains(view, "Address:") {
		t.Error("View should contain 'Address:' label")
	}
	if !strings.Contains(view, "Port:") {
		t.Error("View should contain 'Port:' label")
	}

	// Default status is inactive
	if !strings.Contains(view, "Inactive") {
		t.Error("View should show 'Inactive' status by default")
	}
}

func TestMCPViewComponent_ActiveStatus(t *testing.T) {
	component := NewMCPViewComponent()
	component.SetSize(80, 24)

	// Set active status
	component.status.Active = true
	component.updateViewport()

	view := component.View()
	if !strings.Contains(view, "Active") {
		t.Error("View should show 'Active' status when enabled")
	}
}

func TestMCPViewComponent_StatisticsDisplay(t *testing.T) {
	component := NewMCPViewComponent()
	component.SetSize(80, 24)

	view := component.View()

	// Verify statistics section
	if !strings.Contains(view, "Statistics") {
		t.Error("View should contain 'Statistics' header")
	}
	if !strings.Contains(view, "Available Tools:") {
		t.Error("View should show available tools count")
	}
	if !strings.Contains(view, "Total Requests:") {
		t.Error("View should show total requests count")
	}
	if !strings.Contains(view, "Active Sessions:") {
		t.Error("View should show active sessions count")
	}
}

func TestMCPViewComponent_ToolsListing(t *testing.T) {
	component := NewMCPViewComponent()
	component.SetSize(80, 100) // Tall viewport to see all content

	view := component.View()

	// Verify tool categories
	expectedCategories := []string{"Scripts", "Logs", "Proxy", "Telemetry", "Browser", "REPL"}
	for _, category := range expectedCategories {
		if !strings.Contains(view, category) {
			t.Errorf("View should contain category '%s'", category)
		}
	}

	// Verify specific tools
	expectedTools := []string{"scripts/list", "logs/stream", "proxy/requests", "browser/open"}
	for _, tool := range expectedTools {
		if !strings.Contains(view, tool) {
			t.Errorf("View should contain tool '%s'", tool)
		}
	}
}

func TestMCPViewComponent_SetSize(t *testing.T) {
	component := NewMCPViewComponent()

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

func TestMCPViewComponent_Visibility(t *testing.T) {
	component := NewMCPViewComponent()
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

func TestMCPViewComponent_KeyboardScrolling(t *testing.T) {
	component := NewMCPViewComponent()
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

func TestMCPViewComponent_WindowSizeMsg(t *testing.T) {
	component := NewMCPViewComponent()
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

func TestMCPViewComponent_ToolCategorization(t *testing.T) {
	component := NewMCPViewComponent()
	component.SetSize(80, 24)

	// Verify tools are properly categorized
	component.status.AvailableTools = []string{
		"scripts/run",
		"logs/stream",
		"browser/open",
	}
	component.updateViewport()

	view := component.View()

	// All categories should appear even with limited tools
	if !strings.Contains(view, "Scripts") {
		t.Error("View should contain Scripts category")
	}
	if !strings.Contains(view, "Logs") {
		t.Error("View should contain Logs category")
	}
	if !strings.Contains(view, "Browser") {
		t.Error("View should contain Browser category")
	}
}