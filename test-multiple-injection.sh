#!/bin/bash

# Test script to verify that monitoring script injection happens only once
# and not on AJAX/fetch requests

echo "Testing Brummer monitoring script injection fix..."
echo "================================================="

# Kill any existing brummer processes
echo "Stopping any existing Brummer processes..."
pkill -f "brum" 2>/dev/null || true
sleep 1

# Start a simple test server that returns HTML for different endpoints
cat > /tmp/test-server.js << 'EOF'
const http = require('http');

const server = http.createServer((req, res) => {
  console.log(`${req.method} ${req.url} - Headers:`, JSON.stringify(req.headers, null, 2));
  
  // Main page
  if (req.url === '/' || req.url === '/index.html') {
    res.writeHead(200, {'Content-Type': 'text/html'});
    res.end(`<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Main Page</h1>
    <button id="loadFragment">Load HTML Fragment</button>
    <div id="content"></div>
    
    <script>
    // Test AJAX request that returns HTML
    document.getElementById('loadFragment').addEventListener('click', () => {
        fetch('/fragment', {
            headers: {
                'X-Requested-With': 'XMLHttpRequest'
            }
        })
        .then(r => r.text())
        .then(html => {
            document.getElementById('content').innerHTML = html;
            console.log('Fragment loaded');
        });
    });
    
    // Test modern fetch request
    setTimeout(() => {
        fetch('/api/data')
        .then(r => r.text())
        .then(html => {
            console.log('API data loaded');
        });
    }, 1000);
    </script>
</body>
</html>`);
    
  // HTML fragment endpoint (should NOT get script injection)
  } else if (req.url === '/fragment') {
    res.writeHead(200, {'Content-Type': 'text/html'});
    res.end(`<div class="fragment">
    <h2>This is a fragment</h2>
    <p>This HTML fragment should NOT have the monitoring script injected.</p>
</div>`);
    
  // API endpoint returning HTML (should NOT get script injection)
  } else if (req.url === '/api/data') {
    res.writeHead(200, {'Content-Type': 'text/html'});
    res.end(`<div>API Response: ${new Date().toISOString()}</div>`);
    
  } else {
    res.writeHead(404);
    res.end('Not found');
  }
});

server.listen(3456, () => {
  console.log('Test server running on http://localhost:3456');
});

// Keep server running
process.on('SIGINT', () => {
  console.log('Shutting down test server...');
  process.exit(0);
});
EOF

# Start the test server
echo "Starting test server on port 3456..."
node /tmp/test-server.js &
TEST_SERVER_PID=$!
sleep 2

# Create a test package.json
cat > /tmp/package.json << 'EOF'
{
  "name": "injection-test",
  "scripts": {
    "test-server": "echo 'Test server started'"
  }
}
EOF

# Start Brummer in the test directory with explicit proxy port
echo "Starting Brummer..."
cd /tmp
/home/beagle/work/brummer/brum -p 8888 test-server &
BRUMMER_PID=$!
sleep 3

# Extract the actual proxy port from Brummer's output
PROXY_PORT=8888

# Function to count script injections
count_injections() {
    local url=$1
    local headers=$2
    echo -e "\n--- Testing: $url ---"
    
    if [ -n "$headers" ]; then
        response=$(curl -s -H "$headers" "$url")
    else
        response=$(curl -s "$url")
    fi
    
    # Count occurrences of the monitoring script marker
    count=$(echo "$response" | grep -c "<!-- Brummer Monitoring Script -->" || true)
    
    echo "Script injection count: $count"
    
    # Show if __brummerInitialized check is present
    if echo "$response" | grep -q "__brummerInitialized"; then
        echo "✓ Idempotency check found"
    fi
    
    # Check response size
    size=$(echo "$response" | wc -c)
    echo "Response size: $size bytes"
    
    return $count
}

echo -e "\n\nTesting script injection behavior..."
echo "===================================="

# Test 1: Main page (should have exactly 1 injection)
echo -e "\nTest 1: Main page navigation"
count_injections "http://localhost:${PROXY_PORT}/index.html" ""

# Give time for any background requests
sleep 2

# Test 2: AJAX request with XMLHttpRequest header (should have 0 injections)
echo -e "\nTest 2: AJAX request with X-Requested-With header"
count_injections "http://localhost:${PROXY_PORT}/fragment" "X-Requested-With: XMLHttpRequest"

# Test 3: Fetch request to API endpoint (should have 0 injections based on path pattern)
echo -e "\nTest 3: API endpoint"
count_injections "http://localhost:${PROXY_PORT}/api/data" "Accept: application/json"

# Test 4: Direct fragment load (no special headers - might get injection in current impl)
echo -e "\nTest 4: Direct fragment request"
count_injections "http://localhost:${PROXY_PORT}/fragment" ""

# Clean up
echo -e "\n\nCleaning up..."
kill $TEST_SERVER_PID 2>/dev/null || true
kill $BRUMMER_PID 2>/dev/null || true

echo -e "\nTest complete!"
echo "=============="
echo "Summary:"
echo "- Main page should have exactly 1 script injection"
echo "- AJAX/fetch requests should have 0 script injections"
echo "- The monitoring script should have idempotency protection"