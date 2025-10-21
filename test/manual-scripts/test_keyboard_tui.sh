#!/bin/bash

echo "🧪 Testing keyboard input processing in TUI mode..."
echo "This will start brummer with a dummy package.json for testing"
echo ""

# Create a temporary test directory
TEST_DIR="/tmp/brummer_keyboard_test"
mkdir -p "$TEST_DIR"

# Create a minimal package.json for testing
cat > "$TEST_DIR/package.json" << 'EOF'
{
  "name": "keyboard-test",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'Test script running...' && sleep 1"
  }
}
EOF

cd "$TEST_DIR"

echo "📍 Created test environment at: $TEST_DIR"
echo ""
echo "🎯 Test instructions:"
echo "1. Brummer will start in TUI mode"
echo "2. Try pressing number keys 1-5 to switch views"
echo "3. Try pressing Tab to cycle views"
echo "4. Try pressing '/' to open command palette"
echo "5. Try pressing Ctrl+C to quit"
echo ""
echo "📊 Expected behavior:"
echo "- View switching should work immediately"
echo "- Command palette should open when '/' is pressed"
echo "- No 'SERVICE_NOT_ACTIVE' errors during operation"
echo ""

# Start brummer with timeout
echo "🚀 Starting brummer (timeout 30s)..."
timeout 30s /home/beagle/work/brummer/brum --debug 2>&1 | tee "$TEST_DIR/keyboard_test.log"

# Check results
echo ""
echo "📋 Test results:"
if grep -q "⌨️  DIAGNOSTIC: Received keyboard message" "$TEST_DIR/keyboard_test.log"; then
    echo "✅ Keyboard messages are being received by BubbleTea"
else
    echo "❌ Keyboard messages are NOT being received"
fi

if grep -q "🔄 Routing keyboard message to InputComponent" "$TEST_DIR/keyboard_test.log"; then
    echo "✅ Keyboard messages are being routed to InputComponent"
else
    echo "❌ Keyboard messages are NOT being routed to InputComponent"
fi

if grep -q "🎯 DIAGNOSTIC: Global keyboard handler executed" "$TEST_DIR/keyboard_test.log"; then
    echo "✅ Global keyboard handler is being executed"
else
    echo "❌ Global keyboard handler is NOT being executed"
fi

if grep -q "SERVICE_NOT_ACTIVE" "$TEST_DIR/keyboard_test.log"; then
    echo "❌ SERVICE_NOT_ACTIVE errors detected"
else
    echo "✅ No SERVICE_NOT_ACTIVE errors"
fi

echo ""
echo "📄 Full log saved to: $TEST_DIR/keyboard_test.log"
echo "🔍 Check the log file for detailed diagnostic information"

# Cleanup
echo ""
echo "🧹 Cleaning up test environment..."
rm -rf "$TEST_DIR"