#!/bin/bash

echo "Testing Brummer Component System Debug Mode"
echo "==========================================="
echo ""

# Kill any existing brummer processes
pkill -f "brum" 2>/dev/null
sleep 1

# Create a simple test directory
rm -rf test-component-dir
mkdir -p test-component-dir
cd test-component-dir

# Create a simple package.json
cat > package.json <<'EOF'
{
  "name": "component-test",
  "version": "1.0.0",
  "scripts": {
    "dev": "echo 'Starting dev server...' && sleep 100",
    "test": "echo 'Running tests...' && sleep 5 && echo 'Tests passed!'",
    "build": "echo 'Building project...' && sleep 3 && echo 'Build complete!'"
  }
}
EOF

# Run with component system enabled
echo "Starting Brummer with component system..."
echo "Press Ctrl+C to exit"
echo ""
echo "Environment: BRUMMER_USE_COMPONENTS=true"
echo "Command: ../brum --no-mcp"
echo ""

# Run brummer with component system
BRUMMER_USE_COMPONENTS=true ../brum --no-mcp