# Quit Confirmation and Terminal Focus Improvements

## Overview

This document describes two UX improvements added to Brummer's TUI:

1. **Double-Press Quit Confirmation** - Prevents accidental application exit
2. **Ctrl+T Terminal Focus Toggle** - Quick keyboard shortcut for AI Coder terminal focus

## Features Implemented

### 1. Double-Press Quit Confirmation

**Problem**: Users were accidentally exiting the application by pressing ESC or Ctrl+C once, causing loss of work and context.

**Solution**: Require confirmation for quit commands:
- First press of ESC or Ctrl+C shows warning: "Press ESC or Ctrl+C again to quit"
- Second press within 2 seconds actually quits the application
- After 2 second timeout, the confirmation state resets (requires two presses again)
- Warning messages appear in the system notification area

**Implementation Details**:
- **File**: `internal/tui/model.go`
  - Added `lastQuitAttemptTime time.Time` field
  - Added `awaitingQuitConfirmation bool` field

- **File**: `internal/tui/key_handler.go`
  - Updated `QuitHandler` to implement double-press protection
  - Returns `StatusMsg` with confirmation prompt on first press
  - Checks elapsed time on second press
  - Prevents quit keys from being handled when in AI Coder terminal focus mode

- **File**: `internal/tui/system_message_handler.go` (already existed)
  - Handles `StatusMsg` messages containing "quit"
  - Displays them in the system component for visibility

**Code Flow**:
```
User presses ESC/Ctrl+C
  ↓
QuitHandler.Handle()
  ↓
if awaitingQuitConfirmation == false:
    Set awaitingQuitConfirmation = true
    Record lastQuitAttemptTime = now
    Return StatusMsg("Press ESC or Ctrl+C again to quit")
  ↓
if awaitingQuitConfirmation == true:
    Check if (now - lastQuitAttemptTime) < 2 seconds:
        YES → Return tea.Quit (actually quit)
        NO  → Reset confirmation state, show message again
```

### 2. Ctrl+T Terminal Focus Toggle

**Problem**: Users had to press Enter to enable terminal focus mode, but there was no quick toggle key. ESC released focus but also exited the PTY view.

**Solution**: Added Ctrl+T as a dedicated toggle for terminal focus mode:
- Press `Ctrl+T` to **toggle** terminal focus on/off
- Works in both focused and unfocused states
- Shows status message indicating current state
- Does not exit the PTY view (unlike ESC)

**Key Bindings Summary**:
| Key | Unfocused Mode | Focused Mode |
|-----|----------------|--------------|
| `Enter` | Enable focus + send Enter to terminal | Send Enter to terminal |
| `Ctrl+T` | **Toggle focus ON** | **Toggle focus OFF** |
| `ESC` | Exit to session list | Release focus |
| Other keys | Sent to terminal | Sent to terminal |

**Implementation Details**:
- **File**: `internal/tui/key_handler.go`
  - Updated `AICoderKeyHandler.CanHandle()` to always handle `tea.KeyCtrlT` in AI Coders view
  - Added Ctrl+T handling logic in `AICoderKeyHandler.Handle()`:
    - Toggles `aiState.TerminalFocused` state
    - Returns `StatusMsg` with current focus state
    - Prevents Ctrl+T from being sent to the terminal

- **File**: `internal/tui/components/aicoders_view.go`
  - Updated footer messages to indicate Ctrl+T availability:
    - Unfocused: "Enter or Ctrl+T=Focus | ESC=List | ?=Help"
    - Focused: "🔒 TERMINAL FOCUS MODE - Ctrl+T or ESC to release"

**Status Messages**:
- Focus ON: "Terminal focus: ON (all keys go to terminal, Ctrl+T or ESC to release)"
- Focus OFF: "Terminal focus: OFF (ESC to exit, Enter to focus)"
- ESC release: "Terminal focus released"
- Enter enable: "Terminal focus enabled (Ctrl+T or ESC to release)"

## User Experience Improvements

### Before Changes
| Issue | Impact |
|-------|--------|
| Accidental ESC press exits app | Lost work, frustration |
| No quick toggle for terminal focus | Had to ESC out, then Enter back in |
| Ctrl+C always exits app | Couldn't send Ctrl+C to AI terminal |

### After Changes
| Improvement | Benefit |
|------------|---------|
| Double-press quit confirmation | Prevents accidental exits |
| Ctrl+T toggle | Quick focus control without exiting |
| Context-aware quit handling | Ctrl+C in focus mode goes to terminal |

## Testing

### Manual Testing Checklist

#### Quit Confirmation
- [ ] Press ESC once → Shows "Press ESC or Ctrl+C again to quit"
- [ ] Press ESC again within 2 seconds → Application quits
- [ ] Press ESC, wait 3 seconds, press ESC → Shows confirmation again
- [ ] Press Ctrl+C once → Shows confirmation message
- [ ] Press Ctrl+C twice quickly → Application quits
- [ ] Mix ESC and Ctrl+C → Both count toward confirmation

#### Terminal Focus Toggle (AI Coders View)
- [ ] In unfocused PTY mode, press Ctrl+T → Focus enabled
- [ ] In focused mode, press Ctrl+T → Focus disabled
- [ ] In focused mode, Ctrl+T does NOT get sent to terminal
- [ ] Status messages appear for all focus changes
- [ ] Footer updates to show current mode
- [ ] Enter still works as before (focus + send to terminal)
- [ ] ESC still works as before (release focus or exit)

### Build Verification
```bash
make build
# ✅ Build complete: ./brum
```

## Implementation Files Modified

1. **internal/tui/model.go**
   - Added quit confirmation state fields

2. **internal/tui/key_handler.go**
   - Implemented double-press quit logic in `QuitHandler`
   - Added Ctrl+T toggle in `AICoderKeyHandler`
   - Added time import

3. **internal/tui/components/aicoders_view.go**
   - Updated footer help text to mention Ctrl+T

4. **internal/tui/system_message_handler.go**
   - Already handled `StatusMsg` for quit-related messages (no changes needed)

## Related Documentation

- [AI Coders Feature Guide](/docs/features/ai-coders.md)
- [Key Handling Architecture](/docs/architecture/key-handling.md)
- [AI Coder Configuration](/docs/ai-coder-configuration.md)

## Future Enhancements

Possible improvements for consideration:
- [ ] Make quit confirmation timeout configurable (currently 2 seconds)
- [ ] Add visual indicator in header/footer when awaiting quit confirmation
- [ ] Add Ctrl+T to other views for consistent toggle behavior
- [ ] Persist terminal focus preference across sessions
