# 🐝 Brummer

*A TUI package script manager with intelligent monitoring*

A TUI (Terminal User Interface) for managing npm/yarn/pnpm/bun scripts with integrated MCP server for external tool access. Brummer provides intelligent log management, real-time monitoring, and seamless integration with development tools.

## Features

- **Multi-Package Manager Support**: Automatically detects and uses npm, yarn, pnpm, or bun
- **Interactive TUI**: Navigate through scripts, monitor processes, and view logs in real-time
- **Smart Log Management**: 
  - Automatic error detection and prioritization
  - Log filtering and search capabilities
  - Build event and test result detection
- **MCP Server Integration**: Allows external tools (VSCode, Claude Code, etc.) to:
  - Access log output and errors
  - Execute commands asynchronously
  - Monitor process status
- **Process Management**: Start, stop, and monitor multiple processes simultaneously

## Installation

```bash
go install github.com/beagle/brummer/cmd/brummer@latest
```

Or build from source:

```bash
git clone https://github.com/beagle/brummer.git
cd brummer
go build -o brummer ./cmd/brummer
```

## Usage

### Basic Usage

In a directory with a `package.json` file:

```bash
brummer
```

### Options

```bash
brummer [flags]

Flags:
  -d, --dir string   Working directory containing package.json (default ".")
  -p, --port int     MCP server port (default 7777)
      --no-mcp       Disable MCP server
  -h, --help         help for brummer
```

### TUI Navigation

- **Tab**: Switch between views (Scripts, Processes, Logs, Errors, URLs, Settings)
- **↑/↓** or **j/k**: Navigate items
- **Enter**: Select/execute
- **Esc** or **q**: Go back
- **/**: Search logs
- **p**: Toggle high-priority logs

### Process Management

- **Navigate**: Use ↑/↓ arrows to select a process (shows status with colored indicators)
- **s**: Stop selected process (only works on running processes 🟢)
- **r**: Restart selected process (stops then starts the same script)
- **Ctrl+R**: Restart all running processes
- **Enter**: View logs for selected process

**Process Status Indicators:**
- 🟢 **Running** - Process is active (can stop/restart)
- 🔴 **Stopped** - Process was manually stopped
- ❌ **Failed** - Process exited with error
- ✅ **Success** - Process completed successfully
- ⏸️ **Pending** - Process is starting up

**Automatic Cleanup:**
- All running processes are automatically stopped when Brummer exits
- Use Ctrl+C or 'q' to quit with graceful cleanup
- Process count shown in header: "Running Processes (2)"

### Log Management

- **c**: Copy most recent error to clipboard
- **f**: View/manage filters

### Other

- **?**: Show help
- **Ctrl+C**: Quit

### Settings Tab

The Settings tab provides:

- **Package Manager Selection**: Choose between npm, yarn, pnpm, or bun
- **MCP Server Installation**: One-click installation for development tools:
  - Claude Desktop ✓
  - Claude Code ✓  
  - Cursor ✓
  - VSCode (with MCP extension) ✓
  - Cline ✓
  - Windsurf ✓
  - Roo Code (experimental)
  - Augment (experimental)
  - Cody (experimental)

## MCP Server API

The MCP server runs on port 7777 by default and provides RESTful endpoints:

### Connection

```bash
POST /mcp/connect
{
  "clientName": "your-client-name"
}
```

### Endpoints

- `GET /mcp/scripts` - List available scripts
- `GET /mcp/processes` - List running processes
- `GET /mcp/logs?processId=<id>` - Get logs (optional processId filter)
- `POST /mcp/execute` - Execute a script
- `POST /mcp/stop` - Stop a process
- `GET /mcp/search?query=<query>` - Search logs
- `GET /mcp/events` - SSE endpoint for real-time events

### Event Types

- `process.started`
- `process.exited`
- `log.line`
- `error.detected`
- `build.event`
- `test.failed`
- `test.passed`

## Examples

### Run in a specific directory

```bash
brummer -d ~/projects/my-app
```

### Run with custom MCP port

```bash
brummer -p 8888
```

### Run without MCP server (TUI only)

```bash
brummer --no-mcp
```

### Run in headless mode (MCP server only)

```bash
brummer --no-tui
```

## Browser Integration

### Firefox Extension

Brummer includes a Firefox DevTools extension that connects to the MCP server and displays detected URLs directly in the browser's developer tools.

**Features:**
- View detected URLs from your running scripts
- One-click opening of URLs in new tabs
- Real-time updates as new URLs are detected
- Integrated into Firefox Developer Tools
- **Browser Log Integration**: Forward console logs, JavaScript errors, and network requests to Brummer's log system

**Installation:**
```bash
cd browser-extension
# Add icon files to icons/ directory
# Load in Firefox from about:debugging
```

See `browser-extension/README.md` for detailed installation and usage instructions.

## Development

### Project Structure

```
brummer/
├── cmd/brummer/         # Main application entry point
├── internal/
│   ├── tui/             # Terminal UI components
│   ├── process/         # Process management
│   ├── mcp/             # MCP server implementation
│   ├── logs/            # Log storage and detection
│   └── parser/          # Package.json parsing
├── pkg/
│   ├── events/          # Event system
│   └── filters/         # Log filtering
├── browser-extension/   # Firefox DevTools extension
└── go.mod
```

### Building

```bash
go build -o brummer ./cmd/brummer
```

### Testing

```bash
go test ./...
```

### Cleanup Tools

**Check development ports:**
```bash
./check-ports.sh
```

**Clean up orphaned processes:**
```bash
./cleanup-processes.sh
```

These tools help manage orphaned development processes that can occur during testing or if processes aren't properly terminated.

## License

MIT