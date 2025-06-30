package testutil

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForCondition(t *testing.T) {
	// Test condition that becomes true
	var counter int32
	go func() {
		time.Sleep(50 * time.Millisecond)
		atomic.StoreInt32(&counter, 1)
	}()
	
	result := WaitForCondition(t, 200*time.Millisecond, func() bool {
		return atomic.LoadInt32(&counter) == 1
	})
	
	if !result {
		t.Error("Expected condition to become true")
	}
}

func TestWaitForConditionTimeout(t *testing.T) {
	// Test condition that never becomes true
	result := WaitForCondition(t, 50*time.Millisecond, func() bool {
		return false
	})
	
	if result {
		t.Error("Expected condition to timeout")
	}
}

func TestTimedWaitGroup(t *testing.T) {
	twg := &TimedWaitGroup{}
	
	// Test successful completion within timeout
	twg.Add(1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		twg.Done()
	}()
	
	result := twg.WaitWithTimeout(200 * time.Millisecond)
	if !result {
		t.Error("Expected WaitGroup to complete within timeout")
	}
}

func TestTimedWaitGroupTimeout(t *testing.T) {
	twg := &TimedWaitGroup{}
	
	// Test timeout scenario
	twg.Add(1)
	// Never call Done()
	
	result := twg.WaitWithTimeout(50 * time.Millisecond)
	if result {
		t.Error("Expected WaitGroup to timeout")
	}
}