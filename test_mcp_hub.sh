#!/bin/bash
# Test script for MCP hub functionality

echo "Testing MCP Hub with stdio transport..."
echo

echo "1. Testing initialize request:"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{"tools":{}}}}' | ./brum --mcp 2>/dev/null | jq .

echo
echo "2. Testing tools/list request:"
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{"tools":{}}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./brum --mcp 2>/dev/null | tail -1 | jq .

echo
echo "3. Testing instances/list tool call:"
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{"tools":{}}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"instances/list","arguments":{}}}' | ./brum --mcp 2>/dev/null | tail -1 | jq .

echo
echo "All tests completed!"