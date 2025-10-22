# Color Rendering Diagnosis

## Issue

User reports that the AI Coder terminal PTY view displays all grey text, even though:
- Terminal supports true color (`COLORTERM=truecolor`)
- RGB color codes are present in PTY debug log
- Session header renders in blue (lipgloss styles work)

## Investigation Steps

### 1. Verified Color Pipeline Exists

✅ **Complete color extraction and rendering pipeline implemented:**

```
vt10x Cell (FG/BG colors)
    ↓
getTerminalScreen() extracts via reflection
    ↓
buildANSIColorCode() generates ANSI sequences
    ↓
ScreenBuffer [][]rune with embedded ANSI codes
    ↓
Component renders: string(runeSlice)
    ↓
Bubble Tea View() output
```

**Files:**
- `internal/tui/services/aicoder_service.go:589-621` - Color extraction
- `internal/tui/services/aicoder_service.go:692-750` - ANSI code generation
- `internal/tui/components/aicoders_view.go:117-122` - Component rendering

### 2. Confirmed ANSI Codes in PTY Output

✅ **PTY debug log shows RGB codes:**
```
\e[38;2;215;119;87m  # Claude logo orange
\e[38;2;136;136;136m # Separator gray
\e[48;2;0;0;0m       # Background black
```

**File:** `pty-debug-20251021-153102.log`

### 3. Verified String Conversion Preserves ANSI

✅ **Test confirmed:**
```go
ansiCode := []rune("\x1b[38;2;215;119;87mClaude Orange\x1b[0m")
str := string(ansiCode)
// Output: Colors render correctly outside Bubble Tea
```

### 4. Researched Bubble Tea ANSI Handling

**Key findings:**
- Bubble Tea **manages** all ANSI codes via renderer
- Bubble Tea **downsamples** colors based on detected terminal profile
- Bubble Tea expects View() to return:
  - Plain text (no ANSI), OR
  - Lipgloss-styled strings
- Raw ANSI codes in View() may be **processed/stripped** by renderer

**References:**
- Bubble Tea auto-downsamples ANSI for output device
- Lipgloss preserves ANSI codes in tables, but designed to **control** styling
- termenv provides color profile detection (Ascii, ANSI, ANSI256, TrueColor)

## Current Hypothesis

**Bubble Tea's renderer is processing/stripping raw ANSI codes** because it expects to control all terminal styling.

The issue is architectural:
- We're embedding **raw ANSI codes** in screen buffer
- Bubble Tea expects **lipgloss styles** or **plain text**
- Renderer may be filtering/downsampling our embedded codes

## Diagnostic Build

Added debug output to test hypothesis:

### Debug Features Added

1. **Color Profile Detection** (`aicoders_view.go:114-126`):
   ```go
   profile := termenv.DefaultOutput().Profile
   // Shows: ASCII, ANSI (16), ANSI256, or TrueColor (24-bit)
   ```

2. **Raw ANSI Test Line** (`aicoders_view.go:131-133`):
   ```go
   testANSI := "\x1b[38;2;215;119;87mTest: Claude Orange (raw ANSI)\x1b[0m"
   ```
   - If this renders in orange → Bubble Tea supports raw ANSI
   - If this renders in grey → Bubble Tea is stripping/processing ANSI

3. **ANSI Code Detection** (`aicoders_view.go:138-141`):
   ```go
   if strings.Contains(line, "\x1b[") {
       line = line + " [ANSI]"
   }
   ```
   - Shows `[ANSI]` marker on first 3 lines if they contain ANSI codes
   - Confirms whether screen buffer actually has ANSI codes

## Testing Instructions

### Run the diagnostic build:

```bash
make build
./brum
# Navigate to AI Coders tab (3)
# Start session: /ai claude
```

### Check the output:

1. **Header line** should show:
   ```
   Session: AI: claude [Profile: TrueColor (24-bit RGB)]
   ```

2. **Test line** should appear below header:
   ```
   Test: Claude Orange (raw ANSI)
   ```
   - **If orange** → Bubble Tea renders raw ANSI ✅
   - **If grey** → Bubble Tea strips raw ANSI ❌

3. **First 3 lines** of PTY content:
   - Look for `[ANSI]` markers
   - **If present** → Screen buffer contains ANSI codes ✅
   - **If missing** → ANSI codes not being generated ❌

## Expected Results

| Observation | Meaning | Next Step |
|-------------|---------|-----------|
| Test line is **orange** | Bubble Tea supports raw ANSI | Check if screen buffer has codes |
| Test line is **grey** | Bubble Tea strips raw ANSI | Need to use lipgloss instead |
| `[ANSI]` markers present | Codes are in buffer | Problem is in Bubble Tea rendering |
| `[ANSI]` markers missing | Codes not generated | Problem is in color extraction |
| Profile is **TrueColor** | Terminal supports RGB | Not a terminal issue |
| Profile is **ANSI/ANSI256** | Terminal limited colors | May need downsampling |

## Potential Solutions

### If Bubble Tea strips raw ANSI:

1. **Parse ANSI codes and convert to lipgloss styles**
   - Pro: Works with Bubble Tea's architecture
   - Con: Complex parsing, performance overhead

2. **Use termenv to create styled strings**
   - Pro: termenv works with Bubble Tea
   - Con: Need to parse vt10x colors and convert

3. **Render PTY to separate window/pane**
   - Pro: Direct terminal control
   - Con: Major architectural change

### If screen buffer missing codes:

1. **Debug getTerminalScreen() extraction**
   - Check reflection-based color extraction
   - Verify buildANSIColorCode() being called

2. **Verify vt10x color storage**
   - Check if vt10x actually storing RGB colors
   - May need raw mode instead of vt10x

## Files Modified

- `internal/tui/components/aicoders_view.go`:
  - Added termenv import
  - Added color profile detection (lines 114-126)
  - Added raw ANSI test line (lines 131-133)
  - Added ANSI marker detection (lines 138-141)

## Next Steps

1. **User tests diagnostic build**
2. **Reports observations** (test line color, profile, ANSI markers)
3. **Choose solution** based on diagnosis results
