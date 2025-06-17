#!/bin/bash

echo "Testing MCP Activity Tracking..."

# Start brummer in debug mode with MCP server
echo "Starting brummer with MCP server in debug mode..."
./brum --no-tui --debug &
BRUM_PID=$!

# Wait for server to start
sleep 2

# Test some MCP requests
echo "Sending test MCP requests..."

# Initialize
echo "1. Initialize"
curl -s -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}' | jq .

# List tools
echo -e "\n2. List tools"
curl -s -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | jq .

# List resources
echo -e "\n3. List resources"
curl -s -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"resources/list","params":{}}' | jq .

# Call a tool
echo -e "\n4. Call scripts_list tool"
curl -s -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"scripts_list","arguments":{}}}' | jq .

# Test with SSE connection
echo -e "\n5. Testing SSE connection..."
curl -N -H "Accept: text/event-stream" http://localhost:7777/mcp &
SSE_PID=$!

sleep 2

# Send a request while SSE is connected
echo -e "\n6. Send request while SSE connected"
curl -s -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -H "Mcp-Session-Id: test-session-123" \
  -d '{"jsonrpc":"2.0","id":5,"method":"tools/list","params":{}}' | jq .

# Kill SSE connection
kill $SSE_PID 2>/dev/null

echo -e "\nTest complete. Press Ctrl+C to stop brummer."
wait $BRUM_PID