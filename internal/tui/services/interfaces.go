package services

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// ServiceCommandFactory defines the interface for creating service commands
// This ensures all services are stateless and return commands instead of executing directly
type ServiceCommandFactory interface {
	// CreateIOCommand creates a command for I/O operations
	CreateIOCommand(operation interface{}, ctx types.OperationContext) tea.Cmd

	// CreateProcessCommand creates a command for process management
	CreateProcessCommand(operation interface{}, ctx types.OperationContext) tea.Cmd

	// CreateNetworkCommand creates a command for network operations
	CreateNetworkCommand(operation interface{}, ctx types.OperationContext) tea.Cmd

	// CreateSystemCommand creates a command for system operations
	CreateSystemCommand(operation interface{}, ctx types.OperationContext) tea.Cmd
}

// StatelessInputService defines the interface for input handling without goroutines
type StatelessInputService interface {
	// ProcessInput processes input and returns a command for async handling
	ProcessInput(input string, masterData *state.MasterData) tea.Cmd

	// ValidateInput validates input synchronously
	ValidateInput(input string) error

	// GetInputSuggestions returns input suggestions based on current state
	GetInputSuggestions(masterData *state.MasterData) []string
}

// StatelessPTYService defines the interface for PTY operations without goroutines
type StatelessPTYService interface {
	// StartPTY creates a command to start a PTY session
	StartPTY(command []string, env []string, ctx types.OperationContext) tea.Cmd

	// SendInput creates a command to send input to PTY
	SendInput(ptyID string, input string, ctx types.OperationContext) tea.Cmd

	// TerminatePTY creates a command to terminate a PTY session
	TerminatePTY(ptyID string, ctx types.OperationContext) tea.Cmd

	// GetPTYStatus returns PTY status synchronously from master data
	GetPTYStatus(ptyID string, masterData *state.MasterData) interface{}
}

// StatelessNotificationService defines the interface for notifications without goroutines
type StatelessNotificationService interface {
	// ShowNotification creates a command to show a notification
	ShowNotification(notification interface{}, ctx types.OperationContext) tea.Cmd

	// DismissNotification creates a command to dismiss a notification
	DismissNotification(id string, ctx types.OperationContext) tea.Cmd

	// GetActiveNotifications returns active notifications from master data
	GetActiveNotifications(masterData *state.MasterData) []interface{}
}

// StatelessAICoderService defines the interface for AI coder operations without goroutines
type StatelessAICoderService interface {
	// StartSession creates a command to start an AI coder session
	StartSession(config interface{}, ctx types.OperationContext) tea.Cmd

	// SendInput creates a command to send input to AI coder
	SendInput(sessionID string, input string, ctx types.OperationContext) tea.Cmd

	// TerminateSession creates a command to terminate an AI coder session
	TerminateSession(sessionID string, ctx types.OperationContext) tea.Cmd

	// GetSessionStatus returns session status synchronously from master data
	GetSessionStatus(sessionID string, masterData *state.MasterData) interface{}
}

// StatelessNavigationService defines the interface for navigation without goroutines
type StatelessNavigationService interface {
	// Navigate creates a command to navigate to a view
	Navigate(view string, ctx types.OperationContext) tea.Cmd

	// HandleKey creates a command to handle keyboard input
	HandleKey(key tea.KeyMsg, masterData *state.MasterData) tea.Cmd

	// GetCurrentView returns current view synchronously from master data
	GetCurrentView(masterData *state.MasterData) string
}

// StatelessServiceRegistry manages all stateless services
type StatelessServiceRegistry interface {
	// GetCommandFactory returns the service command factory
	GetCommandFactory() ServiceCommandFactory

	// GetInputService returns the input service
	GetInputService() StatelessInputService

	// GetPTYService returns the PTY service
	GetPTYService() StatelessPTYService

	// GetNotificationService returns the notification service
	GetNotificationService() StatelessNotificationService

	// GetAICoderService returns the AI coder service
	GetAICoderService() StatelessAICoderService

	// GetNavigationService returns the navigation service
	GetNavigationService() StatelessNavigationService

	// Cleanup performs cleanup of all services
	Cleanup() error
}

// ServiceContext provides context for service operations
type ServiceContext struct {
	MasterData    *state.MasterData
	OperationID   string
	RequestTime   int64
	Timeout       int64 // milliseconds
	MaxRetries    int
	InitialDelay  int64 // milliseconds
	MaxDelay      int64 // milliseconds
}

// NewServiceContext creates a new service context
func NewServiceContext(masterData *state.MasterData, operationID string, timeout int64) *ServiceContext {
	return &ServiceContext{
		MasterData:   masterData,
		OperationID:  operationID,
		RequestTime:  getCurrentTimeMillis(),
		Timeout:      timeout,
		MaxRetries:   3,
		InitialDelay: 100, // milliseconds
		MaxDelay:     5000, // milliseconds
	}
}

// IsExpired checks if the service context has expired
func (sc *ServiceContext) IsExpired() bool {
	return getCurrentTimeMillis()-sc.RequestTime > sc.Timeout
}

// getCurrentTimeMillis returns current time in milliseconds
func getCurrentTimeMillis() int64 {
	return 0 // TODO: Implement with actual time
}