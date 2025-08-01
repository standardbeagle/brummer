# Agentic AI Coders Feature (Tmux-Style Design)

## Overview
Brummer will support running AI coding assistants as interactive sessions within the TUI, similar to tmux sessions. This creates a seamless development experience where AI coders receive real-time feedback from Brummer's build system, test runners, and error detection.

## Core Concepts

### Interactive AI Coder Sessions
- **Tmux-style attach/detach**: AI coders run in persistent sessions that users can attach to and detach from
- **Direct interaction**: Users can type commands and see AI responses in real-time within the Brummer TUI
- **Session persistence**: AI coders continue working even when detached, similar to tmux sessions
- **Multiple sessions**: Support for running multiple AI coders simultaneously with easy switching

### Real-Time Brummer Integration
- **Error reporting**: Build errors, test failures, and lint warnings are automatically sent to the AI
- **Context awareness**: AI receives file changes, process outputs, and system events
- **Proactive assistance**: AI can suggest fixes before the user even asks
- **Workspace isolation**: Each AI coder operates in its own sandboxed workspace

## User Interaction Model

### Starting an AI Coder
```bash
# Via TUI
Press 'a' to switch to AI Coders view
Press 'n' to create new AI coder
Select provider (Claude, GPT-4, etc.)
Enter initial task/prompt

# Via MCP
ai_coder_create {"task": "implement user auth", "provider": "claude"}
```

### Interacting with AI Coders
```
┌─────────────────────────────────────────────────────────────┐
│ AI Coder: implement-auth (Claude) - Running                 │
├─────────────────────────────────────────────────────────────┤
│ [AI] I'll implement user authentication. Starting with...   │
│                                                             │
│ [Brummer] Build error in auth.go:15                       │
│   undefined: bcrypt.GenerateFromPassword                   │
│                                                             │
│ [AI] I see the issue. We need to import bcrypt. Let me... │
│                                                             │
│ [User] > also add JWT token generation                     │
│                                                             │
│ [AI] Sure! I'll add JWT token generation to the auth...   │
├─────────────────────────────────────────────────────────────┤
│ Progress: ████████░░ 80% | Files: 5 | Tokens: 12.5k       │
└─────────────────────────────────────────────────────────────┘
```

### Session Management Commands
- `Ctrl+b d` - Detach from current AI coder (continues running)
- `Ctrl+b n` - Next AI coder session
- `Ctrl+b p` - Previous AI coder session  
- `Ctrl+b w` - List all AI coder sessions
- `Ctrl+b c` - Create new AI coder
- `Ctrl+b &` - Kill current AI coder session

## Brummer Hooks and Events

### Automatic Error Reporting
When Brummer detects errors, they're automatically sent to attached AI coders:

```go
// Build error detected
Brummer → AI: "Build failed: undefined variable 'user' at handlers.go:45"
AI → User: "I see there's an undefined variable. Let me check the context..."

// Test failure
Brummer → AI: "Test failed: TestUserLogin - expected 200, got 401"
AI → User: "The login test is failing. This might be due to..."

// Lint warning
Brummer → AI: "Lint: exported function CreateUser should have comment"
AI → User: "I'll add the missing documentation comment..."
```

### Event Hooks Integration
```yaml
AI Coder Hooks:
  - process_failed: Send stdout/stderr to AI
  - build_error: Send error details and file context
  - test_failed: Send test output and failure reason
  - lint_warning: Send lint messages
  - file_changed: Notify AI of external file modifications
```

## Implementation Architecture

### Component Structure
```
/internal/aicoder/
├── manager.go         # AI coder lifecycle management
├── session.go         # Interactive session handling (PTY)
├── provider.go        # AI provider interface (Claude, GPT-4, etc.)
├── workspace.go       # Isolated workspace management
└── hooks.go          # Brummer event integration

/internal/tui/
└── ai_coder_view.go  # Tmux-style TUI view

/internal/mcp/
└── ai_coder_tools.go # MCP tools for external control
```

### Session Lifecycle
1. **Create**: Initialize AI coder with provider and task
2. **Attach**: Connect to PTY for interactive communication
3. **Process**: Handle user input and AI responses
4. **Hook Events**: Receive and process Brummer events
5. **Detach**: Disconnect while keeping session alive
6. **Reattach**: Reconnect to existing session
7. **Terminate**: Clean up resources and workspace

## Example Workflows

### Debugging with AI Assistance
```
User: [Runs tests, sees failure]
Brummer → AI: "Test TestAPIAuth failed: token validation error"
AI: "I see the token validation is failing. Let me check the JWT implementation..."
AI: [Makes code changes]
Brummer → AI: "Build successful, running tests..."
Brummer → AI: "All tests passing"
AI: "Great! The authentication is now working correctly."
```

### Collaborative Feature Development
```
User: "Implement rate limiting for the API"
AI: "I'll implement rate limiting. Which approach would you prefer?"
User: "Use Redis-based sliding window"
AI: [Starts implementation]
Brummer → AI: "Build error: Redis client not found"
AI: "We need to add the Redis dependency. Let me update go.mod..."
```

## Configuration

```toml
[aicoder]
default_provider = "claude"
max_concurrent = 3
workspace_dir = "~/.brummer/ai-workspaces"

[aicoder.providers.claude]
api_key_env = "CLAUDE_API_KEY"
model = "claude-3-sonnet"
max_tokens = 100000

[aicoder.hooks]
enabled = true
include_build_errors = true
include_test_failures = true
include_lint_warnings = true
context_lines = 10
```

## Security and Isolation

- **Workspace Isolation**: Each AI coder operates in a separate directory
- **Resource Limits**: CPU, memory, and disk usage limits per AI coder
- **File Access**: Restricted to workspace and project directories only
- **Network Access**: Configurable restrictions on external API calls
- **Code Execution**: Optional sandboxing for running generated code

## Future Enhancements

See the [main Brummer Roadmap](/docs/ROADMAP.md) for comprehensive feature plans and timelines.

### Phase 3: AI Coders Enhancement (Q2 2025)
- **Multi-AI Collaboration**: Multiple AI coders working on different parts of the same feature
- **Knowledge Persistence**: AI coders maintaining context across sessions
- **Custom Tools Integration**: Ability to give AI coders access to specific development tools

### Phase 4: Environment & Configuration Management (Q2-Q3 2025)
- **Environment Variable Management**: Unified .env file management with TUI interface
- **Secrets Management**: Encrypted environment variables
- **AI Integration**: Suggest appropriate environment configurations

### Phase 6: Team Collaboration (Q4 2025)
- **Session Sharing**: Share AI coder sessions with team members
- **Collaborative Debugging**: Multiple developers on same session

For detailed specifications and implementation plans, see [/docs/ROADMAP.md](/docs/ROADMAP.md).