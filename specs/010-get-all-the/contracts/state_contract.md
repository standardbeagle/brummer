# State Contract: Immutable Lock-Free State Pattern

**Purpose**: Define behavioral contract for all TUI state objects

## Contract Requirements

### 1. Immutability

**Requirement**: All state objects MUST be immutable

**Test Cases**:
```go
func TestStateImmutability(t *testing.T) {
    // Create original state
    original := state.NewErrorsState()

    // Perform mutation
    modified := original.AddGroupedError(testError)

    // ASSERT: Original unchanged
    assert.Equal(t, 0, len(original.GroupedErrors))

    // ASSERT: Modified has new data
    assert.Equal(t, 1, len(modified.GroupedErrors))

    // ASSERT: Different memory addresses
    assert.NotEqual(t, original, modified)
}
```

### 2. No Mutexes

**Requirement**: State objects MUST NOT contain mutexes or locks

**Test Cases**:
```go
func TestNoMutexes(t *testing.T) {
    // Use reflection to verify no mutex fields
    stateType := reflect.TypeOf(state.ErrorsState{})

    for i := 0; i < stateType.NumField(); i++ {
        field := stateType.Field(i)
        fieldType := field.Type.String()

        // ASSERT: No mutex types
        assert.NotContains(t, fieldType, "sync.Mutex")
        assert.NotContains(t, fieldType, "sync.RWMutex")
    }
}
```

### 3. Version Tracking

**Requirement**: All state objects MUST increment version on every mutation

**Test Cases**:
```go
func TestVersionIncrement(t *testing.T) {
    original := state.NewErrorsState()
    originalVersion := original.GetVersion()

    modified := original.AddGroupedError(testError)

    // ASSERT: Version incremented
    assert.Equal(t, originalVersion+1, modified.GetVersion())
}
```

### 4. Copy-on-Write

**Requirement**: All update methods MUST return new instances

**Test Cases**:
```go
func TestCopyOnWrite(t *testing.T) {
    original := state.NewErrorsState()

    // Multiple updates
    updated1 := original.AddGroupedError(error1)
    updated2 := original.AddGroupedError(error2)

    // ASSERT: All different instances
    assert.NotEqual(t, original, updated1)
    assert.NotEqual(t, original, updated2)
    assert.NotEqual(t, updated1, updated2)

    // ASSERT: Original still unchanged
    assert.Equal(t, 0, len(original.GroupedErrors))
}
```

### 5. Thread Safety

**Requirement**: Concurrent reads on same state instance MUST be safe

**Test Cases**:
```go
func TestConcurrentReads(t *testing.T) {
    state := createStateWithData()

    // Launch 100 concurrent readers
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Read operations
            _ = state.GetGroupedErrors()
            _ = state.GetSelectedIndex()
            _ = state.GetFilterProcess()
        }()
    }

    // ASSERT: No race conditions (run with -race flag)
    wg.Wait()
}
```

### 6. Validation

**Requirement**: State objects MUST validate invariants

**Test Cases**:
```go
func TestStateValidation(t *testing.T) {
    state := state.NewErrorsState()

    // ASSERT: Valid initial state
    issues := state.Validate()
    assert.Empty(t, issues)

    // Create invalid state (manually)
    invalid := &state.ErrorsState{
        SelectedIndex: 10,  // Out of bounds
        GroupedErrors: []GroupedError{},  // Empty
    }

    // ASSERT: Validation catches issue
    issues = invalid.Validate()
    assert.NotEmpty(t, issues)
    assert.Contains(t, issues[0], "index out of bounds")
}
```

### 7. Retention Limits

**Requirement**: State objects MUST enforce retention limits

**Test Cases**:
```go
func TestRetentionLimit(t *testing.T) {
    state := state.NewErrorsState()
    state.MaxRetention = 10

    // Add 20 errors
    for i := 0; i < 20; i++ {
        state = state.AddGroupedError(createTestError(i))
    }

    // ASSERT: Only last 10 retained
    assert.Equal(t, 10, len(state.GroupedErrors))

    // ASSERT: Oldest discarded (FIFO)
    assert.Equal(t, "error-10", state.GroupedErrors[0].ID)
    assert.Equal(t, "error-19", state.GroupedErrors[9].ID)
}
```

## Contract Violations

**If any test fails, the implementation violates the contract and MUST be fixed before proceeding**

## Coverage Requirement

All state contract tests MUST have 100% code coverage for state mutation methods.

---
*Contract Version: 1.0*
