#!/bin/bash

echo "=========================================="
echo "🔍 Simple Foreground Test"
echo "=========================================="
echo ""
echo "This will run brummer in the FOREGROUND so it can receive your keystrokes."
echo ""
echo "INSTRUCTIONS:"
echo "1. Brummer will start"
echo "2. Press: 1 1 1 2 2 2 Tab Tab Tab"
echo "3. Open ANOTHER terminal and run: pkill -9 brum"
echo "4. Come back here and check the output"
echo ""
echo "Press ENTER to start..."
read

# Setup
cd /tmp
rm -rf test_kb && mkdir test_kb && cd test_kb
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json
rm -f /tmp/brum-debug.log

echo ""
echo "🚀 Starting brummer in foreground..."
echo "📍 Press keys now, then kill from another terminal: pkill -9 brum"
echo ""

# Run in foreground (THIS is the key difference)
/home/beagle/work/brummer/brum

# After it's killed, show the log
echo ""
echo "=========================================="
echo "📊 Debug Log"
echo "=========================================="
cat /tmp/brum-debug.log

echo ""
echo "🔍 Quick Check:"
grep -c "Keyboard message received" /tmp/brum-debug.log && echo "✓ Keyboard messages received" || echo "❌ No keyboard messages"
grep -q "VIEWSWITCH:" /tmp/brum-debug.log && echo "✓ ViewSwitch messages received" || echo "❌ No ViewSwitch messages"
echo ""
