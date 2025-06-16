#!/bin/bash

echo "🧪 Brummer Enhanced Telemetry Test"
echo "=================================="
echo ""
echo "This test will demonstrate the enhanced telemetry features:"
echo "- Advanced network interception (Fetch & XHR)"
echo "- DOM mutation monitoring"
echo "- Storage event tracking"
echo "- Enhanced performance metrics"
echo "- Custom metrics API"
echo "- Improved interaction tracking"
echo "- Scroll tracking with debouncing"
echo ""
echo "Starting test server on port 3333..."
echo ""

# Start the test server in background
node test-server.js &
SERVER_PID=$!

echo ""
echo "Test server started (PID: $SERVER_PID)"
echo ""
echo "Now you can:"
echo "1. In a new terminal, run: ./brum -d . --settings"
echo "2. Press 's' to see available scripts, or 'p' to see processes"
echo "3. Look for the proxy URL in the output (usually http://localhost:8888/proxy/...)"
echo "4. Open that URL in your browser"
echo "5. Interact with the test page and observe telemetry in Brummer"
echo ""
echo "The test page includes:"
echo "- Buttons to test network requests"
echo "- DOM manipulation controls"
echo "- Storage testing"
echo "- Performance marking tools"
echo "- Forms and input fields"
echo "- Scroll area"
echo ""
echo "Press Ctrl+C to stop the test server"
echo ""

# Wait for interrupt
wait $SERVER_PID