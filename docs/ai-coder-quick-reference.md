# AI Coder Quick Reference Card

## Starting an AI Coder Session

Press `/` then type `/ai <provider>`

| Command | Provider | Auto-Approve | Model |
|---------|----------|--------------|-------|
| `/ai claude` | Anthropic Claude | ✅ Yes | Sonnet (default) |
| `/ai claude-secure` | Anthropic Claude | ❌ No | Sonnet |
| `/ai opus` | Anthropic Claude | ✅ Yes | Opus (most capable) |
| `/ai sonnet` | Anthropic Claude | ✅ Yes | Sonnet (fast) |
| `/ai gemini` | Google Gemini | ✅ Yes | Gemini 1.5 Pro |
| `/ai gemini-secure` | Google Gemini | ❌ No | Gemini 1.5 Pro |
| `/ai copilot` | GitHub Copilot | ✅ Yes | Default model |
| `/ai copilot-secure` | GitHub Copilot | ❌ No | Default model |
| `/ai opencode` | OpenCode | - | Default model |
| `/ai bash` | Bash Shell | N/A | N/A |
| `/ai zsh` | Z Shell | N/A | N/A |
| `/ai fish` | Fish Shell | N/A | N/A |

## Keyboard Shortcuts (in AI Coder view)

| Key | Action |
|-----|--------|
| `Enter` | Enable terminal focus (all keys to AI) |
| `Ctrl+T` | Toggle terminal focus on/off |
| `ESC` | Release focus (or exit to list) |
| `Left/Right` | Navigate views (when not focused) |
| `/` at line start | Open Brummer command palette |
| `/` mid-line | Send to AI coder |

## Terminal Focus Modes

### Unfocused Mode
- ESC = Exit to session list
- Enter = Enable focus + send Enter to terminal
- Ctrl+T = Enable focus
- Other keys = Sent to terminal

### Focused Mode (🔒)
- ESC = Release focus
- Ctrl+T = Release focus
- All other keys = Sent to terminal

## MCP Integration Status

| Provider | MCP Support | Configuration Method |
|----------|-------------|---------------------|
| Claude | ✅ Full | `--mcp-config` JSON with `${BRUMMER_MCP_URL}` |
| Gemini | ✅ Full | `gemini mcp add brummer http URL` + `--allowed-mcp-server-names` |
| Copilot | ✅ Full | `--additional-mcp-config` JSON with `${BRUMMER_MCP_URL}` |
| OpenCode | ⏳ Pending | Ready for when MCP support is added |

## What AIs Can Do via MCP

When connected to Brummer's MCP server, AI assistants can:

✅ **Read/Write Files** - Access project files
✅ **Execute Commands** - Run shell commands via Brummer
✅ **View Processes** - See running development servers
✅ **Access URLs** - View discovered project URLs
✅ **Read Logs** - Access application logs and errors
✅ **Control Brummer** - Interact with Brummer features

## Quick Troubleshooting

### AI Coder won't start
```bash
# Check if CLI is installed
which claude  # or gemini, copilot, etc.

# Install if missing
npm install -g @anthropic-ai/claude-code
```

### MCP connection failed
```bash
# Verify Brummer's MCP server is running
curl http://localhost:7777/mcp

# Should return JSON with server info
```

### Colors not showing
- Press `Ctrl+T` to ensure terminal focus is enabled
- Check terminal supports 24-bit color: `echo $COLORTERM` → should show `truecolor`

### Can't exit application
- Press ESC **twice** within 2 seconds (quit confirmation)
- Or press Ctrl+C **twice** within 2 seconds

## Configuration File Location

`.brum.example.toml` contains all provider configurations.

Copy to create your own:
```bash
cp .brum.example.toml .brum.toml
```

Edit the `[ai_coders.providers.*]` sections to customize.

## First-Time Setup (One-Time)

### Claude
```bash
npm install -g @anthropic-ai/claude-code
# CLI handles authentication automatically
```

### Gemini
```bash
npm install -g @google/generative-ai-cli
gemini mcp add brummer http http://localhost:7777/mcp
```

### Copilot
```bash
npm install -g @githubnext/github-copilot-cli
# Uses GitHub authentication
```

### OpenCode
```bash
npm install -g opencode
```

## Common Use Cases

### 1. Quick Code Review
```
/ai claude
> Review the changes in main.go and suggest improvements
```

### 2. Debug Error
```
/ai opus
> The test in user_test.go is failing. Can you fix it?
```

### 3. Refactor Code
```
/ai gemini
> Refactor the auth middleware to use context-based auth
```

### 4. Run Commands
```
/ai bash
$ npm test
$ git status
```

### 5. Safe Mode (with prompts)
```
/ai claude-secure
> Add error handling to all database calls
[Prompts before each file modification]
```

## View Switching

| Key | View |
|-----|------|
| `1` | Processes |
| `2` | Logs |
| `3` | Errors |
| `4` | URLs |
| `5` | AI Coders |
| `6` | MCP |
| `7` | Settings |
| `Left/Right` | Previous/Next view |

## Exit Confirmation

First press: "Press ESC or Ctrl+C again to quit"
Second press (within 2s): Application quits

Prevents accidental exits!

## Getting Help

- In-app: Press `?` in AI Coder view
- Command palette: Press `/`
- Full docs: `/docs` directory
- Provider setup: `docs/ai-coder-provider-setup.md`
- Configuration: `docs/ai-coder-configuration.md`
