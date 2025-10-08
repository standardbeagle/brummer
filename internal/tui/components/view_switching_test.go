package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// TestNavigateToViewMsgMessages tests that NavigateToViewMsg messages are handled correctly
func TestNavigateToViewMsgMessages(t *testing.T) {
	shell := NewIntegratedShell()

	// Test cases for each view
	testCases := []struct {
		name         string
		viewName     string
		expectedView string
		shouldChange bool
	}{
		{"Switch to logs", "logs", "logs", true},
		{"Switch to processes", "processes", "processes", true},
		{"Switch to URLs", "urls", "urls", true},
		{"Switch to errors", "errors", "errors", true},
		{"Switch to web", "web", "web", true},
		{"Switch to AI coders", "aicoders", "aicoders", true},
		{"Switch to MCP", "mcp", "mcp", true},
		{"Switch to settings", "settings", "settings", true},
		{"Switch to diagnostic", "diagnostic", "diagnostic", true},
		{"Invalid view", "invalid", "logs", false}, // Should stay on current view
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset to logs view initially
			shell.currentView = "logs"

			// Send NavigateToViewMsg
			msg := messages.NavigateToViewMsg{View: tc.viewName}
			model, _ := shell.Update(msg)

			// Check that we got the same model back
			if model != shell {
				t.Errorf("Expected shell to be returned, got different model")
			}

			// Check view state
			if tc.shouldChange {
				if shell.currentView != tc.expectedView {
					t.Errorf("Expected view to be '%s', got '%s'", tc.expectedView, shell.currentView)
				}
			} else {
				if shell.currentView != "logs" {
					t.Errorf("Expected view to stay 'logs' for invalid view, got '%s'", shell.currentView)
				}
			}
		})
	}
}

// TestArrowKeyViewSwitching tests that arrow keys work for view switching
func TestArrowKeyViewSwitching(t *testing.T) {
	shell := NewIntegratedShell()

	// Test cases for arrow key switching
	testCases := []struct {
		name         string
		keyPress     string
		initialView  string
		expectedView string
	}{
		{"Press 1 from any view", "1", "processes", "logs"},
		{"Press 2 from logs", "2", "logs", "processes"},
		{"Press 3 from processes", "3", "processes", "urls"},
		{"Press 6 from logs", "6", "logs", "aicoders"},
		{"Press 7 from any view", "7", "urls", "mcp"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set initial view
			shell.currentView = tc.initialView

			// Create key message
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(tc.keyPress[0])}}
			model, _ := shell.Update(msg)

			// Check that we got the same model back
			if model != shell {
				t.Errorf("Expected shell to be returned, got different model")
			}

			// Check view switched correctly
			if shell.currentView != tc.expectedView {
				t.Errorf("Expected view to be '%s' after pressing %s, got '%s'", tc.expectedView, tc.keyPress, shell.currentView)
			}
		})
	}
}

// TestCommandExecutorViewSwitching tests that the command executor sends the right messages
func TestCommandExecutorViewSwitching(t *testing.T) {
	// Create a command executor with nil AI coder manager (like current implementation)
	executor := NewCommandExecutor(nil, nil, nil, NewCommandRegistry(), nil)

	// Test AI command
	msg := messages.CommandExecuteMsg{
		CommandName: "ai",
		Arguments:   []string{"claude"},
	}

	// Execute the command
	cmd := executor.Execute(msg)

	// The command should return a function that returns NavigateToViewMsg
	if cmd == nil {
		t.Fatal("Expected command to return a function, got nil")
	}

	// Execute the returned command
	resultMsg := cmd()

	// Should be a NavigateToViewMsg
	navigateMsg, ok := resultMsg.(messages.NavigateToViewMsg)
	if !ok {
		t.Errorf("Expected NavigateToViewMsg, got %T", resultMsg)
	} else if navigateMsg.View != "aicoders" {
		t.Errorf("Expected view 'aicoders', got '%s'", navigateMsg.View)
	}
}

// TestIntegratedShellViewRendering tests that the current view is rendered correctly
func TestIntegratedShellViewRendering(t *testing.T) {
	shell := NewIntegratedShell()

	// Set up mock components
	shell.SetLogsView(NewMockComponent("Logs View Content"))
	shell.SetProcessView(NewMockComponent("Processes View Content"))
	shell.SetAICodersView(NewMockComponent("AI Coders View Content"))

	// Test rendering each view
	testCases := []struct {
		name     string
		view     string
		expected string
	}{
		{"Render logs view", "logs", "Logs View Content"},
		{"Render processes view", "processes", "Processes View Content"},
		{"Render AI coders view", "aicoders", "AI Coders View Content"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shell.currentView = tc.view
			result := shell.View()

			if result == "" {
				t.Errorf("Expected view content, got empty string")
			}

			// The result should contain the expected content somewhere in the layout
			// Note: This is a basic check - the actual layout includes headers/tabs/etc.
			if tc.expected != "" && !contains(result, tc.expected) {
				t.Errorf("Expected view content to contain '%s', got: %s", tc.expected, result)
			}
		})
	}
}

// MockComponent for testing
type MockComponent struct {
	content string
	BaseComponent
}

func NewMockComponent(content string) *MockComponent {
	return &MockComponent{
		content:       content,
		BaseComponent: NewBaseComponent(),
	}
}

func (m *MockComponent) Init() tea.Cmd {
	return nil
}

func (m *MockComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *MockComponent) View() string {
	if !m.IsVisible() {
		return ""
	}
	return m.content
}

func (m *MockComponent) SetSize(width, height int) {
	m.BaseComponent.SetSize(width, height)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
