#!/bin/bash

set -e

echo "🐳 Testing NPM Package Installation with Podman"
echo "=============================================="

# Build a simple test container
cat > /tmp/Dockerfile.npm-simple << 'EOF'
FROM node:18-alpine

# Install basic tools
RUN apk add --no-cache bash

# Create test directory
WORKDIR /test

# Copy package files
COPY package.json install.js index.js ./
COPY bin/ ./bin/
COPY dist/ ./dist/

# Install package locally
RUN npm install .

# Keep container running
CMD ["tail", "-f", "/dev/null"]
EOF

echo "📦 Building test container..."
podman build -f /tmp/Dockerfile.npm-simple -t brummer-npm-test .

echo "🚀 Starting test container..."
container_id=$(podman run -d brummer-npm-test)

echo "🧪 Running tests in container..."
echo ""

echo "1. Testing local installation..."
podman exec $container_id ./bin/brum --version

echo ""
echo "2. Testing help command..."
podman exec $container_id ./bin/brum --help | head -5

echo ""
echo "3. Testing npm bin symlink..."
podman exec $container_id ls -la node_modules/.bin/

echo ""
echo "4. Testing npm bin execution..."
podman exec $container_id node_modules/.bin/brum --version

echo ""
echo "✅ All NPM tests passed!"

echo ""
echo "🧹 Cleaning up..."
podman stop $container_id
podman rm $container_id
podman rmi brummer-npm-test

rm /tmp/Dockerfile.npm-simple