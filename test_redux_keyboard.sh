#!/bin/bash

# Test Redux/Elm refactored keyboard navigation
# This script verifies that keyboard input works after the fix

echo "=== Testing Redux/Elm Refactored Keyboard Navigation ==="
echo ""
echo "Instructions:"
echo "1. The TUI will launch"
echo "2. Try pressing keys: 1, 2, 3, Tab, q"
echo "3. Verify:"
echo "   - Number keys (1-7) switch views"
echo "   - Tab cycles through views"
echo "   - 'q' quits the application"
echo "   - Terminal exits cleanly (no corruption)"
echo ""
echo "Expected: ALL keyboard input should work smoothly!"
echo ""
echo "Press Enter to start the test..."
read

# Clean up any old debug log
rm -f /tmp/brum-debug.log

# Run the application
./brum

# Check exit status
EXIT_CODE=$?

echo ""
echo "=== Test Complete ==="
echo "Exit code: $EXIT_CODE"
echo ""

# Show debug log if it exists
if [ -f /tmp/brum-debug.log ]; then
    echo "Debug log (last 50 lines):"
    tail -50 /tmp/brum-debug.log
fi

echo ""
echo "Did keyboard navigation work? (y/n)"
read RESPONSE

if [ "$RESPONSE" = "y" ]; then
    echo "✅ SUCCESS: Keyboard navigation is working!"
    echo "The Redux/Elm refactoring has fixed the issue."
else
    echo "❌ FAILED: Keyboard navigation still broken"
    echo "Further investigation needed."
fi
