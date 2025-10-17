# Research Document: Bubble Tea MVU Pattern Implementation

## Executive Summary
This document consolidates research findings for migrating Brummer's TUI to a pure Bubble Tea MVU pattern, eliminating goroutines and achieving <50ms UI response times.

## Research Areas

### 1. Bubble Tea Command Patterns
**Decision**: Implement typed command wrappers with explicit result messages
**Rationale**:
- Type safety ensures compile-time verification of command/message pairs
- Explicit result messages enable proper error handling and retry logic
- Command wrappers encapsulate context for operation ordering

**Alternatives Considered**:
- Generic interface{} commands: Rejected due to loss of type safety
- Channel-based communication: Violates Bubble Tea's no-goroutine rule
- Direct callbacks: Creates coupling and complicates testing

**Implementation Pattern**:
```go
type IOCommand struct {
    ID        string
    Operation func() (interface{}, error)
    Context   OperationContext
}

func (c IOCommand) Run() tea.Msg {
    result, err := c.Operation()
    return IOResultMsg{
        ID:      c.ID,
        Result:  result,
        Error:   err,
        Context: c.Context,
    }
}
```

### 2. Command Tumbler Behavior
**Decision**: Leverage Bubble Tea's internal command executor with operation tagging
**Rationale**:
- Command tumbler handles concurrency safely without explicit goroutines
- Operation IDs enable tracking and correlation of results
- Built-in backpressure prevents overwhelming the system

**Key Findings**:
- Commands execute in managed goroutine pool (default size: runtime.NumCPU())
- Results arrive non-deterministically based on operation completion time
- No guarantee of FIFO ordering for command results
- Commands can be batched using tea.Batch() for parallel execution

### 3. Context-Aware Operation Ordering
**Decision**: Implement operation contexts with versioning and dependency tracking
**Rationale**:
- Version numbers enable detection of stale results
- Dependency chains ensure correct ordering where required
- Context metadata supports different strategies per operation type

**Strategies by Operation Type**:
- **UI State Updates**: Last-write-wins with version checking
- **Data Fetching**: Version tracking with stale result rejection
- **Sequential Operations**: Dependency chain with queuing
- **Independent Operations**: Parallel execution with result aggregation

### 4. Migration from Goroutines
**Decision**: Systematic refactoring using command factory pattern
**Rationale**:
- Factory pattern centralizes command creation logic
- Gradual migration minimizes risk and maintains functionality
- Clear separation between old and new patterns during transition

**Migration Steps**:
1. Identify all goroutine usage in TUI components
2. Create command equivalents for each async operation
3. Replace goroutine launches with command returns
4. Update message handlers for command results
5. Remove legacy goroutine helper functions

### 5. Performance Optimization
**Decision**: Implement command batching and result caching
**Rationale**:
- Batching reduces overhead for related operations
- Caching prevents redundant I/O operations
- Debouncing user input reduces command frequency

**Performance Techniques**:
- Batch related commands using tea.Batch()
- Cache frequently accessed data with TTL
- Debounce rapid user input (e.g., typing) to 20ms
- Use tea.Tick() for periodic updates instead of polling

### 6. Error Handling and Retry Logic
**Decision**: Implement smart retry with exponential backoff in commands
**Rationale**:
- Retry logic belongs in command implementation, not Update()
- Exponential backoff prevents thundering herd on failures
- Transient vs permanent error classification guides retry decisions

**Retry Implementation**:
```go
func withRetry(op func() (interface{}, error)) func() (interface{}, error) {
    return func() (interface{}, error) {
        for attempt := 0; attempt < 3; attempt++ {
            result, err := op()
            if err == nil || !isTransient(err) {
                return result, err
            }
            time.Sleep(time.Duration(1<<attempt) * time.Second)
        }
        return op() // Final attempt
    }
}
```

### 7. Testing Strategy
**Decision**: Command-centric testing with time-based assertions
**Rationale**:
- Testing commands in isolation ensures correctness
- Time-based assertions verify <50ms response requirement
- Mock command executor enables deterministic testing

**Test Categories**:
- Unit tests for individual commands
- Integration tests for command chains
- Performance tests for response times
- Race tests for concurrent command execution

## Technical Decisions Summary

| Area | Decision | Impact |
|------|----------|--------|
| Command Pattern | Typed wrappers with explicit messages | Type safety, testability |
| Concurrency | Bubble Tea's command tumbler only | No goroutines, safe execution |
| Ordering | Context-aware strategies | Correct behavior per operation type |
| Migration | Factory pattern with gradual refactoring | Low risk, maintainable |
| Performance | Batching and caching | <50ms response times |
| Error Handling | Smart retry in commands | Resilient, user-friendly |
| Testing | Command-centric with time assertions | Comprehensive coverage |

## Implementation Priorities

1. **Critical Path** (Week 1):
   - Create command factory infrastructure
   - Migrate blocking I/O operations
   - Implement response time monitoring

2. **Core Refactoring** (Week 2):
   - Convert all goroutines to commands
   - Implement operation contexts
   - Add retry logic for network operations

3. **Optimization** (Week 3):
   - Add command batching
   - Implement result caching
   - Performance tuning and testing

## Risk Mitigation

- **Risk**: Breaking existing functionality during migration
  - **Mitigation**: Feature flag for gradual rollout, comprehensive test coverage

- **Risk**: Performance regression from command overhead
  - **Mitigation**: Batching, caching, and performance monitoring from day one

- **Risk**: Complex debugging of non-deterministic behavior
  - **Mitigation**: Comprehensive logging, operation IDs, and debug mode

## Conclusion

The migration to pure Bubble Tea MVU pattern is technically feasible and aligns with constitutional principles. The command pattern provides a clear path for eliminating goroutines while maintaining performance and functionality. The research validates that all requirements can be met with proper implementation of the patterns identified above.