package services

import (
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// ExamplePTYNavigationIntegration demonstrates how to use the PTY-NavigationService integration
// for comprehensive TUI testing with realistic view switching simulation.
func ExamplePTYNavigationIntegration() {
	// This example shows how to set up and use the PTY-NavigationService integration
	// for testing view switching in TUI applications without requiring a real TTY.

	// Create event bus for communication
	eventBus := events.NewEventBus()

	// Create navigation service (the same one used in production)
	navigationService := NewNavigationService(eventBus)

	// Create PTY layer for terminal simulation
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.Size = pty.PTYSize{Rows: 24, Cols: 80}
	ptyConfig.EnableDebug = true

	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	if err != nil {
		log.Fatalf("Failed to create PTY layer: %v", err)
	}

	// Allocate PTY resources
	err = ptyLayer.Allocate()
	if err != nil {
		log.Fatalf("Failed to allocate PTY: %v", err)
	}
	defer ptyLayer.Cleanup()

	// Create PTY navigation adapter to connect PTY with NavigationService
	adapterConfig := DefaultPTYNavigationAdapterConfig()
	adapterConfig.EnableDebug = true
	adapterConfig.EnableStatistics = true
	adapterConfig.ValidateNumberKeys = true
	adapterConfig.ValidateTabCycling = true

	adapter, err := NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
	if err != nil {
		log.Fatalf("Failed to create PTY navigation adapter: %v", err)
	}

	// Start the adapter to begin PTY-NavigationService integration
	err = adapter.Start()
	if err != nil {
		log.Fatalf("Failed to start PTY navigation adapter: %v", err)
	}
	defer adapter.Stop()

	// Create validator for comprehensive testing
	validatorConfig := DefaultPTYViewSwitchingValidatorConfig()
	validatorConfig.EnableDebug = true
	validatorConfig.ValidateStateChanges = true
	validatorConfig.ValidatePerformance = true

	validator, err := NewPTYViewSwitchingValidator(ptyLayer, adapter, validatorConfig)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	fmt.Println("🔗 PTY-NavigationService Integration Example")
	fmt.Println("===========================================")

	// Example 1: Test number key navigation
	fmt.Println("\n1. Testing Number Key Navigation:")
	for i := 1; i <= 5; i++ {
		result := validator.ValidateNumberKey(i, getExpectedViewForNumberKey(i))
		if result.Success {
			fmt.Printf("   ✅ Number key %d -> %s (duration: %v)\n", i, result.ActualView, result.Duration)
		} else {
			fmt.Printf("   ❌ Number key %d failed: %s\n", i, result.Error)
		}
	}

	// Example 2: Test tab cycling
	fmt.Println("\n2. Testing Tab Cycling:")
	cyclingResult := validator.ValidateTabCycling(5)
	if cyclingResult.Success {
		fmt.Printf("   ✅ Tab cycling completed: visited %d unique views (duration: %v)\n",
			cyclingResult.Details["unique_views_visited"], cyclingResult.Duration)
	} else {
		fmt.Printf("   ❌ Tab cycling failed: %s\n", cyclingResult.Error)
	}

	// Example 3: Test key sequences
	fmt.Println("\n3. Testing Key Sequences:")
	keySequence := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'2'}}, // Switch to Logs
		{Type: tea.KeyTab},                        // Cycle to next
		{Type: tea.KeyRunes, Runes: []rune{'4'}}, // Switch to AICoders
	}

	sequenceResult := validator.ValidateKeySequence(keySequence, messages.AICodersView, "number_key")
	if sequenceResult.Success {
		fmt.Printf("   ✅ Key sequence completed: final view %s (duration: %v)\n",
			sequenceResult.ActualView, sequenceResult.Duration)
	} else {
		fmt.Printf("   ❌ Key sequence failed: %s\n", sequenceResult.Error)
	}

	// Example 4: Show statistics
	fmt.Println("\n4. Navigation Statistics:")
	adapterStats := adapter.GetStatistics()
	fmt.Printf("   Total key presses: %d\n", adapterStats.KeyPressCount)
	fmt.Printf("   Total view switches: %d\n", adapterStats.SwitchCount)
	fmt.Printf("   Total tab presses: %d\n", adapterStats.TabPresses)
	fmt.Printf("   View history length: %d\n", adapterStats.ViewHistoryLength)

	validatorStats := validator.GetStatistics()
	fmt.Printf("   Total validations: %d\n", validatorStats.TotalValidations)
	fmt.Printf("   Successful validations: %d\n", validatorStats.SuccessfulValidations)
	fmt.Printf("   Average validation time: %v\n", validatorStats.AverageValidationTime)

	fmt.Println("\n✅ PTY-NavigationService integration example completed successfully")
}

// ExamplePTYNavigationTesting demonstrates a complete testing workflow
func ExamplePTYNavigationTesting() {
	// This example shows a typical testing workflow using the PTY-NavigationService integration

	fmt.Println("🧪 PTY Navigation Testing Example")
	fmt.Println("=================================")

	// Setup test environment
	eventBus := events.NewEventBus()
	navigationService := NewNavigationService(eventBus)

	// Create and configure PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.EnableDebug = false // Disable debug for cleaner output
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	if err != nil {
		log.Fatalf("Failed to create PTY layer: %v", err)
	}

	err = ptyLayer.Allocate()
	if err != nil {
		log.Fatalf("Failed to allocate PTY: %v", err)
	}
	defer ptyLayer.Cleanup()

	// Create adapter with testing configuration
	adapterConfig := DefaultPTYNavigationAdapterConfig()
	adapterConfig.EnableStatistics = true

	adapter, err := NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	err = adapter.Start()
	if err != nil {
		log.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop()

	// Create state tracker for detailed monitoring
	trackerConfig := DefaultPTYNavigationStateTrackerConfig()
	trackerConfig.TrackDurations = true
	trackerConfig.TrackErrors = true

	tracker := NewPTYNavigationStateTracker(adapter.GetCurrentView(), trackerConfig)

	// Test workflow
	fmt.Println("\n🔄 Running test workflow...")

	// Test 1: Number key navigation
	fmt.Println("Test 1: Number key navigation")
	testKeys := []int{1, 3, 5, 2, 4}
	for _, key := range testKeys {
		startTime := time.Now()

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('0' + key)}}
		err := adapter.SimulateKeyPress(keyMsg)
		if err != nil {
			fmt.Printf("   ❌ Failed to press key %d: %v\n", key, err)
			continue
		}

		// Wait for navigation to complete
		time.Sleep(50 * time.Millisecond)

		currentView := adapter.GetCurrentView()
		expectedView := getExpectedViewForNumberKey(key)
		duration := time.Since(startTime)

		// Record transition in tracker
		previousView := tracker.GetPreviousView()
		err = tracker.RecordTransition(string(previousView), string(currentView), "number_key", duration, string(rune('0'+key)))
		if err != nil {
			fmt.Printf("   ⚠️  Failed to record transition: %v\n", err)
		}

		if currentView == expectedView {
			fmt.Printf("   ✅ Key %d -> %s (%v)\n", key, currentView, duration)
		} else {
			fmt.Printf("   ❌ Key %d -> %s (expected %s)\n", key, currentView, expectedView)
		}
	}

	// Test 2: Tab cycling
	fmt.Println("\nTest 2: Tab cycling")
	for i := 0; i < 3; i++ {
		startTime := time.Now()

		err := adapter.SimulateKeyPress(tea.KeyMsg{Type: tea.KeyTab})
		if err != nil {
			fmt.Printf("   ❌ Failed to press tab: %v\n", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		currentView := adapter.GetCurrentView()
		duration := time.Since(startTime)

		// Record transition in tracker
		previousView := tracker.GetPreviousView()
		err = tracker.RecordTransition(string(previousView), string(currentView), "cycle", duration, "\t")
		if err != nil {
			fmt.Printf("   ⚠️  Failed to record transition: %v\n", err)
		}

		fmt.Printf("   ✅ Tab %d -> %s (%v)\n", i+1, currentView, duration)
	}

	// Test 3: Rapid key sequence
	fmt.Println("\nTest 3: Rapid key sequence")
	rapidSequence := []string{"1", "\t", "3", "\t", "5"}

	startTime := time.Now()
	for _, keyStr := range rapidSequence {
		var keyMsg tea.KeyMsg
		switch keyStr {
		case "\t":
			keyMsg = tea.KeyMsg{Type: tea.KeyTab}
		default:
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(keyStr[0])}}
		}

		err := adapter.SimulateKeyPress(keyMsg)
		if err != nil {
			fmt.Printf("   ❌ Failed to press key %s: %v\n", keyStr, err)
			continue
		}
		time.Sleep(10 * time.Millisecond) // Small delay for rapid sequence
	}
	totalDuration := time.Since(startTime)

	// Wait for final navigation
	time.Sleep(100 * time.Millisecond)
	finalView := adapter.GetCurrentView()
	fmt.Printf("   ✅ Rapid sequence completed -> %s (total: %v)\n", finalView, totalDuration)

	// Test 4: State validation
	fmt.Println("\nTest 4: State validation")
	err = tracker.ValidateCurrentState()
	if err != nil {
		fmt.Printf("   ❌ State validation failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ State validation passed\n")
	}

	// Show final statistics
	fmt.Println("\n📊 Final Statistics:")
	adapterStats := adapter.GetStatistics()
	trackerStats := tracker.GetStatistics()

	fmt.Printf("   Adapter: %d key presses, %d view switches, %d tab presses\n",
		adapterStats.KeyPressCount, adapterStats.SwitchCount, adapterStats.TabPresses)
	fmt.Printf("   Tracker: %d transitions, %d successful, avg time: %v\n",
		trackerStats.TotalTransitions, trackerStats.SuccessfulTransitions, trackerStats.AverageSwitchTime)

	// View visit counts
	fmt.Println("   View visits:")
	for view, count := range trackerStats.ViewVisitCounts {
		fmt.Printf("     %s: %d times\n", view, count)
	}

	fmt.Println("\n✅ PTY navigation testing example completed")
}

// getExpectedViewForNumberKey returns the expected view for a number key
func getExpectedViewForNumberKey(key int) messages.ViewType {
	switch key {
	case 1:
		return messages.ProcessesView
	case 2:
		return messages.LogsView
	case 3:
		return messages.ErrorsView
	case 4:
		return messages.AICodersView
	case 5:
		return messages.SettingsView
	default:
		return ""
	}
}

// ExamplePTYNavigationAdvanced demonstrates advanced usage patterns
func ExamplePTYNavigationAdvanced() {
	fmt.Println("🚀 Advanced PTY Navigation Example")
	fmt.Println("=================================")

	// Setup with advanced configuration
	eventBus := events.NewEventBus()
	navigationService := NewNavigationService(eventBus)

	// Configure PTY layer for advanced testing
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.Size = pty.PTYSize{Rows: 50, Cols: 120} // Larger terminal
	ptyConfig.EnableDebug = false

	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	if err != nil {
		log.Fatalf("Failed to create PTY layer: %v", err)
	}

	err = ptyLayer.Allocate()
	if err != nil {
		log.Fatalf("Failed to allocate PTY: %v", err)
	}
	defer ptyLayer.Cleanup()

	// Advanced adapter configuration
	adapterConfig := DefaultPTYNavigationAdapterConfig()
	adapterConfig.EnableStatistics = true
	adapterConfig.TrackViewHistory = true
	adapterConfig.RecordTriggers = true
	adapterConfig.ValidateNumberKeys = true
	adapterConfig.ValidateTabCycling = true

	adapter, err := NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	err = adapter.Start()
	if err != nil {
		log.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop()

	// Advanced validator configuration
	validatorConfig := DefaultPTYViewSwitchingValidatorConfig()
	validatorConfig.ValidatePerformance = true
	validatorConfig.TestRapidSwitching = true
	validatorConfig.TestKeySequences = true
	validatorConfig.TestStateConsistency = true

	validator, err := NewPTYViewSwitchingValidator(ptyLayer, adapter, validatorConfig)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	fmt.Println("\n🎯 Running advanced validation tests...")

	// Performance test
	fmt.Println("Performance test: 100 rapid switches")
	startTime := time.Now()

	for i := 0; i < 100; i++ {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('1' + (i % 5))}}
		adapter.SimulateKeyPress(key)
		time.Sleep(1 * time.Millisecond) // Very rapid
	}

	duration := time.Since(startTime)
	fmt.Printf("   Completed 100 switches in %v (avg: %v per switch)\n",
		duration, duration/100)

	// Complex key sequence test
	fmt.Println("\nComplex key sequence test:")
	complexSequence := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'2'}}, // Logs
		{Type: tea.KeyTab},                        // Next
		{Type: tea.KeyTab},                        // Next
		{Type: tea.KeyRunes, Runes: []rune{'1'}}, // Processes
		{Type: tea.KeyTab},                        // Next
		{Type: tea.KeyRunes, Runes: []rune{'5'}}, // Settings
	}

	result := validator.ValidateKeySequence(complexSequence, messages.SettingsView, "number_key")
	if result.Success {
		fmt.Printf("   ✅ Complex sequence succeeded (duration: %v)\n", result.Duration)
	} else {
		fmt.Printf("   ❌ Complex sequence failed: %s\n", result.Error)
	}

	// State consistency test
	fmt.Println("\nState consistency test:")
	consistencyResult := validator.ValidateStateConsistency()
	if consistencyResult.Success {
		fmt.Printf("   ✅ State consistency validated\n")
	} else {
		fmt.Printf("   ❌ State consistency failed: %s\n", consistencyResult.Error)
	}

	// Show comprehensive statistics
	fmt.Println("\n📈 Comprehensive Statistics:")
	adapterStats := adapter.GetStatistics()
	validatorStats := validator.GetStatistics()

	fmt.Printf("Navigation Performance:\n")
	fmt.Printf("   Total operations: %d\n", adapterStats.KeyPressCount)
	fmt.Printf("   View switches: %d\n", adapterStats.SwitchCount)
	fmt.Printf("   Success rate: %.1f%%\n", float64(validatorStats.SuccessfulValidations)/float64(validatorStats.TotalValidations)*100)

	fmt.Printf("\nTiming Analysis:\n")
	fmt.Printf("   Average validation time: %v\n", validatorStats.AverageValidationTime)
	fmt.Printf("   Max validation time: %v\n", validatorStats.MaxValidationTime)
	fmt.Printf("   Min validation time: %v\n", validatorStats.MinValidationTime)

	fmt.Printf("\nUsage Patterns:\n")
	for trigger, count := range adapterStats.SwitchTriggers {
		fmt.Printf("   %s: %d times\n", trigger, count)
	}

	fmt.Println("\n✅ Advanced PTY navigation example completed")
}