package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Example_debuggingDiscoveryIssues shows how to debug when hub isn't finding instances
func Example_debuggingDiscoveryIssues() {
	// This example shows the debugging process, but won't have predictable output
	// since it depends on the runtime environment. Convert to regular test.
}

// TestDebugInstanceNotDiscovered demonstrates how to debug when an instance isn't discovered
func TestDebugInstanceNotDiscovered(t *testing.T) {
	// This test shows the debugging process when discovery isn't working
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "debug-instances")
	
	// Step 1: Create discovery and check for errors
	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
		// Debug: Check permissions, disk space, etc.
	}
	defer func() {
		if discovery.watcher != nil {
			discovery.watcher.Close()
		}
	}()
	
	// Step 2: Set up callback to track discovery events
	var discoveredMu sync.Mutex
	var discovered []string
	discovery.OnUpdate(func(instances map[string]*Instance) {
		t.Logf("Discovery callback triggered with %d instances", len(instances))
		discoveredMu.Lock()
		defer discoveredMu.Unlock()
		for id := range instances {
			discovered = append(discovered, id)
			t.Logf("  - Instance: %s", id)
		}
	})
	
	// Step 3: Start discovery
	discovery.Start()
	defer discovery.Stop()
	
	// Step 4: Register an instance
	instance := &Instance{
		ID:        "debug-instance",
		Name:      "Debug Test",
		Directory: "/test",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(),
			Executable: "test",
		},
	}
	
	t.Logf("Registering instance: %s", instance.ID)
	if err := RegisterInstance(instancesDir, instance); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	
	// Step 5: Check file was created
	instanceFile := filepath.Join(instancesDir, "debug-instance.json")
	if _, err := os.Stat(instanceFile); err != nil {
		t.Errorf("Instance file not created: %v", err)
		
		// Debug: Check directory permissions
		report, _ := GenerateDiagnosticReport(instancesDir)
		PrintDiagnosticReport(os.Stdout, report)
	}
	
	// Step 6: Wait for discovery with timeout
	discoveredMu.Lock()
	discovered = []string{} // Reset
	discoveredMu.Unlock()
	
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		discoveredMu.Lock()
		hasDiscovered := len(discovered) > 0
		discoveredMu.Unlock()
		if hasDiscovered {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	
	// Step 7: If not discovered, generate diagnostic report
	discoveredMu.Lock()
	discoveredCount := len(discovered)
	discoveredCopy := make([]string, len(discovered))
	copy(discoveredCopy, discovered)
	discoveredMu.Unlock()
	
	if discoveredCount == 0 {
		t.Error("Instance was not discovered!")
		
		// Generate comprehensive diagnostic report
		report, err := GenerateDiagnosticReport(instancesDir)
		if err != nil {
			t.Logf("Failed to generate report: %v", err)
		} else {
			// Print detailed diagnostics
			PrintDiagnosticReport(os.Stdout, report)
			
			// Check specific issues
			if !report.DirExists {
				t.Error("Directory doesn't exist!")
			}
			
			if len(report.InvalidFiles) > 0 {
				t.Errorf("Found invalid files: %v", report.InvalidFiles)
			}
			
			// Check if file watcher is working
			t.Logf("File count in directory: %d", report.FileCount)
			t.Logf("Valid instances found: %d", len(report.ValidInstances))
		}
		
		// Get specific diagnosis
		diagnosis, _ := DiagnoseDiscoveryIssue(instancesDir)
		t.Logf("Diagnosis: %s", diagnosis)
	} else {
		t.Logf("Successfully discovered: %v", discoveredCopy)
	}
}

// TestHubDiscoveryTroubleshooting shows how to troubleshoot hub discovery issues
func TestHubDiscoveryTroubleshooting(t *testing.T) {
	// This test demonstrates the complete troubleshooting process
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "hub-instances")
	
	// Step 1: Verify the directory setup
	t.Log("Step 1: Verifying discovery setup...")
	if err := VerifyDiscoverySetup(instancesDir); err != nil {
		t.Logf("Setup issues found: %v", err)
		
		// Create directory if needed
		if err := os.MkdirAll(instancesDir, 0755); err != nil {
			t.Fatalf("Cannot create directory: %v", err)
		}
	}
	
	// Step 2: Create discovery with detailed logging
	t.Log("Step 2: Creating discovery system...")
	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Discovery creation failed: %v", err)
	}
	defer func() {
		if discovery.watcher != nil {
			discovery.watcher.Close()
		}
	}()
	
	// Step 3: Set up comprehensive tracking
	type DiscoveryEvent struct {
		Time      time.Time
		EventType string
		Details   string
	}
	
	var events []DiscoveryEvent
	var eventsMu sync.Mutex
	
	addEvent := func(eventType, details string) {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		events = append(events, DiscoveryEvent{
			Time:      time.Now(),
			EventType: eventType,
			Details:   details,
		})
		t.Logf("[%s] %s: %s", time.Now().Format("15:04:05.000"), eventType, details)
	}
	
	discovery.OnUpdate(func(instances map[string]*Instance) {
		addEvent("CALLBACK", fmt.Sprintf("Called with %d instances", len(instances)))
		for id, inst := range instances {
			addEvent("INSTANCE", fmt.Sprintf("ID=%s, Port=%d", id, inst.Port))
		}
	})
	
	// Step 4: Start discovery
	t.Log("Step 4: Starting discovery...")
	discovery.Start()
	defer discovery.Stop()
	
	// Step 5: Simulate multiple instances registering
	t.Log("Step 5: Registering test instances...")
	instances := []*Instance{
		{
			ID:        "frontend-test",
			Name:      "Frontend",
			Directory: "/frontend",
			Port:      3000,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        os.Getpid(),
				Executable: "node",
			},
		},
		{
			ID:        "backend-test",
			Name:      "Backend",
			Directory: "/backend",
			Port:      8080,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        os.Getpid() + 1,
				Executable: "go",
			},
		},
	}
	
	for _, inst := range instances {
		addEvent("REGISTER", fmt.Sprintf("Registering %s", inst.ID))
		if err := RegisterInstance(instancesDir, inst); err != nil {
			addEvent("ERROR", fmt.Sprintf("Failed to register %s: %v", inst.ID, err))
		}
	}
	
	// Step 6: Wait and check results
	t.Log("Step 6: Waiting for discovery...")
	time.Sleep(500 * time.Millisecond)
	
	// Step 7: Generate final report
	t.Log("Step 7: Generating final diagnostic report...")
	report, _ := GenerateDiagnosticReport(instancesDir)
	
	// Print summary
	t.Logf("\nSummary:")
	t.Logf("- Directory exists: %v", report.DirExists)
	t.Logf("- Files in directory: %d", report.FileCount)
	t.Logf("- Valid instances: %d", len(report.ValidInstances))
	t.Logf("- Invalid files: %d", len(report.InvalidFiles))
	t.Logf("- Discovery callbacks: %d", len(events))
	
	// If issues found, print detailed diagnostics
	if len(report.ValidInstances) != len(instances) {
		t.Error("Not all instances were discovered!")
		PrintDiagnosticReport(os.Stdout, report)
		
		// Print event timeline
		t.Log("\nEvent Timeline:")
		for _, event := range events {
			t.Logf("  %s [%s] %s", 
				event.Time.Format("15:04:05.000"),
				event.EventType,
				event.Details)
		}
	}
}

// Example of using diagnostics in production code
func DebugDiscoveryInProduction(instancesDir string) {
	// This shows how to add discovery debugging to your application
	
	// 1. Quick health check
	if err := VerifyDiscoverySetup(instancesDir); err != nil {
		fmt.Printf("Discovery health check failed: %v\n", err)
	}
	
	// 2. If instances aren't being found, generate report
	report, err := GenerateDiagnosticReport(instancesDir)
	if err != nil {
		fmt.Printf("Cannot generate diagnostic report: %v\n", err)
		return
	}
	
	// 3. Log key metrics
	fmt.Printf("Discovery Status:\n")
	fmt.Printf("  Valid instances: %d\n", len(report.ValidInstances))
	fmt.Printf("  Invalid files: %d\n", len(report.InvalidFiles))
	fmt.Printf("  Directory writable: %v\n", canWriteToDirectory(instancesDir))
	
	// 4. If no instances found, provide actionable diagnosis
	if len(report.ValidInstances) == 0 {
		diagnosis, _ := DiagnoseDiscoveryIssue(instancesDir)
		fmt.Printf("\nDiagnosis:\n%s", diagnosis)
	}
}