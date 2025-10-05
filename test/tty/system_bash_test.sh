#!/bin/bash

# System Bash TTY Test Script
# This runs in a real bash shell with /dev/tty access

set -e

echo "🖥️  SYSTEM BASH TTY TEST"
echo "======================"

# Configuration
BINARY_PATH="${BRUMMER_BINARY:-./brum}"
TEST_DIR="${1:-$(pwd)}"
TIMEOUT="${2:-30}"

echo "📋 Binary: $BINARY_PATH"
echo "📁 Test directory: $TEST_DIR"
echo "⏱️  Timeout: ${TIMEOUT}s"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ Binary not found: $BINARY_PATH"
    echo "Run 'make build' first or set BRUMMER_BINARY env var"
    exit 1
fi

# Check if we have TTY
if [ ! -t 0 ]; then
    echo "❌ Not running in a TTY environment"
    echo "This script must be run from a real terminal"
    exit 1
fi

echo "✅ TTY environment detected"
echo "🎯 Starting system bash test..."

# Create a temporary log file
LOG_FILE="/tmp/brum_tty_test_$(date +%s).log"
echo "📝 Logging to: $LOG_FILE"

# Function to cleanup
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    if [ ! -z "$APP_PID" ]; then
        kill $APP_PID 2>/dev/null || true
        wait $APP_PID 2>/dev/null || true
    fi
    echo "✅ Cleanup complete"
}

# Set up cleanup trap
trap cleanup EXIT INT TERM

# Start the application in background
echo "🚀 Starting Brummer in background..."
$BINARY_PATH -d "$TEST_DIR" --debug > "$LOG_FILE" 2>&1 &
APP_PID=$!

echo "📱 App started with PID: $APP_PID"

# Wait a moment for startup
sleep 3

# Function to send keystrokes to the app
send_keystroke() {
    local key="$1"
    echo "⌨️  Sending: $key"
    # Use tput to send keys to the foreground process
    # This is tricky - we need to bring the app to foreground
    # Alternative: use expect or similar tool
    echo "$key" > /tmp/keystroke_buffer
}

# Function to check if app is still running
check_app_running() {
    if kill -0 $APP_PID 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Function to capture screen content
capture_screen() {
    # This is challenging without proper TTY control
    # We'll check the log file instead
    echo "📋 Recent log output:"
    tail -20 "$LOG_FILE"
}

# Test sequence
echo ""
echo "🧪 TEST SEQUENCE"
echo "==============="

# Test 1: Wait for TUI to initialize
echo "⏳ Test 1: Waiting for TUI initialization..."
sleep 5

if check_app_running; then
    echo "✅ App is running"
else
    echo "❌ App crashed or exited"
    echo "📋 Last log output:"
    tail -20 "$LOG_FILE"
    exit 1
fi

# Test 2: Check for TUI elements in logs
echo ""
echo "🔍 Test 2: Checking for TUI elements in logs..."
if grep -q "Brummer - Development Buddy" "$LOG_FILE"; then
    echo "✅ TUI header detected in logs"
else
    echo "❌ TUI header not found in logs"
fi

if grep -q "Started.*server" "$LOG_FILE"; then
    echo "✅ Server startup detected"
else
    echo "⚠️  No server startup found"
fi

# Test 3: Try to send AI command
echo ""
echo "🤖 Test 3: Attempting AI command test..."
echo "⚠️  Note: Limited input control in this script"
echo "   For full testing, use manual testing or expect/tmux"

# Test 4: Check final state
echo ""
echo "📊 Test 4: Final state analysis"
capture_screen

# Test 5: Let app run for a bit more
echo ""
echo "⏳ Test 5: Letting app run for observation..."
sleep 5

if check_app_running; then
    echo "✅ App still running after timeout"
else
    echo "ℹ️  App stopped during test"
fi

echo ""
echo "🎯 SYSTEM BASH TEST COMPLETED"
echo "==========================="
echo "📄 Full log available at: $LOG_FILE"
echo ""
echo "💡 For full interactive testing, run:"
echo "   $BINARY_PATH -d $TEST_DIR"
echo ""
echo "💡 For automated testing, consider:"
echo "   1. Using expect/tmux for proper TTY control"
echo "   2. Testing with --no-tui flag"
echo "   3. Testing slash commands via API/MCP interface"