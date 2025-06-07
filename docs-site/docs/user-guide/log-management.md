---
sidebar_position: 3
---

# Log Management

Master Brummer's intelligent log management system for efficient debugging.

## Log Views

### Combined Logs View
Access with Tab or press `3`. Shows:
- All process output combined
- Timestamps for each line
- Process name indicators
- Color-coded output

### Errors View
Access with Tab or press `4`. Shows:
- Filtered error messages only
- Stack traces
- Failed assertions
- Build errors

## Log Features

### Automatic Error Detection
Brummer recognizes error patterns:
- `Error:`, `ERROR`, `[ERROR]`
- `Failed`, `FAILED`
- `Warning:`, `WARN`
- Stack traces
- Non-zero exit codes

### Smart Highlighting
- **Red** - Errors and failures
- **Yellow** - Warnings
- **Green** - Success messages
- **Blue** - Info and URLs
- **Gray** - Debug output

### Real-time Streaming
- Live updates as processes run
- Automatic scrolling (unless manually scrolled)
- Buffer management for performance

## Searching Logs

### Basic Search
1. Press `/` to enter search mode
2. Type your search term
3. Press Enter to search
4. Use `n`/`N` to navigate results

### Search Tips
- Case-insensitive by default
- Supports regex patterns
- Highlights all matches
- Shows match count

### Common Searches
```
/error           # Find all errors
/failed.*test    # Failed tests
/localhost:\d+   # Find port numbers
/warning|warn    # Warnings
```

## Filtering

### Priority Filter
Press `p` to toggle high-priority logs:
- Shows only errors and warnings
- Hides verbose debug output
- Useful for quick error scanning

### Process Filter
From Processes view:
1. Select a process
2. Press Enter to see only its logs
3. Press Esc to return to all logs

### Custom Filters (Settings)
Create persistent filters:
- Error patterns
- Warning patterns
- Ignore patterns
- Priority boosts

## Log Actions

### Copy Last Error
Press `c` to copy the most recent error to clipboard:
- Includes stack trace
- Adds context lines
- Ready for issue reports

### Clear Logs
Press `Ctrl+L` to clear current view:
- Doesn't affect log files
- Helps with performance
- Fresh start for debugging

### Export Logs
Future feature - export logs to file

## Advanced Features

### Log Persistence
- Logs are stored in memory
- Rotation at 10,000 lines
- Process-specific buffers

### URL Detection
Automatically detects and highlights:
- `http://` and `https://` URLs
- `localhost:port` patterns
- File paths
- IP addresses

### Build Event Detection
Recognizes:
- Webpack build events
- Test runner output
- Compilation errors
- Linting results

## Best Practices

### Efficient Debugging
1. Start with Errors view (`4`)
2. Use search for specific issues
3. Filter by process if needed
4. Copy errors for sharing

### Performance Tips
- Clear logs periodically
- Use priority filter for large outputs
- Limit concurrent processes

### Log Organization
- Use clear log prefixes in your code
- Consistent error formatting
- Structured logging when possible

## Keyboard Shortcuts

| Key | Action | Context |
|-----|--------|---------|
| `/` | Search | Logs/Errors view |
| `p` | Toggle priority | Logs view |
| `c` | Copy last error | Any view |
| `Ctrl+L` | Clear logs | Logs view |
| `n`/`N` | Next/prev match | Search active |

## Common Patterns

### Debugging Errors
```
1. Press 4 (Errors view)
2. Look for red entries
3. Press c to copy
4. Check stack trace
```

### Finding URLs
```
1. Press 5 (URLs view)
2. Or search: /localhost
3. Click/copy URLs
```

### Monitoring Tests
```
1. Run test script
2. Watch for ✓ and ✗
3. Filter errors only
```

## Troubleshooting

### Missing Logs
- Check process is running
- Verify script outputs to stdout/stderr
- Some tools need --verbose flag

### Garbled Output
- Check terminal encoding
- Disable color output in scripts
- Use --no-color flags

### Performance Issues
- Clear logs with Ctrl+L
- Reduce concurrent processes
- Check system resources

## Tips

1. **Color Meanings**: Red=Error, Yellow=Warning, Green=Success
2. **Quick Error Check**: Press `4` from anywhere
3. **Search History**: Up/down arrows in search mode
4. **Context Lines**: Errors include surrounding context