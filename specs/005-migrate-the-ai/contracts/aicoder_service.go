// Package contracts defines the service interfaces for AI coder functionality.
// This is a design contract - not executable code.
//
// Location (when implemented): internal/tui/services/aicoder_service.go
package contracts

import (
	"context"
	"time"
)

// AICoderService manages AI coding session lifecycle and PTY coordination.
//
// Thread Safety: All methods are safe for concurrent use.
// Error Handling: Returns errors for invalid operations (session not found, etc.)
// Lifecycle: Must call Shutdown() before process exit to clean up resources.
type AICoderService interface {
	// Session Management

	// CreateSession creates a new AI coding session.
	//
	// Parameters:
	//   ctx: Context for cancellation and timeout
	//   name: User-friendly session name (1-100 chars)
	//   command: Command to execute in PTY (non-empty)
	//   workingDir: Working directory for command (defaults to current if empty)
	//
	// Returns:
	//   sessionID: Unique identifier for the created session
	//   error: ErrInvalidName, ErrInvalidCommand, ErrMaxSessionsReached, or system error
	//
	// Post-conditions:
	//   - Session exists in registry with Status = SessionPending
	//   - PTYManager created but not started
	//   - SessionCreated event published to EventBus
	//
	// Example:
	//   sessionID, err := service.CreateSession(ctx, "Debug API", "npx @anthropic-ai/code", ".")
	CreateSession(ctx context.Context, name, command, workingDir string) (sessionID string, err error)

	// StartSession starts the PTY process for a session.
	//
	// Parameters:
	//   ctx: Context for cancellation and timeout
	//   sessionID: ID of session to start
	//   width: Initial terminal width (columns)
	//   height: Initial terminal height (rows)
	//
	// Returns:
	//   error: ErrSessionNotFound, ErrSessionAlreadyStarted, or PTY creation error
	//
	// Pre-conditions:
	//   - Session must exist
	//   - Session status must be Pending
	//   - width ≥ 20, height ≥ 5
	//
	// Post-conditions:
	//   - Session status = Active
	//   - PTY process running
	//   - Monitoring goroutine started
	//   - SessionUpdated event published
	//
	// Example:
	//   err := service.StartSession(ctx, sessionID, 80, 24)
	StartSession(ctx context.Context, sessionID string, width, height int) error

	// StopSession gracefully stops a session and cleans up resources.
	//
	// Parameters:
	//   ctx: Context for cancellation (timeout enforced internally)
	//   sessionID: ID of session to stop
	//
	// Returns:
	//   error: ErrSessionNotFound or cleanup error
	//
	// Post-conditions:
	//   - PTY process terminated
	//   - Monitoring goroutine stopped
	//   - Session status = Stopped
	//   - Session removed from registry
	//   - SessionEnded event published
	//   - Cleanup completed within ptyCleanupTimeout (default 5s)
	//
	// Idempotent: Calling StopSession on already-stopped session is safe (no-op).
	//
	// Example:
	//   err := service.StopSession(ctx, sessionID)
	StopSession(ctx context.Context, sessionID string) error

	// Session Queries

	// ListSessions returns summaries of all active sessions.
	//
	// Returns:
	//   []SessionSummary: Snapshot of current sessions (may be stale immediately)
	//   error: nil (never fails)
	//
	// Notes:
	//   - Results sorted by CreatedAt (oldest first)
	//   - Returns empty slice if no sessions exist
	//   - O(n) operation where n = session count
	//
	// Example:
	//   sessions := service.ListSessions()
	//   for _, s := range sessions {
	//       fmt.Printf("%s: %s\n", s.Name, s.Status)
	//   }
	ListSessions() []SessionSummary

	// GetSession returns full session details.
	//
	// Parameters:
	//   sessionID: ID of session to retrieve
	//
	// Returns:
	//   *AICoderSession: Session object (do not mutate!)
	//   error: ErrSessionNotFound if session doesn't exist
	//
	// Notes:
	//   - Returned session is a snapshot (may be stale)
	//   - Do NOT mutate returned session object
	//
	// Example:
	//   session, err := service.GetSession(sessionID)
	//   if err != nil { return err }
	//   fmt.Printf("Status: %s\n", session.Status)
	GetSession(sessionID string) (*AICoderSession, error)

	// GetActiveSession returns the most recently active session.
	//
	// Returns:
	//   sessionID: ID of most recently active session (empty if none)
	//   error: nil (never fails)
	//
	// Notes:
	//   - "Active" = highest LastActivity timestamp
	//   - Returns empty string if no sessions exist
	//   - Useful for "resume last session" functionality
	//
	// Example:
	//   sessionID := service.GetActiveSession()
	//   if sessionID != "" {
	//       // Resume session
	//   }
	GetActiveSession() string

	// PTY Interaction

	// GetTerminalState returns a read-only snapshot of terminal state.
	//
	// Parameters:
	//   sessionID: ID of session
	//
	// Returns:
	//   *TerminalState: Terminal screen snapshot (safe to read from any goroutine)
	//   error: ErrSessionNotFound or ErrSessionNotStarted
	//
	// Pre-conditions:
	//   - Session must exist
	//   - Session status must be Active or Idle
	//
	// Post-conditions:
	//   - Returns immutable snapshot (safe for concurrent reads)
	//   - Snapshot may be stale (PTY continues to update)
	//
	// Performance:
	//   - O(n) where n = terminal lines (typically <1000)
	//   - Involves copying terminal buffer (lock-free read)
	//   - Expect <1ms for typical terminal sizes
	//
	// Example:
	//   state, err := service.GetTerminalState(sessionID)
	//   if err != nil { return err }
	//   for _, line := range state.Lines {
	//       fmt.Println(line)
	//   }
	GetTerminalState(sessionID string) (*TerminalState, error)

	// SendInput sends keyboard input to the PTY.
	//
	// Parameters:
	//   sessionID: ID of session
	//   input: Raw bytes to send to PTY
	//
	// Returns:
	//   error: ErrSessionNotFound, ErrSessionNotStarted, or PTY write error
	//
	// Pre-conditions:
	//   - Session must exist
	//   - Session status must be Active or Idle
	//   - input length must be > 0
	//
	// Post-conditions:
	//   - Input written to PTY stdin
	//   - Session LastActivity updated
	//   - SessionUpdated event may be published (status changes)
	//
	// Performance:
	//   - Non-blocking write (uses lock-free PTY passthrough)
	//   - Expect <1ms latency
	//
	// Example:
	//   err := service.SendInput(sessionID, []byte("ls\n"))
	SendInput(sessionID string, input []byte) error

	// ResizePTY updates terminal dimensions.
	//
	// Parameters:
	//   sessionID: ID of session
	//   width: New width (columns)
	//   height: New height (rows)
	//
	// Returns:
	//   error: ErrSessionNotFound, ErrSessionNotStarted, or resize error
	//
	// Pre-conditions:
	//   - Session must exist
	//   - Session status must be Active or Idle
	//   - width ≥ 20, height ≥ 5
	//
	// Post-conditions:
	//   - PTY dimensions updated
	//   - SIGWINCH sent to PTY process
	//   - Terminal state reflects new dimensions
	//
	// Example:
	//   err := service.ResizePTY(sessionID, 120, 40)
	ResizePTY(sessionID string, width, height int) error

	// Lifecycle

	// Shutdown gracefully shuts down the service.
	//
	// Parameters:
	//   ctx: Context for shutdown timeout (recommended: 10s)
	//
	// Returns:
	//   error: Context timeout or cleanup error
	//
	// Post-conditions:
	//   - All sessions stopped
	//   - All PTY processes terminated
	//   - All monitoring goroutines stopped
	//   - No resource leaks (verified in tests)
	//
	// Idempotent: Safe to call multiple times.
	//
	// Example:
	//   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//   defer cancel()
	//   if err := service.Shutdown(ctx); err != nil {
	//       log.Printf("Shutdown error: %v", err)
	//   }
	Shutdown(ctx context.Context) error

	// EventSubscription returns the EventBus for subscribing to service events.
	//
	// Returns:
	//   *events.EventBus: EventBus instance (never nil)
	//
	// Event Topics:
	//   - "aicoder.session.created" → AICoderSessionCreatedMsg
	//   - "aicoder.session.updated" → AICoderSessionUpdatedMsg
	//   - "aicoder.session.ended" → AICoderSessionEndedMsg
	//   - "aicoder.pty.output" → AICoderPTYOutputMsg
	//   - "aicoder.error" → AICoderErrorMsg
	//
	// Example:
	//   eventBus := service.EventBus()
	//   sub := eventBus.Subscribe("aicoder.session.created")
	//   go func() {
	//       for msg := range sub {
	//           fmt.Printf("Session created: %v\n", msg)
	//       }
	//   }()
	EventBus() interface{}  // Returns *events.EventBus (interface{} to avoid import cycles)
}

// Error Definitions
//
// Implementation should define these as sentinel errors:
//
// var (
//     ErrInvalidName = errors.New("session name must be 1-100 characters")
//     ErrInvalidCommand = errors.New("command must be non-empty")
//     ErrMaxSessionsReached = errors.New("maximum concurrent sessions reached")
//     ErrSessionNotFound = errors.New("session not found")
//     ErrSessionAlreadyStarted = errors.New("session already started")
//     ErrSessionNotStarted = errors.New("session not started")
//     ErrInvalidDimensions = errors.New("terminal dimensions must be width≥20, height≥5")
//     ErrShutdownTimeout = errors.New("shutdown timeout exceeded")
// )

// Supporting Types (from data-model.md)

type SessionSummary struct {
	ID           string
	Name         string
	Status       SessionStatus
	LastActivity time.Time
}

type AICoderSession struct {
	ID           string
	Name         string
	Status       SessionStatus
	CreatedAt    time.Time
	LastActivity time.Time
	Command      string
	WorkingDir   string
	// PTYManager and metrics omitted (internal implementation detail)
}

type SessionStatus string

const (
	SessionPending SessionStatus = "pending"
	SessionActive  SessionStatus = "active"
	SessionIdle    SessionStatus = "idle"
	SessionError   SessionStatus = "error"
	SessionStopped SessionStatus = "stopped"
)

type TerminalState struct {
	Lines        []string
	Cursor       CursorPos
	Width        int
	Height       int
	TotalLines   int
	ViewportTop  int
	UpdatedAt    time.Time
}

type CursorPos struct {
	Row     int
	Col     int
	Visible bool
}

// Performance Requirements (from spec FR-025)
//
// Typing Responsiveness:
//   - SendInput() must complete in <50ms (p95)
//   - GetTerminalState() must complete in <5ms (p95)
//   - PTY output forwarding must be non-blocking
//
// Cleanup Requirements:
//   - StopSession() must complete in <100ms (p95)
//   - Shutdown() must complete in <10s (all sessions)
//   - No goroutine leaks (verified with make test-race)
//
// Concurrency Requirements:
//   - All methods safe for concurrent use
//   - sync.Map for lock-free session lookup
//   - Atomic counters for metrics
//   - No blocking operations in hot paths
