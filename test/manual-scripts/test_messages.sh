#!/bin/bash

echo "🧪 Testing message flow in TUI..."
echo ""

# Create test directory
TEST_DIR="/tmp/brummer_message_test"
mkdir -p "$TEST_DIR"

# Create minimal package.json
cat > "$TEST_DIR/package.json" << 'EOF'
{
  "name": "message-test",
  "scripts": {
    "test": "echo 'test' && sleep 30"
  }
}
EOF

cd "$TEST_DIR"

echo "📍 Starting brummer to observe message flow..."
echo "1. Watch the Logs view (press 2) to see messages"
echo "2. Try pressing ANY key to see if keyboard events arrive"
echo "3. Press Ctrl+C to quit"
echo ""

# Run brummer
timeout 30s /home/beagle/work/brummer/brum

echo "Test complete!"