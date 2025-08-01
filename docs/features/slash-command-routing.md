# Slash Command Routing

## Problem Statement

Brummer uses slash commands (e.g., `/run`, `/ai`, `/help`) for system operations, but when AI coder sessions are active, there's a conflict: should the "/" key open Brummer's command palette or be sent to the AI coder as input?

## Solution: Context-Aware Routing

We implemented a context-aware input routing system that intelligently determines where "/" commands should be routed based on the current interaction context.

## Routing Logic

### Decision Tree
```
User presses "/"
    ↓
Is AI Coder view active?
    ↓ No → Route to Brummer command palette
    ↓ Yes
        ↓
    Is terminal focused?
        ↓ No → Route to Brummer command palette  
        ↓ Yes
            ↓
        Is cursor at start of line?
            ↓ Yes → Route to Brummer command palette
            ↓ No → Route to AI coder as input
```

### Context Rules

1. **Terminal Not Focused**: Always route to Brummer commands
   - User is navigating the UI, not typing in AI session
   - Preserves expected Brummer behavior

2. **No Active Session**: Always route to Brummer commands
   - No AI coder to send input to
   - Falls back to normal Brummer operation

3. **Cursor at Start of Line**: Route to Brummer commands
   - Typically indicates user wants to start a new command
   - Matches shell-like behavior where commands start lines
   - Allows access to Brummer functionality even when AI focused

4. **Cursor Mid-Line**: Route to AI coder
   - User is likely typing a message to the AI
   - "/" could be part of file paths, URLs, or regular text
   - Preserves natural typing flow

## Implementation Details

### Core Components

#### PTYSession Methods (`internal/aicoder/pty_session.go`)
```go
// IsAtStartOfLine returns true if cursor is at beginning of line
func (s *PTYSession) IsAtStartOfLine() bool

// GetCurrentLineContent returns current line content up to cursor  
func (s *PTYSession) GetCurrentLineContent() string
```

#### AICoderPTYView Methods (`internal/tui/ai_coder_pty_view.go`)
```go
// ShouldInterceptSlashCommand determines routing behavior
func (v *AICoderPTYView) ShouldInterceptSlashCommand() bool
```

#### Main Model Logic (`internal/tui/model.go`)
```go
// Check slash command interception before PTY routing
if msg.String() == "/" && m.width > 0 && m.height > 0 {
    shouldIntercept := true
    if m.currentView == ViewAICoders && m.aiCoderPTYView != nil {
        shouldIntercept = m.aiCoderPTYView.ShouldInterceptSlashCommand()
    }
    
    if shouldIntercept {
        m.showCommandWindow()
        return m, nil
    }
    // Fall through to PTY handling
}
```

### Terminal State Detection

Uses the `vt10x.Terminal` emulator to access cursor position:
- `cursor.X == 0`: Cursor at start of line
- `cursor.Y`: Current line number
- `Terminal.String()`: Full terminal content

## User Experience

### Visual Feedback
- Help text shows: `"/ (start of line): Brummer Commands"`
- Extended help explains the routing behavior
- Status messages indicate when commands are intercepted

### Keyboard Shortcuts
- `Ctrl+H`: Toggle help (shows routing explanation)
- `Ctrl+Q`: Unfocus terminal (enables Brummer commands)
- `Enter`: Focus terminal (enables AI input)

## Testing

### Unit Tests (`internal/tui/slash_command_routing_test.go`)
- Tests basic routing logic for different contexts
- Verifies behavior when terminal focused/unfocused
- Validates session state handling

### Integration Testing
- Manual testing with real AI coder sessions
- Verification of cursor position detection
- End-to-end slash command behavior

## Benefits

1. **Intuitive Behavior**: Matches user expectations based on context
2. **No Lost Functionality**: Both Brummer and AI commands remain accessible
3. **Shell-like Feel**: Commands at start of line feel natural
4. **Graceful Fallback**: Always allows Brummer access when needed
5. **Visual Clarity**: Clear indication of current routing behavior

## Future Enhancements

1. **Custom Escape Sequences**: Alternative ways to access Brummer commands
2. **Configurable Routing**: User preferences for routing behavior
3. **Command History**: Context-aware command completion
4. **Multi-AI Sessions**: Extended routing for multiple AI contexts