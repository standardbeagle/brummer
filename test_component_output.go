package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

func main() {
	// Create dependencies
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	if err != nil {
		panic(err)
	}

	// Create IntegratedShell
	shell := components.NewIntegratedShell(processMgr)
	
	// Initialize
	shell.Init()
	
	// Set size
	shell.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	
	// Get initial view
	fmt.Println("=== Initial View ===")
	view := shell.View()
	fmt.Println(view)
	fmt.Println("=== End Initial View ===")
	
	// Add some test logs directly
	logStore := shell.GetLogStore()
	
	// IMPORTANT: Also populate the process name cache
	// This simulates what happens when a real process is started
	shell.GetProcessNameCacheSync().Store("test-1", "Test Process")
	shell.GetProcessNameCacheSync().Store("test-2", "Another Process")
	
	// Now add logs with proper process names
	logStore.Add("test-1", "Test Process", "First log line", false)
	logStore.Add("test-1", "Test Process", "Second log line", false) 
	logStore.Add("test-2", "Another Process", "Error occurred!", true)
	
	// Notify of new logs
	shell.NotifyNewLogs()
	
	// Wait for debounce
	time.Sleep(100 * time.Millisecond)
	
	// Switch to logs view (tab 2)
	shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	
	// Get view with logs
	fmt.Println("\n=== Logs View (Tab 2) ===")
	view = shell.View()
	fmt.Println(view)
	fmt.Println("=== End View ===")
	
	// Check if logs are in the store
	allLogs := logStore.GetAll()
	fmt.Printf("\nLogs in store: %d\n", len(allLogs))
	for i, log := range allLogs {
		fmt.Printf("Log %d: [%s] %s (error=%v)\n", i, log.ProcessID, log.Content, log.IsError)
	}
	
	// Also check what the logsView component sees
	fmt.Println("\n=== Debug: What logsView sees ===")
	// The logsView is private, but we can check via the store
	fmt.Printf("Store has %d total logs\n", len(allLogs))
	fmt.Printf("Process name cache entries: %d\n", len(shell.GetProcessNameCache()))
	
	// Try to trigger an update manually
	fmt.Println("\n=== Trying manual refresh ===")
	
	// The issue is that the subscription system needs Bubble Tea's runtime
	// to work. In a test, we need to manually send the LogsUpdateMsg
	// that would normally come from the subscription.
	
	// Create LogsUpdateMsg with our logs
	// Convert []logs.LogEntry to []*logs.LogEntry
	logPtrs := make([]*logs.LogEntry, len(allLogs))
	for i := range allLogs {
		logPtrs[i] = &allLogs[i]
	}
	
	logsUpdateMsg := components.LogsUpdateMsg{
		Entries: logPtrs,
		Append:  false, // Replace existing logs
	}
	
	// Send the message to the shell
	shell.Update(logsUpdateMsg)
	
	// Now check if the view updated
	fmt.Println("\n=== Logs View After LogsUpdateMsg ===")
	view = shell.View()
	
	// Print a few lines to see the logs
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if i >= 3 && i <= 10 { // Show lines 4-11 where logs should be
			fmt.Printf("Line %d: %s\n", i, line)
		}
	}
	
	// Just check if logs are showing
	if strings.Contains(view, "First log line") {
		fmt.Println("\n✓ SUCCESS: Logs are now showing!")
	} else if strings.Contains(view, "No logs available") {
		fmt.Println("\n✗ FAILURE: Still showing 'No logs available'")
	} else {
		fmt.Println("\n? UNKNOWN: View doesn't contain expected strings")
	}
}