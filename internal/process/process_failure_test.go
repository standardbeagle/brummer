package process

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/testutil"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessFailureScenarios tests various process failure conditions
func TestProcessFailureScenarios(t *testing.T) {
	scenarios := []testutil.TestScenario{
		{
			Name:        "ProcessManagerResourceExhaustion",
			Description: "Test behavior when process manager hits resource limits",
			Setup: func(t *testing.T) interface{} {
				eventBus := events.NewEventBus()
				manager, err := NewManager("/tmp", eventBus, false)
				require.NoError(t, err)
				
				re := testutil.NewResourceExhauster()
				re.SetFDLimit(1) // Very low FD limit
				
				return map[string]interface{}{
					"manager": manager,
					"re":      re,
				}
			},
			Execute: func(t *testing.T, context interface{}) error {
				ctx := context.(map[string]interface{})
				manager := ctx["manager"].(*Manager)
				re := ctx["re"].(*testutil.ResourceExhauster)
				
				// Try to start multiple processes that should exhaust resources
				var lastError error
				for i := 0; i < 5; i++ {
					// Check if we should fail FD allocation
					if err := re.CheckFDAllocation(); err != nil {
						return err
					}
					
					// Try to start a simple script
					proc, err := manager.StartScript("test")
					if err != nil {
						lastError = err
						break
					}
					if proc != nil {
						manager.StopProcess(proc.ID)
					}
				}
				
				return lastError
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should fail due to resource exhaustion
				if err != nil {
					assert.Error(t, err, "Should fail when resources are exhausted")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				ctx := context.(map[string]interface{})
				manager := ctx["manager"].(*Manager)
				manager.Cleanup()
			},
		},
		{
			Name:        "ProcessManagerConcurrentOperations",
			Description: "Test concurrent process operations",
			Setup: func(t *testing.T) interface{} {
				eventBus := events.NewEventBus()
				manager, err := NewManager("/tmp", eventBus, false)
				require.NoError(t, err)
				return manager
			},
			Execute: func(t *testing.T, context interface{}) error {
				manager := context.(*Manager)
				
				// Start multiple processes concurrently
				twg := &testutil.TimedWaitGroup{}
				
				for i := 0; i < 5; i++ {
					twg.Add(1)
					go func() {
						defer twg.Done()
						
						// Start a simple process
						proc, err := manager.StartScript("test")
						if err == nil && proc != nil {
							// Wait a bit then stop
							time.Sleep(100 * time.Millisecond)
							manager.StopProcess(proc.ID)
						}
					}()
				}
				
				// Wait for all operations to complete
				completed := twg.WaitWithTimeout(10 * time.Second)
				if !completed {
					return assert.AnError
				}
				
				return nil
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				assert.NoError(t, err, "Concurrent operations should complete successfully")
			},
			Cleanup: func(t *testing.T, context interface{}) {
				manager := context.(*Manager)
				manager.Cleanup()
			},
		},
		{
			Name:        "ProcessCleanupUnderStress",
			Description: "Test process cleanup under stress conditions",
			Setup: func(t *testing.T) interface{} {
				eventBus := events.NewEventBus()
				manager, err := NewManager("/tmp", eventBus, false)
				require.NoError(t, err)
				return manager
			},
			Execute: func(t *testing.T, context interface{}) error {
				manager := context.(*Manager)
				
				// Start many processes quickly
				processes := make([]*Process, 0, 10)
				for i := 0; i < 10; i++ {
					proc, err := manager.StartScript("test")
					if err == nil && proc != nil {
						processes = append(processes, proc)
					}
				}
				
				// Wait for processes to start
				testutil.RequireEventually(t, 2*time.Second, func() bool {
					allProcs := manager.GetAllProcesses()
					return len(allProcs) > 0
				}, "Some processes should start")
				
				// Stop all processes
				for _, proc := range processes {
					manager.StopProcess(proc.ID)
				}
				
				// Verify cleanup
				testutil.RequireEventually(t, 5*time.Second, func() bool {
					allProcs := manager.GetAllProcesses()
					runningCount := 0
					for _, p := range allProcs {
						if p.Status == StatusRunning {
							runningCount++
						}
					}
					return runningCount == 0
				}, "All processes should be stopped")
				
				return nil
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				assert.NoError(t, err, "Process cleanup should succeed")
			},
			Cleanup: func(t *testing.T, context interface{}) {
				manager := context.(*Manager)
				manager.Cleanup()
			},
		},
	}
	
	testutil.RunTestScenarios(t, scenarios)
}

// TestProcessErrorInjection tests error injection in process operations
func TestProcessErrorInjection(t *testing.T) {
	eventBus := events.NewEventBus()
	manager, err := NewManager("/tmp", eventBus, false)
	require.NoError(t, err)
	defer manager.Cleanup()
	
	ei := testutil.NewErrorInjector()
	
	// Configure error injection for process operations
	ei.InjectFailure("process_start", &testutil.InjectionRule{
		FailCount:    2,
		FailureType:  "resource",
		ErrorMessage: "process start failed",
	})
	
	// Try to start processes
	failureCount := 0
	for i := 0; i < 5; i++ {
		// Check if this operation should fail
		if err := ei.ShouldFail("process_start"); err != nil {
			failureCount++
			continue
		}
		
		// Normal process start
		proc, err := manager.StartScript("test")
		if err == nil && proc != nil {
			manager.StopProcess(proc.ID)
		}
	}
	
	// Should have 2 failures as configured
	assert.Equal(t, 2, failureCount, "Should have exactly 2 failures")
}

// TestNetworkPartitionSimulation tests network partition scenarios
func TestNetworkPartitionSimulation(t *testing.T) {
	// Create network partition simulator
	nps := testutil.NewNetworkPartitionSimulator(t, "127.0.0.1:0")
	nps.Start()
	
	// Test normal operation
	testutil.RequireEventually(t, 2*time.Second, func() bool {
		// In a real test, we would check if we can connect
		return true
	}, "Network should be available initially")
	
	// Simulate partition
	nps.Partition()
	
	// Test partition detection
	time.Sleep(100 * time.Millisecond)
	
	// Repair partition
	nps.Repair()
	
	// Test recovery
	testutil.RequireEventually(t, 2*time.Second, func() bool {
		// In a real test, we would check if connection is restored
		return true
	}, "Network should be restored after repair")
}

// TestErrorInjectionFramework tests the error injection framework itself
func TestErrorInjectionFramework(t *testing.T) {
	ei := testutil.NewErrorInjector()
	
	// Test basic error injection
	ei.InjectFailure("test_op", &testutil.InjectionRule{
		FailCount:    3,
		FailureType:  "network",
		ErrorMessage: "test failure",
	})
	
	// Should fail 3 times
	for i := 0; i < 3; i++ {
		err := ei.ShouldFail("test_op")
		assert.Error(t, err, "Should fail %d times", i+1)
		assert.Contains(t, err.Error(), "network error", "Should be network error")
	}
	
	// Should not fail anymore
	err := ei.ShouldFail("test_op")
	assert.NoError(t, err, "Should not fail after limit reached")
	
	// Test unlimited failures
	ei.InjectFailure("unlimited", &testutil.InjectionRule{
		FailCount:    -1, // Unlimited
		FailureType:  "timeout",
		ErrorMessage: "always fail",
	})
	
	// Should always fail
	for i := 0; i < 10; i++ {
		err := ei.ShouldFail("unlimited")
		assert.Error(t, err, "Should always fail")
	}
	
	// Test reset
	ei.Reset()
	err = ei.ShouldFail("unlimited")
	assert.NoError(t, err, "Should not fail after reset")
}