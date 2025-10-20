#!/bin/bash

echo "🧪 Testing keyboard navigation with debug logging..."
echo ""

# Create a temporary test directory
TEST_DIR="/tmp/brummer_keyboard_debug"
mkdir -p "$TEST_DIR"

# Create a minimal package.json
cat > "$TEST_DIR/package.json" << 'EOF'
{
  "name": "keyboard-debug-test",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'Running test' && sleep 30",
    "dev": "echo 'Running dev' && sleep 30"
  }
}
EOF

cd "$TEST_DIR"

echo "📍 Test location: $TEST_DIR"
echo ""
echo "🎯 Instructions:"
echo "1. Press number keys 1-5 to test view switching"
echo "2. Press Tab to cycle views"
echo "3. Watch the Logs view (press 2) for debug messages"
echo "4. Press Ctrl+C to quit"
echo ""
echo "Starting brummer with keyboard debugging..."
echo ""

# Run brummer (will show TUI)
/home/beagle/work/brummer/brum

echo ""
echo "Test complete!"