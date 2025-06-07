# Browser Extension Troubleshooting Guide

## Enhanced Logging Features

The Brummer browser extension now includes enhanced logging and connection monitoring to make troubleshooting easier.

## DevTools Panel Features

### Connection Status
The DevTools panel shows detailed connection information:
- **Green "Connected"** - Successfully connected to Brummer server
- **Red "Disconnected"** - Not connected
- **Hover over status** - Shows server URL and Client ID

### Active Browser Tabs Section
The panel now displays all browser tabs with Brummer logging enabled:
- Shows tab title, process name, and tab ID
- Green dot (🟢) indicates the currently active tab
- Gray dot (⚪) indicates inactive tabs
- "Focus" button to quickly switch to a specific tab
- Real-time updates every 2 seconds

## Content Script Features

### 1. URL Parameter Recognition Logging

When a browser tab is opened with Brummer parameters, you'll see styled console logs:

```
🐝 Brummer Extension Activated (yellow background)
URL Parameters Recognized: (green text)
  Token: bt_process_name_1234567890
  Endpoint: http://localhost:7777/api/browser-log
  Process: npm run dev
```

### 2. Connection Status Indicator

A connection status indicator appears in the bottom-right corner of browser tabs with Brummer logging enabled:

- 🟢 **Connected (Xms)** - Successfully connected to Brummer TUI
- 🟡 **Connecting to Brummer...** - Attempting to establish connection
- 🔴 **Connection lost** - No response for >10 seconds
- ⚠️ **Error: message** - Connection error occurred

The indicator shows:
- Connection state with colored icon
- Latency in milliseconds when connected
- Error messages when connection fails

### 3. Ping/Pong Monitoring

The extension sends ping requests every 5 seconds to monitor connection health:

- Success logs: `✓ Brummer ping: 12ms` (green text)
- Failure logs: `✗ Brummer ping failed: Error message` (red text)

### 4. Interactive Status Indicator

- **Click** the status indicator to toggle between full opacity and semi-transparent (30%)
- **Hover** over the indicator for a subtle scale effect
- The indicator automatically appears when Brummer parameters are detected

## Debugging Connection Issues

### Check Console Logs

1. Open browser DevTools (F12)
2. Go to Console tab
3. Look for Brummer-related logs with styled formatting

### Common Issues

1. **"Connection lost" status**
   - Verify Brummer TUI is running
   - Check if port 7777 is accessible
   - Ensure no firewall is blocking connections

2. **"Invalid token" errors**
   - Token may have expired (tokens are cleaned up after 60s of inactivity)
   - Restart the process from Brummer TUI

3. **High latency (>100ms)**
   - Network congestion
   - Brummer server under heavy load
   - Consider restarting Brummer

### Server-Side Monitoring

The Brummer TUI shows browser connection status:
- 🌐 icon appears in the header when browsers are connected
- Browser logs appear with `[browser]` or `[browser:tabId]` prefix
- Active connections are tracked and cleaned up automatically

## Testing the Features

Use the included test script to verify everything works:

```bash
# From Brummer TUI, run:
npm run browser-test
```

This will:
1. Start a test HTTP server on port 8888
2. Open a browser with Brummer parameters
3. Provide buttons to test various logging scenarios
4. Show connection status and ping latency