package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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

	// PTY integration (placeholder - will be implemented when we integrate with actual PTY)
	// PTYManager   *LockFreePTYManager

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

	// TODO: Create and start PTY manager here
	// For now, just update status
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

// StopSession gracefully stops a session and cleans up resources
func (s *AICoderService) StopSession(ctx context.Context, sessionID string) error {
	// Get session
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		// Idempotent - if already stopped, return success
		return nil
	}
	session := value.(*AICoderSession)

	// TODO: Stop PTY manager and cleanup goroutines
	// For now, just update status
	session.Status = messages.SessionStopped

	// Remove from registry
	s.sessions.Delete(sessionID)

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

	// TODO: Get actual terminal state from PTY manager
	// For now, return a placeholder
	return &messages.TerminalState{
		Lines:       []string{"AI Coder Terminal", "Session: " + session.Name, "Status: " + string(session.Status)},
		Cursor:      messages.CursorPos{Row: 0, Col: 0, Visible: true},
		Width:       80,
		Height:      24,
		TotalLines:  3,
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

	// TODO: Forward input to PTY manager
	// For now, just update activity time
	session.LastActivity = time.Now()
	session.BytesWritten.Add(uint64(len(input)))

	return nil
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

	// TODO: Resize PTY manager and send SIGWINCH
	// For now, just validate

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

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ErrShutdownTimeout
	}
}

// EventBus returns the EventBus for subscribing to service events
func (s *AICoderService) EventBus() interface{} {
	return s.eventBus
}
