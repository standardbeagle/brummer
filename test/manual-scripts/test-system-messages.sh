#!/bin/bash

# Test script to verify process display and log streaming

echo "Testing Brummer fixes..."
echo "1. Starting brummer in test-project directory"
echo "2. Will run 'dev' script which generates logs every second"
echo "3. Check that:"
echo "   - Processes tab shows running process"
echo "   - Logs tab shows streaming logs in real-time"
echo ""
echo "Starting in 3 seconds..."
sleep 3

cd test-project
../brum dev