# 🐝 Brummer Firefox Extension

A Firefox DevTools extension that connects to the Brummer package script manager, allowing you to view and open URLs detected in your development logs directly from the browser.

## Features

- **DevTools Integration**: Adds a "🐝 Brummer" panel to Firefox Developer Tools
- **Real-time URL Detection**: Shows URLs detected by Brummer from your running scripts
- **One-click Opening**: Open detected URLs directly in new browser tabs
- **Live Updates**: Automatically refreshes when new URLs are detected
- **Connection Management**: Configure and manage connection to your Brummer instance
- **Browser Log Integration**: Forward console logs, JavaScript errors, and network requests to Brummer
- **Comprehensive Error Tracking**: Captures all browser-side issues in your development workflow

## Installation

### From Source (Development)

1. **Prepare the extension files:**
   ```bash
   cd browser-extension
   ```

2. **Create icon files** (see `icons/README.md` for details):
   - Add `bee-16.png`, `bee-32.png`, `bee-48.png`, and `bee-128.png` to the `icons/` directory

3. **Load in Firefox:**
   - Open Firefox
   - Navigate to `about:debugging`
   - Click "This Firefox"
   - Click "Load Temporary Add-on..."
   - Select the `manifest.json` file from this directory

### For Production Use

Package the extension and submit to Firefox Add-ons store (not yet available).

## Usage

1. **Start Brummer** with MCP server enabled:
   ```bash
   brummer --port 7777
   ```

2. **Open Firefox Developer Tools** (F12)

3. **Navigate to the "🐝 Brummer" tab** in the DevTools

4. **Configure connection:**
   - Default server: `http://localhost:7777`
   - Click "Connect" to establish connection

5. **View detected URLs:**
   - URLs from your running scripts will appear automatically
   - Click "Open" next to any URL to open it in a new tab
   - URLs are sorted by most recent first

6. **Enable browser logging (optional):**
   - Toggle "Forward Browser Logs to Brummer" to capture browser events
   - View console logs, errors, and network requests in Brummer's log tab
   - All captured logs appear alongside your script logs

## Features in Detail

### URL Detection

The extension connects to Brummer's MCP server and:
- Monitors all log output from running scripts
- Extracts HTTP/HTTPS URLs using regex pattern matching
- Deduplicates URLs to show unique entries
- Shows context (the log line where the URL was found)
- Updates in real-time as new logs are generated

### DevTools Integration

- **Panel Location**: Available in Firefox DevTools alongside Console, Network, etc.
- **Connection Status**: Shows connected/disconnected status in the header
- **Settings**: Configure Brummer server URL (default: localhost:7777)
- **Error Handling**: Displays connection errors and troubleshooting info

### URL Management

- **Time Stamps**: Shows when each URL was detected
- **Process Context**: Displays which script/process generated the URL
- **Quick Access**: One-click opening in new browser tabs
- **Smart Filtering**: Automatically removes duplicates

### Browser Log Integration

When enabled, the extension captures and forwards to Brummer:

- **Console Logs**: All console.log, console.warn, console.error output
- **JavaScript Errors**: Runtime errors with file/line information
- **Promise Rejections**: Unhandled promise rejections
- **Resource Errors**: Failed image, CSS, JS file loads
- **Network Requests**: Fetch and XMLHttpRequest calls with timing
- **Network Errors**: Failed API calls and network issues
- **Page Navigation**: SPA route changes and page loads

All logs include:
- Browser tab context (title, URL)
- Timestamps for correlation with script logs
- Error categorization for easy filtering
- Performance metrics for network requests

## Configuration

### Brummer Server Settings

- **Default Port**: 7777
- **Default Host**: localhost
- **Custom Configuration**: Change server URL in the extension panel

### Supported Brummer Versions

- Works with Brummer 1.0.0+
- Requires MCP server enabled (default behavior)
- Compatible with all package managers (npm, yarn, pnpm, bun)

## Development

### File Structure

```
browser-extension/
├── manifest.json          # Extension manifest
├── devtools.html          # DevTools page entry point
├── devtools.js            # DevTools panel creation
├── panel.html             # Main panel UI
├── panel.js               # Panel logic and Brummer connection
├── background.js          # Background script
├── content.js             # Content script (optional features)
├── icons/                 # Extension icons
│   ├── bee-16.png
│   ├── bee-32.png
│   ├── bee-48.png
│   └── bee-128.png
└── README.md              # This file
```

### Key Components

1. **DevTools Panel**: Main interface showing detected URLs
2. **MCP Connection**: Connects to Brummer's HTTP endpoints
3. **Real-time Updates**: Uses Server-Sent Events for live updates
4. **Storage**: Persists connection settings

### API Integration

The extension uses Brummer's MCP server endpoints:

- `POST /mcp/connect` - Establish client connection
- `GET /mcp/logs` - Fetch log entries
- `GET /mcp/events` - Real-time event stream
- URLs are extracted from log content using regex

## Troubleshooting

### Connection Issues

1. **Ensure Brummer is running**:
   ```bash
   brummer --port 7777
   ```

2. **Check server URL** in extension settings

3. **Verify MCP server is enabled** (it is by default)

4. **Check browser console** for error messages

### No URLs Appearing

1. **Run some scripts** in Brummer that generate URLs
2. **Check that scripts are producing log output**
3. **Verify the logs contain HTTP/HTTPS URLs**
4. **Refresh the connection** by disconnecting and reconnecting

### DevTools Panel Not Visible

1. **Ensure extension is loaded** in `about:debugging`
2. **Restart Firefox** if extension was just installed
3. **Check that you're in Developer Tools** (F12)

## Contributing

1. Fork the repository
2. Make changes to the browser extension
3. Test with a local Brummer instance
4. Submit a pull request

## License

Same as Brummer main project (MIT)