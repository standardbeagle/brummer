# Emoji and Unicode Width Handling in Brummer Components

## Overview

The Brummer TUI component system properly handles emoji and Unicode character widths to ensure correct terminal display. This is critical because emojis and other Unicode characters can occupy different numbers of terminal cells than their byte or rune count suggests.

## The Problem

Common mistakes when calculating string widths:

1. **Using `len()`** - Counts bytes, not display width
   - "Hello" = 5 bytes, 5 display cells ✅
   - "🏃" = 4 bytes, 2 display cells ❌

2. **Using `len([]rune())`** - Counts Unicode code points
   - "Hello" = 5 runes, 5 display cells ✅
   - "🏃" = 1 rune, 2 display cells ❌
   - "⚙️" = 2 runes (gear + variation selector), 2 display cells ❌

3. **Not stripping ANSI codes** - Color codes affect calculations
   - "\x1b[31mRed\x1b[0m" = 11 bytes, but only 3 display cells

## The Solution

### Libraries Used

- **`github.com/mattn/go-runewidth`** - Calculates actual terminal display width
- **`github.com/charmbracelet/lipgloss`** - Handles styled text width correctly
- **`github.com/rivo/uniseg`** - Grapheme cluster handling (for cursor movement)

### Width Calculation Rules

| Character Type | Bytes | Runes | Display Width |
|---------------|-------|-------|---------------|
| ASCII         | 1     | 1     | 1 cell        |
| Emoji (🏃)    | 3-4   | 1     | 2 cells       |
| CJK (漢字)    | 3     | 1     | 2 cells       |
| Box Drawing (│)| 3    | 1     | 1 cell        |
| Arrow (▶)     | 3     | 1     | 1 cell        |

### Safe Width Helpers

The component system provides `width_helpers.go` with safe functions:

```go
// ALWAYS use these functions for width calculations
SafeStringWidth(s)      // Display width after stripping ANSI
SafeLipglossWidth(s)    // For styled text
SafeTruncate(s, max)    // Truncate considering display width
SafeCenter(s, width)    // Center with proper width calculation
CalculateTabWidth(...)  // Calculate tab component width
```

## Component Implementation

### Best Practices

1. **Always initialize colors** in constructors:
```go
func NewComponent() *Component {
    MustInitColors() // Force color output
    // ...
}
```

2. **Use proper width calculation**:
```go
// ❌ WRONG
if len(text) > maxWidth {
    text = text[:maxWidth]
}

// ✅ CORRECT
if SafeStringWidth(text) > maxWidth {
    text = SafeTruncateWithEllipsis(text, maxWidth)
}
```

3. **Account for borders**:
```go
// Lipgloss borders add to width, not included
contentWidth := CalculateAvailableWidth(terminalWidth, true) // true = has border
```

4. **Validate component fits**:
```go
err := ValidateComponentWidth(componentWidth, availableWidth)
if err != nil {
    // Switch to more compact rendering mode
}
```

### Responsive Tab Rendering

The tab system adapts to terminal width:

| Width   | Mode    | Display Format                      |
|---------|---------|-------------------------------------|
| <50     | minimal | `1• 2 3• 4 5`                     |
| <80     | compact | `🏃 1 📄 2 🔵 ❌ 3 🔴`           |
| <120    | normal  | `🏃 1.Proc 📄 2.Log 🔵 5`        |
| 120+    | full    | `🏃 1.Processes 📄 2.Logs 🔵 5`  |

### Tab Width Calculation Example

For a full tab "🏃 1.Processes 🔴 3":
- 🏃 = 2 cells (emoji)
- Space = 1 cell
- "1.Processes" = 11 cells
- Space = 1 cell
- 🔴 = 2 cells (emoji)
- Space = 1 cell
- "3" = 1 cell
- **Total: 19 cells**

## Common Issues and Solutions

### Issue 1: Emoji Overflow
**Problem**: Tabs extend past terminal edge
**Solution**: Use `TabWidthCalculator` to determine appropriate mode

### Issue 2: Center Alignment Off
**Problem**: Content not properly centered with emojis
**Solution**: Use `SafeCenter()` which calculates padding with `runewidth`

### Issue 3: Truncation Breaks Emojis
**Problem**: Truncating in middle of multi-byte emoji
**Solution**: Use `SafeTruncate()` which respects rune boundaries

### Issue 4: ANSI Codes in Width
**Problem**: Color codes counted in width calculation
**Solution**: Use `SafeLipglossWidth()` or strip ANSI first

## Testing Width Handling

Run width validation tests:
```bash
go test ./internal/tui/components -run TestWidth
```

Visual verification:
```bash
# Test at different terminal widths
COLUMNS=50 ./brum   # Minimal mode
COLUMNS=80 ./brum   # Compact mode
COLUMNS=120 ./brum  # Normal mode
COLUMNS=150 ./brum  # Full mode
```

## Component Checklist

When creating a new component:

- [ ] Call `MustInitColors()` in constructor
- [ ] Use `SafeStringWidth()` for all width calculations
- [ ] Never use `len()` for display width
- [ ] Account for emoji width (typically 2 cells)
- [ ] Test with narrow terminals (<80 columns)
- [ ] Validate component fits available space
- [ ] Provide responsive/adaptive rendering modes
- [ ] Strip ANSI codes before width calculation
- [ ] Use `CalculateAvailableWidth()` for bordered components
- [ ] Test with actual emojis, not just ASCII

## References

- [go-runewidth documentation](https://github.com/mattn/go-runewidth)
- [Unicode East Asian Width](http://www.unicode.org/reports/tr11/)
- [ANSI Escape Codes](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [Grapheme Clusters](http://www.unicode.org/reports/tr29/)