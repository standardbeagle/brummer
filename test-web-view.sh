#!/bin/bash

echo "Testing new web view with filtering and telemetry..."

# Kill any existing brum processes
pkill -f brum 2>/dev/null
sleep 1

cd /home/beagle/work/brummer/test-project

# Start brum in background
../brum simple-test &
BRUM_PID=$!

# Wait for proxy to start
sleep 3

echo "Making some test requests..."

# Make a page request (will have telemetry)
curl -s http://localhost:20888/ > /dev/null

# Make an API-like request
curl -s -X POST http://localhost:20888/api/test -d '{"test": "data"}' > /dev/null

# Make an image request
curl -s http://localhost:20888/favicon.ico > /dev/null

echo "Web view should now show:"
echo "- Filtering options (all/pages/api/images/other)"
echo "- Split layout with request list and detail panel"  
echo "- Telemetry data for the page request"
echo "- Navigation with f/↑/↓/Enter keys"

echo ""
echo "Press Ctrl+C to stop or switch to brum TUI to see the web view (press '5')"

# Wait for user to interrupt
wait $BRUM_PID