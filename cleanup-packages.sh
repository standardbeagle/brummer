#!/bin/bash

echo "🧹 Removing package manager directories and files..."

# Remove package manager directories
rm -rf chocolatey
rm -rf homebrew  
rm -rf winget
rm -rf packages

# Remove Windows signing documentation
rm -f docs/WINDOWS_SIGNING.md

echo "✅ Cleanup complete!"
echo ""
echo "📋 Remaining distribution methods:"
echo "   ✅ NPM package: @standardbeagle/brum"
echo "   ✅ Go install: github.com/standardbeagle/brummer/cmd/brum@latest"
echo "   ✅ Quick install scripts: quick-install.sh / quick-install.ps1"