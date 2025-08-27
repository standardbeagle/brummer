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
	fmt.Println("=== Testing Fix: Initial Logs Load ===")
	
	// Create dependencies
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	if err != nil {
		panic(err)
	}

	// Create IntegratedShell
	shell := components.NewIntegratedShell(processMgr)
	
	// Simulate logs being added BEFORE Init() is called
	// This simulates what happens when a process starts quickly
	fmt.Println("\n1. Adding logs BEFORE Init()...")
	logStore := shell.GetLogStore()
	shell.GetProcessNameCacheSync().Store("pre-init", "Pre-Init Process")
	logStore.Add("pre-init", "Pre-Init Process", "Log added before Init", false)
	logStore.Add("pre-init", "Pre-Init Process", "Second pre-init log", false)
	
	// Wait for async processing in log store
	time.Sleep(50 * time.Millisecond)
	
	// Verify logs are in the store
	allLogs := logStore.GetAll()
	fmt.Printf("   Added %d logs to store\n", len(allLogs))
	
	// Now call Init() - with our fix, it should load existing logs
	fmt.Println("\n2. Calling Init()...")
	initCmd := shell.Init()
	
	// Execute the init command
	msg := initCmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		fmt.Printf("ERROR: Expected BatchMsg, got %T\n", msg)
		return
	}
	
	fmt.Printf("   Init returned BatchMsg with %d commands\n", len(batch))
	
	// Check each command
	foundInitialLogs := false
	for i, cmd := range batch {
		if cmd != nil {
			// Check if this command returns LogsUpdateMsg immediately
			// (non-blocking commands will return immediately)
			done := make(chan tea.Msg, 1)
			go func(c tea.Cmd) {
				msg := c()
				done <- msg
			}(cmd)
			
			select {
			case msg := <-done:
				if logMsg, ok := msg.(components.LogsUpdateMsg); ok {
					fmt.Printf("\n✅ SUCCESS: Command %d returned initial LogsUpdateMsg with %d entries!\n", i, len(logMsg.Entries))
					for j, entry := range logMsg.Entries {
						fmt.Printf("   Initial log %d: %s\n", j, entry.Content)
					}
					foundInitialLogs = true
				} else {
					fmt.Printf("   Command %d: %T (not initial logs)\n", i, msg)
				}
			case <-time.After(10 * time.Millisecond):
				fmt.Printf("   Command %d: Blocking (likely subscription)\n", i)
			}
		}
	}
	
	if !foundInitialLogs {
		fmt.Println("\n❌ FAILURE: No initial logs found in Init() commands")
		fmt.Println("   The fix might not be working!")
	} else {
		fmt.Println("\n✅ FIX VERIFIED: Initial logs are sent immediately on Init()")
	}
	
	// Test 2: Verify subscription still works for new logs
	fmt.Println("\n3. Testing subscription for new logs...")
	
	// Find the subscription command (it will be blocking)
	var subscriptionCmd tea.Cmd
	for _, cmd := range batch {
		if cmd != nil {
			done := make(chan bool, 1)
			go func(c tea.Cmd) {
				// This will block if it's the subscription
				c()
				done <- true
			}(cmd)
			
			select {
			case <-done:
				// Non-blocking command
			case <-time.After(10 * time.Millisecond):
				// This is likely the subscription
				subscriptionCmd = cmd
				break
			}
		}
	}
	
	if subscriptionCmd != nil {
		fmt.Println("   Found subscription command")
		
		// Start subscription in background
		subMsgChan := make(chan tea.Msg, 1)
		go func() {
			msg := subscriptionCmd()
			subMsgChan <- msg
		}()
		
		// Add new logs
		fmt.Println("   Adding new logs...")
		shell.GetProcessNameCacheSync().Store("post-init", "Post-Init Process")
		logStore.Add("post-init", "Post-Init Process", "New log after Init", false)
		shell.NotifyNewLogs()
		
		// Wait for subscription to respond
		select {
		case msg := <-subMsgChan:
			if logMsg, ok := msg.(components.LogsUpdateMsg); ok {
				fmt.Printf("\n✅ Subscription works: Received %d new logs\n", len(logMsg.Entries))
				for i, entry := range logMsg.Entries {
					fmt.Printf("   New log %d: %s\n", i, entry.Content)
				}
			}
		case <-time.After(200 * time.Millisecond):
			fmt.Println("\n⚠️  Subscription didn't respond to new logs")
		}
	}
	
	fmt.Println("\n=== Test Complete ===")
}