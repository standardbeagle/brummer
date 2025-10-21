# AI Coder Terminal Viewport Fix

## Problem: Duplicate Headers and Content

When using Claude Code (which uses Ink for rendering), the terminal was showing duplicate headers and responses.

### Root Cause

Ink (React terminal renderer) does aggressive screen rewriting using ANSI escape codes:
- `\e[2K` - Erase entire line
- `\e[1A` - Move cursor up 1 line
- Repeatedly clears 5 lines upward and redraws

The vt10x terminal emulator was accumulating old content in its **scrollback buffer**. When we called `terminal.String()`, it returned:
- The scrollback history (old, cleared content)
- The current viewport (new content)

This caused duplicates - old content that Ink "erased" was still visible in scrollback.

### Solution

Changed from using `terminal.String()` to building screen content directly from `terminal.Cell(x, y)` based on viewport dimensions (80×24).

**Before:**
```go
// Get all content including scrollback
visibleContent := term.String()
lines := strings.Split(visibleContent, "\n")  // Includes old + new content
```

**After:**
```go
// Get viewport dimensions
cols, rows := view.Size()  // e.g., 80x24

// Build screen ONLY from viewport cells
for y := 0; y < rows; y++ {
    for x := 0; x < cols; x++ {
        cell := view.Cell(x, y)  // Only current viewport, no scrollback
        char := getRuneField(cell, "Char")
        // ... build line from cells
    }
}
```

### Benefits

1. ✅ **No duplicates** - Only shows current viewport, not scrollback history
2. ✅ **Proper Ink rendering** - Respects screen clears and rewrites
3. ✅ **Full color support** - Extracts FG/BG colors from cells (24-bit)
4. ✅ **Better performance** - Fixed 24-line viewport instead of growing scrollback

### Files Changed

- `internal/tui/services/aicoder_service.go`
  - Modified `getTerminalScreen()` to use Cell(x,y) iteration
  - Preserves color extraction with ANSI codes
  - Removes scrollback accumulation issue

### Testing

Test with Claude Code to verify:
- [x] No duplicate headers
- [x] No duplicate responses
- [x] Interactive terminal works
- [x] Full color depth (24-bit)
- [x] Proper handling of Ink's screen rewrites

### Technical Notes

**vt10x Terminal Emulator:**
- Has internal scrollback buffer that grows as content scrolls
- `String()` method returns scrollback + viewport
- `Cell(x, y)` method returns ONLY viewport cells
- Viewport size set at creation: `vt10x.New(vt10x.WithSize(80, 24))`

**Ink Rendering Pattern:**
```
\e[?2026h          # Enable synchronized updates
\e[2K\e[1A         # Clear line, move up (repeated 5x)
\e[G\r\n           # Go to column 0, newline
<draw content>     # Render new content
\e[?2026l          # Disable synchronized updates
```

This pattern clears old content upward and redraws. The viewport-only approach ensures we only see the final rendered state, not the cleared history.

## Related Issues

- **Bubble Tea v2 Migration**: Blocked due to v2 beta.3 dependency bug (ansi package API mismatch)
- **bubbleterm**: Requires v2 beta.3, cannot use until v2 is stable
- **Future**: Migrate to v2 when stable, use bubbleterm for better terminal emulation

## Recommendation

This viewport-only fix solves the immediate problem within Bubble Tea v1 constraints. When Bubble Tea v2 is stable, consider migrating to use bubbleterm for even better terminal emulation support.
