package tui

import (
    "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// MODEL - Pure Data (like Redux Store / Elm Model)
// ============================================================================

type Model struct {
    // Pure state - no methods that modify state directly
    processes    map[string]*Process
    currentView  View
    logs         []LogEntry

    // Embedded state slices (organized but still pure data)
    viewState    ViewState
    processState ProcessState
}

// Read-only methods are OK on Model
func (m *Model) GetVisibleLogs() []LogEntry {
    // Pure function - no side effects
    if m.currentView == ViewLogs {
        return m.logs
    }
    return nil
}

// ============================================================================
// MESSAGES - Events (like Redux Actions / Elm Msg)
// ============================================================================

type ProcessStartedMsg struct {
    Process *Process
}

type LogReceivedMsg struct {
    ProcessID string
    Line      string
}

type ViewSwitchedMsg struct {
    View View
}

// ============================================================================
// SERVICES - Command Creators (like Action Creators / Elm Cmd creators)
// ============================================================================

type ProcessService struct {
    // No state! Services are stateless
}

// Services create commands, not state changes
func (s *ProcessService) StartProcess(name, cmd string) tea.Cmd {
    return func() tea.Msg {
        // Do I/O or async work here
        process := &Process{
            ID:   generateID(),
            Name: name,
            Cmd:  cmd,
        }
        // Start the actual process...

        // Return a message (like dispatching an action)
        return ProcessStartedMsg{Process: process}
    }
}

func (s *ProcessService) StopProcess(id string) tea.Cmd {
    return func() tea.Msg {
        // Stop the process...
        return ProcessStoppedMsg{ID: id}
    }
}

// ============================================================================
// UPDATE - State Reducer (like Redux Reducer / Elm Update)
// ============================================================================

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    // Handle process started
    case ProcessStartedMsg:
        // Pure state transformation
        m.processes[msg.Process.ID] = msg.Process

        // Can return additional commands
        return m, watchProcessLogs(msg.Process.ID)

    // Handle log received
    case LogReceivedMsg:
        // Another pure transformation
        m.logs = append(m.logs, LogEntry{
            ProcessID: msg.ProcessID,
            Text:      msg.Line,
        })
        return m, nil

    // Handle view switch
    case ViewSwitchedMsg:
        m.currentView = msg.View
        return m, nil
    }

    return m, nil
}

// ============================================================================
// USAGE EXAMPLE
// ============================================================================

func Example() {
    // Services are instantiated once (or could be singletons)
    processService := &ProcessService{}

    // In your Update function, use services to create commands:
    // case key.Matches(msg, "ctrl+n"):
    //     return m, processService.StartProcess("npm", "npm run dev")

    // The command will eventually produce a ProcessStartedMsg
    // which Update() will handle to modify state
}

// ============================================================================
// KEY INSIGHTS
// ============================================================================

/*
1. Model contains ONLY data, no business logic
2. Services create Commands (tea.Cmd), never modify state
3. Update() is the ONLY place state changes happen
4. Messages (tea.Msg) describe what happened
5. Commands (tea.Cmd) describe what should happen

This is essentially:
- Elm Architecture (Model/Update/View)
- Redux pattern (Store/Actions/Reducers)
- Event Sourcing (Messages as events)

Benefits:
- Predictable state changes
- Easy testing (services return commands)
- No state synchronization bugs
- Time-travel debugging possible
- Clear separation of concerns
*/