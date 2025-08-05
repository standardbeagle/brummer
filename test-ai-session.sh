#!/bin/bash
echo "Starting test AI coder session..."
# Navigate to AI Coders tab (Tab 5 times)
sleep 2
printf '\t\t\t\t\t'
sleep 1
# Run test-claude
echo "/ai test-claude"
sleep 3
# Go to logs to see what happened
printf '\t\t'
sleep 1
# Go back to AI Coders
printf '\t\t\t\t'
sleep 2
# Try to focus terminal
printf '\n'
sleep 1
# Type something
echo "Hello AI!"
sleep 2
# Exit
printf 'q'
