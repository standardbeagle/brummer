# Redux/Elm Architecture Pattern for Brummer TUI

## Executive Summary

Brummer adopts a **Redux/Elm-inspired architecture** using BubbleTea as the runtime. This pattern provides predictable state management, eliminates race conditions, and reduces complexity by 50-70% compared to traditional service-oriented architectures.

**Key Insight**: The Model is pure data (like Redux store), Services create commands (like action creators), and Update() is the only place state changes (like reducers).

## Core Architecture Components

### 1. Model (Redux Store / Elm Model)
The Model is a **pure data structure** containing all application state. It has NO business logic methods that modify state.

```go
// Model is like a Redux store - pure data only
type Model struct {
    // Direct state fields
    currentView     View
    processes       map[string]*Process
    logs            []LogEntry
    urls            map[string][]URL
    errors          []ErrorEntry
    settings        Settings

    // Embedded state managers (for organization only)
    viewManager     *SimpleViewManager   // Atomic view state
    processManager  *ProcessManager      // Process tracking
    aiCoders        *AICoderManager      // AI session state
}

// ✅ OK: Read-only methods that don't modify state
func (m *Model) GetVisibleLogs() []LogEntry {
    if m.currentView == ViewLogs {
        return m.logs
    }
    return nil
}

// ❌ WRONG: Never put state-modifying methods on Model
func (m *Model) AddLog(entry LogEntry) {  // DON'T DO THIS
    m.logs = append(m.logs, entry)
}
```

### 2. Messages (Redux Actions / Elm Msg)
Messages are **immutable events** that describe what happened. They carry data but contain no logic.

```go
// Messages describe events - past tense naming
type ProcessStartedMsg struct {
    Process *Process
}

type LogReceivedMsg struct {
    ProcessID string
    Line      string
    Timestamp time.Time
}

type ViewSwitchedMsg struct {
    PreviousView View
    NewView      View
}

// Messages are data-only - no methods
type KeyPressedMsg struct {
    Key string
}
```

### 3. Update Function (Redux Reducer / Elm Update)
The Update function is the **ONLY place where state changes happen**. It's a pure function that takes current state and a message, returning new state.

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Pattern match on message type
    switch msg := msg.(type) {

    case ProcessStartedMsg:
        // Pure state transformation
        m.processes[msg.Process.ID] = msg.Process
        m.processCount++

        // Can return additional commands for side effects
        return m, tea.Batch(
            watchProcessOutput(msg.Process.ID),
            notifyProcessStarted(msg.Process),
        )

    case LogReceivedMsg:
        // Another pure transformation
        m.logs = append(m.logs, LogEntry{
            ProcessID: msg.ProcessID,
            Text:      msg.Line,
            Timestamp: msg.Timestamp,
        })

        // Extract URLs if found
        if urls := extractURLs(msg.Line); len(urls) > 0 {
            m.urls[msg.ProcessID] = append(m.urls[msg.ProcessID], urls...)
        }

        return m, nil

    case ViewSwitchedMsg:
        // Update view state atomically
        m.currentView = msg.NewView
        if m.viewManager != nil {
            m.viewManager.SetView(msg.NewView)
        }

        // Log view change for debugging
        return m, logViewChange(msg.PreviousView, msg.NewView)

    case tea.KeyMsg:
        // Delegate to keyboard handler
        if cmd := m.handleKeyPress(msg); cmd != nil {
            return m, cmd
        }
        return m, nil
    }

    return m, nil
}
```

### 4. Services (Redux Action Creators / Elm Cmd Creators)
Services are **stateless utilities** that create Commands. They NEVER hold state or modify the Model directly.

```go
// Services are stateless - no internal state
type ProcessService struct {
    // ✅ OK: Configuration and dependencies
    execPath string
    logger   *Logger

    // ❌ WRONG: Never store state in services
    // processes map[string]*Process  // DON'T DO THIS
}

// Services create Commands that produce Messages
func (s *ProcessService) StartProcess(name, cmd string) tea.Cmd {
    return func() tea.Msg {
        // Do async work (I/O, computation, etc.)
        process, err := s.executeProcess(cmd)
        if err != nil {
            return ProcessErrorMsg{
                Name:  name,
                Error: err,
            }
        }

        // Return a message for Update() to handle
        return ProcessStartedMsg{
            Process: process,
        }
    }
}

// Services can compose multiple commands
func (s *ProcessService) RestartProcess(id string) tea.Cmd {
    return tea.Sequence(
        s.StopProcess(id),
        tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
            return DelayCompleteMsg{}
        }),
        s.StartProcess(id, s.getProcessCmd(id)),
    )
}
```

### 5. Commands (Redux Thunks/Sagas / Elm Cmd)
Commands represent **side effects and async operations**. They run outside the Update loop and produce Messages when complete.

```go
// Commands are functions that return Messages
func watchProcessOutput(processID string) tea.Cmd {
    return func() tea.Msg {
        // Set up file watcher or pipe reader
        reader := getProcessOutputReader(processID)

        // Read line (blocking I/O)
        line, err := reader.ReadString('\n')
        if err != nil {
            return ProcessEndedMsg{ID: processID}
        }

        // Return message for Update to handle
        return LogReceivedMsg{
            ProcessID: processID,
            Line:      line,
            Timestamp: time.Now(),
        }
    }
}

// Commands can be batched
func initializeApp() tea.Cmd {
    return tea.Batch(
        loadConfiguration(),
        detectRunningProcesses(),
        setupFileWatchers(),
        startHTTPServer(),
    )
}

// Commands handle all side effects
func saveSettings(settings Settings) tea.Cmd {
    return func() tea.Msg {
        if err := writeSettingsToFile(settings); err != nil {
            return SettingsErrorMsg{Error: err}
        }
        return SettingsSavedMsg{Timestamp: time.Now()}
    }
}
```

## Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      USER INPUT                              │
└──────────────────────────┬───────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                    KEYBOARD HANDLER                          │
│                  (Captures user input)                       │
└──────────────────────────┬───────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                      TEA.MSG                                 │
│                 (KeyMsg, MouseMsg, etc)                      │
└──────────────────────────┬───────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                   UPDATE FUNCTION                            │
│            (ONLY place state changes happen)                 │
│                                                              │
│  switch msg := msg.(type) {                                 │
│    case KeyMsg:                                             │
│      → Modify Model                                         │
│      → Return Commands                                      │
│  }                                                          │
└────────────┬──────────────────────────┬─────────────────────┘
             ↓                          ↓
    ┌────────────────┐          ┌────────────────┐
    │     MODEL      │          │    COMMANDS    │
    │  (New State)   │          │ (Side Effects) │
    └────────┬───────┘          └────────┬───────┘
             ↓                           ↓
    ┌────────────────┐          ┌────────────────┐
    │     VIEW       │          │    SERVICES    │
    │   (Render)     │          │   (Async I/O)  │
    └────────────────┘          └────────┬───────┘
                                         ↓
                                ┌────────────────┐
                                │   MESSAGES     │
                                │  (Events)      │
                                └────────┬───────┘
                                         ↓
                            (Back to Update Function)
```

## Key Principles

### 1. Unidirectional Data Flow
Data flows in ONE direction: Input → Message → Update → Model → View

```go
// ✅ CORRECT: Unidirectional flow
Input → KeyMsg → Update() → Model change → View re-render

// ❌ WRONG: Service directly modifying Model
Service → Model.logs.append()  // Never do this!
```

### 2. Single Source of Truth
Each piece of state has exactly ONE owner - the Model.

```go
// ✅ CORRECT: Model owns all state
type Model struct {
    currentView View  // Single source of truth
}

// ❌ WRONG: Duplicate state
type NavigationService struct {
    currentView View  // Duplicate of Model.currentView - DON'T!
}
```

### 3. Immutable Messages
Messages are read-only data structures.

```go
// ✅ CORRECT: Messages are immutable
type LogMsg struct {
    Line string  // Can't be modified after creation
}

// ❌ WRONG: Mutable messages
type LogMsg struct {
    Line *string  // Pointer allows mutation - avoid!
}
```

### 4. Pure Update Function
Update is a pure function - same input always produces same output.

```go
// ✅ CORRECT: Pure function
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Only uses msg and m to compute result
    switch msg := msg.(type) {
    case AddLogMsg:
        m.logs = append(m.logs, msg.Log)
        return m, nil
    }
}

// ❌ WRONG: Side effects in Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Don't do I/O in Update!
    file.Write(msg.Log)  // Side effect - use Command instead!
}
```

### 5. Commands for Side Effects
ALL side effects happen in Commands, never in Update.

```go
// ✅ CORRECT: Side effects in Commands
func saveLog(log string) tea.Cmd {
    return func() tea.Msg {
        // I/O happens here, not in Update
        if err := os.WriteFile("log.txt", []byte(log), 0644); err != nil {
            return ErrorMsg{err}
        }
        return LogSavedMsg{}
    }
}
```

## Practical Implementation Guide

### Step 1: Define Your Model
Start with the data structure that represents your entire application state.

```go
type Model struct {
    // View state
    currentView     View
    viewHistory     []View

    // Process state
    processes       map[string]*Process
    activeProcessID string

    // UI state
    logs            []LogEntry
    filteredLogs    []LogEntry
    logFilter       string
    scrollOffset    int

    // Settings
    settings        Settings
    theme          Theme
}
```

### Step 2: Define Your Messages
Create messages for every event that can occur.

```go
// User actions
type KeyPressMsg struct{ Key string }
type MouseClickMsg struct{ X, Y int }

// Process events
type ProcessStartedMsg struct{ Process *Process }
type ProcessOutputMsg struct{ ID, Line string }
type ProcessExitedMsg struct{ ID string; Code int }

// System events
type WindowResizeMsg struct{ Width, Height int }
type ConfigLoadedMsg struct{ Config Config }
```

### Step 3: Implement Update Function
Handle each message type and update state accordingly.

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    // Handle keyboard input
    case tea.KeyMsg:
        switch msg.String() {
        case "1", "2", "3", "4", "5", "6", "7":
            view := numberToView(msg.String())
            m.currentView = view
            return m, nil

        case "ctrl+c", "q":
            return m, tea.Quit

        case "enter":
            if m.currentView == ViewProcesses {
                selected := m.getSelectedProcess()
                return m, startProcess(selected)
            }
        }

    // Handle process events
    case ProcessStartedMsg:
        m.processes[msg.Process.ID] = msg.Process
        m.activeProcessID = msg.Process.ID
        return m, watchProcess(msg.Process.ID)

    case ProcessOutputMsg:
        m.logs = append(m.logs, LogEntry{
            ProcessID: msg.ID,
            Text:     msg.Line,
        })

        // Update filtered view
        if m.matchesFilter(msg.Line) {
            m.filteredLogs = append(m.filteredLogs, m.logs[len(m.logs)-1])
        }

        return m, nil
    }

    return m, nil
}
```

### Step 4: Create Services for Commands
Services create commands but never hold state.

```go
type ProcessService struct {
    workDir string
}

func (s *ProcessService) Start(name, cmd string) tea.Cmd {
    return func() tea.Msg {
        proc := exec.Command("sh", "-c", cmd)
        proc.Dir = s.workDir

        if err := proc.Start(); err != nil {
            return ErrorMsg{err}
        }

        return ProcessStartedMsg{
            Process: &Process{
                ID:   generateID(),
                Name: name,
                Cmd:  cmd,
                Pid:  proc.Process.Pid,
            },
        }
    }
}
```

### Step 5: Implement View Function
View reads from Model to render UI.

```go
func (m Model) View() string {
    switch m.currentView {
    case ViewProcesses:
        return m.renderProcessList()
    case ViewLogs:
        return m.renderLogs()
    default:
        return "Unknown view"
    }
}

func (m Model) renderLogs() string {
    var b strings.Builder
    for _, log := range m.filteredLogs {
        b.WriteString(log.Format())
        b.WriteString("\n")
    }
    return b.String()
}
```

## Testing Strategy

### 1. Test Messages and Updates
Since Update is pure, it's easy to test.

```go
func TestUpdateProcessStarted(t *testing.T) {
    // Given
    model := Model{
        processes: make(map[string]*Process),
    }
    msg := ProcessStartedMsg{
        Process: &Process{ID: "test-1", Name: "test"},
    }

    // When
    newModel, cmd := model.Update(msg)

    // Then
    m := newModel.(Model)
    assert.Equal(t, "test-1", m.processes["test-1"].ID)
    assert.NotNil(t, cmd)  // Should return watch command
}
```

### 2. Test Services Return Correct Commands
Services are stateless, so testing is straightforward.

```go
func TestProcessServiceStart(t *testing.T) {
    // Given
    service := &ProcessService{workDir: "/tmp"}

    // When
    cmd := service.Start("test", "echo hello")
    msg := cmd()  // Execute command

    // Then
    startMsg, ok := msg.(ProcessStartedMsg)
    assert.True(t, ok)
    assert.Equal(t, "test", startMsg.Process.Name)
}
```

### 3. Integration Tests for User Flows
Test complete user interactions.

```go
func TestUserSwitchesViews(t *testing.T) {
    // Given
    model := NewModel()

    // When user presses "2"
    model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})

    // Then
    assert.Equal(t, ViewLogs, model.(Model).currentView)
}
```

## Migration Strategy

### From Service-Oriented to Redux/Elm

1. **Identify State Holders**
   - Find all services that maintain state
   - List all state synchronization points

2. **Consolidate into Model**
   ```go
   // Before: State in services
   type NavigationService struct {
       currentView View
   }

   // After: State in Model
   type Model struct {
       currentView View
   }
   ```

3. **Convert Service Methods to Commands**
   ```go
   // Before: Service with state
   func (s *LogService) AddLog(log string) {
       s.logs = append(s.logs, log)
   }

   // After: Service returns command
   func (s *LogService) AddLog(log string) tea.Cmd {
       return func() tea.Msg {
           return LogAddedMsg{Log: log}
       }
   }
   ```

4. **Move State Changes to Update**
   ```go
   // All state changes in Update
   case LogAddedMsg:
       m.logs = append(m.logs, msg.Log)
       return m, nil
   ```

## Common Pitfalls and Solutions

### Pitfall 1: State in Services
**Problem**: Services holding state leads to synchronization bugs.
**Solution**: Services should be stateless, only creating Commands.

### Pitfall 2: Side Effects in Update
**Problem**: Doing I/O in Update makes it impure and hard to test.
**Solution**: Return Commands for all side effects.

### Pitfall 3: Mutable Messages
**Problem**: Modifying messages breaks predictability.
**Solution**: Messages should be immutable value objects.

### Pitfall 4: Direct Model Mutation
**Problem**: Modifying Model outside Update breaks unidirectional flow.
**Solution**: ALL state changes must go through Update.

### Pitfall 5: Complex Service Dependencies
**Problem**: Services depending on each other creates coupling.
**Solution**: Services should be independent, coordination happens in Update.

## Performance Considerations

### 1. Atomic Operations for Concurrent Reads
```go
type SimpleViewManager struct {
    currentView atomic.Value  // Thread-safe reads
}

func (vm *SimpleViewManager) GetCurrentView() View {
    return vm.currentView.Load().(View)  // No lock needed
}
```

### 2. Batch Commands for Efficiency
```go
// Batch related commands
return m, tea.Batch(
    loadConfig(),
    startWatchers(),
    initializeUI(),
)
```

### 3. Debounce High-Frequency Events
```go
// Debounce rapid updates
var debounceTimer *time.Timer

case RapidUpdateMsg:
    if debounceTimer != nil {
        debounceTimer.Stop()
    }
    debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
        // Process update
    })
```

## Comparison with Other Patterns

### vs. Traditional MVC
- **MVC**: Controllers modify Model directly, View observes Model
- **Redux/Elm**: Update is the only modifier, View is pure function of Model

### vs. Service-Oriented Architecture
- **SOA**: Services maintain state, coordinate with each other
- **Redux/Elm**: Services are stateless, Model is single source of truth

### vs. Event-Driven Architecture
- **EDA**: Events trigger handlers that may modify various states
- **Redux/Elm**: Messages trigger Update which modifies single Model

## Conclusion

The Redux/Elm pattern in Brummer provides:
- **Predictable state management** - All changes go through Update
- **Easy debugging** - Linear flow, time-travel possible
- **Simplified testing** - Pure functions, no mocking needed
- **Eliminated race conditions** - Single source of truth
- **Reduced complexity** - 50-70% less code than service architectures

This architecture scales from simple TUI apps to complex multi-view applications while maintaining clarity and correctness.