# Architecture Decisions: Agentic AI Coders

## Core Architecture Decisions

### 1. AI Coder Service Architecture
**Decision**: Create a new `internal/aicoder` package as the core service layer
**Rationale**: 
- Follows existing package structure (`internal/process`, `internal/mcp`)
- Provides clean separation of concerns
- Enables isolated testing and development
- Maintains architectural consistency

**Trade-offs**:
- ✅ **Pro**: Clear boundaries, testable components, follows Go best practices
- ❌ **Con**: Additional package overhead, potential for over-abstraction
- **Alternative Considered**: Adding to existing process manager - rejected due to scope complexity

### 2. Process Management Integration
**Decision**: Extend existing Process Manager rather than create separate lifecycle manager
**Rationale**:
- AI coders are fundamentally long-running processes
- Reuses existing process monitoring, cleanup, and event integration
- Maintains consistency with other managed processes in Brummer

**Implementation Pattern**:
```go
type AICoderProcess struct {
    *process.Process  // Embed standard process
    WorkspaceDir string
    AIProvider   string
    Status       AICoderStatus
    // AI-specific fields
}
```

### 3. TUI Integration Strategy  
**Decision**: Add new `ViewAICoders` following existing view pattern
**Rationale**:
- Consistent with existing TUI architecture
- Leverages existing view switching, keyboard navigation
- Reuses proven UI components (list, viewport, textinput)

**View Structure**:
```go
const ViewAICoders View = "ai-coders"

type AICoderView struct {
    coderList    list.Model
    detailPanel  viewport.Model
    commandInput textinput.Model
}
```

### 4. MCP Tool Organization
**Decision**: Create AI coder tool namespace with consistent naming
**Tools to Implement**:
- `ai_coder_create` - Launch new AI coder instance
- `ai_coder_list` - List active AI coders with status
- `ai_coder_control` - Send commands to AI coder (pause/resume/stop)
- `ai_coder_workspace` - Access AI coder workspace files
- `ai_coder_status` - Get detailed status and progress
- `ai_coder_logs` - Stream AI coder output and logs

**Rationale**: Follows MCP naming conventions, provides comprehensive control surface

### 5. Workspace Isolation Strategy
**Decision**: Start with directory-based isolation, evolve to container-based
**Phase 1**: Restricted directory access with Go's file path validation
**Phase 2**: Container/chroot integration for enhanced security

**Directory Structure**:
```
~/.brummer/ai-coders/
├── coder-{uuid}/
│   ├── workspace/     # AI coder working directory
│   ├── config.json    # AI coder configuration
│   ├── logs/          # AI coder execution logs
│   └── .gitignore     # Workspace git ignore
```

## Integration Patterns

### 1. Event System Integration
**Pattern**: Extend existing event bus with AI coder events
**New Event Types**:
```go
type AICoderStartedEvent struct {
    CoderID     string
    WorkspaceDir string
    Provider    string
}

type AICoderStatusEvent struct {
    CoderID string
    Status  AICoderStatus
    Message string
}

type AICoderCompletedEvent struct {
    CoderID   string
    Success   bool
    FilesChanged []string
}
```

### 2. Configuration Integration
**Pattern**: Extend existing config system with AI coder section
```toml
[ai_coders]
default_provider = "claude"
max_concurrent = 3
workspace_base_dir = "~/.brummer/ai-coders"
timeout_minutes = 30

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"

[ai_coders.providers.openai]
api_key_env = "OPENAI_API_KEY" 
model = "gpt-4"
```

### 3. State Management Pattern
**Pattern**: Thread-safe state management following process manager pattern
```go
type AICoderManager struct {
    coders map[string]*AICoderProcess
    mu     sync.RWMutex
    eventBus *events.EventBus
}

func (m *AICoderManager) GetCoder(id string) (*AICoderProcess, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    coder, exists := m.coders[id]
    return coder, exists
}
```

## Risk Mitigation Strategies

### 1. Resource Management Risk
**Risk**: AI coders consuming excessive system resources
**Mitigation**:
- Resource limits per AI coder (memory, CPU, file system)
- Maximum concurrent AI coder limits (configurable)
- Automatic cleanup of idle/failed AI coders
- Health monitoring with automatic intervention

### 2. Security Risk
**Risk**: AI coders accessing unauthorized files or systems
**Mitigation**:
- Workspace directory restrictions with path validation
- API key isolation and secure storage
- Network access controls (if applicable)
- Audit logging of all AI coder file operations

### 3. UI Responsiveness Risk
**Risk**: Long-running AI operations blocking TUI
**Mitigation**:
- All AI operations run in background goroutines
- Status updates via event system (non-blocking)
- Progressive loading states and progress indicators
- Cancel/interrupt capability for long operations

### 4. AI Provider Integration Risk
**Risk**: API failures, rate limiting, provider changes
**Mitigation**:
- Multi-provider architecture with fallback capability
- Exponential backoff and retry logic
- Circuit breaker pattern for API failures
- Provider abstraction layer for easy swapping

## Performance Considerations

### 1. Memory Management
**Strategy**: Efficient resource usage for multiple AI coders
- Context pooling for AI provider connections
- Streaming responses to avoid memory buildup
- Garbage collection optimization for long-running processes
- Memory limits with graceful degradation

### 2. Concurrency Design
**Pattern**: Worker pool for AI coder management
```go
type AICoderWorkerPool struct {
    workers    int
    taskQueue  chan AICoderTask
    resultChan chan AICoderResult
}
```

### 3. TUI Performance
**Optimization**: Minimize TUI update frequency
- Debounced status updates (max 10 updates/second)
- Lazy loading of AI coder details
- Efficient list rendering with pagination
- Background data fetching with caching

## Future Evolution Path

### Phase 1: Core Implementation (Current)
- Basic AI coder lifecycle management
- Simple TUI integration
- Directory-based workspace isolation
- Single AI provider support

### Phase 2: Enhanced Features
- Multiple AI provider support
- Container-based workspace isolation
- Advanced TUI features (split panes, real-time updates)
- Collaboration features (shared workspaces)

### Phase 3: Advanced Integration
- Git integration for AI coder operations
- Plugin architecture for custom AI providers
- Distributed AI coder coordination
- Advanced monitoring and analytics

## Technology Stack Decisions

### Core Technologies
- **Go 1.24.2**: Core implementation language
- **BubbleTea v0.25.0**: TUI framework for consistency
- **MCP-Go v0.32.0**: AI tool integration protocol

### AI Provider Integration
- **Multiple Provider Support**: Plugin architecture for flexibility
- **Default Providers**: Claude (Anthropic), GPT-4 (OpenAI)
- **Local Model Support**: Future Ollama integration

### Testing Strategy
- **Unit Tests**: Core logic with >80% coverage
- **Integration Tests**: End-to-end AI coder lifecycle
- **Mock Testing**: AI provider responses for reliable testing
- **Performance Tests**: Resource usage and concurrency limits

## Implementation Sequencing

### Critical Path Dependencies
1. **Core Service Layer** → **Process Integration** → **TUI Integration**
2. **MCP Tools** → **Event System** → **Configuration**
3. **Basic Features** → **Advanced Features** → **Performance Optimization**

### Parallel Development Opportunities
- TUI components can be developed alongside core service
- MCP tools can be implemented independently of TUI
- Configuration system can be developed in parallel with core features
- Testing infrastructure can be built throughout development