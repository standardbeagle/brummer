#!/bin/bash
set -e

echo "Testing AI Coder debug flow..."

# Run brummer with a script that tests AI coder
(
  sleep 2
  # Switch to AI Coders view (Tab 5 times)
  echo -e '\t\t\t\t\t'
  sleep 1
  # Open command palette
  echo '/'
  sleep 0.5
  # Type ai test-claude
  echo 'ai test-claude'
  sleep 3
  # Switch to logs view to see debug output
  echo -e '\t\t'
  sleep 2
  # Quit
  echo 'q'
) | ./brum 2>&1 | grep -E "(Available providers|Provider names|CLI Tool Command|Calling CreateSessionWithEnv|Session created|Error creating PTY)" || echo "No debug messages found"