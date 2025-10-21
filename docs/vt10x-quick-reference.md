# VT10X Quick Reference

## TL;DR

**vt10x is 69% compatible with modern PTY terminals**, but **90% compatible with Ink (Claude Code)** for what matters.

### ✅ What Works Perfectly
- **24-bit RGB colors** (full true color support)
- **All cursor movement** (up, down, left, right, positioning)
- **Screen/line clearing** (critical for Ink's rewriting)
- **Scrolling** (regions, up/down)
- **Basic text attributes** (bold, italic, underline, reverse, blink)

### ⚠️ What's Missing (But Doesn't Break Ink)
- **Synchronized output** (`\e[?2026h/l`) - Ink uses this but vt10x safely ignores it
- **Dim mode** (`\e[2m`) - Treated as bold, minor visual difference
- **Alternate screen buffer** (`\e[?1049h/l`) - Ink doesn't use this

### ❌ What's Not Supported (And Ink Doesn't Use)
- Hyperlinks, graphics protocols, advanced mouse modes, notifications

---

## Feature Coverage by Category

| Category | Score | Status |
|----------|-------|--------|
| Cursor Movement | 12/12 (100%) | ✅ Perfect |
| Colors (RGB) | 10/10 (100%) | ✅✅✅ Perfect |
| Screen Clearing | 6/7 (86%) | ✅ Excellent |
| Scrolling | 3/3 (100%) | ✅ Perfect |
| Line/Char Ops | 5/5 (100%) | ✅ Perfect |
| Text Attributes | 13/18 (72%) | ⚠️ Good |
| Terminal Modes | 4/14 (29%) | ❌ Poor |
| Modern Features | 0/8 (0%) | ❌ None |
| **TOTAL** | **53/77 (69%)** | **⚠️ Good** |

---

## Ink (Claude Code) Specific Analysis

### Ink's Core Rendering Pattern

```bash
\e[?2026h          # Synchronized output START (vt10x ignores ⚠️)
\e[2K\e[1A         # Clear line, move up (vt10x supports ✅)
\e[2K\e[1A         # Repeat for 5 lines
\e[2K\e[1A
\e[2K\e[1A
\e[2K\e[1A
\e[G\r\n           # Column 0, newline (vt10x supports ✅)
\e[38;2;215;119;87m  # 24-bit RGB color (vt10x supports ✅)
Claude Code...     # Colored text
\e[?2026l          # Synchronized output END (vt10x ignores ⚠️)
```

### Compatibility Score: 90% ✅

**What vt10x handles correctly for Ink:**
1. ✅ `\e[2K` - Erase entire line (critical!)
2. ✅ `\e[1A` - Cursor up 1 line (critical!)
3. ✅ `\e[G` - Cursor to column 0
4. ✅ `\e[38;2;r;g;b` - 24-bit RGB foreground
5. ✅ `\e[48;2;r;g;b` - 24-bit RGB background
6. ✅ `\e[1m` - Bold
7. ✅ `\e[22m` - Normal intensity
8. ✅ `\e[?25l/h` - Hide/show cursor
9. ✅ `\e[?2004h/l` - Bracketed paste mode

**What vt10x ignores (but doesn't break):**
1. ⚠️ `\e[?2026h/l` - Synchronized output (logged, ignored)
2. ⚠️ `\e[2m` - Dim mode (treated as bold)

**Verdict:** vt10x works great for Ink with the viewport fix!

---

## The Viewport Fix Solution

### The Problem
```
vt10x scrollback buffer:
  [Line 1 - old content (cleared by Ink)]
  [Line 2 - old content (cleared by Ink)]
  [Line 3 - old content (cleared by Ink)]
  --- viewport boundary (24 lines) ---
  [Line 4 - current Claude Code header]  ← Viewport starts here
  [Line 5 - current content]
  ...
  [Line 27 - current content]             ← Viewport ends here
```

When we called `terminal.String()`:
- Returns: scrollback + viewport = **duplicates**

### The Solution
```go
// OLD: Get everything (scrollback + viewport)
content := terminal.String()

// NEW: Get only viewport via Cell iteration
for y := 0; y < 24; y++ {
    for x := 0; x < 80; x++ {
        cell := terminal.Cell(x, y)  // Only viewport!
    }
}
```

**Result:** Only see current viewport (24 lines), no scrollback accumulation. ✅

---

## Supported ANSI Sequences (Quick Ref)

### Cursor Control
```bash
\e[<n>A      # Up n lines
\e[<n>B      # Down n lines
\e[<n>C      # Forward n columns
\e[<n>D      # Back n columns
\e[<r>;<c>H  # Position to row,col
\e[<n>E      # Down n lines, column 0
\e[<n>F      # Up n lines, column 0
\e[<c>G      # Column absolute
\e[s         # Save cursor
\e[u         # Restore cursor
```

### Screen/Line Clearing
```bash
\e[J or \e[0J   # Clear cursor to end of screen
\e[1J           # Clear cursor to beginning of screen
\e[2J           # Clear entire screen
\e[K or \e[0K   # Clear cursor to end of line
\e[1K           # Clear cursor to beginning of line
\e[2K           # Clear entire line ← Ink uses this heavily!
```

### Colors
```bash
# 16 basic colors
\e[30-37m       # FG 8 colors
\e[40-47m       # BG 8 colors
\e[90-97m       # FG 8 bright colors
\e[100-107m     # BG 8 bright colors

# 256 colors
\e[38;5;<n>m    # FG color 0-255
\e[48;5;<n>m    # BG color 0-255

# 24-bit RGB (TRUE COLOR) ✅
\e[38;2;<r>;<g>;<b>m   # FG RGB
\e[48;2;<r>;<g>;<b>m   # BG RGB

# Reset
\e[39m          # Reset FG to default
\e[49m          # Reset BG to default
\e[0m           # Reset all attributes
```

### Text Attributes
```bash
\e[0m   # Reset all
\e[1m   # Bold
\e[3m   # Italic
\e[4m   # Underline
\e[5m   # Blink
\e[7m   # Reverse (swap FG/BG)
\e[22m  # Normal intensity (unbold)
\e[23m  # Not italic
\e[24m  # Not underlined
\e[25m  # Not blinking
\e[27m  # Not reversed
```

### Scrolling
```bash
\e[<n>S            # Scroll up n lines
\e[<n>T            # Scroll down n lines
\e[<top>;<bot>r    # Set scroll region
```

### Modes
```bash
\e[?25h    # Show cursor
\e[?25l    # Hide cursor
\e[?2004h  # Enable bracketed paste
\e[?2004l  # Disable bracketed paste
```

---

## What's NOT Supported (But You Might Want)

### Critical Modern Features
```bash
\e[?2026h/l    # Synchronized output (Ink uses, vt10x ignores)
\e[?1049h/l    # Alternate screen buffer (many TUIs use)
\e[?47h/l      # Alt screen (older variant)
\e[3J          # Clear scrollback
```

### Text Attributes
```bash
\e[2m          # Dim/faint (treated as bold)
\e[8m          # Conceal
\e[9m          # Strikethrough
\e[21m         # Double underline
```

### Advanced Features
```bash
\e]8;;<url>\e\\         # Hyperlinks
\e]777;notify;...\e\\   # Notifications
\e[4:3m                 # Undercurl
\e[58;2;<r>;<g>;<b>m    # Underline color
```

---

## Debugging Tips

### Check What Ink is Sending
```bash
# Enable PTY debug logging
DEBUG_PTY_OUTPUT=true ./brum

# Check the log file
cat pty-debug-*.log
```

### Look for These Patterns
```bash
\e[2K\e[1A     # Clear and move up (Ink's signature)
\e[?2026h      # Synchronized output (ignored by vt10x)
\e[38;2;...    # RGB colors (vt10x supports!)
```

### Viewport vs Scrollback
```go
// BAD: Includes scrollback
content := terminal.String()

// GOOD: Only viewport
for y := 0; y < rows; y++ {
    for x := 0; x < cols; x++ {
        cell := terminal.Cell(x, y)
    }
}
```

---

## Summary

**vt10x is a solid VT100/xterm emulator** that:
- ✅ Handles 90% of what Ink (Claude Code) needs
- ✅ Has perfect 24-bit RGB color support
- ✅ Supports all critical cursor and clearing operations
- ⚠️ Ignores synchronized output (harmlessly)
- ⚠️ Missing some modern features (alternate screen)
- ✅ Works great with the viewport-only fix

**For Claude Code integration:** vt10x + viewport fix = **Excellent compatibility** ✅
