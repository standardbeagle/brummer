package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAICodersViewComponent_NewComponent(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	if len(component.sessions) != 0 {
		t.Errorf("Expected 0 initial sessions, got %d", len(component.sessions))
	}

	view := component.View()
	if !strings.Contains(view, "No active AI coder sessions") {
		t.Error("View should show 'No active AI coder sessions' when empty")
	}
}

func TestAICodersViewComponent_AddSession(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	session := &SessionInfo{
		ID:            "session-123",
		Name:          "test-session",
		Status:        "active",
		ProcessID:     "proc-456",
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		LinesExchanged: 42,
		CurrentTask:   "Implementing feature X",
	}

	component.AddSession(session)

	if len(component.sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(component.sessions))
	}

	view := component.View()
	if !strings.Contains(view, "test-session") {
		t.Error("View should contain session name")
	}
	if !strings.Contains(view, "active") {
		t.Error("View should contain session status")
	}
}

func TestAICodersViewComponent_MultipleSession(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	statuses := []string{"active", "idle", "error"}
	for i, status := range statuses {
		session := &SessionInfo{
			ID:            string(rune('a' + i)),
			Name:          status + "-session",
			Status:        status,
			StartTime:     time.Now(),
			LastActivity:  time.Now(),
			LinesExchanged: i * 10,
		}
		component.AddSession(session)
	}

	if len(component.sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(component.sessions))
	}

	view := component.View()
	for _, status := range statuses {
		if !strings.Contains(view, status) {
			t.Errorf("View should contain status '%s'", status)
		}
	}
}

func TestAICodersViewComponent_RemoveSession(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	session1 := &SessionInfo{ID: "session-1", Name: "Session 1", Status: "active", StartTime: time.Now(), LastActivity: time.Now()}
	session2 := &SessionInfo{ID: "session-2", Name: "Session 2", Status: "idle", StartTime: time.Now(), LastActivity: time.Now()}

	component.AddSession(session1)
	component.AddSession(session2)

	component.RemoveSession("session-1")

	if len(component.sessions) != 1 {
		t.Errorf("Expected 1 session after removal, got %d", len(component.sessions))
	}
	if component.sessions[0].ID != "session-2" {
		t.Error("Wrong session was removed")
	}
}

func TestAICodersViewComponent_UpdateSession(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	session := &SessionInfo{
		ID:            "session-123",
		Name:          "test-session",
		Status:        "idle",
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		LinesExchanged: 10,
	}

	component.AddSession(session)

	component.UpdateSession("session-123", func(s *SessionInfo) {
		s.Status = "active"
		s.LinesExchanged = 50
	})

	if component.sessions[0].Status != "active" {
		t.Error("Session status should be updated to 'active'")
	}
	if component.sessions[0].LinesExchanged != 50 {
		t.Errorf("LinesExchanged should be 50, got %d", component.sessions[0].LinesExchanged)
	}
}

func TestAICodersViewComponent_KeyboardNavigation(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	// Add 3 sessions
	for i := 0; i < 3; i++ {
		session := &SessionInfo{
			ID:           string(rune('a' + i)),
			Name:         "session-" + string(rune('0' + i)),
			Status:       "active",
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		}
		component.AddSession(session)
	}

	// Initial selection is 0
	if component.selectedIndex != 0 {
		t.Errorf("Expected initial selection 0, got %d", component.selectedIndex)
	}

	// Press 'j' to move down
	downKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	component.Update(downKey)

	if component.selectedIndex != 1 {
		t.Errorf("Expected selection 1 after 'j', got %d", component.selectedIndex)
	}

	// Press 'k' to move up
	upKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	component.Update(upKey)

	if component.selectedIndex != 0 {
		t.Errorf("Expected selection 0 after 'k', got %d", component.selectedIndex)
	}

	// Wrap around test - press 'k' at index 0
	component.Update(upKey)

	if component.selectedIndex != 2 {
		t.Errorf("Expected selection 2 after wrap-around, got %d", component.selectedIndex)
	}
}

func TestAICodersViewComponent_GetSelectedSession(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 24)

	// No sessions - should return nil
	if component.GetSelectedSession() != nil {
		t.Error("GetSelectedSession should return nil when no sessions")
	}

	// Add sessions
	session1 := &SessionInfo{ID: "session-1", Name: "Session 1", Status: "active", StartTime: time.Now(), LastActivity: time.Now()}
	session2 := &SessionInfo{ID: "session-2", Name: "Session 2", Status: "idle", StartTime: time.Now(), LastActivity: time.Now()}

	component.AddSession(session1)
	component.AddSession(session2)

	// Should return first session
	selected := component.GetSelectedSession()
	if selected == nil || selected.ID != "session-1" {
		t.Error("GetSelectedSession should return first session")
	}

	// Navigate and check
	downKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	component.Update(downKey)

	selected = component.GetSelectedSession()
	if selected == nil || selected.ID != "session-2" {
		t.Error("GetSelectedSession should return second session after navigation")
	}
}

func TestAICodersViewComponent_StatusDisplay(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 100) // Tall viewport

	// Test different statuses
	statuses := []string{"active", "idle", "error", "stopped"}
	for i, status := range statuses {
		session := &SessionInfo{
			ID:            string(rune('a' + i)),
			Name:          status + "-session",
			Status:        status,
			StartTime:     time.Now(),
			LastActivity:  time.Now(),
			LinesExchanged: i * 10,
		}
		component.AddSession(session)
	}

	view := component.View()

	// Verify all statuses are displayed
	for _, status := range statuses {
		if !strings.Contains(view, status) {
			t.Errorf("View should contain status '%s'", status)
		}
	}
}

func TestAICodersViewComponent_SetSize(t *testing.T) {
	component := NewAICodersViewComponent()

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

func TestAICodersViewComponent_Visibility(t *testing.T) {
	component := NewAICodersViewComponent()
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

func TestAICodersViewComponent_WindowSizeMsg(t *testing.T) {
	component := NewAICodersViewComponent()
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

func TestAICodersViewComponent_CurrentTaskDisplay(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 100) // Tall viewport

	session := &SessionInfo{
		ID:            "session-123",
		Name:          "test-session",
		Status:        "active",
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		LinesExchanged: 42,
		CurrentTask:   "Implementing authentication system with JWT tokens",
	}

	component.AddSession(session)

	view := component.View()
	if !strings.Contains(view, "Task:") {
		t.Error("View should contain task label")
	}
	if !strings.Contains(view, "Implementing authentication") {
		t.Error("View should contain task description")
	}
}

func TestAICodersViewComponent_ErrorMessageDisplay(t *testing.T) {
	component := NewAICodersViewComponent()
	component.SetSize(80, 100) // Tall viewport

	session := &SessionInfo{
		ID:            "session-123",
		Name:          "test-session",
		Status:        "error",
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		LinesExchanged: 10,
		ErrorMessage:  "Connection timeout - failed to reach AI service",
	}

	component.AddSession(session)

	view := component.View()
	if !strings.Contains(view, "Error:") {
		t.Error("View should contain error label")
	}
	if !strings.Contains(view, "Connection timeout") {
		t.Error("View should contain error message")
	}
}