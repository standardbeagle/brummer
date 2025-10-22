#!/bin/bash
# Test script for AI Coder viewport fix

echo "🧪 Testing AI Coder Viewport Fix"
echo "================================"
echo ""
echo "This fix eliminates duplicate headers and content by using"
echo "viewport-only rendering instead of scrollback accumulation."
echo ""
echo "Testing Steps:"
echo "1. Start brum with AI Coder"
echo "2. Navigate to AI Coders tab (tab 3)"
echo "3. Start a Claude Code session (/ai claude)"
echo "4. Type some input and verify:"
echo "   - No duplicate 'Claude Code' header"
echo "   - No duplicate responses"
echo "   - Typing appears on single line (not multiple)"
echo "   - Full color support (24-bit)"
echo "   - Screen updates work correctly"
echo ""
echo "Starting brum..."
echo ""

# Build first to ensure latest changes
make build

# Run brum
./brum
