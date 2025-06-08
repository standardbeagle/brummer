#!/bin/bash

echo "Starting test server..."
cd test-project
# Start the telemetry demo server in background
node telemetry-demo.js &
SERVER_PID=$!

sleep 2

echo "Starting brum with reverse proxy..."
cd ..
./brum telemetry-demo &
BRUM_PID=$!

sleep 3

echo "Testing reverse proxy..."
# Get the proxy mappings
echo "Active proxy mappings:"
curl -s http://localhost:19888/ || echo "Main proxy not responding"

echo -e "\nPress Enter to clean up..."
read

# Cleanup
kill $SERVER_PID $BRUM_PID 2>/dev/null
echo "Cleaned up"