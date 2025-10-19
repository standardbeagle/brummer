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
	"github.com/standardbeagle/brummer/internal/logs"
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
	eventBus *events.EventBus
	shutdown atomic.Bool

	// Integration
	logStore *logs.Store // Central log store for AI activity logging

	// Configuration
	maxConcurrentSessions int
	ptyCleanupTimeout     time.Duration
}

// NewAICoderService creates a new AI coder service
func NewAICoderService(ctx context.Context, logStore *logs.Store) *AICoderService {
	service := &AICoderService{
		eventBus:              events.NewEventBus(),
		logStore:              logStore,
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
	s.logSystemEvent(fmt.Sprintf("Creating PTY session for %s (command: %s %s)", session.Name, command, strings.Join(args, " ")))
	s.logPTYActivity(sessionID, fmt.Sprintf("🤖 AI creating PTY session: %s", session.Name), false)

	ptySession, err := aicoder.NewPTYSession(sessionID, session.Name, command, args)
	if err != nil {
		session.ErrorCount.Add(1)
		s.logSystemEvent(fmt.Sprintf("Failed to create PTY session: %v", err))
		s.logPTYActivity(sessionID, fmt.Sprintf("❌ AI PTY creation failed: %v", err), true)
		s.publishError(sessionID, fmt.Errorf("failed to create PTY: %w", err))
		return fmt.Errorf("failed to create PTY: %w", err)
	}
	s.logSystemEvent(fmt.Sprintf("PTY session created successfully: %s", sessionID))
	s.logPTYActivity(sessionID, "✅ AI PTY session created successfully", false)

	// Resize to requested dimensions if different from default
	if width != 80 || height != 24 {
		if err := ptySession.Resize(width, height); err != nil {
			// Log warning but don't fail - PTY is already running
			s.publishError(sessionID, fmt.Errorf("failed to resize PTY: %w", err))
		}
	}

	// Store PTY session
	session.PTYSession = ptySession

	// PTY output monitoring is handled through Bubble Tea message pattern
	// No goroutines - the AI Coders view component will poll for PTY output

	// Update status
	session.Status = messages.SessionActive
	session.LastActivity = time.Now()

	s.logSystemEvent(fmt.Sprintf("Session started successfully: %s (status: %s)", sessionID, session.Status))
	s.logPTYActivity(sessionID, "🚀 AI session activated and ready for commands", false)

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

// Note: PTY output monitoring is handled by the AI Coders view component
// through Bubble Tea's message pattern. No goroutines are used.

// publishError publishes an error event for a session
func (s *AICoderService) publishError(sessionID string, err error) {
	errorMsg := fmt.Sprintf("AI Coder Error: %v", err)

	// Log to event bus for UI components
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

	// Also log to central logStore for unified logging
	if s.logStore != nil {
		s.logStore.Add(sessionID, "ai-coder", errorMsg, true)
	}
}

// logSystemEvent logs to the TUI system using the EventBus and logStore
// This ensures AI coder events appear in both the TUI logs and central logStore
func (s *AICoderService) logSystemEvent(message string) {
	// Log to event bus for UI components
	s.eventBus.Publish(events.Event{
		Type:      "aicoder.log",
		ProcessID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": message,
			"source":  "ai-coder-service",
		},
	})

	// Also log to central logStore for unified logging
	if s.logStore != nil {
		s.logStore.Add("ai-coder", "system", message, false)
	}
}

// logPTYActivity logs PTY session activity (proxy interactions, screenshots, etc.)
// This provides a complete audit trail of AI assistant activities
func (s *AICoderService) logPTYActivity(sessionID string, activity string, isError bool) {
	// Log to event bus for UI components
	s.eventBus.Publish(events.Event{
		Type:      "aicoder.activity",
		ProcessID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"sessionID": sessionID,
			"activity":  activity,
			"isError":   isError,
		},
	})

	// Also log to central logStore for unified logging
	if s.logStore != nil {
		s.logStore.Add(sessionID, "ai-coder", activity, isError)
	}
}

// LogProxyScreenshot logs when AI takes a screenshot via the proxy server
func (s *AICoderService) LogProxyScreenshot(sessionID, url string) {
	activity := fmt.Sprintf("📸 AI took screenshot via proxy: %s", url)
	s.logPTYActivity(sessionID, activity, false)
}

// LogCSSCapture logs when AI reads CSS/styles from development server
func (s *AICoderService) LogCSSCapture(sessionID, selector string) {
	activity := fmt.Sprintf("🎨 AI captured CSS: %s", selector)
	s.logPTYActivity(sessionID, activity, false)
}

// LogProxyError logs when proxy errors are auto-forwarded to AI
func (s *AICoderService) LogProxyError(sessionID, errorMsg string) {
	activity := fmt.Sprintf("⚠️ Proxy error forwarded to AI: %s", errorMsg)
	s.logPTYActivity(sessionID, activity, true)
}

// LogCodeGeneration logs when AI generates/creates code files
func (s *AICoderService) LogCodeGeneration(sessionID, filename string) {
	activity := fmt.Sprintf("✍️ AI generated code: %s", filename)
	s.logPTYActivity(sessionID, activity, false)
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

// CheckPTYOutput returns any pending PTY output for a session
// This is the Bubble Tea-compliant way to get PTY data without goroutines
func (s *AICoderService) CheckPTYOutput(sessionID string) ([]byte, error) {
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

	// Check for available output without blocking
	select {
	case output := <-session.PTYSession.OutputChan:
		// Update metrics
		session.BytesRead.Add(uint64(len(output)))
		session.LastActivity = time.Now()
		return output, nil
	case event := <-session.PTYSession.EventChan:
		// Handle PTY events (errors, state changes, etc.)
		if event.Type == "error" {
			session.ErrorCount.Add(1)
			session.Status = messages.SessionError
			s.publishError(sessionID, fmt.Errorf("PTY error: %v", event.Data))
		}
		return nil, fmt.Errorf("PTY event: %s", event.Type)
	default:
		// No output available
		return nil, nil
	}
}

// CheckSessionStatus checks if a session is still active
func (s *AICoderService) CheckSessionStatus(sessionID string) (messages.SessionStatus, error) {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return messages.SessionStopped, err
	}

	// Check if PTY session is still active
	if session.PTYSession != nil {
		select {
		case <-session.PTYSession.EventChan:
			// Channel closed, session ended
			session.Status = messages.SessionStopped
		default:
			// Channel still open, session active
		}
	}

	return session.Status, nil
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

	// Stop all sessions
	var sessionIDs []string
	s.sessions.Range(func(key, value interface{}) bool {
		sessionIDs = append(sessionIDs, key.(string))
		return true
	})

	for _, sessionID := range sessionIDs {
		if err := s.StopSession(ctx, sessionID); err != nil {
			// Log error but continue shutdown
			s.logSystemEvent(fmt.Sprintf("Error stopping session %s during shutdown: %v", sessionID, err))
		}
	}

	return nil
}

// EventBus returns the EventBus for subscribing to service events
func (s *AICoderService) EventBus() interface{} {
	return s.eventBus
}
