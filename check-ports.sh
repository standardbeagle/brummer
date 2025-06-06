#!/bin/bash

# Quick script to check what's using development ports

echo "🔍 Checking development ports (3000-3009)..."
echo ""

for port in {3000..3009}; do
    PROCESS=$(ss -tlnp | grep ":$port " 2>/dev/null)
    if [ ! -z "$PROCESS" ]; then
        echo "🔴 Port $port is OCCUPIED:"
        echo "   $PROCESS"
    else
        echo "✅ Port $port is free"
    fi
done

echo ""
echo "🌐 Quick port summary:"
OCCUPIED=$(ss -tlnp | grep -E ":300[0-9]" | wc -l)
if [ $OCCUPIED -eq 0 ]; then
    echo "   All development ports (3000-3009) are free! 🎉"
else
    echo "   $OCCUPIED ports are occupied"
    echo ""
    echo "💡 To free them, run: ./cleanup-processes.sh"
fi