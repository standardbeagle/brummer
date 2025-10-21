#!/bin/bash

echo "=== Testing Web Tab Performance Metrics ==="
echo ""
echo "1. Starting Brummer in TUI mode..."
echo "   (Press Tab 3 times to get to Web tab)"
echo "   (Press Ctrl+C to exit when done)"
echo ""

# Start Brummer in TUI mode (not headless)
./brum -d . &
BRUM_PID=$!

echo "Started Brummer with PID: $BRUM_PID"
sleep 3

echo ""
echo "2. Starting a test web server..."
python3 -m http.server 8080 > /dev/null 2>&1 &
SERVER_PID=$!

echo "Started test server on :8080 with PID: $SERVER_PID"
sleep 2

echo ""
echo "3. Making test requests through the proxy..."
echo "   These should appear in the Web tab with performance metrics"

# Make several requests to generate traffic
curl -s -o /dev/null -w "Request 1: %{http_code} in %{time_total}s\n" -x http://localhost:6277 http://localhost:8080/ &
sleep 0.5

curl -s -o /dev/null -w "Request 2: %{http_code} in %{time_total}s\n" -x http://localhost:6277 http://localhost:8080/nonexistent &
sleep 0.5

curl -s -o /dev/null -w "Request 3: %{http_code} in %{time_total}s\n" -x http://localhost:6277 http://localhost:8080/ &
sleep 1

echo ""
echo "✅ Test requests completed!"
echo ""
echo "📱 In the Brummer TUI:"
echo "   • Press Tab until you see 'HTTP Requests' (Web tab)"
echo "   • Use arrow keys to navigate between requests"
echo "   • Look for performance metrics in the detail panel on the right"
echo "   • Press 'f' to filter requests (try 'error' to see failed requests)"
echo ""
echo "🔍 What you should see:"
echo "   • Request list on the left with selection indicator (►)"
echo "   • Detail panel on the right with:"
echo "     – Method, URL, Status, Duration"
echo "     – Performance metrics (TTFB, DOM Load, etc.)"
echo "     – Authentication info if present"
echo ""
echo "Press Ctrl+C to exit Brummer when finished testing."

# Wait for user to stop
wait $BRUM_PID
kill $SERVER_PID 2>/dev/null