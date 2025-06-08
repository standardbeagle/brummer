#!/bin/bash

# Brummer Test Script - Tests monitoring script injection fix

echo "==================================="
echo "Brummer Monitoring Script Test"
echo "==================================="

# Kill any existing processes
echo "Cleaning up existing processes..."
pkill -f "brum" 2>/dev/null || true
pkill -f "python.*http.server" 2>/dev/null || true
sleep 1

# Navigate to test project
cd /home/beagle/work/brummer/test-project

# Start a simple HTTP server
echo "Starting test HTTP server on port 9999..."
python3 -m http.server 9999 > /dev/null 2>&1 &
HTTP_PID=$!
sleep 1

# Start Brummer with proxy on port 8888
echo "Starting Brummer proxy on port 8888..."
/home/beagle/work/brummer/brum --no-tui > /tmp/brummer-test.log 2>&1 &
BRUMMER_PID=$!
sleep 2

echo ""
echo "Test server running at: http://localhost:9999"
echo "Proxy server running at: http://localhost:8888"
echo ""

# Function to test requests
test_request() {
    local path=$1
    local headers=$2
    local desc=$3
    
    echo "Testing: $desc"
    echo "Path: $path"
    
    if [ -n "$headers" ]; then
        response=$(curl -s -x http://localhost:8888 -H "$headers" "http://localhost:9999$path" 2>/dev/null || echo "CURL_ERROR")
    else
        response=$(curl -s -x http://localhost:8888 "http://localhost:9999$path" 2>/dev/null || echo "CURL_ERROR")
    fi
    
    if [[ "$response" == "CURL_ERROR" ]]; then
        echo "  ❌ Request failed"
        return 1
    fi
    
    # Count script injections
    script_count=$(echo "$response" | grep -c "Brummer Monitoring Script" || true)
    init_count=$(echo "$response" | grep -c "__brummerInitialized" || true)
    
    echo "  Script injections: $script_count"
    echo "  Has init protection: $([ $init_count -gt 0 ] && echo 'Yes' || echo 'No')"
    
    # Validate result
    if [[ "$path" == "/multiple-injection-demo.html" ]]; then
        # Main page should have exactly 1 injection
        if [ $script_count -eq 1 ]; then
            echo "  ✅ PASS: Main page has exactly 1 injection"
        else
            echo "  ❌ FAIL: Main page has $script_count injections (expected 1)"
        fi
    elif [[ "$path" == "/fragment.html" ]]; then
        # Fragment with AJAX headers should have 0 injections
        if [[ -n "$headers" ]] && [ $script_count -eq 0 ]; then
            echo "  ✅ PASS: AJAX fragment has no injection"
        elif [[ -n "$headers" ]] && [ $script_count -gt 0 ]; then
            echo "  ❌ FAIL: AJAX fragment has $script_count injections (expected 0)"
        fi
    fi
    
    echo ""
}

echo "==================================="
echo "Running Tests"
echo "==================================="
echo ""

# Test 1: Main page (should have 1 injection)
test_request "/multiple-injection-demo.html" "" "Main page navigation"

# Test 2: Fragment with XMLHttpRequest header (should have 0 injections)
test_request "/fragment.html" "X-Requested-With: XMLHttpRequest" "AJAX request (XMLHttpRequest)"

# Test 3: Fragment with Fetch metadata (should have 0 injections)
test_request "/fragment.html" "Sec-Fetch-Mode: cors" "Fetch request (cors mode)"

# Test 4: Fragment with JSON accept (should have 0 injections)
test_request "/fragment.html" "Accept: application/json" "API request (JSON)"

# Test 5: Direct fragment request (no special headers - should inject)
test_request "/fragment.html" "" "Direct fragment navigation"

# Show Brummer logs
echo "==================================="
echo "Brummer Log Excerpt:"
echo "==================================="
tail -20 /tmp/brummer-test.log | grep -E "(inject|Script|request)" || true

# Cleanup
echo ""
echo "Cleaning up..."
kill $HTTP_PID 2>/dev/null || true
kill $BRUMMER_PID 2>/dev/null || true

echo ""
echo "==================================="
echo "Test Summary"
echo "==================================="
echo "The fix prevents multiple script injections by:"
echo "1. Detecting AJAX/fetch requests via headers"
echo "2. Checking if script is already injected"
echo "3. Making the script idempotent with __brummerInitialized check"
echo ""
echo "To test manually:"
echo "1. cd test-project && python3 -m http.server 9999"
echo "2. ./brum --no-tui (in another terminal)"
echo "3. Configure browser proxy to localhost:8888"
echo "4. Visit http://localhost:9999/multiple-injection-demo.html"