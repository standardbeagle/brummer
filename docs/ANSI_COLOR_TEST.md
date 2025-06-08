# Testing ANSI Color Support in Brummer

## Problem
The logs viewport in Brummer was not displaying ANSI color codes from process output. Instead, the raw escape sequences were being shown.

## Root Cause
The issue was in the `updateLogsView` method in `internal/tui/model.go`. When applying lipgloss styles to log lines, the entire line (including the log content) was being passed through `style.Render()`. This caused lipgloss to strip any existing ANSI codes from the content.

## Solution
Modified the code to only apply lipgloss styling to the timestamp and process name prefix, while keeping the actual log content raw. This preserves any ANSI color codes in the process output.

### Changed Code
```go
// Before: Applied style to entire line including content
line := fmt.Sprintf("[%s] %s: %s", 
    log.Timestamp.Format("15:04:05"),
    log.ProcessName,
    cleanContent,
)
content.WriteString(style.Render(line))

// After: Apply style only to prefix, keep content raw
prefix := fmt.Sprintf("[%s] %s: ", 
    log.Timestamp.Format("15:04:05"),
    log.ProcessName,
)
content.WriteString(style.Render(prefix))
content.WriteString(cleanContent)
```

## Testing
1. Build the updated brummer: `go build -o brum cmd/brummer/main.go`
2. Run brummer in the test-project directory: `cd test-project && ../brum`
3. Select and run the "color-test" script
4. Switch to the Logs view to see colored output

The color-test script outputs various ANSI color codes including:
- Text colors (red, green, yellow, blue, magenta, cyan, white)
- Bold text
- Background colors
- Underlined text
- Combined styles

You should now see properly colored output in the logs viewport instead of raw escape sequences.