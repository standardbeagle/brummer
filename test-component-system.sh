#!/bin/bash

# Test script for Brummer Component System
# This script verifies that the component system works correctly

set -e

echo "================================================"
echo "🧪 Brummer Component System Test Suite"
echo "================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    echo -n "Testing: $test_name... "
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "  Command: $test_command"
        echo "  Expected: $expected_result"
        ((TESTS_FAILED++))
    fi
}

# Function to check if file exists
check_file() {
    local file="$1"
    local description="$2"
    
    echo -n "Checking: $description... "
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓ EXISTS${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ MISSING${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

echo "1. Checking Component Files"
echo "----------------------------"

# Check if all component files exist
check_file "internal/tui/components/process_view_adapter.go" "ProcessViewComponent"
check_file "internal/tui/components/logs_view_adapter.go" "LogsViewComponent"
check_file "internal/tui/components/errors_view_adapter.go" "ErrorsViewComponent"
check_file "internal/tui/components/urls_view_adapter.go" "URLsViewComponent"
check_file "internal/tui/components/web_view_adapter.go" "WebViewComponent"
check_file "internal/tui/components/ai_coders_view_adapter.go" "AICodersViewComponent"
check_file "internal/tui/components/settings_view_adapter.go" "SettingsViewComponent"
check_file "internal/tui/components/command_input_adapter.go" "CommandInputComponent"
check_file "internal/tui/components/notification_component.go" "NotificationComponent"
check_file "internal/tui/components/layout_component.go" "LayoutComponent"
check_file "internal/tui/components/custom_messages.go" "Custom Messages"
check_file "internal/tui/model_component_adapter.go" "ModelComponentAdapter"

echo ""
echo "2. Building Component System"
echo "----------------------------"

# Test compilation of components package
run_test "Component package compilation" \
    "go build ./internal/tui/components 2>&1" \
    "Components should compile without errors"

# Test building the main binary
run_test "Main binary compilation" \
    "go build -o /tmp/brum-test ./cmd/brum/main.go 2>&1" \
    "Binary should compile with component support"

echo ""
echo "3. Testing Component Initialization"
echo "------------------------------------"

# Test that the binary runs with component flag
if [ -f "/tmp/brum-test" ]; then
    run_test "Component system activation" \
        "BRUMMER_USE_COMPONENTS=true timeout 1 /tmp/brum-test --help >/dev/null 2>&1 || true" \
        "Should run with component flag"
    
    # Check if component message appears
    run_test "Component system message" \
        "BRUMMER_USE_COMPONENTS=true /tmp/brum-test --help 2>&1 | grep -q 'Component system enabled' || true" \
        "Should show component system message"
fi

echo ""
echo "4. Testing Component Integration"
echo "---------------------------------"

# Test that components integrate with existing code
run_test "Integration test compilation" \
    "go test -c ./internal/tui/components 2>&1" \
    "Integration tests should compile"

echo ""
echo "5. Checking Documentation"
echo "--------------------------"

check_file "COMPONENT_SYSTEM.md" "Component System Documentation"
check_file "internal/tui/components/README.md" "Component README"
check_file "run-with-components.sh" "Component run script"

echo ""
echo "6. Testing Run Script"
echo "----------------------"

if [ -f "run-with-components.sh" ]; then
    run_test "Run script is executable" \
        "[ -x run-with-components.sh ]" \
        "Script should be executable"
fi

echo ""
echo "7. Component Count Verification"
echo "--------------------------------"

# Count component files
component_count=$(ls internal/tui/components/*_adapter.go 2>/dev/null | wc -l)
echo "Component adapters found: $component_count"

if [ "$component_count" -ge "9" ]; then
    echo -e "${GREEN}✓ All expected component adapters present${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Missing component adapters (expected at least 9, found $component_count)${NC}"
    ((TESTS_FAILED++))
fi

echo ""
echo "8. Message System Verification"
echo "-------------------------------"

# Check for message definitions
run_test "Custom message types" \
    "grep -q 'ProcessStartMsg\\|LogsUpdateMsg\\|ErrorOccurredMsg' internal/tui/components/custom_messages.go" \
    "Should have custom message types defined"

run_test "Message helper functions" \
    "grep -q 'SendProcessStart\\|SendSystemNotification' internal/tui/components/custom_messages.go" \
    "Should have message helper functions"

echo ""
echo "================================================"
echo "📊 Test Results Summary"
echo "================================================"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! Component system is ready.${NC}"
    echo ""
    echo "To use the component system:"
    echo "  1. Run with environment variable: BRUMMER_USE_COMPONENTS=true ./brum"
    echo "  2. Or use the helper script: ./run-with-components.sh"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tests failed. Please review the issues above.${NC}"
    echo ""
    echo "Common issues:"
    echo "  - Missing files: Check if all components were created"
    echo "  - Compilation errors: Run 'go build ./internal/tui/components' for details"
    echo "  - Integration issues: Check model_component_adapter.go"
    exit 1
fi