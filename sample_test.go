package main

import (
	"testing"
	"time"
)

// TestBasicFunctionality tests basic functionality
func TestBasicFunctionality(t *testing.T) {
	t.Log("Testing basic functionality")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	result := add(2, 3)
	expected := 5

	if result != expected {
		t.Errorf("add(2, 3) = %d; want %d", result, expected)
	} else {
		t.Logf("✓ add(2, 3) = %d", result)
	}
}

// TestEdgeCases tests edge cases
func TestEdgeCases(t *testing.T) {
	t.Log("Testing edge cases")

	// Test with zero
	result := add(0, 0)
	if result != 0 {
		t.Errorf("add(0, 0) = %d; want 0", result)
	}

	// Test with negative numbers
	result = add(-1, 1)
	if result != 0 {
		t.Errorf("add(-1, 1) = %d; want 0", result)
	}

	// Simulate some work
	time.Sleep(5 * time.Millisecond)
	t.Log("✓ Edge cases passed")
}

// TestPerformance tests performance
func TestPerformance(t *testing.T) {
	t.Log("Testing performance")

	// Simulate a slow test to test performance monitoring
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	result := fib(10)
	duration := time.Since(start)

	expected := 55
	if result != expected {
		t.Errorf("fib(10) = %d; want %d", result, expected)
	}

	t.Logf("✓ fib(10) = %d (took %v)", result, duration)

	// Mark as slow if it takes too long
	if duration > 50*time.Millisecond {
		t.Logf("⚠️  This test is slow (%v)", duration)
	}
}

// TestFailingCase creates a failing test for demonstration
func TestFailingCase(t *testing.T) {
	t.Log("This test should fail for demonstration")

	// This will always fail
	if 2+2 != 5 {
		t.Error("2+2 != 5 (expected failure)")
	}
}

// TestSkippedTest demonstrates a skipped test
func TestSkippedTest(t *testing.T) {
	t.Skip("This test is skipped for demonstration")
}

// Helper functions
func add(a, b int) int {
	return a + b
}

func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

// BenchmarkAdd benchmarks the add function
func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		add(i, i+1)
	}
}

// BenchmarkFib benchmarks the fib function
func BenchmarkFib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fib(10)
	}
}