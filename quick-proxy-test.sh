#!/bin/bash

echo "Quick Proxy Fix Test"
echo "===================="

# Clean up
pkill -f "brum" 2>/dev/null || true
pkill -f "python.*http.server" 2>/dev/null || true
sleep 1

# Go to test project directory
cd /home/beagle/work/brummer/test-project

# Start HTTP server
echo "Starting HTTP server..."
python3 -m http.server 9999 > /dev/null 2>&1 &
HTTP_PID=$!
sleep 1

# Start Brummer
echo "Starting Brummer..."
/home/beagle/work/brummer/brum --no-tui > /tmp/quick-test.log 2>&1 &
BRUM_PID=$!
sleep 3

# Test 1: Simple request
echo ""
echo "Test 1: Basic proxy request"
curl -s -x http://localhost:8888 http://localhost:9999/ | head -n 5

# Test 2: Check logs for duplicates
echo ""
echo "Test 2: Making 3 XHR requests..."
for i in {1..3}; do
    curl -s -x http://localhost:8888 -H "X-Requested-With: XMLHttpRequest" http://localhost:9999/fragment.html > /dev/null
    echo "  Request $i sent"
    sleep 0.5
done

echo ""
echo "Checking logs for proxy.request events:"
grep "proxy.request" /tmp/quick-test.log | tail -10

# Cleanup
kill $HTTP_PID 2>/dev/null || true
kill $BRUM_PID 2>/dev/null || true

echo ""
echo "Done! Check /tmp/quick-test.log for full logs"