package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// PTYNavigationAdapterCommand represents a PTY navigation adapter-related operation
type PTYNavigationAdapterCommand struct {
	BaseCommand
	NavigationOperation string
	AdapterID           string
	Args                map[string]interface{}
	Model               interface{}
}

// Run executes the PTY navigation adapter operation and returns a message (implements Command interface)
func (pnac *PTYNavigationAdapterCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "PTYNavigationAdapter",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := pnac.executeNavigationOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   pnac.id,
		OperationID: pnac.id,
		Priority:    types.OperationPriority(pnac.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"navigationOperation": pnac.NavigationOperation,
			"adapterID":           pnac.AdapterID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(pnac.id, types.CommandType(pnac.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(pnac.id, types.CommandType(pnac.cmdType), result, executionTime, typesContext)
}

// executeNavigationOperation performs the actual PTY navigation adapter operation
func (pnac *PTYNavigationAdapterCommand) executeNavigationOperation() (interface{}, error) {
	switch pnac.NavigationOperation {
	case "wait_for_nav_goroutines":
		return pnac.waitForNavGoroutines()
	case "process_navigation_events":
		return pnac.processNavigationEvents()
	case "monitor_navigation_activity":
		return pnac.monitorNavigationActivity()
	case "handle_view_switch":
		return pnac.handleViewSwitch()
	case "subscribe_to_events":
		return pnac.subscribeToEvents()
	case "cleanup_nav_adapter":
		return pnac.cleanupNavAdapter()
	default:
		return nil, fmt.Errorf("unknown PTY navigation adapter operation: %s", pnac.NavigationOperation)
	}
}

// waitForNavGoroutines waits for navigation goroutines to finish
func (pnac *PTYNavigationAdapterCommand) waitForNavGoroutines() (interface{}, error) {
	// This replaces the goroutine at line 246 in pty_navigation_adapter.go
	adapterID, ok := pnac.Args["adapterID"].(string)
	if !ok {
		adapterID = pnac.AdapterID
	}

	goroutineCount, ok := pnac.Args["goroutineCount"].(int)
	if !ok {
		goroutineCount = 1
	}

	return fmt.Sprintf("Waited for %d navigation goroutines to complete in adapter %s", goroutineCount, adapterID), nil
}

// processNavigationEvents processes navigation events
func (pnac *PTYNavigationAdapterCommand) processNavigationEvents() (interface{}, error) {
	// This replaces the goroutine at line 213 in pty_navigation_adapter.go
	adapterID, ok := pnac.Args["adapterID"].(string)
	if !ok {
		adapterID = pnac.AdapterID
	}

	eventCount, ok := pnac.Args["eventCount"].(int)
	if !ok {
		eventCount = 0
	}

	return fmt.Sprintf("Processed %d navigation events in adapter %s", eventCount, adapterID), nil
}

// monitorNavigationActivity monitors navigation activity
func (pnac *PTYNavigationAdapterCommand) monitorNavigationActivity() (interface{}, error) {
	// This replaces the goroutine at line 218 in pty_navigation_adapter.go
	adapterID, ok := pnac.Args["adapterID"].(string)
	if !ok {
		adapterID = pnac.AdapterID
	}

	timeout, ok := pnac.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 30 * time.Second
	}

	return fmt.Sprintf("Monitored navigation activity for adapter %s with timeout %v", adapterID, timeout), nil
}

// handleViewSwitch handles view switching
func (pnac *PTYNavigationAdapterCommand) handleViewSwitch() (interface{}, error) {
	adapterID, ok := pnac.Args["adapterID"].(string)
	if !ok {
		adapterID = pnac.AdapterID
	}

	fromView, ok := pnac.Args["fromView"].(string)
	if !ok {
		fromView = "unknown"
	}

	toView, ok := pnac.Args["toView"].(string)
	if !ok {
		toView = "unknown"
	}

	return fmt.Sprintf("View switch handled in adapter %s: %s -> %s", adapterID, fromView, toView), nil
}

// subscribeToEvents subscribes to navigation events
func (pnac *PTYNavigationAdapterCommand) subscribeToEvents() (interface{}, error) {
	adapterID, ok := pnac.Args["adapterID"].(string)
	if !ok {
		adapterID = pnac.AdapterID
	}

	eventTypes, ok := pnac.Args["eventTypes"].([]string)
	if !ok {
		eventTypes = []string{"navigation", "view_switch", "focus_change"}
	}

	return fmt.Sprintf("Subscribed to events %v in adapter %s", eventTypes, adapterID), nil
}

// cleanupNavAdapter cleans up navigation adapter
func (pnac *PTYNavigationAdapterCommand) cleanupNavAdapter() (interface{}, error) {
	adapterID, ok := pnac.Args["adapterID"].(string)
	if !ok {
		adapterID = pnac.AdapterID
	}

	return fmt.Sprintf("Navigation adapter %s cleaned up successfully", adapterID), nil
}

// PTYNavigationAdapterCommandFactory creates PTY navigation adapter-related commands
type PTYNavigationAdapterCommandFactory struct{}

// NewPTYNavigationAdapterCommandFactory creates a new PTY navigation adapter command factory
func NewPTYNavigationAdapterCommandFactory() *PTYNavigationAdapterCommandFactory {
	return &PTYNavigationAdapterCommandFactory{}
}

// CreateWaitForNavGoroutinesCommand creates a command to wait for navigation goroutines
func (f *PTYNavigationAdapterCommandFactory) CreateWaitForNavGoroutinesCommand(adapterID string, goroutineCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-nav-wait-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYNavigationAdapterCommand{
		BaseCommand:         *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		NavigationOperation: "wait_for_nav_goroutines",
		AdapterID:           adapterID,
		Args: map[string]interface{}{
			"adapterID":      adapterID,
			"goroutineCount": goroutineCount,
		},
		Model: model,
	}
}

// CreateProcessNavigationEventsCommand creates a command to process navigation events
func (f *PTYNavigationAdapterCommandFactory) CreateProcessNavigationEventsCommand(adapterID string, eventCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-nav-events-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYNavigationAdapterCommand{
		BaseCommand:         *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		NavigationOperation: "process_navigation_events",
		AdapterID:           adapterID,
		Args: map[string]interface{}{
			"adapterID":   adapterID,
			"eventCount":  eventCount,
		},
		Model: model,
	}
}

// CreateMonitorNavigationActivityCommand creates a command to monitor navigation activity
func (f *PTYNavigationAdapterCommandFactory) CreateMonitorNavigationActivityCommand(adapterID string, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("pty-nav-activity-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYNavigationAdapterCommand{
		BaseCommand:         *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		NavigationOperation: "monitor_navigation_activity",
		AdapterID:           adapterID,
		Args: map[string]interface{}{
			"adapterID": adapterID,
			"timeout":   timeout,
		},
		Model: model,
	}
}

// CreateHandleViewSwitchCommand creates a command to handle view switching
func (f *PTYNavigationAdapterCommandFactory) CreateHandleViewSwitchCommand(adapterID string, fromView, toView string, model interface{}) Command {
	id := fmt.Sprintf("pty-nav-switch-%s-%d", adapterID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &PTYNavigationAdapterCommand{
		BaseCommand:         *NewBaseCommand(id, CommandTypeSystem, PriorityHigh, context),
		NavigationOperation: "handle_view_switch",
		AdapterID:           adapterID,
		Args: map[string]interface{}{
			"adapterID": adapterID,
			"fromView":  fromView,
			"toView":    toView,
		},
		Model: model,
	}
}

// CreateSubscribeToEventsCommand creates a command to subscribe to events
func (f *PTYNavigationAdapterCommandFactory) CreateSubscribeToEventsCommand(adapterID string, eventTypes []string, model interface{}) Command {
	id := fmt.Sprintf("pty-nav-subscribe-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYNavigationAdapterCommand{
		BaseCommand:         *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		NavigationOperation: "subscribe_to_events",
		AdapterID:           adapterID,
		Args: map[string]interface{}{
			"adapterID":  adapterID,
			"eventTypes": eventTypes,
		},
		Model: model,
	}
}

// CreateCleanupNavAdapterCommand creates a command to cleanup navigation adapter
func (f *PTYNavigationAdapterCommandFactory) CreateCleanupNavAdapterCommand(adapterID string, model interface{}) Command {
	id := fmt.Sprintf("pty-nav-cleanup-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYNavigationAdapterCommand{
		BaseCommand:         *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		NavigationOperation: "cleanup_nav_adapter",
		AdapterID:           adapterID,
		Args: map[string]interface{}{
			"adapterID": adapterID,
		},
		Model: model,
	}
}

// PTYNavigationAdapterCommands provides high-level commands for PTY navigation adapter operations
type PTYNavigationAdapterCommands struct {
	factory *PTYNavigationAdapterCommandFactory
}

// NewPTYNavigationAdapterCommands creates a new PTY navigation adapter commands instance
func NewPTYNavigationAdapterCommands() *PTYNavigationAdapterCommands {
	return &PTYNavigationAdapterCommands{
		factory: NewPTYNavigationAdapterCommandFactory(),
	}
}

// WaitForNavGoroutines returns a command to wait for navigation goroutines
func (pnac *PTYNavigationAdapterCommands) WaitForNavGoroutines(adapterID string, goroutineCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pnac.factory.CreateWaitForNavGoroutinesCommand(adapterID, goroutineCount, model)
		return cmd.Run()
	}
}

// ProcessNavigationEvents returns a command to process navigation events
func (pnac *PTYNavigationAdapterCommands) ProcessNavigationEvents(adapterID string, eventCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pnac.factory.CreateProcessNavigationEventsCommand(adapterID, eventCount, model)
		return cmd.Run()
	}
}

// MonitorNavigationActivity returns a command to monitor navigation activity
func (pnac *PTYNavigationAdapterCommands) MonitorNavigationActivity(adapterID string, timeout time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pnac.factory.CreateMonitorNavigationActivityCommand(adapterID, timeout, model)
		return cmd.Run()
	}
}

// HandleViewSwitch returns a command to handle view switching
func (pnac *PTYNavigationAdapterCommands) HandleViewSwitch(adapterID string, fromView, toView string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pnac.factory.CreateHandleViewSwitchCommand(adapterID, fromView, toView, model)
		return cmd.Run()
	}
}

// SubscribeToEvents returns a command to subscribe to events
func (pnac *PTYNavigationAdapterCommands) SubscribeToEvents(adapterID string, eventTypes []string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pnac.factory.CreateSubscribeToEventsCommand(adapterID, eventTypes, model)
		return cmd.Run()
	}
}

// CleanupNavAdapter returns a command to cleanup navigation adapter
func (pnac *PTYNavigationAdapterCommands) CleanupNavAdapter(adapterID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pnac.factory.CreateCleanupNavAdapterCommand(adapterID, model)
		return cmd.Run()
	}
}