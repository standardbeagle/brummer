#!/bin/bash
echo "Creating a simple test project..."
mkdir -p debug-project
cd debug-project

cat > package.json << 'EOF'
{
  "name": "debug-test",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'Hello from test script!'"
  }
}
EOF

echo "Running brummer..."
../brum