#!/bin/bash
# Simple manual test for AI terminal
set -e

echo "Building..."
make build > /dev/null 2>&1

echo "Starting brummer with components enabled..."
echo "Once it starts:"
echo "1. Press '/' to open command palette"
echo "2. Type 'ai terminal' and press Enter"
echo "3. Wait for the terminal to start"
echo "4. Type 'echo HELLO' and press Enter"
echo "5. Look for diagnostic output showing channel stats"
echo ""
echo "Press Enter to start..."
read

BRUMMER_USE_COMPONENTS=true ./brum