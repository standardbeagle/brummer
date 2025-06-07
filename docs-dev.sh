#!/bin/bash

# Script to run documentation site locally

echo "🐝 Starting Brummer documentation site..."

cd docs-site

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

# Start the development server
echo "🚀 Starting development server..."
echo "📍 Documentation will be available at http://localhost:3000"
echo ""

npm start