---
sidebar_position: 4
---

# Quick Start

Get Brummer running in under 2 minutes!

## 1. Install Brummer

```bash
curl -sSL https://raw.githubusercontent.com/beagle/brummer/main/quick-install.sh | bash
```

## 2. Navigate to Your Project

```bash
cd my-javascript-project
```

## 3. Start Brummer

```bash
brum
```

## 4. Run a Script

1. Use **↑/↓** arrows to select a script
2. Press **Enter** to run it
3. Press **Tab** to switch to the Logs view

That's it! You're now using Brummer to manage your npm scripts.

## Essential Shortcuts

| Key | Action |
|-----|--------|
| **Tab** | Switch views |
| **Enter** | Run script / Select |
| **s** | Stop process |
| **r** | Restart process |
| **/** | Search logs |
| **?** | Show help |
| **q** | Quit |

## Next Steps

### Enable MCP Integration

Connect your IDE for enhanced development:

1. Go to Settings tab (press **Tab** until you reach it)
2. Select your IDE (VSCode, Cursor, etc.)
3. Press **Enter** to install MCP configuration

### Try Advanced Features

- **Error Detection**: Errors are automatically highlighted in red
- **URL Detection**: URLs in logs are detected and can be opened
- **Process Management**: Run multiple scripts simultaneously
- **Log Filtering**: Use `/` to search through logs

### Install Browser Extension (Alpha)

For browser debugging integration:
1. See [Browser Extension Guide](./browser-extension/overview)
2. Currently in Alpha - expect some rough edges!

## Common Commands

```bash
# Run in a specific directory
brum -d ./my-app

# Use a different port for MCP server
brum -p 8080

# Run without MCP server
brum --no-mcp
```

## Tips

1. **Package Manager Detection**: Brummer automatically detects npm, yarn, pnpm, or bun
2. **Process Colors**: 
   - 🟢 Running
   - 🔴 Stopped
   - ❌ Failed
   - ✅ Success
3. **Multiple Processes**: Run several scripts and monitor them all in the Processes tab
4. **Smart Scrolling**: Logs auto-scroll unless you manually scroll up

Happy coding with Brummer! 🐝