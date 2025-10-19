# View Contract: Pure Rendering Functions

**Purpose**: Define behavioral contract for all TUI view components

## Contract Requirements

### 1. Pure Functions

**Requirement**: View functions MUST be pure (no side effects)

**Test Cases**:
```go
func TestViewPurity(t *testing.T) {
    state := createTestState()

    // Call view multiple times
    output1 := components.ErrorsView{}.ViewWithState(state)
    output2 := components.ErrorsView{}.ViewWithState(state)
    output3 := components.ErrorsView{}.ViewWithState(state)

    // ASSERT: Identical outputs (deterministic)
    assert.Equal(t, output1, output2)
    assert.Equal(t, output2, output3)

    // ASSERT: State unchanged
    assert.Equal(t, createTestState(), state)
}
```

### 2. No State Modifications

**Requirement**: Views MUST NOT modify state objects

**Test Cases**:
```go
func TestViewDoesNotModifyState(t *testing.T) {
    state := createTestState()
    originalVersion := state.GetVersion()
    originalData := state.GetGroupedErrors()

    // Render view
    _ = components.ErrorsView{}.ViewWithState(state)

    // ASSERT: State unchanged
    assert.Equal(t, originalVersion, state.GetVersion())
    assert.Equal(t, originalData, state.GetGroupedErrors())
}
```

### 3. No Blocking Operations

**Requirement**: Views MUST NOT perform blocking operations (critical for deadlock prevention)

**Test Cases**:
```go
func TestViewDoesNotBlock(t *testing.T) {
    state := createTestState()

    // Render with timeout
    done := make(chan bool)
    go func() {
        _ = components.ErrorsView{}.ViewWithState(state)
        done <- true
    }()

    // ASSERT: Completes within 5ms
    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Millisecond):
        t.Fatal("View rendering blocked or took >5ms")
    }
}
```

### 4. No I/O Operations

**Requirement**: Views MUST NOT perform I/O (file, network, database)

**Test Cases**:
```go
func TestViewNoIO(t *testing.T) {
    // Monitor file descriptors before view
    fdsBefore := getOpenFileDescriptors()

    // Render view
    _ = components.ErrorsView{}.ViewWithState(createTestState())

    // Monitor file descriptors after view
    fdsAfter := getOpenFileDescriptors()

    // ASSERT: No new file descriptors opened
    assert.Equal(t, fdsBefore, fdsAfter)
}
```

### 5. Performance

**Requirement**: Views MUST render in <5ms

**Test Cases**:
```go
func BenchmarkViewPerformance(b *testing.B) {
    state := createLargeTestState(1000) // 1000 errors

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = components.ErrorsView{}.ViewWithState(state)
    }

    // Calculate average time
    avgNs := b.Elapsed().Nanoseconds() / int64(b.N)
    avgMs := float64(avgNs) / 1e6

    // ASSERT: <5ms average
    if avgMs > 5.0 {
        b.Fatalf("View rendering too slow: %.2fms (limit: 5ms)", avgMs)
    }
}
```

### 6. Nil Safety

**Requirement**: Views MUST handle nil state gracefully

**Test Cases**:
```go
func TestViewNilState(t *testing.T) {
    view := components.ErrorsView{}

    // ASSERT: Does not panic with nil
    assert.NotPanics(t, func() {
        output := view.ViewWithState(nil)
        assert.Contains(t, output, "not available")
    })
}
```

### 7. Empty State

**Requirement**: Views MUST render empty state gracefully

**Test Cases**:
```go
func TestViewEmptyState(t *testing.T) {
    emptyState := state.NewErrorsState()

    output := components.ErrorsView{}.ViewWithState(&state.MasterData{
        ErrorsState: emptyState,
    })

    // ASSERT: Shows empty message
    assert.Contains(t, output, "No errors")
    // ASSERT: Does not crash
    assert.NotEmpty(t, output)
}
```

### 8. Large State

**Requirement**: Views MUST handle large datasets without crashes

**Test Cases**:
```go
func TestViewLargeState(t *testing.T) {
    largeState := createTestStateWithErrors(10000) // 10k errors

    // ASSERT: Does not panic
    assert.NotPanics(t, func() {
        output := components.ErrorsView{}.ViewWithState(largeState)
        assert.NotEmpty(t, output)
    })
}
```

### 9. Concurrent Rendering

**Requirement**: Views MUST be safe for concurrent calls

**Test Cases**:
```go
func TestViewConcurrentRendering(t *testing.T) {
    state := createTestState()

    // Launch 50 concurrent renders
    var wg sync.WaitGroup
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = components.ErrorsView{}.ViewWithState(state)
        }()
    }

    // ASSERT: No race conditions (run with -race flag)
    wg.Wait()
}
```

### 10. Terminal Size Handling

**Requirement**: Views MUST handle various terminal sizes

**Test Cases**:
```go
func TestViewTerminalSizes(t *testing.T) {
    state := createTestState()
    view := components.ErrorsView{}

    testCases := []struct {
        width  int
        height int
    }{
        {80, 24},   // Standard
        {120, 40},  // Large
        {40, 10},   // Small
        {200, 60},  // Very large
    }

    for _, tc := range testCases {
        view.SetSize(tc.width, tc.height)

        // ASSERT: Does not panic
        assert.NotPanics(t, func() {
            output := view.ViewWithState(state)
            assert.NotEmpty(t, output)
        })
    }
}
```

## Contract Violations

**If any test fails, the view implementation violates the contract and MUST be fixed**

Key violations to watch for:
- Calling external services from View()
- Modifying state objects
- Using mutexes or locks in rendering
- Performing database queries
- Making HTTP requests
- Reading files

## Coverage Requirement

All view contract tests MUST have 90% code coverage for view rendering methods.

---
*Contract Version: 1.0*
