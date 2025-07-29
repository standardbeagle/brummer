# Task: Core AI Coder Service Implementation
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 5 files - Within target range (3-7 files)
**Estimated Time**: 25 minutes - Within target (15-30min)  
**Token Estimate**: 120k tokens - Within target (<150k)
**Complexity Level**: 2 (Moderate) - Service architecture with state management
**Parallelization Benefit**: HIGH - Core foundation enables other tasks
**Atomicity Assessment**: ✅ ATOMIC - Complete service layer implementation
**Boundary Analysis**: ✅ CLEAR - New package with no external dependencies

## Persona Assignment
**Persona**: Senior Software Engineer (Backend)
**Expertise Required**: Go service architecture, concurrent programming, API design
**Worktree**: `~/work/worktrees/agentic-ai-coders/01-core-service/`

## Context Summary
**Risk Level**: MEDIUM (new architecture, concurrency requirements)
**Integration Points**: Process manager, MCP tools, TUI (future tasks)
**Architecture Pattern**: Service Manager Pattern (from process manager)
**Similar Reference**: `internal/process/manager.go` - Thread-safe manager implementation

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [internal/process/manager.go, pkg/events/events.go]
modify_files: []
create_files: [
  /internal/aicoder/manager.go,
  /internal/aicoder/process.go, 
  /internal/aicoder/types.go,
  /internal/aicoder/provider.go,
  /internal/aicoder/workspace.go
]
# Total: 5 files - appropriate for atomic service layer
```

**Existing Patterns to Follow**:
- `internal/process/manager.go` - Thread-safe manager with RWMutex pattern
- `pkg/events/events.go` - Event emission patterns for status updates
- `internal/mcp/server.go` - Service lifecycle and configuration patterns

**Dependencies Context**:
- `github.com/google/uuid v1.6.0` - For unique AI coder instance IDs
- `golang.org/x/sync v0.15.0` - For advanced concurrency patterns if needed
- Standard library only for core implementation

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/aicoder/manager.go           # Core AI coder manager service
  - /internal/aicoder/process.go           # AI coder process wrapper
  - /internal/aicoder/types.go             # Type definitions and constants
  - /internal/aicoder/provider.go          # AI provider abstraction interface
  - /internal/aicoder/workspace.go         # Workspace management utilities

direct_dependencies: []                    # No existing files to modify
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/process/manager.go           # Review for pattern consistency
  - /pkg/events/events.go                  # Review for event pattern alignment
  - /internal/config/config.go             # Review for future config integration

check_documentation:
  - /docs/architecture.md                  # Review for architectural alignment
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/tui/                         # TUI integration is separate task
  - /internal/mcp/tools.go                 # MCP tools are separate task
  - /internal/proxy/                       # Proxy system unrelated
  - /internal/logs/                        # Log system integration later
  - /cmd/                                  # CLI integration later

ignore_search_patterns:
  - "**/test/**"                          # Test files handled separately
  - "**/testdata/**"                      # Test data unrelated
  - "**/vendor/**"                        # Third-party code
```

**Boundary Analysis Results**:
- **Usage Count**: 0 occurrences (new package)
- **Scope Assessment**: LIMITED scope - new isolated package
- **Impact Radius**: 0 files in MODIFY zone initially, clean foundation

### External Context Sources (from master research)
**Primary Documentation**:
- [Effective Go](https://golang.org/doc/effective_go.html) - Service design patterns, error handling
- [Go Concurrency Patterns](https://blog.golang.org/pipelines) - Worker pools and pipeline patterns
- [Go Project Layout](https://github.com/golang-standards/project-layout) - Package organization

**Standards Applied**:
- Go naming conventions - AICoderManager, AICoderProcess types
- Error handling - Explicit error returns with context
- Concurrency - RWMutex for thread-safe operations

**Reference Implementation**:
- `internal/process/manager.go` - Manager pattern with lifecycle management
- Service initialization, thread-safe state, event integration

## Task Requirements
**Objective**: Implement core AI coder service layer with complete lifecycle management

**Success Criteria**:
- [ ] AICoderManager service with thread-safe operations (following process manager pattern)
- [ ] AICoderProcess type with status tracking and workspace management
- [ ] Provider abstraction interface supporting multiple AI providers
- [ ] Workspace isolation utilities with security path validation
- [ ] Complete type system with status enums and error types
- [ ] Event integration for status broadcasting
- [ ] Configuration support for AI coder settings

**Validation Commands**:
```bash
# Pattern Application Verification
grep -q "sync.RWMutex" internal/aicoder/manager.go     # Thread safety pattern
grep -q "context.Context" internal/aicoder/manager.go  # Context pattern applied
go build ./internal/aicoder                            # Package compiles
go test ./internal/aicoder -v                          # Basic tests pass
go vet ./internal/aicoder                              # Go vet passes
```

## Implementation Specifications

### Core Types Implementation
```go
// internal/aicoder/types.go
type AICoderStatus string

const (
    StatusCreating   AICoderStatus = "creating"
    StatusRunning    AICoderStatus = "running" 
    StatusPaused     AICoderStatus = "paused"
    StatusCompleted  AICoderStatus = "completed"
    StatusFailed     AICoderStatus = "failed"
    StatusStopped    AICoderStatus = "stopped"
)

type AICoderProcess struct {
    ID           string
    Name         string
    Provider     string
    WorkspaceDir string
    Status       AICoderStatus
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Task         string
    Progress     float64
    cancel       context.CancelFunc
    mu           sync.RWMutex
}
```

### Manager Implementation
```go
// internal/aicoder/manager.go - Follow process manager pattern
type AICoderManager struct {
    coders   map[string]*AICoderProcess
    mu       sync.RWMutex
    eventBus EventBus  // Interface to be defined
    config   *Config   // Configuration interface
}

// Core operations following existing patterns
func (m *AICoderManager) CreateCoder(ctx context.Context, req CreateCoderRequest) (*AICoderProcess, error)
func (m *AICoderManager) GetCoder(id string) (*AICoderProcess, bool)
func (m *AICoderManager) ListCoders() []*AICoderProcess
func (m *AICoderManager) DeleteCoder(id string) error
```

### Provider Abstraction
```go
// internal/aicoder/provider.go
type AIProvider interface {
    Name() string
    GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error)
    StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error)
    ValidateConfig(config ProviderConfig) error
}

type GenerateOptions struct {
    Model       string
    MaxTokens   int
    Temperature float64
    WorkspaceContext []string
}
```

### Workspace Management
```go
// internal/aicoder/workspace.go
type WorkspaceManager struct {
    baseDir string
}

func (w *WorkspaceManager) CreateWorkspace(coderID string) (string, error)
func (w *WorkspaceManager) ValidatePath(workspaceDir, requestedPath string) error  
func (w *WorkspaceManager) WriteFile(workspaceDir, relativePath string, content []byte) error
func (w *WorkspaceManager) ReadFile(workspaceDir, relativePath string) ([]byte, error)
func (w *WorkspaceManager) ListFiles(workspaceDir string) ([]string, error)
func (w *WorkspaceManager) CleanupWorkspace(workspaceDir string) error
```

## Risk Mitigation (from master analysis)
**Medium-Risk Mitigations**:
- Concurrency safety - Use RWMutex pattern from process manager - Testing: unit tests with race detection
- Provider abstraction - Interface design allows easy provider swapping - Fallback: Mock provider implementation
- Workspace security - Path validation prevents directory traversal - Testing: security-focused path validation tests

**Context Validation**:
- [ ] Thread-safe patterns from `internal/process/manager.go` successfully adapted
- [ ] Event emission patterns from `pkg/events/events.go` properly implemented
- [ ] Service lifecycle follows established Brummer patterns

## Integration with Other Tasks
**Dependencies**: None (foundation task)
**Integration Points**: 
- Task 02 (MCP Tools) will consume AICoderManager interface
- Task 04 (TUI Integration) will use AICoderManager for UI data
- Task 05 (Process Integration) will extend AICoderProcess lifecycle

**Shared Context**: Core types and interfaces will be used by all subsequent tasks

## Execution Notes
- **Start Pattern**: Use `internal/process/manager.go` as implementation template for thread safety
- **Key Context**: Focus on clean interfaces for future integration points
- **Security Focus**: Implement workspace path validation with comprehensive test cases
- **Review Focus**: Thread safety implementation and interface design for extensibility

## Additional Implementation Guidelines

### Error Handling Strategy
```go
// Define AI coder specific errors
type AICoderError struct {
    CoderID string
    Op      string  
    Err     error
}

func (e *AICoderError) Error() string {
    return fmt.Sprintf("ai coder %s: %s: %v", e.CoderID, e.Op, e.Err)
}
```

### Configuration Interface
```go
// Configuration interface for dependency injection
type Config interface {
    GetAICoderConfig() AICoderConfig
}

type AICoderConfig struct {
    MaxConcurrent    int
    WorkspaceBaseDir string
    DefaultProvider  string
    TimeoutMinutes   int
}
```

### Event Integration Interface
```go
// Event bus interface for dependency injection
type EventBus interface {
    Emit(eventType string, data interface{})
}

// Event types to emit
type AICoderEvent struct {
    Type     string    `json:"type"`
    CoderID  string    `json:"coder_id"`
    Status   string    `json:"status"`
    Message  string    `json:"message"`
    Time     time.Time `json:"time"`
}
```

This task creates the foundational service layer that all other AI coder features will build upon, following established Brummer patterns for consistency and maintainability.