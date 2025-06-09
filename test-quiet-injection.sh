#!/bin/bash

# Test script to verify quiet injection mode

echo "Testing Brummer proxy with quiet injection mode..."
echo "1. Build the project"
make build

echo "2. Start test web server"
cd test-project
python3 -m http.server 3000 &
SERVER_PID=$!
cd ..

echo "3. Start Brummer with proxy"
./brum --proxy-port 8888 --no-tui &
BRUM_PID=$!

echo "4. Wait for services to start"
sleep 3

echo "5. Test proxy with curl"
echo "   Making request through proxy..."
curl -x http://localhost:8888 http://localhost:3000/fragment.html -o /tmp/test-response.html 2>/dev/null

echo "6. Check injection"
if grep -q "Brummer: Console monitoring active" /tmp/test-response.html; then
    echo "   ✓ Injection script found"
else
    echo "   ✗ Injection script NOT found"
fi

echo "7. Cleanup"
kill $SERVER_PID 2>/dev/null
kill $BRUM_PID 2>/dev/null

echo "Test complete."