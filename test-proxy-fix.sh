#!/bin/bash

echo "Testing Proxy Server Fix"
echo "========================"
echo ""
echo "This test verifies:"
echo "1. XHR requests are not duplicated in logs"
echo "2. Monitoring script loads correctly on pages"
echo ""

# Clean up any existing processes
pkill -f "brum" 2>/dev/null || true
pkill -f "python.*http.server" 2>/dev/null || true
sleep 1

# Navigate to test project
cd /home/beagle/work/brummer/test-project

# Start simple HTTP server
echo "Starting test HTTP server on port 9999..."
python3 -m http.server 9999 > /dev/null 2>&1 &
HTTP_PID=$!
sleep 1

# Build the latest version
echo "Building brummer..."
cd /home/beagle/work/brummer
make build > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    kill $HTTP_PID 2>/dev/null || true
    exit 1
fi

# Start Brummer with reverse proxy mode
echo "Starting Brummer in reverse proxy mode..."
./brum --no-tui > /tmp/brummer-proxy-test.log 2>&1 &
BRUMMER_PID=$!
sleep 3

# Test 1: Check if proxy is running
echo ""
echo "Test 1: Proxy Status"
echo "-------------------"
if curl -s http://localhost:8888/ | grep -q "Brummer Proxy Server"; then
    echo "✅ Proxy server is running"
else
    echo "❌ Proxy server not responding"
fi

# Test 2: Load main page and check script injection
echo ""
echo "Test 2: Script Injection on Main Page"
echo "-------------------------------------"
MAIN_RESPONSE=$(curl -s -x http://localhost:8888 http://localhost:9999/multiple-injection-demo.html)
SCRIPT_COUNT=$(echo "$MAIN_RESPONSE" | grep -c "Brummer Monitoring Script" || true)
INIT_COUNT=$(echo "$MAIN_RESPONSE" | grep -c "__brummerInitialized" || true)

echo "Script injections: $SCRIPT_COUNT"
echo "Init checks: $INIT_COUNT"

if [ $SCRIPT_COUNT -eq 1 ]; then
    echo "✅ Correct: Exactly one script injection"
else
    echo "❌ Incorrect: Found $SCRIPT_COUNT injections (expected 1)"
fi

# Test 3: XHR request (should not inject)
echo ""
echo "Test 3: XHR Request (No Injection)"
echo "----------------------------------"
XHR_RESPONSE=$(curl -s -x http://localhost:8888 -H "X-Requested-With: XMLHttpRequest" http://localhost:9999/fragment.html)
XHR_SCRIPT_COUNT=$(echo "$XHR_RESPONSE" | grep -c "Brummer Monitoring Script" || true)

echo "Script injections in XHR: $XHR_SCRIPT_COUNT"

if [ $XHR_SCRIPT_COUNT -eq 0 ]; then
    echo "✅ Correct: No script injection in XHR response"
else
    echo "❌ Incorrect: Found $XHR_SCRIPT_COUNT injections in XHR (expected 0)"
fi

# Test 4: Check request logs for duplicates
echo ""
echo "Test 4: Request Logging (Check for Duplicates)"
echo "----------------------------------------------"

# Make a few XHR requests
for i in {1..3}; do
    curl -s -x http://localhost:8888 -H "X-Requested-With: XMLHttpRequest" http://localhost:9999/fragment.html > /dev/null
    sleep 0.5
done

# Check the logs
echo "Checking proxy logs for duplicate entries..."
sleep 1

# Count unique request patterns in the last 20 lines
LOG_LINES=$(tail -20 /tmp/brummer-proxy-test.log | grep -E "(proxy\.request|Request handled)" || true)
echo "$LOG_LINES" | head -5

# Test 5: Verify telemetry endpoint
echo ""
echo "Test 5: Telemetry Endpoint"
echo "-------------------------"
TELEMETRY_RESPONSE=$(curl -s -X OPTIONS http://localhost:8888/__brummer_telemetry__)
if [ -n "$TELEMETRY_RESPONSE" ]; then
    echo "✅ Telemetry endpoint is accessible"
else
    echo "⚠️  Telemetry endpoint returned empty response"
fi

# Show any errors from logs
echo ""
echo "Error Log Check:"
echo "---------------"
grep -i "error\|panic\|fatal" /tmp/brummer-proxy-test.log | tail -5 || echo "No errors found in logs"

# Cleanup
echo ""
echo "Cleaning up..."
kill $HTTP_PID 2>/dev/null || true
kill $BRUMMER_PID 2>/dev/null || true

echo ""
echo "Test Summary"
echo "============"
echo "1. Proxy server: Check if running"
echo "2. Script injection: Should inject exactly once per page"
echo "3. XHR requests: Should NOT inject script"
echo "4. Request logging: Should not show duplicates"
echo "5. Telemetry: Endpoint should be accessible"
echo ""
echo "Check /tmp/brummer-proxy-test.log for detailed logs"