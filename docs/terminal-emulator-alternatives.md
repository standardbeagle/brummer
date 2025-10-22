# Terminal Emulator Alternatives for AI Coder

## Current Issue with vt10x
- Doesn't handle screen clearing properly
- Accumulates content instead of replacing
- Limited ANSI escape sequence support
- Causes duplicate headers and responses

## Recommended Solution: Raw ANSI Passthrough

Instead of using a terminal emulator that parses and re-renders, we can:
1. **Store raw PTY output** with all ANSI codes intact
2. **Pass it directly to the display**
3. **Let the user's terminal handle rendering**

### Benefits:
- 100% compatibility with all ANSI codes
- No parsing/rendering overhead
- Supports all features Claude Code uses
- Simplest and most reliable approach

### Implementation:
```go
// Instead of using vt10x.Terminal, just store raw bytes
type PTYBuffer struct {
    lines [][]byte
    currentLine []byte
}

// Process output by detecting newlines but keeping ANSI codes
func (b *PTYBuffer) Write(data []byte) {
    // Split on \n but keep all ANSI sequences intact
    // Store last N lines for display
}
```

## Alternative: Azure go-ansiterm

If we need parsing for features like scrollback:

```bash
go get github.com/Azure/go-ansiterm
```

```go
import "github.com/Azure/go-ansiterm"

// Create ANSI parser
parser := ansiterm.CreateParser("Ground", eventHandler)
parser.Parse(ptyOutput)
```

### Benefits:
- Complete VT100/ANSI implementation
- Used by Docker - very reliable
- Handles all escape sequences correctly
- Proper screen buffer management

## Quick Fix for Now

Change vt10x buffer size from 1000 to 24 lines:
```go
terminal := vt10x.New(vt10x.WithSize(80, 24))
```

This prevents accumulation but loses scrollback.