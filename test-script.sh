#!/bin/bash

echo "Testing reverse proxy with simple script injection..."

# Kill any existing brum processes
pkill -f brum 2>/dev/null
sleep 1

cd /home/beagle/work/brummer/test-project

# Start brum in background
../brum --no-tui simple-test &
BRUM_PID=$!

# Wait for proxy to start
sleep 3

echo "Checking if reverse proxy is up on port 20888..."
lsof -i :20888 | grep LISTEN

echo -e "\nTesting basic proxy functionality:"
curl -s http://localhost:20888/ | grep -E "(Simple Test|Current time)" | head -5

echo -e "\nChecking if script tag was injected:"
curl -s http://localhost:20888/ | grep -E "(script|__brummer)" | head -5

echo -e "\nFull response (first 1000 chars):"
curl -s http://localhost:20888/ | head -c 1000

# Cleanup
kill $BRUM_PID 2>/dev/null
echo -e "\n\nTest complete."