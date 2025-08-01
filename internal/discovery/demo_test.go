package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDemoDiscoveryDiagnostics demonstrates the diagnostic tools in action
func TestDemoDiscoveryDiagnostics(t *testing.T) {
	// Create a test directory
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")

	// Case 1: Directory doesn't exist yet
	t.Log("=== Case 1: Directory doesn't exist ===")
	report1, _ := GenerateDiagnosticReport(instancesDir)
	// Use t.Logf instead of printing to stdout
	t.Logf("Directory exists: %v, Errors: %v", report1.DirExists, report1.Errors)

	// Create discovery system
	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	t.Cleanup(func() {
		if err := discovery.Stop(); err != nil {
			t.Logf("Failed to stop discovery: %v", err)
		}
	})

	// Start discovery
	discovery.Start()

	// Case 2: Empty directory
	t.Log("\n=== Case 2: Empty directory (no instances) ===")
	report2, _ := GenerateDiagnosticReport(instancesDir)
	t.Logf("Directory exists: %v", report2.DirExists)
	t.Logf("File count: %d", report2.FileCount)
	t.Logf("Valid instances: %d", len(report2.ValidInstances))

	// Case 3: Register a valid instance
	instance1 := &Instance{
		ID:        "demo-instance-1",
		Name:      "Demo Frontend Server",
		Directory: "/demo/frontend",
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
	}

	if err := RegisterInstance(instancesDir, instance1); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}
	// Wait for file watcher to detect the new instance
	time.Sleep(100 * time.Millisecond)

	t.Log("\n=== Case 3: After registering valid instance ===")
	report3, _ := GenerateDiagnosticReport(instancesDir)
	t.Logf("Valid instances: %d", len(report3.ValidInstances))
	for id, inst := range report3.ValidInstances {
		t.Logf("  - %s: %s on port %d", id, inst.Name, inst.Port)
	}

	// Case 4: Add a stale instance
	staleInstance := &Instance{
		ID:        "demo-stale",
		Name:      "Stale Service",
		Directory: "/demo/stale",
		Port:      8080,
		StartedAt: time.Now().Add(-2 * time.Hour),
		LastPing:  time.Now().Add(-10 * time.Minute), // Stale!
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        99999, // Non-existent process
			Executable: "ghost",
		},
	}

	if err := RegisterInstance(instancesDir, staleInstance); err != nil {
		t.Fatalf("Failed to register stale instance: %v", err)
	}
	// Wait for file watcher
	time.Sleep(100 * time.Millisecond)

	t.Log("\n=== Case 4: With stale instance ===")
	diagnosis, _ := DiagnoseDiscoveryIssue(instancesDir)
	t.Log(diagnosis)

	// Case 5: Run cleanup
	discovery.CleanupStaleInstances()

	t.Log("\n=== Case 5: After cleanup ===")
	report5, _ := GenerateDiagnosticReport(instancesDir)
	t.Logf("Valid instances after cleanup: %d", len(report5.ValidInstances))
	for id, inst := range report5.ValidInstances {
		t.Logf("  - %s: %s (still running)", id, inst.Name)
	}
}

// TestDemoVerifySetup shows how to verify discovery setup
func TestDemoVerifySetup(t *testing.T) {
	t.Log("=== Discovery Setup Verification ===")

	// Check default directory
	defaultDir := GetDefaultInstancesDir()
	t.Logf("Default instances directory: %s", defaultDir)

	// Verify setup
	err := VerifyDiscoverySetup(defaultDir)
	if err != nil {
		t.Logf("Setup issues found: %v", err)

		// Create directory if needed
		if err := os.MkdirAll(defaultDir, 0700); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Verify again
		err = VerifyDiscoverySetup(defaultDir)
		if err == nil {
			t.Log("Setup fixed!")
		}
	} else {
		t.Log("Discovery setup is valid!")
	}
}
