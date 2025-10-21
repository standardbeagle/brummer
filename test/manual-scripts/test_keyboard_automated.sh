#!/bin/bash

echo "🧪 Automated keyboard test with debug logging..."
echo ""

# Create a temporary test directory
TEST_DIR="/tmp/brummer_keyboard_auto"
mkdir -p "$TEST_DIR"

# Create a minimal package.json
cat > "$TEST_DIR/package.json" << 'EOF'
{
  "name": "keyboard-auto-test",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'Test running' && sleep 1"
  }
}
EOF

cd "$TEST_DIR"

echo "📍 Test location: $TEST_DIR"
echo ""

# Create an expect script to send keyboard input
cat > "$TEST_DIR/test_input.exp" << 'EOF'
#!/usr/bin/expect -f

set timeout 10
spawn /home/beagle/work/brummer/brum

# Wait for TUI to load
sleep 1

# Test number keys
send "1"
sleep 0.5
send "2"
sleep 0.5
send "3"
sleep 0.5

# Test Tab key
send "\t"
sleep 0.5

# Test slash command
send "/"
sleep 0.5
send "\033"  ;# ESC to close

# Quit
send "\003"  ;# Ctrl+C

expect eof
EOF

chmod +x "$TEST_DIR/test_input.exp"

# Check if expect is available
if command -v expect &> /dev/null; then
    echo "Running automated test with expect..."
    "$TEST_DIR/test_input.exp" 2>&1 | tee "$TEST_DIR/output.log"

    echo ""
    echo "📊 Analyzing results..."
    echo ""

    # Check for keyboard events
    if grep -q "🎹 Keyboard event:" "$TEST_DIR/output.log"; then
        echo "✅ Keyboard events are being received"
        grep "🎹 Keyboard event:" "$TEST_DIR/output.log" | head -5
    else
        echo "❌ No keyboard events logged"
    fi

    echo ""

    # Check for handler processing
    if grep -q "🔍 Processing key" "$TEST_DIR/output.log"; then
        echo "✅ Key handler registry is processing keys"
        grep "🔍 Processing key" "$TEST_DIR/output.log" | head -5
    else
        echo "❌ Key handler registry not processing"
    fi

    echo ""

    # Check for navigation handling
    if grep -q "🧭 Handling key:" "$TEST_DIR/output.log"; then
        echo "✅ Navigation handler is active"
        grep "🧭 Handling key:" "$TEST_DIR/output.log" | head -5
    else
        echo "❌ Navigation handler not active"
    fi

    echo ""
    echo "Full log saved to: $TEST_DIR/output.log"
else
    echo "⚠️ 'expect' not installed. Installing it would allow automated testing."
    echo "For now, please run ./test_keyboard_debug.sh for manual testing."
fi

# Cleanup
echo ""
echo "🧹 Cleaning up..."
rm -rf "$TEST_DIR"