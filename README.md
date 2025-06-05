# Beagle Run

A powerful TUI (Terminal User Interface) for managing npm/yarn/pnpm/bun scripts with integrated MCP server for external tool access.

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
go install github.com/beagle/beagle-run/cmd/beagle-run@latest
```

Or build from source:

```bash
git clone https://github.com/beagle/beagle-run.git
cd beagle-run
go build -o beagle-run ./cmd/beagle-run
```

## Usage

### Basic Usage

In a directory with a `package.json` file:

```bash
beagle-run
```

### Options

```bash
beagle-run [flags]

Flags:
  -d, --dir string   Working directory containing package.json (default ".")
  -p, --port int     MCP server port (default 7777)
      --no-mcp       Disable MCP server
  -h, --help         help for beagle-run
```

### TUI Navigation

- **Tab**: Switch between views (Scripts, Processes, Logs)
- **↑/↓** or **j/k**: Navigate items
- **Enter**: Select/execute
- **Esc** or **q**: Go back
- **/**: Search logs
- **p**: Toggle high-priority logs
- **s**: Stop selected process
- **f**: View/manage filters
- **?**: Show help
- **Ctrl+C**: Quit

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
beagle-run -d ~/projects/my-app
```

### Run with custom MCP port

```bash
beagle-run -p 8888
```

### Run without MCP server (TUI only)

```bash
beagle-run --no-mcp
```

## Development

### Project Structure

```
beagle-run/
├── cmd/beagle-run/      # Main application entry point
├── internal/
│   ├── tui/             # Terminal UI components
│   ├── process/         # Process management
│   ├── mcp/             # MCP server implementation
│   ├── logs/            # Log storage and detection
│   └── parser/          # Package.json parsing
├── pkg/
│   ├── events/          # Event system
│   └── filters/         # Log filtering
└── go.mod
```

### Building

```bash
go build -o beagle-run ./cmd/beagle-run
```

### Testing

```bash
go test ./...
```

## License

MIT