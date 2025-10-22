# AI Coder Provider Setup Guide

## Overview

This guide explains how to configure and use all available AI coding assistants in Brummer. Each provider integrates with Brummer's MCP server to give AI assistants access to your development environment.

## Quick Start

### 1. Start Brummer
```bash
./brum
# MCP server automatically starts on http://localhost:7777/mcp
```

### 2. Launch an AI Coder
Press `/` to open the command palette, then type:
- `/ai claude` - Anthropic Claude (default)
- `/ai gemini` - Google Gemini
- `/ai copilot` - GitHub Copilot
- `/ai opencode` - OpenCode
- `/ai bash` - Interactive bash shell

## Provider Configurations

### Claude (Anthropic)

**Best For**: General coding, refactoring, complex problem solving

**Installation**:
```bash
npm install -g @anthropic-ai/claude-code
```

**Available Variants**:
| Command | Model | Auto-Approve MCP |
|---------|-------|------------------|
| `/ai claude` | Sonnet (default) | ✅ Yes |
| `/ai claude-secure` | Sonnet | ❌ No (prompts each time) |
| `/ai opus` | Opus (most capable) | ✅ Yes |
| `/ai sonnet` | Sonnet (fast & smart) | ✅ Yes |

**MCP Integration**: ✅ Full support via `--mcp-config`

**Configuration**:
```toml
[ai_coders.providers.claude]
[ai_coders.providers.claude.cli_tool]
command = "claude"
base_args = [
    "--mcp-config",
    "{\"mcpServers\":{\"brummer\":{\"type\":\"http\",\"url\":\"${BRUMMER_MCP_URL}\"}}}",
    "--dangerously-skip-permissions"
]
model = "sonnet"
```

**What the AI Can Do**:
- Read/write files in your project
- Execute commands via Brummer
- View running processes
- Access project URLs
- Read logs and errors

---

### Gemini (Google)

**Best For**: Multi-modal tasks, data analysis, large context windows

**Installation**: Follow [Google AI Studio](https://ai.google.dev/tutorials/setup) instructions

**Available Variants**:
| Command | Model |
|---------|-------|
| `/ai gemini` | Gemini 1.5 Pro |

**MCP Integration**: ⏳ Pending (placeholder ready for when Gemini CLI adds MCP support)

**Configuration**:
```toml
[ai_coders.providers.gemini]
[ai_coders.providers.gemini.cli_tool]
command = "gemini"
base_args = [
    # NOTE: Update when Gemini CLI adds MCP support
    # "--mcp-server", "${BRUMMER_MCP_URL}"
]
```

**Environment Setup**:
```bash
export GEMINI_API_KEY="your-api-key-here"
```

---

### GitHub Copilot

**Best For**: Code completion, GitHub integration, inline suggestions

**Installation**:
```bash
gh extension install github/gh-copilot
```

**Available Variants**:
| Command | Mode |
|---------|------|
| `/ai copilot` | Chat mode |

**MCP Integration**: ⏳ Pending (placeholder ready for when Copilot CLI adds MCP support)

**Configuration**:
```toml
[ai_coders.providers.copilot]
[ai_coders.providers.copilot.cli_tool]
command = "gh"
base_args = [
    "copilot",
    "chat"
    # NOTE: Add MCP config when Copilot CLI supports it
]
```

**Authentication**: Uses `gh` CLI authentication (run `gh auth login` first)

---

### OpenCode

**Best For**: Open-source AI coding assistance

**Installation**: Follow OpenCode documentation

**Available Variants**:
| Command | Mode |
|---------|------|
| `/ai opencode` | Run mode |

**MCP Integration**: ⏳ Pending (placeholder ready)

**Configuration**:
```toml
[ai_coders.providers.opencode]
[ai_coders.providers.opencode.cli_tool]
command = "opencode"
base_args = [
    "run"
    # NOTE: Add MCP config when OpenCode supports it
    # "--mcp-url", "${BRUMMER_MCP_URL}"
]
```

---

### Bash Shell

**Best For**: Manual command execution, scripting, system administration

**Installation**: Pre-installed on most systems

**Available Variants**:
| Command | Shell |
|---------|-------|
| `/ai bash` | Bash (Bourne Again Shell) |
| `/ai zsh` | Zsh (Z Shell) |
| `/ai fish` | Fish (Friendly Interactive Shell) |

**MCP Integration**: N/A (direct shell access)

**Configuration**:
```toml
[ai_coders.providers.bash]
[ai_coders.providers.bash.cli_tool]
command = "bash"
base_args = ["-i"]  # Interactive mode
working_dir = "."
```

**Use Cases**:
- Manual testing of commands
- Running build scripts
- System diagnostics
- Package management
- Git operations

---

## Additional Providers

### Aider

**Best For**: AI pair programming with git integration

**Installation**:
```bash
pip install aider-chat
```

**Usage**: `/ai aider`

**Configuration**:
```toml
[ai_coders.providers.aider]
[ai_coders.providers.aider.cli_tool]
command = "aider"
base_args = ["--yes"]
working_dir = "."
```

**Environment**: Set `OPENAI_API_KEY` or other provider API keys

---

### Local (Ollama)

**Best For**: Privacy-focused, offline coding, running local models

**Installation**:
```bash
# Install Ollama
curl https://ollama.ai/install.sh | sh

# Pull a coding model
ollama pull codellama
```

**Usage**: `/ai local`

**Configuration**:
```toml
[ai_coders.providers.local]
base_url = "http://localhost:11434"
model = "codellama"
max_tokens = 2048
temperature = 0.7
```

**Available Models**:
- `codellama` - Meta's CodeLlama
- `deepseek-coder` - DeepSeek Coder
- `phind-codellama` - Phind's fine-tuned CodeLlama
- `wizardcoder` - WizardCoder

---

## Variable Replacement

Brummer automatically replaces variables in CLI arguments:

| Variable | Example Value | Description |
|----------|--------------|-------------|
| `${BRUMMER_MCP_URL}` | `http://localhost:7777/mcp` | Full MCP server URL |
| `${BRUMMER_MCP_PORT}` | `7777` | Just the port number |
| Any env var | - | Use any system environment variable |

**Example**:
```toml
base_args = [
    "--mcp-config",
    "{\"mcpServers\":{\"brummer\":{\"type\":\"http\",\"url\":\"${BRUMMER_MCP_URL}\"}}}"
]
```

At runtime, becomes:
```toml
base_args = [
    "--mcp-config",
    "{\"mcpServers\":{\"brummer\":{\"type\":\"http\",\"url\":\"http://localhost:7777/mcp\"}}}"
]
```

---

## Keyboard Shortcuts

When in AI Coder view:

| Key | Action |
|-----|--------|
| `Enter` or `Ctrl+T` | Enable terminal focus (all keys go to AI) |
| `ESC` or `Ctrl+T` | Release terminal focus |
| `ESC` (when unfocused) | Exit to session list |
| Left/Right arrows | Navigate views (when not in terminal) |
| `/` (at line start) | Open Brummer command palette |
| `/` (mid-line) | Sent to AI coder |

---

## Comparison Table

| Provider | MCP Support | Auto-Approve | Best For |
|----------|-------------|--------------|----------|
| **Claude** | ✅ Full | ✅ Yes | General coding, refactoring |
| **Claude Secure** | ✅ Full | ❌ No | Security-sensitive work |
| **Gemini** | ⏳ Pending | - | Multi-modal, large context |
| **Copilot** | ⏳ Pending | - | Code completion, GitHub |
| **OpenCode** | ⏳ Pending | - | Open-source alternative |
| **Bash** | N/A | - | Manual commands, scripting |
| **Aider** | ❌ No | ✅ Yes | Git-integrated pair programming |
| **Ollama** | ❌ No | - | Privacy, offline, local models |

---

## Troubleshooting

### Provider Not Found
```
Error: command not found: claude
```

**Solution**: Install the provider CLI tool:
```bash
npm install -g @anthropic-ai/claude-code
```

### MCP Connection Failed
```
Error: Failed to connect to MCP server
```

**Solution**: Verify Brummer's MCP server is running:
```bash
# Check if port 7777 is open
curl http://localhost:7777/mcp
```

### API Key Not Set
```
Error: ANTHROPIC_API_KEY not set
```

**Solution**: Set the environment variable:
```bash
export ANTHROPIC_API_KEY="your-key-here"
# Or add to ~/.bashrc or ~/.zshrc
```

### Permission Denied
```
Error: Permission denied when accessing files
```

**Solution**: Use `claude-secure` provider to see what permissions are requested:
```bash
/ai claude-secure
```

---

## Custom Provider Configuration

To add your own AI coding assistant:

1. **Add provider configuration**:
```toml
[ai_coders.providers.my-custom-ai]
[ai_coders.providers.my-custom-ai.cli_tool]
command = "my-ai"
base_args = [
    "--mcp-url", "${BRUMMER_MCP_URL}",
    "--interactive"
]
working_dir = "."

[ai_coders.providers.my-custom-ai.cli_tool.flag_mapping]
model = "--model"
debug = "--debug"

[ai_coders.providers.my-custom-ai.cli_tool.environment]
MY_API_KEY = "${MY_API_KEY}"
```

2. **Use it**:
```bash
/ai my-custom-ai
```

---

## Related Documentation

- [AI Coder Configuration Reference](./ai-coder-configuration.md) - Complete config options
- [AI Coder Feature Guide](./features/ai-coders.md) - Feature overview
- [Keyboard Priority Order](./keyboard-priority-order.md) - Keyboard handling
- [Quit Confirmation & Terminal Focus](./quit-confirmation-and-terminal-focus.md) - UX features

---

## Getting Help

- **Documentation**: `/docs` directory
- **Issues**: [GitHub Issues](https://github.com/standardbeagle/brummer/issues)
- **In-App Help**: Press `?` in AI Coder view
- **Command Palette**: Press `/` for all available commands
