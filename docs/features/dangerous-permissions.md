# AI Tool Permission Bypass Configuration

## Overview

Many AI tools (especially Claude with security restrictions) support special CLI flags to bypass permission checks and safety restrictions. Brummer allows users to configure these flags per-provider using the `extra_args` configuration field.

## Configuration Format

In your `.brum.toml` file, you can specify additional CLI arguments for each AI provider:

```toml
[ai_coders]
  [ai_coders.providers]

[ai_coders.providers.claude]
  extra_args = [
    "--yolo",
    "--dangerously-skip-permissions"
  ]

[ai_coders.providers.claude-secure]
  extra_args = [
    "--yolo",
    "--dangerously-skip-permissions"
  ]

[ai_coders.providers.gemini]
  extra_args = [
    "--yolo"
  ]
```

## Examples

### Claude with MCP Connection
```toml
[ai_coders.providers.claude]
  command = "claude"
  # MCP URL is passed via .claude/settings.local.json file created automatically
```

### Claude Secure with MCP Connection
```toml
[ai_coders.providers.claude-secure]
  command = "claude-secure"
  # MCP URL is passed via .claude/settings.local.json file created automatically
```

### Gemini with YOLO Mode
```toml
[ai_coders.providers.gemini]
  extra_args = [
    "--yolo"
  ]
```

### Cursor with No Restrictions
```toml
[ai_coders.providers.cursor]
  extra_args = [
    "--yolo"
  ]
```

## Available AI Tool Flags

Different AI tools support various permission bypass flags:

### Common Flags
- `--yolo`: "You Only Live Once" - bypass most restrictions and safety checks
- `--dangerously-skip-permissions`: Skip all permission prompts and safety checks

### Tool-Specific Examples
- **Claude**: MCP connection via `.claude/settings.local.json` file (auto-created)
- **Claude-secure**: MCP connection via `.claude/settings.local.json` file (auto-created)
- **Gemini**: `--yolo` (command-line flag)
- **Cursor**: `--yolo` (command-line flag)
- **Aider**: `--yolo` (command-line flag)
- **OpenAI CLI**: `--dangerously-skip-permissions` (command-line flag)

## How It Works

### For Claude/Claude-secure: Settings File Approach
1. **Configuration Loading**: When user runs `/ai claude`, Brummer loads the provider config
2. **Settings File Creation**: Brummer automatically creates `.claude/settings.local.json` with MCP server configuration
3. **Environment Setup**: Brummer sets `CLAUDE_SETTINGS_PATH` to point to the settings file
4. **Claude Startup**: Claude reads the settings file and connects to the configured MCP servers
5. **MCP Integration**: Claude can now interact with Brummer's MCP server seamlessly

### Settings File Format
```json
{
  "mcpServers": [
    {
      "name": "Brummer",
      "command": "mcp-server",
      "args": ["--port", "${BRUMMER_MCP_PORT:-7777}"]
    }
  ]
}
```

**Benefits of Environment Variable Approach**:
- ✅ **Dynamic**: Port can change based on actual port Brummer uses
- ✅ **Flexible**: Can reference multiple environment variables with defaults
- ✅ **Portable**: Same settings file works across different environments
- ✅ **Clean**: Minimal configuration with maximum flexibility
- ✅ **Claude Native**: Uses Claude's `${VAR:-default}` syntax for defaults

### For Other AI Tools: Command Line Flags
1. **Configuration Loading**: When user runs `/ai <provider>`, Brummer loads the provider config
2. **Argument Assembly**: CLI tool provider combines `base_args` + `extra_args` + runtime args
3. **Execution**: AI tool receives complete argument list including permission bypasses
4. **PTY Integration**: Arguments are passed to AI executable running in PTY session

### Argument Processing Order
```
Final Command = base_args + extra_args + runtime_args
```

## Security Notes

⚠️ **Warning**: These flags give AI tools elevated privileges that bypass important safety restrictions.

- `--yolo`: Bypasses most AI safety mechanisms and content filters
- `--dangerously-skip-permissions`: Skips all interactive permission prompts
- File access restrictions may be completely disabled
- Content safety filters are typically bypassed
- Output token limits may be removed

### Recommended Usage

- **Development**: Use with trusted AI tools in controlled environments
- **Testing**: Excellent for automated testing without permission prompts
- **Production**: Use with extreme caution and only with tools you fully trust

## Provider Configuration Examples

### Claude (Standard Version)
```toml
[ai_coders.providers.claude]
  command = "claude"
  extra_args = [
    "--dangerously-skip-permissions"
  ]
```

### Claude Secure
```toml
[ai_coders.providers.claude-secure]
  command = "claude-secure"
  # claude-secure doesn't need extra_args for permission bypass
```

### Gemini
```toml
[ai_coders.providers.gemini]
  command = "gemini"
  extra_args = [
    "--yolo"
  ]
```

This configuration system allows maximum flexibility while keeping permission bypass settings clearly documented per provider.