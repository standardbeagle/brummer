#!/bin/bash

# TUI Integration Test Execution Script
# Provides comprehensive test suite execution for PTY-based TUI testing
# This script serves as the main entry point for running TUI tests manually
# or in CI/CD environments

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_RESULTS_DIR="$PROJECT_ROOT/test/results"
TEST_LOGS_DIR="$PROJECT_ROOT/test/logs"

# Default values
BRUMMER_BINARY="${BRUMMER_BINARY:-$PROJECT_ROOT/brum}"
TEST_TIMEOUT="${TEST_TIMEOUT:-120s}"
VERBOSE="${VERBOSE:-false}"
KEEP_LOGS="${KEEP_LOGS:-false}"
PARALLEL="${PARALLEL:-false}"
GENERATE_REPORTS="${GENERATE_REPORTS:-true}"
TEST_FILTER="${TEST_FILTER:-}"

# Ensure directories exist
mkdir -p "$TEST_RESULTS_DIR" "$TEST_LOGS_DIR"

# Utility functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] ✓ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] ⚠ $1${NC}"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ✗ $1${NC}"
}

log_info() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${WHITE}[$(date '+%Y-%m-%d %H:%M:%S')] ℹ $1${NC}"
    fi
}

show_usage() {
    cat << EOF
TUI Integration Test Execution Script

USAGE:
    $0 [OPTIONS] [TEST_PATTERNS...]

OPTIONS:
    -b, --binary PATH        Path to brummer binary (default: ./brum)
    -t, --timeout DURATION   Test timeout (default: 120s)
    -v, --verbose            Enable verbose output
    -k, --keep-logs          Keep test logs after execution
    -p, --parallel           Run tests in parallel where possible
    -r, --no-reports         Don't generate test reports
    -f, --filter PATTERN     Filter tests by pattern
    -h, --help              Show this help message

TEST PATTERNS:
    Contract              Run contract tests only
    Integration           Run integration tests only
    PTY                   Run PTY layer tests only
    Keyboard              Run keyboard input tests only
    ViewSwitching         Run view switching tests only
    CommandPalette        Run command palette tests only
    All                   Run all tests (default)

EXAMPLES:
    $0                                    # Run all tests with default settings
    $0 -v -p Contract                     # Run contract tests verbosely in parallel
    $0 -t 300s -k Integration             # Run integration tests with 5s timeout, keep logs
    $0 -f "Keyboard.*PTY"                 # Run tests matching pattern
    $0 Contract Integration                # Run specific test categories

ENVIRONMENT VARIABLES:
    BRUMMER_BINARY        Path to brummer binary
    TEST_TIMEOUT          Default test timeout
    VERBOSE               Enable verbose output (true/false)
    KEEP_LOGS             Keep test logs (true/false)
    PARALLEL              Run tests in parallel (true/false)
    GENERATE_REPORTS      Generate test reports (true/false)
    TEST_FILTER           Filter tests by pattern

EXIT CODES:
    0                     All tests passed
    1                     Some tests failed
    2                     Invalid arguments or configuration
    3                     Brummer binary not found or not executable
    4                     PTY layer not available
    5                     Test environment setup failed

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--binary)
                BRUMMER_BINARY="$2"
                shift 2
                ;;
            -t|--timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -k|--keep-logs)
                KEEP_LOGS="true"
                shift
                ;;
            -p|--parallel)
                PARALLEL="true"
                shift
                ;;
            -r|--no-reports)
                GENERATE_REPORTS="false"
                shift
                ;;
            -f|--filter)
                TEST_FILTER="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            Contract|Integration|PTY|Keyboard|ViewSwitching|CommandPalette|All)
                if [[ -z "$TEST_CATEGORIES" ]]; then
                    TEST_CATEGORIES="$1"
                else
                    TEST_CATEGORIES="$TEST_CATEGORIES $1"
                fi
                shift
                ;;
            *)
                log_error "Unknown argument: $1"
                show_usage
                exit 2
                ;;
        esac
    done

    # Default to all tests if no categories specified
    if [[ -z "$TEST_CATEGORIES" ]]; then
        TEST_CATEGORIES="All"
    fi
}

# Validate environment and configuration
validate_environment() {
    log "Validating test environment..."

    # Check if we're in the right directory
    if [[ ! -d "$PROJECT_ROOT/test/tui" ]]; then
        log_error "TUI test directory not found at $PROJECT_ROOT/test/tui"
        exit 5
    fi

    # Check brummer binary
    if [[ ! -f "$BRUMMER_BINARY" ]]; then
        log_error "Brummer binary not found at $BRUMMER_BINARY"
        exit 3
    fi

    if [[ ! -x "$BRUMMER_BINARY" ]]; then
        log_error "Brummer binary is not executable: $BRUMMER_BINARY"
        exit 3
    fi

    # Check Go environment
    if ! command -v go > /dev/null 2>&1; then
        log_error "Go is not installed or not in PATH"
        exit 5
    fi

    # Check for required Go packages
    log_info "Checking Go dependencies..."
    if ! go mod tidy > "$TEST_LOGS_DIR/mod-tidy.log" 2>&1; then
        log_error "Failed to tidy Go modules. Check $TEST_LOGS_DIR/mod-tidy.log"
        exit 5
    fi

    # Check for PTY dependencies
    log_info "Checking PTY dependencies..."
    if ! go list -m github.com/creack/pty > /dev/null 2>&1; then
        log_error "PTY dependency (github.com/creack/pty) not found"
        log_info "Adding PTY dependency..."
        go get github.com/creack/pty@v1.1.21
    else
        log_success "PTY dependency found"
    fi

    # Test PTY availability
    log_info "Checking PTY availability..."
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality > "$TEST_LOGS_DIR/pty-check.log" 2>&1; then
        log_success "PTY layer is available and functional"
        PTY_AVAILABLE=true
    else
        log_warning "PTY layer test failed - some tests may be skipped"
        PTY_AVAILABLE=false
    fi

    log_success "Environment validation completed"
}

# Setup test environment
setup_test_environment() {
    log "Setting up test environment..."

    # Create test directories
    mkdir -p "$TEST_RESULTS_DIR" "$TEST_LOGS_DIR"

    # Set environment variables for tests
    export BRUMMER_BINARY="$BRUMMER_BINARY"
    export TEST_RESULTS_DIR="$TEST_RESULTS_DIR"
    export TEST_LOGS_DIR="$TEST_LOGS_DIR"
    export PTY_AVAILABLE="$PTY_AVAILABLE"

    # Clean up previous test artifacts if not keeping logs
    if [[ "$KEEP_LOGS" != "true" ]]; then
        log_info "Cleaning up previous test artifacts..."
        rm -f "$TEST_RESULTS_DIR"/*.json "$TEST_RESULTS_DIR"/*.xml "$TEST_RESULTS_DIR"/*.html
        find "$TEST_LOGS_DIR" -name "*.log" -type f -mtime +1 -delete 2>/dev/null || true
    fi

    log_success "Test environment setup completed"
}

# Run a single test and capture results
run_test() {
    local test_name="$1"
    local test_command="$2"
    local test_category="$3"
    local log_file="$TEST_LOGS_DIR/${test_category,,}_${test_name,,}.log"
    local result_file="$TEST_RESULTS_DIR/${test_category,,}_${test_name,,}.json"

    log_info "Running $test_category test: $test_name"

    # Apply test filter if specified
    if [[ -n "$TEST_FILTER" && ! "$test_name" =~ $TEST_FILTER ]]; then
        log_info "Skipping test due to filter: $test_name"
        return 0
    fi

    local start_time=$(date +%s)
    local exit_code=0

    # Run the test with timeout
    if timeout "$TEST_TIMEOUT" bash -c "$test_command" > "$log_file" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))

        log_success "$test_name passed (${duration}s)"

        # Create result JSON
        cat > "$result_file" << EOF
{
    "test_name": "$test_name",
    "category": "$test_category",
    "status": "PASS",
    "duration_seconds": $duration,
    "timestamp": "$(date -Iseconds)",
    "log_file": "$log_file"
}
EOF

        return 0
    else
        local exit_code=$?
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))

        log_error "$test_name failed (${duration}s, exit code: $exit_code)"

        # Create result JSON
        cat > "$result_file" << EOF
{
    "test_name": "$test_name",
    "category": "$test_category",
    "status": "FAIL",
    "duration_seconds": $duration,
    "exit_code": $exit_code,
    "timestamp": "$(date -Iseconds)",
    "log_file": "$log_file",
    "error_summary": "$(tail -5 "$log_file" | tr '\n' ';')"
}
EOF

        return 1
    fi
}

# Run contract tests
run_contract_tests() {
    log "Running Contract Tests..."

    local contract_tests=(
        "TestHarness_Contract:go test -tags=tty,integration ./test/tui/harness -v -run TestTestHarness_Contract"
        "InputSimulator_Contract:go test -tags=tty,integration ./test/tui/harness -v -run TestInputSimulator_Contract"
        "StateValidator_Contract:go test -tags=tty,integration ./test/tui/harness -v -run TestStateValidator_Contract"
        "TestReporter_Contract:go test -tags=tty,integration ./test/tui/harness -v -run TestTestReporter_Contract"
    )

    local passed=0
    local failed=0

    for test_spec in "${contract_tests[@]}"; do
        IFS=':' read -r test_name test_command <<< "$test_spec"

        if run_test "$test_name" "$test_command" "Contract"; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    log "Contract Tests: $passed passed, $failed failed"
    return $failed
}

# Run integration tests
run_integration_tests() {
    log "Running Integration Tests..."

    local integration_tests=(
        "PTY_Layer_Integration:go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality"
        "Keyboard_Input_PTY:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_Simulation"
        "View_Switching_PTY:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestViewSwitchingPTY_NumberKeys"
        "Command_Palette_PTY:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestCommandPalettePTY_AllocationAndBasicFunctionality"
        "InputService_Integration:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_Integration"
        "NavigationService_Integration:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestNavigationServicePTY_Integration"
    )

    local passed=0
    local failed=0

    for test_spec in "${integration_tests[@]}"; do
        IFS=':' read -r test_name test_command <<< "$test_spec"

        # Skip PTY tests if not available
        if [[ "$PTY_AVAILABLE" != "true" && "$test_name" == PTY_* ]]; then
            log_warning "Skipping $test_name (PTY not available)"
            continue
        fi

        if run_test "$test_name" "$test_command" "Integration"; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    log "Integration Tests: $passed passed, $failed failed"
    return $failed
}

# Run PTY-specific tests
run_pty_tests() {
    if [[ "$PTY_AVAILABLE" != "true" ]]; then
        log_warning "PTY layer not available, skipping PTY tests"
        return 0
    fi

    log "Running PTY Layer Tests..."

    local pty_tests=(
        "PTY_Allocation:go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality"
        "PTY_Input_Simulation:go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_InputSimulation"
        "PTY_Output_Capture:go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_OutputCapture"
        "PTY_Cleanup:go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_Cleanup"
    )

    local passed=0
    local failed=0

    for test_spec in "${pty_tests[@]}"; do
        IFS=':' read -r test_name test_command <<< "$test_spec"

        if run_test "$test_name" "$test_command" "PTY"; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    log "PTY Tests: $passed passed, $failed failed"
    return $failed
}

# Run keyboard-specific tests
run_keyboard_tests() {
    if [[ "$PTY_AVAILABLE" != "true" ]]; then
        log_warning "PTY layer not available, skipping keyboard tests"
        return 0
    fi

    log "Running Keyboard Input Tests..."

    local keyboard_tests=(
        "Keyboard_Basic_Input:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_BasicInput"
        "Keyboard_Special_Keys:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_SpecialKeys"
        "Keyboard_Key_Sequences:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_KeySequences"
        "Keyboard_Input_Timing:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_InputTiming"
    )

    local passed=0
    local failed=0

    for test_spec in "${keyboard_tests[@]}"; do
        IFS=':' read -r test_name test_command <<< "$test_spec"

        if run_test "$test_name" "$test_command" "Keyboard"; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    log "Keyboard Tests: $passed passed, $failed failed"
    return $failed
}

# Run view switching tests
run_view_switching_tests() {
    if [[ "$PTY_AVAILABLE" != "true" ]]; then
        log_warning "PTY layer not available, skipping view switching tests"
        return 0
    fi

    log "Running View Switching Tests..."

    local view_tests=(
        "View_Number_Keys:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestViewSwitchingPTY_NumberKeys"
        "View_Tab_Cycling:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestViewSwitchingPTY_TabCycling"
        "View_State_Validation:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestViewSwitchingPTY_StateValidation"
    )

    local passed=0
    local failed=0

    for test_spec in "${view_tests[@]}"; do
        IFS=':' read -r test_name test_command <<< "$test_spec"

        if run_test "$test_name" "$test_command" "ViewSwitching"; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    log "View Switching Tests: $passed passed, $failed failed"
    return $failed
}

# Run command palette tests
run_command_palette_tests() {
    if [[ "$PTY_AVAILABLE" != "true" ]]; then
        log_warning "PTY layer not available, skipping command palette tests"
        return 0
    fi

    log "Running Command Palette Tests..."

    local command_tests=(
        "Command_Palette_Open:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestCommandPalettePTY_Open"
        "Command_Input_Execute:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestCommandPalettePTY_InputExecute"
        "Command_Window_State:BRUMMER_BINARY=$BRUMMER_BINARY go test -tags=tty,integration ./test/tui/integration -v -run TestCommandPalettePTY_WindowState"
    )

    local passed=0
    local failed=0

    for test_spec in "${command_tests[@]}"; do
        IFS=':' read -r test_name test_command <<< "$test_spec"

        if run_test "$test_name" "$test_command" "CommandPalette"; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    log "Command Palette Tests: $passed passed, $failed failed"
    return $failed
}

# Generate test reports
generate_reports() {
    if [[ "$GENERATE_REPORTS" != "true" ]]; then
        log_info "Report generation disabled"
        return 0
    fi

    log "Generating test reports..."

    local report_file="$TEST_RESULTS_DIR/test_report_$(date +%Y%m%d_%H%M%S).json"
    local summary_file="$TEST_RESULTS_DIR/test_summary.txt"

    # Collect all result files
    local result_files=("$TEST_RESULTS_DIR"/*.json)

    # Generate JSON report
    echo "{" > "$report_file"
    echo "  \"test_run\": {" >> "$report_file"
    echo "    \"timestamp\": \"$(date -Iseconds)\"," >> "$report_file"
    echo "    \"brummer_binary\": \"$BRUMMER_BINARY\"," >> "$report_file"
    echo "    \"test_timeout\": \"$TEST_TIMEOUT\"," >> "$report_file"
    echo "    \"pty_available\": $PTY_AVAILABLE," >> "$report_file"
    echo "    \"environment\": {" >> "$report_file"
    echo "      \"go_version\": \"$(go version)\"," >> "$report_file"
    echo "      \"os\": \"$(uname -s)\"," >> "$report_file"
    echo "      \"arch\": \"$(uname -m)\"," >> "$report_file"
    echo "      \"working_directory\": \"$PROJECT_ROOT\"" >> "$report_file"
    echo "    }" >> "$report_file"
    echo "  }," >> "$report_file"
    echo "  \"test_results\": [" >> "$report_file"

    local first=true
    local total_passed=0
    local total_failed=0

    for result_file in "${result_files[@]}"; do
        if [[ -f "$result_file" && "$result_file" != "$report_file" ]]; then
            if [[ "$first" == "true" ]]; then
                first=false
            else
                echo "," >> "$report_file"
            fi
            cat "$result_file" >> "$report_file"

            # Count results
            local status=$(jq -r '.status' "$result_file" 2>/dev/null || echo "UNKNOWN")
            if [[ "$status" == "PASS" ]]; then
                ((total_passed++))
            elif [[ "$status" == "FAIL" ]]; then
                ((total_failed++))
            fi
        fi
    done

    echo "" >> "$report_file"
    echo "  ]," >> "$report_file"
    echo "  \"summary\": {" >> "$report_file"
    echo "    \"total_tests\": $((total_passed + total_failed))," >> "$report_file"
    echo "    \"passed\": $total_passed," >> "$report_file"
    echo "    \"failed\": $total_failed," >> "$report_file"
    echo "    \"success_rate\": $(echo "scale=2; $total_passed * 100 / ($total_passed + $total_failed)" | bc -l 2>/dev/null || echo "0")" >> "$report_file"
    echo "  }" >> "$report_file"
    echo "}" >> "$report_file"

    # Generate text summary
    cat > "$summary_file" << EOF
TUI Integration Test Summary
============================
Generated: $(date)
Binary: $BRUMMER_BINARY
PTY Available: $PTY_AVAILABLE

Test Results:
- Total Tests: $((total_passed + total_failed))
- Passed: $total_passed
- Failed: $total_failed
- Success Rate: $(echo "scale=1; $total_passed * 100 / ($total_passed + total_failed)" | bc -l 2>/dev/null || echo "0")%

Failed Tests:
EOF

    # Add failed test details
    for result_file in "${result_files[@]}"; do
        if [[ -f "$result_file" && "$result_file" != "$report_file" ]]; then
            local status=$(jq -r '.status' "$result_file" 2>/dev/null || echo "UNKNOWN")
            if [[ "$status" == "FAIL" ]]; then
                local test_name=$(jq -r '.test_name' "$result_file" 2>/dev/null || echo "Unknown")
                local error_summary=$(jq -r '.error_summary' "$result_file" 2>/dev/null || echo "No error summary")
                echo "- $test_name: $error_summary" >> "$summary_file"
            fi
        fi
    done

    log_success "Test reports generated:"
    log_info "  JSON Report: $report_file"
    log_info "  Summary: $summary_file"

    # Display summary
    if [[ -f "$summary_file" ]]; then
        echo ""
        cat "$summary_file"
    fi
}

# Cleanup function
cleanup() {
    if [[ "$KEEP_LOGS" != "true" ]]; then
        log_info "Cleaning up temporary files..."
        # Keep only recent logs
        find "$TEST_LOGS_DIR" -name "*.log" -type f -mtime +7 -delete 2>/dev/null || true
    fi
}

# Main execution function
main() {
    log "Starting TUI Integration Test Execution"
    log "Configuration: Binary=$BRUMMER_BINARY, Timeout=$TEST_TIMEOUT, Verbose=$VERBOSE"

    local total_failed=0

    # Change to project directory
    cd "$PROJECT_ROOT"

    # Setup
    parse_args "$@"
    validate_environment
    setup_test_environment

    # Run tests based on categories
    for category in $TEST_CATEGORIES; do
        case $category in
            All)
                log "Running ALL test categories..."
                run_contract_tests || ((total_failed+=$?))
                run_integration_tests || ((total_failed+=$?))
                ;;
            Contract)
                run_contract_tests || ((total_failed+=$?))
                ;;
            Integration)
                run_integration_tests || ((total_failed+=$?))
                ;;
            PTY)
                run_pty_tests || ((total_failed+=$?))
                ;;
            Keyboard)
                run_keyboard_tests || ((total_failed+=$?))
                ;;
            ViewSwitching)
                run_view_switching_tests || ((total_failed+=$?))
                ;;
            CommandPalette)
                run_command_palette_tests || ((total_failed+=$?))
                ;;
            *)
                log_error "Unknown test category: $category"
                exit 2
                ;;
        esac
    done

    # Generate reports
    generate_reports

    # Cleanup
    cleanup

    # Final result
    echo ""
    if [[ $total_failed -eq 0 ]]; then
        log_success "🎉 All tests passed!"
        exit 0
    else
        log_error "❌ $total_failed test(s) failed!"
        exit 1
    fi
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi