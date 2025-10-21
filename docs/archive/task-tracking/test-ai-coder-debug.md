# AI Coder Debug Test Instructions

## What We're Debugging

The AI Coder view is showing correct dimensions but the VT10x terminal output isn't displaying. We've added debug logging to understand:

1. Whether the PTY session is receiving output
2. Whether the VT10x terminal is parsing the output
3. Whether the terminal cells contain any characters
4. The terminal buffer dimensions

## Test Steps

1. Run brummer:
   ```bash
   ./brum
   ```

2. Switch to AI Coders view (press `5` or Tab 5 times)

3. Start a test session:
   ```
   /ai test-claude
   ```

4. Look for debug output in the AI Coders view:
   - `[DEBUG] Terminal: true/false, Output bytes: N`
   - `[DEBUG] Terminal size: WxH, View size: WxH`
   - `[DEBUG] Non-empty cells found: N`

5. Switch to Logs view (press `2`) and look for:
   - "Started monitoring PTY output for session..."
   - "Received N bytes of PTY output"
   - "PTY Terminal: true, Active: true"

## What the Debug Info Means

- **Terminal: false** - VT10x terminal isn't initialized (BAD)
- **Output bytes: 0** - No output received from PTY (BAD)
- **Non-empty cells: 0** - Terminal buffer is empty (BAD)
- **Terminal size: 80x24** - Default terminal size (OK)
- **View size: should match window** - Calculated display size

## Expected Good Output

You should see:
1. Terminal: true
2. Output bytes: > 0
3. Non-empty cells: > 0
4. The actual terminal content below the debug info

## If Terminal Is Empty

This could mean:
1. PTY session isn't starting properly
2. Output isn't being fed to VT10x terminal
3. Terminal isn't being read correctly

Please report what debug output you see!