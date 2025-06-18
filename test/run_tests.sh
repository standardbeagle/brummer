#!/bin/bash

# Brummer Test Runner
# This script builds Brummer and runs the test suite using standard Go testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERBOSE=""
SKIP_BUILD=false
TEST_FILTER=""
SHORT_MODE=false
COVERAGE=false

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Brummer Test Runner - Uses Go standard testing

OPTIONS:
    -h, --help          Show this help message
    -v, --verbose       Enable verbose test output
    -s, --skip-build    Skip building, use existing binary
    -f, --filter TERM   Only run tests matching TERM (uses -run flag)
    --short             Run in short mode (skip long tests)
    --coverage          Generate coverage report
    
EXAMPLES:
    $0                          # Build and run all tests
    $0 --verbose                # Run with verbose output
    $0 --skip-build             # Use existing ./brum binary
    $0 --filter TestMCP         # Only run MCP tests
    $0 --short                  # Skip long-running tests
    $0 --coverage               # Generate test coverage

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -s|--skip-build)
            SKIP_BUILD=true
            shift
            ;;
        -f|--filter)
            TEST_FILTER="$2"
            shift 2
            ;;
        --short)
            SHORT_MODE=true
            shift
            ;;
        --coverage)
            COVERAGE=true
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

print_status "Brummer Test Suite"
print_status "Project root: $PROJECT_ROOT"
print_status "Test directory: $TEST_DIR"

# Build the binary if needed
if [[ "$SKIP_BUILD" != "true" ]]; then
    print_status "Building Brummer binary..."
    cd "$PROJECT_ROOT"
    
    if ! go build -o brum ./cmd/brum/; then
        print_error "Failed to build Brummer binary"
        exit 1
    fi
    
    print_success "Binary built successfully"
fi

# Check if binary exists
BINARY_PATH="$PROJECT_ROOT/brum"
if [[ ! -f "$BINARY_PATH" ]]; then
    print_error "Binary not found: $BINARY_PATH"
    print_error "Run without --skip-build to build it"
    exit 1
fi

# Make binary executable
chmod +x "$BINARY_PATH"

# Export binary path for tests
export BRUMMER_BINARY="$BINARY_PATH"

# Build test command
TEST_CMD="go test -tags=integration"

# Add verbose flag
if [[ -n "$VERBOSE" ]]; then
    TEST_CMD="$TEST_CMD $VERBOSE"
fi

# Add test filter
if [[ -n "$TEST_FILTER" ]]; then
    TEST_CMD="$TEST_CMD -run $TEST_FILTER"
fi

# Add short mode
if [[ "$SHORT_MODE" == "true" ]]; then
    TEST_CMD="$TEST_CMD -short"
fi

# Add coverage
if [[ "$COVERAGE" == "true" ]]; then
    TEST_CMD="$TEST_CMD -coverprofile=coverage.out"
fi

# Add timeout
TEST_CMD="$TEST_CMD -timeout 5m"

# Run the tests
print_status "Running tests..."
print_status "Command: $TEST_CMD ./test/..."
cd "$PROJECT_ROOT"

if $TEST_CMD ./test/...; then
    print_success "All tests passed!"
    
    # Show coverage if enabled
    if [[ "$COVERAGE" == "true" ]]; then
        print_status "Coverage report saved to coverage.out"
        print_status "View with: go tool cover -html=coverage.out"
    fi
    
    exit 0
else
    print_error "Some tests failed!"
    exit 1
fi
