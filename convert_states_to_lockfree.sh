#!/bin/bash

# Script to convert remaining state files to lock-free immutable pattern

STATE_DIR="/home/beagle/work/brummer/internal/tui/state"

# List of state files to convert
STATE_FILES=(
    "web_view_state.go"
    "command_window_state.go"
    "mcp_state.go"
    "mcp_debug_state.go"
    "settings_state.go"
    "test_view_state.go"
    "application_state.go"
)

echo "Converting state files to lock-free immutable pattern..."
echo ""

for file in "${STATE_FILES[@]}"; do
    filepath="$STATE_DIR/$file"

    if [ ! -f "$filepath" ]; then
        echo "⚠️  Skipping $file (not found)"
        continue
    fi

    echo "Processing $file..."

    # Count mutex operations before
    before=$(grep -c "mu\.\(Lock\|RLock\)" "$filepath" 2>/dev/null || echo "0")

    # Remove mutex operations
    sed -i '/s\.mu\.Lock()/d; /s\.mu\.RLock()/d; /defer s\.mu\.Unlock()/d; /defer s\.mu\.RUnlock()/d' "$filepath"

    # Remove sync import if it's the only usage (check if "sync" appears in imports but not elsewhere)
    if grep -q "\"sync\"" "$filepath"; then
        # Check if sync is used elsewhere (e.g., sync.Once, sync.Map, etc.)
        other_sync=$(grep -c "sync\." "$filepath" 2>/dev/null || echo "0")
        if [ "$other_sync" -eq "0" ]; then
            # Remove sync from imports
            sed -i '/^[[:space:]]*"sync"$/d' "$filepath"
            echo "  - Removed sync import"
        fi
    fi

    # Remove mutex field from struct (mu sync.RWMutex or mu sync.Mutex)
    sed -i '/mu[[:space:]]*sync\.\(RW\)\?Mutex/d' "$filepath"

    # Count mutex operations after
    after=$(grep -c "mu\.\(Lock\|RLock\)" "$filepath" 2>/dev/null || echo "0")

    echo "  ✅ Removed $before mutex operations"
done

echo ""
echo "Conversion complete!"
echo ""
echo "Next steps:"
echo "1. Run 'make build' to verify compilation"
echo "2. Run 'make test-race' to check for race conditions"
echo "3. Test keyboard navigation in TUI"
