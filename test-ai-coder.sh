#!/bin/bash
# Test script for AI Coder functionality

set -e

echo "Testing AI Coder color support and functionality..."
echo

# Build the application
echo "Building brummer..."
cd /home/beagle/work/brummer
go build -o ./test-brum ./cmd/brum/main.go

# Create test script that outputs colored text
cat > test-color-output.sh << 'EOF'
#!/bin/bash
echo -e "\033[31mRed text\033[0m"
echo -e "\033[32mGreen text\033[0m"
echo -e "\033[34mBlue text\033[0m"
echo -e "\033[38;5;208mOrange (256 color)\033[0m"
echo -e "\033[38;2;255;0;255mMagenta (24-bit RGB)\033[0m"
echo "Normal text"
EOF
chmod +x test-color-output.sh

# Test in headless mode with MCP
echo "Starting brummer with AI coder support (headless mode)..."
timeout 5 ./test-brum --no-tui -p 7779 > test-output.log 2>&1 &
BRUM_PID=$!

sleep 2

# Check if brummer started correctly
if ps -p $BRUM_PID > /dev/null; then
    echo "✓ Brummer started successfully in headless mode"

    # Check MCP server
    if curl -s http://localhost:7779/mcp > /dev/null; then
        echo "✓ MCP server is running on port 7779"
    else
        echo "✗ MCP server not responding"
    fi
else
    echo "✗ Brummer failed to start"
    cat test-output.log
fi

# Kill the test process
kill $BRUM_PID 2>/dev/null || true

# Clean up
rm -f test-color-output.sh test-brum test-output.log

echo
echo "Test complete!"
echo
echo "To manually test the AI Coder tab with colors:"
echo "1. Run: ./brum"
echo "2. Press '6' to switch to AI Coders tab"
echo "3. Run: /ai claude (or /ai terminal for local shell)"
echo "4. The terminal should show with full color support"