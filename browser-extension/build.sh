#!/bin/bash

# Build script for Brummer Firefox Extension

set -e

echo "Building Brummer Firefox Extension..."

# Check if we're in the right directory
if [ ! -f "manifest.json" ]; then
    echo "Error: manifest.json not found. Run this script from the browser-extension directory."
    exit 1
fi

# Create build directory
BUILD_DIR="build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Copy extension files
echo "Copying extension files..."
cp manifest.json "$BUILD_DIR/"
cp devtools.html "$BUILD_DIR/"
cp devtools.js "$BUILD_DIR/"
cp panel.html "$BUILD_DIR/"
cp panel.js "$BUILD_DIR/"
cp background.js "$BUILD_DIR/"
cp content.js "$BUILD_DIR/"

# Copy icons if they exist
if [ -d "icons" ]; then
    cp -r icons "$BUILD_DIR/"
else
    echo "Warning: icons directory not found. Extension will need icons to work properly."
    mkdir -p "$BUILD_DIR/icons"
    echo "# Icons missing - see icons/README.md for instructions" > "$BUILD_DIR/icons/README.md"
fi

# Create zip package for distribution
PACKAGE_NAME="brummer-firefox-extension.zip"
echo "Creating package: $PACKAGE_NAME"

cd "$BUILD_DIR"
zip -r "../$PACKAGE_NAME" .
cd ..

echo "✅ Extension built successfully!"
echo "📦 Package: $PACKAGE_NAME"
echo ""
echo "To install in Firefox:"
echo "1. Open Firefox"
echo "2. Go to about:debugging"
echo "3. Click 'This Firefox'"
echo "4. Click 'Load Temporary Add-on...'"
echo "5. Select manifest.json from the build/ directory"
echo ""
echo "For permanent installation, submit the .zip file to Firefox Add-ons store."