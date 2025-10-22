# AI Coder Color Support Implementation

## Overview
Successfully implemented full 24-bit color support for the AI Coder tab by extracting cell attributes from the vt10x terminal emulator. This provides rich, colorful terminal output matching what users expect from modern terminal applications.

## Implementation Details

### Cell Attribute Extraction
The key innovation is directly accessing vt10x's cell grid to extract color information:

1. **Cell Access**: Using vt10x's `Cell(x, y)` method to get individual character cells
2. **Color Extraction**: Each cell contains:
   - `Char`: The character (rune)
   - `FG`: Foreground color (uint32)
   - `BG`: Background color (uint32)

3. **Reflection Usage**: Since we can't import vt10x directly in the service layer, we use Go's reflection to safely extract field values from the Glyph struct

### Color Code Generation
The implementation supports multiple color modes:

1. **Standard ANSI Colors (0-15)**:
   - Colors 0-7: Standard colors (30-37 for FG, 40-47 for BG)
   - Colors 8-15: Bright colors (90-97 for FG, 100-107 for BG)

2. **256-Color Mode (16-255)**:
   - Uses `\x1b[38;5;N]` for foreground
   - Uses `\x1b[48;5;N]` for background

3. **24-bit True Color (RGB)**:
   - For colors > 256, extracts RGB components
   - Uses `\x1b[38;2;R;G;B]` for foreground
   - Uses `\x1b[48;2;R;G;B]` for background

### Performance Optimizations

1. **Color Change Detection**: Only emits color codes when colors change between cells
2. **Line-by-Line Processing**: Resets colors at end of each line
3. **Trailing Space Trimming**: Removes unnecessary trailing spaces while preserving structure
4. **Fallback Mode**: Falls back to plain text if cell extraction fails

## Code Location

The implementation is in `/home/beagle/work/brummer/internal/tui/services/aicoder_service.go`:

- `getTerminalScreen()`: Main function that converts vt10x terminal to colored screen buffer
- `getRuneField()`: Extracts character from cell using reflection
- `getColorField()`: Extracts color value from cell using reflection
- `buildANSIColorCode()`: Generates appropriate ANSI escape sequences

## Testing

Created test script at `/home/beagle/work/brummer/test-ai-coder.sh` that:
1. Builds the application
2. Tests headless mode with MCP
3. Provides instructions for manual testing

## Usage

To see the color support in action:
1. Run `./brum` to start the TUI
2. Press '6' to switch to AI Coders tab
3. Run `/ai terminal` for a local shell or `/ai claude` for AI assistant
4. The terminal will display with full color support

## Benefits

1. **Rich Visual Experience**: Full 24-bit color support matching modern terminals
2. **Accurate Rendering**: Preserves all color information from applications
3. **Performance**: Efficient color change detection minimizes ANSI code overhead
4. **Compatibility**: Works with standard ANSI, 256-color, and true color applications

## Technical Notes

- Uses lock-free immutable state pattern for thread safety
- Integrates with 60 FPS debounced screen updates
- Maintains compatibility with existing PTY infrastructure
- Gracefully falls back to plain text if needed