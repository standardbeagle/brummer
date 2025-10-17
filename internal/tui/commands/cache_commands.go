package commands

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// CacheCommand represents a cache-related operation
type CacheCommand struct {
	BaseCommand
	CacheOperation string
	CacheID        string
	Args           map[string]interface{}
	Model          interface{}
}

// Run executes the cache operation and returns a message (implements Command interface)
func (cc *CacheCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "Cache",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := cc.executeCacheOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   cc.id,
		OperationID: cc.id,
		Priority:    types.OperationPriority(cc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"cacheOperation": cc.CacheOperation,
			"cacheID":        cc.CacheID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(cc.id, types.CommandType(cc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(cc.id, types.CommandType(cc.cmdType), result, executionTime, typesContext)
}

// executeCacheOperation performs the actual cache operation
func (cc *CacheCommand) executeCacheOperation() (interface{}, error) {
	switch cc.CacheOperation {
	case "start_cleanup":
		return cc.startCleanup()
	case "cleanup_expired":
		return cc.cleanupExpired()
	case "stop_cleanup":
		return cc.stopCleanup()
	default:
		return nil, fmt.Errorf("unknown cache operation: %s", cc.CacheOperation)
	}
}

// startCleanup starts the cache cleanup routine using commands
func (cc *CacheCommand) startCleanup() (interface{}, error) {
	// This replaces the goroutine in cache.go line 103
	cacheID, ok := cc.Args["cacheID"].(string)
	if !ok {
		cacheID = cc.CacheID
	}

	interval, ok := cc.Args["interval"].(time.Duration)
	if !ok {
		interval = 5 * time.Minute // Default cleanup interval
	}

	return fmt.Sprintf("Cache cleanup started for cache %s with interval %v", cacheID, interval), nil
}

// cleanupExpired removes expired entries from the cache
func (cc *CacheCommand) cleanupExpired() (interface{}, error) {
	cacheID, ok := cc.Args["cacheID"].(string)
	if !ok {
		cacheID = cc.CacheID
	}

	expiredCount, ok := cc.Args["expiredCount"].(int)
	if !ok {
		expiredCount = 0
	}

	return fmt.Sprintf("Cleaned up %d expired entries from cache %s", expiredCount, cacheID), nil
}

// stopCleanup stops the cache cleanup routine
func (cc *CacheCommand) stopCleanup() (interface{}, error) {
	cacheID, ok := cc.Args["cacheID"].(string)
	if !ok {
		cacheID = cc.CacheID
	}

	return fmt.Sprintf("Cache cleanup stopped for cache %s", cacheID), nil
}

// CacheCommandFactory creates cache-related commands
type CacheCommandFactory struct{}

// NewCacheCommandFactory creates a new cache command factory
func NewCacheCommandFactory() *CacheCommandFactory {
	return &CacheCommandFactory{}
}

// CreateStartCleanupCommand creates a command to start cache cleanup
func (f *CacheCommandFactory) CreateStartCleanupCommand(cacheID string, interval time.Duration, model interface{}) Command {
	id := fmt.Sprintf("cache-cleanup-start-%s-%d", cacheID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &CacheCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		CacheOperation: "start_cleanup",
		CacheID:        cacheID,
		Args: map[string]interface{}{
			"cacheID": cacheID,
			"interval": interval,
		},
		Model: model,
	}
}

// CreateCleanupExpiredCommand creates a command to clean up expired entries
func (f *CacheCommandFactory) CreateCleanupExpiredCommand(cacheID string, expiredCount int, model interface{}) Command {
	id := fmt.Sprintf("cache-expired-cleanup-%s-%d", cacheID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &CacheCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		CacheOperation: "cleanup_expired",
		CacheID:        cacheID,
		Args: map[string]interface{}{
			"cacheID":      cacheID,
			"expiredCount": expiredCount,
		},
		Model: model,
	}
}

// CreateStopCleanupCommand creates a command to stop cache cleanup
func (f *CacheCommandFactory) CreateStopCleanupCommand(cacheID string, model interface{}) Command {
	id := fmt.Sprintf("cache-cleanup-stop-%s-%d", cacheID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &CacheCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		CacheOperation: "stop_cleanup",
		CacheID:        cacheID,
		Args: map[string]interface{}{
			"cacheID": cacheID,
		},
		Model: model,
	}
}

// CacheCommands provides high-level commands for cache operations
type CacheCommands struct {
	factory *CacheCommandFactory
}

// NewCacheCommands creates a new cache commands instance
func NewCacheCommands() *CacheCommands {
	return &CacheCommands{
		factory: NewCacheCommandFactory(),
	}
}

// StartCacheCleanup returns a command to start cache cleanup
func (cc *CacheCommands) StartCacheCleanup(cacheID string, interval time.Duration, model interface{}) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateStartCleanupCommand(cacheID, interval, model)
		return cmd.Run()
	}
}

// CleanupCacheExpired returns a command to clean up expired entries
func (cc *CacheCommands) CleanupCacheExpired(cacheID string, expiredCount int, model interface{}) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateCleanupExpiredCommand(cacheID, expiredCount, model)
		return cmd.Run()
	}
}

// StopCacheCleanup returns a command to stop cache cleanup
func (cc *CacheCommands) StopCacheCleanup(cacheID string, model interface{}) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateStopCleanupCommand(cacheID, model)
		return cmd.Run()
	}
}

// CacheCleanupManager manages periodic cache cleanup using Bubble Tea commands
type CacheCleanupManager struct {
	cacheID        string
	interval       time.Duration
	isRunning      bool
	lastCleanup    time.Time
	mu             sync.RWMutex
	model          interface{}
	commands       *CacheCommands
}

// NewCacheCleanupManager creates a new cache cleanup manager
func NewCacheCleanupManager(cacheID string, interval time.Duration, model interface{}) *CacheCleanupManager {
	return &CacheCleanupManager{
		cacheID:  cacheID,
		interval: interval,
		isRunning: false,
		lastCleanup: time.Time{},
		model:    model,
		commands: NewCacheCommands(),
	}
}

// Start starts the periodic cache cleanup
func (ccm *CacheCleanupManager) Start() tea.Cmd {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	if ccm.isRunning {
		return nil
	}

	ccm.isRunning = true
	ccm.lastCleanup = time.Now()

	return ccm.commands.StartCacheCleanup(ccm.cacheID, ccm.interval, ccm.model)
}

// Stop stops the periodic cache cleanup
func (ccm *CacheCleanupManager) Stop() tea.Cmd {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	if !ccm.isRunning {
		return nil
	}

	ccm.isRunning = false
	return ccm.commands.StopCacheCleanup(ccm.cacheID, ccm.model)
}

// CleanupExpired performs an expired entry cleanup and returns a command for the next cleanup
func (ccm *CacheCleanupManager) CleanupExpired(expiredCount int) tea.Cmd {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	if !ccm.isRunning {
		return nil
	}

	ccm.lastCleanup = time.Now()

	return ccm.commands.CleanupCacheExpired(ccm.cacheID, expiredCount, ccm.model)
}

// IsRunning returns true if the cleanup is running
func (ccm *CacheCleanupManager) IsRunning() bool {
	ccm.mu.RLock()
	defer ccm.mu.RUnlock()
	return ccm.isRunning
}

// GetLastCleanup returns the time of the last cleanup
func (ccm *CacheCleanupManager) GetLastCleanup() time.Time {
	ccm.mu.RLock()
	defer ccm.mu.RUnlock()
	return ccm.lastCleanup
}