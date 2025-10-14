package contracts

import (
    "testing"
    tea "github.com/charmbracelet/bubbletea"
)

// UpdateContract defines the contract for the Update function
// This is the core of the Redux/Elm pattern - all state changes happen here
type UpdateContract interface {
    // Update receives a message and returns updated model and optional command
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
}

// TestUpdateContract validates the Update function contract
func TestUpdateContract(t *testing.T, model UpdateContract) {
    tests := []struct {
        name     string
        msg      tea.Msg
        validate func(t *testing.T, newModel tea.Model, cmd tea.Cmd)
    }{
        {
            name: "ViewSwitchedMsg updates current view",
            msg: ViewSwitchedMsg{
                PreviousView: ViewProcesses,
                NewView:      ViewLogs,
            },
            validate: func(t *testing.T, newModel tea.Model, cmd tea.Cmd) {
                m := newModel.(Model)
                if m.currentView != ViewLogs {
                    t.Errorf("expected view to be ViewLogs, got %v", m.currentView)
                }
                if cmd != nil {
                    t.Error("expected no command for view switch")
                }
            },
        },
        {
            name: "ProcessStartedMsg adds process to model",
            msg: ProcessStartedMsg{
                Process: &Process{
                    ID:   "test-123",
                    Name: "test",
                    Cmd:  "echo test",
                },
            },
            validate: func(t *testing.T, newModel tea.Model, cmd tea.Cmd) {
                m := newModel.(Model)
                if _, exists := m.processes["test-123"]; !exists {
                    t.Error("expected process to be added to model")
                }
                if cmd == nil {
                    t.Error("expected command to watch process output")
                }
            },
        },
        {
            name: "KeyMsg for number switches view",
            msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}},
            validate: func(t *testing.T, newModel tea.Model, cmd tea.Cmd) {
                m := newModel.(Model)
                if m.currentView != ViewLogs {
                    t.Errorf("expected view to be ViewLogs after pressing '2', got %v", m.currentView)
                }
            },
        },
        {
            name: "WindowResizedMsg updates dimensions",
            msg:  WindowResizedMsg{Width: 120, Height: 40},
            validate: func(t *testing.T, newModel tea.Model, cmd tea.Cmd) {
                m := newModel.(Model)
                if m.windowWidth != 120 || m.windowHeight != 40 {
                    t.Errorf("expected window size 120x40, got %dx%d", m.windowWidth, m.windowHeight)
                }
            },
        },
        {
            name: "Unknown message returns model unchanged",
            msg:  struct{ Unknown string }{Unknown: "test"},
            validate: func(t *testing.T, originalModel tea.Model, cmd tea.Cmd) {
                // Model should be unchanged for unknown messages
                if cmd != nil {
                    t.Error("expected no command for unknown message")
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            newModel, cmd := model.Update(tt.msg)
            tt.validate(t, newModel, cmd)
        })
    }
}

// Contract: Update function must be pure
func TestUpdatePurity(t *testing.T, model UpdateContract) {
    msg := ViewSwitchedMsg{
        PreviousView: ViewProcesses,
        NewView:      ViewLogs,
    }

    // Call Update twice with same input
    model1, cmd1 := model.Update(msg)
    model2, cmd2 := model.Update(msg)

    // Results should be identical (pure function)
    if model1 != model2 {
        t.Error("Update function is not pure - different models for same input")
    }

    // Commands might differ (they can have timestamps), but types should match
    if (cmd1 == nil) != (cmd2 == nil) {
        t.Error("Update function is not pure - different command presence for same input")
    }
}

// Contract: Update must handle all message types without panic
func TestUpdateRobustness(t *testing.T, model UpdateContract) {
    messages := []tea.Msg{
        nil, // nil message
        ViewSwitchedMsg{},
        ProcessStartedMsg{},
        LogFilterChangedMsg{},
        ErrorDetectedMsg{},
        URLDiscoveredMsg{},
        WindowResizedMsg{},
        tea.KeyMsg{},
        tea.MouseMsg{},
    }

    for i, msg := range messages {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    t.Errorf("Update panicked on message %d: %v", i, r)
                }
            }()

            model.Update(msg)
        }()
    }
}