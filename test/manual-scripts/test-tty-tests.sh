#!/bin/bash

# AI Coder TTY Test Runner
# This script runs the AI coder test suite with proper TTY support

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERBOSE=false
TIMEOUT="60s"
TEST_PATTERN=""
RUN_TTY=false
RUN_HEADLESS=false

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Help function
show_help() {
    cat << EOF
AI Coder TTY Test Runner

Usage: $0 [OPTIONS] [COMMAND]

COMMANDS:
    all             Run all tests (TTY + headless) [default]
    tty             Run only TTY-dependent tests
    headless        Run only headless tests (no TTY)
    help            Show this help message

OPTIONS:
    -v, --verbose   Enable verbose output
    -t, --timeout   Test timeout (default: 60s)
    -p, --pattern   Run specific test pattern
    -b, --binary    Path to brummer binary (default: \$BRUMMER_BINARY or ./brum)
    --no-tty        Skip TTY tests
    --no-headless   Skip headless tests

EXAMPLES:
    $0                          # Run all tests
    $0 tty                      # Run only TTY tests
    $0 headless                 # Run only headless tests
    $0 -v -p TestAICoder        # Run specific test pattern with verbose output
    $0 --timeout 120s           # Run with longer timeout

ENVIRONMENT VARIABLES:
    BRUMMER_BINARY    Path to brummer binary
    FORCE_TTY         Force TTY allocation for tests

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -p|--pattern)
            TEST_PATTERN="$2"
            shift 2
            ;;
        -b|--binary)
            BINARY_PATH="$2"
            shift 2
            ;;
        --no-tty)
            RUN_TTY=false
            shift
            ;;
        --no-headless)
            RUN_HEADLESS=false
            shift
            ;;
        tty)
            RUN_TTY=true
            RUN_HEADLESS=false
            shift
            ;;
        headless)
            RUN_TTY=false
            RUN_HEADLESS=true
            shift
            ;;
        all)
            RUN_TTY=true
            RUN_HEADLESS=true
            shift
            ;;
        help|--help|-h)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Set defaults if not specified
if [[ "$RUN_TTY" == false && "$RUN_HEADLESS" == false ]]; then
    RUN_TTY=true
    RUN_HEADLESS=true
fi

# Check if brummer binary exists
if [[ -z "$BINARY_PATH" ]]; then
    if [[ -n "$BRUMMER_BINARY" ]]; then
        BINARY_PATH="$BRUMMER_BINARY"
    else
        BINARY_PATH="$PROJECT_ROOT/brum"
    fi
fi

if [[ ! -f "$BINARY_PATH" ]]; then
    echo -e "${RED}Error: Brummer binary not found at: $BINARY_PATH${NC}"
    echo -e "${YELLOW}Please build brummer first: make build${NC}"
    echo -e "${YELLOW}Or set BRUMMER_BINARY environment variable${NC}"
    exit 1
fi

echo -e "${BLUE}AI Coder TTY Test Runner${NC}"
echo -e "${BLUE}Binary: $BINARY_PATH${NC}"
echo -e "${BLUE}Timeout: $TIMEOUT${NC}"
echo -e "${BLUE}Verbose: $VERBOSE${NC}"

# Build test runner
echo -e "${YELLOW}Building test runner...${NC}"
cd "$PROJECT_ROOT"
go build -o /tmp/tty-test-runner test/tty/test_runner.go

if [[ $? -ne 0 ]]; then
    echo -e "${RED}Failed to build test runner${NC}"
    exit 1
fi

# Prepare environment
export BRUMMER_BINARY="$BINARY_PATH"
export FORCE_TTY=1

# Build test command
TEST_CMD=("/tmp/tty-test-runner")
TEST_CMD+=("--binary" "$BINARY_PATH")
TEST_CMD+=("--timeout" "$TIMEOUT")

if [[ "$VERBOSE" == true ]]; then
    TEST_CMD+=("--verbose")
fi

if [[ -n "$TEST_PATTERN" ]]; then
    TEST_CMD+=("--run" "$TEST_PATTERN")
fi

if [[ "$RUN_TTY" == true ]]; then
    TEST_CMD+=("--tty")
fi

if [[ "$RUN_HEADLESS" == true ]]; then
    TEST_CMD+=("--headless")
fi

# Run tests
echo -e "${YELLOW}Running tests...${NC}"
echo -e "${BLUE}Command: ${TEST_CMD[*]}${NC}"
echo

if "${TEST_CMD[@]}"; then
    echo -e "${GREEN}✅ All tests passed successfully!${NC}"
    EXIT_CODE=0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    EXIT_CODE=1
fi

# Cleanup
rm -f /tmp/tty-test-runner

exit $EXIT_CODE