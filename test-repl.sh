#!/bin/bash

# Test script for REPL functionality

echo "Testing REPL functionality..."

# Start brummer in the background with MCP server
./brum --no-tui --port 7777 &
BRUM_PID=$!

# Wait for server to start
sleep 2

# Test REPL execute command
echo "Sending REPL test command..."
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "repl_execute",
      "arguments": {
        "code": "2 + 2"
      }
    }
  }'

echo ""
echo "Test complete. Killing brummer..."
kill $BRUM_PID