# PTY Color Rendering in Brummer: A Deep Dive

## Executive Summary

This document details the journey of implementing full ANSI color support in Brummer's PTY (pseudo-terminal) view, including the challenges faced, discoveries made, and the final solution that enables rich terminal applications like Claude Code to display with full color fidelity in a BubbleTea TUI.

## The Problem

When running terminal applications (like Claude Code) inside Brummer's AI Coder PTY view, colors were not displaying correctly. While emojis showed color, regular text appeared monochrome despite the terminal application sending ANSI color escape sequences.

## Key Discoveries

### 1. The vt10x Terminal Emulator

Brummer uses the `github.com/hinshun/vt10x` library as a terminal emulator. This library:
- Parses raw PTY output containing ANSI escape sequences
- Maintains a cell-based buffer (like a real terminal)
- Each cell contains: character, foreground color, background color, and text attributes
- Handles cursor positioning, screen clearing, and other terminal operations

### 2. Color Encoding in vt10x

The most critical discovery was how vt10x encodes colors in its `Color` type (uint32):

```
Color Value Range    | Meaning
---------------------|------------------------------------------
0-7                  | Standard ANSI colors (black, red, green, etc.)
8-15                 | Bright ANSI colors
16-255               | 256-color palette
256-16777215         | 24-bit true color (RGB packed)
16777216 (0x1000000) | DefaultFG (special value)
16777217 (0x1000001) | DefaultBG (special value)
```

For 24-bit true color, vt10x packs RGB values into a single uint32:
```
Color = (R << 16) | (G << 8) | B
```

Example: RGB(255,153,51) becomes 16750899 (0xFF9933)

### 3. The Lipgloss Problem

The initial implementation used lipgloss (a terminal styling library) to render borders. However, **lipgloss.Style.Render() strips ANSI escape codes from content**. This was the primary reason colors weren't displaying - our carefully reconstructed ANSI codes were being stripped when rendering the border!

### 4. Modern Terminal Applications Use True Color

Claude Code (and many modern terminal apps) use 24-bit true color ANSI sequences:
```
\033[38;2;255;153;51m  # Foreground RGB(255,153,51)
\033[48;2;0;0;0m       # Background RGB(0,0,0)
```

Our initial implementation only handled 8-bit and 256-color modes, missing true color support entirely.

## The Solution

### Step 1: Proper Color Reconstruction

We implemented comprehensive ANSI code reconstruction from vt10x cells:

```go
// For each cell in the terminal buffer
cell := terminal.Cell(x, y)

// Handle all color modes
if cell.FG < 8 {
    // Standard ANSI: \033[3{n}m
    codes = append(codes, fmt.Sprintf("3%d", cell.FG))
} else if cell.FG < 16 {
    // Bright ANSI: \033[9{n}m
    codes = append(codes, fmt.Sprintf("9%d", cell.FG-8))
} else if cell.FG < 256 {
    // 256-color: \033[38;5;{n}m
    codes = append(codes, fmt.Sprintf("38;5;%d", cell.FG))
} else if cell.FG < vt10x.DefaultFG {
    // 24-bit true color: \033[38;2;{r};{g};{b}m
    r := (cell.FG >> 16) & 0xFF
    g := (cell.FG >> 8) & 0xFF
    b := cell.FG & 0xFF
    codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
}
```

### Step 2: Raw ANSI Border Rendering

Instead of using lipgloss for borders, we use raw ANSI codes:

```go
// Don't do this - it strips ANSI codes!
// borderStyle.Render(content)

// Do this instead - preserve ANSI codes
borderColorCode := "\033[38;5;62m"  // Raw ANSI
result.WriteString(borderColorCode)
result.WriteString("╭")              // Border character
result.WriteString(resetCode)
result.WriteString(content)          // Content with preserved ANSI
```

### Step 3: Optimization

To minimize ANSI code generation, we track the current style and only emit codes when it changes:

```go
currentFG := vt10x.DefaultFG
currentBG := vt10x.DefaultBG

for each cell {
    if cell.FG != currentFG || cell.BG != currentBG {
        // Style changed, emit new ANSI codes
        if styleActive {
            output.WriteString("\033[0m")  // Reset previous
        }
        // ... generate new codes ...
        currentFG = cell.FG
        currentBG = cell.BG
    }
    output.WriteRune(cell.Char)
}
```

## Architecture

```
PTY Output (with ANSI codes)
    ↓
vt10x Terminal Emulator
    ├── Parses ANSI sequences
    ├── Updates cell buffer
    └── Tracks colors/attributes
    ↓
renderTerminalContent()
    ├── Reads cell buffer
    ├── Reconstructs ANSI codes
    └── Optimizes output
    ↓
renderTerminalWithBorder()
    ├── Uses raw ANSI for border
    └── Preserves content ANSI
    ↓
BubbleTea View
    └── Displays colored output
```

## Testing Insights

Our testing revealed:

1. **vt10x correctly parses all color modes** - The issue was never with vt10x parsing
2. **Lipgloss is incompatible with ANSI content** - Any use of Style.Render() strips codes
3. **True color is essential** - Modern apps expect 24-bit color support
4. **Cell-based rendering works** - Reading from vt10x's buffer is the right approach

## Performance Considerations

1. **Minimize ANSI code generation** - Only emit codes when style changes
2. **Use string.Builder** - More efficient than string concatenation
3. **Lock terminal buffer** - Ensure consistent reads with minimal lock time
4. **Limit visible area** - Only render cells that fit in the view

## Future Improvements

1. **Scrollback buffer** - Currently limited; could be enhanced
2. **Mouse support** - vt10x supports mouse events we don't yet handle
3. **Alternate screen buffer** - Full support for terminal apps that use it
4. **Performance profiling** - Optimize for very large terminals

## Lessons Learned

1. **Understand your dependencies** - Deep knowledge of vt10x's internals was crucial
2. **Test with real applications** - Claude Code's true color usage revealed gaps
3. **Beware of "helpful" libraries** - Lipgloss's ANSI stripping was unexpected
4. **Raw ANSI is sometimes necessary** - Not everything needs abstraction
5. **Terminal emulation is complex** - But cell-based approaches simplify rendering

## Code References

- Color reconstruction: `internal/tui/ai_coder_pty_view.go:renderTerminalContent()`
- Border rendering: `internal/tui/ai_coder_pty_view.go:renderTerminalWithBorder()`
- PTY integration: `internal/aicoder/pty_session.go`

## Conclusion

Full color support in PTY rendering requires:
1. Understanding how your terminal emulator stores colors
2. Supporting all color modes (8-bit, 256-color, 24-bit)
3. Avoiding libraries that strip ANSI codes
4. Optimizing ANSI code generation
5. Testing with real-world applications

The final implementation provides rich, colorful terminal output that matches what users expect from modern terminal applications.