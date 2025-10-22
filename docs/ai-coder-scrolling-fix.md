# AI Coder Terminal Scrolling Fix

## Problem
When running Claude Code in the AI Coder terminal, the output was not scrolling properly and control codes were overlapping. The terminal showed duplicate lines and misaligned text instead of a clean, scrolling display.

## Root Cause
The vt10x terminal emulator was configured with 1000 lines of scrollback buffer for history, but our display code was trying to render all 1000 lines instead of just the visible viewport. This caused:
- Overlapping text as multiple screens worth of content tried to display in 24 lines
- No proper scrolling as new content appeared
- Misaligned cursor positioning

## Solution
Modified the `getTerminalScreen()` function in `/home/beagle/work/brummer/internal/tui/services/aicoder_service.go` to:

1. **Use vt10x's String() method** to get only the visible viewport content
2. **Map colors correctly** to the visible lines only
3. **Respect terminal boundaries** by processing only what's actually displayed

### Key Changes:
- First extract visible content using `terminal.String()` which returns the current viewport
- Parse visible lines and process only those for color extraction
- Map Cell(x, y) calls to the visible line indices, not the full buffer
- Maintain color state only for displayed content

## Technical Details

### Before:
```go
// Was trying to read all 1000 lines
for y := 0; y < rows; y++ {  // rows = 1000!
    cell := view.Cell(x, y)
    // Process ALL lines in buffer
}
```

### After:
```go
// Get only visible content
visibleContent := term.String()
visibleLines := strings.Split(visibleContent, "\n")

// Process only visible lines
for lineIdx, visibleLine := range visibleLines {
    cell := view.Cell(x, lineIdx)  // Map to visible line index
    // Process only displayed content
}
```

## Benefits
- Clean scrolling behavior - new content pushes old content up
- Proper cursor positioning - cursor stays at the input position
- No overlapping text - each line displays only once
- Maintains full color support while fixing display issues

## Testing
To test the fix:
1. Run `./brum`
2. Press '6' for AI Coders tab
3. Run `/ai claude` to start Claude Code
4. Type commands and observe:
   - Text scrolls up properly as new content appears
   - No overlapping or duplicate lines
   - Colors are preserved correctly
   - Cursor stays at the correct position

The terminal now behaves like a standard terminal emulator with proper scrolling and positioning.