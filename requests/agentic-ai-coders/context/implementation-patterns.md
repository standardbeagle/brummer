# Implementation Patterns: Agentic AI Coders

## Code Patterns from Existing Codebase

### 1. Service Manager Pattern (from Process Manager)
**Reference**: `internal/process/manager.go`
**Pattern**: Centralized manager with thread-safe operations
```go
// Follow this pattern for AICoderManager
type AICoderManager struct {
    coders   map[string]*AICoderProcess
    mu       sync.RWMutex
    eventBus *events.EventBus
    config   *config.Config
}

// Thread-safe operations pattern
func (m *AICoderManager) GetCoder(id string) (*AICoderProcess, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    coder, exists := m.coders[id]
    return coder, exists
}
```

### 2. TUI View Pattern (from TUI Model)
**Reference**: `internal/tui/model.go` lines 31-45
**Pattern**: View constants and configuration structure
```go
// Add to existing View constants
const ViewAICoders View = "ai-coders"

// Follow ViewConfig pattern
var aiCoderViewConfig = ViewConfig{
    Title:       "AI Coders",
    Description: "Manage and monitor agentic AI coding assistants",
    KeyMap:      aiCoderKeyMap,
}
```

### 3. MCP Tool Registration Pattern (from MCP Tools)
**Reference**: `internal/mcp/tools.go`
**Pattern**: Tool metadata with handler functions
```go
// Follow this tool registration pattern
func registerAICoderTools(server *server.MCPServer) {
    server.RegisterTool("ai_coder_create", mcp.Tool{
        Name:        "ai_coder_create",
        Description: "Create and launch a new AI coder instance",
        InputSchema: aiCoderCreateSchema,
    }, handleAICoderCreate)
}

func handleAICoderCreate(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
    // Implementation follows existing tool handler pattern
}
```

### 4. Event Emission Pattern (from Event Bus)  
**Reference**: `pkg/events/events.go`
**Pattern**: Typed events with async handling
```go
// Define AI coder events following existing pattern
type AICoderEvent struct {
    Type     string    `json:"type"`
    CoderID  string    `json:"coder_id"`
    Status   string    `json:"status"`
    Message  string    `json:"message"`
    Time     time.Time `json:"time"`
}

// Emit events following existing pattern
func (m *AICoderManager) emitEvent(event AICoderEvent) {
    m.eventBus.Emit("ai_coder", event)
}
```

### 5. Process Lifecycle Pattern (from Process Manager)
**Reference**: `internal/process/manager.go`
**Pattern**: Context-based process management with cleanup
```go
// Follow process creation pattern
func (m *AICoderManager) CreateCoder(ctx context.Context, req CreateCoderRequest) (*AICoderProcess, error) {
    // Generate unique ID (follow existing pattern)
    coderID := fmt.Sprintf("ai-coder-%d", time.Now().Unix())
    
    // Create workspace directory
    workspaceDir := filepath.Join(m.config.WorkspaceBaseDir, coderID)
    if err := os.MkdirAll(workspaceDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create workspace: %w", err)
    }
    
    // Create context with cancellation (follow existing pattern)
    ctx, cancel := context.WithCancel(ctx)
    
    coder := &AICoderProcess{
        ID:           coderID,
        WorkspaceDir: workspaceDir,
        Status:       StatusCreating,
        CreatedAt:    time.Now(),
        cancel:       cancel,
    }
    
    // Register with manager
    m.mu.Lock()
    m.coders[coderID] = coder
    m.mu.Unlock()
    
    // Emit creation event
    m.emitEvent(AICoderEvent{
        Type:    "created",
        CoderID: coderID,
        Time:    time.Now(),
    })
    
    return coder, nil
}
```

## Naming Conventions and Style Guidelines

### Package Naming
- **Package Name**: `aicoder` (single word, lowercase)
- **Import Path**: `github.com/standardbeagle/brummer/internal/aicoder`
- **Rationale**: Follows Go naming conventions, avoids mixed case

### Type Naming
- **Manager Type**: `AICoderManager` (exported, clear purpose)
- **Process Type**: `AICoderProcess` (exported, embedded process.Process)
- **Status Type**: `AICoderStatus` (exported enum)
- **Event Types**: `AICoderEvent`, `AICoderStatusEvent` (descriptive, consistent)

### Function Naming
- **CRUD Operations**: `CreateCoder`, `GetCoder`, `ListCoders`, `DeleteCoder`
- **Control Operations**: `StartCoder`, `StopCoder`, `PauseCoder`
- **Status Operations**: `GetCoderStatus`, `UpdateCoderStatus`
- **Workspace Operations**: `GetWorkspaceFiles`, `WriteWorkspaceFile`

### Constants and Enums
```go
// Status constants follow existing pattern
type AICoderStatus string

const (
    StatusCreating   AICoderStatus = "creating"
    StatusRunning    AICoderStatus = "running"
    StatusPaused     AICoderStatus = "paused"
    StatusCompleted  AICoderStatus = "completed"
    StatusFailed     AICoderStatus = "failed"
    StatusStopped    AICoderStatus = "stopped"
)
```

## Error Handling Patterns

### Error Types (following existing patterns)
```go
// Define specific error types
type AICoderError struct {
    CoderID string
    Op      string
    Err     error
}

func (e *AICoderError) Error() string {
    return fmt.Sprintf("ai coder %s: %s: %v", e.CoderID, e.Op, e.Err)
}

// Wrap errors with context
func (m *AICoderManager) StartCoder(coderID string) error {
    coder, exists := m.GetCoder(coderID)
    if !exists {
        return &AICoderError{
            CoderID: coderID,
            Op:      "start",
            Err:     errors.New("coder not found"),
        }
    }
    
    // Implementation...
    if err := coder.start(); err != nil {
        return &AICoderError{
            CoderID: coderID,
            Op:      "start",
            Err:     err,
        }
    }
    
    return nil
}
```

### Error Recovery Patterns
```go
// Follow existing retry pattern with exponential backoff
func (c *AICoderProcess) executeTask(task AICoderTask) error {
    const maxRetries = 3
    const baseDelay = time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        if err := c.attemptTask(task); err != nil {
            if attempt == maxRetries-1 {
                return fmt.Errorf("task failed after %d attempts: %w", maxRetries, err)
            }
            
            delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
            time.Sleep(delay)
            continue
        }
        return nil
    }
    return nil
}
```

## Testing Patterns and Requirements

### Unit Test Pattern (following existing conventions)
```go
// Test file naming: *_test.go
// Test function naming: TestFunctionName_Scenario

func TestAICoderManager_CreateCoder_Success(t *testing.T) {
    // Setup
    manager := setupTestManager(t)
    req := CreateCoderRequest{
        Provider: "claude",
        Task:     "implement user authentication",
    }
    
    // Execute
    coder, err := manager.CreateCoder(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, coder)
    assert.Equal(t, StatusCreating, coder.Status)
    assert.DirExists(t, coder.WorkspaceDir)
}

func TestAICoderManager_CreateCoder_WorkspaceError(t *testing.T) {
    // Test error conditions...
}
```

### Integration Test Pattern
```go
func TestAICoderIntegration_FullLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup test environment
    manager := setupIntegrationTest(t)
    defer cleanupIntegrationTest(t, manager)
    
    // Test full lifecycle: create → start → monitor → complete
    // ...
}
```

### Mock Pattern for AI Providers
```go
type MockAIProvider struct {
    responses []string
    callCount int
}

func (m *MockAIProvider) GenerateCode(ctx context.Context, prompt string) (string, error) {
    if m.callCount >= len(m.responses) {
        return "", errors.New("no more mock responses")
    }
    
    response := m.responses[m.callCount]
    m.callCount++
    return response, nil
}
```

## Configuration Patterns

### Config Structure (following existing TOML pattern)
```go
// Add to existing config.Config struct
type AICoderConfig struct {
    Enabled             bool              `toml:"enabled"`
    MaxConcurrent       int               `toml:"max_concurrent"`
    WorkspaceBaseDir    string            `toml:"workspace_base_dir"`
    DefaultProvider     string            `toml:"default_provider"`
    TimeoutMinutes      int               `toml:"timeout_minutes"`
    Providers           map[string]ProviderConfig `toml:"providers"`
}

type ProviderConfig struct {
    APIKeyEnv string `toml:"api_key_env"`
    Model     string `toml:"model"`
    BaseURL   string `toml:"base_url,omitempty"`
}
```

### Config Validation Pattern
```go
func (c *AICoderConfig) Validate() error {
    if c.MaxConcurrent <= 0 {
        return errors.New("max_concurrent must be positive")
    }
    
    if c.WorkspaceBaseDir == "" {
        return errors.New("workspace_base_dir is required")
    }
    
    // Validate providers
    for name, provider := range c.Providers {
        if provider.APIKeyEnv == "" {
            return fmt.Errorf("provider %s: api_key_env is required", name)
        }
    }
    
    return nil
}
```

## Logging and Monitoring Patterns

### Structured Logging (following existing pattern)
```go
// Use structured logging with context
func (m *AICoderManager) logEvent(coderID string, level string, message string, fields map[string]interface{}) {
    entry := map[string]interface{}{
        "component": "ai_coder_manager",
        "coder_id":  coderID,
        "level":     level,
        "message":   message,
        "timestamp": time.Now(),
    }
    
    // Merge additional fields
    for k, v := range fields {
        entry[k] = v
    }
    
    // Emit to log store (following existing pattern)
    m.logStore.Add("ai_coder", "AICoderManager", message, level == "error")
}
```

### Metrics Collection Pattern
```go
// Define metrics following existing patterns
type AICoderMetrics struct {
    ActiveCoders    int64     `json:"active_coders"`
    CompletedTasks  int64     `json:"completed_tasks"`
    FailedTasks     int64     `json:"failed_tasks"`
    AverageRuntime  time.Duration `json:"average_runtime"`
    LastUpdated     time.Time `json:"last_updated"`
}

func (m *AICoderManager) GetMetrics() AICoderMetrics {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    metrics := AICoderMetrics{
        ActiveCoders: int64(len(m.coders)),
        LastUpdated:  time.Now(),
    }
    
    // Calculate aggregated metrics
    // ...
    
    return metrics
}
```

## Security Implementation Patterns

### Workspace Isolation Pattern
```go
// Secure path validation
func validateWorkspacePath(workspaceDir, requestedPath string) error {
    // Clean and resolve paths
    cleanWorkspace := filepath.Clean(workspaceDir)
    cleanRequested := filepath.Clean(filepath.Join(workspaceDir, requestedPath))
    
    // Ensure requested path is within workspace
    if !strings.HasPrefix(cleanRequested, cleanWorkspace) {
        return errors.New("path outside workspace not allowed")
    }
    
    return nil
}

// File operation wrapper with security checks
func (c *AICoderProcess) WriteFile(relativePath string, content []byte) error {
    if err := validateWorkspacePath(c.WorkspaceDir, relativePath); err != nil {
        return fmt.Errorf("security check failed: %w", err)
    }
    
    fullPath := filepath.Join(c.WorkspaceDir, relativePath)
    return os.WriteFile(fullPath, content, 0644)
}
```

### API Key Management Pattern
```go
// Secure credential access
func (p *ProviderConfig) GetAPIKey() (string, error) {
    if p.APIKeyEnv == "" {
        return "", errors.New("no API key environment variable configured")
    }
    
    apiKey := os.Getenv(p.APIKeyEnv)
    if apiKey == "" {
        return "", fmt.Errorf("API key not found in environment variable %s", p.APIKeyEnv)
    }
    
    return apiKey, nil
}
```

These patterns ensure consistency with the existing Brummer codebase while providing robust, secure, and maintainable AI coder functionality.