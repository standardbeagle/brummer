#!/bin/bash

cd /tmp
mkdir -p test_real_model && cd test_real_model
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json

# Clear debug log
rm -f /tmp/brum-debug.log

echo "=========================================="
echo "Testing REAL Model (should be broken)"
echo "=========================================="
echo ""
echo "INSTRUCTIONS:"
echo "1. Press keys: 1 1 1 2 2 2 Tab Tab"
echo "2. Then press Ctrl+C from another terminal to kill"
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
if grep -q "UPDATE: Keyboard message received" /tmp/brum-debug.log; then
    echo "✅ Keyboard messages RECEIVED!"
    echo ""
    echo "Keys detected:"
    grep "UPDATE: Keyboard" /tmp/brum-debug.log | head -10
else
    echo "❌ NO keyboard messages received (AS EXPECTED - Real Model is broken)"
    echo ""
    echo "Messages found:"
    grep "UPDATE:" /tmp/brum-debug.log | head -20
fi
