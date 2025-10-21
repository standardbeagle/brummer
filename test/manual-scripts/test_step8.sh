#!/bin/bash

cd /tmp
mkdir -p test_step8 && cd test_step8
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json

# Clear debug log
rm -f /tmp/brum-debug.log

echo "=========================================="
echo "Testing Progressive Model - Step 8"
echo "InputComponent + NavigationComponent"
echo "=========================================="
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
    echo "Keys detected:"
    grep "PROGRESSIVE: Keyboard" /tmp/brum-debug.log | head -10
else
    echo "❌ NO keyboard messages received"
    echo ""
    echo "Messages found:"
    head -20 /tmp/brum-debug.log
fi
