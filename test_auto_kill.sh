#!/bin/bash

echo "=========================================="
echo "🔍 Auto-Kill Keyboard Handler Test"
echo "=========================================="
echo ""
echo "This script will:"
echo "1. Clear the debug log"
echo "2. Start brummer"
echo "3. Give you 5 seconds to press keys (1, 2, Tab, etc.)"
echo "4. Automatically kill brummer after 5 seconds"
echo "5. Display the debug log"
echo ""
echo "Press ENTER when ready to start..."
read

# Clear debug log
echo "📝 Clearing debug log..."
rm -f /tmp/brum-debug.log

# Create test directory
TEST_DIR="/tmp/brummer_handler_test"
rm -rf "$TEST_DIR"
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
echo "⏰ You have 5 seconds to press keys!"
echo ""
echo "Press: 1, 1, 1, 2, 2, 2, Tab, Tab"
echo ""

# Start brummer in background
/home/beagle/work/brummer/brum &
BRUM_PID=$!

echo "✓ Brummer started (PID: $BRUM_PID)"
echo ""

# Wait 5 seconds
for i in 5 4 3 2 1; do
    echo "⏰ $i seconds remaining..."
    sleep 1
done

echo ""
echo "⚡ Time's up! Killing brummer..."

# Try SIGTERM first
kill $BRUM_PID 2>/dev/null
sleep 0.5

# If still running, use SIGKILL
if ps -p $BRUM_PID > /dev/null 2>&1; then
    echo "⚡ SIGTERM didn't work, using SIGKILL..."
    kill -9 $BRUM_PID 2>/dev/null
fi

# Wait for process to die
sleep 0.5

echo "✓ Brummer terminated"
echo ""
echo "=========================================="
echo "📊 Debug Log Results"
echo "=========================================="
echo ""

if [ -f /tmp/brum-debug.log ]; then
    echo "📄 Full log:"
    echo "----------------------------------------"
    cat /tmp/brum-debug.log
    echo "----------------------------------------"
    echo ""
    echo "🔍 Analysis:"
    echo "----------------------------------------"

    # Count different event types
    INIT_COUNT=$(grep -c "MODEL: NewModel completed" /tmp/brum-debug.log 2>/dev/null || echo "0")
    UPDATE_COUNT=$(grep -c "UPDATE: " /tmp/brum-debug.log 2>/dev/null || echo "0")
    KEY_COUNT=$(grep -c "UPDATE: Keyboard message received" /tmp/brum-debug.log 2>/dev/null || echo "0")
    HANDLER_COUNT=$(grep -c "HANDLER: Processing key" /tmp/brum-debug.log 2>/dev/null || echo "0")
    CAN_HANDLE=$(grep -c "CAN HANDLE" /tmp/brum-debug.log 2>/dev/null || echo "0")
    GLOBALNAV=$(grep -c "GLOBALNAV: Handle called" /tmp/brum-debug.log 2>/dev/null || echo "0")
    VIEW_SWITCH=$(grep -c "GLOBALNAV: Number key.*→ view" /tmp/brum-debug.log 2>/dev/null || echo "0")

    echo "Model initialized: $INIT_COUNT"
    echo "Total Update() calls: $UPDATE_COUNT"
    echo "Keyboard messages received: $KEY_COUNT"
    echo "Handler processing calls: $HANDLER_COUNT"
    echo "Handlers that matched: $CAN_HANDLE"
    echo "GlobalNav.Handle() calls: $GLOBALNAV"
    echo "View switch attempts: $VIEW_SWITCH"
    echo ""

    # Diagnose the issue
    if [ "$KEY_COUNT" -eq 0 ]; then
        echo "❌ NO KEYBOARD MESSAGES RECEIVED"
        echo "   → Either you didn't press keys OR BubbleTea isn't receiving input"
    elif [ "$HANDLER_COUNT" -eq 0 ]; then
        echo "⚠️ PROBLEM FOUND: Keys received but HandleKey() NEVER called"
        echo "   → Issue is in model.go Update() - keyRegistry.HandleKey() not being called"
        echo "   → Check if m.keyRegistry is nil or if there's an early return"
    elif [ "$CAN_HANDLE" -eq 0 ]; then
        echo "⚠️ PROBLEM FOUND: HandleKey() called but NO handler matched"
        echo "   → Issue is in handler CanHandle() logic"
        echo "   → Check GlobalNavigationHandler.CanHandle() implementation"
    elif [ "$GLOBALNAV" -eq 0 ]; then
        echo "⚠️ PROBLEM FOUND: Handler matched but Handle() NOT called"
        echo "   → Issue between CanHandle() returning true and Handle() being called"
        echo "   → This shouldn't be possible - check HandleKey() logic"
    elif [ "$VIEW_SWITCH" -eq 0 ]; then
        echo "⚠️ PROBLEM FOUND: Handle() called but NO view switch attempted"
        echo "   → Issue in GlobalNavigationHandler.Handle() logic"
        echo "   → Check GetViewForNumberKey() or key.Type switch"
    else
        echo "✅ Handler chain is WORKING!"
        echo "   → View switches were attempted: $VIEW_SWITCH times"
        echo "   → Problem must be in viewManager.SwitchToView() or visual rendering"
    fi

    echo ""
    echo "💡 Next Steps:"
    if [ "$HANDLER_COUNT" -eq 0 ]; then
        echo "   → Check model.go lines 968-987 (keyboard routing to keyRegistry)"
    elif [ "$VIEW_SWITCH" -gt 0 ]; then
        echo "   → Check viewManager.SwitchToView() implementation"
        echo "   → Check if ViewSwitchedMsg is being processed"
    fi

else
    echo "❌ No debug log found at /tmp/brum-debug.log"
    echo "   → Brummer may have failed to start"
fi

echo ""
echo "=========================================="
