package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

func TestCommandInputComponent_NewComponent(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)

	if component == nil {
		t.Fatal("Expected component to be created")
	}

	if component.IsActive() {
		t.Error("Component should not be active initially")
	}
}

func TestCommandInputComponent_OpenClose(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)

	// Should start inactive
	if component.IsActive() {
		t.Error("Component should start inactive")
	}

	// Open should activate
	component.Open()
	if !component.IsActive() {
		t.Error("Component should be active after Open()")
	}

	// Close should deactivate
	component.Close()
	if component.IsActive() {
		t.Error("Component should be inactive after Close()")
	}
}

func TestCommandInputComponent_CommandParsing(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)

	tests := []struct {
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{
			input:    "/run dev",
			wantCmd:  "run",
			wantArgs: []string{"dev"},
		},
		{
			input:    "show error",
			wantCmd:  "show",
			wantArgs: []string{"error"},
		},
		{
			input:    "/ai claude test task",
			wantCmd:  "ai",
			wantArgs: []string{"claude", "test", "task"},
		},
		{
			input:    "/clear",
			wantCmd:  "clear",
			wantArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			msg := component.parseCommand(tt.input)

			if msg.CommandName != tt.wantCmd {
				t.Errorf("Expected command %q, got %q", tt.wantCmd, msg.CommandName)
			}

			if len(msg.Arguments) != len(tt.wantArgs) {
				t.Errorf("Expected %d arguments, got %d", len(tt.wantArgs), len(msg.Arguments))
			}

			for i, arg := range tt.wantArgs {
				if i < len(msg.Arguments) && msg.Arguments[i] != arg {
					t.Errorf("Argument %d: expected %q, got %q", i, arg, msg.Arguments[i])
				}
			}
		})
	}
}

func TestCommandInputComponent_CommandSuggestions(t *testing.T) {
	registry := NewCommandRegistry()

	// Register some test commands
	registry.Register(messages.Command{Name: "run", Description: "Run script"})
	registry.Register(messages.Command{Name: "restart", Description: "Restart script"})
	registry.Register(messages.Command{Name: "show", Description: "Show logs"})
	registry.Register(messages.Command{Name: "stop", Description: "Stop script"})

	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)
	component.Open()

	// Type "r" - should suggest "run" and "restart"
	component.input.SetValue("r")
	component.updateSuggestions()

	if !component.showSuggestions {
		t.Error("Should show suggestions for 'r'")
	}

	if len(component.suggestions) != 2 {
		t.Errorf("Expected 2 suggestions for 'r', got %d", len(component.suggestions))
	}

	// Type "ru" - should suggest only "run"
	component.input.SetValue("ru")
	component.updateSuggestions()

	if len(component.suggestions) != 1 {
		t.Errorf("Expected 1 suggestion for 'ru', got %d", len(component.suggestions))
	}

	if len(component.suggestions) > 0 && component.suggestions[0] != "run" {
		t.Errorf("Expected suggestion 'run', got %q", component.suggestions[0])
	}
}

func TestCommandInputComponent_ArgumentSuggestions(t *testing.T) {
	registry := NewCommandRegistry()

	// Register command with argument suggestions
	registry.Register(messages.Command{
		Name:        "run",
		Description: "Run script",
		Arguments: []messages.CommandArgument{
			{
				Name:        "script",
				Required:    true,
				Suggestions: []string{"dev", "test", "build"},
			},
		},
	})

	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)
	component.Open()

	// Type "run " - should show script suggestions
	component.input.SetValue("run ")
	component.updateSuggestions()

	if !component.showSuggestions {
		t.Error("Should show suggestions for 'run ' arguments")
	}

	if len(component.suggestions) != 3 {
		t.Errorf("Expected 3 script suggestions, got %d", len(component.suggestions))
	}

	// Type "run d" - should filter to "dev"
	component.input.SetValue("run d")
	component.updateSuggestions()

	if len(component.suggestions) != 1 {
		t.Errorf("Expected 1 suggestion for 'run d', got %d", len(component.suggestions))
	}

	if len(component.suggestions) > 0 && component.suggestions[0] != "dev" {
		t.Errorf("Expected suggestion 'dev', got %q", component.suggestions[0])
	}
}

func TestCommandInputComponent_TabCompletion(t *testing.T) {
	registry := NewCommandRegistry()
	registry.Register(messages.Command{Name: "run", Description: "Run script"})

	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)
	component.Open()

	// Type "r" and press Tab
	component.input.SetValue("r")
	component.updateSuggestions()

	if len(component.suggestions) == 0 {
		t.Fatal("Expected suggestions for 'r'")
	}

	component.applySuggestion()

	value := strings.TrimSpace(component.input.Value())
	if !strings.HasPrefix(value, "run") {
		t.Errorf("Expected value to start with 'run', got %q", value)
	}
}

func TestCommandInputComponent_EnterExecutes(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)
	component.Open()

	// Type command
	component.input.SetValue("run dev")

	// Press Enter
	_, cmd := component.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if component.IsActive() {
		t.Error("Component should close after Enter")
	}

	// Execute the command
	if cmd == nil {
		t.Fatal("Expected command to be returned")
	}

	msg := cmd()
	cmdMsg, ok := msg.(messages.CommandExecuteMsg)
	if !ok {
		t.Fatalf("Expected CommandExecuteMsg, got %T", msg)
	}

	if cmdMsg.CommandName != "run" {
		t.Errorf("Expected command 'run', got %q", cmdMsg.CommandName)
	}

	if len(cmdMsg.Arguments) != 1 || cmdMsg.Arguments[0] != "dev" {
		t.Errorf("Expected arguments ['dev'], got %v", cmdMsg.Arguments)
	}
}

func TestCommandInputComponent_EscCloses(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)
	component.Open()

	if !component.IsActive() {
		t.Fatal("Component should be active")
	}

	// Press Esc
	component.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if component.IsActive() {
		t.Error("Component should close after Esc")
	}
}

func TestCommandInputComponent_NavigateSuggestions(t *testing.T) {
	registry := NewCommandRegistry()
	registry.Register(messages.Command{Name: "run", Description: "Run"})
	registry.Register(messages.Command{Name: "restart", Description: "Restart"})
	registry.Register(messages.Command{Name: "stop", Description: "Stop"})

	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)
	component.Open()

	// Trigger suggestions
	component.input.SetValue("r")
	component.updateSuggestions()

	if len(component.suggestions) < 2 {
		t.Fatal("Expected multiple suggestions")
	}

	// Should start at 0
	if component.selectedSuggestion != 0 {
		t.Errorf("Expected selected suggestion 0, got %d", component.selectedSuggestion)
	}

	// Press Down
	component.Update(tea.KeyMsg{Type: tea.KeyDown})
	if component.selectedSuggestion != 1 {
		t.Errorf("Expected selected suggestion 1, got %d", component.selectedSuggestion)
	}

	// Press Up
	component.Update(tea.KeyMsg{Type: tea.KeyUp})
	if component.selectedSuggestion != 0 {
		t.Errorf("Expected selected suggestion 0, got %d", component.selectedSuggestion)
	}
}

func TestCommandInputComponent_View(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)
	component.SetSize(80, 24)

	// View should be empty when inactive
	view := component.View()
	if view != "" {
		t.Error("View should be empty when inactive")
	}

	// View should show input when active
	component.Open()
	view = component.View()
	if view == "" {
		t.Error("View should not be empty when active")
	}

	if !strings.Contains(view, "/") {
		t.Error("View should contain prompt '/'")
	}
}

func TestCommandInputComponent_SetSize(t *testing.T) {
	registry := NewCommandRegistry()
	component := NewCommandInputComponent(registry)

	component.SetSize(100, 50)

	if component.width != 100 {
		t.Errorf("Expected width 100, got %d", component.width)
	}

	if component.height != 50 {
		t.Errorf("Expected height 50, got %d", component.height)
	}
}
