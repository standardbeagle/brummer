---
sidebar_position: 2
---

# Process Management

Learn how to effectively manage multiple processes in Brummer.

## Process States

Brummer tracks processes with visual indicators:

- 🟢 **Running** - Process is actively running
- 🔴 **Stopped** - Process was manually stopped
- ❌ **Failed** - Process exited with an error
- ✅ **Success** - Process completed successfully
- ⏸️ **Pending** - Process is starting up

## Starting Processes

### From Scripts View
1. Navigate to Scripts view (Tab or press `1`)
2. Use ↑/↓ to select a script
3. Press **Enter** to start

### Quick Start
```
1 → Enter on script name
```

## Managing Running Processes

### View All Processes
Press `2` or Tab to Processes view to see:
- Process name and status
- Start time
- Duration
- Exit code (if stopped)

### Stop a Process
1. Navigate to Processes view
2. Select the running process (🟢)
3. Press **s** to stop

### Restart a Process
1. Select any process
2. Press **r** to restart
3. Process will stop (if running) then start again

### Restart All Processes
Press **Ctrl+R** to restart all running processes simultaneously.

## Process Monitoring

### Real-time Status
The header shows process count:
```
🐝 Brummer - Package Script Manager (3 processes, 2 running)
```

### Process Details
Each process shows:
```
🟢 npm run dev                    [Running for 2m 34s]
❌ npm run test                   [Failed - Exit code: 1]
✅ npm run build                  [Success - Took 45s]
```

## Viewing Process Logs

### Individual Process Logs
1. In Processes view, select a process
2. Press **Enter** to filter logs for that process
3. Press **Esc** to return to all logs

### Combined Logs
Switch to Logs view (`3`) to see all process output combined with timestamps and process indicators.

## Advanced Features

### Process Priority
Brummer automatically manages process priority:
- Interactive processes get terminal access
- Background processes run without TTY
- Error output is prioritized in logs

### Process Groups
Related processes are managed together:
- Parent/child relationships maintained
- Graceful shutdown of process trees
- Zombie process prevention

### Resource Management
- Automatic cleanup on exit
- Process limit warnings
- Memory usage monitoring (future feature)

## Best Practices

### Development Workflow
1. Start your dev server first
2. Run watch processes (tests, linting)
3. Keep build processes on-demand

### Process Organization
- Use descriptive script names
- Group related scripts with prefixes
- Add comments in package.json

### Error Handling
- Check Errors view (`4`) for failures
- Use process restart for transient errors
- Stop and debug for persistent issues

## Keyboard Shortcuts

| Key | Action | Context |
|-----|--------|---------|
| **s** | Stop process | On running process |
| **r** | Restart process | Any process |
| **Ctrl+R** | Restart all | Anywhere |
| **Enter** | View logs | On any process |
| **q** | Stop all & quit | Anywhere |

## Common Scenarios

### Running Multiple Services
```bash
# Start all services
1. Run database: npm run db
2. Run backend: npm run server  
3. Run frontend: npm run dev
```

### Development Setup
```bash
# Typical dev workflow
1. npm run dev (frontend)
2. npm run api (backend)
3. npm run test:watch (tests)
```

### Build Pipeline
```bash
# Sequential builds
1. npm run clean
2. npm run build
3. npm run test
```

## Troubleshooting

### Process Won't Stop
- Use **s** key (not Ctrl+C)
- Check for child processes
- Force quit with **q** if needed

### Process Keeps Failing
1. Check error logs (`4`)
2. Verify dependencies
3. Check port conflicts
4. Review environment variables

### High CPU/Memory Usage
- Monitor process duration
- Check for infinite loops
- Use system monitor alongside

## Tips

1. **Color Coding**: Learn the status colors for quick scanning
2. **Batch Operations**: Use Ctrl+R to restart everything
3. **Log Filtering**: Use Enter on process for focused debugging
4. **Quick Stop**: Press `2` then `s` from any view