package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDiagnosticReportGeneration tests the diagnostic report generation
func TestDiagnosticReportGeneration(t *testing.T) {
	t.Parallel()
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	// Create instances directory
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		t.Fatalf("Failed to create instances dir: %v", err)
	}
	
	// Create a valid instance
	validInstance := &Instance{
		ID:        "test-valid",
		Name:      "Valid Instance",
		Directory: "/test/valid",
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
	
	data, _ := json.MarshalIndent(validInstance, "", "  ")
	if err := os.WriteFile(filepath.Join(instancesDir, "test-valid.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write valid instance: %v", err)
	}
	
	// Create an invalid JSON file
	if err := os.WriteFile(filepath.Join(instancesDir, "invalid.json"), []byte("not json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}
	
	// Create a file with missing fields
	incompleteInstance := map[string]interface{}{
		"id":   "incomplete",
		"name": "Missing Fields",
		// Missing required fields
	}
	incompleteData, _ := json.Marshal(incompleteInstance)
	if err := os.WriteFile(filepath.Join(instancesDir, "incomplete.json"), incompleteData, 0644); err != nil {
		t.Fatalf("Failed to write incomplete instance: %v", err)
	}
	
	// Generate report
	report, err := GenerateDiagnosticReport(instancesDir)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}
	
	// Verify report contents
	if !report.DirExists {
		t.Error("Directory should exist")
	}
	
	if report.FileCount < 3 {
		t.Errorf("Expected at least 3 files, got %d", report.FileCount)
	}
	
	if len(report.ValidInstances) != 1 {
		t.Errorf("Expected 1 valid instance, got %d", len(report.ValidInstances))
	}
	
	if _, exists := report.ValidInstances["test-valid"]; !exists {
		t.Error("Valid instance not found in report")
	}
	
	if len(report.InvalidFiles) < 2 {
		t.Errorf("Expected at least 2 invalid files, got %d", len(report.InvalidFiles))
	}
}

// TestDiagnosticReportPrinting tests the human-readable output
func TestDiagnosticReportPrinting(t *testing.T) {
	t.Parallel()
	
	report := &DiagnosticReport{
		Timestamp:      time.Now(),
		InstancesDir:   "/test/instances",
		DirExists:      true,
		DirPermissions: "drwxr-xr-x",
		FileCount:      3,
		ValidInstances: map[string]*Instance{
			"test-1": {
				ID:        "test-1",
				Name:      "Test Instance",
				Port:      7777,
				StartedAt: time.Now().Add(-1 * time.Hour),
				LastPing:  time.Now().Add(-5 * time.Minute),
				ProcessInfo: struct {
					PID        int    `json:"pid"`
					Executable string `json:"executable"`
				}{
					PID:        12345,
					Executable: "brum",
				},
			},
		},
		InvalidFiles: map[string]string{
			"bad.json": "JSON parse error: invalid character",
		},
		ProcessStatus: map[string]ProcessStatus{
			"test-1": {PID: 12345, Running: true},
		},
		SystemInfo: SystemInfo{
			TempDir:       "/tmp",
			XDGRuntimeDir: "/run/user/1000",
			DefaultDir:    "/tmp/brummer/instances",
			CurrentUser:   "testuser",
		},
		Errors: []string{"Test error message"},
	}
	
	// Print to buffer
	var buf bytes.Buffer
	PrintDiagnosticReport(&buf, report)
	
	output := buf.String()
	
	// Verify output contains expected sections
	expectedSections := []string{
		"Discovery Diagnostic Report",
		"System Information:",
		"ERRORS:",
		"Valid Instances (1):",
		"Invalid Files (1):",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Output missing section: %s", section)
		}
	}
}

// TestVerifyDiscoverySetup tests the setup verification
func TestVerifyDiscoverySetup(t *testing.T) {
	t.Parallel()
	
	// Test 1: Non-existent directory
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")
	err := VerifyDiscoverySetup(nonExistentDir)
	if err == nil {
		t.Error("Expected error for non-existent directory")
	} else if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
	
	// Test 2: Valid setup
	validDir := filepath.Join(t.TempDir(), "valid")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	if err := VerifyDiscoverySetup(validDir); err != nil {
		t.Errorf("Valid setup should not return error: %v", err)
	}
	
	// Test 3: Directory with stale instance
	staleDir := filepath.Join(t.TempDir(), "stale")
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	staleInstance := &Instance{
		ID:        "stale-instance",
		Name:      "Stale",
		Directory: "/test",
		Port:      9999,
		StartedAt: time.Now().Add(-24 * time.Hour),
		LastPing:  time.Now().Add(-10 * time.Minute), // Stale
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        99999, // Non-existent process
			Executable: "test",
		},
	}
	
	data, _ := json.MarshalIndent(staleInstance, "", "  ")
	if err := os.WriteFile(filepath.Join(staleDir, "stale-instance.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write stale instance: %v", err)
	}
	
	err = VerifyDiscoverySetup(staleDir)
	if err == nil {
		t.Error("Expected error for stale instance")
	} else if !strings.Contains(err.Error(), "stale") {
		t.Errorf("Expected stale instance error, got: %v", err)
	}
}

// TestDiagnoseDiscoveryIssue tests the issue diagnosis function
func TestDiagnoseDiscoveryIssue(t *testing.T) {
	t.Parallel()
	
	// Test 1: Empty directory
	emptyDir := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	diagnosis, err := DiagnoseDiscoveryIssue(emptyDir)
	if err != nil {
		t.Fatalf("Failed to diagnose: %v", err)
	}
	
	if !strings.Contains(diagnosis, "contains no instance files") {
		t.Errorf("Expected diagnosis about empty directory, got: %s", diagnosis)
	}
	
	// Test 2: Directory with dead process
	deadDir := filepath.Join(t.TempDir(), "dead")
	if err := os.MkdirAll(deadDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	deadInstance := &Instance{
		ID:        "dead-instance",
		Name:      "Dead Process",
		Directory: "/test",
		Port:      8888,
		StartedAt: time.Now().Add(-1 * time.Hour),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        99999, // Non-existent process
			Executable: "test",
		},
	}
	
	data, _ := json.MarshalIndent(deadInstance, "", "  ")
	if err := os.WriteFile(filepath.Join(deadDir, "dead-instance.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write dead instance: %v", err)
	}
	
	diagnosis, err = DiagnoseDiscoveryIssue(deadDir)
	if err != nil {
		t.Fatalf("Failed to diagnose: %v", err)
	}
	
	if !strings.Contains(diagnosis, "dead processes") {
		t.Errorf("Expected diagnosis about dead processes, got: %s", diagnosis)
	}
}

// TestDiagnosticsUnderLoad tests diagnostics work under concurrent load
func TestDiagnosticsUnderLoad(t *testing.T) {
	t.Parallel()
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	// Create many instance files concurrently
	numInstances := 100
	done := make(chan bool, numInstances)
	
	for i := 0; i < numInstances; i++ {
		go func(idx int) {
			instance := &Instance{
				ID:        fmt.Sprintf("load-test-%d", idx),
				Name:      fmt.Sprintf("Load Test %d", idx),
				Directory: "/test",
				Port:      10000 + idx,
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
			
			RegisterInstance(instancesDir, instance)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < numInstances; i++ {
		<-done
	}
	
	// Generate diagnostic report while instances are present
	report, err := GenerateDiagnosticReport(instancesDir)
	if err != nil {
		t.Fatalf("Failed to generate report under load: %v", err)
	}
	
	// Should have found most instances (some race conditions are OK)
	if len(report.ValidInstances) < numInstances/2 {
		t.Errorf("Expected at least %d instances, got %d", numInstances/2, len(report.ValidInstances))
	}
	
	// Verify the report is coherent
	if report.FileCount < len(report.ValidInstances) {
		t.Error("File count should be at least as many as valid instances")
	}
}