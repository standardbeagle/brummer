#!/bin/bash
# Test that color extraction is working in AI Coder view

echo "🎨 Testing Color Extraction in AI Coder"
echo "======================================="
echo ""
echo "This test verifies that:"
echo "1. vt10x Cell() extraction works"
echo "2. ANSI color codes are generated"
echo "3. Colors are preserved in screen buffer"
echo "4. Component renders with colors"
echo ""
echo "Test Steps:"
echo "1. Start brum and navigate to AI Coders tab (3)"
echo "2. Start Claude Code: /ai claude"
echo "3. Look for colorized output:"
echo "   - Orange Claude logo (RGB: 215,119,87)"
echo "   - Dim gray separator lines (SGR 2m)"
echo "   - Various colored text"
echo ""
echo "Expected: Full 24-bit RGB colors displayed"
echo "If colors missing: Check terminal supports true color"
echo ""
echo "Terminal color support:"
echo "  TERM=$TERM"
echo "  COLORTERM=$COLORTERM"
echo ""

# Test if terminal supports 24-bit color
echo "Testing 24-bit color support:"
echo -e "\033[38;2;255;0;0mRED\033[0m (RGB 255,0,0)"
echo -e "\033[38;2;0;255;0mGREEN\033[0m (RGB 0,255,0)"
echo -e "\033[38;2;0;0;255mBLUE\033[0m (RGB 0,0,255)"
echo -e "\033[38;2;215;119;87mCLAUDE ORANGE\033[0m (RGB 215,119,87 - like logo)"
echo ""

if [ "$COLORTERM" = "truecolor" ] || [ "$COLORTERM" = "24bit" ]; then
    echo "✅ Terminal supports true color (24-bit)"
else
    echo "⚠️  COLORTERM not set to 'truecolor' or '24bit'"
    echo "   You may not see full RGB colors"
    echo "   Try: export COLORTERM=truecolor"
fi

echo ""
echo "Starting brum..."
echo ""
make build && ./brum
