# System Message Panel Test Guide

## Overview
The system message panel displays internal Brummer messages at the bottom of the screen, separate from application errors which appear in the Errors tab (View 3).

## What Shows in System Panel
- Process control messages (start/stop/restart)
- Settings and configuration messages
- MCP installation messages
- File operation messages
- Proxy startup messages ("Starting reverse proxy...")
- Any internal Brummer errors or warnings

## Test Instructions

1. **Start Brummer with a script**
   ```bash
   cd test-project
   ../brum dev
   ```
   
   You'll see reverse proxy startup messages at the bottom of the logs view.
   These can be dismissed by pressing 'm'.

2. **Generate System Messages**
   - Press `1` to go to Processes view
   - Press `s` without selecting a process → Shows "No process selected to stop" ❌
   - Press `r` without selecting a process → Shows "No process selected to restart" ❌
   - Start a process, then stop it → Shows "Stopping process: [name]" ℹ️
   - Go to Settings (6) and try operations → Shows success ✅ or error ❌ messages

3. **View System Panel**
   - Only shows when there are system messages (empty by default)
   - Shows last 5 system messages at the bottom when present
   - Press `e` to expand to full screen (only works if there are messages)
   - Press `e` again to collapse back to 5 lines
   - Press `m` to clear all system messages

## Message Format
```
[timestamp] [icon] [context]: [message]
```

Examples:
```
[14:32:15] ❌ Process Control: No process selected to stop
[14:32:20] ℹ️ Process Control: Stopping process: dev
[14:32:25] ✅ Settings: PAC URL copied to clipboard
[14:32:30] ⚠️ Settings: Package manager preference already set
```

## Icons
- ❌ Error - Something went wrong
- ⚠️ Warning - Important notice
- ✅ Success - Operation completed
- ℹ️ Info - General information

## Key Features
- **Auto-hide**: Panel is completely hidden when there are no messages
- **Quick dismiss**: Press `m` to clear all system messages instantly
- **Compact view**: Shows only last 5 messages by default
- **Full view**: Press `e` to see all messages with scroll support
- **Headers always visible**: Tab navigation remains accessible except when command palette is open

## Key Differences from Error Tab
- **System Panel**: Internal Brummer messages (process control, settings, etc.)
- **Errors Tab (3)**: Application errors from your running processes (build errors, runtime errors, etc.)