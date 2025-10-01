package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHeaderComponent_NewComponent(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(80, 1)

	view := component.View()
	if !strings.Contains(view, "Brummer") {
		t.Error("View should contain title 'Brummer'")
	}
}

func TestHeaderComponent_MCPStatus(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(80, 1)

	// Inactive by default
	view := component.View()
	if !strings.Contains(view, "MCP") {
		t.Error("View should contain 'MCP' label")
	}

	// Set active
	component.SetMCPStatus(true, "port 7777")
	view = component.View()
	if !strings.Contains(view, "MCP") {
		t.Error("View should contain 'MCP' when active")
	}
	if !strings.Contains(view, "port 7777") {
		t.Error("View should contain MCP details")
	}
}

func TestHeaderComponent_ProxyStatus(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(80, 1)

	component.SetProxyStatus(true, "localhost:8080")

	view := component.View()
	if !strings.Contains(view, "Proxy") {
		t.Error("View should contain 'Proxy' label")
	}
	if !strings.Contains(view, "localhost:8080") {
		t.Error("View should contain proxy details")
	}
}

func TestHeaderComponent_AICodersStatus(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(80, 1)

	component.SetAICodersStatus(true, "3 active")

	view := component.View()
	if !strings.Contains(view, "AI Coders") {
		t.Error("View should contain 'AI Coders' label")
	}
	if !strings.Contains(view, "3 active") {
		t.Error("View should contain AI coders details")
	}
}

func TestHeaderComponent_ProcessCount(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(80, 1)

	// No processes initially
	view := component.View()
	if strings.Contains(view, "⚙️") {
		t.Error("View should not show process count when zero")
	}

	// Set process count
	component.SetProcessCount(5)
	view = component.View()
	if !strings.Contains(view, "5") {
		t.Error("View should show process count '5'")
	}
}

func TestHeaderComponent_AllStatusIndicators(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(200, 1) // Wide viewport

	component.SetMCPStatus(true, "7777")
	component.SetProxyStatus(true, "8080")
	component.SetAICodersStatus(true, "2")
	component.SetProcessCount(3)

	view := component.View()

	// Check all indicators present
	if !strings.Contains(view, "MCP") {
		t.Error("View should contain MCP status")
	}
	if !strings.Contains(view, "Proxy") {
		t.Error("View should contain Proxy status")
	}
	if !strings.Contains(view, "AI Coders") {
		t.Error("View should contain AI Coders status")
	}
	if !strings.Contains(view, "3") {
		t.Error("View should contain process count")
	}
}

func TestHeaderComponent_StatusIcons(t *testing.T) {
	component := NewHeaderComponent("Brummer")
	component.SetSize(80, 1)

	// Inactive should have ○ icon
	view := component.View()
	if !strings.Contains(view, "○") {
		t.Error("View should contain inactive icon '○'")
	}

	// Active should have ● icon
	component.SetMCPStatus(true, "")
	view = component.View()
	if !strings.Contains(view, "●") {
		t.Error("View should contain active icon '●'")
	}
}

func TestHeaderComponent_SetTitle(t *testing.T) {
	component := NewHeaderComponent("Original Title")
	component.SetSize(80, 1)

	view := component.View()
	if !strings.Contains(view, "Original Title") {
		t.Error("View should contain original title")
	}

	component.SetTitle("New Title")
	view = component.View()
	if !strings.Contains(view, "New Title") {
		t.Error("View should contain new title")
	}
	if strings.Contains(view, "Original Title") {
		t.Error("View should not contain old title")
	}
}

func TestHeaderComponent_SetSize(t *testing.T) {
	component := NewHeaderComponent("Brummer")

	component.SetSize(100, 1)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 1 {
		t.Errorf("Expected height 1, got %d", component.Height())
	}
}

func TestHeaderComponent_Visibility(t *testing.T) {
	component := NewHeaderComponent("Brummer")
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

func TestHeaderComponent_WindowSizeMsg(t *testing.T) {
	component := NewHeaderComponent("Brummer")
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
