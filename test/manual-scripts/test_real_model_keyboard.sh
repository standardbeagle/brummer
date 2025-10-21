#!/bin/bash

cd /tmp
mkdir -p final_test && cd final_test
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json

# Clear debug log
rm -f /tmp/brum-debug.log

echo "Starting brum with REAL Model..."
echo "IMPORTANT: This terminal must have FOCUS!"
echo "Press these keys NOW: 1 1 1 2 2 2 Tab Tab"
echo ""

# Run brum - it will capture keyboard if it works
/home/beagle/work/brummer/brum
