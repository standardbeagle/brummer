#!/bin/bash

# Create simple placeholder icons for the Brummer Firefox extension
# These are basic yellow squares with black bee text

set -e

echo "Creating placeholder icons for Brummer Firefox Extension..."

# Check if ImageMagick is available
if ! command -v convert &> /dev/null; then
    echo "ImageMagick not found. Creating simple HTML/CSS based icons instead."
    
    # Create a simple SVG icon
    cat > icons/bee.svg << 'EOF'
<svg width="128" height="128" xmlns="http://www.w3.org/2000/svg">
  <rect width="128" height="128" fill="#FFC107"/>
  <text x="64" y="80" font-family="Arial, sans-serif" font-size="80" text-anchor="middle" fill="#000">🐝</text>
</svg>
EOF
    
    echo "Created bee.svg - you can convert this to PNG files manually"
    echo "Or install ImageMagick and run this script again for automatic conversion"
    
else
    echo "ImageMagick found. Creating PNG icons..."
    
    # Create directory if it doesn't exist
    mkdir -p icons
    
    # Create icons with bee emoji (if font supports it)
    convert -size 16x16 xc:'#FFC107' -pointsize 12 -fill black -gravity center -annotate +0+0 '🐝' icons/bee-16.png 2>/dev/null || \
    convert -size 16x16 xc:'#FFC107' -pointsize 8 -fill black -gravity center -annotate +0+0 'B' icons/bee-16.png
    
    convert -size 32x32 xc:'#FFC107' -pointsize 24 -fill black -gravity center -annotate +0+0 '🐝' icons/bee-32.png 2>/dev/null || \
    convert -size 32x32 xc:'#FFC107' -pointsize 16 -fill black -gravity center -annotate +0+0 'B' icons/bee-32.png
    
    convert -size 48x48 xc:'#FFC107' -pointsize 36 -fill black -gravity center -annotate +0+0 '🐝' icons/bee-48.png 2>/dev/null || \
    convert -size 48x48 xc:'#FFC107' -pointsize 24 -fill black -gravity center -annotate +0+0 'B' icons/bee-48.png
    
    convert -size 128x128 xc:'#FFC107' -pointsize 96 -fill black -gravity center -annotate +0+0 '🐝' icons/bee-128.png 2>/dev/null || \
    convert -size 128x128 xc:'#FFC107' -pointsize 64 -fill black -gravity center -annotate +0+0 'B' icons/bee-128.png
    
    echo "✅ Created icon files:"
    ls -la icons/*.png
fi

echo ""
echo "Icons created! You can now build the extension with ./build.sh"