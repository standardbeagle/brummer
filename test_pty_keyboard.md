# PTY Terminal Keyboard Testing

## Fixed Issues

1. **Keyboard input capture when PTY terminal is focused** ✅
   - When the PTY terminal is focused, ALL keyboard input (except ESC) is now routed to the PTY
   - Number keys 1-9 no longer switch tabs when PTY is focused
   - The global key handler is completely bypassed when PTY is focused

2. **Proper ESC key handling** ✅
   - ESC key now properly exits PTY focus mode
   - ESC is handled specially even when terminal is focused

3. **Tab navigation disabled when PTY focused** ✅
   - Number keys (1-9) are sent to the PTY instead of switching views
   - Tab key is sent to the PTY instead of cycling views
   - Arrow keys are sent to the PTY instead of navigation

## Testing Instructions

1. Start brummer: `./brum`
2. Navigate to AI Coders view (press 8)
3. Create a new AI coder session: `/ai mock` (for interactive mode)
4. Press Enter to focus the PTY terminal
5. Test the following:
   - Type numbers 1-9: They should appear in the terminal, NOT switch tabs
   - Type regular text: Should appear in the terminal
   - Press Tab: Should insert a tab character, NOT cycle views
   - Press Arrow keys: Should move cursor in terminal, NOT navigate views
   - Press ESC: Should exit PTY focus mode (purple border disappears)

## Implementation Details

The fix involved two main changes:

1. In `model.go`, when PTY is focused, we immediately route ALL key messages to the PTY view:
   ```go
   if m.currentView == ViewAICoders && m.aiCoderPTYView != nil && m.aiCoderPTYView.IsTerminalFocused() {
       // When PTY is focused, ALL keys go to the PTY view
       var cmd tea.Cmd
       m.aiCoderPTYView, cmd = m.aiCoderPTYView.Update(msg)
       return m, cmd
   }
   ```

2. In `ai_coder_pty_view.go`, we handle ESC specially even when focused:
   ```go
   if v.terminalFocused && v.currentSession != nil {
       // Check for ESC key to unfocus terminal
       if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
           v.terminalFocused = false
           return v, nil
       }
       // ... send other keys to PTY
   }
   ```

This ensures that:
- All keyboard input goes to the PTY when focused (fixing the main issue)
- ESC still works to exit focus mode (preserving user control)
- No duplicate key handling occurs