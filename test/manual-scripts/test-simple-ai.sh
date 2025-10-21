#!/bin/bash

echo "Testing simple AI coder creation..."

# Create a temporary package.json if it doesn't exist
if [ ! -f package.json ]; then
    echo '{"name": "test", "scripts": {"dev": "echo dev"}}' > package.json
    CREATED_PACKAGE=1
fi

# Run brummer and immediately try to create AI coder
./brum << 'EOF' 2>&1 | tee test-ai-output.log
/ai test-claude
q
EOF

# Clean up temporary package.json
if [ "$CREATED_PACKAGE" = "1" ]; then
    rm package.json
fi

echo ""
echo "=== Checking debug output ==="
grep -E "(Available AI providers|Available providers:|Provider names:|CLI Tool Command:|Session created|Error)" test-ai-output.log | head -20