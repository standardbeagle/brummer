package services

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// PTYIntegrationExample demonstrates how to use the PTY input integration
// for realistic keyboard input testing in TUI applications
type PTYIntegrationExample struct {
	ptyLayer     *pty.PTYLayer
	inputService *InputService
	adapter      *PTYInputAdapter
	validator    *KeyboardRoutingValidator
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewPTYIntegrationExample creates a new example instance
func NewPTYIntegrationExample() (*PTYIntegrationExample, error) {
	ctx, cancel := context.WithCancel(context.Background())

	example := &PTYIntegrationExample{
		ctx:    ctx,
		cancel: cancel,
	}

	return example, nil
}

// RunExample demonstrates the complete PTY integration workflow
func (e *PTYIntegrationExample) RunExample() error {
	fmt.Println("=== PTY Input Integration Example ===")

	// Step 1: Create and allocate PTY layer
	if err := e.setupPTYLayer(); err != nil {
		return fmt.Errorf("failed to setup PTY layer: %w", err)
	}

	// Step 2: Create InputService with PTY support
	if err := e.setupInputService(); err != nil {
		return fmt.Errorf("failed to setup InputService: %w", err)
	}

	// Step 3: Create and start PTY input adapter
	if err := e.setupPTYAdapter(); err != nil {
		return fmt.Errorf("failed to setup PTY adapter: %w", err)
	}

	// Step 4: Setup keyboard routing validation
	if err := e.setupValidation(); err != nil {
		return fmt.Errorf("failed to setup validation: %w", err)
	}

	// Step 5: Demonstrate keyboard input simulation
	if err := e.demonstrateKeyboardInput(); err != nil {
		return fmt.Errorf("failed to demonstrate keyboard input: %w", err)
	}

	// Step 6: Run validation
	if err := e.runValidation(); err != nil {
		return fmt.Errorf("failed to run validation: %w", err)
	}

	// Step 7: Show statistics
	e.showStatistics()

	fmt.Println("=== PTY Integration Example Completed Successfully ===")
	return nil
}

// setupPTYLayer creates and configures the PTY layer
func (e *PTYIntegrationExample) setupPTYLayer() error {
	fmt.Println("\n1. Setting up PTY layer...")

	// Create PTY configuration
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.Size.Rows = 24
	ptyConfig.Size.Cols = 80
	ptyConfig.Timeout = 30 * time.Second
	ptyConfig.EnableDebug = true

	// Create PTY layer
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	if err != nil {
		return fmt.Errorf("failed to create PTY layer: %w", err)
	}

	// Allocate PTY
	if err := ptyLayer.Allocate(); err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	e.ptyLayer = ptyLayer
	fmt.Printf("   ✓ PTY layer allocated: %dx%d\n", ptyConfig.Size.Cols, ptyConfig.Size.Rows)
	return nil
}

// setupInputService creates and configures the InputService with PTY support
func (e *PTYIntegrationExample) setupInputService() error {
	fmt.Println("\n2. Setting up InputService with PTY support...")

	// Create InputService with PTY layer
	inputService := NewInputServiceWithPTY(e.ptyLayer)
	if inputService == nil {
		return fmt.Errorf("failed to create InputService with PTY")
	}

	// Start InputService
	if err := inputService.Start(e.ctx); err != nil {
		return fmt.Errorf("failed to start InputService: %w", err)
	}

	// Register some example handlers
	if err := e.registerExampleHandlers(inputService); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	e.inputService = inputService
	fmt.Printf("   ✓ InputService started with PTY enabled\n")
	fmt.Printf("   ✓ PTY mode: %s\n", inputService.GetPTYMode())
	fmt.Printf("   ✓ Registered %d handlers\n", inputService.GetHandlerCount())
	return nil
}

// registerExampleHandlers registers example keyboard handlers
func (e *PTYIntegrationExample) registerExampleHandlers(inputService *InputService) error {
	// Global handler for 'q' key
	err := inputService.SubscribeToGlobalKeys(e.ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'q' {
			fmt.Println("   → Global handler processed 'q' key (quit)")
			return func() tea.Msg { return tea.QuitMsg{} }
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Context-specific handler for navigation keys
	err = inputService.SubscribeWithContextKeys(e.ctx, "navigation", func(keyMsg tea.KeyMsg) tea.Cmd {
		switch keyMsg.Type {
		case tea.KeyUp:
			fmt.Println("   → Navigation handler processed 'up' key")
		case tea.KeyDown:
			fmt.Println("   → Navigation handler processed 'down' key")
		case tea.KeyLeft:
			fmt.Println("   → Navigation handler processed 'left' key")
		case tea.KeyRight:
			fmt.Println("   → Navigation handler processed 'right' key")
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Register key bindings
	testBindings := []KeyBinding{
		{
			Key:      tea.Key{Type: tea.KeyRunes, Runes: []rune{'h'}},
			Action:   "help",
			Context:  "global",
			Priority: 1,
		},
		{
			Key:      tea.Key{Type: tea.KeyCtrlS},
			Action:   "save",
			Context:  "global",
			Priority: 1,
		},
		{
			Key:      tea.Key{Type: tea.KeyTab},
			Action:   "next_view",
			Context:  "navigation",
			Priority: 1,
		},
	}

	for _, binding := range testBindings {
		if err := inputService.RegisterKeyBinding(e.ctx, binding); err != nil {
			return fmt.Errorf("failed to register binding for %v: %w", binding.Key, err)
		}
	}

	return nil
}

// setupPTYAdapter creates and configures the PTY input adapter
func (e *PTYIntegrationExample) setupPTYAdapter() error {
	fmt.Println("\n3. Setting up PTY input adapter...")

	// Create adapter configuration
	adapterConfig := DefaultPTYInputAdapterConfig()
	adapterConfig.EnableDebug = true
	adapterConfig.EventBufferSize = 100
	adapterConfig.ForwardUnknownKeys = true

	// Create adapter
	adapter, err := NewPTYInputAdapter(e.ptyLayer, e.inputService, adapterConfig)
	if err != nil {
		return fmt.Errorf("failed to create PTY input adapter: %w", err)
	}

	// Start adapter
	if err := adapter.Start(); err != nil {
		return fmt.Errorf("failed to start PTY input adapter: %w", err)
	}

	e.adapter = adapter
	fmt.Printf("   ✓ PTY input adapter started\n")
	fmt.Printf("   ✓ Adapter active: %v\n", adapter.IsActive())
	return nil
}

// setupValidation creates and configures the keyboard routing validator
func (e *PTYIntegrationExample) setupValidation() error {
	fmt.Println("\n4. Setting up keyboard routing validation...")

	// Create validator configuration
	validatorConfig := DefaultKeyboardRoutingValidatorConfig()
	validatorConfig.EnableDebug = true
	validatorConfig.MaxTestCases = 20
	validatorConfig.ValidateTiming = true
	validatorConfig.CheckEventFlow = true

	// Create validator
	validator, err := NewKeyboardRoutingValidator(e.ptyLayer, e.inputService, validatorConfig)
	if err != nil {
		return fmt.Errorf("failed to create keyboard routing validator: %w", err)
	}

	// Start validator
	if err := validator.Start(); err != nil {
		return fmt.Errorf("failed to start keyboard routing validator: %w", err)
	}

	e.validator = validator
	fmt.Printf("   ✓ Keyboard routing validator started\n")
	fmt.Printf("   ✓ Validator active: %v\n", validator.IsActive())
	return nil
}

// demonstrateKeyboardInput shows how to simulate keyboard input through PTY
func (e *PTYIntegrationExample) demonstrateKeyboardInput() error {
	fmt.Println("\n5. Demonstrating keyboard input simulation...")

	// Test various key types
	testInputs := []struct {
		name string
		key  tea.KeyMsg
		ctx  string
	}{
		{"Character input", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}, "test"},
		{"Navigation (up)", tea.KeyMsg{Type: tea.KeyUp}, "navigation"},
		{"Navigation (down)", tea.KeyMsg{Type: tea.KeyDown}, "navigation"},
		{"Control key (Ctrl+S)", tea.KeyMsg{Type: tea.KeyCtrlS}, "test"},
		{"Tab key", tea.KeyMsg{Type: tea.KeyTab}, "navigation"},
		{"Quit key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, "test"},
		{"Enter key", tea.KeyMsg{Type: tea.KeyEnter}, "test"},
	}

	for _, test := range testInputs {
		fmt.Printf("   • Simulating %s...\n", test.name)

		// Method 1: Direct PTY simulation through InputService
		if len(test.key.Runes) == 1 {
			// For single character keys
			err := e.inputService.SimulatePTYKeyPress(test.key)
			if err != nil {
				fmt.Printf("     ✗ PTY simulation failed: %v\n", err)
			} else {
				fmt.Printf("     ✓ PTY simulation succeeded\n")
			}
		} else {
			// For multi-character sequences or special keys
			for _, r := range test.key.Runes {
				key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
				err := e.inputService.SimulatePTYKeyPress(key)
				if err != nil {
					fmt.Printf("     ✗ PTY simulation failed for '%c': %v\n", r, err)
				}
			}
		}

		// Method 2: Through PTY adapter
		err := e.adapter.SimulateKeyPress(test.key)
		if err != nil {
			fmt.Printf("     ✗ Adapter simulation failed: %v\n", err)
		} else {
			fmt.Printf("     ✓ Adapter simulation succeeded\n")
		}

		// Small delay to see the effects
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// runValidation executes keyboard routing validation
func (e *PTYIntegrationExample) runValidation() error {
	fmt.Println("\n6. Running keyboard routing validation...")

	// Validate PTY->InputService path
	fmt.Println("   • Validating PTY->InputService path...")
	if err := e.validator.ValidatePTYToInputServicePath(); err != nil {
		fmt.Printf("     ✗ PTY->InputService validation failed: %v\n", err)
		return err
	}
	fmt.Printf("     ✓ PTY->InputService validation passed\n")

	// Validate event flow
	fmt.Println("   • Validating event flow...")
	if err := e.validator.ValidateEventFlow(); err != nil {
		fmt.Printf("     ✗ Event flow validation failed: %v\n", err)
		return err
	}
	fmt.Printf("     ✓ Event flow validation passed\n")

	// Run comprehensive validation
	fmt.Println("   • Running comprehensive validation...")
	results, err := e.validator.ValidateKeyboardRouting()
	if err != nil {
		fmt.Printf("     ✗ Comprehensive validation failed: %v\n", err)
		return err
	}

	passed := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			passed++
		} else {
			failed++
			if result.Error != nil {
				fmt.Printf("     ✗ Test '%s' failed: %v\n", result.TestCase.Name, result.Error)
			}
		}
	}

	fmt.Printf("     ✓ Comprehensive validation completed: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		return fmt.Errorf("%d validation tests failed", failed)
	}

	return nil
}

// showStatistics displays final statistics
func (e *PTYIntegrationExample) showStatistics() {
	fmt.Println("\n7. Final Statistics...")

	// InputService statistics
	if e.inputService != nil {
		inputStats := e.inputService.GetPTYStatistics()
		fmt.Printf("   InputService:\n")
		fmt.Printf("     • PTY enabled: %v\n", inputStats.PTYEnabled)
		fmt.Printf("     • PTY mode: %s\n", inputStats.PTYMode)
		fmt.Printf("     • PTY layer active: %v\n", inputStats.PTYLayerActive)
		fmt.Printf("     • PTY keys processed: %d\n", inputStats.PTYKeyCount)
		fmt.Printf("     • Total keys processed: %d\n", inputStats.TotalKeyCount)
		fmt.Printf("     • Active handlers: %d\n", e.inputService.GetHandlerCount())

		status := e.inputService.GetServiceStatus()
		fmt.Printf("     • Service status: %s (%s)\n", status.Status, status.Details)
	}

	// PTY adapter statistics
	if e.adapter != nil {
		adapterStats := e.adapter.GetStatistics()
		fmt.Printf("   PTY Adapter:\n")
		fmt.Printf("     • Active: %v\n", adapterStats.Active)
		fmt.Printf("     • Keys processed: %d\n", adapterStats.KeyPressCount)
		fmt.Printf("     • Errors: %d\n", adapterStats.ErrorCount)
		if !adapterStats.LastActivity.IsZero() {
			fmt.Printf("     • Last activity: %v\n", adapterStats.LastActivity.Format(time.RFC3339))
		}
	}

	// Validator statistics
	if e.validator != nil {
		validatorStats := e.validator.GetValidationStatistics()
		fmt.Printf("   Validator:\n")
		fmt.Printf("     • Validation runs: %d\n", validatorStats.ValidationRun)
		fmt.Printf("     • Total tests: %d\n", validatorStats.TotalTests)
		fmt.Printf("     • Passed tests: %d\n", validatorStats.PassedTests)
		fmt.Printf("     • Failed tests: %d\n", validatorStats.FailedTests)
		fmt.Printf("     • Success rate: %.2f%%\n", validatorStats.SuccessRate*100)
		fmt.Printf("     • Average test duration: %v\n", validatorStats.AverageTestDuration)
		fmt.Printf("     • Max test duration: %v\n", validatorStats.MaxTestDuration)
	}

	// PTY layer statistics
	if e.ptyLayer != nil {
		fmt.Printf("   PTY Layer:\n")
		fmt.Printf("     • Allocated: %v\n", e.ptyLayer.IsAllocated())
		inputBuffer := e.ptyLayer.GetInputBuffer()
		outputBuffer := e.ptyLayer.GetOutputBuffer()
		fmt.Printf("     • Input buffer size: %d bytes\n", len(inputBuffer))
		fmt.Printf("     • Output buffer size: %d bytes\n", len(outputBuffer))
	}
}

// Cleanup cleans up all resources
func (e *PTYIntegrationExample) Cleanup() {
	fmt.Println("\nCleaning up resources...")

	if e.validator != nil && e.validator.IsActive() {
		if err := e.validator.Stop(); err != nil {
			log.Printf("Warning: failed to stop validator: %v", err)
		}
	}

	if e.adapter != nil && e.adapter.IsActive() {
		if err := e.adapter.Stop(); err != nil {
			log.Printf("Warning: failed to stop adapter: %v", err)
		}
	}

	if e.inputService != nil && e.inputService.IsRunning() {
		if err := e.inputService.Shutdown(e.ctx); err != nil {
			log.Printf("Warning: failed to shutdown InputService: %v", err)
		}
	}

	if e.ptyLayer != nil {
		if err := e.ptyLayer.Cleanup(); err != nil {
			log.Printf("Warning: failed to cleanup PTY layer: %v", err)
		}
	}

	if e.cancel != nil {
		e.cancel()
	}

	fmt.Println("✓ Cleanup completed")
}

// RunPTYIntegrationExample is a convenience function to run the complete example
func RunPTYIntegrationExample() error {
	example, err := NewPTYIntegrationExample()
	if err != nil {
		return fmt.Errorf("failed to create example: %w", err)
	}
	defer example.Cleanup()

	return example.RunExample()
}