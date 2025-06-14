#!/bin/bash

set -e

echo "🐳 Starting Brummer Package Deployment Tests with Podman"
echo "=================================================="

# Change to the project root directory
cd "$(dirname "$0")/.."

# List of test containers to build and run
TESTS=(
    "npm-node18:Test NPM package on Node.js 18"
    "npm-node20:Test NPM package on Node.js 20" 
    "ubuntu-npm:Test NPM package on Ubuntu"
    "alpine-pnpm:Test PNPM package on Alpine"
    "go-install:Test Go install method"
    "cross-platform:Test cross-platform binary installation"
)

# Function to run a single test
run_test() {
    local test_name="$1"
    local description="$2"
    local dockerfile="docker-tests/Dockerfile.${test_name}"
    
    echo ""
    echo "🧪 Running: $description"
    echo "───────────────────────────────────────────────"
    
    # Build the image
    echo "📦 Building container image..."
    podman build -f "$dockerfile" -t "brummer-test-${test_name}" .
    
    # Run the test
    echo "🚀 Running test container..."
    if podman run --rm "brummer-test-${test_name}"; then
        echo "✅ Test passed: $test_name"
    else
        echo "❌ Test failed: $test_name"
        return 1
    fi
}

# Function to cleanup images
cleanup_images() {
    echo ""
    echo "🧹 Cleaning up test images..."
    for test in "${TESTS[@]}"; do
        test_name="${test%%:*}"
        podman rmi "brummer-test-${test_name}" 2>/dev/null || true
    done
}

# Trap to cleanup on exit
trap cleanup_images EXIT

# Check if podman is available
if ! command -v podman &> /dev/null; then
    echo "❌ Podman is not installed. Please install Podman to run these tests."
    echo "   On Ubuntu/Debian: sudo apt install podman"
    echo "   On RHEL/CentOS: sudo dnf install podman"
    echo "   On macOS: brew install podman"
    exit 1
fi

echo "📋 Found $(echo ${#TESTS[@]}) test scenarios"
echo ""

# Run all tests
failed_tests=()
passed_tests=()

for test in "${TESTS[@]}"; do
    test_name="${test%%:*}"
    description="${test#*:}"
    
    if run_test "$test_name" "$description"; then
        passed_tests+=("$test_name")
    else
        failed_tests+=("$test_name")
    fi
done

# Summary
echo ""
echo "📊 Test Results Summary"
echo "======================"
echo "✅ Passed: ${#passed_tests[@]}"
echo "❌ Failed: ${#failed_tests[@]}"

if [ ${#passed_tests[@]} -gt 0 ]; then
    echo ""
    echo "Passed tests:"
    for test in "${passed_tests[@]}"; do
        echo "  ✅ $test"
    done
fi

if [ ${#failed_tests[@]} -gt 0 ]; then
    echo ""
    echo "Failed tests:"
    for test in "${failed_tests[@]}"; do
        echo "  ❌ $test"
    done
    echo ""
    echo "❌ Some tests failed. Check the logs above for details."
    exit 1
else
    echo ""
    echo "🎉 All tests passed! Package deployment is working correctly."
fi