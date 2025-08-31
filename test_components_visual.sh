#!/bin/bash

# Test script to verify component system with URLs and proxy display

echo "Testing Brummer Component System - URLs and Proxy Display"
echo "=========================================================="
echo ""
echo "This test will:"
echo "1. Start brummer with the component system enabled"
echo "2. Run a dev script that outputs URLs"
echo "3. Show the URLs view with detected URLs"
echo ""
echo "Instructions:"
echo "- Press '1' to see Processes view" 
echo "- Press '2' to see Logs view"
echo "- Press '3' to see Errors view"
echo "- Press '4' to see URLs view (with proxy info if available)"
echo "- Type 'dev' and press Enter to start the dev script"
echo "- Press 'q' to quit"
echo ""
echo "Starting in 3 seconds..."
sleep 3

# Copy test package.json
cp test-proxy-package.json package.json

# Run brummer with component system
BRUMMER_USE_COMPONENTS=true ./brum