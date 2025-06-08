#!/bin/bash

# Simple test to verify monitoring script injection behavior

echo "Testing Brummer monitoring script injection fix..."
echo "================================================="

# Create a simple HTML file
cat > /tmp/test.html << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Test Page</h1>
</body>
</html>
EOF

# Test the injection logic directly
echo -e "\nBuilding test program..."
cat > /tmp/test-injection.go << 'EOF'
package main

import (
    "bytes"
    "fmt"
    "io"
    "net/http"
    "strings"
)

// Mock response
type mockResponse struct {
    Body       io.ReadCloser
    Header     http.Header
    StatusCode int
    Request    *http.Request
}

func (m *mockResponse) Cookies() []*http.Cookie { return nil }

func testInjection(htmlContent string, headers map[string]string) {
    // Create mock request
    req, _ := http.NewRequest("GET", "http://example.com/test", nil)
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    
    // Create mock response
    resp := &mockResponse{
        Body:       io.NopCloser(bytes.NewReader([]byte(htmlContent))),
        Header:     make(http.Header),
        StatusCode: 200,
        Request:    req,
    }
    resp.Header.Set("Content-Type", "text/html")
    
    fmt.Printf("\nTest Case:\n")
    fmt.Printf("Headers: %v\n", headers)
    
    // Check if this would be considered a background request
    isBackground := false
    
    // XMLHttpRequest check
    if req.Header.Get("X-Requested-With") == "XMLHttpRequest" {
        isBackground = true
    }
    
    // Fetch metadata check
    fetchMode := req.Header.Get("Sec-Fetch-Mode")
    fetchDest := req.Header.Get("Sec-Fetch-Dest")
    
    if fetchMode != "" && fetchMode != "navigate" {
        isBackground = true
    }
    
    if fetchDest != "" && fetchDest != "document" {
        isBackground = true
    }
    
    // Accept header check
    accept := req.Header.Get("Accept")
    if strings.Contains(accept, "application/json") ||
       strings.Contains(accept, "application/xml") ||
       strings.Contains(accept, "text/xml") {
        isBackground = true
    }
    
    fmt.Printf("Is Background Request: %v\n", isBackground)
    fmt.Printf("Should inject script: %v\n", !isBackground)
    
    // Check for existing injection
    hasScript := strings.Contains(htmlContent, "<!-- Brummer Monitoring Script -->")
    fmt.Printf("Already has script: %v\n", hasScript)
}

func main() {
    html := `<!DOCTYPE html><html><body><h1>Test</h1></body></html>`
    
    fmt.Println("=== Testing injection detection logic ===")
    
    // Test 1: Normal navigation
    fmt.Println("\n1. Normal page navigation:")
    testInjection(html, map[string]string{})
    
    // Test 2: jQuery AJAX
    fmt.Println("\n2. jQuery AJAX request:")
    testInjection(html, map[string]string{
        "X-Requested-With": "XMLHttpRequest",
    })
    
    // Test 3: Modern fetch with metadata
    fmt.Println("\n3. Modern fetch request:")
    testInjection(html, map[string]string{
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Dest": "empty",
    })
    
    // Test 4: Navigation with fetch metadata
    fmt.Println("\n4. Navigation with fetch metadata:")
    testInjection(html, map[string]string{
        "Sec-Fetch-Mode": "navigate",
        "Sec-Fetch-Dest": "document",
    })
    
    // Test 5: JSON API request
    fmt.Println("\n5. JSON API request:")
    testInjection(html, map[string]string{
        "Accept": "application/json",
    })
    
    // Test 6: Already injected
    fmt.Println("\n6. HTML with existing script:")
    htmlWithScript := html + "\n<!-- Brummer Monitoring Script -->"
    testInjection(htmlWithScript, map[string]string{})
}
EOF

cd /tmp && go run test-injection.go

echo -e "\n\n=== Testing actual proxy behavior ===\n"

# Create a test directory with package.json
mkdir -p /tmp/proxy-test
cd /tmp/proxy-test

cat > package.json << 'EOF'
{
  "name": "proxy-injection-test",
  "scripts": {
    "serve": "python3 -m http.server 4567"
  }
}
EOF

# Create test HTML files
mkdir -p public
cat > public/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Main Page</title>
</head>
<body>
    <h1>Main Page</h1>
    <button onclick="loadFragment()">Load Fragment</button>
    <div id="content"></div>
    
    <script>
    function loadFragment() {
        // Test with XMLHttpRequest header
        fetch('/fragment.html', {
            headers: {
                'X-Requested-With': 'XMLHttpRequest'
            }
        })
        .then(r => r.text())
        .then(html => {
            document.getElementById('content').innerHTML = html;
            
            // Count script injections
            const count = (document.documentElement.innerHTML.match(/Brummer Monitoring Script/g) || []).length;
            console.log('Total script injections on page:', count);
            alert('Script injections found: ' + count + ' (should be 1)');
        });
    }
    
    // Also test idempotency
    console.log('__brummerInitialized:', window.__brummerInitialized);
    </script>
</body>
</html>
EOF

cat > public/fragment.html << 'EOF'
<div class="fragment">
    <h2>Fragment Content</h2>
    <p>This should NOT have monitoring script.</p>
</div>
EOF

echo "Test files created. You can now:"
echo "1. Start Python server: cd /tmp/proxy-test && python3 -m http.server 4567"
echo "2. Start Brummer: cd /tmp/proxy-test && /home/beagle/work/brummer/brum serve"
echo "3. Configure browser proxy to http://localhost:8888"
echo "4. Visit http://localhost:4567"
echo "5. Click 'Load Fragment' button"
echo ""
echo "Expected behavior:"
echo "- Main page loads with exactly 1 monitoring script"
echo "- Fragment loads WITHOUT monitoring script"
echo "- Console shows __brummerInitialized = true"