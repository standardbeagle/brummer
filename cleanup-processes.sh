#!/bin/bash

# Script to clean up hanging npm/pnpm/node development processes
# while preserving important system processes like Claude

echo "🧹 Cleaning up hanging development processes..."

# Show what processes we're about to kill
echo ""
echo "📋 Current npm/pnpm/node processes (excluding Claude):"
ps aux | grep -E "(npm|pnpm|node)" | grep -v grep | grep -v claude | grep -v "cleanup-processes"

echo ""
echo "🎯 Targeting development server processes..."

# Kill Next.js dev servers
NEXT_PIDS=$(ps aux | grep -E "next dev" | grep -v grep | awk '{print $2}')
if [ ! -z "$NEXT_PIDS" ]; then
    echo "🔴 Killing Next.js dev servers: $NEXT_PIDS"
    echo $NEXT_PIDS | xargs kill -9 2>/dev/null || true
fi

# Kill pnpm dev processes  
PNPM_PIDS=$(ps aux | grep -E "pnpm.*dev" | grep -v grep | awk '{print $2}')
if [ ! -z "$PNPM_PIDS" ]; then
    echo "🔴 Killing pnpm dev processes: $PNPM_PIDS"
    echo $PNPM_PIDS | xargs kill -9 2>/dev/null || true
fi

# Kill npm dev processes
NPM_PIDS=$(ps aux | grep -E "npm.*dev" | grep -v grep | awk '{print $2}')
if [ ! -z "$NPM_PIDS" ]; then
    echo "🔴 Killing npm dev processes: $NPM_PIDS"
    echo $NPM_PIDS | xargs kill -9 2>/dev/null || true
fi

# Kill orphaned pnpm tool processes
PNPM_TOOL_PIDS=$(ps aux | grep -E "\.tools/pnpm" | grep -v grep | awk '{print $2}')
if [ ! -z "$PNPM_TOOL_PIDS" ]; then
    echo "🔴 Killing orphaned pnpm tools: $PNPM_TOOL_PIDS"
    echo $PNPM_TOOL_PIDS | xargs kill -9 2>/dev/null || true
fi

# Kill any remaining node processes that look like dev servers
DEV_NODE_PIDS=$(ps aux | grep -E "node.*(:3000|:8080|:4000|dev-server)" | grep -v grep | grep -v claude | awk '{print $2}')
if [ ! -z "$DEV_NODE_PIDS" ]; then
    echo "🔴 Killing development node processes: $DEV_NODE_PIDS"
    echo $DEV_NODE_PIDS | xargs kill -9 2>/dev/null || true
fi

# Kill next-server processes that might be holding ports
NEXT_SERVER_PIDS=$(ps aux | grep -E "next-server" | grep -v grep | awk '{print $2}')
if [ ! -z "$NEXT_SERVER_PIDS" ]; then
    echo "🔴 Killing next-server processes: $NEXT_SERVER_PIDS"
    echo $NEXT_SERVER_PIDS | xargs kill -9 2>/dev/null || true
fi

# Force free ports 3000-3009 if still occupied
echo "🔍 Checking for locked ports..."
LOCKED_PORTS=$(ss -tlnp | grep -E ":300[0-9]" | grep -o "pid=[0-9]*" | cut -d= -f2 | sort | uniq)
if [ ! -z "$LOCKED_PORTS" ]; then
    echo "🔓 Freeing locked ports (PIDs: $LOCKED_PORTS)"
    echo $LOCKED_PORTS | xargs kill -9 2>/dev/null || true
fi

sleep 1

echo ""
echo "✅ Cleanup complete!"
echo ""
echo "📋 Remaining npm/pnpm/node processes:"
REMAINING=$(ps aux | grep -E "(npm|pnpm|node)" | grep -v grep | grep -v claude | grep -v "cleanup-processes")
if [ -z "$REMAINING" ]; then
    echo "   (none - all development processes cleaned up!)"
else
    echo "$REMAINING"
fi

echo ""
echo "💡 Tips:"
echo "   • Use 'brummer' to properly manage development processes"
echo "   • Press 'q' in Brummer to cleanly stop all processes"  
echo "   • This script preserves important processes like Claude"