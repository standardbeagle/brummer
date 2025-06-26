#!/bin/bash
# Comprehensive test script for MCP hub foundation (Phase 1)

set -e

echo "=== MCP Hub Foundation Tests ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

test_count=0
passed_count=0

# Function to run a test
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_contains="$3"
    
    test_count=$((test_count + 1))
    echo -n "Test $test_count: $test_name... "
    
    result=$(eval "$command" 2>/dev/null || echo "ERROR")
    
    if [[ "$result" == *"$expected_contains"* ]]; then
        echo -e "${GREEN}PASSED${NC}"
        passed_count=$((passed_count + 1))
    else
        echo -e "${RED}FAILED${NC}"
        echo "  Expected to contain: $expected_contains"
        echo "  Got: $result"
    fi
}

# Build the binary first
echo "Building brummer..."
make build >/dev/null 2>&1

# Test 1: Initialize protocol
run_test "Initialize protocol" \
    "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}' | ./brum --mcp | jq -r '.result.serverInfo.name'" \
    "brummer-hub"

# Test 2: Protocol version
run_test "Protocol version check" \
    "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}' | ./brum --mcp | jq -r '.result.protocolVersion'" \
    "2025-03-26"

# Test 3: Tools list includes both tools
run_test "Tools list contains instances/list" \
    "echo -e '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}' | ./brum --mcp | tail -1 | jq -r '.result.tools[].name' | grep -c 'instances/list'" \
    "1"

run_test "Tools list contains instances/connect" \
    "echo -e '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}' | ./brum --mcp | tail -1 | jq -r '.result.tools[].name' | grep -c 'instances/connect'" \
    "1"

# Test 4: instances/list returns empty array
run_test "instances/list returns empty array" \
    "echo -e '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"instances/list\",\"arguments\":{}}}' | ./brum --mcp | tail -1 | jq -r '.result.content[0].text'" \
    "[]"

# Test 5: instances/connect returns not implemented
run_test "instances/connect returns not implemented" \
    "echo -e '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"instances/connect\",\"arguments\":{\"instance_id\":\"test-123\"}}}' | ./brum --mcp | tail -1 | jq -r '.result.content[0].text'" \
    "not implemented"

# Test 6: Invalid tool call returns error
run_test "Invalid tool returns error" \
    "echo -e '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"invalid/tool\",\"arguments\":{}}}' | ./brum --mcp | tail -1 | jq -r '.error.message' | grep -o 'not found'" \
    "not found"

# Test 7: Multiple sequential requests
run_test "Multiple sequential requests work" \
    "echo -e '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"capabilities\":{\"tools\":{}}}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"instances/list\",\"arguments\":{}}}' | ./brum --mcp | wc -l | xargs" \
    "3"

# Test 8: Hub mode doesn't start TUI
run_test "Hub mode runs without TUI" \
    "timeout 2 ./brum --mcp < /dev/null 2>&1 | grep -c 'TUI'" \
    "0"

# Test 9: Connection manager stub exists
run_test "Connection manager file exists" \
    "test -f internal/mcp/connection_manager.go && echo 'exists'" \
    "exists"

echo
echo "=== Test Summary ==="
echo "Total tests: $test_count"
echo -e "Passed: ${GREEN}$passed_count${NC}"
echo -e "Failed: ${RED}$((test_count - passed_count))${NC}"

if [ $passed_count -eq $test_count ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi