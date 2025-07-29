# Task: Event System Extensions for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 3 files - Within target range (3-7 files)
**Estimated Time**: 18 minutes - Within target (15-30min)
**Token Estimate**: 75k tokens - Within target (<150k)
**Complexity Level**: 2 (Moderate) - Event system extension with async patterns
**Parallelization Benefit**: MEDIUM - Requires core service completion
**Atomicity Assessment**: ✅ ATOMIC - Complete event system extension
**Boundary Analysis**: ✅ CLEAR - Extends existing event infrastructure cleanly

## Persona Assignment
**Persona**: Software Engineer (Event Systems/Async)
**Expertise Required**: Event-driven architecture, Go channels, async patterns
**Worktree**: `~/work/worktrees/agentic-ai-coders/06-event-system/`

## Context Summary
**Risk Level**: MEDIUM (async complexity, event coordination)
**Integration Points**: Event bus, AI coder service, TUI, MCP tools
**Architecture Pattern**: Event Extension Pattern (from existing event system)
**Similar Reference**: `pkg/events/events.go` - Event bus and handler patterns

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [pkg/events/events.go, internal/logs/store.go]
modify_files: [pkg/events/events.go]
create_files: [
  /pkg/events/ai_coder_events.go,
  /pkg/events/ai_coder_handlers.go
]
# Total: 3 files (1 modify, 2 create) - focused event extension
```

**Existing Patterns to Follow**:
- `pkg/events/events.go` - Event bus architecture and registration patterns
- Event handler signature: `func(data interface{})`
- Async event processing with worker pools

**Dependencies Context**:
- Integration with Task 01 (Core Service) - AI coder event emission
- Extension of existing event bus infrastructure
- Event coordination for all AI coder operations

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /pkg/events/events.go                     # Add AI coder event types
  - /pkg/events/ai_coder_events.go            # AI coder event definitions
  - /pkg/events/ai_coder_handlers.go          # Specialized event handlers

direct_dependencies:
  - /internal/aicoder/manager.go              # Event emission from AI coder service
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/tui/model.go                    # TUI event subscription patterns
  - /internal/process/manager.go              # Process event coordination
  - /internal/mcp/server.go                   # MCP event integration

check_documentation:
  - /docs/events.md                           # Event system documentation
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/proxy/                          # Proxy events separate system
  - /internal/discovery/                      # Discovery events separate
  - /internal/logs/                           # Log events separate handling

ignore_search_patterns:
  - "**/testdata/**"                          # Test data files
  - "**/vendor/**"                            # Third-party code
  - "**/node_modules/**"                      # JavaScript dependencies
```

**Boundary Analysis Results**:
- **Usage Count**: Limited to event subsystem extension
- **Scope Assessment**: LIMITED scope - extends well-defined event patterns
- **Impact Radius**: 1 core file to modify, 2 new files for event types

### External Context Sources (from master research)
**Primary Documentation**:
- [Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html) - Event design patterns
- [Go Concurrency Patterns](https://blog.golang.org/pipelines) - Async event processing
- [Observer Pattern](https://refactoring.guru/design-patterns/observer) - Event notification patterns

**Standards Applied**:
- Event naming: `ai_coder_*` prefix for consistency
- Async processing with buffered channels
- Type-safe event data structures

**Reference Implementation**:
- Existing event types and handler patterns in `pkg/events/events.go`
- Event subscription and emission patterns
- Worker pool patterns for async processing

## Task Requirements
**Objective**: Extend event system with comprehensive AI coder event types and specialized handlers

**Success Criteria**:
- [ ] AI coder event types integrated with existing event bus
- [ ] Specialized event handlers for AI coder state transitions
- [ ] Event aggregation and filtering for AI coder operations
- [ ] Integration with TUI for real-time AI coder status updates
- [ ] Integration with MCP for external event access
- [ ] Event persistence for AI coder operation history
- [ ] Performance optimization for high-frequency AI coder events

**Event Types to Implement**:
1. **Lifecycle Events** - Created, started, paused, completed, failed, deleted
2. **Progress Events** - Task progress updates, milestone completions
3. **Workspace Events** - File changes, workspace operations
4. **Provider Events** - API calls, rate limiting, errors
5. **Resource Events** - Memory usage, CPU consumption, disk operations

**Validation Commands**:
```bash
# Event System Integration Verification
grep -q "ai_coder_created" pkg/events/events.go         # Event types registered
go build ./pkg/events                                   # Events package compiles
go test ./pkg/events -v                                 # Event tests pass
./brum --events | grep -i "ai.*coder"                  # Events appear in debug output
```

## Implementation Specifications

### Event Type Definitions
```go
// pkg/events/ai_coder_events.go
import (
    "time"
)

// AI Coder Event Types
const (
    // Lifecycle events
    EventAICoderCreated    = "ai_coder_created"
    EventAICoderStarted    = "ai_coder_started"
    EventAICoderPaused     = "ai_coder_paused"
    EventAICoderResumed    = "ai_coder_resumed"
    EventAICoderCompleted  = "ai_coder_completed"
    EventAICoderFailed     = "ai_coder_failed"
    EventAICoderStopped    = "ai_coder_stopped"
    EventAICoderDeleted    = "ai_coder_deleted"
    
    // Progress events
    EventAICoderProgress   = "ai_coder_progress"
    EventAICoderMilestone  = "ai_coder_milestone"
    EventAICoderOutput     = "ai_coder_output"
    
    // Workspace events
    EventAICoderFileCreated   = "ai_coder_file_created"
    EventAICoderFileModified  = "ai_coder_file_modified"
    EventAICoderFileDeleted   = "ai_coder_file_deleted"
    EventAICoderWorkspaceSync = "ai_coder_workspace_sync"
    
    // Provider events
    EventAICoderAPICall     = "ai_coder_api_call"
    EventAICoderAPIError    = "ai_coder_api_error"
    EventAICoderRateLimit   = "ai_coder_rate_limit"
    
    // Resource events
    EventAICoderResourceUsage = "ai_coder_resource_usage"
    EventAICoderResourceLimit = "ai_coder_resource_limit"
)

// Core AI Coder Event Structure
type AICoderEvent struct {
    Type      string                 `json:"type"`
    CoderID   string                 `json:"coder_id"`
    CoderName string                 `json:"coder_name"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}

// Lifecycle Event Data
type AICoderLifecycleEvent struct {
    AICoderEvent
    PreviousStatus string `json:"previous_status"`
    CurrentStatus  string `json:"current_status"`
    Reason         string `json:"reason,omitempty"`
    ErrorMessage   string `json:"error_message,omitempty"`
}

// Progress Event Data
type AICoderProgressEvent struct {
    AICoderEvent
    Progress    float64 `json:"progress"`
    Stage       string  `json:"stage"`
    Description string  `json:"description"`
    Milestone   string  `json:"milestone,omitempty"`
}

// Workspace Event Data
type AICoderWorkspaceEvent struct {
    AICoderEvent
    Operation    string `json:"operation"`
    FilePath     string `json:"file_path"`
    FileSize     int64  `json:"file_size,omitempty"`
    ContentHash  string `json:"content_hash,omitempty"`
}

// Provider Event Data
type AICoderProviderEvent struct {
    AICoderEvent
    Provider     string        `json:"provider"`
    Model        string        `json:"model"`
    TokensUsed   int          `json:"tokens_used,omitempty"`
    Duration     time.Duration `json:"duration,omitempty"`
    ErrorCode    string       `json:"error_code,omitempty"`
    ErrorMessage string       `json:"error_message,omitempty"`
}

// Resource Event Data
type AICoderResourceEvent struct {
    AICoderEvent
    MemoryMB     int64 `json:"memory_mb"`
    CPUPercent   float64 `json:"cpu_percent"`
    DiskUsageMB  int64 `json:"disk_usage_mb"`
    NetworkBytes int64 `json:"network_bytes"`
    FileCount    int   `json:"file_count"`
}

// Event Factory Functions
func NewAICoderLifecycleEvent(coderID, coderName, eventType, prevStatus, currStatus, reason string) *AICoderLifecycleEvent {
    return &AICoderLifecycleEvent{
        AICoderEvent: AICoderEvent{
            Type:      eventType,
            CoderID:   coderID,
            CoderName: coderName,
            Timestamp: time.Now(),
        },
        PreviousStatus: prevStatus,
        CurrentStatus:  currStatus,
        Reason:         reason,
    }
}

func NewAICoderProgressEvent(coderID, coderName string, progress float64, stage, description string) *AICoderProgressEvent {
    return &AICoderProgressEvent{
        AICoderEvent: AICoderEvent{
            Type:      EventAICoderProgress,
            CoderID:   coderID,
            CoderName: coderName,
            Timestamp: time.Now(),
        },
        Progress:    progress,
        Stage:       stage,
        Description: description,
    }
}

func NewAICoderWorkspaceEvent(coderID, coderName, operation, filePath string) *AICoderWorkspaceEvent {
    return &AICoderWorkspaceEvent{
        AICoderEvent: AICoderEvent{
            Type:      getWorkspaceEventType(operation),
            CoderID:   coderID,
            CoderName: coderName,
            Timestamp: time.Now(),
        },
        Operation: operation,
        FilePath:  filePath,
    }
}

func NewAICoderProviderEvent(coderID, coderName, provider, model string) *AICoderProviderEvent {
    return &AICoderProviderEvent{
        AICoderEvent: AICoderEvent{
            Type:      EventAICoderAPICall,
            CoderID:   coderID,
            CoderName: coderName,
            Timestamp: time.Now(),
        },
        Provider: provider,
        Model:    model,
    }
}

func NewAICoderResourceEvent(coderID, coderName string, memMB int64, cpuPercent float64, diskMB int64) *AICoderResourceEvent {
    return &AICoderResourceEvent{
        AICoderEvent: AICoderEvent{
            Type:      EventAICoderResourceUsage,
            CoderID:   coderID,
            CoderName: coderName,
            Timestamp: time.Now(),
        },
        MemoryMB:    memMB,
        CPUPercent:  cpuPercent,
        DiskUsageMB: diskMB,
    }
}

// Helper functions
func getWorkspaceEventType(operation string) string {
    switch operation {
    case "create":
        return EventAICoderFileCreated
    case "modify":
        return EventAICoderFileModified
    case "delete":
        return EventAICoderFileDeleted
    default:
        return EventAICoderWorkspaceSync
    }
}
```

### Specialized Event Handlers
```go
// pkg/events/ai_coder_handlers.go
import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"
    "time"
)

// AI Coder Event Aggregator
type AICoderEventAggregator struct {
    events     []AICoderEvent
    mu         sync.RWMutex
    maxEvents  int
    eventBus   *EventBus
    
    // Event statistics
    stats      AICoderEventStats
    statsMu    sync.RWMutex
}

type AICoderEventStats struct {
    TotalEvents     int64            `json:"total_events"`
    EventsByType    map[string]int64 `json:"events_by_type"`
    EventsByCoder   map[string]int64 `json:"events_by_coder"`
    LastEvent       time.Time        `json:"last_event"`
    EventsPerMinute float64          `json:"events_per_minute"`
}

func NewAICoderEventAggregator(eventBus *EventBus, maxEvents int) *AICoderEventAggregator {
    aggregator := &AICoderEventAggregator{
        events:    make([]AICoderEvent, 0, maxEvents),
        maxEvents: maxEvents,
        eventBus:  eventBus,
        stats: AICoderEventStats{
            EventsByType:  make(map[string]int64),
            EventsByCoder: make(map[string]int64),
        },
    }
    
    // Register handlers for all AI coder event types
    aggregator.registerHandlers()
    
    return aggregator
}

// Register event handlers
func (a *AICoderEventAggregator) registerHandlers() {
    eventTypes := []string{
        EventAICoderCreated, EventAICoderStarted, EventAICoderPaused,
        EventAICoderResumed, EventAICoderCompleted, EventAICoderFailed,
        EventAICoderStopped, EventAICoderDeleted, EventAICoderProgress,
        EventAICoderMilestone, EventAICoderOutput, EventAICoderFileCreated,
        EventAICoderFileModified, EventAICoderFileDeleted, EventAICoderWorkspaceSync,
        EventAICoderAPICall, EventAICoderAPIError, EventAICoderRateLimit,
        EventAICoderResourceUsage, EventAICoderResourceLimit,
    }
    
    for _, eventType := range eventTypes {
        a.eventBus.Subscribe(eventType, a.handleAICoderEvent)
    }
}

// Handle AI coder events
func (a *AICoderEventAggregator) handleAICoderEvent(data interface{}) {
    event, ok := data.(AICoderEvent)
    if !ok {
        // Try to convert from interface{} with type assertion
        if eventMap, ok := data.(map[string]interface{}); ok {
            event = a.mapToAICoderEvent(eventMap)
        } else {
            log.Printf("Invalid AI coder event data type: %T", data)
            return
        }
    }
    
    // Add to event history
    a.addEvent(event)
    
    // Update statistics
    a.updateStats(event)
    
    // Handle specialized processing
    a.processSpecializedEvent(event)
}

// Add event to history with size limit
func (a *AICoderEventAggregator) addEvent(event AICoderEvent) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    // Add new event
    a.events = append(a.events, event)
    
    // Trim if over limit
    if len(a.events) > a.maxEvents {
        // Remove oldest events
        a.events = a.events[len(a.events)-a.maxEvents:]
    }
}

// Update event statistics
func (a *AICoderEventAggregator) updateStats(event AICoderEvent) {
    a.statsMu.Lock()
    defer a.statsMu.Unlock()
    
    a.stats.TotalEvents++
    a.stats.EventsByType[event.Type]++
    a.stats.EventsByCoder[event.CoderID]++
    a.stats.LastEvent = event.Timestamp
    
    // Calculate events per minute (simple moving average)
    if a.stats.TotalEvents > 1 {
        duration := event.Timestamp.Sub(time.Now().Add(-time.Minute))
        if duration > 0 {
            a.stats.EventsPerMinute = float64(a.stats.TotalEvents) / duration.Minutes()
        }
    }
}

// Process specialized event handling
func (a *AICoderEventAggregator) processSpecializedEvent(event AICoderEvent) {
    switch event.Type {
    case EventAICoderFailed:
        a.handleFailureEvent(event)
    case EventAICoderCompleted:
        a.handleCompletionEvent(event)
    case EventAICoderRateLimit:
        a.handleRateLimitEvent(event)
    case EventAICoderResourceLimit:
        a.handleResourceLimitEvent(event)
    }
}

// Handle failure events
func (a *AICoderEventAggregator) handleFailureEvent(event AICoderEvent) {
    // Log failure for debugging
    log.Printf("AI Coder %s failed: %v", event.CoderID, event.Data)
    
    // Emit aggregated failure alert if multiple failures
    failureCount := a.getRecentFailureCount(event.CoderID)
    if failureCount >= 3 {
        a.eventBus.Emit("ai_coder_failure_alert", map[string]interface{}{
            "coder_id":      event.CoderID,
            "failure_count": failureCount,
            "time":          event.Timestamp,
        })
    }
}

// Handle completion events
func (a *AICoderEventAggregator) handleCompletionEvent(event AICoderEvent) {
    // Calculate completion time from creation
    creationTime := a.getCreationTime(event.CoderID)
    if !creationTime.IsZero() {
        duration := event.Timestamp.Sub(creationTime)
        
        // Emit completion metrics
        a.eventBus.Emit("ai_coder_completion_metrics", map[string]interface{}{
            "coder_id": event.CoderID,
            "duration": duration,
            "time":     event.Timestamp,
        })
    }
}

// Handle rate limit events
func (a *AICoderEventAggregator) handleRateLimitEvent(event AICoderEvent) {
    // Emit system-wide rate limit warning
    a.eventBus.Emit("ai_coder_system_warning", map[string]interface{}{
        "type":     "rate_limit",
        "coder_id": event.CoderID,
        "message":  "AI provider rate limit reached",
        "time":     event.Timestamp,
    })
}

// Handle resource limit events  
func (a *AICoderEventAggregator) handleResourceLimitEvent(event AICoderEvent) {
    // Emit resource warning
    a.eventBus.Emit("ai_coder_system_warning", map[string]interface{}{
        "type":     "resource_limit",
        "coder_id": event.CoderID,
        "message":  "AI coder resource limit exceeded",
        "time":     event.Timestamp,
    })
}

// Query methods for event history
func (a *AICoderEventAggregator) GetEvents(filter AICoderEventFilter) []AICoderEvent {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    var filtered []AICoderEvent
    for _, event := range a.events {
        if filter.matches(event) {
            filtered = append(filtered, event)
        }
    }
    
    return filtered
}

func (a *AICoderEventAggregator) GetStats() AICoderEventStats {
    a.statsMu.RLock()
    defer a.statsMu.RUnlock()
    
    return a.stats
}

// Event Filter
type AICoderEventFilter struct {
    CoderID   string
    EventType string
    Since     time.Time
    Until     time.Time
}

func (f AICoderEventFilter) matches(event AICoderEvent) bool {
    if f.CoderID != "" && event.CoderID != f.CoderID {
        return false
    }
    
    if f.EventType != "" && event.Type != f.EventType {
        return false
    }
    
    if !f.Since.IsZero() && event.Timestamp.Before(f.Since) {
        return false
    }
    
    if !f.Until.IsZero() && event.Timestamp.After(f.Until) {
        return false
    }
    
    return true
}

// Helper methods
func (a *AICoderEventAggregator) getRecentFailureCount(coderID string) int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    count := 0
    cutoff := time.Now().Add(-10 * time.Minute) // Recent = last 10 minutes
    
    for _, event := range a.events {
        if event.CoderID == coderID && 
           event.Type == EventAICoderFailed && 
           event.Timestamp.After(cutoff) {
            count++
        }
    }
    
    return count
}

func (a *AICoderEventAggregator) getCreationTime(coderID string) time.Time {
    a.mu.RLock()
    defer a.mu.RUnlock()
    
    for _, event := range a.events {
        if event.CoderID == coderID && event.Type == EventAICoderCreated {
            return event.Timestamp
        }
    }
    
    return time.Time{}
}

// Convert map to AICoderEvent
func (a *AICoderEventAggregator) mapToAICoderEvent(eventMap map[string]interface{}) AICoderEvent {
    event := AICoderEvent{
        Data: make(map[string]interface{}),
    }
    
    if eventType, ok := eventMap["type"].(string); ok {
        event.Type = eventType
    }
    
    if coderID, ok := eventMap["coder_id"].(string); ok {
        event.CoderID = coderID
    }
    
    if coderName, ok := eventMap["coder_name"].(string); ok {
        event.CoderName = coderName
    }
    
    if timestamp, ok := eventMap["timestamp"].(time.Time); ok {
        event.Timestamp = timestamp
    } else {
        event.Timestamp = time.Now()
    }
    
    // Copy remaining data
    for k, v := range eventMap {
        if k != "type" && k != "coder_id" && k != "coder_name" && k != "timestamp" {
            event.Data[k] = v
        }
    }
    
    return event
}
```

### Event Bus Integration
```go
// Addition to pkg/events/events.go
// Add AI coder event types to existing event bus

// Register AI coder event aggregator
func (eb *EventBus) RegisterAICoderAggregator(maxEvents int) *AICoderEventAggregator {
    return NewAICoderEventAggregator(eb, maxEvents)
}

// AI coder specific subscription helper
func (eb *EventBus) SubscribeAICoderEvents(coderID string, handler EventHandler) {
    // Subscribe to all AI coder events for specific coder
    eventTypes := []string{
        EventAICoderCreated, EventAICoderStarted, EventAICoderPaused,
        EventAICoderResumed, EventAICoderCompleted, EventAICoderFailed,
        EventAICoderStopped, EventAICoderDeleted, EventAICoderProgress,
    }
    
    for _, eventType := range eventTypes {
        eb.Subscribe(eventType, func(data interface{}) {
            if event, ok := data.(AICoderEvent); ok && event.CoderID == coderID {
                handler(data)
            }
        })
    }
}

// Emit AI coder event with validation
func (eb *EventBus) EmitAICoderEvent(event AICoderEvent) {
    if event.Type == "" {
        log.Printf("Warning: AI coder event missing type")
        return
    }
    
    if event.CoderID == "" {
        log.Printf("Warning: AI coder event missing coder ID")
        return
    }
    
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }
    
    eb.Emit(event.Type, event)
}
```

## Risk Mitigation (from master analysis)
**Medium-Risk Mitigations**:
- Async complexity - Use established event bus patterns with worker pools - Testing: Concurrent event processing tests
- Event coordination - Implement event ordering and deduplication - Recovery: Event replay mechanism for failures
- Memory management - Bounded event history with LRU eviction - Monitoring: Event memory usage tracking

**Context Validation**:
- [ ] Event bus patterns from `pkg/events/events.go` successfully extended
- [ ] Event processing maintains system responsiveness
- [ ] Event aggregation provides useful insights without performance impact

## Integration with Other Tasks
**Dependencies**: Task 01 (Core Service) - Requires AI coder event emission
**Integration Points**: 
- Task 04 (TUI Integration) will subscribe to events for real-time updates
- Task 05 (Process Integration) will coordinate events between systems
- Task 02 (MCP Tools) will access event history via tools

**Shared Context**: Event system becomes the nervous system for all AI coder operations

## Execution Notes
- **Start Pattern**: Use existing event bus patterns from `pkg/events/events.go`
- **Key Context**: Focus on event aggregation and intelligent filtering
- **Performance Focus**: Implement bounded event history and efficient filtering
- **Review Focus**: Event handler performance and memory usage patterns

This task creates a comprehensive event system that provides real-time visibility into all AI coder operations while maintaining high performance and system responsiveness.