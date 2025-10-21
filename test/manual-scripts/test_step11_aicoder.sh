#!/bin/bash

cd /tmp
mkdir -p test_step11 && cd test_step11
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json

# Clear debug log
rm -f /tmp/brum-debug.log

echo "=========================================="
echo "Testing Progressive Model - Step 11"
echo "AICoderService + Event Subscriptions"
echo "=========================================="
echo ""
echo "This step adds:"
echo "  - AICoderService (creates its own EventBus)"
echo "  - 5 event subscriptions pushing to updateChan"
echo "  - AICodersViewComponent"
echo ""
echo "INSTRUCTIONS:"
echo "1. Press keys: 1 1 1 2 2 2 Tab Tab"
echo "2. Then press q to quit"
echo ""
echo "Starting in 2 seconds..."
sleep 2

# Run brum
/home/beagle/work/brummer/brum

echo ""
echo "=========================================="
echo "Test Results:"
echo "=========================================="
echo ""

# Check if keyboard messages were received
if grep -q "PROGRESSIVE: Keyboard message received" /tmp/brum-debug.log; then
    echo "✅ Keyboard messages RECEIVED!"
    echo ""
    echo "❓ This means AICoderService is NOT the culprit."
    echo "   We need to keep testing other components."
    echo ""
    echo "Keys detected:"
    grep "PROGRESSIVE: Keyboard" /tmp/brum-debug.log | head -10
else
    echo "❌ NO keyboard messages received"
    echo ""
    echo "🎯 FOUND THE CULPRIT: AICoderService breaks keyboard!"
    echo ""
    echo "The problem is:"
    echo "  - AICoderService creates its own EventBus with worker goroutines"
    echo "  - Event subscriptions run in background threads"
    echo "  - These goroutines interfere with stdin/BubbleTea input"
    echo ""
    echo "Solution: Refactor AICoderService to use tea.Cmd pattern"
    echo "          instead of goroutines and EventBus."
    echo ""
    echo "Messages found:"
    grep "PROGRESSIVE:" /tmp/brum-debug.log | head -20
fi
