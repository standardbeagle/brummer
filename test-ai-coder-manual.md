# Manual AI Coder Testing Instructions

## Changes Made

1. **Fixed duplicate provider registration** - Built-in providers (claude, gemini, terminal) now only register if there's no CLI tool configuration for them in the config file.

2. **Added extensive debug logging** to trace the AI coder session creation flow:
   - Available providers logging
   - Provider configuration details
   - CLI tool command and arguments
   - Session creation success/failure
   - PTY output monitoring

3. **Fixed PTY view dimension updates** - The view now properly sets dimensions before rendering.

## Testing Steps

1. Build the latest version:
   ```bash
   make build
   ```

2. Run brummer in a real terminal:
   ```bash
   ./brum
   ```

3. Switch to AI Coders view (Tab 5 times or use number key `5`)

4. Open command palette with `/` and type:
   ```
   ai test-claude
   ```

5. Switch to Logs view (Tab twice or use number key `2`) to see debug output

## Expected Debug Output

You should see messages like:
- "Available AI providers before CreateCoder: [...]"
- "PTY Manager available, creating session for provider: test-claude"
- "Available providers: X"
- "Provider names: [...]"
- "Provider has CLI tool configured"
- "CLI Tool Command: /bin/bash, Args: [...]"
- "Calling CreateSessionWithEnv: name=test-claude AI Coder, cmd=/bin/bash"
- "Session created successfully: ID=..., Active=true"
- "Started test-claude AI coder session (ID: ...)"
- "Started monitoring PTY output for session ..."

## What Should Happen

1. The AI Coders view should show the terminal with the test-claude session
2. You should see "Claude AI Coder (Test Mode)" and "MCP URL: ..." in the terminal
3. The terminal should be interactive (you can type commands)

## If It Still Shows "Initializing..."

Check the logs for:
1. Any error messages about provider configuration
2. Whether the session was created successfully
3. Whether PTY output monitoring started

The session count should increase from 0 to 1 when the session is created.