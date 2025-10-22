# AI Coder Terminal Fix Summary

## Problem Solved
- **Duplicate headers** in Claude Code output
- **Duplicate responses** when running commands
- **Poor ANSI control code handling** with vt10x
- **Screen clearing not working** properly

## Root Cause
The vt10x terminal emulator was configured with 1000 lines of scrollback buffer. When Claude Code sent screen clear commands (`\e[2J`), vt10x wasn't properly clearing its buffer, causing old content to accumulate and duplicate.

## Solution Implemented

### 1. Raw PTY Buffer (`/home/beagle/work/brummer/internal/aicoder/raw_pty_buffer.go`)
Created a simple buffer that stores raw PTY output with ANSI codes intact, without any parsing or interpretation.

### 2. Direct ANSI Passthrough
Modified PTY session to use raw mode by default:
- `UseRawMode: true` - Bypasses vt10x terminal emulation
- `RawBuffer` - Stores raw ANSI output
- Direct passthrough to Bubble Tea

### 3. Debug Logging
Added comprehensive PTY debug logging (`DEBUG_PTY_OUTPUT=true`) to diagnose control code issues.

## How It Works Now

```
PTY Output → Raw Buffer → Bubble Tea → Terminal
    ↑           ↑            ↑           ↑
 Raw bytes   No parsing   ANSI codes  Perfect
             preserved    passthrough  rendering
```

## Benefits
- ✅ **No duplicate content** - Screen clears work properly
- ✅ **Full ANSI support** - All escape sequences work
- ✅ **24-bit color** - True color support
- ✅ **Better performance** - No parsing overhead
- ✅ **Simpler code** - Less complexity

## Alternative Considered: bubbleterm
We discovered `github.com/taigrr/bubbleterm`, a terminal emulator designed for Bubble Tea. However, it requires Bubble Tea v2 (beta), while our project uses v1. The raw PTY approach works well and is simpler.

## Testing
Run with debug logging to verify:
```bash
DEBUG_PTY_OUTPUT=true ./brum
```

1. Press '6' for AI Coders
2. Run `/ai claude`
3. Observe:
   - No duplicate headers
   - Clean screen updates
   - Proper scrolling
   - Full color support

## Files Modified
- `/home/beagle/work/brummer/internal/aicoder/pty_session.go` - Added raw mode support
- `/home/beagle/work/brummer/internal/aicoder/raw_pty_buffer.go` - New raw buffer implementation
- `/home/beagle/work/brummer/internal/tui/services/aicoder_service.go` - Updated to use raw content

## Debug Log Analysis
The debug log (`pty-debug-*.log`) revealed Claude Code sends complex ANSI sequences:
- Multiple screen clears and cursor repositioning
- Hide/show cursor operations
- Color and formatting codes

These are now properly handled by passing them directly to the terminal.