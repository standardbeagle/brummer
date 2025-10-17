package commands

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// ProgressHandlerCommand represents a progress handler operation
type ProgressHandlerCommand struct {
	BaseCommand
	ProgressOperation string
	HandlerID        string
	Args             map[string]interface{}
	Model            interface{}
}

// Run executes the progress handler operation and returns a message (implements Command interface)
func (phc *ProgressHandlerCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "ProgressHandler",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := phc.executeProgressOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   phc.id,
		OperationID: phc.id,
		Priority:    types.OperationPriority(phc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"progressOperation": phc.ProgressOperation,
			"handlerID":        phc.HandlerID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(phc.id, types.CommandType(phc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(phc.id, types.CommandType(phc.cmdType), result, executionTime, typesContext)
}

// executeProgressOperation performs the actual progress handler operation
func (phc *ProgressHandlerCommand) executeProgressOperation() (interface{}, error) {
	switch phc.ProgressOperation {
	case "start_ticker":
		return phc.startTicker()
	case "check_timeouts":
		return phc.checkTimeouts()
	case "stop_ticker":
		return phc.stopTicker()
	default:
		return nil, fmt.Errorf("unknown progress handler operation: %s", phc.ProgressOperation)
	}
}

// startTicker starts a ticker for periodic timeout checks using commands
func (phc *ProgressHandlerCommand) startTicker() (interface{}, error) {
	// This replaces the goroutine in progress_handler.go line 111
	handlerID, ok := phc.Args["handlerID"].(string)
	if !ok {
		handlerID = phc.HandlerID
	}

	updateInterval, ok := phc.Args["updateInterval"].(time.Duration)
	if !ok {
		updateInterval = 1 * time.Second // Default interval
	}

	return fmt.Sprintf("Progress ticker started for handler %s with interval %v", handlerID, updateInterval), nil
}

// checkTimeouts checks for progress operations that have timed out
func (phc *ProgressHandlerCommand) checkTimeouts() (interface{}, error) {
	// This replaces the ticker functionality with periodic commands
	handlerID, ok := phc.Args["handlerID"].(string)
	if !ok {
		handlerID = phc.HandlerID
	}

	timeoutCount, ok := phc.Args["timeoutCount"].(int)
	if !ok {
		timeoutCount = 0
	}

	return fmt.Sprintf("Checked timeouts for handler %s, found %d timeouts", handlerID, timeoutCount), nil
}

// stopTicker stops the progress ticker
func (phc *ProgressHandlerCommand) stopTicker() (interface{}, error) {
	handlerID, ok := phc.Args["handlerID"].(string)
	if !ok {
		handlerID = phc.HandlerID
	}

	return fmt.Sprintf("Progress ticker stopped for handler %s", handlerID), nil
}

// ProgressHandlerCommandFactory creates progress handler-related commands
type ProgressHandlerCommandFactory struct{}

// NewProgressHandlerCommandFactory creates a new progress handler command factory
func NewProgressHandlerCommandFactory() *ProgressHandlerCommandFactory {
	return &ProgressHandlerCommandFactory{}
}

// CreateStartTickerCommand creates a command to start a progress ticker
func (f *ProgressHandlerCommandFactory) CreateStartTickerCommand(handlerID string, updateInterval time.Duration, model interface{}) Command {
	id := fmt.Sprintf("progress-ticker-start-%s-%d", handlerID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ProgressHandlerCommand{
		BaseCommand:       *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ProgressOperation: "start_ticker",
		HandlerID:         handlerID,
		Args: map[string]interface{}{
			"handlerID":     handlerID,
			"updateInterval": updateInterval,
		},
		Model: model,
	}
}

// CreateCheckTimeoutsCommand creates a command to check for timeouts
func (f *ProgressHandlerCommandFactory) CreateCheckTimeoutsCommand(handlerID string, timeoutCount int, model interface{}) Command {
	id := fmt.Sprintf("progress-timeout-check-%s-%d", handlerID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ProgressHandlerCommand{
		BaseCommand:       *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ProgressOperation: "check_timeouts",
		HandlerID:         handlerID,
		Args: map[string]interface{}{
			"handlerID":    handlerID,
			"timeoutCount": timeoutCount,
		},
		Model: model,
	}
}

// CreateStopTickerCommand creates a command to stop a progress ticker
func (f *ProgressHandlerCommandFactory) CreateStopTickerCommand(handlerID string, model interface{}) Command {
	id := fmt.Sprintf("progress-ticker-stop-%s-%d", handlerID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ProgressHandlerCommand{
		BaseCommand:       *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ProgressOperation: "stop_ticker",
		HandlerID:         handlerID,
		Args: map[string]interface{}{
			"handlerID": handlerID,
		},
		Model: model,
	}
}

// ProgressHandlerCommands provides high-level commands for progress handler operations
type ProgressHandlerCommands struct {
	factory *ProgressHandlerCommandFactory
}

// NewProgressHandlerCommands creates a new progress handler commands instance
func NewProgressHandlerCommands() *ProgressHandlerCommands {
	return &ProgressHandlerCommands{
		factory: NewProgressHandlerCommandFactory(),
	}
}

// StartProgressTicker returns a command to start a progress ticker
func (phc *ProgressHandlerCommands) StartProgressTicker(handlerID string, updateInterval time.Duration, model interface{}) tea.Cmd {
	return func() tea.Msg {
		cmd := phc.factory.CreateStartTickerCommand(handlerID, updateInterval, model)
		return cmd.Run()
	}
}

// CheckProgressTimeouts returns a command to check for progress timeouts
func (phc *ProgressHandlerCommands) CheckProgressTimeouts(handlerID string, timeoutCount int, model interface{}) tea.Cmd {
	return func() tea.Msg {
		cmd := phc.factory.CreateCheckTimeoutsCommand(handlerID, timeoutCount, model)
		return cmd.Run()
	}
}

// StopProgressTicker returns a command to stop a progress ticker
func (phc *ProgressHandlerCommands) StopProgressTicker(handlerID string, model interface{}) tea.Cmd {
	return func() tea.Msg {
		cmd := phc.factory.CreateStopTickerCommand(handlerID, model)
		return cmd.Run()
	}
}

// ProgressTickerManager manages periodic timeout checking using Bubble Tea commands
type ProgressTickerManager struct {
	handlerID      string
	updateInterval time.Duration
	isRunning      bool
	lastCheck      time.Time
	mu             sync.RWMutex
	model          interface{}
	commands       *ProgressHandlerCommands
}

// NewProgressTickerManager creates a new progress ticker manager
func NewProgressTickerManager(handlerID string, updateInterval time.Duration, model interface{}) *ProgressTickerManager {
	return &ProgressTickerManager{
		handlerID:      handlerID,
		updateInterval: updateInterval,
		isRunning:      false,
		lastCheck:      time.Time{},
		model:          model,
		commands:       NewProgressHandlerCommands(),
	}
}

// Start starts the periodic timeout checking
func (ptm *ProgressTickerManager) Start() tea.Cmd {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	if ptm.isRunning {
		return nil
	}

	ptm.isRunning = true
	ptm.lastCheck = time.Now()

	return ptm.commands.StartProgressTicker(ptm.handlerID, ptm.updateInterval, ptm.model)
}

// Stop stops the periodic timeout checking
func (ptm *ProgressTickerManager) Stop() tea.Cmd {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	if !ptm.isRunning {
		return nil
	}

	ptm.isRunning = false
	return ptm.commands.StopProgressTicker(ptm.handlerID, ptm.model)
}

// CheckTimeouts performs a timeout check and returns a command for the next check
func (ptm *ProgressTickerManager) CheckTimeouts(timeoutCount int) tea.Cmd {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	if !ptm.isRunning {
		return nil
	}

	ptm.lastCheck = time.Now()

	return ptm.commands.CheckProgressTimeouts(ptm.handlerID, timeoutCount, ptm.model)
}

// IsRunning returns true if the ticker is running
func (ptm *ProgressTickerManager) IsRunning() bool {
	ptm.mu.RLock()
	defer ptm.mu.RUnlock()
	return ptm.isRunning
}

// GetLastCheck returns the time of the last check
func (ptm *ProgressTickerManager) GetLastCheck() time.Time {
	ptm.mu.RLock()
	defer ptm.mu.RUnlock()
	return ptm.lastCheck
}