#!/bin/bash

echo "Testing Full Proxy Mode"
echo "======================="

# Clean up
pkill -f "brum" 2>/dev/null || true
pkill -f "python.*http.server" 2>/dev/null || true
sleep 1

cd /home/beagle/work/brummer/test-project

# Start HTTP server
python3 -m http.server 9999 > /dev/null 2>&1 &
HTTP_PID=$!
sleep 1

# Start Brummer in FULL proxy mode on port 8888
echo "Starting Brummer in full proxy mode on port 8888..."
/home/beagle/work/brummer/brum --no-tui --proxy-mode full --proxy-port 8888 > /tmp/full-proxy.log 2>&1 &
BRUM_PID=$!
sleep 3

# Test proxy
echo ""
echo "Testing proxy request..."
RESPONSE=$(curl -s -x http://localhost:8888 http://localhost:9999/multiple-injection-demo.html 2>&1)

if echo "$RESPONSE" | grep -q "Brummer Multiple Injection Test"; then
    echo "✅ Proxy is working - page loaded"
    
    # Check script injection
    SCRIPT_COUNT=$(echo "$RESPONSE" | grep -c "Brummer Monitoring Script" || echo "0")
    echo "Script injections: $SCRIPT_COUNT"
    
    if [ "$SCRIPT_COUNT" = "1" ]; then
        echo "✅ Script injected exactly once"
    else
        echo "❌ Script injected $SCRIPT_COUNT times"
    fi
else
    echo "❌ Failed to load page through proxy"
    echo "Response: $RESPONSE"
fi

# Test XHR duplicate issue
echo ""
echo "Testing XHR duplication..."
for i in {1..3}; do
    curl -s -x http://localhost:8888 -H "X-Requested-With: XMLHttpRequest" http://localhost:9999/fragment.html > /dev/null
done

echo ""
echo "Proxy request events:"
grep "proxy.request" /tmp/full-proxy.log | wc -l

# Cleanup
kill $HTTP_PID 2>/dev/null || true
kill $BRUM_PID 2>/dev/null || true

echo ""
echo "Logs saved to /tmp/full-proxy.log"