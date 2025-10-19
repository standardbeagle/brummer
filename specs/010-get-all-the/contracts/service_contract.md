# Service Contract: Stateless Command Creators

**Purpose**: Define behavioral contract for all TUI services

## Contract Requirements

### 1. Statelessness

**Requirement**: Services MUST NOT store internal state

**Test Cases**:
```go
func TestServiceStatelessness(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Use reflection to check for state fields
    serviceType := reflect.TypeOf(service)

    for i := 0; i < serviceType.NumField(); i++ {
        field := serviceType.Field(i)

        // ASSERT: No slice/map/channel fields (indicate state)
        assert.NotEqual(t, reflect.Slice, field.Type.Kind())
        assert.NotEqual(t, reflect.Map, field.Type.Kind())
        assert.NotEqual(t, reflect.Chan, field.Type.Kind())

        // Configuration fields are allowed (const-like)
        // But NO mutable state
    }
}
```

### 2. Return Commands

**Requirement**: Services MUST return tea.Cmd for async operations

**Test Cases**:
```go
func TestServiceReturnsCommands(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Call service method
    cmd := service.ParseAndGroupError(testInput)

    // ASSERT: Returns tea.Cmd
    assert.NotNil(t, cmd)

    // Execute command and verify message
    msg := cmd()
    assert.IsType(t, messages.ErrorGroupedMsg{}, msg)
}
```

### 3. No Direct State Modifications

**Requirement**: Services MUST NOT modify state objects directly

**Test Cases**:
```go
func TestServiceDoesNotModifyState(t *testing.T) {
    service := services.NewErrorGroupingService()
    state := createTestState()
    originalVersion := state.GetVersion()

    // Call service method
    cmd := service.ParseAndGroupError(testInput)

    // Execute command
    msg := cmd()

    // ASSERT: State unchanged by service
    assert.Equal(t, originalVersion, state.GetVersion())

    // State is modified in Update() when msg is processed
    newState := state.AddGroupedError(msg.(messages.ErrorGroupedMsg).Error)
    assert.Equal(t, originalVersion+1, newState.GetVersion())
}
```

### 4. Deterministic Results

**Requirement**: Services with same inputs MUST produce same outputs

**Test Cases**:
```go
func TestServiceDeterministic(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Call multiple times with same input
    cmd1 := service.ParseAndGroupError(testInput)
    cmd2 := service.ParseAndGroupError(testInput)
    cmd3 := service.ParseAndGroupError(testInput)

    msg1 := cmd1()
    msg2 := cmd2()
    msg3 := cmd3()

    // ASSERT: Identical results
    assert.Equal(t, msg1, msg2)
    assert.Equal(t, msg2, msg3)
}
```

### 5. Error Handling

**Requirement**: Services MUST return error messages on failure

**Test Cases**:
```go
func TestServiceErrorHandling(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Invalid input
    cmd := service.ParseAndGroupError("")

    msg := cmd()

    // ASSERT: Returns error message
    errorMsg, ok := msg.(messages.ErrorMsg)
    assert.True(t, ok)
    assert.NotEmpty(t, errorMsg.Error)
}
```

### 6. No Goroutines in Service

**Requirement**: Services MUST NOT launch goroutines (use tea.Cmd pattern instead)

**Test Cases**:
```go
func TestServiceNoGoroutines(t *testing.T) {
    // Count goroutines before
    before := runtime.NumGoroutine()

    service := services.NewErrorGroupingService()
    cmd := service.ParseAndGroupError(testInput)

    // Execute command
    _ = cmd()

    // Small delay to let any goroutines start
    time.Sleep(10 * time.Millisecond)

    // Count goroutines after
    after := runtime.NumGoroutine()

    // ASSERT: No new goroutines
    assert.Equal(t, before, after)
}
```

### 7. Configuration Only

**Requirement**: Services MAY have configuration fields, but no mutable state

**Test Cases**:
```go
func TestServiceConfigurationOnly(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Configuration is allowed
    config := service.GetConfig()
    assert.NotNil(t, config)

    // But calling service methods doesn't change config
    cmd1 := service.ParseAndGroupError(testInput1)
    cmd2 := service.ParseAndGroupError(testInput2)

    _ = cmd1()
    _ = cmd2()

    // ASSERT: Config unchanged
    assert.Equal(t, config, service.GetConfig())
}
```

### 8. Thread Safety

**Requirement**: Services MUST be safe for concurrent calls

**Test Cases**:
```go
func TestServiceConcurrentCalls(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Launch 100 concurrent calls
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(input string) {
            defer wg.Done()
            cmd := service.ParseAndGroupError(input)
            _ = cmd()
        }(fmt.Sprintf("error-%d", i))
    }

    // ASSERT: No race conditions (run with -race flag)
    wg.Wait()
}
```

### 9. Performance

**Requirement**: Service operations MUST complete in <50ms

**Test Cases**:
```go
func BenchmarkServicePerformance(b *testing.B) {
    service := services.NewErrorGroupingService()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cmd := service.ParseAndGroupError(testInput)
        _ = cmd()
    }

    avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
    avgMs := float64(avgNs) / 1e6

    // ASSERT: <50ms average
    if avgMs > 50.0 {
        b.Fatalf("Service too slow: %.2fms (limit: 50ms)", avgMs)
    }
}
```

### 10. Command Batching

**Requirement**: Services MAY return batched commands using tea.Batch

**Test Cases**:
```go
func TestServiceCommandBatching(t *testing.T) {
    service := services.NewErrorGroupingService()

    // Service that returns multiple operations
    cmd := service.ProcessMultipleErrors([]string{input1, input2, input3})

    // ASSERT: Returns valid command
    assert.NotNil(t, cmd)

    // Execute and verify all messages
    msg := cmd()
    // tea.Batch returns nil, messages sent separately
    // Verify in integration test instead
}
```

## Anti-Patterns to Avoid

**DO NOT**:
- Store processed data in service fields
- Launch goroutines from service methods
- Use mutexes in services
- Store counters, caches, or history
- Directly modify passed state objects
- Make blocking I/O calls without returning tea.Cmd

**DO**:
- Return tea.Cmd for all async operations
- Keep services pure and stateless
- Store only configuration (immutable)
- Use tea.Batch for multiple operations
- Return error messages on failure

## Contract Violations

**If any test fails, the service implementation violates the contract and MUST be refactored**

## Coverage Requirement

All service contract tests MUST have 100% code coverage for service methods.

---
*Contract Version: 1.0*
