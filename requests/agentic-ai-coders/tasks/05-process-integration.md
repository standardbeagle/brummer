# Task: Process Integration for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 4 files - Within target range (3-7 files)
**Estimated Time**: 20 minutes - Within target (15-30min)
**Token Estimate**: 90k tokens - Within target (<150k)
**Complexity Level**: 2 (Moderate) - Integration with existing process system
**Parallelization Benefit**: MEDIUM - Requires core service and TUI completion
**Atomicity Assessment**: ✅ ATOMIC - Complete process system integration
**Boundary Analysis**: ✅ CLEAR - Extends existing process manager patterns

## Persona Assignment
**Persona**: Software Engineer (Systems Integration)
**Expertise Required**: Process management, Go concurrency, system integration
**Worktree**: `~/work/worktrees/agentic-ai-coders/05-process-integration/`

## Context Summary
**Risk Level**: MEDIUM (integration complexity, process coordination)
**Integration Points**: Process manager, AI coder service, event system
**Architecture Pattern**: Process Integration Pattern (extending existing manager)
**Similar Reference**: `internal/process/manager.go` - Process lifecycle management

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [internal/process/manager.go, internal/process/process.go, pkg/events/events.go]
modify_files: [internal/process/manager.go]
create_files: [
  /internal/process/ai_coder_integration.go,
  /internal/process/ai_coder_process.go,
  /internal/process/ai_coder_events.go
]
# Total: 4 files (1 modify, 3 create) - targeted process integration
```

**Existing Patterns to Follow**:
- `internal/process/manager.go` - Process lifecycle management and event emission
- `internal/process/process.go` - Process wrapper with status tracking
- Event handling patterns for process status updates

**Dependencies Context**:
- Integration with Task 01 (Core Service) - AI coder manager interface
- Extension of existing process management infrastructure
- Event system integration for status coordination

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/process/manager.go              # Add AI coder process registration
  - /internal/process/ai_coder_integration.go # AI coder process factory
  - /internal/process/ai_coder_process.go     # AI coder process wrapper
  - /internal/process/ai_coder_events.go      # Process event integration

direct_dependencies:
  - /internal/aicoder/manager.go              # Core AI coder service (from Task 01)
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/tui/model.go                    # TUI process display updates
  - /pkg/events/events.go                     # Event system integration
  - /cmd/main.go                              # Process manager initialization

check_documentation:
  - /docs/process-management.md               # Process documentation updates
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/mcp/                            # MCP system separate integration
  - /internal/proxy/                          # Proxy system unrelated
  - /internal/discovery/                      # Discovery system unrelated
  - /internal/logs/                           # Log system separate integration

ignore_search_patterns:
  - "**/testdata/**"                          # Test data files
  - "**/vendor/**"                            # Third-party dependencies
  - "**/node_modules/**"                      # JavaScript dependencies
```

**Boundary Analysis Results**:
- **Usage Count**: Limited to process subsystem integration
- **Scope Assessment**: MODERATE scope - extends established process patterns
- **Impact Radius**: 1 core file to modify, 3 new files for integration

### External Context Sources (from master research)
**Primary Documentation**:
- [Go Concurrency Patterns](https://blog.golang.org/pipelines) - Process coordination patterns
- [Process Management Best Practices](https://golang.org/doc/effective_go.html#goroutines) - Lifecycle management
- [Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html) - Event coordination

**Standards Applied**:
- Process wrapper pattern for external process integration
- Event-driven status synchronization between systems
- Resource cleanup and lifecycle management

**Reference Implementation**:
- Existing process management patterns in `internal/process/manager.go`
- Process status tracking and event emission
- Thread-safe process registration and cleanup

## Task Requirements
**Objective**: Integrate AI coder processes with existing process management system for unified monitoring

**Success Criteria**:
- [ ] AI coder processes appear in standard process views and tools
- [ ] Process status synchronization between AI coder service and process manager
- [ ] Event coordination for process lifecycle (start, stop, status changes)
- [ ] Resource tracking integration (memory, CPU, disk usage)
- [ ] Process cleanup integration with AI coder workspace management
- [ ] TUI process view displays AI coder processes with special indicators
- [ ] MCP process tools work with AI coder processes

**Integration Areas to Implement**:
1. **Process Registration** - Register AI coder instances as processes
2. **Status Synchronization** - Sync status between AI coder service and process manager
3. **Event Coordination** - Bridge events between systems
4. **Resource Monitoring** - Track AI coder resource usage
5. **Lifecycle Management** - Coordinate start/stop operations

**Validation Commands**:
```bash
# Process Integration Verification
grep -q "ai.coder" internal/process/manager.go          # Integration exists
go build ./internal/process                             # Process package compiles
./brum scripts_status | grep -i "ai.*coder"            # AI coders in process list
go test ./internal/process -v                           # Process tests pass
```

## Implementation Specifications

### Process Manager Integration
```go
// Addition to internal/process/manager.go
import (
    "github.com/standardbeagle/brummer/internal/aicoder"
)

// Add AI coder manager to process manager
type ProcessManager struct {
    // Existing fields...
    processes    map[string]*Process
    mu           sync.RWMutex
    eventBus     *events.EventBus
    
    // Add AI coder integration
    aiCoderMgr   *aicoder.AICoderManager
}

// Initialize AI coder integration
func (pm *ProcessManager) SetAICoderManager(mgr *aicoder.AICoderManager) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    pm.aiCoderMgr = mgr
    
    // Start AI coder process monitoring
    go pm.monitorAICoders()
}

// Monitor AI coder processes and sync with process manager
func (pm *ProcessManager) monitorAICoders() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        if pm.aiCoderMgr == nil {
            continue
        }
        
        coders := pm.aiCoderMgr.ListCoders()
        pm.syncAICoderProcesses(coders)
    }
}

// Sync AI coder processes with process manager
func (pm *ProcessManager) syncAICoderProcesses(coders []*aicoder.AICoderProcess) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    // Create or update process entries for AI coders
    for _, coder := range coders {
        processID := fmt.Sprintf("ai-coder-%s", coder.ID)
        
        if existing, exists := pm.processes[processID]; exists {
            // Update existing process
            existing.updateFromAICoder(coder)
        } else {
            // Create new process entry
            process := NewAICoderProcess(coder)
            pm.processes[processID] = process
            
            // Emit process started event
            pm.eventBus.Emit("process_started", ProcessEvent{
                ProcessID: processID,
                Name:      coder.Name,
                Type:      "ai-coder",
                Status:    string(coder.Status),
                Time:      time.Now(),
            })
        }
    }
    
    // Remove processes for deleted AI coders
    pm.cleanupStaleAICoderProcesses(coders)
}
```

### AI Coder Process Wrapper
```go
// internal/process/ai_coder_process.go
type AICoderProcess struct {
    *Process                    // Embed standard process
    aiCoder  *aicoder.AICoderProcess
    mu       sync.RWMutex
}

func NewAICoderProcess(coder *aicoder.AICoderProcess) *AICoderProcess {
    process := &Process{
        ID:        fmt.Sprintf("ai-coder-%s", coder.ID),
        Name:      fmt.Sprintf("AI Coder: %s", coder.Name),
        Script:    coder.Task,
        Status:    mapAICoderStatus(coder.Status),
        StartTime: coder.CreatedAt,
        Type:      "ai-coder",
    }
    
    return &AICoderProcess{
        Process: process,
        aiCoder: coder,
    }
}

// Map AI coder status to process status
func mapAICoderStatus(status aicoder.AICoderStatus) ProcessStatus {
    switch status {
    case aicoder.StatusCreating:
        return ProcessPending
    case aicoder.StatusRunning:
        return ProcessRunning
    case aicoder.StatusPaused:
        return ProcessStopped
    case aicoder.StatusCompleted:
        return ProcessSuccess
    case aicoder.StatusFailed:
        return ProcessFailed
    case aicoder.StatusStopped:
        return ProcessStopped
    default:
        return ProcessPending
    }
}

// Update process from AI coder state
func (acp *AICoderProcess) updateFromAICoder(coder *aicoder.AICoderProcess) {
    acp.mu.Lock()
    defer acp.mu.Unlock()
    
    oldStatus := acp.Status
    newStatus := mapAICoderStatus(coder.Status)
    
    acp.Status = newStatus
    acp.aiCoder = coder
    
    // Update process-specific fields
    acp.Progress = coder.Progress
    acp.LastUpdate = time.Now()
    
    // Emit status change event if changed
    if oldStatus != newStatus {
        // Event emission handled by process manager
    }
}

// Get AI coder specific information
func (acp *AICoderProcess) GetAICoderInfo() map[string]interface{} {
    acp.mu.RLock()
    defer acp.mu.RUnlock()
    
    return map[string]interface{}{
        "provider":      acp.aiCoder.Provider,
        "workspace":     acp.aiCoder.WorkspaceDir,
        "task":          acp.aiCoder.Task,
        "progress":      acp.aiCoder.Progress,
        "created_at":    acp.aiCoder.CreatedAt,
        "updated_at":    acp.aiCoder.UpdatedAt,
    }
}

// Control operations that delegate to AI coder manager
func (acp *AICoderProcess) Stop() error {
    // This would require access to the AI coder manager
    // Implementation depends on how manager reference is maintained
    return fmt.Errorf("AI coder stop operations must go through AI coder manager")
}

func (acp *AICoderProcess) Pause() error {
    return fmt.Errorf("AI coder pause operations must go through AI coder manager")
}

func (acp *AICoderProcess) Resume() error {
    return fmt.Errorf("AI coder resume operations must go through AI coder manager")
}
```

### Event System Integration
```go
// internal/process/ai_coder_events.go
type AICoderEventBridge struct {
    processMgr *ProcessManager
    aiCoderMgr *aicoder.AICoderManager
    eventBus   *events.EventBus
}

func NewAICoderEventBridge(processMgr *ProcessManager, aiCoderMgr *aicoder.AICoderManager, eventBus *events.EventBus) *AICoderEventBridge {
    return &AICoderEventBridge{
        processMgr: processMgr,
        aiCoderMgr: aiCoderMgr,
        eventBus:   eventBus,
    }
}

// Start event bridging between systems
func (bridge *AICoderEventBridge) Start() {
    // Listen for AI coder events and translate to process events
    bridge.eventBus.Subscribe("ai_coder", bridge.handleAICoderEvent)
    
    // Listen for process events and translate to AI coder events if needed
    bridge.eventBus.Subscribe("process", bridge.handleProcessEvent)
}

// Handle AI coder events and create corresponding process events
func (bridge *AICoderEventBridge) handleAICoderEvent(data interface{}) {
    event, ok := data.(aicoder.AICoderEvent)
    if !ok {
        return
    }
    
    processID := fmt.Sprintf("ai-coder-%s", event.CoderID)
    
    // Translate AI coder events to process events
    var processEventType string
    switch event.Type {
    case "created":
        processEventType = "process_started"
    case "status_changed":
        processEventType = "process_status_changed"
    case "completed":
        processEventType = "process_completed"
    case "failed":
        processEventType = "process_failed"
    case "deleted":
        processEventType = "process_stopped"
    default:
        return // Unknown event type
    }
    
    // Emit corresponding process event
    bridge.eventBus.Emit(processEventType, ProcessEvent{
        ProcessID: processID,
        Name:      fmt.Sprintf("AI Coder %s", event.CoderID),
        Type:      "ai-coder",
        Status:    event.Status,
        Message:   event.Message,
        Time:      event.Time,
    })
}

// Handle process control events for AI coders
func (bridge *AICoderEventBridge) handleProcessEvent(data interface{}) {
    event, ok := data.(ProcessEvent)
    if !ok {
        return
    }
    
    // Only handle AI coder processes
    if event.Type != "ai-coder" {
        return
    }
    
    // Extract AI coder ID from process ID
    if !strings.HasPrefix(event.ProcessID, "ai-coder-") {
        return
    }
    coderID := strings.TrimPrefix(event.ProcessID, "ai-coder-")
    
    // Handle process control events
    switch event.Type {
    case "process_stop_requested":
        if err := bridge.aiCoderMgr.StopCoder(coderID); err != nil {
            bridge.eventBus.Emit("process_error", ProcessEvent{
                ProcessID: event.ProcessID,
                Message:   fmt.Sprintf("Failed to stop AI coder: %v", err),
                Time:      time.Now(),
            })
        }
        
    case "process_pause_requested":
        if err := bridge.aiCoderMgr.PauseCoder(coderID); err != nil {
            bridge.eventBus.Emit("process_error", ProcessEvent{
                ProcessID: event.ProcessID,
                Message:   fmt.Sprintf("Failed to pause AI coder: %v", err),
                Time:      time.Now(),
            })
        }
        
    case "process_resume_requested":
        if err := bridge.aiCoderMgr.ResumeCoder(coderID); err != nil {
            bridge.eventBus.Emit("process_error", ProcessEvent{
                ProcessID: event.ProcessID,
                Message:   fmt.Sprintf("Failed to resume AI coder: %v", err),
                Time:      time.Now(),
            })
        }
    }
}
```

### Integration Factory
```go
// internal/process/ai_coder_integration.go
type AICoderIntegration struct {
    processMgr  *ProcessManager
    aiCoderMgr  *aicoder.AICoderManager
    eventBridge *AICoderEventBridge
    eventBus    *events.EventBus
}

func NewAICoderIntegration(processMgr *ProcessManager, eventBus *events.EventBus) *AICoderIntegration {
    return &AICoderIntegration{
        processMgr: processMgr,
        eventBus:   eventBus,
    }
}

// Initialize integration with AI coder manager
func (integration *AICoderIntegration) Initialize(aiCoderMgr *aicoder.AICoderManager) error {
    integration.aiCoderMgr = aiCoderMgr
    
    // Set up AI coder manager in process manager
    integration.processMgr.SetAICoderManager(aiCoderMgr)
    
    // Create and start event bridge
    integration.eventBridge = NewAICoderEventBridge(
        integration.processMgr,
        aiCoderMgr,
        integration.eventBus,
    )
    integration.eventBridge.Start()
    
    return nil
}

// Get AI coder processes for display
func (integration *AICoderIntegration) GetAICoderProcesses() []*AICoderProcess {
    if integration.aiCoderMgr == nil {
        return nil
    }
    
    coders := integration.aiCoderMgr.ListCoders()
    processes := make([]*AICoderProcess, len(coders))
    
    for i, coder := range coders {
        processes[i] = NewAICoderProcess(coder)
    }
    
    return processes
}

// Control AI coder through process interface
func (integration *AICoderIntegration) ControlAICoder(coderID, action string) error {
    if integration.aiCoderMgr == nil {
        return fmt.Errorf("AI coder manager not initialized")
    }
    
    switch action {
    case "start":
        return integration.aiCoderMgr.StartCoder(coderID)
    case "stop":
        return integration.aiCoderMgr.StopCoder(coderID)
    case "pause":
        return integration.aiCoderMgr.PauseCoder(coderID)
    case "resume":
        return integration.aiCoderMgr.ResumeCoder(coderID)
    default:
        return fmt.Errorf("unsupported action: %s", action)
    }
}

// Get AI coder process status for display
func (integration *AICoderIntegration) GetProcessStatus(processID string) (*ProcessStatus, error) {
    if !strings.HasPrefix(processID, "ai-coder-") {
        return nil, fmt.Errorf("not an AI coder process")
    }
    
    coderID := strings.TrimPrefix(processID, "ai-coder-")
    
    if integration.aiCoderMgr == nil {
        return nil, fmt.Errorf("AI coder manager not initialized")
    }
    
    coder, exists := integration.aiCoderMgr.GetCoder(coderID)
    if !exists {
        return nil, fmt.Errorf("AI coder not found")
    }
    
    return &ProcessStatus{
        ID:       processID,
        Name:     fmt.Sprintf("AI Coder: %s", coder.Name),
        Status:   mapAICoderStatus(coder.Status),
        Progress: coder.Progress,
        Runtime:  time.Since(coder.CreatedAt),
        Extra:    map[string]interface{}{
            "provider":   coder.Provider,
            "workspace":  coder.WorkspaceDir,
            "task":       coder.Task,
        },
    }, nil
}
```

## Risk Mitigation (from master analysis)
**Medium-Risk Mitigations**:
- Integration complexity - Follow established process manager patterns - Testing: Integration tests with both systems
- Process coordination - Use event-driven architecture for loose coupling - Recovery: Graceful degradation if one system fails
- Status synchronization - Implement eventual consistency with periodic sync - Monitoring: Status drift detection and correction

**Context Validation**:
- [ ] Process manager patterns from `internal/process/manager.go` successfully applied
- [ ] Event integration maintains system responsiveness
- [ ] AI coder processes appear correctly in existing process views

## Integration with Other Tasks
**Dependencies**: Task 01 (Core Service) - Requires AICoderManager interface
**Integration Points**: 
- Task 04 (TUI Integration) will display integrated AI coder processes
- Task 02 (MCP Tools) will control processes through unified interface  
- Task 06 (Event System) extends this event coordination

**Shared Context**: Process integration enables unified monitoring and control

## Execution Notes
- **Start Pattern**: Use existing process manager patterns from `internal/process/manager.go`
- **Key Context**: Focus on loose coupling through events rather than tight integration
- **Integration Test**: Verify AI coder processes appear in standard process views
- **Review Focus**: Event coordination and status synchronization accuracy

This task creates seamless integration between AI coder instances and the existing process management system, enabling unified monitoring, control, and status tracking across the Brummer development environment.