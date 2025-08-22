#!/bin/bash

# Run Brummer with the new component system enabled
echo "🚀 Starting Brummer with component system enabled..."
echo ""
echo "This uses the new component-based UI architecture that:"
echo "  • Uses Bubbles components for better UI handling"
echo "  • Provides cleaner separation of concerns"
echo "  • Enables gradual migration from monolithic Model"
echo ""
echo "To disable and use traditional UI, run without the environment variable."
echo "----------------------------------------"
echo ""

# Enable the component system and run
export BRUMMER_USE_COMPONENTS=true
exec ./brum "$@"