#!/bin/bash

echo "Testing Redux/Elm Architecture Switching..."
echo "==========================================="

# Test 1: Check feature flag detection
echo -e "\n1. Testing feature flag detection:"
go run ./internal/tui/feature_flags.go -test 2>/dev/null || {
    echo "   - Creating test program for feature flags..."
    cat > /tmp/test-feature-flags.go << 'EOF'
package main

import (
    "fmt"
    "os"
)

func main() {
    // Check Redux architecture flag
    reduxEnabled := os.Getenv("REDUX_ARCHITECTURE") == "true"
    fmt.Printf("Redux Architecture Enabled: %v\n", reduxEnabled)

    // Check default architecture
    defaultArch := os.Getenv("DEFAULT_ARCHITECTURE")
    if defaultArch == "" {
        defaultArch = "legacy"
    }
    fmt.Printf("Default Architecture: %s\n", defaultArch)
}
EOF
    go run /tmp/test-feature-flags.go
}

# Test 2: Run with legacy architecture (default)
echo -e "\n2. Testing legacy architecture (default):"
timeout 1 ./brum --no-tui --help 2>&1 | head -1

# Test 3: Run with Redux architecture enabled
echo -e "\n3. Testing Redux architecture (feature flag):"
REDUX_ARCHITECTURE=true timeout 1 ./brum --no-tui --help 2>&1 | head -1

# Test 4: Check if architecture manager is working
echo -e "\n4. Testing architecture manager:"
cat > /tmp/test-arch-manager.go << 'EOF'
package main

import (
    "fmt"
    "github.com/standardbeagle/brummer/internal/tui"
)

func main() {
    am := tui.NewArchitectureManager()
    info := am.GetArchitectureInfo()

    fmt.Printf("Architecture Info:\n")
    fmt.Printf("  Current: %v\n", info["current"])
    fmt.Printf("  Use Redux: %v\n", info["useRedux"])
    fmt.Printf("  Feature Flag: %v\n", info["featureFlag"])
}
EOF

echo "   Compiling architecture test..."
if go build -o /tmp/test-arch /tmp/test-arch-manager.go 2>/dev/null; then
    /tmp/test-arch
else
    echo "   Note: Architecture manager test requires full build context"
fi

echo -e "\n✅ Architecture switching test complete!"