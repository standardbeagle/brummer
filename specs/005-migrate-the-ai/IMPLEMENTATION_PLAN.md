# Implementation Plan: Remaining Tasks

**Current Status**: Phase 1 Complete (T001-T016), Phase 2 Foundation Complete (T017-T022)

**Remaining**: T023-T038 (16 tasks, ~8-10 hours estimated)

## Critical Path Analysis

### Immediate Priority: T023-T027 (Component Implementation)

These tasks MUST be completed sequentially as they build on each other:

#### T023: Message Handlers (45 min) - **START HERE**
**File**: `internal/tui/components/aicoders_view.go`

Add to `Update()` method:
```go
func (c *AICodersViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case messages.AICoderSessionCreatedMsg:
        // Fetch updated session list from service
        return c, c.fetchSessionList()

    case messages.SessionListMsg:
        // Update cached sessions
        c.cachedSessions = msg.Sessions
        c.updateViewport()
        return c, nil

    case messages.AICoderSessionUpdatedMsg:
        // Update specific session in cache
        for i, sess := range c.cachedSessions {
            if sess.ID == msg.SessionID {
                c.cachedSessions[i].Status = msg.Status
                c.cachedSessions[i].LastActivity = msg.LastActivity
                break
            }
        }
        c.updateViewport()
        return c, nil

    case messages.AICoderSessionEndedMsg:
        // Remove from cached sessions
        for i, sess := range c.cachedSessions {
            if sess.ID == msg.SessionID {
                c.cachedSessions = append(c.cachedSessions[:i], c.cachedSessions[i+1:]...)
                break
            }
        }
        c.updateViewport()
        return c, nil

    case messages.AICoderPTYOutputMsg:
        // Fetch updated terminal state
        if c.viewState.Mode == ViewModePTY && c.viewState.FocusedSessionID == msg.SessionID {
            return c, c.fetchTerminalState(msg.SessionID)
        }
        return c, nil

    case messages.TerminalStateMsg:
        // Update cached terminal state
        c.cachedTerminal = msg.State
        c.updatePTYView()
        return c, nil

    case messages.AICoderErrorMsg:
        // Display error (could use notification system)
        // For now, just log to console or show in UI
        return c, nil
    }

    // Existing WindowSizeMsg and KeyMsg handling...
    return c.BaseUpdate(msg)
}
```

#### T024: Keyboard Routing (40 min)
**File**: `internal/tui/components/aicoders_view.go`

Update `Update()` method's KeyMsg handling:
```go
case tea.KeyMsg:
    switch c.viewState.Mode {
    case ViewModeList:
        switch msg.String() {
        case "j", "down":
            if len(c.cachedSessions) > 0 {
                c.viewState.SelectedIndex = (c.viewState.SelectedIndex + 1) % len(c.cachedSessions)
                c.updateViewport()
            }
            return c, nil
        case "k", "up":
            if len(c.cachedSessions) > 0 {
                c.viewState.SelectedIndex = (c.viewState.SelectedIndex - 1 + len(c.cachedSessions)) % len(c.cachedSessions)
                c.updateViewport()
            }
            return c, nil
        case "enter":
            if len(c.cachedSessions) > 0 {
                c.viewState.Mode = ViewModePTY
                c.viewState.FocusedSessionID = c.cachedSessions[c.viewState.SelectedIndex].ID
                return c, c.fetchTerminalState(c.viewState.FocusedSessionID)
            }
            return c, nil
        }

    case ViewModePTY:
        switch msg.String() {
        case "esc":
            c.viewState.Mode = ViewModeList
            c.viewState.FocusedSessionID = ""
            c.cachedTerminal = nil
            c.updateViewport()
            return c, nil
        default:
            // Forward all other keys to PTY
            if c.viewState.FocusedSessionID != "" {
                return c, c.sendInput(c.viewState.FocusedSessionID, []byte(msg.String()))
            }
        }
    }
```

#### T025: PTY Rendering (50 min)
**File**: `internal/tui/components/aicoders_view.go`

Add new method:
```go
func (c *AICodersViewComponent) updatePTYView() {
    if c.viewState.Mode != ViewModePTY || c.cachedTerminal == nil {
        return
    }

    var lines []string

    // Header
    sessionName := "Unknown"
    for _, sess := range c.cachedSessions {
        if sess.ID == c.viewState.FocusedSessionID {
            sessionName = sess.Name
            break
        }
    }

    lines = append(lines, c.headerStyle.Render(fmt.Sprintf("Session: %s", sessionName)))
    lines = append(lines, strings.Repeat("─", c.viewState.ViewportWidth))

    // Terminal content
    termState := c.cachedTerminal.(*messages.TerminalState)
    for _, line := range termState.Lines {
        lines = append(lines, line)
    }

    // Cursor indicator
    if termState.Cursor.Visible {
        cursorLine := fmt.Sprintf("Cursor: %d,%d", termState.Cursor.Row, termState.Cursor.Col)
        lines = append(lines, "")
        lines = append(lines, c.normalStyle.Render(cursorLine))
    }

    // Footer
    lines = append(lines, strings.Repeat("─", c.viewState.ViewportWidth))
    lines = append(lines, c.normalStyle.Render("ESC=List | ?=Help | ↑↓=Scroll"))

    content := strings.Join(lines, "\n")
    c.viewport.SetContent(content)
}
```

#### T026: Command Constructors (30 min)
**File**: `internal/tui/components/aicoders_view.go`

Add helper methods:
```go
func (c *AICodersViewComponent) fetchSessionList() tea.Cmd {
    return func() tea.Msg {
        if c.service == nil {
            return nil
        }
        svc := c.service.(*services.AICoderService)
        sessions := svc.ListSessions()
        return messages.SessionListMsg{Sessions: sessions}
    }
}

func (c *AICodersViewComponent) fetchTerminalState(sessionID string) tea.Cmd {
    return func() tea.Msg {
        if c.service == nil {
            return nil
        }
        svc := c.service.(*services.AICoderService)
        state, err := svc.GetTerminalState(sessionID)
        if err != nil {
            return messages.AICoderErrorMsg{
                SessionID: sessionID,
                Error:     err,
                Severity:  "error",
            }
        }
        return messages.TerminalStateMsg{
            SessionID: sessionID,
            State:     state,
        }
    }
}

func (c *AICodersViewComponent) sendInput(sessionID string, input []byte) tea.Cmd {
    return func() tea.Msg {
        if c.service == nil {
            return nil
        }
        svc := c.service.(*services.AICoderService)
        if err := svc.SendInput(sessionID, input); err != nil {
            return messages.AICoderErrorMsg{
                SessionID: sessionID,
                Error:     err,
                Severity:  "error",
            }
        }
        return nil
    }
}
```

#### T027: Resize Handling (20 min)
**File**: `internal/tui/components/aicoders_view.go`

Update WindowSizeMsg handling:
```go
case tea.WindowSizeMsg:
    c.viewState.ViewportWidth = msg.Width
    c.viewState.ViewportHeight = msg.Height
    c.viewport.Width = msg.Width
    c.viewport.Height = msg.Height

    // If in PTY mode, resize the PTY
    if c.viewState.Mode == ViewModePTY && c.viewState.FocusedSessionID != "" {
        if c.service != nil {
            svc := c.service.(*services.AICoderService)
            _ = svc.ResizePTY(c.viewState.FocusedSessionID, msg.Width, msg.Height)
        }
    }

    c.updateViewport()
    return c, nil
```

### Next Priority: T028 (Integration) - 30 min

**File**: `internal/tui/model.go`

In model initialization:
```go
// Create AI Coder Service
ctx := context.Background()
aiCoderService := services.NewAICoderService(ctx)

// Create component with service
aicodersView := components.NewAICodersViewComponent(aiCoderService)

// Subscribe to service events
eventBus := aiCoderService.EventBus().(*events.EventBus)
eventBus.Subscribe("aicoder.session.created", func(event events.Event) {
    // Send to BubbleTea
    p.Send(messages.AICoderSessionCreatedMsg{
        SessionID: event.Data["sessionID"].(string),
        Name:      event.Data["name"].(string),
    })
})

// Similar subscriptions for other events...

// Add to model
m.aiCoderService = aiCoderService
m.shell.SetAICodersView(aicodersView)
```

### Final Priority: T029-T032 (Cleanup) - 20 min

Simply delete the old files:
```bash
git rm internal/tui/ai_coder_controller.go
git rm internal/tui/ai_coder_pty_view.go
git rm internal/tui/ai_coder_debug_forwarder.go
git rm internal/tui/ai_coder_message_handler.go
```

### Validation: T033-T038 - 3-4 hours

Run tests and manual validation as per quickstart.md

## Estimated Total Time: 8-10 hours

## Key Implementation Notes

1. **Type Assertions**: When accessing c.service, cast to `*services.AICoderService`
2. **Nil Checks**: Always check if service is nil before calling methods
3. **Error Handling**: Use AICoderErrorMsg for all errors
4. **Performance**: Cache aggressively, only fetch terminal state when in PTY view
5. **Testing**: Enable skipped tests one by one as features are implemented

## Quick Wins

If time is limited, focus on this minimal implementation:
1. T023 (message handlers) - CRITICAL
2. T026 (command constructors) - CRITICAL
3. T028 (integration) - CRITICAL

This gives you a working but limited component that can be enhanced later.
