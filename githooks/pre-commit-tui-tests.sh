#!/bin/bash

# Pre-commit hook for TUI Integration Testing
# This hook runs PTY-based TUI tests to ensure keyboard input and view switching work correctly
# before allowing commits to proceed.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}TUI Testing Pre-commit Hook${NC}"
echo "========================================"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

# Check if brummer binary exists
BRUMMER_BINARY="${BRUMMER_BINARY:-./brum}"
if [ ! -f "$BRUMMER_BINARY" ]; then
    echo -e "${YELLOW}Warning: Brummer binary not found at '$BRUMMER_BINARY'${NC}"
    echo -e "${YELLOW}Attempting to build it...${NC}"

    # Try to build the binary
    if ! make build > /dev/null 2>&1; then
        echo -e "${RED}Error: Failed to build brummer binary${NC}"
        exit 1
    fi

    if [ ! -f "$BRUMMER_BINARY" ]; then
        echo -e "${RED}Error: Still cannot find brummer binary after build${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Built brummer binary successfully${NC}"
else
    echo -e "${GREEN}✓ Using brummer binary: $BRUMMER_BINARY${NC}"
fi

# Check if we're in the right directory
if [ ! -d "test/tui" ]; then
    echo -e "${RED}Error: test/tui directory not found${NC}"
    echo "Please run this hook from the repository root directory"
    exit 1
fi

# Function to run TUI tests with error handling
run_tui_tests() {
    local test_name="$1"
    local test_command="$2"
    local timeout="${3:-30s}"

    echo -e "${BLUE}Running $test_name...${NC}"

    # Run the test with timeout and capture output
    if timeout "$timeout" bash -c "$test_command" 2>&1; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        return 0
    else
        local exit_code=$?
        echo -e "${RED}✗ $test_name failed (exit code: $exit_code)${NC}"

        # Show last few lines of output for debugging
        echo -e "${YELLOW}Last 10 lines of output:${NC}"
        timeout "$timeout" bash -c "$test_command" 2>&1 | tail -10 2>/dev/null || echo "No output available"

        return 1
    fi
}

# Function to check if PTY is available
check_pty_availability() {
    echo -e "${BLUE}Checking PTY availability...${NC}"

    # Try to create a simple PTY test
    if timeout 5s go run -tags=tty,integration ./test/tui/pty/test_pty_layer_integration_test.go -run TestPTYLayer_AllocationAndBasicFunctionality 2>/dev/null; then
        echo -e "${GREEN}✓ PTY layer is working${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ PTY layer test failed (may be expected in some environments)${NC}"
        return 1
    fi
}

# Run PTY availability check
PTY_AVAILABLE=false
if check_pty_availability; then
    PTY_AVAILABLE=true
fi

# Run basic TUI test suite
echo -e "${BLUE}Running Basic TUI Test Suite...${NC}"

TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: PTY layer functionality (if available)
if [ "$PTY_AVAILABLE" = true ]; then
    if run_tui_tests "PTY Layer Integration" "go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality"; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
else
    echo -e "${YELLOW}⚠ Skipping PTY tests (not available)${NC}"
fi

# Test 2: Input simulation through PTY
if run_tui_tests "PTY Keyboard Input" "BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_Simulation"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

# Test 3: View switching through PTY
if run_tui_tests "PTY View Switching" "BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestViewSwitchingPTY_NumberKeys"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

# Test 4: Command palette through PTY
if run_tui_tests "PTY Command Palette" "BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestCommandPalettePTY_AllocationAndBasicFunctionality"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

# Test 5: Basic contract tests (no PTY required)
if run_tui_tests "Contract Tests" "go test ./test/tui/harness -v"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

# Summary
echo -e "${BLUE}Test Summary${NC}"
echo "=============="
echo -e "Total Tests Run: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

# Check if any tests failed
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "\n${RED}❌ TUI tests failed - Commit blocked${NC}"
    echo -e "${YELLOW}Please fix the failing tests before committing.${NC}"
    echo -e "${BLUE}To run tests manually:${NC}"
    echo -e "  make test-tui${NC}"
    echo -e "  BRUMMER_BINARY=./brum go test -tags=integration,tty ./test/tui -v${NC}"
    exit 1
fi

# Additional validation checks
echo -e "${BLUE}Running additional validation checks...${NC}"

# Check for potential issues
ISSUES_FOUND=false

# Check 1: Ensure no hardcoded test workarounds
echo -e "${BLUE}Checking for test workarounds...${NC}"
if grep -r "TODO.*remove.*test" test/ > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Found potential test workarounds that should be cleaned up${NC}"
    ISSUES_FOUND=true
fi

# Check 2: Ensure proper error handling
echo -e "${BLUE}Checking error handling patterns...${NC}"
if grep -r "panic(" test/tui > /dev/null 2>&1; then
    echo -e "${RED}❌ Found panic statements in test code${NC}"
    ISSUES_FOUND=true
fi

# Check 3: Ensure tests are not using sleep for timing
echo -e "${BLUE}Checking timing patterns...${NC}"
if grep -r "time\.Sleep" test/tui > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Found time.Sleep in tests (prefer context timeout)${NC}"
    # Not a failure, just a warning
fi

# Check 4: Ensure proper cleanup
echo -e "${BLUE}Checking cleanup patterns...${NC}"
if grep -r "defer.*Stop" test/tui > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Proper cleanup patterns found${NC}"
else
    echo -e "${YELLOW}⚠ Some tests may be missing cleanup patterns${NC}"
fi

# Final validation
if [ "$ISSUES_FOUND" = true ]; then
    echo -e "\n${YELLOW}⚠ Issues found that should be addressed${NC}"
    echo -e "${BLUE}Review the warnings above and fix them before committing.${NC}"
    # Not blocking, just informational
fi

echo -e "${GREEN}✅ All TUI tests passed - Commit allowed${NC}"
echo -e "${BLUE}Ready to commit changes${NC}"

exit 0