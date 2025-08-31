package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// FakeSuggestionProvider is a test provider
type FakeSuggestionProvider struct {
	suggestions []CommandSuggestion
	callCount   int
	lastPrefix  string
}

func (f *FakeSuggestionProvider) Suggest(prefix string) []CommandSuggestion {
	f.callCount++
	f.lastPrefix = prefix
	
	// Filter based on prefix
	var filtered []CommandSuggestion
	for _, s := range f.suggestions {
		if strings.HasPrefix(strings.ToLower(s.Command), strings.ToLower(prefix)) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// FakeCommandExecutor is a test executor
type FakeCommandExecutor struct {
	executedCommands []string
	returnCmd        tea.Cmd
}

func (f *FakeCommandExecutor) Execute(command string) tea.Cmd {
	f.executedCommands = append(f.executedCommands, command)
	if f.returnCmd != nil {
		return f.returnCmd
	}
	return func() tea.Msg {
		return CommandExecuteMsg{Command: command}
	}
}

func TestCommandInputComponent_WithProvider(t *testing.T) {
	// Create fake provider with test suggestions
	provider := &FakeSuggestionProvider{
		suggestions: []CommandSuggestion{
			{Command: "/ai openai", Description: "Start AI with OpenAI", Category: "AI"},
			{Command: "/ai claude", Description: "Start AI with Claude", Category: "AI"},
			{Command: "/logs", Description: "View logs", Category: "Navigation"},
			{Command: "/processes", Description: "View processes", Category: "Navigation"},
		},
	}
	
	// Create component and set provider
	component := NewCommandInputComponent()
	component.SetProvider(provider)
	
	// Test 1: Provider is called on creation
	if provider.callCount == 0 {
		t.Error("Provider should be called when set")
	}
	
	// Test 2: Typing updates suggestions
	component.textInput.SetValue("/a")
	component.updateSuggestions()
	
	if provider.lastPrefix != "/a" {
		t.Errorf("Expected prefix '/a', got '%s'", provider.lastPrefix)
	}
	
	if len(component.suggestions) != 2 {
		t.Errorf("Expected 2 AI suggestions, got %d", len(component.suggestions))
	}
	
	// Test 3: Suggestions are filtered correctly
	for _, s := range component.suggestions {
		if !strings.HasPrefix(s.Command, "/ai") {
			t.Errorf("Expected all suggestions to start with '/ai', got '%s'", s.Command)
		}
	}
}

func TestCommandInputComponent_WithExecutor(t *testing.T) {
	// Create fake executor
	executor := &FakeCommandExecutor{
		returnCmd: func() tea.Msg {
			return CommandExecuteMsg{Command: "test"}
		},
	}
	
	// Create component and set executor
	component := NewCommandInputComponent()
	component.SetExecutor(executor)
	
	// Simulate entering a command
	component.textInput.SetValue("/start dev")
	
	// Simulate pressing enter
	_, cmd := component.Update(tea.KeyMsg{Type: tea.KeyEnter})
	
	// Verify executor was called
	if len(executor.executedCommands) != 1 {
		t.Errorf("Expected 1 command execution, got %d", len(executor.executedCommands))
	}
	
	if executor.executedCommands[0] != "/start dev" {
		t.Errorf("Expected command '/start dev', got '%s'", executor.executedCommands[0])
	}
	
	// Verify command was returned
	if cmd == nil {
		t.Error("Expected command to be returned")
	}
}

func TestCommandPalette_WithAdapter(t *testing.T) {
	// Create fake provider and executor
	provider := &FakeSuggestionProvider{
		suggestions: []CommandSuggestion{
			{Command: "/test", Description: "Test command", Category: "Test"},
		},
	}
	executor := &FakeCommandExecutor{}
	
	// Create adapter
	adapter := NewPaletteAdapter(provider, executor)
	
	// Test GetSuggestions through adapter
	suggestions := adapter.GetSuggestions("/t")
	
	if len(suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(suggestions))
	}
	
	// Verify format includes description
	if !strings.Contains(suggestions[0], "Test command") {
		t.Errorf("Expected suggestion to contain description, got '%s'", suggestions[0])
	}
	
	// Test ExecuteCommand through adapter
	cmd := adapter.ExecuteCommand("/test")
	
	if len(executor.executedCommands) != 1 {
		t.Errorf("Expected 1 command execution, got %d", len(executor.executedCommands))
	}
	
	if cmd == nil {
		t.Error("Expected command to be returned")
	}
}

func TestDynamicSuggestionProvider(t *testing.T) {
	// Create provider with test data
	scripts := map[string]string{
		"dev": "Development server",
		"test": "Run tests",
	}
	
	provider := NewDynamicSuggestionProvider(nil, scripts)
	
	// Test script suggestions
	suggestions := provider.Suggest("/start")
	
	found := false
	for _, s := range suggestions {
		if s.Command == "/start dev" {
			found = true
			if !strings.Contains(s.Description, "Development server") {
				t.Errorf("Expected description to contain 'Development server', got '%s'", s.Description)
			}
			break
		}
	}
	
	if !found {
		t.Error("Expected to find '/start dev' suggestion")
	}
	
	// Test navigation suggestions
	suggestions = provider.Suggest("/log")
	
	found = false
	for _, s := range suggestions {
		if s.Command == "/logs" {
			found = true
			if s.Shortcut != "2" {
				t.Errorf("Expected shortcut '2' for logs, got '%s'", s.Shortcut)
			}
			break
		}
	}
	
	if !found {
		t.Error("Expected to find '/logs' suggestion")
	}
}

func TestCommandInputComponent_Integration(t *testing.T) {
	// Integration test with real provider and executor
	scripts := map[string]string{
		"dev": "Development server",
	}
	
	provider := NewDynamicSuggestionProvider(nil, scripts)
	executor := &FakeCommandExecutor{}
	
	// Create component with injected dependencies
	component := NewCommandInputComponent()
	component.SetProvider(provider)
	component.SetExecutor(executor)
	
	// Simulate user typing "/st"
	component.textInput.SetValue("/st")
	component.updateSuggestions()
	
	// Should suggest start commands
	found := false
	for _, s := range component.suggestions {
		if strings.HasPrefix(s.Command, "/start") {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected to find start command suggestions")
	}
	
	// Simulate selecting first suggestion with tab
	if len(component.suggestions) > 0 {
		// Need to ensure suggestions are shown first
		component.showSuggestions = true
		
		// Simulate tab key to select suggestion
		model, _ := component.Update(tea.KeyMsg{Type: tea.KeyTab})
		updatedComponent := model.(*CommandInputComponent)
		
		// Input should be updated to the suggestion
		if !strings.HasPrefix(updatedComponent.textInput.Value(), "/start") {
			t.Errorf("Expected input to start with '/start', got '%s'", updatedComponent.textInput.Value())
		}
		component = updatedComponent
	}
	
	// Simulate executing command
	_, cmd := component.Update(tea.KeyMsg{Type: tea.KeyEnter})
	
	if cmd == nil {
		t.Error("Expected command to be executed")
	}
	
	if len(executor.executedCommands) == 0 {
		t.Error("Expected command to be executed through executor")
	}
}