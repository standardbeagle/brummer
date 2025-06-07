---
sidebar_position: 3
---

# Browser Extension Features

The Brummer browser extension enhances your debugging experience by providing real-time integration between your browser and the Brummer TUI.

## Core Features

### 🔗 Automatic URL Detection

The extension automatically detects and highlights URLs from your development servers:

- **Localhost URLs**: `http://localhost:3000`, `http://127.0.0.1:8080`
- **Network URLs**: `http://192.168.1.100:3000`
- **Custom domains**: Any configured development domains

When Brummer detects a URL in your logs, the extension:
1. Adds a visual indicator in the browser
2. Shows which process is serving the URL
3. Displays the current status (running, building, error)

### 📊 Real-time Status Monitoring

See the status of your development servers directly in the browser:

- **Green indicator**: Server is running normally
- **Yellow indicator**: Build in progress
- **Red indicator**: Server error or crash
- **Gray indicator**: Server stopped

### 🐛 Error Overlay

When errors occur in your development server:

1. **Non-intrusive notification** appears in the browser
2. **Click to expand** full error details
3. **Direct link** to the error location in your code
4. **Copy error** to clipboard for debugging

### 🔄 Auto-refresh on Build

The extension can automatically refresh your page when:
- Build completes successfully
- Hot Module Replacement (HMR) fails
- Server restarts after a crash

Configure auto-refresh behavior in the extension settings.

### 📝 Console Integration

Enhanced console features:

- **Log filtering**: Show only logs from specific processes
- **Error highlighting**: Errors are highlighted in red
- **Timestamp display**: See when each log was generated
- **Search functionality**: Search through historical logs

## DevTools Panel

Access the Brummer DevTools panel for advanced features:

1. Open Chrome/Firefox DevTools (F12)
2. Navigate to the "Brummer" tab
3. View detailed process information:
   - Process tree
   - Resource usage
   - Log history
   - Environment variables

### Process Management

Control processes directly from the browser:

- **Start/Stop**: Control individual processes
- **Restart**: Quick restart with one click
- **Clear logs**: Clear log buffer for a clean view

### Log Viewer

Advanced log viewing capabilities:

- **Syntax highlighting**: Color-coded log levels
- **Filtering**: Filter by log level, process, or content
- **Export**: Save logs for later analysis
- **Real-time updates**: See logs as they happen

## Network Monitoring

Track network requests from your development server:

- **Request timing**: See how long each request takes
- **Status codes**: Quickly identify failed requests
- **Headers**: Inspect request and response headers
- **Payload**: View request/response bodies

## Integration Features

### VSCode Integration

When used with VSCode:
- Click on file paths in logs to open in editor
- See which files triggered rebuilds
- Navigate to error locations directly

### Git Integration

- See which branch is currently active
- View uncommitted changes that might affect builds
- Quick access to git commands

## Performance Monitoring

Track your development server's performance:

- **Build times**: Historical build time trends
- **Memory usage**: Monitor for memory leaks
- **CPU usage**: Identify performance bottlenecks
- **Bundle size**: Track bundle size changes

## Configuration

Customize the extension behavior:

### Appearance
- Light/Dark theme
- Compact/Expanded view
- Icon position and size

### Behavior
- Auto-refresh settings
- Notification preferences
- Log retention period
- Process filtering

### Privacy
- Local data only (no external connections)
- Configurable data retention
- Clear all data option

## Keyboard Shortcuts

Quick actions with keyboard shortcuts:

- **Alt+B**: Toggle Brummer panel
- **Alt+R**: Restart current process
- **Alt+L**: Clear logs
- **Alt+E**: Jump to latest error

## Coming Soon

Features in development:

- 🔍 Advanced search across all logs
- 📊 Performance profiling
- 🎯 Breakpoint management
- 🔌 Plugin system for custom integrations
- 📱 Mobile browser support