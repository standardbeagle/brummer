#!/bin/bash

echo "=== Testing ViewWithState Blocking Issue ==="
echo ""
echo "This test will:"
echo "1. Clean up old debug log"
echo "2. Run brum in background"
echo "3. Wait 5 seconds"
echo "4. Kill the process"
echo "5. Show debug log to see where it blocked"
echo ""

# Clean up
rm -f /tmp/brum-debug.log
pkill -f './brum' 2>/dev/null

# Run in background
echo "Starting brum..."
./brum > /tmp/brum-output.log 2>&1 &
BRUM_PID=$!

echo "Brum PID: $BRUM_PID"
echo "Waiting 5 seconds..."
sleep 5

# Check if still running
if ps -p $BRUM_PID > /dev/null 2>&1; then
    echo "Process still running - killing it..."
    kill $BRUM_PID 2>/dev/null
    sleep 1
    if ps -p $BRUM_PID > /dev/null 2>&1; then
        echo "Force killing..."
        kill -9 $BRUM_PID 2>/dev/null
    fi
else
    echo "Process already exited"
fi

echo ""
echo "=== Debug Log ==="
if [ -f /tmp/brum-debug.log ]; then
    cat /tmp/brum-debug.log
else
    echo "No debug log found"
fi

echo ""
echo "=== Last 20 lines of output ==="
if [ -f /tmp/brum-output.log ]; then
    tail -20 /tmp/brum-output.log
else
    echo "No output log found"
fi
