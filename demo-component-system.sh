#!/bin/bash

# Demo script for Brummer Component System
# This showcases the new component-based UI

echo "================================================"
echo "🎨 Brummer Component System Demo"
echo "================================================"
echo ""
echo "This demo showcases the new component-based UI architecture."
echo "The component system provides:"
echo "  • Modular, maintainable components"
echo "  • Better separation of concerns"
echo "  • Modern UI with Bubbles components"
echo "  • Gradual migration from monolithic Model"
echo ""
echo "Press Ctrl+C to exit at any time."
echo ""
echo "================================================"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if demo project exists
if [ ! -f "demo-project/package.json" ]; then
    echo -e "${YELLOW}Creating demo project...${NC}"
    mkdir -p demo-project
    cat > demo-project/package.json <<'EOF'
{
  "name": "brummer-component-demo",
  "version": "1.0.0",
  "scripts": {
    "dev": "echo 'Dev server on http://localhost:3000' && sleep 10",
    "test": "echo 'Running tests...' && sleep 2 && echo 'Tests passed'",
    "build": "echo 'Building...' && sleep 3 && echo 'Build complete'"
  }
}
EOF
fi

echo -e "${BLUE}Demo 1: Component System Status${NC}"
echo "--------------------------------"
echo "Checking if component system can be enabled..."
echo ""

# Test with component flag
echo "Command: BRUMMER_USE_COMPONENTS=true ./brum --version"
env BRUMMER_USE_COMPONENTS=true ./brum --version 2>&1 | head -3
echo ""

echo -e "${BLUE}Demo 2: Component Files${NC}"
echo "------------------------"
echo "Component system includes these view components:"
ls -1 internal/tui/components/*_adapter.go 2>/dev/null | while read file; do
    basename "$file" | sed 's/_adapter.go//' | sed 's/_/ /g' | sed 's/\b\(.\)/\u\1/g'
done
echo ""

echo -e "${BLUE}Demo 3: Running with Components${NC}"
echo "--------------------------------"
echo "To run Brummer with the component system:"
echo ""
echo "  1. Using environment variable:"
echo "     ${GREEN}BRUMMER_USE_COMPONENTS=true ./brum${NC}"
echo ""
echo "  2. Using the helper script:"
echo "     ${GREEN}./run-with-components.sh${NC}"
echo ""
echo "  3. In the demo project:"
echo "     ${GREEN}cd demo-project && BRUMMER_USE_COMPONENTS=true ../brum${NC}"
echo ""

echo -e "${BLUE}Demo 4: Component Architecture${NC}"
echo "-------------------------------"
echo "The component system uses a message-based architecture:"
echo ""
echo "  User Input"
echo "      ↓"
echo "  BubbleTea"
echo "      ↓"
echo "  Component.Update(msg)"
echo "      ↓"
echo "  Custom Messages"
echo "      ↓"
echo "  Component State"
echo "      ↓"
echo "  Component.View()"
echo "      ↓"
echo "  Display"
echo ""

echo -e "${BLUE}Demo 5: Key Features${NC}"
echo "---------------------"
echo "✓ ProcessViewComponent    - Process management with smooth scrolling"
echo "✓ LogsViewComponent       - Real-time log viewing with filtering"
echo "✓ ErrorsViewComponent     - Error tracking and detailed display"
echo "✓ URLsViewComponent       - URL detection and browser integration"
echo "✓ WebViewComponent        - Web/proxy inspector with multiple modes"
echo "✓ AICodersViewComponent   - AI coder session management"
echo "✓ SettingsViewComponent   - Settings with category organization"
echo "✓ CommandInputComponent   - Command palette with suggestions"
echo "✓ NotificationComponent   - System-wide notifications"
echo "✓ LayoutComponent         - Flexible layout management"
echo ""

echo -e "${BLUE}Demo 6: Migration Strategy${NC}"
echo "--------------------------"
echo "The component system supports gradual migration:"
echo ""
echo "  1. Both UIs coexist - toggle with environment variable"
echo "  2. ModelComponentAdapter bridges old and new systems"
echo "  3. Components read from existing Model/data structures"
echo "  4. No breaking changes to existing functionality"
echo ""

echo -e "${GREEN}Ready to Try?${NC}"
echo "-------------"
echo ""
echo "Start the component system in the demo project:"
echo "  ${GREEN}cd demo-project && BRUMMER_USE_COMPONENTS=true ../brum${NC}"
echo ""
echo "Or use the helper script:"
echo "  ${GREEN}./run-with-components.sh -d demo-project${NC}"
echo ""
echo "Key bindings in the TUI:"
echo "  • Tab        - Switch between views"
echo "  • 1-8        - Quick view selection"
echo "  • Ctrl+L     - Cycle layout modes"
echo "  • Ctrl+P     - Command palette"
echo "  • Ctrl+N     - Show notifications"
echo "  • /          - Search/filter"
echo "  • ?          - Help"
echo "  • q          - Quit"
echo ""
echo "================================================"
echo "🚀 Component system is ready for testing!"
echo "================================================"