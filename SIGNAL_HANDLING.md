# Signal Handling

Brummer implements robust signal handling with double Ctrl+C force exit.

## Behavior

### First Ctrl+C (Accidental Press Protection)
- **Action**: No shutdown - just shows warning
- **Message**: "⚠️  Interrupt received. Press Ctrl+C again within 2 seconds to exit."
- **What happens**:
  - Nothing! The app keeps running
  - You can continue working
  - Protects against accidental Ctrl+C

### Second Ctrl+C (within 2 seconds)
- **Action**: Start graceful shutdown
- **Message**: "⏳ Shutting down gracefully... (Press Ctrl+C again to force quit)"
- **What happens**:
  - Stops all running processes
  - Stops MCP server
  - Stops proxy server
  - Unregisters instance
  - Clean exit

### Third Ctrl+C (during shutdown)
- **Action**: Force exit immediately
- **Message**: "🚨 Force quit! Exiting immediately..."
- **Exit code**: 130 (standard for Ctrl+C)
- **What happens**:
  - Exits immediately via `os.Exit(130)`
  - No cleanup
  - Used when graceful shutdown hangs or takes too long

## Timeout Reset

If no second Ctrl+C is received within 2 seconds, the counter resets and the app continues running normally. You can safely hit Ctrl+C by accident and then go back to typing.

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

### Accidental Press Protection
Users often hit Ctrl+C by accident while typing. With single Ctrl+C shutdown, this would interrupt their work. Now:
- First Ctrl+C is harmless - just shows a warning
- User can continue working if it was accidental
- Must press Ctrl+C twice to actually exit

### Force Quit Safety Hatch
If graceful shutdown hangs (service not responding, deadlock, network timeout), users can force quit with third Ctrl+C:
- ServiceBridge is blocked → force quit works
- Components are hung → force quit works
- Services won't stop → force quit works
- Network operations timeout → force quit works
- Any blocking situation → force quit works

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
# Test accidental press (no shutdown)
./brum
# Press Ctrl+C once
# Observe: "⚠️  Interrupt received. Press Ctrl+C again within 2 seconds to exit."
# App keeps running! You can continue working.

# Test graceful shutdown (double Ctrl+C)
./brum
# Press Ctrl+C twice quickly
# Observe:
#   First: "⚠️  Interrupt received. Press Ctrl+C again within 2 seconds to exit."
#   Second: "⏳ Shutting down gracefully... (Press Ctrl+C again to force quit)"
# Wait for clean exit

# Test force quit (triple Ctrl+C)
./brum
# Press Ctrl+C twice to start shutdown
# While shutting down, press Ctrl+C again
# Observe: "🚨 Force quit! Exiting immediately..."
# Immediate exit (exit code 130)

# Test timeout reset (accidental press recovery)
./brum
# Press Ctrl+C once
# Wait 3 seconds
# App still running! Counter reset. Continue working normally.
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
