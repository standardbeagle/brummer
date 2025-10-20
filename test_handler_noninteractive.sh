#!/bin/bash

echo "=========================================="
echo "🔍 Non-Interactive Keyboard Handler Test"
echo "=========================================="
echo ""

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

echo "🚀 Starting brummer in background with simulated input..."
echo ""

# This approach: run brummer with timeout and send it keystrokes via expect/script
# But since we don't have expect, let's use a simpler approach:
# Run with timeout and let it capture what it can

# Give brummer 3 seconds to start, then kill it
timeout 3s /home/beagle/work/brummer/brum 2>&1 &
BRUM_PID=$!

# Wait a moment for brum to initialize
sleep 0.5

# Check if it's actually running
if ps -p $BRUM_PID > /dev/null 2>&1; then
    echo "✓ Brummer started (PID: $BRUM_PID)"
    echo "✓ Waiting 2.5 seconds then terminating..."
    wait $BRUM_PID 2>/dev/null
else
    echo "❌ Brummer failed to start"
fi

echo ""
echo "=========================================="
echo "📊 Debug Log Results"
echo "=========================================="
echo ""

if [ -f /tmp/bum-debug.log ]; then
    cat /tmp/brum-debug.log
    echo ""
    echo "=========================================="
    echo "🔍 Key Findings"
    echo "=========================================="
    echo ""

    # Check initialization
    if grep -q "NewModel completed" /tmp/brum-debug.log; then
        echo "✓ Model initialization completed"
    fi

    if grep -q "tea.NewProgram" /tmp/brum-debug.log; then
        echo "✓ BubbleTea program created"
    fi

    if grep -q "Calling p.Run()" /tmp/brum-debug.log; then
        echo "✓ BubbleTea Run() called"
    fi

    # Check for Update calls
    UPDATE_COUNT=$(grep -c "UPDATE: Message type" /tmp/brum-debug.log 2>/dev/null || echo "0")
    echo "✓ Update() called $UPDATE_COUNT times"

    # Most importantly: check if keyboard handler registry was called
    if grep -q "HANDLER: Processing key" /tmp/brum-debug.log; then
        echo "✓ Keyboard handler registry WAS called"

        # Check what happened
        if grep -q "CAN HANDLE" /tmp/brum-debug.log; then
            echo "✓ A handler matched the key"

            if grep -q "GLOBALNAV: Handle called" /tmp/brum-debug.log; then
                echo "✓ GlobalNavigationHandler.Handle() WAS called"

                if grep -q "GLOBALNAV: Number key" /tmp/brum-debug.log; then
                    echo "✅ View switch WAS attempted!"
                    echo "   → Handler logic is WORKING"
                    echo "   → Problem must be in viewManager or View update"
                fi
            else
                echo "⚠️ Handler matched but Handle() NOT called"
            fi
        else
            echo "⚠️ No handler matched the key"
        fi
    else
        echo "⚠️ Keyboard handler registry was NEVER called"
        echo "   → Problem is in Update() routing to keyRegistry"
    fi

else
    echo "❌ No debug log found at /tmp/brum-debug.log"
    echo ""
    echo "This means the Model was never created or Update() never called."
    echo "Check if brummer started at all."
fi

echo ""
echo "=========================================="
echo ""
echo "💡 INSIGHT: Since we can't send keyboard input programmatically,"
echo "   the real question is: does the handler even get called when"
echo "   YOU press keys in the actual TUI?"
echo ""
echo "The previous test showed keyboard messages ARE received."
echo "Now we need to see if HandleKey() is being called."
echo ""
echo "Run brummer manually and check the debug log for:"
echo "  - 'HANDLER: Processing key' messages"
echo "  - 'HANDLER: ✅ ... CAN HANDLE this key!'"
echo "  - 'GLOBALNAV: Handle called for key: ...'"
echo ""
