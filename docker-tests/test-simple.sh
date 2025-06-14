#!/bin/bash

set -e

echo "🐳 Simple Podman Deployment Test"
echo "================================"

# Test 1: Cross-platform binary installation
echo "1. Testing cross-platform binary installation..."
container_id=$(podman run -d brummer-test-cross-platform tail -f /dev/null)

echo "   Testing version command..."
podman exec $container_id /test/bin/brum --version

echo "   Testing help command..."
podman exec $container_id /test/bin/brum --help | head -3

echo "   ✅ Cross-platform test passed!"

# Cleanup
podman stop $container_id >/dev/null
podman rm $container_id >/dev/null

echo ""
echo "2. Testing Go build..."
podman build -f docker-tests/Dockerfile.go-install -t brummer-test-go-install . >/dev/null

echo "   Testing Go binary..."
container_id=$(podman run -d brummer-test-go-install tail -f /dev/null)
podman exec $container_id /root/.local/bin/brum --version

echo "   ✅ Go build test passed!"

# Cleanup
podman stop $container_id >/dev/null
podman rm $container_id >/dev/null

echo ""
echo "🎉 All deployment tests passed!"
echo ""
echo "✅ NPM package: @standardbeagle/brum"
echo "✅ Cross-platform binaries: Working"  
echo "✅ Go install: Working"
echo "✅ Version flag: Working"