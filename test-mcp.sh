#!/bin/bash

# Run MCP tests and generate summary

echo "Running MCP Protocol Tests..."
echo "============================"
echo

# Run tests with coverage
echo "Running tests with coverage..."
go test -v -cover ./internal/mcp -timeout=30s 2>&1 | tee mcp-test-results.txt

# Extract summary
echo
echo "Test Summary:"
echo "============="
grep -E "^(PASS|FAIL|ok|---)" mcp-test-results.txt | tail -20

# Count results
PASSED=$(grep -c "PASS:" mcp-test-results.txt || echo 0)
FAILED=$(grep -c "FAIL:" mcp-test-results.txt || echo 0)
SKIPPED=$(grep -c "SKIP:" mcp-test-results.txt || echo 0)

echo
echo "Results:"
echo "  Passed:  $PASSED"
echo "  Failed:  $FAILED"
echo "  Skipped: $SKIPPED"

# Check coverage
COVERAGE=$(grep "coverage:" mcp-test-results.txt | tail -1 | awk '{print $2}')
if [ -n "$COVERAGE" ]; then
    echo "  Coverage: $COVERAGE"
fi

# Exit with appropriate code
if [ "$FAILED" -gt 0 ]; then
    echo
    echo "Some tests failed. Check mcp-test-results.txt for details."
    exit 1
else
    echo
    echo "All tests passed!"
    exit 0
fi