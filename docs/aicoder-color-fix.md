# AI Coder Color Rendering Fix Plan

## Current Issue
The AI Coder tab is not showing colors because:
1. vt10x processes ANSI codes and provides clean text in `Terminal.String()`
2. The screen buffer is plain text without color information
3. We're not extracting color attributes from vt10x cells

## vt10x Terminal Structure
The vt10x terminal maintains a grid of cells, each with:
- Character (rune)
- Foreground color
- Background color
- Attributes (bold, italic, underline, etc)

## Solutions

### Option 1: Use Raw PTY Output (Quick Fix)
Instead of using vt10x processed output, display the raw PTY bytes directly:
- Store raw PTY output with ANSI codes intact
- Render directly to screen without processing
- Let the terminal emulator handle ANSI codes

### Option 2: Extract vt10x Cell Attributes (Proper Solution)
Access vt10x's internal cell grid to get color information:
- Iterate through terminal cells
- Extract foreground/background colors
- Convert to lipgloss styles
- Render with proper formatting

### Option 3: Use a Different Terminal Emulator
Replace vt10x with a terminal emulator that provides formatted output:
- Consider github.com/ActiveState/termtest/xterm
- Or github.com/aymanbagabas/go-pty + custom ANSI renderer

## Immediate Fix
For now, we can:
1. Store raw PTY output in the PTYSession
2. Send raw output through the debouncer
3. Display raw ANSI codes in the view
4. Let the user's terminal handle color rendering

## Long-term Solution
Implement proper cell-based rendering:
1. Access vt10x.Terminal's internal state
2. Get cell attributes for each position
3. Convert to appropriate styles
4. Render with full color support