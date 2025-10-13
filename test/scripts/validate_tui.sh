#!/bin/bash

# TUI Functionality Validation Script
# Provides comprehensive validation of TUI functionality beyond basic testing
# This script validates that the TUI application behaves correctly in various scenarios

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
VALIDATION_RESULTS_DIR="$PROJECT_ROOT/test/validation"
VALIDATION_LOGS_DIR="$PROJECT_ROOT/test/logs/validation"

# Default values
BRUMMER_BINARY="${BRUMMER_BINARY:-$PROJECT_ROOT/brum}"
VALIDATION_TIMEOUT="${VALIDATION_TIMEOUT:-60s}"
VERBOSE="${VERBOSE:-false}"
STRICT_MODE="${STRICT_MODE:-false}"
PERFORMANCE_CHECK="${PERFORMANCE_CHECK:-true}"
PTY_AVAILABLE=false

# Ensure directories exist
mkdir -p "$VALIDATION_RESULTS_DIR" "$VALIDATION_LOGS_DIR"

# Validation results tracking
VALIDATION_RESULTS=()
VALIDATION_PASSED=0
VALIDATION_FAILED=0
VALIDATION_WARNINGS=0

# Utility functions
log() {
    echo -e "${BLUE}[VALIDATION] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[VALIDATION] ✓ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[VALIDATION] ⚠ $1${NC}"
    ((VALIDATION_WARNINGS++))
}

log_error() {
    echo -e "${RED}[VALIDATION] ✗ $1${NC}"
    ((VALIDATION_WARNINGS++))
}

log_info() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${WHITE}[VALIDATION] ℹ $1${NC}"
    fi
}

show_usage() {
    cat << EOF
TUI Functionality Validation Script

USAGE:
    $0 [OPTIONS] [VALIDATION_TYPES...]

OPTIONS:
    -b, --binary PATH        Path to brummer binary (default: ./brum)
    -t, --timeout DURATION   Validation timeout (default: 60s)
    -v, --verbose            Enable verbose output
    -s, --strict             Enable strict mode (warnings become failures)
    -p, --no-performance     Skip performance checks
    -h, --help              Show this help message

VALIDATION TYPES:
    All                   Run all validations (default)
    Startup               Validate application startup
    Keyboard              Validate keyboard input handling
    Navigation            Validate view navigation
    State                 Validate state management
    Performance           Validate performance characteristics
    PTY                   Validate PTY layer functionality
    Integration           Validate integration points

EXAMPLES:
    $0                                    # Run all validations
    $0 -v -s Keyboard Navigation          # Run keyboard and navigation validations strictly
    $0 -p Startup Performance             # Run startup and performance without performance checks
    $0 --timeout 120s All                 # Run all validations with extended timeout

ENVIRONMENT VARIABLES:
    BRUMMER_BINARY        Path to brummer binary
    VALIDATION_TIMEOUT    Default validation timeout
    VERBOSE               Enable verbose output (true/false)
    STRICT_MODE           Enable strict mode (true/false)
    PERFORMANCE_CHECK     Run performance checks (true/false)

EXIT CODES:
    0                     All validations passed
    1                     Some validations failed
    2                     Invalid arguments or configuration
    3                     Brummer binary not found
    4                     Required dependencies missing

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
                VALIDATION_TIMEOUT="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -s|--strict)
                STRICT_MODE="true"
                shift
                ;;
            -p|--no-performance)
                PERFORMANCE_CHECK="false"
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            All|Startup|Keyboard|Navigation|State|Performance|PTY|Integration)
                if [[ -z "$VALIDATION_TYPES" ]]; then
                    VALIDATION_TYPES="$1"
                else
                    VALIDATION_TYPES="$VALIDATION_TYPES $1"
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

    # Default to all validations if none specified
    if [[ -z "$VALIDATION_TYPES" ]]; then
        VALIDATION_TYPES="All"
    fi
}

# Record validation result
record_validation() {
    local validation_name="$1"
    local status="$2"
    local message="$3"
    local details="$4"

    local result="{\"name\":\"$validation_name\",\"status\":\"$status\",\"message\":\"$message\",\"details\":\"$details\",\"timestamp\":\"$(date -Iseconds)\"}"
    VALIDATION_RESULTS+=("$result")

    case "$status" in
        "PASS")
            ((VALIDATION_PASSED++))
            log_success "$validation_name: $message"
            ;;
        "FAIL")
            ((VALIDATION_FAILED++))
            log_error "$validation_name: $message"
            ;;
        "WARN")
            ((VALIDATION_WARNINGS++))
            log_warning "$validation_name: $message"
            ;;
    esac
}

# Validate application startup
validate_startup() {
    log "Validating application startup..."

    local startup_log="$VALIDATION_LOGS_DIR/startup_validation.log"
    local startup_time=0
    local startup_success=false

    # Measure startup time
    local start_time=$(date +%s%N)

    if timeout 10s "$BRUMMER_BINARY" --no-tui --help > "$startup_log" 2>&1; then
        local end_time=$(date +%s%N)
        startup_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

        if [[ $startup_time -lt 5000 ]]; then
            record_validation "Startup_Time" "PASS" "Application starts in ${startup_time}ms" "Startup time: ${startup_time}ms"
            startup_success=true
        else
            record_validation "Startup_Time" "WARN" "Slow startup: ${startup_time}ms (target: <5000ms)" "Startup time: ${startup_time}ms"
            startup_success=true
        fi

        # Check for help output
        if grep -q "Usage\|help\|options" "$startup_log"; then
            record_validation "Startup_Help" "PASS" "Help system functional" "Help output detected"
        else
            record_validation "Startup_Help" "FAIL" "Help system not working" "No help output found"
            startup_success=false
        fi

        # Check for error messages
        if grep -qi "error\|fail\|panic" "$startup_log"; then
            record_validation "Startup_Errors" "WARN" "Error messages detected in startup" "Check startup log for details"
        else
            record_validation "Startup_Errors" "PASS" "Clean startup with no errors" "No error messages detected"
        fi

    else
        record_validation "Startup_Execution" "FAIL" "Application failed to start" "Timeout or error during startup"
    fi

    # TUI startup validation (if PTY available)
    if [[ "$PTY_AVAILABLE" == "true" ]]; then
        log_info "Testing TUI startup with PTY..."

        if timeout 5s go test -tags=tty,integration ./test/tui/integration -v -run TestTUIStartup_Basic > "$VALIDATION_LOGS_DIR/tui_startup.log" 2>&1; then
            record_validation "TUI_Startup" "PASS" "TUI starts successfully with PTY" "PTY-based TUI startup validated"
        else
            record_validation "TUI_Startup" "FAIL" "TUI startup failed" "Check TUI startup log"
        fi
    else
        record_validation "TUI_Startup" "WARN" "PTY not available, cannot validate TUI startup" "PTY layer required for TUI validation"
    fi

    return $([[ "$startup_success" == "true" ]] && echo 0 || echo 1)
}

# Validate keyboard input handling
validate_keyboard() {
    log "Validating keyboard input handling..."

    if [[ "$PTY_AVAILABLE" != "true" ]]; then
        record_validation "Keyboard_Input" "WARN" "PTY not available, skipping keyboard validation" "PTY layer required"
        return 0
    fi

    # Test basic keyboard input simulation through PTY
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_Simulation > "$VALIDATION_LOGS_DIR/keyboard_simulation.log" 2>&1; then
        record_validation "Keyboard_Simulation" "PASS" "Keyboard input simulation works" "PTY-based input validation passed"
    else
        record_validation "Keyboard_Simulation" "FAIL" "Keyboard input simulation failed" "Check keyboard simulation validation log"
    fi

    # Test complex keyboard sequences and combinations
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_ComplexSequences > "$VALIDATION_LOGS_DIR/keyboard_complex.log" 2>&1; then
        record_validation "Keyboard_Complex" "PASS" "Complex keyboard sequences work" "Complex input validation passed"
    else
        record_validation "Keyboard_Complex" "WARN" "Complex keyboard sequences issues detected" "Check keyboard complex validation log"
    fi

    # Test input timing and delays
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_TimingAndDelays > "$VALIDATION_LOGS_DIR/keyboard_timing.log" 2>&1; then
        record_validation "Keyboard_Timing" "PASS" "Keyboard input timing works" "Input timing validation passed"
    else
        record_validation "Keyboard_Timing" "WARN" "Keyboard input timing issues detected" "Check keyboard timing validation log"
    fi

    # Test special keys and modifiers
    if timeout 10s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_SpecialKeys > "$VALIDATION_LOGS_DIR/keyboard_special.log" 2>&1; then
        record_validation "Keyboard_Special" "PASS" "Special keys work" "Special keys validation passed"
    else
        record_validation "Keyboard_Special" "WARN" "Special keys issues detected" "Check keyboard special validation log"
    fi

    # Test InputService PTY integration
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_Integration > "$VALIDATION_LOGS_DIR/input_service_pty.log" 2>&1; then
        record_validation "InputService_PTY_Integration" "PASS" "InputService PTY integration works" "InputService PTY integration validated"
    else
        record_validation "InputService_PTY_Integration" "FAIL" "InputService PTY integration failed" "Check InputService PTY integration log"
    fi

    # Test InputService key binding integration
    if timeout 10s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_KeyBindingIntegration > "$VALIDATION_LOGS_DIR/input_service_keybinding.log" 2>&1; then
        record_validation "InputService_KeyBinding" "PASS" "InputService key binding works" "Key binding validation passed"
    else
        record_validation "InputService_KeyBinding" "WARN" "InputService key binding issues detected" "Check key binding validation log"
    fi

    # Test InputService handler integration
    if timeout 10s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_HandlerIntegration > "$VALIDATION_LOGS_DIR/input_service_handler.log" 2>&1; then
        record_validation "InputService_Handler" "PASS" "InputService handler integration works" "Handler integration validated"
    else
        record_validation "InputService_Handler" "WARN" "InputService handler issues detected" "Check handler validation log"
    fi

    # Test concurrent keyboard input access
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_ConcurrentAccess > "$VALIDATION_LOGS_DIR/keyboard_concurrent.log" 2>&1; then
        record_validation "Keyboard_Concurrent" "PASS" "Concurrent keyboard access works" "Concurrent access validation passed"
    else
        record_validation "Keyboard_Concurrent" "WARN" "Concurrent keyboard access issues detected" "Check concurrent access validation log"
    fi

    # Test keyboard input performance
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_Performance > "$VALIDATION_LOGS_DIR/keyboard_performance.log" 2>&1; then
        record_validation "Keyboard_Performance" "PASS" "Keyboard input performance acceptable" "Keyboard performance validated"
    else
        record_validation "Keyboard_Performance" "WARN" "Keyboard input performance issues detected" "Check keyboard performance validation log"
    fi

    # Validate key mapping consistency through harness
    if timeout 10s go test -tags=tty,integration ./test/tui/harness -v -run TestInputSimulator_KeyMapping > "$VALIDATION_LOGS_DIR/key_mapping_validation.log" 2>&1; then
        record_validation "Key_Mapping" "PASS" "Key mapping is consistent" "Key mapping validation passed"
    else
        record_validation "Key_Mapping" "WARN" "Key mapping issues detected" "Check key mapping validation log"
    fi

    return 0
}

# Validate view navigation
validate_navigation() {
    log "Validating view navigation..."

    if [[ "$PTY_AVAILABLE" != "true" ]]; then
        record_validation "View_Navigation" "WARN" "PTY not available, skipping navigation validation" "PTY layer required"
        return 0
    fi

    # Test view switching
    local navigation_tests=("NumberKeys" "TabCycling" "StateValidation")

    for test_type in "${navigation_tests[@]}"; do
        log_info "Testing view navigation: $test_type"

        if timeout 10s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run "TestViewSwitchingPTY_${test_type}" > "$VALIDATION_LOGS_DIR/navigation_${test_type,,}.log" 2>&1; then
            record_validation "Navigation_${test_type}" "PASS" "View navigation ($test_type) works" "Navigation validation passed"
        else
            record_validation "Navigation_${test_type}" "FAIL" "View navigation ($test_type) failed" "Check navigation_${test_type,,} validation log"
        fi
    done

    # Validate navigation service integration
    if timeout 5s go test -tags=tty,integration ./test/tui/integration -v -run TestNavigationService_Integration > "$VALIDATION_LOGS_DIR/navigation_service_validation.log" 2>&1; then
        record_validation "Navigation_Service" "PASS" "Navigation service integration works" "Service integration validated"
    else
        record_validation "Navigation_Service" "FAIL" "Navigation service integration failed" "Check navigation service validation log"
    fi

    # Validate view state consistency
    if timeout 5s go test -tags=tty,integration ./test/tui/harness -v -run TestStateValidator_ViewConsistency > "$VALIDATION_LOGS_DIR/view_consistency_validation.log" 2>&1; then
        record_validation "View_Consistency" "PASS" "View state consistency maintained" "State consistency validated"
    else
        record_validation "View_Consistency" "FAIL" "View state consistency issues" "Check view consistency validation log"
    fi

    return 0
}

# Validate state management
validate_state() {
    log "Validating state management..."

    # Test state capture and validation
    if timeout 5s go test -tags=tty,integration ./test/tui/harness -v -run TestStateValidator_CaptureAndValidate > "$VALIDATION_LOGS_DIR/state_capture_validation.log" 2>&1; then
        record_validation "State_Capture" "PASS" "State capture and validation works" "State management validated"
    else
        record_validation "State_Capture" "FAIL" "State capture and validation failed" "Check state capture validation log"
    fi

    # Test state comparison algorithms
    if timeout 5s go test -tags=tty,integration ./test/tui/harness -v -run TestStateValidator_Comparison > "$VALIDATION_LOGS_DIR/state_comparison_validation.log" 2>&1; then
        record_validation "State_Comparison" "PASS" "State comparison algorithms work" "State comparison validated"
    else
        record_validation "State_Comparison" "FAIL" "State comparison issues detected" "Check state comparison validation log"
    fi

    # Test expected state management
    if timeout 5s go test -tags=tty,integration ./test/tui/harness -v -run TestExpectedState_Management > "$VALIDATION_LOGS_DIR/expected_state_validation.log" 2>&1; then
        record_validation "Expected_State" "PASS" "Expected state management works" "Expected state validated"
    else
        record_validation "Expected_State" "FAIL" "Expected state management failed" "Check expected state validation log"
    fi

    # Test model state consistency
    if timeout 5s go test -tags=tty,integration ./test/tui -v -run TestModelState_Consistency > "$VALIDATION_LOGS_DIR/model_state_validation.log" 2>&1; then
        record_validation "Model_State" "PASS" "Model state consistency maintained" "Model state validated"
    else
        record_validation "Model_State" "FAIL" "Model state consistency issues" "Check model state validation log"
    fi

    return 0
}

# Validate performance characteristics
validate_performance() {
    if [[ "$PERFORMANCE_CHECK" != "true" ]]; then
        log_info "Performance checks disabled"
        return 0
    fi

    log "Validating performance characteristics..."

    # Test PTY allocation performance
    if timeout 15s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_Performance > "$VALIDATION_LOGS_DIR/performance_pty_allocation.log" 2>&1; then
        record_validation "Performance_PTY_Allocation" "PASS" "PTY allocation performance acceptable" "PTY allocation performance validated"
    else
        record_validation "Performance_PTY_Allocation" "WARN" "PTY allocation performance issues detected" "Check PTY allocation performance log"
    fi

    # Test input simulation performance
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_Performance > "$VALIDATION_LOGS_DIR/performance_input_simulation.log" 2>&1; then
        record_validation "Performance_Input_Simulation" "PASS" "Input simulation performance acceptable" "Input simulation performance validated"
    else
        record_validation "Performance_Input_Simulation" "WARN" "Input simulation performance issues detected" "Check input simulation performance log"
    fi

    # Test navigation service performance
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestNavigationServicePTY_Performance > "$VALIDATION_LOGS_DIR/performance_navigation.log" 2>&1; then
        record_validation "Performance_Navigation" "PASS" "Navigation performance acceptable" "Navigation performance validated"
    else
        record_validation "Performance_Navigation" "WARN" "Navigation performance issues detected" "Check navigation performance log"
    fi

    # Test keyboard input response time
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestPerformance_InputResponse > "$VALIDATION_LOGS_DIR/performance_input_response.log" 2>&1; then
        record_validation "Performance_Input_Response" "PASS" "Input response time within limits" "Input response performance validated"
    else
        record_validation "Performance_Input_Response" "WARN" "Input response time issues detected" "Check input response performance log"
    fi

    # Test view switching performance
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestPerformance_ViewSwitching > "$VALIDATION_LOGS_DIR/performance_view_switching.log" 2>&1; then
        record_validation "Performance_View_Switching" "PASS" "View switching performance within limits" "View switching performance validated"
    else
        record_validation "Performance_View_Switching" "WARN" "View switching performance issues detected" "Check view switching performance log"
    fi

    # Test command palette performance
    if timeout 15s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestPerformance_CommandPalette > "$VALIDATION_LOGS_DIR/performance_command_palette.log" 2>&1; then
        record_validation "Performance_Command_Palette" "PASS" "Command palette performance acceptable" "Command palette performance validated"
    else
        record_validation "Performance_Command_Palette" "WARN" "Command palette performance issues detected" "Check command palette performance log"
    fi

    # Test PTY memory usage patterns
    if timeout 15s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_MemoryUsage > "$VALIDATION_LOGS_DIR/performance_pty_memory.log" 2>&1; then
        record_validation "Performance_PTY_Memory" "PASS" "PTY memory usage patterns acceptable" "PTY memory performance validated"
    else
        record_validation "Performance_PTY_Memory" "WARN" "PTY memory usage issues detected" "Check PTY memory performance log"
    fi

    # Test TUI memory usage patterns
    if timeout 15s go test -tags=tty,integration ./test/tui -v -run TestPerformance_MemoryUsage > "$VALIDATION_LOGS_DIR/performance_tui_memory.log" 2>&1; then
        record_validation "Performance_TUI_Memory" "PASS" "TUI memory usage patterns acceptable" "TUI memory performance validated"
    else
        record_validation "Performance_TUI_Memory" "WARN" "TUI memory usage issues detected" "Check TUI memory performance log"
    fi

    # Test concurrent access performance
    if timeout 20s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestPerformance_ConcurrentAccess > "$VALIDATION_LOGS_DIR/performance_concurrent.log" 2>&1; then
        record_validation "Performance_Concurrent" "PASS" "Concurrent access performance acceptable" "Concurrent performance validated"
    else
        record_validation "Performance_Concurrent" "WARN" "Concurrent access performance issues detected" "Check concurrent performance log"
    fi

    # Test PTY layer throughput
    if timeout 15s go test -tags=tty,integration ./test/tui/pty -v -run TestPerformance_Throughput > "$VALIDATION_LOGS_DIR/performance_throughput.log" 2>&1; then
        record_validation "Performance_Throughput" "PASS" "PTY layer throughput acceptable" "Throughput performance validated"
    else
        record_validation "Performance_Throughput" "WARN" "PTY layer throughput issues detected" "Check throughput performance log"
    fi

    # Test state validation performance
    if timeout 10s go test -tags=tty,integration ./test/tui/harness -v -run TestPerformance_StateValidation > "$VALIDATION_LOGS_DIR/performance_state_validation.log" 2>&1; then
        record_validation "Performance_State_Validation" "PASS" "State validation performance acceptable" "State validation performance validated"
    else
        record_validation "Performance_State_Validation" "WARN" "State validation performance issues detected" "Check state validation performance log"
    fi

    # Test test harness performance
    if timeout 15s go test -tags=tty,integration ./test/tui/harness -v -run TestPerformance_TestHarness > "$VALIDATION_LOGS_DIR/performance_test_harness.log" 2>&1; then
        record_validation "Performance_Test_Harness" "PASS" "Test harness performance acceptable" "Test harness performance validated"
    else
        record_validation "Performance_Test_Harness" "WARN" "Test harness performance issues detected" "Check test harness performance log"
    fi

    return 0
}

# Validate PTY layer functionality
validate_pty() {
    log "Validating PTY layer functionality..."

    # Test PTY allocation and basic functionality
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality > "$VALIDATION_LOGS_DIR/pty_allocation.log" 2>&1; then
        record_validation "PTY_Allocation" "PASS" "PTY allocation works" "PTY allocation validated"
        PTY_AVAILABLE=true
    else
        record_validation "PTY_Allocation" "FAIL" "PTY allocation failed" "PTY layer not functional"
        PTY_AVAILABLE=false
        return 1
    fi

    # Test PTY input/output operations
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_InputOutputOperations > "$VALIDATION_LOGS_DIR/pty_io.log" 2>&1; then
        record_validation "PTY_IO" "PASS" "PTY input/output works" "PTY I/O validated"
    else
        record_validation "PTY_IO" "FAIL" "PTY input/output failed" "Check PTY I/O validation log"
    fi

    # Test PTY cleanup and resource management
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_CleanupAndResourceManagement > "$VALIDATION_LOGS_DIR/pty_cleanup.log" 2>&1; then
        record_validation "PTY_Cleanup" "PASS" "PTY cleanup works" "PTY cleanup validated"
    else
        record_validation "PTY_Cleanup" "WARN" "PTY cleanup issues detected" "Check PTY cleanup validation log"
    fi

    # Test PTY size management and terminal emulation
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_TerminalEmulation > "$VALIDATION_LOGS_DIR/pty_terminal.log" 2>&1; then
        record_validation "PTY_Terminal" "PASS" "PTY terminal emulation works" "PTY terminal emulation validated"
    else
        record_validation "PTY_Terminal" "WARN" "PTY terminal emulation issues detected" "Check PTY terminal validation log"
    fi

    # Test PTY error handling and edge cases
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_ErrorHandling > "$VALIDATION_LOGS_DIR/pty_error_handling.log" 2>&1; then
        record_validation "PTY_ErrorHandling" "PASS" "PTY error handling works" "PTY error handling validated"
    else
        record_validation "PTY_ErrorHandling" "WARN" "PTY error handling issues detected" "Check PTY error handling validation log"
    fi

    # Test PTY integration with BubbleTea
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYIntegration_BubbleTeaIntegration > "$VALIDATION_LOGS_DIR/pty_bubbletea.log" 2>&1; then
        record_validation "PTY_BubbleTea" "PASS" "PTY BubbleTea integration works" "PTY BubbleTea integration validated"
    else
        record_validation "PTY_BubbleTea" "WARN" "PTY BubbleTea integration issues detected" "Check PTY BubbleTea validation log"
    fi

    # Test PTY performance characteristics
    if timeout 15s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_Performance > "$VALIDATION_LOGS_DIR/pty_performance.log" 2>&1; then
        record_validation "PTY_Performance" "PASS" "PTY performance acceptable" "PTY performance validated"
    else
        record_validation "PTY_Performance" "WARN" "PTY performance issues detected" "Check PTY performance validation log"
    fi

    # Test PTY memory usage patterns
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_MemoryUsage > "$VALIDATION_LOGS_DIR/pty_memory.log" 2>&1; then
        record_validation "PTY_Memory" "PASS" "PTY memory usage acceptable" "PTY memory usage validated"
    else
        record_validation "PTY_Memory" "WARN" "PTY memory usage issues detected" "Check PTY memory validation log"
    fi

    return 0
}

# Validate integration points
validate_integration() {
    log "Validating integration points..."

    # Test InputService PTY integration
    if timeout 10s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestInputServicePTY_Integration > "$VALIDATION_LOGS_DIR/input_service_integration.log" 2>&1; then
        record_validation "Integration_InputService" "PASS" "InputService PTY integration works" "InputService PTY integration validated"
    else
        record_validation "Integration_InputService" "FAIL" "InputService PTY integration failed" "Check InputService integration log"
    fi

    # Test NavigationService PTY integration
    if timeout 10s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui/integration -v -run TestNavigationServicePTY_Integration > "$VALIDATION_LOGS_DIR/navigation_service_integration.log" 2>&1; then
        record_validation "Integration_NavigationService" "PASS" "NavigationService PTY integration works" "NavigationService PTY integration validated"
    else
        record_validation "Integration_NavigationService" "FAIL" "NavigationService PTY integration failed" "Check NavigationService integration log"
    fi

    # Test NavigationService PTY transition callbacks
    if timeout 10s go test -tags=tty,integration ./test/tui/integration -v -run TestNavigationServicePTY_TransitionCallbacks > "$VALIDATION_LOGS_DIR/navigation_service_callbacks.log" 2>&1; then
        record_validation "Integration_NavigationCallbacks" "PASS" "Navigation callback integration works" "Navigation callback integration validated"
    else
        record_validation "Integration_NavigationCallbacks" "WARN" "Navigation callback integration issues detected" "Check navigation callbacks validation log"
    fi

    # Test NavigationService PTY view order and mappings
    if timeout 10s go test -tags=tty,integration ./test/tui/integration -v -run TestNavigationServicePTY_ViewOrderAndMappings > "$VALIDATION_LOGS_DIR/navigation_service_mappings.log" 2>&1; then
        record_validation "Integration_NavigationMappings" "PASS" "Navigation mapping integration works" "Navigation mapping integration validated"
    else
        record_validation "Integration_NavigationMappings" "WARN" "Navigation mapping integration issues detected" "Check navigation mappings validation log"
    fi

    # Test TestHarness integration with PTY
    if timeout 10s go test -tags=tty,integration ./test/tui/harness -v -run TestTestHarness_PTIntegration > "$VALIDATION_LOGS_DIR/test_harness_integration.log" 2>&1; then
        record_validation "Integration_TestHarness" "PASS" "TestHarness PTY integration works" "TestHarness PTY integration validated"
    else
        record_validation "Integration_TestHarness" "WARN" "TestHarness PTY integration issues detected" "Check TestHarness integration log"
    fi

    # Test PTY to BubbleTea integration components
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYIntegration_BubbleTeaIntegration > "$VALIDATION_LOGS_DIR/bubbletea_integration.log" 2>&1; then
        record_validation "Integration_BubbleTea" "PASS" "PTY BubbleTea integration works" "PTY BubbleTea integration validated"
    else
        record_validation "Integration_BubbleTea" "FAIL" "PTY BubbleTea integration failed" "Check BubbleTea integration log"
    fi

    # Test PTY program integration
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYIntegration_ProgramIntegration > "$VALIDATION_LOGS_DIR/pty_program_integration.log" 2>&1; then
        record_validation "Integration_PTYProgram" "PASS" "PTY program integration works" "PTY program integration validated"
    else
        record_validation "Integration_PTYProgram" "WARN" "PTY program integration issues detected" "Check PTY program validation log"
    fi

    # Test service layer PTY integration
    if timeout 10s go test -tags=tty,integration ./test/tui/integration -v -run TestServiceLayerPTYIntegration > "$VALIDATION_LOGS_DIR/service_layer_integration.log" 2>&1; then
        record_validation "Integration_ServiceLayer" "PASS" "Service layer PTY integration works" "Service layer PTY integration validated"
    else
        record_validation "Integration_ServiceLayer" "WARN" "Service layer PTY integration issues detected" "Check service layer validation log"
    fi

    # Test event system PTY integration
    if timeout 10s go test -tags=tty,integration ./test/tui/integration -v -run TestEventSystemPTYIntegration > "$VALIDATION_LOGS_DIR/event_system_integration.log" 2>&1; then
        record_validation "Integration_EventSystem" "PASS" "Event system PTY integration works" "Event system PTY integration validated"
    else
        record_validation "Integration_EventSystem" "WARN" "Event system PTY integration issues detected" "Check event system validation log"
    fi

    # Test state validator PTY integration
    if timeout 10s go test -tags=tty,integration ./test/tui/harness -v -run TestStateValidator_PTIntegration > "$VALIDATION_LOGS_DIR/state_validator_integration.log" 2>&1; then
        record_validation "Integration_StateValidator" "PASS" "State validator PTY integration works" "State validator PTY integration validated"
    else
        record_validation "Integration_StateValidator" "WARN" "State validator PTY integration issues detected" "Check state validator validation log"
    fi

    # Test input simulator PTY integration
    if timeout 10s go test -tags=tty,integration ./test/tui/harness -v -run TestInputSimulator_PTIntegration > "$VALIDATION_LOGS_DIR/input_simulator_integration.log" 2>&1; then
        record_validation "Integration_InputSimulator" "PASS" "Input simulator PTY integration works" "Input simulator PTY integration validated"
    else
        record_validation "Integration_InputSimulator" "WARN" "Input simulator PTY integration issues detected" "Check input simulator validation log"
    fi

    # Test test reporter PTY integration
    if timeout 10s go test -tags=tty,integration ./test/tui/harness -v -run TestTestReporter_PTIntegration > "$VALIDATION_LOGS_DIR/test_reporter_integration.log" 2>&1; then
        record_validation "Integration_TestReporter" "PASS" "Test reporter PTY integration works" "Test reporter PTY integration validated"
    else
        record_validation "Integration_TestReporter" "WARN" "Test reporter PTY integration issues detected" "Check test reporter validation log"
    fi

    # Test comprehensive end-to-end PTY integration
    if timeout 20s BRUMMER_BINARY="$BRUMMER_BINARY" go test -tags=tty,integration ./test/tui -v -run TestEndToEndPTY_Integration > "$VALIDATION_LOGS_DIR/e2e_pty_integration.log" 2>&1; then
        record_validation "Integration_EndToEnd" "PASS" "End-to-end PTY integration works" "E2E PTY integration validated"
    else
        record_validation "Integration_EndToEnd" "WARN" "End-to-end PTY integration issues detected" "Check E2E PTY integration log"
    fi

    return 0
}

# Generate validation report
generate_validation_report() {
    local report_file="$VALIDATION_RESULTS_DIR/validation_report_$(date +%Y%m%d_%H%M%S).json"
    local summary_file="$VALIDATION_RESULTS_DIR/validation_summary.txt"

    log "Generating validation report..."

    # Generate JSON report
    echo "{" > "$report_file"
    echo "  \"validation_run\": {" >> "$report_file"
    echo "    \"timestamp\": \"$(date -Iseconds)\"," >> "$report_file"
    echo "    \"brummer_binary\": \"$BRUMMER_BINARY\"," >> "$report_file"
    echo "    \"validation_timeout\": \"$VALIDATION_TIMEOUT\"," >> "$report_file"
    echo "    \"pty_available\": $PTY_AVAILABLE," >> "$report_file"
    echo "    \"strict_mode\": $STRICT_MODE," >> "$report_file"
    echo "    \"performance_check\": $PERFORMANCE_CHECK" >> "$report_file"
    echo "  }," >> "$report_file"
    echo "  \"validation_results\": [" >> "$report_file"

    local first=true
    for result in "${VALIDATION_RESULTS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo "," >> "$report_file"
        fi
        echo "    $result" >> "$report_file"
    done

    echo "" >> "$report_file"
    echo "  ]," >> "$report_file"
    echo "  \"summary\": {" >> "$report_file"
    echo "    \"total_validations\": ${#VALIDATION_RESULTS[@]}," >> "$report_file"
    echo "    \"passed\": $VALIDATION_PASSED," >> "$report_file"
    echo "    \"failed\": $VALIDATION_FAILED," >> "$report_file"
    echo "    \"warnings\": $VALIDATION_WARNINGS," >> "$report_file"
    echo "    \"success_rate\": $(echo "scale=2; $VALIDATION_PASSED * 100 / ${#VALIDATION_RESULTS[@]}" | bc -l 2>/dev/null || echo "0")" >> "$report_file"
    echo "  }" >> "$report_file"
    echo "}" >> "$report_file"

    # Generate text summary
    cat > "$summary_file" << EOF
TUI Functionality Validation Summary
===================================
Generated: $(date)
Binary: $BRUMMER_BINARY
PTY Available: $PTY_AVAILABLE
Strict Mode: $STRICT_MODE
Performance Checks: $PERFORMANCE_CHECK

Validation Results:
- Total Validations: ${#VALIDATION_RESULTS[@]}
- Passed: $VALIDATION_PASSED
- Failed: $VALIDATION_FAILED
- Warnings: $VALIDATION_WARNINGS
- Success Rate: $(echo "scale=1; $VALIDATION_PASSED * 100 / ${#VALIDATION_RESULTS[@]}" | bc -l 2>/dev/null || echo "0")%

Failed Validations:
EOF

    # Add failed validation details
    for result in "${VALIDATION_RESULTS[@]}"; do
        local status=$(echo "$result" | jq -r '.status' 2>/dev/null || echo "UNKNOWN")
        if [[ "$status" == "FAIL" ]]; then
            local name=$(echo "$result" | jq -r '.name' 2>/dev/null || echo "Unknown")
            local message=$(echo "$result" | jq -r '.message' 2>/dev/null || echo "No message")
            echo "- $name: $message" >> "$summary_file"
        fi
    done

    if [[ $VALIDATION_WARNINGS -gt 0 ]]; then
        echo "" >> "$summary_file"
        echo "Warnings:" >> "$summary_file"
        for result in "${VALIDATION_RESULTS[@]}"; do
            local status=$(echo "$result" | jq -r '.status' 2>/dev/null || echo "UNKNOWN")
            if [[ "$status" == "WARN" ]]; then
                local name=$(echo "$result" | jq -r '.name' 2>/dev/null || echo "Unknown")
                local message=$(echo "$result" | jq -r '.message' 2>/dev/null || echo "No message")
                echo "- $name: $message" >> "$summary_file"
            fi
        done
    fi

    log_success "Validation report generated:"
    log_info "  JSON Report: $report_file"
    log_info "  Summary: $summary_file"

    # Display summary
    if [[ -f "$summary_file" ]]; then
        echo ""
        cat "$summary_file"
    fi
}

# Validate environment
validate_environment() {
    log "Validating environment..."

    # Check if we're in the right directory
    if [[ ! -d "$PROJECT_ROOT/test/tui" ]]; then
        log_error "TUI test directory not found at $PROJECT_ROOT/test/tui"
        exit 4
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
        exit 4
    fi

    log_success "Environment validation completed"
}

# Main execution function
main() {
    log "Starting TUI Functionality Validation"

    # Change to project directory
    cd "$PROJECT_ROOT"

    # Parse arguments and validate environment
    parse_args "$@"
    validate_environment

    # Run validations based on types
    for validation_type in $VALIDATION_TYPES; do
        case $validation_type in
            All)
                log "Running ALL validation types..."
                validate_startup || true
                validate_keyboard || true
                validate_navigation || true
                validate_state || true
                validate_performance || true
                validate_pty || true
                validate_integration || true
                ;;
            Startup)
                validate_startup || true
                ;;
            Keyboard)
                validate_keyboard || true
                ;;
            Navigation)
                validate_navigation || true
                ;;
            State)
                validate_state || true
                ;;
            Performance)
                validate_performance || true
                ;;
            PTY)
                validate_pty || true
                ;;
            Integration)
                validate_integration || true
                ;;
            *)
                log_error "Unknown validation type: $validation_type"
                exit 2
                ;;
        esac
    done

    # Generate report
    generate_validation_report

    # Final result
    echo ""
    local total_issues=$((VALIDATION_FAILED + VALIDATION_WARNINGS))

    if [[ "$STRICT_MODE" == "true" ]]; then
        # In strict mode, warnings count as failures
        total_issues=$((VALIDATION_FAILED + VALIDATION_WARNINGS))
    else
        # In normal mode, only actual failures count
        total_issues=$VALIDATION_FAILED
    fi

    if [[ $total_issues -eq 0 ]]; then
        log_success "🎉 All validations passed!"
        exit 0
    else
        log_error "❌ $total_issues validation(s) failed!"
        exit 1
    fi
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi