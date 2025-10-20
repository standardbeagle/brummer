#!/bin/bash

echo "=========================================="
echo "🔍 Testing Keyboard Handler Execution Flow"
echo "=========================================="
echo ""
echo "This test will help us understand WHY keyboard handlers"
echo "aren't processing the keys even though keys ARE received."
echo ""
echo "INSTRUCTIONS:"
echo "1. Clear the debug log"
echo "2. Start brummer"
echo "3. Press '1' THREE TIMES (to switch to Processes view)"
echo "4. Press 'q' to quit"
echo "5. Review the debug log"
echo ""
echo "WHAT WE'RE LOOKING FOR:"
echo "- UPDATE: Keyboard message received! 1"
echo "- HANDLER: Processing key '1' with N handlers"
echo "- HANDLER: [0] Checking GlobalNavigationHandler"
echo "- HANDLER: ✅ GlobalNavigationHandler CAN HANDLE this key!"
echo "- GLOBALNAV: Handle called for key: 1"
echo "- GLOBALNAV: Number key 1 → view ViewProcesses"
echo ""
echo "If we see 'CAN HANDLE' but NOT 'Handle called', the problem"
echo "is between HandleKey() and Handle()."
echo ""
echo "If we see 'Handle called' but no view switch, the problem"
echo "is in viewManager.SwitchToView()."
echo ""
echo "Press ENTER to continue..."
read

# Clear debug log
echo "📝 Clearing debug log..."
rm -f /tmp/brum-debug.log

# Create test directory
TEST_DIR="/tmp/brummer_handler_test"
mkdir -p "$TEST_DIR"
cat > "$TEST_DIR/package.json" << 'EOF'
{
  "name": "handler-test",
  "scripts": {
    "test": "echo 'test' && sleep 30"
  }
}
EOF

cd "$TEST_DIR"

echo ""
echo "🚀 Starting brummer..."
echo "Remember: Press '1' three times, then 'q' to quit"
echo ""
sleep 2

# Run brummer
/home/beagle/work/brummer/brum

echo ""
echo "=========================================="
echo "📊 Debug Log Results"
echo "=========================================="
echo ""

if [ -f /tmp/brum-debug.log ]; then
    cat /tmp/brum-debug.log
    echo ""
    echo "=========================================="
    echo "🔍 Analysis"
    echo "=========================================="
    echo ""

    # Count key presses
    KEY_COUNT=$(grep -c "UPDATE: Keyboard message received! 1" /tmp/brum-debug.log || echo "0")
    echo "✓ '1' key presses detected: $KEY_COUNT"

    # Check if handlers were called
    HANDLER_COUNT=$(grep -c "HANDLER: Processing key" /tmp/brum-debug.log || echo "0")
    echo "✓ Handler processing calls: $HANDLER_COUNT"

    # Check if any handler matched
    CAN_HANDLE=$(grep -c "CAN HANDLE" /tmp/brum-debug.log || echo "0")
    echo "✓ Handlers that CAN handle: $CAN_HANDLE"

    # Check if GlobalNav was called
    GLOBALNAV=$(grep -c "GLOBALNAV: Handle called" /tmp/brum-debug.log || echo "0")
    echo "✓ GlobalNav Handle() calls: $GLOBALNAV"

    # Check view switches
    VIEW_SWITCH=$(grep -c "GLOBALNAV: Number key" /tmp/brum-debug.log || echo "0")
    echo "✓ View switch attempts: $VIEW_SWITCH"

    echo ""
    if [ "$KEY_COUNT" -gt 0 ] && [ "$HANDLER_COUNT" -eq 0 ]; then
        echo "⚠️ PROBLEM: Keys received but handlers NOT called"
        echo "   → Issue is in Update() keyboard routing"
    elif [ "$HANDLER_COUNT" -gt 0 ] && [ "$CAN_HANDLE" -eq 0 ]; then
        echo "⚠️ PROBLEM: Handlers called but NONE can handle the key"
        echo "   → Issue is in handler.CanHandle() logic"
    elif [ "$CAN_HANDLE" -gt 0 ] && [ "$GLOBALNAV" -eq 0 ]; then
        echo "⚠️ PROBLEM: Handler CAN handle but Handle() NOT called"
        echo "   → Issue is between CanHandle() and Handle()"
    elif [ "$GLOBALNAV" -gt 0 ] && [ "$VIEW_SWITCH" -eq 0 ]; then
        echo "⚠️ PROBLEM: Handle() called but view switch NOT attempted"
        echo "   → Issue is in GlobalNavigationHandler.Handle() logic"
    elif [ "$VIEW_SWITCH" -gt 0 ]; then
        echo "✅ View switches attempted: $VIEW_SWITCH times"
        echo "   → Handler logic is working!"
        echo "   → Problem might be in viewManager.SwitchToView()"
        echo "   → OR the view IS switching but not visually updating"
    else
        echo "❓ UNCLEAR: Check the full log above for details"
    fi
else
    echo "❌ No debug log found at /tmp/brum-debug.log"
fi

echo ""
echo "=========================================="
