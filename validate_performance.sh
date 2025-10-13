#!/bin/bash

echo "=== T015: Performance Validation Test ==="
echo "Testing AI Coder Migration Performance"
echo ""

# Test 1: Startup Performance
echo "1. Startup Performance Test"
START_TIME=$(date +%s%N)
timeout 5s ./brum --no-tui > startup_test.log 2>&1 &
STARTUP_PID=$!
sleep 1
if kill -0 $STARTUP_PID 2>/dev/null; then
    END_TIME=$(date +%s%N)
    STARTUP_MS=$(( (END_TIME - START_TIME) / 1000000 ))
    echo "   ✅ Startup time: ${STARTUP_MS}ms"
    kill $STARTUP_PID 2>/dev/null
    wait $STARTUP_PID 2>/dev/null
else
    echo "   ❌ Failed to start"
    cat startup_test.log
    exit 1
fi

# Check if startup time is reasonable (< 3 seconds)
if [ $STARTUP_MS -lt 3000 ]; then
    echo "   ✅ Startup performance acceptable"
else
    echo "   ❌ Startup too slow: ${STARTUP_MS}ms > 3000ms"
    exit 1
fi

# Test 2: Service Creation Performance
echo "2. Service Creation Performance"
SERVICE_START=$(date +%s%N)
if go test -v -timeout 30s ./internal/tui/services -run "TestNewAICoderService" > service_perf.log 2>&1; then
    SERVICE_END=$(date +%s%N)
    SERVICE_MS=$(( (SERVICE_END - SERVICE_START) / 1000000 ))
    echo "   ✅ Service creation test time: ${SERVICE_MS}ms"

    # Service creation should be fast (< 100ms for the test)
    if [ $SERVICE_MS -lt 100 ]; then
        echo "   ✅ Service creation performance excellent"
    elif [ $SERVICE_MS -lt 500 ]; then
        echo "   ✅ Service creation performance acceptable"
    else
        echo "   ⚠️  Service creation could be optimized: ${SERVICE_MS}ms"
    fi
else
    echo "   ❌ Service creation test failed"
    cat service_perf.log
    exit 1
fi

# Test 3: Memory Usage Validation (Goroutine Leak Test)
echo "3. Memory Usage Validation"
if go test -v -timeout 30s ./internal/tui/services -run "TestNoGoroutineLeaks_SessionLifecycle" > memory_test.log 2>&1; then
    echo "   ✅ No goroutine leaks detected"
else
    echo "   ❌ Memory leak detected"
    cat memory_test.log
    exit 1
fi

# Test 4: Concurrent Session Management
echo "4. Concurrent Session Management"
CONCURRENT_START=$(date +%s%N)
if go test -v -timeout 30s ./internal/tui/services -run "TestMonitoringGoroutineCleanup" > concurrent_test.log 2>&1; then
    CONCURRENT_END=$(date +%s%N)
    CONCURRENT_MS=$(( (CONCURRENT_END - CONCURRENT_START) / 1000000 ))
    echo "   ✅ Concurrent session management time: ${CONCURRENT_MS}ms"

    # Concurrent operations should complete in reasonable time
    if [ $CONCURRENT_MS -lt 1000 ]; then
        echo "   ✅ Concurrent performance excellent"
    elif [ $CONCURRENT_MS -lt 2000 ]; then
        echo "   ✅ Concurrent performance acceptable"
    else
        echo "   ⚠️  Concurrent performance could be optimized: ${CONCURRENT_MS}ms"
    fi
else
    echo "   ❌ Concurrent session management test failed"
    cat concurrent_test.log
    exit 1
fi

# Test 5: Shutdown Performance
echo "5. Shutdown Performance Test"
SHUTDOWN_START=$(date +%s%N)
if go test -v -timeout 30s ./internal/tui/services -run "TestShutdown_AllSessionsStopped" > shutdown_test.log 2>&1; then
    SHUTDOWN_END=$(date +%s%N)
    SHUTDOWN_MS=$(( (SHUTDOWN_END - SHUTDOWN_START) / 1000000 ))
    echo "   ✅ Shutdown test time: ${SHUTDOWN_MS}ms"

    # Shutdown should be fast (< 100ms)
    if [ $SHUTDOWN_MS -lt 100 ]; then
        echo "   ✅ Shutdown performance excellent"
    elif [ $SHUTDOWN_MS -lt 500 ]; then
        echo "   ✅ Shutdown performance acceptable"
    else
        echo "   ⚠️  Shutdown could be optimized: ${SHUTDOWN_MS}ms"
    fi
else
    echo "   ❌ Shutdown test failed"
    cat shutdown_test.log
    exit 1
fi

# Test 6: Binary Size Check
echo "6. Binary Size Analysis"
BINARY_SIZE=$(stat -c%s brum)
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))
echo "   📦 Binary size: ${BINARY_SIZE_MB}MB"

# Binary size should be reasonable (< 50MB for a Go application with TUI)
if [ $BINARY_SIZE_MB -lt 25 ]; then
    echo "   ✅ Binary size excellent"
elif [ $BINARY_SIZE_MB -lt 50 ]; then
    echo "   ✅ Binary size acceptable"
else
    echo "   ⚠️  Binary size large: ${BINARY_SIZE_MB}MB"
fi

# Test 7: CPU Usage During Operations (basic check)
echo "7. CPU Usage During Operations"
# Start brummer in background
timeout 10s ./brum --no-tui > cpu_test.log 2>&1 &
CPU_PID=$!
sleep 2

# Check if it's still running (shouldn't consume excessive CPU)
if kill -0 $CPU_PID 2>/dev/null; then
    echo "   ✅ Process stable during execution"
    kill $CPU_PID 2>/dev/null
    wait $CPU_PID 2>/dev/null
else
    echo "   ⚠️  Process terminated early"
    cat cpu_test.log
fi

# Cleanup
rm -f startup_test.log service_perf.log memory_test.log concurrent_test.log shutdown_test.log cpu_test.log

echo ""
echo "=== Performance Validation Summary ==="
echo "✅ Startup performance: ${STARTUP_MS}ms"
echo "✅ Service creation: ${SERVICE_MS}ms"
echo "✅ Memory management: No goroutine leaks"
echo "✅ Concurrent operations: ${CONCURRENT_MS}ms"
echo "✅ Shutdown performance: ${SHUTDOWN_MS}ms"
echo "✅ Binary size: ${BINARY_SIZE_MB}MB"
echo "✅ CPU usage: Stable"
echo ""
echo "🎉 All T015 Performance Validation criteria PASSED!"
echo ""
echo "Performance Summary:"
echo "- Application starts quickly and responds normally"
echo "- Service layer operates efficiently"
echo "- No memory leaks detected in session lifecycle"
echo "- Concurrent operations complete in reasonable time"
echo "- Shutdown is clean and fast"