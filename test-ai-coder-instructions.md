# Testing AI Coder Tab

## Quick Test

1. Run `./brum` to start Brummer
2. Press Tab to navigate to the "AI Coders" tab
3. Try one of these commands:
   - `/ai terminal` - Opens a basic terminal session
   - `/ai test-claude` - Opens a test session that shows MCP integration
   - `/ai claude` - Opens Claude CLI (requires Claude to be installed)

## Controls in AI Coder View

- **Enter**: Focus the terminal (allows typing)
- **Ctrl+Q**: Unfocus the terminal (return control to Brummer)
- **F11**: Toggle full-screen mode
- **F12**: Toggle debug mode (auto-forwards events to AI)
- **Ctrl+H**: Show/hide help
- **Ctrl+N/P**: Next/Previous session (if multiple)
- **ESC**: Exit full screen or unfocus terminal
- **Mouse wheel / PgUp/PgDn**: Scroll through output

## What to Expect

When you run `/ai test-claude`, you should see:
1. The AI Coders tab becomes active
2. A terminal session starts showing:
   - "Claude AI Coder (Test Mode)"
   - The MCP URL (e.g., "MCP URL: http://localhost:7777/mcp")
   - "Ready for input..."
   - A bash prompt

3. Press Enter to focus the terminal and start typing bash commands

## Troubleshooting

- If you see "Command 'claude' not found", the Claude CLI is not installed
- Check the Logs tab for any error messages
- The test-claude provider is a good way to test PTY functionality without Claude

## Available Providers

Currently configured providers:
- `terminal`: Basic bash terminal
- `test-claude`: Test provider that simulates Claude
- `claude`: Real Claude CLI (requires installation)
- `claude-secure`: Claude without dangerous permissions
- `opencode`: OpenCode CLI
- `gemini`: Gemini CLI