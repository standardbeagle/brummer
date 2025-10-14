# Quickstart: TUI Redux/Elm Architecture

This guide demonstrates the simplified Redux/Elm architecture in Brummer's TUI.

## Prerequisites

- Go 1.21+ installed
- Brummer repository cloned
- Terminal with TTY support

## Quick Validation Steps

### 1. Run the Refactored TUI

```bash
# Build with the new architecture
make build

# Run and observe the simplified architecture
./brum

# You should see:
# - Instant keyboard response (<5ms)
# - Smooth view transitions
# - No state synchronization issues
```

### 2. Test Keyboard Navigation

Press these keys to validate the simplified handler system:

```
1 - Switch to Processes view
2 - Switch to Logs view
3 - Switch to Errors view
4 - Switch to URLs view
5 - Switch to Web view
6 - Switch to AI Coders view
7 - Switch to Settings view
Tab - Cycle through views
q - Quit application
```

Each keypress should:
- Respond immediately (<5ms)
- Show correct view
- Not cause any freezes

### 3. Verify State Management

```bash
# Start a process
Press '1' to go to Processes view
Press 'Enter' to start selected script

# Switch views and return
Press '2' for Logs
Press '1' for Processes

# State should be preserved:
- Process still running
- Selection maintained
- No duplicate processes
```

### 4. Test Redux/Elm Flow

```bash
# Run with debug logging
BRUMMER_DEBUG=redux ./brum

# You'll see the flow:
[redux] KeyMsg received: '2'
[redux] Update: Handling ViewSwitchedMsg
[redux] Model: currentView changed to 'logs'
[redux] View: Rendering logs view
```

### 5. Validate Service Statelessness

```go
// Test that services don't hold state
go test ./internal/tui/services -run TestStateless

// Output:
PASS: ProcessService is stateless
PASS: LogService is stateless
PASS: ConfigService is stateless
```

### 6. Performance Benchmarks

```bash
# Run performance tests
go test -bench=. ./internal/tui/...

# Expected results:
BenchmarkUpdate-8         1000000      950 ns/op    <1ms per update
BenchmarkKeyHandler-8     5000000      210 ns/op    <5ms response
BenchmarkViewRender-8      500000     2100 ns/op    60+ fps
```

### 7. Race Condition Testing

```bash
# Verify no race conditions
make test-race

# Should show:
Testing with race detector...
PASS - No races detected
```

### 8. Code Reduction Validation

```bash
# Compare old vs new line counts
echo "Old architecture:"
find ./internal/tui.backup -name "*.go" | xargs wc -l | tail -1

echo "New architecture:"
find ./internal/tui -name "*.go" | xargs wc -l | tail -1

# Expected: ~50% reduction
Old: 15,234 lines
New:  7,125 lines (53% reduction)
```

## Architecture Validation

### Test Update Function Purity

```go
// internal/tui/update_test.go
func TestUpdatePurity(t *testing.T) {
    model := NewModel()
    msg := ViewSwitchedMsg{NewView: ViewLogs}

    // Same input should produce same output
    model1, _ := model.Update(msg)
    model2, _ := model.Update(msg)

    assert.Equal(t, model1, model2, "Update is not pure")
}
```

### Test Service Statelessness

```go
// internal/tui/services/process_service_test.go
func TestServiceStateless(t *testing.T) {
    service := &ProcessService{workDir: "/tmp"}

    // Service methods return commands, not state changes
    cmd := service.Start("test", "echo hello")
    assert.NotNil(t, cmd, "Service must return command")

    // Verify service has no state fields
    // (This is a compile-time check)
}
```

### Test Message Flow

```go
// internal/tui/integration_test.go
func TestReduxFlow(t *testing.T) {
    model := NewModel()

    // User presses '2'
    keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}

    // Goes through Update
    newModel, cmd := model.Update(keyMsg)

    // View changes
    assert.Equal(t, ViewLogs, newModel.(Model).currentView)
}
```

## Migration Validation

### Before (Complex Flow)
```
KeyPress → InputService → MessageRouter → NavigationService →
NavigationComponent → Model → View
(6 layers, 1000+ lines)
```

### After (Simple Flow)
```
KeyPress → Update → Model → View
(2 layers, 400 lines)
```

## Success Criteria

- [ ] All 7 views accessible via number keys
- [ ] <5ms keyboard response time
- [ ] Zero race conditions in tests
- [ ] 50%+ code reduction achieved
- [ ] All existing functionality preserved
- [ ] Tests pass with race detector
- [ ] Performance benchmarks meet targets

## Troubleshooting

### Issue: Keyboard not responding
```bash
# Check handler registration
BRUMMER_DEBUG=keyboard ./brum
# Look for: "Registered handler: GlobalNavigation"
```

### Issue: State not updating
```bash
# Enable Update function logging
BRUMMER_DEBUG=update ./brum
# Verify messages are being processed
```

### Issue: Performance regression
```bash
# Run profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
# Look for bottlenecks in Update function
```

## Next Steps

1. Review the [Architecture Documentation](../../docs/architecture/redux-elm-architecture.md)
2. Study the [Migration Guide](./research.md#migration-approach)
3. Run the contract tests: `go test ./specs/008-create-a-specification/contracts/...`
4. Begin incremental refactoring using the tasks.md guide

---
*This quickstart validates the Redux/Elm architecture implementation*