#!/bin/bash

# Test script to check URL stability
echo "Testing URL list stability with multiple processes..."

# Start brummer in background
./brum &
BRUM_PID=$!

# Give it time to start
sleep 2

# Start multiple processes that output URLs
cat > test-server1.js << 'EOF'
setInterval(() => {
    console.log("Server 1 running at http://localhost:3001");
}, 1000);
EOF

cat > test-server2.js << 'EOF'
setInterval(() => {
    console.log("Server 2 running at http://localhost:3002");
}, 1500);
EOF

cat > test-server3.js << 'EOF'
setInterval(() => {
    console.log("Server 3 running at http://localhost:3003");
}, 2000);
EOF

# Run the test servers
node test-server1.js &
SERVER1_PID=$!

node test-server2.js &
SERVER2_PID=$!

node test-server3.js &
SERVER3_PID=$!

echo "Started test servers with PIDs: $SERVER1_PID, $SERVER2_PID, $SERVER3_PID"
echo "Monitor the URLs tab in brummer to check if the list is stable"
echo "Press Enter to stop the test..."
read

# Cleanup
kill $SERVER1_PID $SERVER2_PID $SERVER3_PID 2>/dev/null
kill $BRUM_PID 2>/dev/null
rm -f test-server*.js

echo "Test completed"