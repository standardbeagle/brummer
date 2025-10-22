# PTY Debug Logging

## Overview
A debug logging feature has been added to capture raw PTY input/output for troubleshooting terminal control code issues, particularly with complex terminal applications like Claude Code.

## Usage

### Enabling Debug Logging
Set the environment variable `DEBUG_PTY_OUTPUT=true` when running Brummer:

```bash
DEBUG_PTY_OUTPUT=true ./brum
```

Or use the provided test script:
```bash
./test-pty-debug.sh
```

### Output Format
Debug logs are created as `pty-debug-YYYYMMDD-HHMMSS.log` in the current directory.

Each log entry contains:
1. **Timestamp**: When the data was captured
2. **Source**:
   - `PTY_OUT` - Data received from the PTY (application output)
   - `PTY_IN` - Data sent to the PTY (keyboard input)
   - `PTY_IN_DIRECT` - Direct input bypassing the input channel
3. **Hex dump**: Raw bytes in hexadecimal
4. **ASCII representation**: Human-readable format with control codes shown

### Example Log Entry
```
[15:04:05.123] [PTY_OUT] 1B 5B 32 4A 1B 5B 48
[15:04:05.123] [PTY_OUT] \e[2J\e[H

[15:04:05.456] [PTY_IN] 68 65 6C 6C 6F 0D
[15:04:05.456] [PTY_IN] hello\r
```

## Common ANSI Control Codes

### Screen Control
- `\e[2J` - Clear entire screen
- `\e[1J` - Clear from cursor to beginning of screen
- `\e[0J` - Clear from cursor to end of screen

### Cursor Movement
- `\e[H` or `\e[1;1H` - Move cursor to home (1,1)
- `\e[n;mH` - Move cursor to row n, column m
- `\e[nA` - Move cursor up n lines
- `\e[nB` - Move cursor down n lines
- `\e[nC` - Move cursor right n columns
- `\e[nD` - Move cursor left n columns

### Line Control
- `\e[K` or `\e[0K` - Clear from cursor to end of line
- `\e[1K` - Clear from cursor to beginning of line
- `\e[2K` - Clear entire line

### Save/Restore
- `\e[s` - Save cursor position
- `\e[u` - Restore cursor position
- `\e7` - Save cursor and attributes
- `\e8` - Restore cursor and attributes

## Troubleshooting Duplicate Headers

If you see duplicate headers (like the Claude Code banner appearing twice), look for:

1. **Missing screen clear**: Application may be writing new content without clearing old
2. **Partial updates**: Application updating only parts of the screen
3. **Scrollback issues**: Terminal emulator keeping old content when it should clear

### Analysis Commands
```bash
# View all clear screen commands
grep "1B 5B 32 4A" pty-debug-*.log  # \e[2J

# View cursor positioning
grep "1B 5B.*48" pty-debug-*.log     # \e[...H

# See full sequence of a session startup
head -100 pty-debug-*.log
```

## Implementation Details

The debug logging is implemented in `/home/beagle/work/brummer/internal/aicoder/pty_session.go`:

- `initDebugLogging()` - Creates the debug log file
- `logPTYData()` - Writes data to the log with formatting
- Logging points:
  - `readLoop()` - Captures all PTY output
  - `writeLoop()` - Captures all PTY input
  - `SendInputDirect()` - Captures direct input

The logging is designed to have minimal performance impact and automatically flushes data to disk to ensure nothing is lost if the application crashes.

## Privacy Note
Debug logs may contain sensitive information including:
- Commands typed
- Passwords (if entered)
- File contents displayed
- API responses

Always review and sanitize logs before sharing them.