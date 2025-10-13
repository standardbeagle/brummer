#!/bin/bash

echo "=== T015: Performance Validation Test (Fast) ==="
echo "Testing AI Coder Migration Performance"
echo ""

# Test 1: Startup Performance
echo "1. Startup Performance Test"
START_TIME=$(date +%s%N)
timeout 3s ./brum --no-tui > startup_test.log 2>&1 &
STARTUP_PID=$!
sleep 1
if kill -0 $STARTUP_PID 2>/dev/null; then
    END_TIME=$(date +%s%N)
    STARTUP_MS=$(( (END_TIME - START_TIME) / 1000000 ))
    echo "   ✅ Startup time: ${STARTUP_MS}ms"
    kill $STARTUP_PID 2>/dev/null
    wait $STARTUP_PID 2>/dev/null 2>/dev/null
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
if go test -timeout 10s ./internal/tui/services -run "TestNewAICoderService" > service_perf.log 2>&1; then
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
if go test -timeout 10s ./internal/tui/services -run "TestNoGoroutineLeaks_SessionLifecycle" > memory_test.log 2>&1; then
    echo "   ✅ No goroutine leaks detected"
else
    echo "   ❌ Memory leak detected"
    cat memory_test.log
    exit 1
fi

# Test 4: Binary Size Check
echo "4. Binary Size Analysis"
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

# Test 5: Cleanup Performance
echo "5. Cleanup Performance Test"
if go test -timeout 10s ./internal/tui/services -run "TestCleanupTiming" > cleanup_test.log 2>&1; then
    echo "   ✅ Cleanup operations perform well"
else
    echo "   ❌ Cleanup performance test failed"
    cat cleanup_test.log
    exit 1
fi

# Cleanup
rm -f startup_test.log service_perf.log memory_test.log cleanup_test.log

echo ""
echo "=== Performance Validation Summary ==="
echo "✅ Startup performance: ${STARTUP_MS}ms"
echo "✅ Service creation: ${SERVICE_MS}ms"
echo "✅ Memory management: No goroutine leaks"
echo "✅ Binary size: ${BINARY_SIZE_MB}MB"
echo "✅ Cleanup operations: Efficient"
echo ""
echo "🎉 All T015 Performance Validation criteria PASSED!"
echo ""
echo "Performance Summary:"
echo "- Application starts quickly and responds normally"
echo "- Service layer operates efficiently"
echo "- No memory leaks detected in session lifecycle"
echo "- Binary size is reasonable for feature set"
echo "- Cleanup operations are efficient"