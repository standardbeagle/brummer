---
sidebar_position: 4
---

# Settings

Configure Brummer and set up integrations with development tools.

## Accessing Settings

Press `6` or Tab to the Settings view to access:
- Package manager selection
- MCP server configuration
- IDE integrations
- Log filters

## Package Manager Configuration

### Auto-Detection
Brummer automatically detects your package manager based on:
- Lock files (package-lock.json, yarn.lock, etc.)
- Global availability
- User preference

### Manual Selection
In Settings view:
1. Navigate to "Package Manager"
2. Press Enter to cycle through options:
   - npm
   - yarn
   - pnpm
   - bun
3. Selection persists for session

## MCP Server Configuration

### Server Settings
- **Port**: Default 7777 (configurable via --port)
- **Auto-start**: Enabled by default
- **Endpoints**: RESTful API + SSE events

### Disabling MCP Server
```bash
brum --no-mcp
```

## IDE Integrations

### Available Integrations

#### Claude Desktop ✓
One-click setup:
1. Select "Claude Desktop" in Settings
2. Press Enter to install
3. Restart Claude Desktop

#### VSCode ✓
Requires MCP extension:
1. Install MCP extension in VSCode
2. Select "VSCode" in Settings
3. Press Enter to configure

#### Cursor ✓
Native MCP support:
1. Select "Cursor" in Settings
2. Press Enter to install
3. Restart Cursor

#### Other IDEs
- Claude Code ✓
- Cline ✓
- Windsurf ✓
- Roo Code (experimental)
- Augment (experimental)

### Installation Process
When installing IDE integration:
1. Brummer creates config file
2. Adds MCP server configuration
3. Shows success/error message
4. May require IDE restart

## Log Filters

### Creating Filters
Future feature - will allow:
- Pattern-based filtering
- Priority adjustments
- Ignore patterns
- Custom highlighting

### Filter Types
- **Error Patterns**: Boost priority for matches
- **Warning Patterns**: Highlight as warnings
- **Ignore Patterns**: Hide matching lines
- **Info Patterns**: Highlight as info

## Configuration Files

### Project Settings
Brummer looks for:
- `.brummer.json` (future)
- `package.json` scripts
- `.env` files

### Global Settings
Stored in:
- `~/.config/brummer/` (Linux/Mac)
- `%APPDATA%\brummer\` (Windows)

## Environment Variables

### Brummer Variables
```bash
BRUMMER_PORT=8080          # MCP server port
BRUMMER_NO_MCP=true        # Disable MCP
BRUMMER_PACKAGE_MANAGER=pnpm  # Force package manager
```

### Process Variables
Scripts inherit environment with:
- Original shell environment
- Project .env files (if present)
- Brummer-specific variables

## Advanced Configuration

### Custom Scripts Location
Future feature - configure custom script paths

### Plugin System
Future feature - extend Brummer with plugins

### Theme Customization
Future feature - custom color schemes

## Troubleshooting Settings

### IDE Integration Failed
1. Check IDE is installed
2. Verify config directory exists
3. Check write permissions
4. Try manual configuration

### Settings Not Saving
- Check disk space
- Verify permissions
- Clear config cache

### Package Manager Issues
1. Ensure manager is installed
2. Check PATH variable
3. Try manual selection
4. Verify lock files

## Best Practices

### IDE Setup
1. Install IDE first
2. Configure MCP integration
3. Test with simple commands
4. Check logs for errors

### Performance Tuning
- Adjust MCP port if conflicts
- Disable MCP if not needed
- Limit concurrent processes

### Security
- MCP server binds to localhost only
- No external access by default
- Token-based authentication (future)

## Common Configurations

### Development Setup
```bash
# High-performance setup
BRUMMER_PORT=8888 brum

# Minimal setup
brum --no-mcp

# Specific package manager
BRUMMER_PACKAGE_MANAGER=yarn brum
```

### CI/CD Usage
```bash
# Headless mode (future)
brum --headless --output=json

# Single script execution (future)
brum run build
```

## Tips

1. **Quick IDE Setup**: Most IDEs auto-configure with one click
2. **Port Conflicts**: Change port with --port flag
3. **Package Manager**: Auto-detection usually works best
4. **Check Status**: MCP status shown in header (🌐 icon)