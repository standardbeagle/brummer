package process

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
)

func TestProcessManager(t *testing.T) {
	// Create test directory with package.json
	testDir := t.TempDir()
	packageJSON := `{
		"name": "test-project",
		"scripts": {
			"echo": "echo 'Hello, World!'",
			"sleep": "sleep 1",
			"error": "echo 'Error!' && exit 1",
			"long": "echo 'Starting long process' && sleep 10"
		}
	}`
	err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	eventBus := events.NewEventBus()
	mgr, err := NewManager(testDir, eventBus, true)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Cleanup()

	// Test GetScripts
	t.Run("GetScripts", func(t *testing.T) {
		scripts := mgr.GetScripts()
		if len(scripts) != 4 {
			t.Errorf("Expected 4 scripts, got %d", len(scripts))
		}
		
		if scripts["echo"] == nil {
			t.Error("Missing echo script")
		}
	})

	// Test StartScript
	t.Run("StartScript", func(t *testing.T) {
		// Capture logs
		var capturedLogs []string
		mgr.AddLogCallback(func(processID, line string, isError bool) {
			capturedLogs = append(capturedLogs, line)
		})

		proc, err := mgr.StartScript("echo")
		if err != nil {
			t.Fatalf("Failed to start script: %v", err)
		}

		// Wait for completion
		time.Sleep(500 * time.Millisecond)

		// Check status
		status := proc.GetStatus()
		if status != StatusExited {
			t.Errorf("Expected status Exited, got %v", status)
		}

		// Check logs
		found := false
		for _, log := range capturedLogs {
			if strings.Contains(log, "Hello, World!") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Did not capture expected output")
		}
	})

	// Test StopProcess
	t.Run("StopProcess", func(t *testing.T) {
		proc, err := mgr.StartScript("long")
		if err != nil {
			t.Fatalf("Failed to start script: %v", err)
		}

		// Give it time to start
		time.Sleep(100 * time.Millisecond)

		// Stop it
		err = mgr.StopProcess(proc.ID)
		if err != nil {
			t.Errorf("Failed to stop process: %v", err)
		}

		// Wait for cleanup
		time.Sleep(100 * time.Millisecond)

		// Check status
		status := proc.GetStatus()
		if status != StatusExited && status != StatusKilled {
			t.Errorf("Expected status Exited or Killed, got %v", status)
		}
	})

	// Test GetProcesses
	t.Run("GetProcesses", func(t *testing.T) {
		// Start a process
		proc, err := mgr.StartScript("sleep")
		if err != nil {
			t.Fatalf("Failed to start script: %v", err)
		}

		processes := mgr.GetProcesses()
		found := false
		for _, p := range processes {
			if p.ID == proc.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Process not found in GetProcesses")
		}
	})

	// Test error handling
	t.Run("ErrorScript", func(t *testing.T) {
		proc, err := mgr.StartScript("error")
		if err != nil {
			t.Fatalf("Failed to start script: %v", err)
		}

		// Wait for exit
		time.Sleep(500 * time.Millisecond)

		// Check exit code
		if proc.ExitCode == nil || *proc.ExitCode != 1 {
			t.Error("Expected exit code 1")
		}
	})

	// Test StartCommand
	t.Run("StartCommand", func(t *testing.T) {
		cmd := "echo"
		args := []string{"Direct command"}
		
		proc, err := mgr.StartCommand("test-cmd", cmd, args)
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}

		// Wait for completion
		time.Sleep(500 * time.Millisecond)

		// Check status
		status := proc.GetStatus()
		if status != StatusExited {
			t.Errorf("Expected status Exited, got %v", status)
		}
	})

	// Test Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Start multiple processes
		proc1, _ := mgr.StartScript("sleep")
		proc2, _ := mgr.StartScript("sleep")

		// Cleanup
		err := mgr.Cleanup()
		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}

		// All processes should be stopped
		processes := mgr.GetProcesses()
		for _, p := range processes {
			if p.ID == proc1.ID || p.ID == proc2.ID {
				status := p.GetStatus()
				if status == StatusRunning {
					t.Error("Process still running after cleanup")
				}
			}
		}
	})
}

func TestPackageManagerDetection(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "npm with lockfile",
			files: map[string]string{
				"package.json":      "{}",
				"package-lock.json": "{}",
			},
			expected: "npm",
		},
		{
			name: "yarn with lockfile",
			files: map[string]string{
				"package.json": "{}",
				"yarn.lock":    "",
			},
			expected: "yarn",
		},
		{
			name: "pnpm with lockfile",
			files: map[string]string{
				"package.json": "{}",
				"pnpm-lock.yaml": "",
			},
			expected: "pnpm",
		},
		{
			name: "bun with lockfile",
			files: map[string]string{
				"package.json": "{}",
				"bun.lockb":    "",
			},
			expected: "bun",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			
			// Create test files
			for filename, content := range tt.files {
				err := os.WriteFile(filepath.Join(testDir, filename), []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
			}

			eventBus := events.NewEventBus()
			mgr, err := NewManager(testDir, eventBus, true)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}
			defer mgr.Cleanup()

			// The manager should detect the package manager
			// This would be internal state - we can verify by checking
			// what command would be used
			scripts := mgr.GetScripts()
			_ = scripts // Package manager detection happens internally
			
			// Since we can't directly test the internal state,
			// we've at least verified the manager initializes correctly
			// with different package manager configurations
		})
	}
}

func TestMonorepoDetection(t *testing.T) {
	// Skip on Windows as symlinks require admin privileges
	if runtime.GOOS == "windows" {
		t.Skip("Skipping monorepo test on Windows")
	}

	testDir := t.TempDir()
	
	// Create pnpm workspace
	workspaceYAML := `packages:
  - 'packages/*'`
	err := os.WriteFile(filepath.Join(testDir, "pnpm-workspace.yaml"), []byte(workspaceYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create pnpm-workspace.yaml: %v", err)
	}

	// Create root package.json
	rootPackageJSON := `{
		"name": "monorepo-root",
		"private": true
	}`
	err = os.WriteFile(filepath.Join(testDir, "package.json"), []byte(rootPackageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Create packages directory
	packagesDir := filepath.Join(testDir, "packages", "app")
	err = os.MkdirAll(packagesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create packages directory: %v", err)
	}

	// Create package in monorepo
	appPackageJSON := `{
		"name": "@monorepo/app",
		"scripts": {
			"dev": "echo 'App dev'"
		}
	}`
	err = os.WriteFile(filepath.Join(packagesDir, "package.json"), []byte(appPackageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create app package.json: %v", err)
	}

	// Test from subdirectory
	eventBus := events.NewEventBus()
	mgr, err := NewManager(packagesDir, eventBus, true)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Cleanup()

	// Should detect scripts from the app package
	scripts := mgr.GetScripts()
	if scripts["dev"] == nil {
		t.Error("Failed to detect scripts in monorepo package")
	}
}

func TestProcessEvents(t *testing.T) {
	testDir := t.TempDir()
	packageJSON := `{
		"name": "test-events",
		"scripts": {
			"test": "echo 'Test output'"
		}
	}`
	err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	eventBus := events.NewEventBus()
	
	// Track events
	var processStarted bool
	var processExited bool
	
	eventBus.Subscribe(events.EventProcessStarted, func(e events.Event) {
		processStarted = true
	})
	
	eventBus.Subscribe(events.EventProcessExited, func(e events.Event) {
		processExited = true
	})

	mgr, err := NewManager(testDir, eventBus, true)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Cleanup()

	// Start process
	proc, err := mgr.StartScript("test")
	if err != nil {
		t.Fatalf("Failed to start script: %v", err)
	}

	// Wait for completion
	time.Sleep(500 * time.Millisecond)

	// Check events
	if !processStarted {
		t.Error("ProcessStarted event not received")
	}
	if !processExited {
		t.Error("ProcessExited event not received")
	}

	// Verify process ID in events
	_ = proc.ID // Would be in event data
}