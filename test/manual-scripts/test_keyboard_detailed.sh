#!/bin/bash

echo "🧪 Testing keyboard handler execution..."
echo ""
echo "Instructions:"
echo "1. The TUI will start"
echo "2. Press '2' to switch to Logs view"
echo "3. Press '1' several times and watch for keyboard logs"
echo "4. Look for these log patterns:"
echo "   - 🎹 Keyboard event"
echo "   - 📍 Context"
echo "   - 🔀 Routing to keyRegistry"
echo "   - 🔍 Processing key"
echo "   - ✅ Handler returned command"
echo "5. Press Ctrl+C to quit"
echo ""
echo "Starting in 3 seconds..."
sleep 3

# Clear debug log
rm -f /tmp/brum-debug.log

# Create minimal test directory
TEST_DIR="/tmp/brummer_keyboard_test"
mkdir -p "$TEST_DIR"
cat > "$TEST_DIR/package.json" << 'EOF'
{
  "name": "keyboard-test",
  "scripts": {
    "test": "echo 'test' && sleep 30"
  }
}
EOF

cd "$TEST_DIR"

# Run brummer
/home/beagle/work/brummer/brum

echo ""
echo "📊 Debug file contents:"
cat /tmp/brum-debug.log
