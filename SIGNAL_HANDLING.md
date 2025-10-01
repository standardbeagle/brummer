# Signal Handling

Brummer implements robust signal handling with double Ctrl+C force exit.

## Behavior

### First Ctrl+C
- **Action**: Graceful shutdown
- **Message**: "⚠️  Interrupt received, shutting down gracefully... (Press Ctrl+C again to force quit)"
- **What happens**:
  - Stops all running processes
  - Stops MCP server
  - Stops proxy server
  - Unregisters instance
  - Clean exit

### Second Ctrl+C (within 2 seconds)
- **Action**: Force exit immediately
- **Message**: "🚨 Force quit! Exiting immediately..."
- **Exit code**: 130 (standard for Ctrl+C)
- **What happens**:
  - Exits immediately via `os.Exit(130)`
  - No cleanup
  - Used when graceful shutdown hangs or takes too long

## Timeout Reset

If no second Ctrl+C is received within 2 seconds, the counter resets. The next Ctrl+C will be treated as a first interrupt again.

## Implementation

Signal handling is implemented at the highest level in:
- `cmd/brum/signals_unix.go` (Linux, macOS, BSD)
- `cmd/brum/signals_windows.go` (Windows)

The implementation uses:
- **atomic operations** for thread-safe interrupt counting
- **goroutine monitoring** raw signal channel
- **time.AfterFunc** for 2-second reset
- **os.Exit(130)** for immediate force quit

## Why This Matters

### Problem Without Double Ctrl+C
If the application hangs during shutdown (e.g., service not responding, deadlock, network timeout), the user is stuck with no way to exit except `kill -9` from another terminal.

### Solution: Force Quit Safety Hatch
Users can always exit with double Ctrl+C, even if:
- ServiceBridge is blocked
- Components are hung
- Services won't stop
- Network operations timeout
- Any other blocking situation

## Platform Support

### Unix (Linux, macOS, BSD)
- Handles `SIGINT` (Ctrl+C)
- Handles `SIGTERM` (kill command)
- Both trigger graceful shutdown on first signal
- Double interrupt forces exit

### Windows
- Handles `os.Interrupt` (Ctrl+C)
- Same double-interrupt behavior
- Note: SIGTERM not supported on Windows

## Testing

```bash
# Test graceful shutdown
./brum
# Press Ctrl+C once
# Observe: "Interrupt received, shutting down gracefully..."
# Wait for clean exit

# Test force quit
./brum
# Press Ctrl+C twice quickly
# Observe: "Force quit! Exiting immediately..."
# Immediate exit (exit code 130)

# Test timeout reset
./brum
# Press Ctrl+C once
# Wait 3 seconds
# Press Ctrl+C again
# Observe: Treated as first interrupt (graceful shutdown)
```

## Exit Codes

- **0**: Clean exit (normal operation completed)
- **1**: Error during startup or operation
- **130**: Interrupted by Ctrl+C (standard Unix convention)

## Design Philosophy

**Safety First**: Users must always be able to exit the application.

**Graceful by Default**: First interrupt gives services time to clean up.

**Force as Fallback**: Second interrupt is the emergency exit when graceful fails.

**Highest Level**: Signal handling happens before any components can interfere.

**Cross-Platform**: Consistent behavior on all supported platforms.
