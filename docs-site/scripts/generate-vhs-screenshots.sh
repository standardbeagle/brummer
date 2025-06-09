#!/bin/bash

# Script to generate all screenshots using VHS
# This script will FAIL if screenshots cannot be generated properly

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VHS_DIR="$SCRIPT_DIR/vhs"
STATIC_DIR="$SCRIPT_DIR/../static/img"

echo "🎬 Generating screenshots with VHS..."
echo "This will fail if VHS is not installed or Brummer is not available."

# Check if vhs is installed
if ! command -v vhs &> /dev/null; then
    echo "❌ ERROR: VHS is not installed!"
    echo ""
    echo "Install VHS first:"
    echo "  brew install vhs                    # macOS"
    echo "  sudo snap install vhs               # Linux"
    echo "  go install github.com/charmbracelet/vhs@latest"
    echo ""
    echo "See: https://github.com/charmbracelet/vhs"
    exit 1
fi

# Check if brum is installed
if ! command -v brum &> /dev/null; then
    echo "❌ ERROR: Brummer (brum) is not installed!"
    echo ""
    echo "Install Brummer first:"
    echo "  cd $(dirname "$SCRIPT_DIR")"
    echo "  make install-user"
    exit 1
fi

# Create output directories
mkdir -p "$STATIC_DIR/screenshots"

# Track failures
FAILED_SCREENSHOTS=()

# Function to run a VHS tape
run_tape() {
    local tape_file=$1
    local tape_name=$(basename "$tape_file" .tape)
    
    echo "📸 Generating: $tape_name"
    cd "$VHS_DIR"
    
    if ! vhs "$tape_file"; then
        echo "❌ FAILED to generate $tape_name"
        FAILED_SCREENSHOTS+=("$tape_name")
        return 1
    fi
    
    echo "✅ Successfully generated $tape_name"
    return 0
}

# Generate individual screenshots
for tape in "$VHS_DIR"/*.tape; do
    if [ -f "$tape" ]; then
        run_tape "$tape" || true
    fi
done

# Check for required screenshots
REQUIRED_SCREENSHOTS=(
    "$STATIC_DIR/brummer-tui.png"
    "$STATIC_DIR/screenshots/tutorial-first-launch.png"
    "$STATIC_DIR/screenshots/react-scripts.png"
    "$STATIC_DIR/screenshots/nextjs-scripts.png"
    "$STATIC_DIR/screenshots/monorepo-overview.png"
    "$STATIC_DIR/screenshots/microservices-scripts.png"
)

MISSING_SCREENSHOTS=()
for screenshot in "${REQUIRED_SCREENSHOTS[@]}"; do
    if [ ! -f "$screenshot" ]; then
        MISSING_SCREENSHOTS+=("$(basename "$screenshot")")
    fi
done

# Report results
echo ""
echo "========================================="

if [ ${#FAILED_SCREENSHOTS[@]} -gt 0 ]; then
    echo "❌ Failed to generate ${#FAILED_SCREENSHOTS[@]} screenshots:"
    for failed in "${FAILED_SCREENSHOTS[@]}"; do
        echo "   - $failed"
    done
    echo ""
fi

if [ ${#MISSING_SCREENSHOTS[@]} -gt 0 ]; then
    echo "❌ Missing required screenshots:"
    for missing in "${MISSING_SCREENSHOTS[@]}"; do
        echo "   - $missing"
    done
    echo ""
    echo "The documentation build will FAIL without these screenshots!"
    exit 1
else
    echo "✅ All required screenshots generated successfully!"
    echo ""
    echo "Generated files in:"
    echo "  $STATIC_DIR/"
    echo "  $STATIC_DIR/screenshots/"
fi