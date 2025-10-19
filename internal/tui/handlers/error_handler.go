package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/commands"
	tea "github.com/charmbracelet/bubbletea"
)

// ErrorHandler manages error handling, classification, and retry logic
type ErrorHandler struct {
	errorHistory      []*ErrorRecord
	errorPatterns    map[string]*ErrorPattern
	retryStrategies  map[string]*RetryStrategy
	mu               sync.RWMutex
	maxHistorySize   int
	retryAttempts    map[string]int
	classifier       *ErrorClassifier
	recoveryActions  map[string]*RecoveryAction
}

// ErrorRecord represents a recorded error with context
type ErrorRecord struct {
	ID           string                  `json:"id"`
	Error        error                   `json:"error"`
	CommandType  commands.CommandType    `json:"commandType"`
	CommandID    string                  `json:"commandId"`
	Timestamp    time.Time               `json:"timestamp"`
	Context      commands.OperationContext `json:"context"`
	Retryable    bool                    `json:"retryable"`
	RetryCount   int                     `json:"retryCount"`
	Severity     ErrorSeverity           `json:"severity"`
	Category     ErrorCategory           `json:"category"`
	Recovered    bool                    `json:"recovered"`
	RecoveryTime time.Duration           `json:"recoveryTime,omitempty"`
}

// ErrorPattern represents a pattern of errors that can be matched
type ErrorPattern struct {
	Name        string            `json:"name"`
	Pattern     string            `json:"pattern"`
	Category    ErrorCategory     `json:"category"`
	Severity    ErrorSeverity     `json:"severity"`
	Retryable   bool              `json:"retryable"`
	Actions     []string          `json:"actions"`
	MatchCount  int               `json:"matchCount"`
	LastMatched time.Time         `json:"lastMatched"`
}

// RetryStrategy defines how to retry specific types of errors
type RetryStrategy struct {
	Name            string        `json:"name"`
	ErrorTypes      []string      `json:"errorTypes"`
	MaxAttempts     int           `json:"maxAttempts"`
	InitialDelay    time.Duration `json:"initialDelay"`
	MaxDelay        time.Duration `json:"maxDelay"`
	BackoffFactor   float64       `json:"backoffFactor"`
	JitterEnabled   bool          `json:"jitterEnabled"`
	CustomCondition func(error) bool `json:"-"`
}

// RecoveryAction defines an action to take for error recovery
type RecoveryAction struct {
	Name        string                 `json:"name"`
	Category    ErrorCategory          `json:"category"`
	Condition   func(error) bool       `json:"-"`
	Action      func(*ErrorRecord) tea.Msg `json:"-"`
	Enabled     bool                   `json:"enabled"`
	SuccessRate float64                `json:"successRate"`
	UsageCount  int                    `json:"usageCount"`
}

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// ErrorCategory represents the category of an error
type ErrorCategory int

const (
	CategoryNetwork ErrorCategory = iota
	CategoryFileSystem
	CategoryProcess
	CategorySystem
	CategoryPermission
	CategoryTimeout
	CategoryConfiguration
	CategoryUnknown
)

// ErrorClassifier classifies errors based on their content
type ErrorClassifier struct {
	patterns map[string]*ErrorPattern
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(maxHistorySize int) *ErrorHandler {
	handler := &ErrorHandler{
		errorHistory:     make([]*ErrorRecord, 0, maxHistorySize),
		errorPatterns:    make(map[string]*ErrorPattern),
		retryStrategies:  make(map[string]*RetryStrategy),
		maxHistorySize:   maxHistorySize,
		retryAttempts:    make(map[string]int),
		recoveryActions:  make(map[string]*RecoveryAction),
		classifier:       NewErrorClassifier(),
	}

	// Initialize default patterns and strategies
	handler.initializeDefaultPatterns()
	handler.initializeDefaultStrategies()
	handler.initializeRecoveryActions()

	return handler
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	classifier := &ErrorClassifier{
		patterns: make(map[string]*ErrorPattern),
	}

	// Initialize default error patterns
	classifier.addPattern(&ErrorPattern{
		Name:      "connection_refused",
		Pattern:   "connection refused",
		Category:  CategoryNetwork,
		Severity:  SeverityMedium,
		Retryable: true,
		Actions:   []string{"retry_with_backoff", "check_network"},
	})

	classifier.addPattern(&ErrorPattern{
		Name:      "timeout",
		Pattern:   "timeout",
		Category:  CategoryTimeout,
		Severity:  SeverityMedium,
		Retryable: true,
		Actions:   []string{"retry_with_longer_timeout", "check_resources"},
	})

	classifier.addPattern(&ErrorPattern{
		Name:      "permission_denied",
		Pattern:   "permission denied",
		Category:  CategoryPermission,
		Severity:  SeverityHigh,
		Retryable: false,
		Actions:   []string{"check_permissions", "request_elevation"},
	})

	classifier.addPattern(&ErrorPattern{
		Name:      "file_not_found",
		Pattern:   "no such file",
		Category:  CategoryFileSystem,
		Severity:  SeverityLow,
		Retryable: false,
		Actions:   []string{"verify_path", "create_file"},
	})

	classifier.addPattern(&ErrorPattern{
		Name:      "command_not_found",
		Pattern:   "command not found",
		Category:  CategoryProcess,
		Severity:  SeverityHigh,
		Retryable: false,
		Actions:   []string{"check_installation", "verify_path"},
	})

	return classifier
}

// addPattern adds an error pattern to the classifier
func (c *ErrorClassifier) addPattern(pattern *ErrorPattern) {
	c.patterns[pattern.Name] = pattern
}

// ClassifyError classifies an error and returns its category and severity
func (c *ErrorClassifier) ClassifyError(err error) (ErrorCategory, ErrorSeverity) {
	if err == nil {
		return CategoryUnknown, SeverityLow
	}

	errorStr := strings.ToLower(err.Error())

	// Check against known patterns
	for _, pattern := range c.patterns {
		if strings.Contains(errorStr, pattern.Pattern) {
			pattern.MatchCount++
			pattern.LastMatched = time.Now()
			return pattern.Category, pattern.Severity
		}
	}

	// Default classification based on content
	if strings.Contains(errorStr, "network") || strings.Contains(errorStr, "connection") {
		return CategoryNetwork, SeverityMedium
	}
	if strings.Contains(errorStr, "permission") || strings.Contains(errorStr, "access denied") {
		return CategoryPermission, SeverityHigh
	}
	if strings.Contains(errorStr, "file") || strings.Contains(errorStr, "directory") {
		return CategoryFileSystem, SeverityLow
	}
	if strings.Contains(errorStr, "process") || strings.Contains(errorStr, "pid") {
		return CategoryProcess, SeverityMedium
	}
	if strings.Contains(errorStr, "timeout") || strings.Contains(errorStr, "deadline") {
		return CategoryTimeout, SeverityMedium
	}

	return CategoryUnknown, SeverityLow
}

// HandleError processes an error and determines the appropriate action
func (h *ErrorHandler) HandleError(commandID string, commandType commands.CommandType, err error, context commands.OperationContext, retryCount int) tea.Msg {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Classify the error
	category, severity := h.classifier.ClassifyError(err)

	// Create error record
	record := &ErrorRecord{
		ID:          fmt.Sprintf("error-%s-%d", commandID, time.Now().UnixNano()),
		Error:       err,
		CommandType: commandType,
		CommandID:   commandID,
		Timestamp:   time.Now(),
		Context:     context,
		Retryable:   h.isRetryable(err, category),
		RetryCount:  retryCount,
		Severity:    severity,
		Category:    category,
		Recovered:   false,
	}

	// Add to history
	h.addToHistory(record)

	// Check for recovery actions
	if recoveryMsg := h.attemptRecovery(record); recoveryMsg != nil {
		record.Recovered = true
		record.RecoveryTime = time.Since(record.Timestamp)
		return recoveryMsg
	}

	// Determine retry strategy
	if record.Retryable && retryCount < h.getMaxRetries(err) {
		h.retryAttempts[commandID] = retryCount + 1
		return h.createRetryMessage(record)
	}

	// No recovery or retry possible, return failure message
	return h.createFailureMessage(record)
}

// addToHistory adds an error record to the history
func (h *ErrorHandler) addToHistory(record *ErrorRecord) {
	if len(h.errorHistory) >= h.maxHistorySize {
		h.errorHistory = h.errorHistory[1:]
	}
	h.errorHistory = append(h.errorHistory, record)
}

// isRetryable determines if an error is retryable
func (h *ErrorHandler) isRetryable(err error, category ErrorCategory) bool {
	switch category {
	case CategoryNetwork, CategoryTimeout:
		return true
	case CategoryFileSystem, CategoryProcess, CategorySystem:
		// Check specific error conditions
		errorStr := strings.ToLower(err.Error())
		if strings.Contains(errorStr, "temporarily unavailable") ||
		   strings.Contains(errorStr, "resource temporarily unavailable") ||
		   strings.Contains(errorStr, "busy") {
			return true
		}
		return false
	case CategoryPermission, CategoryConfiguration:
		return false
	default:
		return false
	}
}

// getMaxRetries returns the maximum number of retries for an error
func (h *ErrorHandler) getMaxRetries(err error) int {
	errorStr := strings.ToLower(err.Error())

	// Check for specific retry strategies
	for _, strategy := range h.retryStrategies {
		for _, errorType := range strategy.ErrorTypes {
			if strings.Contains(errorStr, errorType) {
				return strategy.MaxAttempts
			}
		}
	}

	// Default retry counts by category
	category, _ := h.classifier.ClassifyError(err)
	switch category {
	case CategoryNetwork:
		return 5
	case CategoryTimeout:
		return 3
	case CategoryFileSystem:
		return 2
	default:
		return 1
	}
}

// attemptRecovery attempts to recover from an error using recovery actions
func (h *ErrorHandler) attemptRecovery(record *ErrorRecord) tea.Msg {
	for _, action := range h.recoveryActions {
		if action.Enabled && action.Condition(record.Error) {
			action.UsageCount++
			result := action.Action(record)
			if result != nil {
				// Update success rate
				h.updateActionSuccessRate(action.Name, true)
				return result
			}
			h.updateActionSuccessRate(action.Name, false)
		}
	}
	return nil
}

// updateActionSuccessRate updates the success rate of a recovery action
func (h *ErrorHandler) updateActionSuccessRate(actionName string, success bool) {
	if action, exists := h.recoveryActions[actionName]; exists {
		if success {
			action.SuccessRate = (action.SuccessRate*float64(action.UsageCount-1) + 1.0) / float64(action.UsageCount)
		} else {
			action.SuccessRate = (action.SuccessRate * float64(action.UsageCount-1)) / float64(action.UsageCount)
		}
	}
}

// createRetryMessage creates a retry message for a retryable error
func (h *ErrorHandler) createRetryMessage(record *ErrorRecord) tea.Msg {
	strategy := h.getRetryStrategy(record.Error)
	if strategy == nil {
		strategy = h.getDefaultRetryStrategy(record.Category)
	}

	_ = h.calculateRetryDelay(record.RetryCount, strategy)

	// TODO: Fix type mismatch in messages package
	// return messages.NewRetryAttemptMessage(
	//	fmt.Sprintf("retry-%s-%d", record.CommandID, record.RetryCount+1),
	//	record.CommandID,
	//	record.RetryCount+1,
	//	strategy.MaxAttempts,
	//	record.Error,
	//	delay,
	//	record.Context,
	// )
	return tea.Msg(nil)
}

// createFailureMessage creates a failure message for a non-retryable error
func (h *ErrorHandler) createFailureMessage(record *ErrorRecord) tea.Msg {
	// TODO: Fix type mismatch in messages package
	// return messages.NewCommandFailedMessage(
	//	record.CommandID,
	//	record.CommandType,
	//	record.Error,
	//	time.Since(record.Timestamp),
	//	record.Context,
	//	false, // Not retryable
	// )
	return tea.Msg(nil)
}

// getRetryStrategy returns the retry strategy for an error
func (h *ErrorHandler) getRetryStrategy(err error) *RetryStrategy {
	errorStr := strings.ToLower(err.Error())

	for _, strategy := range h.retryStrategies {
		for _, errorType := range strategy.ErrorTypes {
			if strings.Contains(errorStr, errorType) {
				return strategy
			}
		}
	}

	return nil
}

// getDefaultRetryStrategy returns the default retry strategy for a category
func (h *ErrorHandler) getDefaultRetryStrategy(category ErrorCategory) *RetryStrategy {
	switch category {
	case CategoryNetwork:
		return h.retryStrategies["network"]
	case CategoryTimeout:
		return h.retryStrategies["timeout"]
	case CategoryFileSystem:
		return h.retryStrategies["filesystem"]
	default:
		return h.retryStrategies["default"]
	}
}

// calculateRetryDelay calculates the delay before the next retry
func (h *ErrorHandler) calculateRetryDelay(attempt int, strategy *RetryStrategy) time.Duration {
	delay := strategy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * strategy.BackoffFactor)
		if delay > strategy.MaxDelay {
			delay = strategy.MaxDelay
			break
		}
	}

	// Add jitter if enabled
	if strategy.JitterEnabled {
		jitter := time.Duration(float64(delay) * 0.1 * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))
		delay += jitter
	}

	return delay
}

// initializeDefaultPatterns initializes default error patterns
func (h *ErrorHandler) initializeDefaultPatterns() {
	patterns := []*ErrorPattern{
		{
			Name:      "network_unreachable",
			Pattern:   "network unreachable",
			Category:  CategoryNetwork,
			Severity:  SeverityHigh,
			Retryable: true,
			Actions:   []string{"retry_with_backoff", "check_connectivity"},
		},
		{
			Name:      "dns_resolution_failed",
			Pattern:   "no such host",
			Category:  CategoryNetwork,
			Severity:  SeverityMedium,
			Retryable: true,
			Actions:   []string{"retry_with_different_dns", "check_dns_config"},
		},
		{
			Name:      "disk_full",
			Pattern:   "no space left",
			Category:  CategoryFileSystem,
			Severity:  SeverityCritical,
			Retryable: false,
			Actions:   []string{"cleanup_disk", "alert_user"},
		},
		{
			Name:      "process_already_running",
			Pattern:   "already in use",
			Category:  CategoryProcess,
			Severity:  SeverityMedium,
			Retryable: false,
			Actions:   []string{"kill_existing_process", "use_different_port"},
		},
	}

	for _, pattern := range patterns {
		h.classifier.addPattern(pattern)
	}
}

// initializeDefaultStrategies initializes default retry strategies
func (h *ErrorHandler) initializeDefaultStrategies() {
	strategies := map[string]*RetryStrategy{
		"network": {
			Name:          "network",
			ErrorTypes:    []string{"connection", "network", "timeout"},
			MaxAttempts:   5,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 1.5,
			JitterEnabled: true,
		},
		"timeout": {
			Name:          "timeout",
			ErrorTypes:    []string{"timeout", "deadline"},
			MaxAttempts:   3,
			InitialDelay:  2 * time.Second,
			MaxDelay:      60 * time.Second,
			BackoffFactor: 2.0,
			JitterEnabled: true,
		},
		"filesystem": {
			Name:          "filesystem",
			ErrorTypes:    []string{"file", "directory", "disk"},
			MaxAttempts:   2,
			InitialDelay:  500 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			JitterEnabled: false,
		},
		"default": {
			Name:          "default",
			ErrorTypes:    []string{},
			MaxAttempts:   1,
			InitialDelay:  1 * time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 1.5,
			JitterEnabled: false,
		},
	}

	for name, strategy := range strategies {
		h.retryStrategies[name] = strategy
	}
}

// initializeRecoveryActions initializes default recovery actions
func (h *ErrorHandler) initializeRecoveryActions() {
	actions := map[string]*RecoveryAction{
		"retry_with_backoff": {
			Name:     "retry_with_backoff",
			Category: CategoryNetwork,
			Condition: func(err error) bool {
				errorStr := strings.ToLower(err.Error())
				return strings.Contains(errorStr, "connection") || strings.Contains(errorStr, "timeout")
			},
			Action: func(record *ErrorRecord) tea.Msg {
				// TODO: Fix type mismatch in messages package
				// return messages.NewRetryAttemptMessage(
				//	fmt.Sprintf("auto-retry-%s", record.ID),
				//	record.CommandID,
				//	record.RetryCount+1,
				//	h.getMaxRetries(record.Error),
				//	record.Error,
				//	time.Second,
				//	record.Context,
				// )
				return tea.Msg(nil)
			},
			Enabled: true,
		},
		"check_permissions": {
			Name:     "check_permissions",
			Category: CategoryPermission,
			Condition: func(err error) bool {
				errorStr := strings.ToLower(err.Error())
				return strings.Contains(errorStr, "permission denied") || strings.Contains(errorStr, "access denied")
			},
			Action: func(record *ErrorRecord) tea.Msg {
				// TODO: Fix type mismatch in messages package
				// return messages.NewStateUpdateMessage(
				//	fmt.Sprintf("perm-check-%s", record.CommandID),
				//	"permissions",
				//	record.Context.Version,
				//	record.Context.Version+1,
				//	"error",
				//	fmt.Sprintf("Permission error: %s. Please check file/directory permissions.", record.Error.Error()),
				//	record.Context,
				// )
				return tea.Msg(nil)
			},
			Enabled: true,
		},
		"cleanup_cache": {
			Name:     "cleanup_cache",
			Category: CategoryFileSystem,
			Condition: func(err error) bool {
				errorStr := strings.ToLower(err.Error())
				return strings.Contains(errorStr, "no space left") || strings.Contains(errorStr, "disk full")
			},
			Action: func(record *ErrorRecord) tea.Msg {
				// TODO: Fix type mismatch in messages package
				// return messages.NewResourceCleanupMessage(
				//	fmt.Sprintf("cache-cleanup-%s", record.CommandID),
				//	"cache",
				//	"disk_space",
				//	"disk full, cleaning cache",
				//	record.Context,
				// )
				return tea.Msg(nil)
			},
			Enabled: true,
		},
	}

	for name, action := range actions {
		h.recoveryActions[name] = action
	}
}

// Public accessor methods

// GetErrorHistory returns recent error records
func (h *ErrorHandler) GetErrorHistory(limit int) []*ErrorRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit >= len(h.errorHistory) {
		result := make([]*ErrorRecord, len(h.errorHistory))
		copy(result, h.errorHistory)
		return result
	}

	result := make([]*ErrorRecord, limit)
	start := len(h.errorHistory) - limit
	copy(result, h.errorHistory[start:])
	return result
}

// GetErrorStats returns error statistics
func (h *ErrorHandler) GetErrorStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"total_errors":   len(h.errorHistory),
		"by_category":    make(map[ErrorCategory]int),
		"by_severity":    make(map[ErrorSeverity]int),
		"by_command":     make(map[commands.CommandType]int),
		"retryable_rate": 0.0,
		"recovery_rate":  0.0,
	}

	retryableCount := 0
	recoveredCount := 0

	for _, record := range h.errorHistory {
		stats["by_category"].(map[ErrorCategory]int)[record.Category]++
		stats["by_severity"].(map[ErrorSeverity]int)[record.Severity]++
		stats["by_command"].(map[commands.CommandType]int)[record.CommandType]++

		if record.Retryable {
			retryableCount++
		}
		if record.Recovered {
			recoveredCount++
		}
	}

	if len(h.errorHistory) > 0 {
		stats["retryable_rate"] = float64(retryableCount) / float64(len(h.errorHistory))
		stats["recovery_rate"] = float64(recoveredCount) / float64(len(h.errorHistory))
	}

	return stats
}

// GetRetryAttempts returns retry attempt counts
func (h *ErrorHandler) GetRetryAttempts() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]int)
	for id, count := range h.retryAttempts {
		result[id] = count
	}
	return result
}

// ClearHistory clears the error history
func (h *ErrorHandler) ClearHistory() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errorHistory = h.errorHistory[:0]
	h.retryAttempts = make(map[string]int)
}

// ClearRetries clears retry attempt tracking
func (h *ErrorHandler) ClearRetries() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.retryAttempts = make(map[string]int)
}