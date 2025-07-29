# Task: MCP Tools Implementation for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 4 files - Within target range (3-7 files)
**Estimated Time**: 20 minutes - Within target (15-30min)
**Token Estimate**: 100k tokens - Within target (<150k)
**Complexity Level**: 2 (Moderate) - MCP tool integration with established patterns
**Parallelization Benefit**: HIGH - Independent from TUI integration
**Atomicity Assessment**: ✅ ATOMIC - Complete MCP tool suite for AI coders
**Boundary Analysis**: ✅ CLEAR - Extends existing MCP system with new tool namespace

## Persona Assignment
**Persona**: Software Engineer (API/Integration)
**Expertise Required**: MCP protocol, JSON-RPC, API design, Go interfaces
**Worktree**: `~/work/worktrees/agentic-ai-coders/02-mcp-tools/`

## Context Summary
**Risk Level**: LOW (well-established MCP patterns)
**Integration Points**: Core AI coder service, MCP server infrastructure
**Architecture Pattern**: MCP Tool Registration Pattern (from existing tools)
**Similar Reference**: `internal/mcp/tools.go` - Tool handler implementations

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [internal/mcp/tools.go, internal/mcp/server.go]
modify_files: [internal/mcp/tools.go]
create_files: [
  /internal/mcp/ai_coder_tools.go,
  /internal/mcp/ai_coder_schemas.go,  
  /internal/mcp/ai_coder_handlers.go
]
# Total: 4 files (1 modify, 3 create) - appropriate for tool suite
```

**Existing Patterns to Follow**:
- `internal/mcp/tools.go` - Tool registration and metadata patterns
- Tool handler signature: `func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error)`
- JSON schema definitions for tool input validation

**Dependencies Context**:
- `github.com/mark3labs/mcp-go v0.32.0` - MCP protocol implementation
- JSON schema validation for tool parameters
- Integration with core AI coder service (dependency on Task 01)

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/mcp/tools.go                 # Add AI coder tool registration
  - /internal/mcp/ai_coder_tools.go        # AI coder tool definitions
  - /internal/mcp/ai_coder_schemas.go      # JSON schemas for tool parameters
  - /internal/mcp/ai_coder_handlers.go     # Tool handler implementations

direct_dependencies:
  - /internal/aicoder/manager.go           # Will interface with AI coder service (from Task 01)
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/mcp/server.go                # Review tool registration integration
  - /internal/mcp/streamable_server.go     # Check streaming tool compatibility
  - /cmd/main.go                           # Review for server initialization

check_documentation:
  - /docs/mcp-tools.md                     # Tool documentation updates needed
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/tui/                         # TUI integration separate task
  - /internal/process/                     # Process integration separate task
  - /internal/proxy/                       # Proxy system unrelated
  - /internal/discovery/                   # Discovery system unrelated
  - /pkg/events/                           # Event system separate integration

ignore_search_patterns:
  - "**/testdata/**"                       # Test data unrelated
  - "**/vendor/**"                         # Third-party code
  - "**/node_modules/**"                   # Node.js dependencies
```

**Boundary Analysis Results**:
- **Usage Count**: Limited to MCP subsystem
- **Scope Assessment**: LIMITED scope - extends well-defined MCP pattern
- **Impact Radius**: 1 existing file to modify, 3 new files to create

### External Context Sources (from master research)
**Primary Documentation**:
- [MCP Specification 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18) - Tool definition standards
- [JSON Schema](https://json-schema.org/) - Parameter validation schemas
- [MCP-Go Documentation](https://github.com/mark3labs/mcp-go) - Go implementation patterns

**Standards Applied**:
- MCP tool naming: `ai_coder_*` prefix for consistency
- JSON-RPC 2.0 error handling standards
- Tool metadata requirements (name, description, input schema)

**Reference Implementation**:
- Existing MCP tools in `internal/mcp/tools.go` - Follow same registration pattern
- Tool handler error handling and response formatting

## Task Requirements
**Objective**: Implement complete MCP tool suite for AI coder management and interaction

**Success Criteria**:
- [ ] AI coder tool suite with 6 core tools implemented
- [ ] JSON schema validation for all tool parameters
- [ ] Integration with core AI coder service (Task 01 dependency)
- [ ] Error handling following MCP standards
- [ ] Tool registration integrated with existing MCP server
- [ ] Streaming support for long-running operations
- [ ] Comprehensive tool metadata and documentation

**Tool Suite to Implement**:
1. `ai_coder_create` - Create and launch new AI coder instance
2. `ai_coder_list` - List active AI coders with status
3. `ai_coder_control` - Control AI coder (start/pause/stop/resume)
4. `ai_coder_status` - Get detailed status and progress
5. `ai_coder_workspace` - Access workspace files and structure
6. `ai_coder_logs` - Stream AI coder execution logs

**Validation Commands**:
```bash
# MCP Tool Integration Verification
grep -q "ai_coder_create" internal/mcp/tools.go        # Tool registration exists
go build ./internal/mcp                                # MCP package compiles
grep -q "InputSchema" internal/mcp/ai_coder_schemas.go # Schemas defined
curl -X POST http://localhost:7777/mcp -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' | grep ai_coder # Tools exposed
```

## Implementation Specifications

### Tool Registration Pattern
```go
// internal/mcp/ai_coder_tools.go
func RegisterAICoderTools(server *Server) {
    // Register all 6 AI coder tools following existing pattern
    server.RegisterTool("ai_coder_create", mcp.Tool{
        Name:        "ai_coder_create", 
        Description: "Create and launch a new AI coder instance with specified task and provider",
        InputSchema: aiCoderCreateSchema,
    }, handleAICoderCreate)
    
    server.RegisterTool("ai_coder_list", mcp.Tool{
        Name:        "ai_coder_list",
        Description: "List all active AI coder instances with current status and progress",
        InputSchema: aiCoderListSchema,
    }, handleAICoderList)
    
    // ... additional tools
}
```

### JSON Schema Definitions
```go
// internal/mcp/ai_coder_schemas.go
var aiCoderCreateSchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "task": map[string]interface{}{
            "type":        "string",
            "description": "The coding task or request for the AI coder to perform",
            "minLength":   1,
            "maxLength":   2000,
        },
        "provider": map[string]interface{}{
            "type":        "string", 
            "description": "AI provider to use (claude, gpt4, local)",
            "enum":        []string{"claude", "gpt4", "local"},
            "default":     "claude",
        },
        "workspace_files": map[string]interface{}{
            "type":        "array",
            "description": "Initial files to include in AI coder workspace context",
            "items": map[string]interface{}{
                "type": "string",
            },
            "maxItems": 50,
        },
    },
    "required": []string{"task"},
}

var aiCoderListSchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "status_filter": map[string]interface{}{
            "type":        "string",
            "description": "Filter coders by status (running, completed, failed, all)",
            "enum":        []string{"running", "completed", "failed", "all"},
            "default":     "all",
        },
        "limit": map[string]interface{}{
            "type":        "integer",
            "description": "Maximum number of coders to return",
            "minimum":     1,
            "maximum":     100,
            "default":     20,
        },
    },
}

// ... schemas for other tools
```

### Tool Handler Implementation
```go
// internal/mcp/ai_coder_handlers.go
func handleAICoderCreate(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
    // Extract and validate parameters
    task, ok := args["task"].(string)
    if !ok || task == "" {
        return nil, fmt.Errorf("task parameter is required and must be a non-empty string")
    }
    
    provider := "claude" // default
    if p, ok := args["provider"].(string); ok {
        provider = p
    }
    
    // Get AI coder manager from context or dependency injection
    manager := getAICoderManager(ctx)
    if manager == nil {
        return nil, fmt.Errorf("AI coder manager not available")
    }
    
    // Create AI coder instance
    req := aicoder.CreateCoderRequest{
        Task:     task,
        Provider: provider,
    }
    
    if workspaceFiles, ok := args["workspace_files"].([]interface{}); ok {
        req.WorkspaceFiles = make([]string, len(workspaceFiles))
        for i, f := range workspaceFiles {
            if file, ok := f.(string); ok {
                req.WorkspaceFiles[i] = file
            }
        }
    }
    
    coder, err := manager.CreateCoder(ctx, req)
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{{
                Type: "text",
                Text: fmt.Sprintf("Failed to create AI coder: %v", err),
            }},
        }, nil
    }
    
    // Return success response with coder details
    return &mcp.CallToolResult{
        Content: []mcp.Content{{
            Type: "text", 
            Text: fmt.Sprintf("AI coder created successfully:\nID: %s\nStatus: %s\nWorkspace: %s", 
                coder.ID, coder.Status, coder.WorkspaceDir),
        }},
    }, nil
}

func handleAICoderList(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
    manager := getAICoderManager(ctx)
    if manager == nil {
        return nil, fmt.Errorf("AI coder manager not available")
    }
    
    // Extract filter parameters
    statusFilter := "all"
    if s, ok := args["status_filter"].(string); ok {
        statusFilter = s
    }
    
    limit := 20
    if l, ok := args["limit"].(float64); ok {
        limit = int(l)
    }
    
    // Get filtered list of coders
    coders := manager.ListCoders()
    
    // Apply status filter
    filteredCoders := make([]*aicoder.AICoderProcess, 0)
    for _, coder := range coders {
        if statusFilter == "all" || string(coder.Status) == statusFilter {
            filteredCoders = append(filteredCoders, coder)
        }
        if len(filteredCoders) >= limit {
            break
        }
    }
    
    // Format response
    var response strings.Builder
    response.WriteString(fmt.Sprintf("Active AI Coders (%d):\n\n", len(filteredCoders)))
    
    for _, coder := range filteredCoders {
        response.WriteString(fmt.Sprintf("ID: %s\n", coder.ID))
        response.WriteString(fmt.Sprintf("Status: %s\n", coder.Status))
        response.WriteString(fmt.Sprintf("Provider: %s\n", coder.Provider))
        response.WriteString(fmt.Sprintf("Created: %s\n", coder.CreatedAt.Format(time.RFC3339)))
        response.WriteString(fmt.Sprintf("Progress: %.1f%%\n", coder.Progress*100))
        response.WriteString("---\n")
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{{
            Type: "text",
            Text: response.String(),
        }},
    }, nil
}

// Helper function for dependency injection
func getAICoderManager(ctx context.Context) *aicoder.AICoderManager {
    // Implementation depends on how manager is injected
    // Could be from context, global variable, or service locator
    if manager, ok := ctx.Value("ai_coder_manager").(*aicoder.AICoderManager); ok {
        return manager
    }
    return nil
}
```

### Integration with Existing MCP System
```go
// Addition to internal/mcp/tools.go
func RegisterAllTools(server *Server) {
    // Existing tool registrations...
    registerScriptTools(server)
    registerLogTools(server)
    registerProxyTools(server)
    // ... other existing tools
    
    // Add AI coder tools
    RegisterAICoderTools(server)
}
```

## Risk Mitigation (from master analysis)
**Low-Risk Mitigations**:
- MCP protocol compliance - Follow existing tool patterns from `internal/mcp/tools.go` - Validation: MCP client integration tests
- JSON schema validation - Use established schema patterns - Testing: Parameter validation test suite
- Error handling - Follow MCP error response standards - Recovery: Graceful error responses with helpful messages

**Context Validation**:
- [ ] Tool registration patterns from `internal/mcp/tools.go` successfully applied
- [ ] JSON schema validation consistent with existing tool schemas
- [ ] Error handling follows MCP CallToolResult standards

## Integration with Other Tasks
**Dependencies**: Task 01 (Core Service) - Requires AICoderManager interface
**Integration Points**: 
- Task 04 (TUI Integration) will use these tools for UI operations
- Task 05 (Process Integration) will extend tool capabilities
- Task 07 (Testing) will test MCP tool integration

**Shared Context**: MCP tool interface becomes primary control surface for AI coders

## Execution Notes
- **Start Pattern**: Use existing tool registration in `internal/mcp/tools.go` as template
- **Key Context**: Focus on consistent error handling and response formatting
- **Integration Test**: Verify tools work with MCP client after implementation
- **Review Focus**: Parameter validation and error response consistency

## Additional Tool Specifications

### Control Tool Implementation
```go
func handleAICoderControl(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
    coderID, ok := args["coder_id"].(string)
    if !ok {
        return nil, fmt.Errorf("coder_id parameter is required")
    }
    
    action, ok := args["action"].(string)
    if !ok {
        return nil, fmt.Errorf("action parameter is required")
    }
    
    manager := getAICoderManager(ctx)
    var err error
    
    switch action {
    case "start":
        err = manager.StartCoder(coderID)
    case "pause": 
        err = manager.PauseCoder(coderID)
    case "resume":
        err = manager.ResumeCoder(coderID)
    case "stop":
        err = manager.StopCoder(coderID)
    default:
        return nil, fmt.Errorf("invalid action: %s", action)
    }
    
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{{Type: "text", Text: err.Error()}},
        }, nil
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{{
            Type: "text",
            Text: fmt.Sprintf("AI coder %s: %s action completed successfully", coderID, action),
        }},
    }, nil
}
```

### Workspace Tool with File Operations
```go
func handleAICoderWorkspace(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
    coderID, ok := args["coder_id"].(string)
    if !ok {
        return nil, fmt.Errorf("coder_id parameter is required")
    }
    
    operation := "list" // default
    if op, ok := args["operation"].(string); ok {
        operation = op
    }
    
    manager := getAICoderManager(ctx)
    coder, exists := manager.GetCoder(coderID)
    if !exists {
        return nil, fmt.Errorf("AI coder %s not found", coderID)
    }
    
    switch operation {
    case "list":
        files, err := coder.ListWorkspaceFiles()
        if err != nil {
            return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{{Type: "text", Text: err.Error()}}}, nil
        }
        return &mcp.CallToolResult{
            Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Workspace files:\n%s", strings.Join(files, "\n"))}},
        }, nil
        
    case "read":
        filePath, ok := args["file_path"].(string)
        if !ok {
            return nil, fmt.Errorf("file_path parameter required for read operation")
        }
        content, err := coder.ReadWorkspaceFile(filePath)
        if err != nil {
            return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{{Type: "text", Text: err.Error()}}}, nil
        }
        return &mcp.CallToolResult{
            Content: []mcp.Content{{Type: "text", Text: string(content)}},
        }, nil
        
    default:
        return nil, fmt.Errorf("invalid operation: %s", operation)
    }
}
```

This task creates a comprehensive MCP tool suite that provides full control over AI coder instances through the established MCP protocol, enabling integration with AI assistants and external tools.