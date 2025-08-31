#!/bin/bash
# Test with stderr visible
cd /tmp
mkdir -p test_brummer
cd test_brummer
echo '{"name":"test","scripts":{"hello":"echo Hello"}}' > package.json
BRUMMER_USE_COMPONENTS=true timeout 2 /home/beagle/work/brummer/brum 2>&1 | head -50