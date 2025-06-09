#!/bin/bash

echo "Verifying Proxy Fix"
echo "==================="
echo ""

# Clean up
pkill -f "brum" 2>/dev/null || true
pkill -f "python.*http.server" 2>/dev/null || true
sleep 1

# Go to test project
cd /home/beagle/work/brummer/test-project

# Start HTTP server
echo "Starting HTTP server on port 9999..."
python3 -m http.server 9999 > /dev/null 2>&1 &
HTTP_PID=$!
sleep 1

# Start Brummer with correct proxy port
echo "Starting Brummer (proxy on port 19888)..."
/home/beagle/work/brummer/brum --no-tui > /tmp/verify-fix.log 2>&1 &
BRUM_PID=$!
sleep 3

# Test 1: Check proxy is running
echo ""
echo "Test 1: Proxy Status"
echo "-------------------"
if curl -s http://localhost:19888/ | grep -q "Brummer Proxy Server"; then
    echo "✅ Proxy server is running on port 19888"
else
    echo "❌ Proxy server not responding"
fi

# Test 2: Make multiple XHR requests and check for duplicates
echo ""
echo "Test 2: XHR Request Duplication Test"
echo "------------------------------------"
echo "Making 5 XHR requests..."

for i in {1..5}; do
    curl -s -x http://localhost:19888 -H "X-Requested-With: XMLHttpRequest" http://localhost:9999/fragment.html > /dev/null
    echo -n "."
    sleep 0.2
done
echo " Done"

# Count proxy.request events
echo ""
echo "Counting proxy.request events in logs:"
REQUEST_COUNT=$(grep -c "proxy.request" /tmp/verify-fix.log || echo "0")
echo "Total proxy.request events: $REQUEST_COUNT"

if [ $REQUEST_COUNT -eq 5 ]; then
    echo "✅ PASS: Correct number of requests logged (no duplicates)"
elif [ $REQUEST_COUNT -gt 5 ]; then
    echo "❌ FAIL: Too many requests logged ($REQUEST_COUNT > 5) - duplicates detected!"
else
    echo "⚠️  WARNING: Too few requests logged ($REQUEST_COUNT < 5)"
fi

# Test 3: Check script injection
echo ""
echo "Test 3: Script Injection Test"
echo "-----------------------------"
MAIN_PAGE=$(curl -s -x http://localhost:19888 http://localhost:9999/multiple-injection-demo.html)
SCRIPT_COUNT=$(echo "$MAIN_PAGE" | grep -c "Brummer Monitoring Script" || echo "0")

echo "Script injections in main page: $SCRIPT_COUNT"
if [ $SCRIPT_COUNT -eq 1 ]; then
    echo "✅ PASS: Script injected exactly once"
else
    echo "❌ FAIL: Script injected $SCRIPT_COUNT times (expected 1)"
fi

# Test 4: Check XHR has no injection
XHR_PAGE=$(curl -s -x http://localhost:19888 -H "X-Requested-With: XMLHttpRequest" http://localhost:9999/fragment.html)
XHR_SCRIPT=$(echo "$XHR_PAGE" | grep -c "Brummer Monitoring Script" || echo "0")

echo ""
echo "Script injections in XHR response: $XHR_SCRIPT"
if [ $XHR_SCRIPT -eq 0 ]; then
    echo "✅ PASS: No script injection in XHR response"
else
    echo "❌ FAIL: Script injected in XHR response"
fi

# Show some log entries
echo ""
echo "Sample log entries:"
echo "------------------"
grep "proxy.request" /tmp/verify-fix.log | tail -5

# Cleanup
kill $HTTP_PID 2>/dev/null || true
kill $BRUM_PID 2>/dev/null || true

echo ""
echo "Test complete! Full logs at /tmp/verify-fix.log"