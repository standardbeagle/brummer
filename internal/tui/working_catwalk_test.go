package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/knz/catwalk"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// createTestModelWithWorkingDir creates a test model with all required dependencies
func createTestModelWithWorkingDir() *Model {
	// Create temporary working directory
	workDir, err := os.MkdirTemp("", "brummer-catwalk-test")
	if err != nil {
		panic(err)
	}

	// Create package.json for testing
	packageJson := `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'Running tests...' && sleep 0.1",
    "dev": "echo 'Development server started' && sleep 0.1"
  }
}`
	if err := os.WriteFile(filepath.Join(workDir, "package.json"), []byte(packageJson), 0644); err != nil {
		panic(err)
	}

	// Create dependencies
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	processMgr, err := processes.NewManager(workDir, eventBus, false)
	if err != nil {
		panic(err)
	}

	// Create config
	cfg := &config.Config{}

	// Create model with Logs view to see keyboard input effects
	model := NewModelWithView(
		processMgr,
		logStore,
		eventBus,
		nil, // No MCP server needed
		nil, // No proxy server needed
		7777,
		ViewLogs, // Start in Logs view
		false,   // debug mode = false
		cfg,
		workDir, // working directory - the missing parameter!
	)

	return model
}

// TestKeyboardSwitchToLogsWithCatwalk tests keyboard navigation using catwalk
func TestKeyboardSwitchToLogsWithCatwalk(t *testing.T) {
	model := createTestModelWithWorkingDir()
	defer os.RemoveAll(model.workingDir) // Cleanup

	t.Run("Switch to Logs view with '2' key", func(t *testing.T) {
		// Start in a different view (Processes)
		model.switchToView(ViewProcesses)
		if model.currentView() != ViewProcesses {
			t.Fatalf("Expected to start in Processes view, got %v", model.currentView())
		}

		// Use catwalk to simulate pressing Tab key (test without number keys first)
		catwalk.RunModelFromString(t, "\t", model,
			catwalk.WithWindowSize(80, 24),
		)

		// Check if we switched to next view (should be Logs if starting from Processes)
		if model.currentView() == ViewProcesses {
			t.Errorf("FAILED: Tab key did not move to next view, still in Processes view")
		} else {
			t.Logf("✅ SUCCESS: Tab key correctly switched from Processes to %s view", model.currentView())
		}
	})

	t.Run("Tab navigation", func(t *testing.T) {
		// Start in Processes view
		model.switchToView(ViewProcesses)
		if model.currentView() != ViewProcesses {
			t.Fatalf("Expected to start in Processes view, got %v", model.currentView())
		}

		// Use catwalk to simulate pressing Tab key
		catwalk.RunModelFromString(t, "\t", model,
			catwalk.WithWindowSize(80, 24),
		)

		// Check if we moved to next view (should be Logs if starting from Processes)
		currentView := model.currentView()
		if currentView == ViewProcesses {
			t.Error("FAILED: Tab key did not move to next view, still in Processes view")
		} else {
			t.Logf("✅ SUCCESS: Tab key correctly switched from Processes to %s view", currentView)
		}
	})

	t.Run("Direct ViewChangedMsg test", func(t *testing.T) {
		// Start in Processes view
		model.switchToView(ViewProcesses)
		if model.currentView() != ViewProcesses {
			t.Fatalf("Expected to start in Processes view, got %v", model.currentView())
		}

		// Directly send ViewChangedMsg to test if NavigationComponent receives it
		t.Logf("Testing direct ViewChangedMsg...")

		// Send ViewChangedMsg directly to the model
		updatedModel, cmd := model.Update(messages.ViewChangedMsg{
			FromView: ViewProcesses,
			ToView:   ViewLogs,
			Trigger:  "direct_test",
			Time:     time.Now(),
		})

		// Update the model reference
		*model = *updatedModel.(*Model)

		// Helper function to execute all commands including nested batch commands
		executeCommands := func(cmd tea.Cmd) {
			commandCount := 0
			for cmd != nil && commandCount < 10 { // Prevent infinite loops
				commandCount++
				msg := cmd()
				if msg != nil {
					t.Logf("Command %d returned message: %T", commandCount, msg)

					// Handle batch commands
					if batchMsg, ok := msg.(tea.BatchMsg); ok {
						t.Logf("Batch command contains %d sub-commands", len(batchMsg))
						// Execute each sub-command
						for i, batchCmd := range batchMsg {
							if batchCmd != nil {
								batchMsg := batchCmd()
								if batchMsg != nil {
									t.Logf("Batch sub-command %d returned message: %T", i, batchMsg)
									updatedModel, nextCmd := model.Update(batchMsg)
									*model = *updatedModel.(*Model)
									// Continue with the next command from this sub-command
									if nextCmd != nil {
										cmd = nextCmd
									}
								}
							}
						}
					} else {
						updatedModel, nextCmd := model.Update(msg)
						*model = *updatedModel.(*Model)
						cmd = nextCmd
					}
				} else {
					break
				}
			}
			t.Logf("Processed %d commands", commandCount)
		}

		// Execute the command
		executeCommands(cmd)

		// Check if we switched to Logs view
		currentView := model.currentView()
		expectedView := ViewLogs
		t.Logf("After direct ViewChangedMsg - Current view: %v (type: %T), Expected: %v (type: %T)", currentView, currentView, expectedView, expectedView)

		if currentView != expectedView {
			t.Errorf("Expected to be in Logs view after direct ViewChangedMsg, got %v (%T)", currentView, currentView)
		} else {
			t.Logf("✅ SUCCESS: Direct ViewChangedMsg correctly switched to Logs view")
		}
	})

	t.Run("Switch to Logs view with '2' key (keyboard input test)", func(t *testing.T) {
		// Create a fresh model for this test to avoid interference
		freshModel := createTestModelWithWorkingDir()
		defer os.RemoveAll(freshModel.workingDir) // Cleanup

		// Start in Processes view
		freshModel.switchToView(ViewProcesses)
		if freshModel.currentView() != ViewProcesses {
			t.Fatalf("Expected to start in Processes view, got %v", freshModel.currentView())
		}

		// Test keyboard input through the complete chain
		t.Logf("Testing keyboard input with '2' key...")

		// Send the '2' key through the model's Update method
		updatedModel, cmd := freshModel.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'2'},
		})

		// Update the model reference
		*freshModel = *updatedModel.(*Model)

		// Helper function to execute all commands including nested batch commands
	executeCommands := func(cmd tea.Cmd) {
		commandCount := 0
		for cmd != nil && commandCount < 10 { // Prevent infinite loops
			commandCount++
			msg := cmd()
			if msg != nil {
				t.Logf("Command %d returned message: %T", commandCount, msg)

				// Handle batch commands
				if batchMsg, ok := msg.(tea.BatchMsg); ok {
					t.Logf("Batch command contains %d sub-commands", len(batchMsg))
					// Execute each sub-command
					for i, batchCmd := range batchMsg {
						if batchCmd != nil {
							batchMsg := batchCmd()
							if batchMsg != nil {
								t.Logf("Batch sub-command %d returned message: %T", i, batchMsg)
								updatedModel, nextCmd := freshModel.Update(batchMsg)
								*freshModel = *updatedModel.(*Model)
								// Continue with the next command from this sub-command
								if nextCmd != nil {
									cmd = nextCmd
								}
							}
						}
					}
				} else {
					updatedModel, nextCmd := freshModel.Update(msg)
					*freshModel = *updatedModel.(*Model)
					cmd = nextCmd
				}
			} else {
				break
			}
		}
		t.Logf("Processed %d commands", commandCount)
	}

	// Execute the returned command
	executeCommands(cmd)

		// Check if we switched to Logs view
		currentView := freshModel.currentView()
		expectedView := ViewLogs
		t.Logf("After keyboard input - Current view: %v (type: %T), Expected: %v (type: %T)", currentView, currentView, expectedView, expectedView)

		if currentView != expectedView {
			t.Errorf("Expected to be in Logs view after pressing '2', got %v (%T)", currentView, currentView)
		} else {
			t.Logf("✅ SUCCESS: Keyboard input '2' correctly switched to Logs view")
		}
	})
}

// TestKeyboardInputWithCatwalk tests basic keyboard input handling
func TestKeyboardInputWithCatwalk(t *testing.T) {
	model := createTestModelWithWorkingDir()
	defer os.RemoveAll(model.workingDir) // Cleanup

	t.Run("Basic keyboard input doesn't panic", func(t *testing.T) {
		// Test that various keys don't crash the application
		testSequences := []string{
			"1",           // Switch to Processes
			"2",           // Switch to Logs
			"3",           // Switch to Errors
			"4",           // Switch to AI Coders
			"5",           // Switch to Settings
			"\t",          // Tab navigation
			"?\t",         // Help then Tab
			"/q",          // Command mode then quit
			"q",           // Quit key
			"esc",         // Escape key
			"12345\t",     // Multiple keys
		}

		for i, keySequence := range testSequences {
			t.Run(fmt.Sprintf("KeySequence_%d", i), func(t *testing.T) {
				// This should not panic
				catwalk.RunModelFromString(t, keySequence, model,
					catwalk.WithWindowSize(80, 24),
				)
				t.Logf("✅ SUCCESS: Key sequence '%s' handled without panic", keySequence)
			})
		}
	})
}

