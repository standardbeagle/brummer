# Codebase Context Documentation: Agentic AI Coders

## Existing Architecture Patterns

### Service Architecture
- **Modular Go Architecture**: Clean separation of concerns across `internal/` packages
- **TUI Framework**: Uses BubbleTea framework (`github.com/charmbracelet/bubbletea v0.25.0`) for terminal interface
- **MCP Integration**: Full Model Context Protocol support via `github.com/mark3labs/mcp-go v0.32.0`
- **Event-Driven**: Central event bus architecture for component communication
- **Files**: `internal/tui/model.go`, `internal/mcp/server.go`, `pkg/events/events.go`

### Data Layer
- **Event Bus**: Central event system for inter-component communication
- **Log Store**: Centralized log storage with filtering and search capabilities
- **Process State Management**: Thread-safe process lifecycle tracking
- **Files**: `pkg/events/events.go`, `internal/logs/`, `internal/process/manager.go`

### TUI Architecture Patterns
- **View-Based Architecture**: Multiple views with clear separation (`ViewScripts`, `ViewProcesses`, `ViewLogs`, etc.)
- **BubbleTea Model**: Uses Elm architecture pattern (Model-View-Update)
- **Component Composition**: Reusable UI components (viewport, textinput, list)
- **Files**: `internal/tui/model.go` (lines 31-45 show View constants), `internal/tui/script_selector.go`

### MCP Integration Patterns
- **Streamable HTTP Server**: Supports both JSON-RPC and Server-Sent Events
- **Tool Registration**: Dynamic tool registration with metadata
- **Session Management**: Per-client session tracking and context
- **Hub Mode**: Multi-instance coordination capabilities
- **Files**: `internal/mcp/streamable_server.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`

### Process Management
- **Manager Pattern**: Centralized process lifecycle management
- **Thread-Safe Operations**: RWMutex protection for concurrent access
- **Context-Based Cancellation**: Proper cleanup and graceful shutdown
- **Event Integration**: Process events broadcast via event bus
- **Files**: `internal/process/manager.go`, `internal/process/manager_unix.go`

## Similar Feature Implementations

### TUI View Management
- **File**: `internal/tui/model.go` - Lines 31-45 define view constants
- **Pattern**: Enum-based view switching with centralized model state
- **Relevance**: New AI coder view would follow same pattern (`ViewAICoders`)

### MCP Tool Integration
- **File**: `internal/mcp/tools.go` - Tool registration and execution patterns
- **Pattern**: Tool metadata with handler functions and validation
- **Relevance**: AI coder tools would register as new MCP tool category

### Process Spawning & Management
- **File**: `internal/process/manager.go` - Process lifecycle management
- **Pattern**: Context-based process creation with event emission
- **Relevance**: AI coders would be managed as specialized processes

### Event-Driven Communication
- **File**: `pkg/events/events.go` - Central event bus architecture
- **Pattern**: Typed events with async handling and worker pools  
- **Relevance**: AI coder status updates would use same event patterns

## Dependency Analysis

### Core Dependencies
- **BubbleTea v0.25.0** - [docs](https://github.com/charmbracelet/bubbletea) - TUI framework, Model-View-Update pattern
- **MCP-Go v0.32.0** - [docs](https://github.com/mark3labs/mcp-go) - Model Context Protocol implementation
- **Lipgloss v0.10.0** - [docs](https://github.com/charmbracelet/lipgloss) - Styling and layout for TUI components
- **Cobra v1.8.0** - [docs](https://github.com/spf13/cobra) - CLI command structure

### Dev Dependencies  
- **Testify v1.10.0** - Testing framework with assertions and mocks
- **FSNotify v1.9.0** - File system event monitoring (used in discovery)
- **Gorilla WebSocket v1.5.3** - WebSocket support for real-time communication

### External Services
- **AI Provider APIs** - Would need integration with Claude, OpenAI, or other AI services
- **File System** - Direct file manipulation for AI coder workspaces
- **Git Integration** - Version control integration for AI coder operations

## File Dependency Mapping

```yaml
high_change_areas:
  - /internal/tui/: [new AI coder view, model updates, UI components]
  - /internal/mcp/: [new MCP tools for AI coder interaction]
  - /internal/ai-coder/: [new service package for AI coder management]
  - /internal/process/: [process manager extensions for AI coders]

medium_change_areas:
  - /pkg/events/: [new event types for AI coder status]
  - /internal/config/: [AI coder configuration settings]
  - /cmd/: [CLI integration for AI coder commands]

low_change_areas:
  - /internal/logs/: [log integration for AI coder output]
  - /internal/proxy/: [potential proxy integration]
  - /docs/: [documentation updates]
```

## Integration Points Analysis

### TUI Integration Requirements
- **New View Type**: `ViewAICoders` constant addition
- **Model Extensions**: State management for AI coder instances
- **UI Components**: List view for AI coders, detail panels, control interfaces
- **Navigation**: Tab switching and keyboard shortcuts

### MCP Tool Extensions
- **AI Coder Management Tools**: `ai_coder_create`, `ai_coder_list`, `ai_coder_control`
- **Workspace Tools**: `ai_coder_workspace`, `ai_coder_files`
- **Status Tools**: `ai_coder_status`, `ai_coder_logs`

### Process Management Extensions
- **AI Coder Process Type**: New process category with specialized handling
- **Workspace Isolation**: Directory sandboxing and resource limits
- **Communication Channels**: IPC for AI coder commands and responses

### Event System Extensions
- **New Event Types**: `AICoderStarted`, `AICoderCompleted`, `AICoderError`
- **Status Broadcasting**: Real-time status updates to TUI
- **Progress Tracking**: Task completion and file modification events

## Architecture Constraints & Considerations

### Thread Safety Requirements
- **Pattern**: All AI coder state must use RWMutex protection (following `internal/process/manager.go` pattern)
- **Event Bus**: Asynchronous event handling for AI coder status updates
- **Shared Resources**: File system locks for workspace isolation

### Resource Management
- **Memory Limits**: AI coders may consume significant memory for context
- **File System**: Workspace quotas and cleanup procedures needed
- **Process Counts**: Limits on concurrent AI coder instances

### Security Considerations
- **Sandbox Isolation**: AI coders must be restricted to designated workspaces
- **API Key Management**: Secure storage and access for AI provider credentials
- **File Access Control**: Prevent unauthorized file system access

### Performance Requirements
- **TUI Responsiveness**: AI coder operations must not block TUI updates
- **Async Operations**: Long-running AI tasks need background processing
- **Resource Monitoring**: CPU and memory usage tracking for AI processes