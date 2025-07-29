# Task: Testing Suite for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 6 files - Within target range (3-7 files)
**Estimated Time**: 25 minutes - Within target (15-30min)
**Token Estimate**: 130k tokens - Within target (<150k)
**Complexity Level**: 2 (Moderate) - Comprehensive testing across multiple components
**Parallelization Benefit**: LOW - Requires implementation of other tasks
**Atomicity Assessment**: ✅ ATOMIC - Complete testing infrastructure
**Boundary Analysis**: ✅ CLEAR - Testing infrastructure with clear scope

## Persona Assignment
**Persona**: Software Engineer (Testing/QA)
**Expertise Required**: Go testing patterns, integration testing, mock frameworks
**Worktree**: `~/work/worktrees/agentic-ai-coders/07-testing-suite/`

## Context Summary
**Risk Level**: LOW (testing infrastructure, well-established patterns)
**Integration Points**: All AI coder components, testing frameworks
**Architecture Pattern**: Testing Infrastructure Pattern (from existing tests)
**Similar Reference**: Existing test files in various packages

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [internal/process/manager_test.go, internal/mcp/server_test.go]
modify_files: []
create_files: [
  /internal/aicoder/manager_test.go,
  /internal/aicoder/integration_test.go,
  /internal/mcp/ai_coder_tools_test.go,
  /internal/tui/ai_coder_view_test.go,
  /pkg/events/ai_coder_events_test.go,
  /test/ai_coder_system_test.go
]
# Total: 6 files - comprehensive test coverage
```

**Existing Patterns to Follow**:
- Test file naming: `*_test.go`
- Test function naming: `TestFunctionName_Scenario`
- Setup/teardown patterns from existing tests

**Dependencies Context**:
- Testing all components from Tasks 01-06
- Integration with existing testing infrastructure
- Mock frameworks for AI provider testing

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/aicoder/manager_test.go          # Core service unit tests
  - /internal/aicoder/integration_test.go      # Integration tests for AI coder service
  - /internal/mcp/ai_coder_tools_test.go       # MCP tools testing
  - /internal/tui/ai_coder_view_test.go        # TUI component tests
  - /pkg/events/ai_coder_events_test.go        # Event system tests
  - /test/ai_coder_system_test.go              # End-to-end system tests

direct_dependencies: []                        # New test files only
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /Makefile                                  # Build and test target updates
  - /.github/workflows/test.yml                # CI pipeline updates
  - /go.mod                                    # Test dependency updates

check_documentation:
  - /docs/testing.md                           # Testing documentation updates
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/proxy/                           # Proxy tests separate
  - /internal/discovery/                       # Discovery tests separate
  - /internal/logs/                            # Log tests separate

ignore_search_patterns:
  - "**/testdata/**"                           # Existing test data
  - "**/vendor/**"                             # Third-party code
  - "**/node_modules/**"                       # JavaScript dependencies
```

**Boundary Analysis Results**:
- **Usage Count**: 0 files to modify (new test files only)
- **Scope Assessment**: LIMITED scope - testing infrastructure only
- **Impact Radius**: 6 new test files, no modification of existing code

### External Context Sources (from master research)
**Primary Documentation**:
- [Go Testing](https://golang.org/pkg/testing/) - Testing framework and patterns
- [Go Testing Best Practices](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests) - Table-driven tests
- [Testify Framework](https://github.com/stretchr/testify) - Assertion and mock library

**Standards Applied**:
- Table-driven test patterns for comprehensive coverage
- Mock interfaces for external dependencies (AI providers)
- Integration tests with proper setup/teardown

**Reference Implementation**:
- Existing test patterns in various Brummer packages
- Testing infrastructure and helper functions
- CI/CD integration patterns

## Task Requirements
**Objective**: Implement comprehensive testing suite covering all AI coder functionality

**Success Criteria**:
- [ ] Unit tests for all AI coder service functions with >90% coverage
- [ ] Integration tests for AI coder lifecycle and workspace operations
- [ ] MCP tool tests validating all input/output scenarios
- [ ] TUI component tests for AI coder view interactions
- [ ] Event system tests for all AI coder event types
- [ ] End-to-end system tests covering complete user workflows
- [ ] Mock AI providers for deterministic testing
- [ ] Performance tests for concurrent AI coder operations
- [ ] Error handling tests for all failure scenarios

**Test Categories to Implement**:
1. **Unit Tests** - Individual function and method testing
2. **Integration Tests** - Component interaction testing  
3. **System Tests** - End-to-end workflow testing
4. **Performance Tests** - Load and concurrency testing
5. **Error Tests** - Failure scenario and recovery testing

**Validation Commands**:
```bash
# Testing Suite Verification
go test ./internal/aicoder -v                           # Core service tests
go test ./internal/mcp -run TestAICoder -v              # MCP tool tests
go test ./internal/tui -run TestAICoder -v              # TUI tests
go test ./pkg/events -run TestAICoder -v                # Event tests
go test ./test -run TestAICoderSystem -v                # System tests
go test -race ./...                                     # Race condition detection
```

## Implementation Specifications

### Core Service Unit Tests
```go
// internal/aicoder/manager_test.go
package aicoder

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
)

// Mock AI Provider for testing
type MockAIProvider struct {
    mock.Mock
}

func (m *MockAIProvider) Name() string {
    args := m.Called()
    return args.String(0)
}

func (m *MockAIProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
    args := m.Called(ctx, prompt, options)
    return args.Get(0).(*GenerateResult), args.Error(1)
}

func (m *MockAIProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
    args := m.Called(ctx, prompt, options)
    return args.Get(0).(<-chan GenerateUpdate), args.Error(1)
}

func (m *MockAIProvider) ValidateConfig(config ProviderConfig) error {
    args := m.Called(config)
    return args.Error(0)
}

// Mock Event Bus
type MockEventBus struct {
    mock.Mock
    events []interface{}
}

func (m *MockEventBus) Emit(eventType string, data interface{}) {
    m.Called(eventType, data)
    m.events = append(m.events, data)
}

func (m *MockEventBus) Subscribe(eventType string, handler func(interface{})) {
    m.Called(eventType, handler)
}

// Test Setup
func setupTestManager(t *testing.T) (*AICoderManager, *MockEventBus, string) {
    tmpDir := t.TempDir()
    
    eventBus := &MockEventBus{}
    config := &TestConfig{
        WorkspaceBaseDir: tmpDir,
        MaxConcurrent:    3,
        DefaultProvider:  "mock",
    }
    
    manager := NewAICoderManager(config, eventBus)
    
    // Register mock provider
    mockProvider := &MockAIProvider{}
    mockProvider.On("Name").Return("mock")
    manager.RegisterProvider("mock", mockProvider)
    
    return manager, eventBus, tmpDir
}

// Unit Tests
func TestAICoderManager_CreateCoder_Success(t *testing.T) {
    manager, eventBus, _ := setupTestManager(t)
    
    req := CreateCoderRequest{
        Provider: "mock",
        Task:     "implement user authentication",
    }
    
    // Set up mock expectations
    eventBus.On("Emit", "ai_coder_created", mock.Anything)
    
    // Execute
    coder, err := manager.CreateCoder(context.Background(), req)
    
    // Assert
    require.NoError(t, err)
    require.NotNil(t, coder)
    assert.Equal(t, StatusCreating, coder.Status)
    assert.Equal(t, "mock", coder.Provider)
    assert.Equal(t, req.Task, coder.Task)
    assert.DirExists(t, coder.WorkspaceDir)
    
    // Verify event was emitted
    eventBus.AssertCalled(t, "Emit", "ai_coder_created", mock.Anything)
}

func TestAICoderManager_CreateCoder_InvalidProvider(t *testing.T) {
    manager, _, _ := setupTestManager(t)
    
    req := CreateCoderRequest{
        Provider: "nonexistent",
        Task:     "some task",
    }
    
    // Execute
    coder, err := manager.CreateCoder(context.Background(), req)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, coder)
    assert.Contains(t, err.Error(), "provider not found")
}

func TestAICoderManager_CreateCoder_WorkspaceError(t *testing.T) {
    manager, _, tmpDir := setupTestManager(t)
    
    // Make workspace directory read-only to cause error
    err := os.Chmod(tmpDir, 0444)
    require.NoError(t, err)
    defer os.Chmod(tmpDir, 0755)
    
    req := CreateCoderRequest{
        Provider: "mock",
        Task:     "some task",
    }
    
    // Execute
    coder, err := manager.CreateCoder(context.Background(), req)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, coder)
}

func TestAICoderManager_GetCoder_Exists(t *testing.T) {
    manager, eventBus, _ := setupTestManager(t)
    
    // Create a coder first
    eventBus.On("Emit", mock.Anything, mock.Anything)
    coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
        Provider: "mock",
        Task:     "test task",
    })
    require.NoError(t, err)
    
    // Execute
    retrieved, exists := manager.GetCoder(coder.ID)
    
    // Assert
    assert.True(t, exists)
    assert.Equal(t, coder.ID, retrieved.ID)
}

func TestAICoderManager_GetCoder_NotExists(t *testing.T) {
    manager, _, _ := setupTestManager(t)
    
    // Execute
    coder, exists := manager.GetCoder("nonexistent-id")
    
    // Assert
    assert.False(t, exists)
    assert.Nil(t, coder)
}

func TestAICoderManager_ListCoders_Empty(t *testing.T) {
    manager, _, _ := setupTestManager(t)
    
    // Execute
    coders := manager.ListCoders()
    
    // Assert
    assert.Empty(t, coders)
}

func TestAICoderManager_ListCoders_Multiple(t *testing.T) {
    manager, eventBus, _ := setupTestManager(t)
    eventBus.On("Emit", mock.Anything, mock.Anything)
    
    // Create multiple coders
    req1 := CreateCoderRequest{Provider: "mock", Task: "task 1"}
    req2 := CreateCoderRequest{Provider: "mock", Task: "task 2"}
    
    coder1, err := manager.CreateCoder(context.Background(), req1)
    require.NoError(t, err)
    
    coder2, err := manager.CreateCoder(context.Background(), req2)
    require.NoError(t, err)
    
    // Execute
    coders := manager.ListCoders()
    
    // Assert
    assert.Len(t, coders, 2)
    coderIDs := []string{coders[0].ID, coders[1].ID}
    assert.Contains(t, coderIDs, coder1.ID)
    assert.Contains(t, coderIDs, coder2.ID)
}

func TestAICoderManager_DeleteCoder_Success(t *testing.T) {
    manager, eventBus, _ := setupTestManager(t)
    eventBus.On("Emit", mock.Anything, mock.Anything)
    
    // Create a coder first
    coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
        Provider: "mock",
        Task:     "test task",
    })
    require.NoError(t, err)
    
    // Execute
    err = manager.DeleteCoder(coder.ID)
    
    // Assert
    assert.NoError(t, err)
    
    // Verify coder is gone
    _, exists := manager.GetCoder(coder.ID)
    assert.False(t, exists)
    
    // Verify workspace is cleaned up
    assert.NoDirExists(t, coder.WorkspaceDir)
}

func TestAICoderManager_DeleteCoder_NotExists(t *testing.T) {
    manager, _, _ := setupTestManager(t)
    
    // Execute
    err := manager.DeleteCoder("nonexistent-id")
    
    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found")
}

// Concurrent access tests
func TestAICoderManager_ConcurrentAccess(t *testing.T) {
    manager, eventBus, _ := setupTestManager(t)
    eventBus.On("Emit", mock.Anything, mock.Anything)
    
    const numGoroutines = 10
    const numOperations = 100
    
    // Start multiple goroutines performing operations
    done := make(chan bool, numGoroutines)
    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            defer func() { done <- true }()
            
            for j := 0; j < numOperations; j++ {
                // Create coder
                coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
                    Provider: "mock",
                    Task:     fmt.Sprintf("task-%d-%d", id, j),
                })
                if err != nil {
                    t.Errorf("Failed to create coder: %v", err)
                    return
                }
                
                // List coders
                coders := manager.ListCoders()
                assert.NotEmpty(t, coders)
                
                // Get coder
                retrieved, exists := manager.GetCoder(coder.ID)
                assert.True(t, exists)
                assert.NotNil(t, retrieved)
                
                // Delete coder
                err = manager.DeleteCoder(coder.ID)
                if err != nil {
                    t.Errorf("Failed to delete coder: %v", err)
                    return
                }
            }
        }(i)
    }
    
    // Wait for all goroutines to complete
    for i := 0; i < numGoroutines; i++ {
        select {
        case <-done:
        case <-time.After(30 * time.Second):
            t.Fatal("Test timed out")
        }
    }
}

// Table-driven tests for status transitions
func TestAICoderManager_StatusTransitions(t *testing.T) {
    tests := []struct {
        name           string
        initialStatus  AICoderStatus
        operation      string
        expectedStatus AICoderStatus
        expectedError  bool
    }{
        {
            name:           "Start from Creating",
            initialStatus:  StatusCreating,
            operation:      "start",
            expectedStatus: StatusRunning,
            expectedError:  false,
        },
        {
            name:           "Pause from Running", 
            initialStatus:  StatusRunning,
            operation:      "pause",
            expectedStatus: StatusPaused,
            expectedError:  false,
        },
        {
            name:           "Resume from Paused",
            initialStatus:  StatusPaused,
            operation:      "resume", 
            expectedStatus: StatusRunning,
            expectedError:  false,
        },
        {
            name:           "Stop from Running",
            initialStatus:  StatusRunning,
            operation:      "stop",
            expectedStatus: StatusStopped,
            expectedError:  false,
        },
        {
            name:           "Invalid transition",
            initialStatus:  StatusCompleted,
            operation:      "start",
            expectedStatus: StatusCompleted,
            expectedError:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            manager, eventBus, _ := setupTestManager(t)
            eventBus.On("Emit", mock.Anything, mock.Anything)
            
            // Create coder and set initial status
            coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
                Provider: "mock",
                Task:     "test task",
            })
            require.NoError(t, err)
            
            // Set initial status
            coder.Status = tt.initialStatus
            
            // Execute operation
            var opErr error
            switch tt.operation {
            case "start":
                opErr = manager.StartCoder(coder.ID)
            case "pause":
                opErr = manager.PauseCoder(coder.ID)
            case "resume":
                opErr = manager.ResumeCoder(coder.ID)
            case "stop":
                opErr = manager.StopCoder(coder.ID)
            }
            
            // Assert
            if tt.expectedError {
                assert.Error(t, opErr)
            } else {
                assert.NoError(t, opErr)
            }
            
            // Check final status
            retrieved, exists := manager.GetCoder(coder.ID)
            require.True(t, exists)
            assert.Equal(t, tt.expectedStatus, retrieved.Status)
        })
    }
}

// TestConfig implementation for testing
type TestConfig struct {
    WorkspaceBaseDir string
    MaxConcurrent    int
    DefaultProvider  string
}

func (c *TestConfig) GetAICoderConfig() AICoderConfig {
    return AICoderConfig{
        WorkspaceBaseDir: c.WorkspaceBaseDir,
        MaxConcurrent:    c.MaxConcurrent,
        DefaultProvider:  c.DefaultProvider,
    }
}
```

### Integration Tests
```go
// internal/aicoder/integration_test.go
package aicoder

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAICoderIntegration_FullLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup
    manager, eventBus, tmpDir := setupIntegrationTest(t)
    defer cleanupIntegrationTest(t, tmpDir)
    
    // Test data
    req := CreateCoderRequest{
        Provider: "mock",
        Task:     "Create a simple REST API with authentication",
        WorkspaceFiles: []string{
            "main.go",
            "auth.go", 
            "handlers.go",
        },
    }
    
    // Step 1: Create AI coder
    coder, err := manager.CreateCoder(context.Background(), req)
    require.NoError(t, err)
    assert.Equal(t, StatusCreating, coder.Status)
    
    // Verify workspace created
    assert.DirExists(t, coder.WorkspaceDir)
    
    // Step 2: Start AI coder
    err = manager.StartCoder(coder.ID)
    require.NoError(t, err)
    
    // Wait for status update
    time.Sleep(100 * time.Millisecond)
    
    updated, exists := manager.GetCoder(coder.ID)
    require.True(t, exists)
    assert.Equal(t, StatusRunning, updated.Status)
    
    // Step 3: Simulate some progress
    err = updated.UpdateProgress(0.5, "Generated authentication module")
    require.NoError(t, err)
    
    // Step 4: Create some workspace files
    err = updated.WriteFile("main.go", []byte("package main\n\nfunc main() {\n\t// TODO: implement\n}"))
    require.NoError(t, err)
    
    err = updated.WriteFile("auth.go", []byte("package main\n\n// Authentication module"))
    require.NoError(t, err)
    
    // Verify files exist
    files, err := updated.ListWorkspaceFiles()
    require.NoError(t, err) 
    assert.Contains(t, files, "main.go")
    assert.Contains(t, files, "auth.go")
    
    // Step 5: Complete the task
    err = updated.UpdateProgress(1.0, "Task completed successfully")
    require.NoError(t, err)
    
    err = updated.SetStatus(StatusCompleted)
    require.NoError(t, err)
    
    // Step 6: Verify final state
    final, exists := manager.GetCoder(coder.ID)
    require.True(t, exists)
    assert.Equal(t, StatusCompleted, final.Status)
    assert.Equal(t, 1.0, final.Progress)
    
    // Verify events were emitted
    assert.Greater(t, len(eventBus.events), 0)
}

func TestAICoderIntegration_WorkspaceOperations(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup
    manager, _, tmpDir := setupIntegrationTest(t)
    defer cleanupIntegrationTest(t, tmpDir)
    
    // Create coder
    coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
        Provider: "mock",
        Task:     "test workspace operations",
    })
    require.NoError(t, err)
    
    // Test file operations
    testCases := []struct {
        filename string
        content  string
    }{
        {"hello.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"},
        {"README.md", "# Test Project\n\nThis is a test project."},
        {"config.json", "{\"name\": \"test\", \"version\": \"1.0.0\"}"},
    }
    
    // Write files
    for _, tc := range testCases {
        err := coder.WriteFile(tc.filename, []byte(tc.content))
        require.NoError(t, err, "Failed to write %s", tc.filename)
    }
    
    // List files
    files, err := coder.ListWorkspaceFiles()
    require.NoError(t, err)
    
    for _, tc := range testCases {
        assert.Contains(t, files, tc.filename)
    }
    
    // Read files back
    for _, tc := range testCases {
        content, err := coder.ReadFile(tc.filename)
        require.NoError(t, err, "Failed to read %s", tc.filename)
        assert.Equal(t, tc.content, string(content))
    }
    
    // Test path validation (security)
    err = coder.WriteFile("../outside.txt", []byte("should fail"))
    assert.Error(t, err, "Should prevent writing outside workspace")
    
    err = coder.WriteFile("/etc/passwd", []byte("should fail"))
    assert.Error(t, err, "Should prevent writing to system files")
}

func TestAICoderIntegration_ErrorHandling(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup
    manager, _, tmpDir := setupIntegrationTest(t)
    defer cleanupIntegrationTest(t, tmpDir)
    
    // Test provider error handling
    mockProvider := &MockAIProvider{}
    mockProvider.On("Name").Return("failing-provider")
    mockProvider.On("GenerateCode", mock.Anything, mock.Anything, mock.Anything).
        Return((*GenerateResult)(nil), errors.New("API error"))
    
    manager.RegisterProvider("failing-provider", mockProvider)
    
    // Create coder with failing provider
    coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
        Provider: "failing-provider",
        Task:     "test error handling",
    })
    require.NoError(t, err)
    
    // Start coder - should handle provider failure gracefully
    err = manager.StartCoder(coder.ID)
    assert.Error(t, err)
    
    // Verify coder status reflects failure
    updated, exists := manager.GetCoder(coder.ID)
    require.True(t, exists)
    assert.Equal(t, StatusFailed, updated.Status)
}

// Helper functions
func setupIntegrationTest(t *testing.T) (*AICoderManager, *MockEventBus, string) {
    tmpDir := t.TempDir()
    
    eventBus := &MockEventBus{}
    config := &TestConfig{
        WorkspaceBaseDir: tmpDir,
        MaxConcurrent:    5,
        DefaultProvider:  "mock",
    }
    
    manager := NewAICoderManager(config, eventBus)
    
    // Register working mock provider
    mockProvider := &MockAIProvider{}
    mockProvider.On("Name").Return("mock")
    mockProvider.On("GenerateCode", mock.Anything, mock.Anything, mock.Anything).
        Return(&GenerateResult{
            Code:    "// Generated code",
            Summary: "Code generated successfully",
        }, nil)
    
    manager.RegisterProvider("mock", mockProvider)
    
    return manager, eventBus, tmpDir
}

func cleanupIntegrationTest(t *testing.T, tmpDir string) {
    err := os.RemoveAll(tmpDir)
    if err != nil {
        t.Logf("Warning: failed to cleanup test directory: %v", err)
    }
}
```

### System Tests
```go
// test/ai_coder_system_test.go
package test

import (
    "context"
    "os"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/standardbeagle/brummer/internal/aicoder"
    "github.com/standardbeagle/brummer/internal/mcp"
    "github.com/standardbeagle/brummer/pkg/events"
)

func TestAICoderSystem_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping system test")
    }
    
    // Setup full system
    system := setupFullSystem(t)
    defer system.cleanup()
    
    // Test complete workflow: MCP -> AI Coder -> Events -> TUI
    
    // Step 1: Create AI coder via MCP
    createResult, err := system.mcpServer.CallTool(context.Background(), "ai_coder_create", map[string]interface{}{
        "task":     "Create a simple web server",
        "provider": "mock",
    })
    require.NoError(t, err)
    require.False(t, createResult.IsError)
    
    // Extract coder ID from result
    coderID := extractCoderID(t, createResult)
    
    // Step 2: Monitor via events
    eventReceived := make(chan bool, 1)
    system.eventBus.Subscribe("ai_coder_created", func(data interface{}) {
        eventReceived <- true
    })
    
    // Wait for creation event
    select {
    case <-eventReceived:
        // Event received as expected
    case <-time.After(5 * time.Second):
        t.Fatal("Did not receive ai_coder_created event")
    }
    
    // Step 3: Control via MCP
    controlResult, err := system.mcpServer.CallTool(context.Background(), "ai_coder_control", map[string]interface{}{
        "coder_id": coderID,
        "action":   "start",
    })
    require.NoError(t, err)
    require.False(t, controlResult.IsError)
    
    // Step 4: Check status via MCP
    statusResult, err := system.mcpServer.CallTool(context.Background(), "ai_coder_status", map[string]interface{}{
        "coder_id": coderID,
    })
    require.NoError(t, err)
    require.False(t, statusResult.IsError)
    
    // Verify status shows running
    assert.Contains(t, statusResult.Content[0].Text, "running")
    
    // Step 5: Simulate completion
    time.Sleep(100 * time.Millisecond) // Allow for async processing
    
    // Step 6: List all coders
    listResult, err := system.mcpServer.CallTool(context.Background(), "ai_coder_list", map[string]interface{}{})
    require.NoError(t, err)
    require.False(t, listResult.IsError)
    
    // Verify our coder appears in list
    assert.Contains(t, listResult.Content[0].Text, coderID)
    
    // Step 7: Clean up
    deleteResult, err := system.mcpServer.CallTool(context.Background(), "ai_coder_delete", map[string]interface{}{
        "coder_id": coderID,
    })
    require.NoError(t, err)
    require.False(t, deleteResult.IsError)
}

func TestAICoderSystem_ConcurrentUsers(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping system test")
    }
    
    system := setupFullSystem(t)
    defer system.cleanup()
    
    const numUsers = 5
    const codersPerUser = 3
    
    results := make(chan error, numUsers)
    
    // Simulate multiple concurrent users
    for user := 0; user < numUsers; user++ {
        go func(userID int) {
            var err error
            defer func() { results <- err }()
            
            coderIDs := make([]string, 0, codersPerUser)
            
            // Create multiple coders per user
            for i := 0; i < codersPerUser; i++ {
                createResult, createErr := system.mcpServer.CallTool(context.Background(), "ai_coder_create", map[string]interface{}{
                    "task":     fmt.Sprintf("User %d Task %d", userID, i),
                    "provider": "mock",
                })
                if createErr != nil {
                    err = createErr
                    return
                }
                
                coderID := extractCoderID(t, createResult)
                coderIDs = append(coderIDs, coderID)
                
                // Start the coder
                _, startErr := system.mcpServer.CallTool(context.Background(), "ai_coder_control", map[string]interface{}{
                    "coder_id": coderID,
                    "action":   "start",
                })
                if startErr != nil {
                    err = startErr
                    return
                }
                
                // Brief delay to avoid overwhelming system
                time.Sleep(10 * time.Millisecond)
            }
            
            // Clean up coders
            for _, coderID := range coderIDs {
                _, deleteErr := system.mcpServer.CallTool(context.Background(), "ai_coder_delete", map[string]interface{}{
                    "coder_id": coderID,
                })
                if deleteErr != nil {
                    err = deleteErr
                    return
                }
            }
        }(user)
    }
    
    // Wait for all users to complete
    for i := 0; i < numUsers; i++ {
        select {
        case err := <-results:
            require.NoError(t, err, "User %d failed", i)
        case <-time.After(30 * time.Second):
            t.Fatal("System test timed out")
        }
    }
}

// System test infrastructure
type TestSystem struct {
    aiCoderMgr *aicoder.AICoderManager
    mcpServer  *mcp.Server
    eventBus   *events.EventBus
    tmpDir     string
}

func (s *TestSystem) cleanup() {
    os.RemoveAll(s.tmpDir)
}

func setupFullSystem(t *testing.T) *TestSystem {
    tmpDir := t.TempDir()
    
    // Setup event bus
    eventBus := events.NewEventBus()
    
    // Setup AI coder manager
    config := &TestAICoderConfig{
        WorkspaceBaseDir: tmpDir,
        MaxConcurrent:    10,
        DefaultProvider:  "mock",
    }
    
    aiCoderMgr := aicoder.NewAICoderManager(config, eventBus)
    
    // Register mock provider
    mockProvider := &MockAIProvider{}
    mockProvider.On("Name").Return("mock")
    mockProvider.On("GenerateCode", mock.Anything, mock.Anything, mock.Anything).
        Return(&aicoder.GenerateResult{
            Code:    "// System test generated code",
            Summary: "System test completion",
        }, nil)
    
    aiCoderMgr.RegisterProvider("mock", mockProvider)
    
    // Setup MCP server with AI coder tools
    mcpServer := mcp.NewServer()
    mcp.RegisterAICoderTools(mcpServer, aiCoderMgr)
    
    return &TestSystem{
        aiCoderMgr: aiCoderMgr,
        mcpServer:  mcpServer,
        eventBus:   eventBus,
        tmpDir:     tmpDir,
    }
}

func extractCoderID(t *testing.T, result *mcp.CallToolResult) string {
    // Parse coder ID from MCP result
    // Implementation depends on exact response format
    content := result.Content[0].Text
    // Simple extraction for test - would be more robust in real implementation
    lines := strings.Split(content, "\n")
    for _, line := range lines {
        if strings.HasPrefix(line, "ID: ") {
            return strings.TrimPrefix(line, "ID: ")
        }
    }
    t.Fatal("Could not extract coder ID from result")
    return ""
}

type TestAICoderConfig struct {
    WorkspaceBaseDir string
    MaxConcurrent    int
    DefaultProvider  string
}

func (c *TestAICoderConfig) GetAICoderConfig() aicoder.AICoderConfig {
    return aicoder.AICoderConfig{
        WorkspaceBaseDir: c.WorkspaceBaseDir,
        MaxConcurrent:    c.MaxConcurrent,
        DefaultProvider:  c.DefaultProvider,
    }
}
```

## Risk Mitigation (from master analysis)
**Low-Risk Mitigations**:
- Test infrastructure complexity - Follow established Go testing patterns - Coverage: Comprehensive test suite across all components
- Mock framework integration - Use testify for consistent mocking - Isolation: Mock external dependencies for deterministic tests
- Performance test stability - Use reasonable timeouts and concurrency limits - Monitoring: Test execution time and resource usage

**Context Validation**:
- [ ] Testing patterns from existing test files successfully applied
- [ ] Mock frameworks properly integrated for external dependencies
- [ ] Test coverage meets quality standards (>90% for core functionality)

## Integration with Other Tasks
**Dependencies**: All other tasks (01-06) - Testing requires complete implementation
**Integration Points**: 
- Tests validate all components work together correctly
- Provides regression testing for future changes
- Enables safe refactoring and optimization

**Shared Context**: Testing suite becomes the quality gate for AI coder feature

## Execution Notes
- **Start Pattern**: Use existing test patterns from various Brummer packages
- **Key Context**: Focus on comprehensive coverage while maintaining test performance
- **Mock Strategy**: Mock external AI providers for deterministic testing
- **Review Focus**: Test coverage analysis and performance impact of test suite

This task creates a comprehensive testing infrastructure that ensures the reliability, performance, and correctness of all AI coder functionality across the entire system.