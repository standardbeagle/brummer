# Keyboard Handler Priority Order

## Overview

This document describes the keyboard handler priority system in Brummer's TUI. The priority system ensures that keys are processed in the correct order, with more specific/critical handlers taking precedence over generic ones.

## Priority Hierarchy

**Lower number = Higher priority** (processed first)

| Priority | Handler | Description | Keys Handled |
|----------|---------|-------------|--------------|
| **10** | AICoderKeyHandler | Terminal input in AI Coder view | ALL keys when in PTY/terminal focus mode |
| **50** | QuitHandler | Application exit with confirmation | ESC, Ctrl+C (with double-press protection) |
| **100** | SlashCommandHandler | Command palette | `/` key |
| **200** | ViewSpecificKeyHandler | View-specific actions | Varies by view |
| **900** | GlobalNavigationHandler | View navigation (LOWEST) | Left/Right arrows, number keys 1-7 |
| **1000** | GeneralKeyHandler | Catch-all for unhandled keys | Common keys with help messages |

## Rationale

### 1. AI Coder Terminal Input (Priority 10 - HIGHEST)
**Why highest?** When user is interacting with an AI coder terminal:
- ALL keys must go to the terminal (including Tab, arrows, etc.)
- Terminal needs complete keyboard control
- Prevents accidental view switches while typing

**Example**: User typing in AI Coder terminal shouldn't trigger view switches with arrow keys.

### 2. Exit Keys (Priority 50)
**Why second?** Application exit should work everywhere except:
- When terminal has focus (Ctrl+C goes to terminal)
- Requires confirmation to prevent accidents

**Example**: ESC exits the app, but only after confirmation.

### 3. Command Palette (Priority 100)
**Why before view-specific?** Command palette is a global feature:
- Available in most views
- More important than view-specific actions
- Users expect `/` to always work

**Example**: Pressing `/` in any view should open command palette.

### 4. View-Specific Handlers (Priority 200)
**Why before navigation?** View-specific actions take precedence:
- Keys like `c` for clear in Logs view
- Keys like `r` for restart in Processes view
- More specific than generic navigation

**Example**: `c` clears logs instead of being ignored.

### 5. Global Navigation (Priority 900 - LOWEST)
**Why lowest?** Navigation is the fallback:
- Shouldn't interfere with terminal input
- Shouldn't interfere with commands or actions
- Number keys and arrows are last resort

**Example**: Left/Right arrows navigate views UNLESS terminal has focus.

## Key Binding Changes

### Tab Key Removed from View Navigation
**Before**: Tab cycled through views
**After**: Tab reserved for input element navigation (future use)
**New Navigation**: Use **Left/Right arrow keys** to cycle views

### Navigation Keys Summary
| Key | Action |
|-----|--------|
| **Left Arrow** | Previous view |
| **Right Arrow** | Next view |
| **Number Keys (1-7)** | Jump to specific view |
| **Tab** | *(Reserved for input elements)* |

## Context-Aware Priority

Handlers check context before processing keys:

### AI Coder Terminal Focus Mode
```
Ctrl+T pressed in AI Coder view:
  ✓ AICoderKeyHandler (priority 10) - Handles it (toggles focus)
  ✗ QuitHandler (priority 50) - Skipped
  ✗ SlashCommandHandler (priority 100) - Skipped
  ✗ GlobalNavigationHandler (priority 900) - Skipped
```

### Terminal Focus Mode (all keys to terminal)
```
Left Arrow pressed in terminal focus mode:
  ✓ AICoderKeyHandler (priority 10) - Handles it (sends to terminal)
  ✗ GlobalNavigationHandler (priority 900) - Never reached
```

### Normal View (no terminal)
```
Left Arrow pressed in Processes view:
  ✗ AICoderKeyHandler (priority 10) - Wrong view, skips
  ✗ QuitHandler (priority 50) - Wrong key, skips
  ✗ SlashCommandHandler (priority 100) - Wrong key, skips
  ✗ ViewSpecificKeyHandler (priority 200) - Not handled in this view
  ✓ GlobalNavigationHandler (priority 900) - Handles it (previous view)
```

### Slash Key
```
/ pressed in Processes view:
  ✗ AICoderKeyHandler (priority 10) - Wrong view, skips
  ✗ QuitHandler (priority 50) - Wrong key, skips
  ✓ SlashCommandHandler (priority 100) - Handles it (opens command palette)
  ✗ ViewSpecificKeyHandler (priority 200) - Never reached
  ✗ GlobalNavigationHandler (priority 900) - Never reached
```

## Implementation Details

### Handler Registration Order
Handlers are registered in `internal/tui/model.go` during initialization:

```go
// Register handlers in any order - they're auto-sorted by priority
keyRegistry.Register(NewGlobalNavigationHandler(viewManager, debugMode))    // Priority 900
keyRegistry.Register(NewQuitHandler())                                      // Priority 50
keyRegistry.Register(NewSlashCommandHandler(openPalette))                   // Priority 100
keyRegistry.Register(NewAICoderKeyHandler(aicodersView))                    // Priority 10
// ... etc
```

The `KeyHandlerRegistry` automatically sorts handlers by priority (lowest number first).

### Adding New Handlers

To add a new keyboard handler:

1. **Choose appropriate priority**:
   - 1-49: Terminal/critical input
   - 50-99: Exit/global system keys
   - 100-199: Global commands (slash, help, etc.)
   - 200-899: View-specific handlers
   - 900+: Navigation and fallback

2. **Implement KeyHandler interface**:
```go
type MyHandler struct{}

func (h *MyHandler) Priority() int {
    return 150 // Choose appropriate priority
}

func (h *MyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
    // Return true if this handler should process this key
}

func (h *MyHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
    // Process the key and return command
}
```

3. **Register in model initialization**:
```go
keyRegistry.Register(NewMyHandler())
```

## Testing Priority Order

### Manual Test Cases

1. **Terminal Input Priority**:
   - [ ] Open AI Coder session
   - [ ] Press Enter to focus terminal
   - [ ] Press Left/Right arrows → Should send to terminal, NOT navigate views
   - [ ] Press Tab → Should send to terminal, NOT cycle views
   - [ ] Press Ctrl+T → Should toggle focus, NOT sent to terminal

2. **Exit Key Priority**:
   - [ ] Press ESC once → Shows confirmation message
   - [ ] Press ESC again within 2 seconds → App quits
   - [ ] In AI Coder terminal focus mode, Ctrl+C → Sent to terminal

3. **Command Palette Priority**:
   - [ ] In any view, press `/` → Opens command palette
   - [ ] In AI Coder terminal (unfocused), press `/` at start → Opens palette
   - [ ] In AI Coder terminal (focused), press `/` → Sent to terminal

4. **Navigation Priority (Lowest)**:
   - [ ] In Processes view, press Left/Right → Navigate views
   - [ ] In AI Coder terminal focus, press Left/Right → Sent to terminal
   - [ ] Press number key 1-7 → Jump to view

## Related Documentation

- [Key Handling Architecture](/docs/architecture/key-handling.md)
- [Quit Confirmation and Terminal Focus](/docs/quit-confirmation-and-terminal-focus.md)
- [AI Coders Feature Guide](/docs/features/ai-coders.md)

## Migration Notes

### For Users
- **Tab key no longer cycles views** - use Left/Right arrows instead
- Number keys (1-7) still work for jumping to specific views
- New Ctrl+T shortcut for terminal focus toggle

### For Developers
If you have custom key handlers, review their priority:
- Ensure priorities don't conflict with system handlers
- Test in all contexts (normal view, terminal focus, etc.)
- Use appropriate priority range for handler type
