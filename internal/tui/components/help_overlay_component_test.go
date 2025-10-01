package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpOverlayComponent_NewComponent(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	if len(component.helpSections) == 0 {
		t.Error("Expected default help sections to be loaded")
	}

	// Should not be active by default
	if component.IsActive() {
		t.Error("Help overlay should not be active by default")
	}

	view := component.View()
	if view != "" {
		t.Error("View should be empty when inactive")
	}
}

func TestHelpOverlayComponent_OpenClose(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	// Open help
	component.Open()
	if !component.IsActive() {
		t.Error("Help overlay should be active after Open()")
	}

	view := component.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("View should contain title when active")
	}

	// Close help
	component.Close()
	if component.IsActive() {
		t.Error("Help overlay should not be active after Close()")
	}

	view = component.View()
	if view != "" {
		t.Error("View should be empty after close")
	}
}

func TestHelpOverlayComponent_DefaultSections(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	// Check for expected sections
	found := make(map[string]bool)
	for _, section := range component.helpSections {
		found[section.Title] = true
	}

	expectedSections := []string{
		"Navigation",
		"Global Commands",
		"Logs View",
		"Processes View",
		"Errors View",
		"AI Coders View",
	}

	for _, title := range expectedSections {
		if !found[title] {
			t.Errorf("Expected help section '%s' to exist", title)
		}
	}
}

func TestHelpOverlayComponent_ShortcutContent(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(200, 50) // Large viewport
	component.Open()

	view := component.View()

	// Check for specific shortcuts
	expectedShortcuts := []string{
		"1-8",
		"Switch between views",
		"Ctrl+P",
		"q",
		"Quit",
	}

	for _, text := range expectedShortcuts {
		if !strings.Contains(view, text) {
			t.Errorf("View should contain '%s'", text)
		}
	}
}

func TestHelpOverlayComponent_EscapeKey(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)
	component.Open()

	if !component.IsActive() {
		t.Error("Help overlay should be active after Open()")
	}

	// Simulate Esc key
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	component.Update(escKey)

	if component.IsActive() {
		t.Error("Help overlay should close on Esc key")
	}
}

func TestHelpOverlayComponent_QuestionMarkKey(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)
	component.Open()

	// Simulate ? key
	qKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	component.Update(qKey)

	if component.IsActive() {
		t.Error("Help overlay should close on ? key")
	}
}

func TestHelpOverlayComponent_AddSection(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	initialCount := len(component.helpSections)

	customSection := HelpSection{
		Title: "Custom Section",
		Shortcuts: []HelpShortcut{
			{Key: "x", Description: "Custom action"},
		},
	}

	component.AddSection(customSection)

	if len(component.helpSections) != initialCount+1 {
		t.Errorf("Expected %d sections after adding, got %d", initialCount+1, len(component.helpSections))
	}

	// Check section was added
	found := false
	for _, section := range component.helpSections {
		if section.Title == "Custom Section" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom section was not added")
	}
}

func TestHelpOverlayComponent_AddShortcut(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	// Find initial shortcut count for Navigation section
	var initialCount int
	for _, section := range component.helpSections {
		if section.Title == "Navigation" {
			initialCount = len(section.Shortcuts)
			break
		}
	}

	// Add shortcut to existing section
	component.AddShortcut("Navigation", HelpShortcut{
		Key:         "n",
		Description: "New navigation action",
	})

	// Check shortcut was added
	var newCount int
	for _, section := range component.helpSections {
		if section.Title == "Navigation" {
			newCount = len(section.Shortcuts)
			break
		}
	}

	if newCount != initialCount+1 {
		t.Errorf("Expected %d shortcuts in Navigation, got %d", initialCount+1, newCount)
	}
}

func TestHelpOverlayComponent_AddShortcutNewSection(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	initialSectionCount := len(component.helpSections)

	// Add shortcut to non-existent section (should create it)
	component.AddShortcut("New Section", HelpShortcut{
		Key:         "z",
		Description: "New section action",
	})

	if len(component.helpSections) != initialSectionCount+1 {
		t.Errorf("Expected %d sections after adding to new section, got %d", initialSectionCount+1, len(component.helpSections))
	}

	// Check new section exists
	found := false
	for _, section := range component.helpSections {
		if section.Title == "New Section" {
			found = true
			if len(section.Shortcuts) != 1 {
				t.Error("New section should have 1 shortcut")
			}
			break
		}
	}
	if !found {
		t.Error("New section was not created")
	}
}

func TestHelpOverlayComponent_SetCurrentView(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	component.SetCurrentView("processes")

	if component.currentView != "processes" {
		t.Errorf("Expected currentView 'processes', got '%s'", component.currentView)
	}
}

func TestHelpOverlayComponent_GetShortcutSummary(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	summary := component.GetShortcutSummary()

	if summary == "" {
		t.Error("Summary should not be empty")
	}

	// Check summary contains key info
	if !strings.Contains(summary, "Help") {
		t.Error("Summary should mention Help")
	}
	if !strings.Contains(summary, "Quit") {
		t.Error("Summary should mention Quit")
	}
}

func TestHelpOverlayComponent_SetSize(t *testing.T) {
	component := NewHelpOverlayComponent()

	component.SetSize(100, 30)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 30 {
		t.Errorf("Expected height 30, got %d", component.Height())
	}
}

func TestHelpOverlayComponent_Visibility(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)

	// Default is visible
	if !component.IsVisible() {
		t.Error("Component should be visible by default")
	}

	component.Open()
	view := component.View()
	if view == "" {
		t.Error("View should not be empty when active")
	}

	// Test hiding
	component.SetVisible(false)
	if component.IsVisible() {
		t.Error("Component should be hidden")
	}

	// View should be empty when hidden
	view = component.View()
	if view != "" {
		t.Error("Hidden component should return empty view")
	}
}

func TestHelpOverlayComponent_WindowSizeMsg(t *testing.T) {
	component := NewHelpOverlayComponent()
	component.SetSize(80, 24)
	component.Open()

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	component.Update(msg)

	if component.Width() != 120 {
		t.Errorf("Expected width 120 after resize, got %d", component.Width())
	}
	if component.Height() != 40 {
		t.Errorf("Expected height 40 after resize, got %d", component.Height())
	}
}
