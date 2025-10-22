# AI Coder Configuration Guide

Complete reference for configuring AI Coder providers and settings in `.brum.toml`.

## Table of Contents

- [Global AI Coder Settings](#global-ai-coder-settings)
- [Provider Configuration](#provider-configuration)
- [Variable Replacement](#variable-replacement)
- [Pre-configured Providers](#pre-configured-providers)
- [Resource Limits](#resource-limits)
- [Workspace Settings](#workspace-settings)
- [Logging Configuration](#logging-configuration)

---

## Global AI Coder Settings

Located under `[ai_coders]`:

```toml
[ai_coders]
enabled = true                              # Enable/disable AI Coder feature
max_concurrent = 3                          # Max simultaneous AI sessions
workspace_base_dir = "~/.brummer/ai-coders" # Base directory for workspaces
default_provider = "claude"                 # Default provider when none specified
timeout_minutes = 30                        # Session timeout
auto_cleanup = true                         # Auto-cleanup old sessions
cleanup_after_hours = 24                    # Cleanup sessions older than this
```

### Settings Explained

- **`enabled`**: Master switch for AI Coder feature
- **`max_concurrent`**: Limit concurrent sessions to prevent resource exhaustion
- **`workspace_base_dir`**: Where AI coder workspaces are created (one per session)
- **`default_provider`**: Used when you type `/ai` without specifying a provider
- **`timeout_minutes`**: Auto-terminate sessions inactive for this long
- **`auto_cleanup`**: Remove old session directories automatically
- **`cleanup_after_hours`**: How long before a session is considered old

---

## Provider Configuration

Two types of providers:

1. **API-based providers**: Direct API integration (Claude, Gemini)
2. **CLI tool providers**: Wrap existing CLI tools (claude, aider, opencode)

### API-Based Provider Structure

```toml
[ai_coders.providers.PROVIDER_NAME]
api_key_env = "ENV_VAR_NAME"          # Environment variable with API key
model = "model-name"                   # Model identifier
max_tokens = 4096                      # Max tokens per request
temperature = 0.7                      # Randomness (0.0-1.0)
request_timeout_seconds = 30           # Request timeout

[ai_coders.providers.PROVIDER_NAME.rate_limit]
requests_per_minute = 50               # RPM limit
tokens_per_minute = 150000             # TPM limit
```

### CLI Tool Provider Structure

```toml
[ai_coders.providers.PROVIDER_NAME]
[ai_coders.providers.PROVIDER_NAME.cli_tool]
command = "command-name"               # Executable name
base_args = ["--flag", "value"]        # Always-included arguments
working_dir = "."                      # Working directory (optional)
model = "model-name"                   # Default model (optional)

[ai_coders.providers.PROVIDER_NAME.cli_tool.flag_mapping]
model = "--model"                      # How to pass model parameter
message = "--message"                  # How to pass messages
debug = "--debug"                      # How to enable debug mode

[ai_coders.providers.PROVIDER_NAME.cli_tool.environment]
VAR_NAME = "value"                     # Additional environment variables
```

---

## Variable Replacement

Brummer provides **automatic variable replacement** in CLI arguments using `${VAR_NAME}` syntax.

### Built-in Variables

Brummer automatically provides these variables:

| Variable | Example Value | Description |
|----------|--------------|-------------|
| `${BRUMMER_MCP_URL}` | `http://localhost:7777/mcp` | Full MCP server URL |
| `${BRUMMER_MCP_PORT}` | `7777` | Just the port number |
| Any system env var | - | Use any environment variable |

### Example Usage

```toml
[ai_coders.providers.claude]
[ai_coders.providers.claude.cli_tool]
command = "claude"
base_args = [
    "--mcp-config",
    "{\"mcpServers\":{\"brummer\":{\"type\":\"http\",\"url\":\"${BRUMMER_MCP_URL}\"}}}",
    "--dangerously-skip-permissions"
]
```

At runtime, `${BRUMMER_MCP_URL}` is replaced with the actual URL!

---

## Pre-configured Providers

### Claude (CLI Tool)

Uses the `claude` CLI from Anthropic.

```toml
[ai_coders.providers.claude]
[ai_coders.providers.claude.cli_tool]
command = "claude"
base_args = [
    "--mcp-config",
    "{\"mcpServers\":{\"brummer\":{\"type\":\"http\",\"url\":\"${BRUMMER_MCP_URL}\"}}}",
    "--dangerously-skip-permissions"  # Auto-approve MCP actions
]
model = "sonnet"

[ai_coders.providers.claude.cli_tool.flag_mapping]
model = "--model"
output_format = "--output-format"
debug = "--debug"
```

**Usage**: `/ai claude` or just `/ai` (if default provider)

**Models available**:
- `sonnet` - Claude 3.5 Sonnet (fast, smart)
- `opus` - Claude 3 Opus (most capable)
- `haiku` - Claude 3 Haiku (fastest)

### Claude Secure

Same as Claude but **without** `--dangerously-skip-permissions` - prompts for each MCP action.

```toml
[ai_coders.providers.claude-secure]
[ai_coders.providers.claude-secure.cli_tool]
command = "claude"
base_args = [
    "--mcp-config",
    "{\"mcpServers\":{\"brummer\":{\"type\":\"http\",\"url\":\"${BRUMMER_MCP_URL}\"}}}"
    # No --dangerously-skip-permissions
]
```

**Usage**: `/ai claude-secure`

### Opus / Sonnet (Specific Models)

Pre-configured for specific Claude models:

```toml
[ai_coders.providers.opus]
[ai_coders.providers.opus.cli_tool]
command = "claude"
base_args = [
    "--mcp-config", "{...}",
    "--model", "opus",
    "--dangerously-skip-permissions"
]
```

**Usage**: `/ai opus` or `/ai sonnet`

### Aider

Integrates with [aider](https://aider.chat/) - AI pair programming tool.

```toml
[ai_coders.providers.aider]
[ai_coders.providers.aider.cli_tool]
command = "aider"
base_args = ["--yes"]  # Auto-confirm changes
working_dir = "."

[ai_coders.providers.aider.cli_tool.flag_mapping]
model = "--model"
message = "--message"

[ai_coders.providers.aider.cli_tool.environment]
# Aider uses OPENAI_API_KEY from environment
```

**Usage**: `/ai aider`

### OpenCode

Integrates with OpenCode CLI.

```toml
[ai_coders.providers.opencode]
[ai_coders.providers.opencode.cli_tool]
command = "opencode"
base_args = ["run"]

[ai_coders.providers.opencode.cli_tool.flag_mapping]
model = "--model"
prompt = "--prompt"
debug = "--log-level"
```

**Usage**: `/ai opencode`

### Gemini

Google's Gemini via CLI or API.

```toml
[ai_coders.providers.gemini]
api_key_env = "GEMINI_API_KEY"
model = "gemini-1.5-pro"
max_tokens = 8192
temperature = 0.7
request_timeout_seconds = 30

[ai_coders.providers.gemini.rate_limit]
requests_per_minute = 15
tokens_per_minute = 1000000
```

**Usage**: `/ai gemini`

### Bash (Interactive Shell)

Simple bash shell session - no AI, just terminal.

```toml
[ai_coders.providers.bash]
model = "bash"
max_tokens = 1000
temperature = 0.0
request_timeout_seconds = 10
```

**Usage**: `/ai bash`

**Available shells**: `bash`, `sh`, `zsh`, `fish`, `python`, `node`, `ruby`

### Local (Ollama)

For local models via Ollama.

```toml
[ai_coders.providers.local]
base_url = "http://localhost:11434"
model = "codellama"
max_tokens = 2048
temperature = 0.7
request_timeout_seconds = 60
```

**Usage**: `/ai local`

---

## Resource Limits

Protect your system from runaway AI sessions:

```toml
[ai_coders.resource_limits]
max_memory_mb = 512           # Max RAM per session
max_disk_space_mb = 1024      # Max disk space per session
max_cpu_percent = 50          # Max CPU usage (%)
max_processes = 5             # Max concurrent processes
max_files_per_coder = 100     # Max files in workspace
```

### How Limits Work

- **Memory**: Session killed if RAM usage exceeds limit
- **Disk**: Prevents workspace from growing too large
- **CPU**: Throttles CPU-intensive operations
- **Processes**: Limits subprocess spawning
- **Files**: Prevents workspace clutter

---

## Workspace Settings

Control AI coder workspace behavior:

```toml
[ai_coders.workspace]
template = "basic"                      # Workspace template to use
gitignore_rules = [                     # Always ignore these patterns
    "node_modules/",
    ".env",
    "*.log",
    "dist/",
    "build/"
]
allowed_extensions = [                  # Only allow these file types
    ".go", ".js", ".ts", ".py", ".md",
    ".json", ".yaml", ".toml", ".txt", ".sh"
]
forbidden_paths = [                     # Never allow access to these
    "/etc", "/var", "/sys", "/proc"
]
max_file_size_mb = 10                   # Max individual file size
backup_enabled = true                   # Backup files before modification
```

### Workspace Templates

Available templates:
- **`basic`**: Empty directory
- **`node`**: Node.js project structure
- **`python`**: Python project structure
- **`go`**: Go module structure

---

## Logging Configuration

Control AI Coder logging:

```toml
[ai_coders.logging]
level = "info"                 # Logging level: debug, info, warn, error
output_file = "ai-coders.log"  # Log file path
rotate_size_mb = 50            # Rotate when file exceeds this size
keep_rotations = 5             # Keep this many old log files
include_ai_output = false      # Include AI responses in logs
```

### Log Levels

- **`debug`**: Very verbose - all operations
- **`info`**: Normal operations (default)
- **`warn`**: Warnings and errors only
- **`error`**: Errors only

---

## Usage Examples

### Example 1: Add Your Own AI Tool

```toml
[ai_coders.providers.my-ai-tool]
[ai_coders.providers.my-ai-tool.cli_tool]
command = "my-ai"
base_args = ["--interactive", "--context", "${BRUMMER_MCP_URL}"]
working_dir = "."

[ai_coders.providers.my-ai-tool.cli_tool.flag_mapping]
model = "--model"
temperature = "--temp"

[ai_coders.providers.my-ai-tool.cli_tool.environment]
MY_AI_API_KEY = "${MY_API_KEY}"  # Uses env var
```

Then use: `/ai my-ai-tool`

### Example 2: Multiple Claude Configurations

```toml
# Fast mode - auto-approve everything
[ai_coders.providers.claude-fast]
[ai_coders.providers.claude-fast.cli_tool]
command = "claude"
base_args = ["--mcp-config", "{...}", "--model", "sonnet", "--dangerously-skip-permissions"]

# Safe mode - confirm each action
[ai_coders.providers.claude-safe]
[ai_coders.providers.claude-safe.cli_tool]
command = "claude"
base_args = ["--mcp-config", "{...}", "--model", "opus"]  # No dangerous flag

# Powerful mode - use Opus with auto-approve
[ai_coders.providers.claude-power]
[ai_coders.providers.claude-power.cli_tool]
command = "claude"
base_args = ["--mcp-config", "{...}", "--model", "opus", "--dangerously-skip-permissions"]
```

Use: `/ai claude-fast`, `/ai claude-safe`, `/ai claude-power`

### Example 3: Customized Resource Limits

For a powerful workstation:

```toml
[ai_coders.resource_limits]
max_memory_mb = 2048          # 2GB RAM
max_disk_space_mb = 10240     # 10GB disk
max_cpu_percent = 80          # 80% CPU
max_processes = 10
max_files_per_coder = 1000
```

For a limited environment:

```toml
[ai_coders.resource_limits]
max_memory_mb = 256           # 256MB RAM
max_disk_space_mb = 512       # 512MB disk
max_cpu_percent = 25          # 25% CPU
max_processes = 2
max_files_per_coder = 50
```

---

## Configuration File Locations

Brummer looks for configuration in this order:

1. `./.brum.toml` - Current directory
2. `~/.config/brummer/config.toml` - User config
3. `/etc/brummer/config.toml` - System config

**Use `.brum.example.toml` as a starting point!**

---

## See Also

- [AI Coders Feature Guide](./features/ai-coders.md) - How to use AI Coders
- [MCP Integration](./mcp/integration-overview.md) - How MCP connects to AI Coders
- [Configuration Examples](./configuration/examples.md) - More config examples
