package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

func main() {
	fmt.Println("=== Testing Subscription System ===")
	
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
	
	fmt.Println("Init() returned a command")
	
	// Execute the command in a goroutine (simulating Bubble Tea runtime)
	msgChan := make(chan tea.Msg, 10)
	var subscriptionCmd tea.Cmd
	
	go func() {
		// This simulates what Bubble Tea does - execute the command
		msg := initCmd()
		if msg != nil {
			// Check if it's a batch message
			if batch, ok := msg.(tea.BatchMsg); ok {
				fmt.Printf("Init returned BatchMsg with %d commands\n", len(batch))
				// Find the subscription command
				for _, cmd := range batch {
					if cmd != nil {
						// Try executing to see what type of message it produces
						testMsg := cmd()
						fmt.Printf("  Command produces: %T\n", testMsg)
						// Keep the subscription command for later
						subscriptionCmd = cmd
					}
				}
			}
			msgChan <- msg
		}
	}()
	
	// Now add some logs to trigger the subscription
	fmt.Println("\n=== Adding logs to trigger subscription ===")
	logStore := shell.GetLogStore()
	
	// Populate process name cache
	shell.GetProcessNameCacheSync().Store("test-1", "Test Process")
	
	// Add a log
	logStore.Add("test-1", "Test Process", "First log line", false)
	
	// Notify the subscription
	shell.NotifyNewLogs()
	fmt.Println("Called NotifyNewLogs()")
	
	// Wait for message from subscription
	fmt.Println("\n=== Waiting for subscription message ===")
	select {
	case msg := <-msgChan:
		switch m := msg.(type) {
		case components.LogsUpdateMsg:
			fmt.Printf("✓ Received LogsUpdateMsg with %d entries\n", len(m.Entries))
			for i, entry := range m.Entries {
				fmt.Printf("  Log %d: %s\n", i, entry.Content)
			}
		case components.ContinueLogSubscriptionMsg:
			fmt.Println("✓ Received ContinueLogSubscriptionMsg (no new logs)")
		default:
			fmt.Printf("? Received unexpected message type: %T\n", m)
		}
	case <-time.After(2 * time.Second):
		fmt.Println("✗ TIMEOUT: No message received from subscription after 2 seconds")
	}
	
	// Try adding more logs
	fmt.Println("\n=== Adding more logs ===")
	logStore.Add("test-1", "Test Process", "Second log line", false)
	logStore.Add("test-2", "Another Process", "Error occurred!", true)
	shell.NotifyNewLogs()
	
	// The subscription command needs to be re-executed after each message
	// This is what Bubble Tea does automatically
	fmt.Println("Note: In real Bubble Tea, the subscription would be re-executed after each message")
}