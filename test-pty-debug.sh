#!/bin/bash

echo "Starting Brummer with PTY debug logging enabled..."
echo "Log file will be created as: pty-debug-*.log"
echo
echo "Instructions:"
echo "1. The TUI will start"
echo "2. Press '6' to go to AI Coders tab"
echo "3. Run '/ai claude' to start Claude Code"
echo "4. Type some commands to generate output"
echo "5. Exit with Ctrl+C or Esc when done"
echo
echo "Press Enter to continue..."
read

# Run brummer with debug logging enabled
DEBUG_PTY_OUTPUT=true ./brum

echo
echo "Debug log files created:"
ls -la pty-debug-*.log 2>/dev/null

echo
echo "To examine the control codes, use:"
echo "  cat pty-debug-*.log | less"
echo
echo "Look for patterns like:"
echo "  \\e[2J - Clear screen"
echo "  \\e[H - Cursor home"
echo "  \\e[K - Clear line"
echo "  \\e[nA - Cursor up n lines"
echo "  \\e[nB - Cursor down n lines"