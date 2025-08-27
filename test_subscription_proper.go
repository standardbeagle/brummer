package main

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

func main() {
	fmt.Println("=== Testing Subscription System Properly ===")
	
	// Create dependencies
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	if err != nil {
		panic(err)
	}

	// Create IntegratedShell
	shell := components.NewIntegratedShell(processMgr)
	
	// Get the Init command
	initCmd := shell.Init()
	if initCmd == nil {
		fmt.Println("ERROR: Init() returned nil")
		return
	}
	
	// Parse the batch command
	msg := initCmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		fmt.Printf("ERROR: Expected BatchMsg, got %T\n", msg)
		return
	}
	
	fmt.Printf("Init returned BatchMsg with %d commands\n", len(batch))
	
	// Find and execute the subscription command
	var wg sync.WaitGroup
	subscriptionRunning := false
	
	for i, cmd := range batch {
		if cmd != nil {
			// Start each command in a goroutine
			wg.Add(1)
			go func(idx int, c tea.Cmd) {
				defer wg.Done()
				
				// Execute the command
				// The subscription command will block waiting for updates
				fmt.Printf("Starting command %d...\n", idx)
				msg := c()
				
				// This will only return when the command produces a message
				fmt.Printf("Command %d produced: %T\n", idx, msg)
				
				// Check if it's a log update
				if logMsg, ok := msg.(components.LogsUpdateMsg); ok {
					fmt.Printf("✓ LogsUpdateMsg received with %d entries!\n", len(logMsg.Entries))
					for j, entry := range logMsg.Entries {
						fmt.Printf("  Log %d: %s\n", j, entry.Content)
					}
				}
			}(i, cmd)
			
			// Assume one of these is the subscription
			subscriptionRunning = true
		}
	}
	
	// Give commands time to start
	time.Sleep(100 * time.Millisecond)
	
	if !subscriptionRunning {
		fmt.Println("ERROR: No subscription command found")
		return
	}
	
	// Now add some logs to trigger the subscription
	fmt.Println("\n=== Adding logs to trigger subscription ===")
	logStore := shell.GetLogStore()
	
	// Populate process name cache
	shell.GetProcessNameCacheSync().Store("test-1", "Test Process")
	
	// Add a log
	logStore.Add("test-1", "Test Process", "First log line", false)
	
	// Notify the subscription - this should trigger the waiting Subscribe() command
	fmt.Println("Calling NotifyNewLogs()...")
	shell.NotifyNewLogs()
	
	// Wait a bit for the debounce delay (50ms default)
	fmt.Println("Waiting for debounce...")
	time.Sleep(100 * time.Millisecond)
	
	// Add more logs
	fmt.Println("\n=== Adding more logs ===")
	logStore.Add("test-1", "Test Process", "Second log line", false)
	logStore.Add("test-2", "Another Process", "Error occurred!", true)
	shell.GetProcessNameCacheSync().Store("test-2", "Another Process")
	shell.NotifyNewLogs()
	
	// Wait for subscription to process
	fmt.Println("Waiting for subscription to process...")
	time.Sleep(100 * time.Millisecond)
	
	// Give time for any messages to be printed
	time.Sleep(500 * time.Millisecond)
	
	fmt.Println("\n=== Test Complete ===")
	fmt.Println("Note: The subscription command is likely still waiting for more updates.")
	fmt.Println("In a real Bubble Tea app, it would be re-executed after each message.")
}