#!/bin/bash

# Brummer Regression Test Runner
# This script builds Brummer and runs the comprehensive regression test suite

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERBOSE=false
BINARY_PATH=""
SKIP_BUILD=false
TEST_FILTER=""

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

Brummer Regression Test Runner

OPTIONS:
    -h, --help          Show this help message
    -v, --verbose       Enable verbose output
    -b, --binary PATH   Use existing binary instead of building
    -s, --skip-build    Skip building, use existing binary in project root
    -f, --filter TERM   Only run tests containing TERM in their name
    
EXAMPLES:
    $0                          # Build and run all tests
    $0 --verbose                # Run with verbose output
    $0 --binary ./my-brum       # Use specific binary
    $0 --skip-build             # Use existing ./brum binary
    $0 --filter MCP             # Only run MCP-related tests

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
            VERBOSE=true
            shift
            ;;
        -b|--binary)
            BINARY_PATH="$2"
            shift 2
            ;;
        -s|--skip-build)
            SKIP_BUILD=true
            shift
            ;;
        -f|--filter)
            TEST_FILTER="$2"
            shift 2
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

print_status "Brummer Regression Test Suite"
print_status "Project root: $PROJECT_ROOT"
print_status "Test directory: $TEST_DIR"

# Determine binary path
if [[ -n "$BINARY_PATH" ]]; then
    print_status "Using provided binary: $BINARY_PATH"
elif [[ "$SKIP_BUILD" == "true" ]]; then
    BINARY_PATH="$PROJECT_ROOT/brum"
    print_status "Using existing binary: $BINARY_PATH"
else
    BINARY_PATH="$PROJECT_ROOT/brum"
    print_status "Will build binary: $BINARY_PATH"
fi

# Build the binary if needed
if [[ "$SKIP_BUILD" != "true" && -z "$2" ]]; then
    print_status "Building Brummer binary..."
    cd "$PROJECT_ROOT"
    
    if ! go build -o brum ./cmd/brum; then
        print_error "Failed to build Brummer binary"
        exit 1
    fi
    
    print_success "Binary built successfully"
fi

# Check if binary exists
if [[ ! -f "$BINARY_PATH" ]]; then
    print_error "Binary not found: $BINARY_PATH"
    exit 1
fi

# Make binary executable
chmod +x "$BINARY_PATH"

# Run the test suite
print_status "Running regression test suite..."
cd "$TEST_DIR"

# Build test arguments
TEST_ARGS=("$BINARY_PATH")
if [[ "$VERBOSE" == "true" ]]; then
    TEST_ARGS+=("--verbose")
fi

# Run the tests
if go run *.go "${TEST_ARGS[@]}"; then
    print_success "All regression tests passed!"
    exit 0
else
    print_error "Some regression tests failed!"
    exit 1
fi
