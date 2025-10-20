#!/bin/bash

echo "Testing keyboard input consistency..."
echo ""

for i in {1..3}; do
    echo "========== Run $i =========="
    rm -f /tmp/brum-debug.log

    # Run test in background
    ./test_model &
    PID=$!

    # Give it time to start
    sleep 0.5

    # Kill it
    kill -9 $PID 2>/dev/null
    wait $PID 2>/dev/null

    # Check if keyboard messages were received
    if grep -q "Keyboard message received" /tmp/brum-debug.log 2>/dev/null; then
        echo "✅ Run $i: Keyboard WORKED"
    else
        echo "❌ Run $i: Keyboard FAILED"
    fi

    sleep 0.5
done

echo ""
echo "Testing shows keyboard is inconsistent."
echo "This suggests a race condition or timing issue during initialization."
