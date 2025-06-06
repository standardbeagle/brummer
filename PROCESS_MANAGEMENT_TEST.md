# Testing Process Management Features

## Quick Test Guide

### 1. Start Brummer
```bash
cd test-project
../brummer
```

### 2. Start Some Processes

**In the Scripts tab:**
- Navigate to `long-running` script and press **Enter**
- Navigate to `docs` script and press **Enter** 
- Navigate to `dev` script and press **Enter**

You should now have multiple processes running.

### 3. Test Process Management

**Switch to Processes tab** (press **Tab** until you reach it)

You should see:
```
Processes (3 total, 3 running)
Select process: ↑/↓ | Stop: s | Restart: r | Restart All: Ctrl+R | View Logs: Enter

🟢 [running] long-running
PID: abc123 | Started: 14:30:15 | Press 's' to stop, 'r' to restart

🟢 [running] docs  
PID: def456 | Started: 14:31:20 | Press 's' to stop, 'r' to restart

🟢 [running] dev
PID: ghi789 | Started: 14:32:10 | Press 's' to stop, 'r' to restart
```

### 4. Test Individual Process Control

**Select a process** (use ↑/↓ arrows):
- Press **s** to stop the selected process
- You should see a log message: "Stopping process: [name]"
- The process status should change to 🔴 [stopped]

**Restart a process**:
- Select any process and press **r**
- You should see: "Restarting process: [name]"
- The process should stop then start again

### 5. Test Restart All

- Make sure you have multiple running processes
- Press **Ctrl+R** 
- You should see: "Restarting all running processes..."
- All running processes should restart

### 6. Test Log Viewing

- Select any process and press **Enter**
- Should switch to Logs tab showing logs for that process

### 7. Test Graceful Exit

- Press **q** or **Ctrl+C**
- Should see: "Stopping X running processes..."
- Should see the bee goodbye screen
- All processes should be terminated

## Troubleshooting

### If keyboard shortcuts don't work:

1. **Check you're in the Processes tab** - shortcuts only work there
2. **Check log messages** - go to Logs tab to see error messages
3. **Check a process is selected** - use ↑/↓ to select first

### If processes don't stop:

1. **Check system logs** - look for "Failed to stop process" messages
2. **Check process IDs** - make sure processes actually exist
3. **Force kill manually** if needed: `ps aux | grep node` then `kill -9 <PID>`

### Expected Log Messages:

When working correctly, you should see these in the Logs tab:
- "Stopping process: [name]" when pressing 's'
- "Restarting process: [name]" when pressing 'r'  
- "Restarting all running processes..." when pressing Ctrl+R
- "Failed to stop process [name]: [error]" if there are issues

## Debugging Process Killing

The app now uses aggressive process termination:
1. **SIGINT** (Ctrl+C equivalent) - graceful shutdown
2. **SIGKILL** (force kill) - if graceful fails

Processes should terminate within a few seconds. If they don't, check:
- Process is actually running: `ps aux | grep [process-name]`
- Process has permission to be killed
- System resources are available

## Test Scenarios

### Scenario 1: Basic Stop/Start
1. Start `long-running` script
2. Go to Processes tab
3. Press 's' to stop it
4. Verify it shows 🔴 [stopped]
5. Press 'r' to restart it  
6. Verify it shows 🟢 [running] again

### Scenario 2: Multiple Process Management
1. Start 3 different scripts
2. Stop one with 's'
3. Restart all with Ctrl+R
4. Verify all are running again

### Scenario 3: Exit Cleanup
1. Start 2-3 scripts
2. Press 'q' to quit
3. Verify cleanup message appears
4. Check no zombie processes remain: `ps aux | grep node`