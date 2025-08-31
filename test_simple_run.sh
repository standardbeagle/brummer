#!/bin/bash
# Simple test to see stderr output

TESTDIR=$(mktemp -d)
cd $TESTDIR
echo '{"name":"test","scripts":{"hello":"echo Hello"}}' > package.json

echo "Running in $TESTDIR with package.json:"
cat package.json

echo ""
echo "Starting brummer with components..."
BRUMMER_USE_COMPONENTS=true timeout 2 /home/beagle/work/brummer/brum 2>&1 | head -50

cd -
rm -rf $TESTDIR