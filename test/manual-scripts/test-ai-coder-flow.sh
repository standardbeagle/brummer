#!/bin/bash
set -e

echo "Testing AI Coder flow with logging..."

# Create a simple test script that logs to help debug
cat > test-ai-session.sh << 'EOF'
#!/bin/bash
echo "Starting test AI coder session..."
# Navigate to AI Coders tab (Tab 5 times)
sleep 2
printf '\t\t\t\t\t'
sleep 1
# Run test-claude
echo "/ai test-claude"
sleep 3
# Go to logs to see what happened
printf '\t\t'
sleep 1
# Go back to AI Coders
printf '\t\t\t\t'
sleep 2
# Try to focus terminal
printf '\n'
sleep 1
# Type something
echo "Hello AI!"
sleep 2
# Exit
printf 'q'
EOF

chmod +x test-ai-session.sh

# Run the test
./test-ai-session.sh | ./brum 2>&1 | tee ai-coder-test.log

echo "Test completed. Checking log for session info..."
echo "=== Session Creation ==="
grep -i "Started.*AI coder session" ai-coder-test.log || echo "No session creation found"
echo "=== PTY Output ==="
grep -i "monitoring PTY output" ai-coder-test.log || echo "No PTY monitoring found"
echo "=== Dimensions ==="
grep -i "dimensions:" ai-coder-test.log || echo "No dimension info found"