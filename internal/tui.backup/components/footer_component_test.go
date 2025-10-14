package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFooterComponent_NewComponent(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(80, 1)

	view := component.View()
	if !strings.Contains(view, "Views") {
		t.Error("View should contain 'Views' shortcut")
	}
	if !strings.Contains(view, "Help") {
		t.Error("View should contain 'Help' shortcut")
	}
	if !strings.Contains(view, "Quit") {
		t.Error("View should contain 'Quit' shortcut")
	}
}

func TestFooterComponent_GlobalShortcuts(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(80, 1)

	view := component.View()
	if !strings.Contains(view, "1-8") {
		t.Error("View should contain '1-8' key")
	}
	if !strings.Contains(view, "q") {
		t.Error("View should contain 'q' key")
	}
	if !strings.Contains(view, "?") {
		t.Error("View should contain '?' key")
	}
}

func TestFooterComponent_ContextShortcuts(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(80, 1)

	// No context shortcuts initially
	view := component.View()
	if strings.Contains(view, "Scroll") {
		t.Error("View should not contain context shortcuts initially")
	}

	// Add context shortcuts
	component.SetContextShortcuts(GetDefaultLogsShortcuts())
	view = component.View()
	if !strings.Contains(view, "Scroll") {
		t.Error("View should contain 'Scroll' after setting logs shortcuts")
	}
	if !strings.Contains(view, "Filter") {
		t.Error("View should contain 'Filter' shortcut")
	}
}

func TestFooterComponent_AddContextShortcut(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(80, 1)

	component.AddContextShortcut("x", "Export")

	view := component.View()
	if !strings.Contains(view, "x") {
		t.Error("View should contain 'x' key")
	}
	if !strings.Contains(view, "Export") {
		t.Error("View should contain 'Export' label")
	}
}

func TestFooterComponent_ClearContextShortcuts(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(80, 1)

	component.SetContextShortcuts(GetDefaultLogsShortcuts())
	view := component.View()
	if !strings.Contains(view, "Scroll") {
		t.Error("View should contain context shortcuts")
	}

	component.ClearContextShortcuts()
	view = component.View()
	if strings.Contains(view, "Scroll") {
		t.Error("View should not contain context shortcuts after clear")
	}
}

func TestFooterComponent_SetShowHelp(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(80, 1)

	// Help shown by default
	view := component.View()
	if !strings.Contains(view, "Press ? for help") {
		t.Error("View should show help hint by default")
	}

	// Hide help
	component.SetShowHelp(false)
	view = component.View()
	if strings.Contains(view, "Press ? for help") {
		t.Error("View should not show help hint when disabled")
	}
}

func TestFooterComponent_ProcessesShortcuts(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(200, 1) // Wide viewport

	component.SetContextShortcuts(GetDefaultProcessesShortcuts())

	view := component.View()
	if !strings.Contains(view, "Select") {
		t.Error("View should contain 'Select' for processes view")
	}
	if !strings.Contains(view, "Restart") {
		t.Error("View should contain 'Restart' shortcut")
	}
	if !strings.Contains(view, "Stop") {
		t.Error("View should contain 'Stop' shortcut")
	}
}

func TestFooterComponent_ErrorsShortcuts(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(200, 1)

	component.SetContextShortcuts(GetDefaultErrorsShortcuts())

	view := component.View()
	if !strings.Contains(view, "j/k") {
		t.Error("View should contain 'j/k' navigation keys")
	}
	if !strings.Contains(view, "Navigate") {
		t.Error("View should contain 'Navigate' label")
	}
}

func TestFooterComponent_AICodersShortcuts(t *testing.T) {
	component := NewFooterComponent()
	component.SetSize(200, 1)

	component.SetContextShortcuts(GetDefaultAICodersShortcuts())

	view := component.View()
	if !strings.Contains(view, "Attach") {
		t.Error("View should contain 'Attach' for AI coders view")
	}
	if !strings.Contains(view, "New") {
		t.Error("View should contain 'New' shortcut")
	}
	if !strings.Contains(view, "Detach") {
		t.Error("View should contain 'Detach' shortcut")
	}
}

func TestFooterComponent_SetSize(t *testing.T) {
	component := NewFooterComponent()

	component.SetSize(100, 1)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 1 {
		t.Errorf("Expected height 1, got %d", component.Height())
	}
}

func TestFooterComponent_Visibility(t *testing.T) {
	component := NewFooterComponent()
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

func TestFooterComponent_WindowSizeMsg(t *testing.T) {
	component := NewFooterComponent()
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
