# 🐝 Brummer

*Your Terminal UI Development Buddy with intelligent monitoring*

A TUI (Terminal User Interface) for managing npm/yarn/pnpm/bun scripts with integrated MCP server for external tool access. Brummer provides intelligent log management, real-time monitoring, and seamless integration with development tools.

## 📖 Documentation

📚 **Full documentation available at: [https://standardbeagle.github.io/brummer/](https://standardbeagle.github.io/brummer/)**

Quick links:
- [Getting Started Guide](https://standardbeagle.github.io/brummer/docs/getting-started)
- [Installation Options](https://standardbeagle.github.io/brummer/docs/installation)
- [MCP Integration](https://standardbeagle.github.io/brummer/docs/mcp-integration/overview)

## Features

- **Multi-Package Manager Support**: Automatically detects and uses npm, yarn, pnpm, or bun
- **Monorepo Support**: Full support for pnpm workspaces, npm workspaces, yarn workspaces, Lerna, Nx, and Rush
- **Multi-Language Detection**: Auto-detects commands for Node.js, Go, Rust, Java (Gradle/Maven), .NET, Python, Ruby, PHP, Flutter, and more
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
- **VS Code Tasks**: Detects and runs VS Code tasks from .vscode/tasks.json

## Installation

### Quick Install (One-Liner)

<details>
<summary><b>🐧 Linux/macOS</b></summary>

```bash
curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
```

Or with wget:
```bash
wget -qO- https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
```
</details>

<details>
<summary><b>🪟 Windows (PowerShell)</b></summary>

```powershell
irm https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.ps1 | iex
```
</details>

### Package Managers

<details>
<summary><b>📦 NPM/NPX</b></summary>

```bash
# Run directly with npx (no installation)
npx @standardbeagle/brum

# Install globally
npm install -g @standardbeagle/brum

# Or with yarn
yarn global add @standardbeagle/brum

# Or with pnpm
pnpm add -g @standardbeagle/brum
```
</details>


<details>
<summary><b>🐹 Go Install</b></summary>

```bash
go install github.com/standardbeagle/brummer/cmd/brum@latest
```
</details>

### Install from Source

<details>
<summary><b>Build from source</b></summary>

```bash
# Clone the repository
git clone https://github.com/standardbeagle/brummer
cd brummer

# Using Make (recommended)
make install-user    # Install for current user (~/.local/bin)
# OR
make install        # Install system-wide (requires sudo)

# Using the interactive installer
./install.sh

# Manual build
go build -o brum ./cmd/brum
mv brum ~/.local/bin/  # Add to PATH
```
</details>

## Usage

### Basic Usage

In a directory with a `package.json` file:

```bash
brum
```

### Options

```bash
brum [flags]

Flags:
  -d, --dir string   Working directory containing package.json (default ".")
  -p, --port int     MCP server port (default 7777)
      --no-mcp       Disable MCP server
      --settings     Show current configuration settings with sources
  -h, --help         help for brum
```

### TUI Navigation

- **Tab**: Switch between views (Scripts, Processes, Logs, Errors, URLs, Settings)
- **↑/↓** or **j/k**: Navigate items
- **Enter**: Select/execute
- **n**: Open run command dialog (from Scripts tab)
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

## Configuration

Brummer supports hierarchical configuration through `.brum.toml` files. Configuration is loaded in the following order (later values override earlier ones):

1. `~/.brum.toml` (global user settings)
2. Project root and parent directories (walking up to root)
3. Current working directory `.brum.toml`

### Viewing Current Configuration

```bash
# Show current settings with source files
brum --settings

# Create a configuration file from current settings
brum --settings > .brum.example.toml
```

### Configuration Options

Create a `.brum.toml` file in your project or home directory:

```toml
# Package manager preference
preferred_package_manager = "pnpm"  # npm, yarn, pnpm, bun

# MCP Server settings
mcp_port = 7777
no_mcp = false

# Proxy settings
proxy_port = 19888
proxy_mode = "reverse"  # "reverse" or "full"
proxy_url = "http://localhost:3000"  # Optional: auto-proxy this URL
standard_proxy = false
no_proxy = false
```

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

## MCP Hub Mode 🆕

Brummer Hub Mode enables MCP clients to discover and control multiple brummer instances through a single connection:

### Quick Setup

1. **Configure your MCP client once:**
   ```json
   {
     "servers": {
       "brummer-hub": {
         "command": "brum",
         "args": ["--mcp"]
       }
     }
   }
   ```

2. **Run brummer in each project:**
   ```bash
   cd ~/projects/frontend && brum
   cd ~/projects/backend && brum
   ```

3. **Use hub tools in your MCP client:**
   - `instances/list` - See all running instances
   - `instances/connect` - Connect to a specific instance
   - All instance tools become available with prefixes

[📚 Full Hub Mode Documentation](docs/hub-mode.md)

## MCP Server Integration

The MCP server runs on port 7777 by default and implements the official MCP Streamable HTTP transport protocol:

### Transport Protocol

**Single Endpoint**: `http://localhost:7777/mcp`

**Connection Types**:
1. **Standard JSON-RPC** (POST with `Accept: application/json`)
2. **Server-Sent Events** (GET with `Accept: text/event-stream`)
3. **SSE Response** (POST with `Accept: text/event-stream`)

### Example Connection

```javascript
// SSE streaming connection
const eventSource = new EventSource('http://localhost:7777/mcp');
eventSource.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Received:', msg);
};

// Send JSON-RPC requests
fetch('http://localhost:7777/mcp', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json'
  },
  body: JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    method: 'tools/list'
  })
});
```

### Available Tools

**Script Management**: `scripts_list`, `scripts_run`, `scripts_stop`, `scripts_status`
**Log Management**: `logs_stream`, `logs_search`
**Browser Tools**: `browser_open`, `browser_screenshot`, `browser_navigate`, `repl_execute`
**Proxy Tools**: `proxy_requests`
**Telemetry**: `telemetry_sessions`, `telemetry_events`

## Examples

### Run in a specific directory

```bash
brum -d ~/projects/my-app
```

### Run with custom MCP port

```bash
brum -p 8888
```

### Run without MCP server (TUI only)

```bash
brum --no-mcp
```

### Run in headless mode (MCP server only)

```bash
brum --no-tui
```


## Development

### Project Structure

```
brummer/
├── cmd/brum/            # Main application entry point
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
go build -o brum ./cmd/brum
```

### Testing

```bash
go test ./...
```



## License

MIT