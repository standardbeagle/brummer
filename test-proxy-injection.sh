#!/bin/bash

echo "Testing proxy injection behavior..."
echo "===================================="

# Kill any existing processes
pkill -f "brum" 2>/dev/null || true
pkill -f "python3.*http.server" 2>/dev/null || true
sleep 1

# Start simple HTTP server
cd /tmp
mkdir -p injection-test
cd injection-test

# Create test HTML files
cat > index.html << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Main Page</title></head>
<body>
<h1>Main Page</h1>
<p>This should have the monitoring script injected.</p>
</body>
</html>
EOF

cat > fragment.html << 'EOF'
<div>
<h2>Fragment</h2>
<p>This is an HTML fragment.</p>
</div>
EOF

# Create package.json
cat > package.json << 'EOF'
{
  "name": "injection-test",
  "scripts": {
    "start": "echo 'Starting server...'"
  }
}
EOF

# Start HTTP server in background
python3 -m http.server 9876 > /dev/null 2>&1 &
HTTP_PID=$!
sleep 1

# Start Brummer with standard proxy on port 8888
echo "Starting Brummer proxy..."
/home/beagle/work/brummer/brum --no-tui start > /tmp/brummer.log 2>&1 &
BRUMMER_PID=$!
sleep 2

# Function to test injection
test_injection() {
    local url=$1
    local headers=$2
    local desc=$3
    
    echo -e "\n--- $desc ---"
    echo "URL: $url"
    echo "Headers: $headers"
    
    if [ -n "$headers" ]; then
        response=$(curl -s -x http://localhost:8888 -H "$headers" "$url")
    else
        response=$(curl -s -x http://localhost:8888 "$url")
    fi
    
    # Count monitoring script occurrences
    script_count=$(echo "$response" | grep -c "Brummer Monitoring Script" || true)
    init_check=$(echo "$response" | grep -c "__brummerInitialized" || true)
    
    echo "Script injections: $script_count"
    echo "Has init check: $init_check"
    
    # Show first 200 chars of response
    echo "Response preview: $(echo "$response" | head -c 200)..."
}

echo -e "\n=== Testing Injection Behavior ===\n"

# Test 1: Normal page request (should inject)
test_injection "http://localhost:9876/index.html" "" "Normal page navigation"

# Test 2: AJAX request (should NOT inject)
test_injection "http://localhost:9876/fragment.html" "X-Requested-With: XMLHttpRequest" "AJAX request"

# Test 3: Fetch with metadata (should NOT inject)
test_injection "http://localhost:9876/fragment.html" "Sec-Fetch-Mode: cors" "Fetch request"

# Test 4: JSON API request (should NOT inject)
test_injection "http://localhost:9876/fragment.html" "Accept: application/json" "JSON API request"

# Cleanup
echo -e "\n\nCleaning up..."
kill $HTTP_PID 2>/dev/null || true
kill $BRUMMER_PID 2>/dev/null || true

echo -e "\nTest complete!"
echo "=============="
echo "Summary of expected behavior:"
echo "✓ Normal navigation: Script IS injected (1 occurrence)"
echo "✓ AJAX requests: Script is NOT injected (0 occurrences)"
echo "✓ Fetch requests: Script is NOT injected (0 occurrences)"
echo "✓ API requests: Script is NOT injected (0 occurrences)"
echo "✓ All injected scripts have __brummerInitialized check"