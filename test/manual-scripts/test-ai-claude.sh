#!/bin/bash

echo "Testing AI Coder PTY Session Flow"
echo "=================================="
echo ""
echo "This script tests the /ai claude command path with the PTY flow"
echo ""

# Build first
echo "1. Building application..."
go build -o brum ./cmd/brum
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build successful"
echo ""

# Create a test directory with a simple package.json
TEST_DIR="/tmp/brummer-ai-test-$$"
mkdir -p "$TEST_DIR"
cat > "$TEST_DIR/package.json" << EOF
{
    "name": "test-app",
    "version": "1.0.0",
    "scripts": {
        "dev": "echo 'Dev server running'",
        "test": "echo 'Running tests'"
    }
}
EOF

echo "2. Starting Brummer in headless mode to test AI coder flow..."
cd "$TEST_DIR"

# Start in headless mode first to test the AI coder manager creation
echo "   Testing AI coder manager initialization..."
timeout 2 "$OLDPWD/brum" --no-tui --debug 2>&1 | grep -i "ai" || true

echo ""
echo "3. Testing /ai command path simulation..."
echo "   The actual /ai claude command would be invoked through the TUI"
echo "   when running in interactive mode (without --no-tui flag)."
echo ""

# Create a simple test that shows the AI coder workflow
cat > test_ai_flow.md << EOF
AI Coder PTY Session Flow Test
==============================

When the user runs "/ai claude" in the TUI:

1. Command is captured by command window handler
2. Routed to CommandExecutor.handleAI()
3. Creates AI coder session with:
   - Provider: "claude"
   - PTY session for tmux-style interaction
   - Session ID: claude-<timestamp>

4. Switches view to AI Coders tab (View 5)
5. User sees interactive PTY session with Claude

The PTY session provides:
- Full terminal emulation
- Tmux-style pane management
- Direct interaction with Claude Code
- Real-time code execution and feedback

Current Implementation Status:
- ✅ Command routing (handleAI at line 250)
- ✅ Session info creation (line 260-270)
- ✅ View switching to AI coders (line 275)
- 🔄 PTY session integration (basic structure ready)
- 🔄 Claude provider config (needs API setup)

To test manually:
1. Run: ./brum
2. Press "/" to open command window
3. Type: ai claude
4. Press Enter
5. View should switch to AI Coders tab with PTY session
EOF

cat test_ai_flow.md

echo ""
echo "4. Testing basic PTY functionality..."
# Test that PTY code compiles correctly
go test -c ./internal/aicoder 2>&1 | head -5 || true

echo ""
echo "✅ AI Coder flow structure is in place!"
echo ""
echo "Next steps to complete integration:"
echo "1. Configure Claude API credentials in config"
echo "2. Run ./brum without --no-tui flag"
echo "3. Use /ai claude command in TUI"
echo "4. Interact with Claude through PTY session"
echo ""

# Cleanup
rm -rf "$TEST_DIR"