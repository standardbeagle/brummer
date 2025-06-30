#!/bin/bash
# Test the full hub workflow

echo "=== Testing Hub Workflow ==="

# 1. List instances
echo "1. Listing instances:"
INSTANCE_ID=$(echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"instances_list","arguments":{}}}' | \
  ./brum --mcp 2>/dev/null | \
  jq -r '.result.content[0].text' | \
  jq -r '.[0].id')

echo "Found instance: $INSTANCE_ID"

# 2. Connect to the instance
echo -e "\n2. Connecting to instance:"
echo "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"instances_connect\",\"arguments\":{\"instance_id\":\"$INSTANCE_ID\"}}}" | \
  ./brum --mcp 2>&1 | \
  jq -r '.result.content[0].text' 2>/dev/null || echo "Connection response shown in stderr"

# 3. List tools to see if proxy tools are available
echo -e "\n3. Listing tools after connection:"
echo '{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}' | \
  ./brum --mcp 2>/dev/null | \
  jq -r '.result.tools[].name' | \
  grep "${INSTANCE_ID}_" | \
  head -5

# 4. Try to use logs_stream
echo -e "\n4. Trying to use logs_stream:"
echo "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"${INSTANCE_ID}_logs_stream\",\"arguments\":{\"limit\":5,\"follow\":false}}}" | \
  ./brum --mcp 2>&1 | \
  jq . 2>/dev/null || echo "See stderr for debug output"