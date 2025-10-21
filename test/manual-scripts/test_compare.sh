#!/bin/bash

echo "=========================================="
echo "🔬 Comparison Test: Minimal vs Real Brum"
echo "=========================================="
echo ""
echo "This will test if the minimal BubbleTea program receives"
echo "keyboard input when the real brum doesn't."
echo ""

# Build minimal test
echo "📦 Building minimal test..."
go build -o test_minimal ./test_minimal_bubbletea.go

echo ""
echo "Test 1: Minimal BubbleTea (known to work)"
echo "----------------------------------------"
echo "Press a few keys, then 'q' to quit"
echo "Press ENTER to start..."
read

rm -f /tmp/minimal-test.log
./test_minimal 2>&1 | tee /tmp/minimal-test.log

echo ""
echo "Test 2: Real Brum"
echo "----------------------------------------"
echo "Press: 1 1 1 2 2 2"
echo "Then kill from another terminal: pkill -9 brum"
echo "Press ENTER to start..."
read

cd /tmp
rm -rf test_kb && mkdir test_kb && cd test_kb
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json
rm -f /tmp/brum-debug.log

/home/beagle/work/brummer/brum

echo ""
echo "=========================================="
echo "📊 Comparison Results"
echo "=========================================="
echo ""
echo "Minimal test output:"
if [ -f /tmp/minimal-test.log ]; then
    tail -5 /tmp/minimal-test.log
else
    echo "No log found"
fi

echo ""
echo "Brum test output:"
if [ -f /tmp/brum-debug.log ]; then
    grep "Keyboard message" /tmp/brum-debug.log || echo "No keyboard messages in brum"
else
    echo "No log found"
fi

echo ""
echo "=========================================="
