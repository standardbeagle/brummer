package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Sentinel errors
var (
	ErrInvalidName           = errors.New("session name must be 1-100 characters")
	ErrInvalidCommand        = errors.New("command must be non-empty")
	ErrMaxSessionsReached    = errors.New("maximum concurrent sessions reached")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionAlreadyStarted = errors.New("session already started")
	ErrSessionNotStarted     = errors.New("session not started")
	ErrInvalidDimensions     = errors.New("terminal dimensions must be width≥20, height≥5")
	ErrShutdownTimeout       = errors.New("shutdown timeout exceeded")
)

// AICoderSession represents an active AI coding session
type AICoderSession struct {
	ID           string
	Name         string
	Status       messages.SessionStatus
	CreatedAt    time.Time
	LastActivity time.Time
	Command      string
	WorkingDir   string

	// PTY integration
	PTYSession *aicoder.PTYSession

	// Metrics
	BytesRead    atomic.Uint64
	BytesWritten atomic.Uint64
	ErrorCount   atomic.Uint32
}

// AICoderService manages AI coding session lifecycle and PTY coordination
type AICoderService struct {
	// Session Registry (sync.Map for lock-free reads)
	sessions sync.Map // map[string]*AICoderSession

	// Lifecycle
	eventBus   *events.EventBus
	wg         sync.WaitGroup
	shutdownCh chan struct{}
	shutdown   atomic.Bool

	// Configuration
	maxConcurrentSessions int
	ptyCleanupTimeout     time.Duration
}

// NewAICoderService creates a new AI coder service
func NewAICoderService(ctx context.Context) *AICoderService {
	service := &AICoderService{
		eventBus:              events.NewEventBus(),
		shutdownCh:            make(chan struct{}),
		maxConcurrentSessions: 0, // 0 = unlimited
		ptyCleanupTimeout:     5 * time.Second,
	}
	return service
}

// CreateSession creates a new AI coding session
func (s *AICoderService) CreateSession(ctx context.Context, name, command, workingDir string) (string, error) {
	// Validate parameters
	if len(name) == 0 || len(name) > 100 {
		return "", ErrInvalidName
	}
	if len(command) == 0 {
		return "", ErrInvalidCommand
	}

	// Check max sessions limit (if configured)
	if s.maxConcurrentSessions > 0 {
		count := 0
		s.sessions.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		if count >= s.maxConcurrentSessions {
			return "", ErrMaxSessionsReached
		}
	}

	// Create session
	sessionID := uuid.New().String()
	now := time.Now()

	session := &AICoderSession{
		ID:           sessionID,
		Name:         name,
		Status:       messages.SessionPending,
		CreatedAt:    now,
		LastActivity: now,
		Command:      command,
		WorkingDir:   workingDir,
	}

	// Store in registry
	s.sessions.Store(sessionID, session)

	// Publish event
	s.eventBus.Publish(events.Event{
		Type:      "aicoder.session.created",
		ProcessID: sessionID,
		Timestamp: now,
		Data: map[string]interface{}{
			"sessionID": sessionID,
			"name":      name,
		},
	})

	return sessionID, nil
}

// StartSession starts the PTY process for a session
func (s *AICoderService) StartSession(ctx context.Context, sessionID string, width, height int) error {
	// Validate dimensions
	if width < 20 || height < 5 {
		return ErrInvalidDimensions
	}

	// Get session
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}
	session := value.(*AICoderSession)

	// Check status
	if session.Status != messages.SessionPending {
		return ErrSessionAlreadyStarted
	}

	// Parse command string into command + args
	parts := strings.Fields(session.Command)
	if len(parts) == 0 {
		return ErrInvalidCommand
	}

	command := parts[0]
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Create PTY session (this also starts the PTY process and I/O loops)
	// Note: NewPTYSession starts with default 80x24 dimensions
	// TODO: Support custom dimensions in NewPTYSession or resize after creation
	ptySession, err := aicoder.NewPTYSession(sessionID, session.Name, command, args)
	if err != nil {
		session.ErrorCount.Add(1)
		s.publishError(sessionID, fmt.Errorf("failed to create PTY: %w", err))
		return fmt.Errorf("failed to create PTY: %w", err)
	}

	// Resize to requested dimensions if different from default
	if width != 80 || height != 24 {
		if err := ptySession.Resize(width, height); err != nil {
			// Log warning but don't fail - PTY is already running
			s.publishError(sessionID, fmt.Errorf("failed to resize PTY: %w", err))
		}
	}

	// Store PTY session
	session.PTYSession = ptySession

	// Start monitoring PTY output through command pattern
	// TODO: Replace with proper command pattern for PTY monitoring
	// For now, disable monitoring to avoid goroutines
	// s.wg.Add(1)
	// s.monitorPTYOutput(sessionID)

	// Update status
	session.Status = messages.SessionActive
	session.LastActivity = time.Now()

	// Publish update
	s.eventBus.Publish(events.Event{
		Type:      "aicoder.session.updated",
		ProcessID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"sessionID":    sessionID,
			"status":       string(session.Status),
			"lastActivity": session.LastActivity,
		},
	})

	return nil
}

// monitorPTYOutput monitors PTY output and publishes events
func (s *AICoderService) monitorPTYOutput(sessionID string) {
	defer s.wg.Done()

	value, ok := s.sessions.Load(sessionID)
	if !ok {
		return
	}
	session := value.(*AICoderSession)

	if session.PTYSession == nil {
		return
	}

	sequence := atomic.Uint64{}

	for {
		select {
		case <-s.shutdownCh:
			return

		case output, ok := <-session.PTYSession.OutputChan:
			if !ok {
				// PTY session ended - check if session still exists before updating
				if value, exists := s.sessions.Load(sessionID); exists {
					sess := value.(*AICoderSession)
					sess.Status = messages.SessionStopped
				}
				s.eventBus.Publish(events.Event{
					Type:      "aicoder.session.ended",
					ProcessID: sessionID,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"sessionID": sessionID,
						"reason":    "pty_closed",
					},
				})
				return
			}

			// Update metrics - check if session still exists before updating
			if value, exists := s.sessions.Load(sessionID); exists {
				sess := value.(*AICoderSession)
				sess.BytesRead.Add(uint64(len(output)))
				sess.LastActivity = time.Now()
			}

			// Publish PTY output event
			s.eventBus.Publish(events.Event{
				Type:      "aicoder.pty.output",
				ProcessID: sessionID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"sessionID": sessionID,
					"data":      output,
					"sequence":  sequence.Add(1),
				},
			})

		case event, ok := <-session.PTYSession.EventChan:
			if !ok {
				return
			}

			// Handle PTY events (errors, state changes, etc.)
			if event.Type == "error" {
				// Check if session still exists before updating
				if value, exists := s.sessions.Load(sessionID); exists {
					sess := value.(*AICoderSession)
					sess.ErrorCount.Add(1)
					sess.Status = messages.SessionError
				}
				s.publishError(sessionID, fmt.Errorf("PTY error: %v", event.Data))
			}
		}
	}
}

// publishError publishes an error event for a session
func (s *AICoderService) publishError(sessionID string, err error) {
	s.eventBus.Publish(events.Event{
		Type:      "aicoder.error",
		ProcessID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"sessionID": sessionID,
			"error":     err.Error(),
			"severity":  "error",
		},
	})
}

// StopSession gracefully stops a session and cleans up resources
func (s *AICoderService) StopSession(ctx context.Context, sessionID string) error {
	// Get session
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		// Idempotent - if already stopped, return success
		return nil
	}
	session := value.(*AICoderSession)

	// Remove from registry FIRST (before stopping PTY) to prevent monitoring goroutine races
	s.sessions.Delete(sessionID)

	// Update status (safe now that it's removed from registry)
	session.Status = messages.SessionStopped

	// Stop PTY session if running (monitoring goroutine will stop when channels close)
	if session.PTYSession != nil {
		if err := session.PTYSession.Close(); err != nil {
			// Log error but continue cleanup
			s.publishError(sessionID, fmt.Errorf("PTY cleanup error: %w", err))
		}
	}

	// Publish event
	s.eventBus.Publish(events.Event{
		Type:      "aicoder.session.ended",
		ProcessID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"sessionID": sessionID,
			"reason":    "stopped",
		},
	})

	return nil
}

// ListSessions returns summaries of all active sessions
func (s *AICoderService) ListSessions() []messages.SessionSummary {
	var sessions []messages.SessionSummary

	s.sessions.Range(func(key, value interface{}) bool {
		session := value.(*AICoderSession)
		sessions = append(sessions, messages.SessionSummary{
			ID:           session.ID,
			Name:         session.Name,
			Status:       session.Status,
			LastActivity: session.LastActivity,
		})
		return true
	})

	// Sort by CreatedAt (oldest first) - for now just return as-is
	return sessions
}

// GetSession returns full session details
func (s *AICoderService) GetSession(sessionID string) (*AICoderSession, error) {
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}
	return value.(*AICoderSession), nil
}

// GetActiveSession returns the most recently active session
func (s *AICoderService) GetActiveSession() string {
	var mostRecent *AICoderSession

	s.sessions.Range(func(key, value interface{}) bool {
		session := value.(*AICoderSession)
		if mostRecent == nil || session.LastActivity.After(mostRecent.LastActivity) {
			mostRecent = session
		}
		return true
	})

	if mostRecent == nil {
		return ""
	}
	return mostRecent.ID
}

// GetTerminalState returns a read-only snapshot of terminal state
func (s *AICoderService) GetTerminalState(sessionID string) (*messages.TerminalState, error) {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != messages.SessionActive && session.Status != messages.SessionIdle {
		return nil, ErrSessionNotStarted
	}

	if session.PTYSession == nil {
		return nil, ErrSessionNotStarted
	}

	// Get terminal state from PTY session
	// Note: PTYSession.Terminal access should be thread-safe via vt10x
	termContent := session.PTYSession.Terminal.String()
	lines := strings.Split(termContent, "\n")

	cursor := session.PTYSession.Terminal.Cursor()

	// Get terminal dimensions - default to 80x24
	width, height := 80, 24

	return &messages.TerminalState{
		Lines:       lines,
		Cursor:      messages.CursorPos{Row: cursor.Y, Col: cursor.X, Visible: true},
		Width:       width,
		Height:      height,
		TotalLines:  len(lines),
		ViewportTop: 0,
		UpdatedAt:   time.Now(),
	}, nil
}

// SendInput sends keyboard input to the PTY
func (s *AICoderService) SendInput(sessionID string, input []byte) error {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return err
	}

	if session.Status != messages.SessionActive && session.Status != messages.SessionIdle {
		return ErrSessionNotStarted
	}

	if len(input) == 0 {
		return nil
	}

	if session.PTYSession == nil {
		return ErrSessionNotStarted
	}

	// Forward input to PTY session's input channel
	select {
	case session.PTYSession.InputChan <- input:
		session.LastActivity = time.Now()
		session.BytesWritten.Add(uint64(len(input)))
		return nil
	case <-time.After(50 * time.Millisecond):
		// Timeout to meet <50ms latency requirement
		return fmt.Errorf("input timeout: PTY not responsive")
	}
}

// ResizePTY updates terminal dimensions
func (s *AICoderService) ResizePTY(sessionID string, width, height int) error {
	if width < 20 || height < 5 {
		return ErrInvalidDimensions
	}

	session, err := s.GetSession(sessionID)
	if err != nil {
		return err
	}

	if session.Status != messages.SessionActive && session.Status != messages.SessionIdle {
		return ErrSessionNotStarted
	}

	if session.PTYSession == nil {
		return ErrSessionNotStarted
	}

	// Resize the PTY
	if err := session.PTYSession.Resize(width, height); err != nil {
		return fmt.Errorf("failed to resize PTY: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the service
func (s *AICoderService) Shutdown(ctx context.Context) error {
	// Check if already shutdown
	if s.shutdown.Swap(true) {
		return nil // Already shutdown, idempotent
	}

	// Close shutdown channel to signal goroutines
	close(s.shutdownCh)

	// Stop all sessions
	var sessionIDs []string
	s.sessions.Range(func(key, value interface{}) bool {
		sessionIDs = append(sessionIDs, key.(string))
		return true
	})

	for _, sessionID := range sessionIDs {
		if err := s.StopSession(ctx, sessionID); err != nil {
			// Log error but continue shutdown
			fmt.Printf("Error stopping session %s during shutdown: %v\n", sessionID, err)
		}
	}

	// Wait for all operations through command pattern
	// TODO: Replace with proper command pattern for shutdown coordination
	// For now, perform immediate wait to avoid goroutines
	s.wg.Wait()

	return nil
}

// EventBus returns the EventBus for subscribing to service events
func (s *AICoderService) EventBus() interface{} {
	return s.eventBus
}
