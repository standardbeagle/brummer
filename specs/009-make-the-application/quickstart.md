# Quickstart: Bubble Tea MVU Pattern Implementation

## Overview
This guide demonstrates the Bubble Tea MVU pattern implementation in Brummer, showing how all I/O operations go through commands for a responsive, non-blocking UI.

## Prerequisites
- Go 1.21+ installed
- Brummer repository cloned
- Terminal with TTY support

## Quick Verification Steps

### 1. Build and Run
```bash
# Build the application
make build

# Run with performance monitoring
./brum --debug-perf

# Alternatively, run tests
make test-tui
```

### 2. Verify Non-Blocking UI (< 50ms response)

#### Test 1: Navigation During Heavy I/O
1. Start the application
2. Press `Tab` to cycle through views rapidly
3. Observe: UI responds immediately (<50ms) even with background operations
4. Check debug output: `[PERF] UI Response: 12ms`

Expected: All tab switches complete in <50ms

#### Test 2: Rapid Input Handling
1. Press `/` to open command input
2. Type rapidly: "test command input"
3. Observe: Characters appear instantly without lag
4. Press `Esc` to cancel

Expected: No character drops or delays

#### Test 3: Concurrent Operations
1. Start multiple processes (press `1`, `2`, `3` for scripts)
2. While processes are starting, navigate tabs
3. Open logs view (press `l`)
4. Scroll through logs while they're updating

Expected: Smooth scrolling and navigation during process operations

### 3. Verify Command Pattern

#### Test 1: I/O Command Execution
```bash
# Run with command tracing
./brum --trace-commands

# In another terminal, monitor the trace
tail -f /tmp/brum-commands.log
```

Expected output:
```
[CMD] IOCommand{ID: io-1234, Type: FileRead, Context: {Version: 42}}
[MSG] IOResultMsg{ID: io-1234, Success: true, Time: 8ms}
[STATE] Version 42 -> 43, ResponseTime: 8ms
```

#### Test 2: Retry Logic
1. Disconnect network (for testing network operations)
2. Trigger a network operation in the UI
3. Observe retry attempts in logs
4. Reconnect network
5. Observe successful completion

Expected: 3 retry attempts with exponential backoff, then success

### 4. Performance Validation

#### Run Performance Tests
```bash
# Run response time tests
go test -run TestResponseTime ./internal/tui/... -v

# Run with race detection
go test -race -run TestConcurrentCommands ./internal/tui/...

# Benchmark command execution
go test -bench=BenchmarkCommand ./internal/tui/commands/...
```

Expected results:
- TestResponseTime: All operations <50ms
- Race detector: No races detected
- Benchmark: Command overhead <1ms

### 5. Manual Acceptance Testing

#### Scenario 1: Process Management
1. Start application: `./brum`
2. Start a long-running process: Press `1` for npm dev
3. While process is starting:
   - Switch to logs view: Press `l`
   - Switch to URLs view: Press `u`
   - Open command palette: Press `/`
4. Verify all actions respond immediately

✅ Pass Criteria: All UI actions complete in <50ms

#### Scenario 2: Error Handling
1. Start application with a broken package.json
2. Attempt to start a non-existent script
3. Observe error display
4. Continue using other features
5. Verify UI remains responsive

✅ Pass Criteria: Errors displayed clearly, UI never freezes

#### Scenario 3: Heavy Load
1. Start 5+ processes simultaneously
2. Open log view to see all outputs
3. Rapidly switch between tabs
4. Use keyboard shortcuts quickly
5. Monitor performance indicator

✅ Pass Criteria: Response time stays <50ms under load

## Troubleshooting

### UI Feels Sluggish
1. Check performance mode: `./brum --debug-perf`
2. Look for operations >50ms in logs
3. Verify no goroutines in TUI code: `grep -r "go func" internal/tui/`

### Commands Not Executing
1. Enable command trace: `./brum --trace-commands`
2. Check command queue isn't full
3. Verify Bubble Tea runtime is active

### Race Conditions
1. Run with race detector: `go run -race ./cmd/brum/main.go`
2. Check for concurrent state modifications
3. Ensure all state changes go through Update()

## Success Metrics

| Metric | Target | Validation |
|--------|--------|------------|
| UI Response Time | <50ms | Performance tests pass |
| Command Overhead | <1ms | Benchmark results |
| Concurrent Operations | Framework-managed | No goroutines in TUI |
| Error Recovery | Smart retry | Retry tests pass |
| Memory Usage | <100MB | Resource monitor |
| Race Conditions | Zero | Race detector clean |

## Next Steps

1. **For Developers**:
   - Review command implementations in `internal/tui/commands/`
   - Study Update() function in `internal/tui/model.go`
   - Add new commands following the pattern

2. **For Testers**:
   - Run full test suite: `make test-all`
   - Perform load testing with multiple instances
   - Validate on different terminal emulators

3. **For Users**:
   - Enjoy the responsive UI!
   - Report any UI freezes as bugs
   - Provide feedback on response times

## Architecture Validation

To confirm the MVU pattern is correctly implemented:

```bash
# Check for goroutines (should find none in TUI code)
grep -r "go func" internal/tui/ --include="*.go" | grep -v test | grep -v disabled

# Verify command pattern usage
grep -r "tea.Cmd" internal/tui/ --include="*.go" | wc -l
# Expected: >50 occurrences

# Confirm Update() pattern
grep -r "func.*Update.*tea.Msg.*tea.Model.*tea.Cmd" internal/tui/model.go
# Expected: Exactly 1 match

# Validate no blocking operations
grep -r "time.Sleep\|.Wait()\|<-" internal/tui/ --include="*.go" | grep -v test
# Expected: No matches in production code
```

## Conclusion

The Bubble Tea MVU pattern implementation ensures:
- ✅ UI never freezes (<50ms response guaranteed)
- ✅ All I/O through commands (no goroutines)
- ✅ Proper error handling with smart retry
- ✅ Clean separation of concerns (Model-View-Update)
- ✅ Race-condition free concurrent operations

The implementation meets all requirements and provides a responsive, reliable terminal UI experience.