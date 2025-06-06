# MCP Server Configuration Examples

This document shows how to manually configure Brummer (the bumble bee package script manager) as an MCP server in various development tools.

## Claude Desktop

Add to `claude_desktop_config.json`:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`  
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "brummer": {
      "command": "/path/to/brummer",
      "args": ["--port", "7777", "--no-tui"],
      "env": {
        "BRUMMER_MODE": "mcp"
      }
    }
  }
}
```

## Claude Code

Use the Claude Code CLI to install:

```bash
claude mcp add brummer /path/to/brummer --port 7777 --no-tui
```

Or manually add to `~/.claude/claude_code_config.json`:

```json
{
  "mcpServers": {
    "brummer": {
      "command": "/path/to/brummer",
      "args": ["--port", "7777", "--no-tui"],
      "env": {
        "BRUMMER_MODE": "mcp"
      }
    }
  }
}
```

## Cursor

Add to Cursor's MCP configuration:

**macOS**: `~/Library/Application Support/Cursor/User/mcp_servers.json`
**Windows**: `%APPDATA%\Cursor\User\mcp_servers.json`
**Linux**: `~/.config/Cursor/User/mcp_servers.json`

```json
{
  "mcpServers": {
    "brummer": {
      "command": "/path/to/brummer",
      "args": ["--port", "7777", "--no-tui"],
      "env": {
        "BRUMMER_MODE": "mcp"
      }
    }
  }
}
```

## VSCode (with MCP Extension)

Use the VSCode CLI to install:

```bash
code --add-mcp '{"name":"brummer","command":"/path/to/brummer","args":["--port","7777","--no-tui"],"env":{"BRUMMER_MODE":"mcp"}}'
```

Or manually add to VSCode settings.json:

**macOS**: `~/Library/Application Support/Code/User/settings.json`
**Windows**: `%APPDATA%\Code\User\settings.json`
**Linux**: `~/.config/Code/User/settings.json`

```json
{
  "mcp.servers": {
    "brummer": {
      "name": "brummer",
      "command": "/path/to/brummer",
      "args": ["--port", "7777", "--no-tui"],
      "env": {
        "BEAGLE_RUN_MODE": "mcp"
      }
    }
  }
}
```

## Cline

Add to `~/.cline/mcp_config.json`:

```json
[
  {
    "name": "brummer",
    "command": "/path/to/brummer",
    "args": ["--port", "7777", "--no-tui"],
    "env": {
      "BEAGLE_RUN_MODE": "mcp"
    }
  }
]
```

## Windsurf

Add to `~/.windsurf/mcp_servers.json`:

```json
[
  {
    "name": "brummer",
    "command": "/path/to/brummer",
    "args": ["--port", "7777", "--no-tui"],
    "env": {
      "BEAGLE_RUN_MODE": "mcp"
    }
  }
]
```

## Roo Code (Experimental)

Add to `~/.roo/mcp_config.json`:

```json
{
  "name": "brummer",
  "command": "/path/to/brummer",
  "args": ["--port", "7777", "--no-tui"],
  "env": {
    "BEAGLE_RUN_MODE": "mcp"
  }
}
```

## Augment (Experimental)

Add to `~/.augment/mcp_config.json`:

```json
{
  "name": "brummer", 
  "command": "/path/to/brummer",
  "args": ["--port", "7777", "--no-tui"],
  "env": {
    "BEAGLE_RUN_MODE": "mcp"
  }
}
```

## Cody (Experimental)

Add to `~/.cody/mcp_config.json`:

```json
{
  "name": "brummer",
  "command": "/path/to/brummer", 
  "args": ["--port", "7777", "--no-tui"],
  "env": {
    "BEAGLE_RUN_MODE": "mcp"
  }
}
```

## Usage

1. Replace `/path/to/brummer` with the actual path to your brummer executable
2. Adjust the port number if needed (default is 7777)
3. Restart your development tool
4. The MCP server will automatically start when the tool connects

## MCP API Endpoints

When running as an MCP server, Brummer provides these capabilities:

- **Process Management**: Start/stop scripts, view running processes
- **Log Access**: Get logs, search logs, filter by priority
- **Real-time Events**: Subscribe to process events, errors, build events
- **Script Discovery**: List available npm/yarn/pnpm/bun scripts

## Environment Variables

- `BRUMMER_MODE=mcp`: Enables MCP-specific optimizations
- `BRUMMER_LOG_LEVEL=debug`: Enable debug logging for troubleshooting