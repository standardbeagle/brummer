# AI Coder PTY Viewport Scrolling

## Overview

The AI Coder PTY terminal view supports scrollback buffer navigation through a **client-side viewport** system. This document explains how scrolling works and the separation between viewport scrolling and terminal interaction.

## Architecture

### PTY vs Terminal Emulator vs Viewport

1. **PTY (Pseudo-Terminal)**:
   - Pipe between Claude Code process and our terminal emulator
   - Handles terminal size via SIGWINCH signals
   - No knowledge of scrollback viewing

2. **vt10x Terminal Emulator**:
   - Maintains 1000-row scrollback buffer
   - Width matches PTY dimensions (dynamic)
   - Height is fixed at 1000 rows for scrollback history
   - Processes VT100/ANSI escape sequences

3. **Bubble Tea Viewport**:
   - Client-side scrolling component
   - Displays a window into the scrollback buffer
   - Handles Up/Down/PgUp/PgDn/j/k navigation
   - Completely separate from PTY I/O

## Two Modes of Operation

### Viewing Mode (Default)
**State**: `TerminalFocused == false`

**Behavior**:
- Arrow keys (↑↓) and PageUp/PageDown control **viewport scrolling**
- Vim keys (j/k) also work for scrolling
- You can scroll through the scrollback buffer to view history
- Keys are NOT sent to the Claude Code process
- Enter key switches to Interactive Mode

**Use Case**: Review previous output, scroll through long responses

### Interactive Mode (Terminal Focus)
**State**: `TerminalFocused == true`

**Behavior**:
- ALL keys are sent to the PTY (Claude Code process)
- Arrow keys control cursor position in Claude Code
- No viewport scrolling available
- Ctrl+T or ESC returns to Viewing Mode

**Use Case**: Type commands, interact with Claude Code directly

## How to Use

### Entering AI Coder View
1. Press `6` to switch to AI Coders view
2. Press `n` to create a new session (or select existing session)
3. Press Enter to view the PTY terminal
4. You start in **Viewing Mode** (not focused)

### Scrolling Through History
**In Viewing Mode**:
- `↑` / `k`: Scroll up one line
- `↓` / `j`: Scroll down one line
- `PageUp`: Scroll up one page
- `PageDown`: Scroll down one page

### Interacting with Terminal
1. Press `Enter` or `Ctrl+T` to enter **Interactive Mode**
2. Type commands, use arrow keys for cursor movement
3. Press `Ctrl+T` or `ESC` to return to **Viewing Mode**

### Status Messages
- **"Terminal focus: OFF (arrows=scroll, Enter=focus)"**: You're in Viewing Mode, can scroll
- **"Terminal focus: ON (all keys go to terminal, Ctrl+T or ESC to release)"**: Interactive Mode active

## Technical Implementation

### Key Handler Logic (`internal/tui/key_handler.go`)

```go
// When not in terminal focus mode, handle scrolling keys for viewport
if !aiState.TerminalFocused {
    isScrollKey := false
    switch key.Type {
    case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
        isScrollKey = true
    }

    // Also handle j/k vim-style scrolling
    if key.Type == tea.KeyRunes {
        switch string(key.Runes) {
        case "j", "k":
            isScrollKey = true
        }
    }

    // If it's a scroll key, forward to viewport
    if isScrollKey && model.aicodersView != nil {
        return model.aicodersView.UpdateViewport(key)
    }
}
```

### Viewport Content (`internal/tui/services/aicoder_service.go`)

```go
// Get cursor position to determine the visible viewport
cursor := vt10xTerm.Cursor()

// Calculate the viewport window (show the most recent content around cursor)
viewportHeight := 24
startRow := cursor.Y - viewportHeight + 1

// Build screen buffer from viewport (Cell positions)
for i := 0; i < viewportHeight; i++ {
    y := startRow + i
    // Process row y from vt10x terminal
}

// Set viewport content
c.viewport.SetContent(content)
```

### Viewport Update (`internal/tui/components/aicoders_view.go`)

```go
// UpdateViewport handles viewport scrolling for arrow keys and page up/down
func (c *AICodersViewComponent) UpdateViewport(msg tea.Msg) tea.Cmd {
    var cmd tea.Cmd
    c.viewport, cmd = c.viewport.Update(msg)
    return cmd
}
```

## Why Client-Side Scrolling?

**PTY signals don't handle scrollback** because:
1. SIGWINCH only resizes the active terminal window
2. The running application (Claude Code) doesn't need to know you're reviewing history
3. Scrollback is a feature of terminal **emulators**, not the PTY device itself

This is identical to how real terminals work:
- **iTerm2, gnome-terminal, Windows Terminal**: All implement client-side scrollback
- **ssh sessions**: Scrollback stays on the client, server doesn't know
- **tmux/screen**: These implement their own scrollback buffers

## Performance Characteristics

- **Scrollback buffer**: 1000 rows (vt10x default)
- **Viewport size**: 24 rows (typical terminal height)
- **Scroll operations**: <1ms (viewport is lightweight)
- **Memory**: ~100KB for scrollback buffer (1000 rows × 100 cols × 1 byte)

## Future Enhancements

Potential improvements:
- [ ] Search through scrollback buffer (Ctrl+F)
- [ ] Copy text from scrollback
- [ ] Save scrollback to file
- [ ] Configurable scrollback size
- [ ] Jump to top/bottom shortcuts (Home/End)
- [ ] Scroll position indicator (percentage)

## Related Files

- `internal/tui/key_handler.go`: Keyboard handling and mode switching
- `internal/tui/components/aicoders_view.go`: Viewport component and rendering
- `internal/tui/services/aicoder_service.go`: PTY session management and screen capture
- `internal/aicoder/pty_session.go`: PTY creation and vt10x terminal emulator
- `internal/tui/state/aicoder_state.go`: Terminal focus state management

## See Also

- [AI Coders Feature Guide](/docs/features/ai-coders.md)
- [Signal Handling Documentation](/docs/features/signal-handling.md)
- [TUI Architecture](/docs/architecture/overview.md)
