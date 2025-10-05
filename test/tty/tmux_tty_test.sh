#!/bin/bash

# Tmux TTY Test Script
# Uses tmux to create a real TTY environment for testing

set -e

echo "🖥️  TMUX TTY TEST"
echo "==============="

# Configuration
BINARY_PATH="${BRUMMER_BINARY:-./brum}"
TEST_DIR="${1:-$(pwd)}"
SESSION_NAME="brum_tty_test_$$"

echo "📋 Binary: $BINARY_PATH"
echo "📁 Test directory: $TEST_DIR"
echo "🏷️  Session name: $SESSION_NAME"

# Check dependencies
if ! command -v tmux >/dev/null 2>&1; then
    echo "❌ tmux not found. Install tmux first:"
    echo "   sudo apt-get install tmux  # Ubuntu/Debian"
    echo "   brew install tmux           # macOS"
    exit 1
fi

if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ Binary not found: $BINARY_PATH"
    echo "Run 'make build' first or set BRUMMER_BINARY env var"
    exit 1
fi

echo "✅ Dependencies checked"

# Function to cleanup tmux session
cleanup() {
    echo ""
    echo "🧹 Cleaning up tmux session..."
    if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        tmux kill-session -t "$SESSION_NAME"
        echo "✅ Tmux session killed"
    fi
}

# Set up cleanup trap
trap cleanup EXIT INT TERM

# Create new tmux session
echo "🚀 Creating tmux session: $SESSION_NAME"
tmux new-session -d -s "$SESSION_NAME"

# Start the application in tmux
echo "📱 Starting Brummer in tmux..."
tmux send-keys -t "$SESSION_NAME" "cd '$TEST_DIR'" Enter
tmux send-keys -t "$SESSION_NAME" "$BINARY_PATH -d '$TEST_DIR' --debug" Enter

# Wait for startup
echo "⏳ Waiting for app to start..."
sleep 5

# Test 1: Check if session is still alive
echo ""
echo "🔍 Test 1: Checking if app is running..."
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    echo "✅ Tmux session is alive"
else
    echo "❌ Tmux session died"
    exit 1
fi

# Test 2: Send keystrokes for navigation
echo ""
echo "🔄 Test 2: Testing navigation..."

# Send tab navigation keys
tmux send-keys -t "$SESSION_NAME" "2"  # Processes tab
sleep 1

tmux send-keys -t "$SESSION_NAME" "6"  # AI Coders tab
sleep 1

tmux send-keys -t "$SESSION_NAME" "1"  # Logs tab
sleep 1

echo "✅ Tab navigation keys sent"

# Test 3: Send slash command
echo ""
echo "💬 Test 3: Testing slash command..."

tmux send-keys -t "$SESSION_NAME" "/"  # Open command
sleep 1

tmux send-keys -t "$SESSION_NAME" "help"  # Type help
sleep 1

tmux send-keys -t "$SESSION_NAME" Enter  # Execute
sleep 2

echo "✅ Help command sent"

# Test 4: Send AI command
echo ""
echo "🤖 Test 4: Testing AI command..."

tmux send-keys -t "$SESSION_NAME" "/"  # Open command
sleep 1

tmux send-keys -t "$SESSION_NAME" "ai claude test tmux script"  # Type AI command
sleep 1

tmux send-keys -t "$SESSION_NAME" Enter  # Execute
sleep 3

echo "✅ AI command sent"

# Test 5: Capture screen content
echo ""
echo "📸 Test 5: Capturing screen content..."

# Capture tmux buffer
CAPTURE_FILE="/tmp/tmux_capture_$(date +%s).txt"
tmux capture-pane -t "$SESSION_NAME" -S "$CAPTURE_FILE"

if [ -f "$CAPTURE_FILE" ]; then
    echo "✅ Screen captured to: $CAPTURE_FILE"
    echo "📋 First 10 lines of captured content:"
    head -10 "$CAPTURE_FILE" | cat -n
else
    echo "❌ Failed to capture screen"
fi

# Test 6: Let it run a bit more
echo ""
echo "⏳ Test 6: Letting app run for observation..."
sleep 5

# Test 7: Try to quit
echo ""
echo "🚪 Test 7: Attempting to quit..."

tmux send-keys -t "$SESSION_NAME" "q"  # Try quit
sleep 2

tmux send-keys -t "$SESSION_NAME" C-c  # Try Ctrl+C
sleep 2

# Final check
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    echo "⚠️  App still running, force killing session"
    # The cleanup trap will handle this
else
    echo "✅ App appears to have exited cleanly"
fi

echo ""
echo "🎯 TMUX TTY TEST COMPLETED"
echo "==========================="
echo "📄 Screen capture: $CAPTURE_FILE"
echo ""
echo "💡 This demonstrates that TTY testing is possible with tmux"
echo "💡 For automated testing, consider:"
echo "   1. Extending this tmux script with more assertions"
echo "   2. Using tmux capture-pane for screen validation"
echo "   3. Combining with expect for precise control"