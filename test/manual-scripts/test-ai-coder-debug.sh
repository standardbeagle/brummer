#!/bin/bash
set -e

echo "Testing AI Coder with debug logging..."

# Start brummer in a way that we can interact with it
(
    echo "Starting brummer..."
    sleep 2
    # Navigate to AI Coders tab (press Tab 5 times)
    printf '\t\t\t\t\t'
    sleep 1
    # Run the test-claude AI coder
    echo "/ai test-claude"
    sleep 3
    # Try to see logs
    printf '\t\t'  # Go to logs tab
    sleep 2
    # Exit
    printf 'q'
) | ./brum 2>&1 | tee ai-coder-debug.log

echo "Test completed. Check ai-coder-debug.log for output."