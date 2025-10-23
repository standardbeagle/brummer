package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hinshun/vt10x"
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

	// NEW: PTY output debouncing
	debouncer       *PTYDebouncer
	activeSessionID atomic.Value // string - currently active session

	// NEW: Single shared vt10x terminal for active session
	activeTerminal *sync.Map // sessionID -> *vt10x.Terminal (reused instance)

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
		activeTerminal:        &sync.Map{},
		eventBus:              events.NewEventBus(),
		logStore:              logStore,
		maxConcurrentSessions: 0, // 0 = unlimited
		ptyCleanupTimeout:     5 * time.Second,
	}

	// Initialize debouncer with callback - will be connected later
	service.debouncer = nil // Will be set by SetDebounceCallback

	// Initialize active session ID as empty
	service.activeSessionID.Store("")

	return service
}

// SetDebounceCallback sets the callback for batched PTY output
// This should be called by the TUI to connect the debouncer
func (s *AICoderService) SetDebounceCallback(callback func(sessionID string, data []byte)) {
	// Create debouncer with 16ms interval for 60 FPS
	s.debouncer = NewPTYDebouncer(16*time.Millisecond, callback)
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

	// NEW: Start monitoring PTY output and send to debouncer
	// This replaces the old polling pattern with proactive debounced updates
	if s.debouncer != nil {
		go s.monitorPTYOutput(sessionID, ptySession)
	}

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

// === NEW BATCH PROCESSING METHODS ===

// ProcessBatchedOutput processes debounced PTY output through vt10x
// This is called by the debouncer callback after batching updates
func (s *AICoderService) ProcessBatchedOutput(sessionID string, data []byte) ([][]rune, error) {
	// Only process if this is the active session
	activeID, _ := s.activeSessionID.Load().(string)
	if activeID != sessionID {
		// Store in raw buffer but don't process
		return nil, nil
	}

	session, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.PTYSession == nil {
		return nil, ErrSessionNotStarted
	}

	// Process through vt10x terminal
	// Process the output data (already written in PTYSession.readLoop)
	// Just get the rendered screen - pass PTYSession for raw mode support
	screen := s.getTerminalScreen(session.PTYSession)

	// Update activity
	session.LastActivity = time.Now()
	session.BytesRead.Add(uint64(len(data)))

	return screen, nil
}

// SwitchActiveSession switches which session is actively being processed
func (s *AICoderService) SwitchActiveSession(newSessionID string) error {
	oldID, _ := s.activeSessionID.Load().(string)

	// Update active session ID
	s.activeSessionID.Store(newSessionID)

	// Update debouncer smart mode
	if s.debouncer != nil {
		// Mark old session as inactive
		if oldID != "" {
			s.debouncer.SetSessionActive(oldID, false)
		}

		// Mark new session as active
		if newSessionID != "" {
			s.debouncer.SetSessionActive(newSessionID, true)
		}
	}

	// Log the switch
	s.logSystemEvent(fmt.Sprintf("Switched active session: %s -> %s", oldID, newSessionID))

	return nil
}

// GetScreenBuffer returns the rendered terminal screen for the active session
func (s *AICoderService) GetScreenBuffer(sessionID string) ([][]rune, error) {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.PTYSession == nil {
		return nil, ErrSessionNotStarted
	}

	// Pass the PTYSession itself for raw mode support
	return s.getTerminalScreen(session.PTYSession), nil
}

// getTerminalScreen gets the terminal screen content with full color support
func (s *AICoderService) getTerminalScreen(terminal interface{}) [][]rune {
	// If this is a PTYSession, extract the vt10x.Terminal
	if ptySession, ok := terminal.(*aicoder.PTYSession); ok {
		terminal = ptySession.GetTerminal()
		if terminal == nil {
			return make([][]rune, 0)
		}
	}

	// Now we should have a vt10x.Terminal interface
	vt10xTerm, ok := terminal.(vt10x.Terminal)
	if !ok {
		s.logSystemEvent(fmt.Sprintf("DEBUG: Expected vt10x.Terminal, got %T", terminal))
		return make([][]rune, 0)
	}

	// Get terminal dimensions directly from vt10x.Terminal
	cols, rows := vt10xTerm.Size()
	if cols == 0 {
		cols = 80 // Default width
	}
	if rows == 0 {
		rows = 24 // Default height
	}

	// Get cursor position for reference
	cursor := vt10xTerm.Cursor()
	_ = cursor // Keep for future cursor indicator feature

	// Build screen buffer from FULL scrollback (all rows from vt10x)
	// The Bubble Tea viewport component will handle windowing and scrolling
	// This is how real terminal emulators work: full scrollback buffer + client-side viewport
	screen := make([][]rune, 0, rows)

	// Process each row in the FULL scrollback buffer
	for y := 0; y < rows; y++ {
		var outputLine []rune
		lastFG := -1
		lastBG := -1
		hasContent := false

		// Process each column position
		for x := 0; x < cols; x++ {
			// Get the cell from vt10x at this position (returns vt10x.Glyph)
			cell := vt10xTerm.Cell(x, y)

			// Extract cell properties directly from Glyph struct
			char := cell.Char
			fg := int(cell.FG)  // vt10x.Color is uint32
			bg := int(cell.BG)

			// Skip trailing spaces on empty lines
			if char == 0 || char == ' ' {
				if !hasContent {
					continue
				}
			} else {
				hasContent = true
			}

			// Add ANSI color codes if colors changed
			if fg != lastFG || bg != lastBG {
				if colorCode := s.buildANSIColorCode(fg, bg); len(colorCode) > 0 {
					outputLine = append(outputLine, colorCode...)
					lastFG = fg
					lastBG = bg
				}
			}

			// Add the character (use space if null)
			if char == 0 {
				outputLine = append(outputLine, ' ')
			} else {
				outputLine = append(outputLine, char)
			}
		}

		// Reset colors at end of line if any were set
		if lastFG != -1 || lastBG != -1 {
			outputLine = append(outputLine, []rune("\x1b[0m")...)
		}

		// Add the processed line to screen (even if empty, to preserve line structure)
		screen = append(screen, outputLine)
	}

	// Remove trailing empty lines (but keep internal empty lines)
	for len(screen) > 0 && len(screen[len(screen)-1]) == 0 {
		screen = screen[:len(screen)-1]
	}

	if len(screen) == 0 {
		return make([][]rune, 0)
	}

	return screen
}

// getRuneField extracts a rune field from a struct using reflection
func (s *AICoderService) getRuneField(obj interface{}, fieldName string) rune {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return 0
	}

	if r, ok := field.Interface().(rune); ok {
		return r
	}
	return 0
}

// getColorField extracts a Color field (uint32) from a struct using reflection
func (s *AICoderService) getColorField(obj interface{}, fieldName string) int {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return -1
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return -1
	}

	// vt10x.Color is a uint32
	switch field.Kind() {
	case reflect.Uint32, reflect.Uint, reflect.Uint64:
		return int(field.Uint())
	case reflect.Int32, reflect.Int, reflect.Int64:
		return int(field.Int())
	default:
		return -1
	}
}

// buildANSIColorCode creates ANSI escape sequence for foreground and background colors
func (s *AICoderService) buildANSIColorCode(fg, bg int) []rune {
	const DefaultFG = 1<<24 + 0 // vt10x default foreground color flag
	const DefaultBG = 1<<24 + 1 // vt10x default background color flag

	var codes []string

	// Handle foreground color
	if fg >= 0 && fg != int(DefaultFG) {
		if fg < 16 {
			// Standard ANSI colors (0-15)
			if fg < 8 {
				codes = append(codes, fmt.Sprintf("%d", 30+fg))
			} else {
				codes = append(codes, fmt.Sprintf("%d", 82+fg)) // Bright colors: 90-97
			}
		} else if fg < 256 {
			// 256-color mode
			codes = append(codes, fmt.Sprintf("38;5;%d", fg))
		} else if fg < (1 << 24) {
			// Possible 24-bit color (RGB in lower 24 bits)
			r := (fg >> 16) & 0xFF
			g := (fg >> 8) & 0xFF
			b := fg & 0xFF
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
		}
	}

	// Handle background color
	if bg >= 0 && bg != int(DefaultBG) {
		if bg < 16 {
			// Standard ANSI colors (0-15)
			if bg < 8 {
				codes = append(codes, fmt.Sprintf("%d", 40+bg))
			} else {
				codes = append(codes, fmt.Sprintf("%d", 92+bg)) // Bright colors: 100-107
			}
		} else if bg < 256 {
			// 256-color mode
			codes = append(codes, fmt.Sprintf("48;5;%d", bg))
		} else if bg < (1 << 24) {
			// Possible 24-bit color (RGB in lower 24 bits)
			r := (bg >> 16) & 0xFF
			g := (bg >> 8) & 0xFF
			b := bg & 0xFF
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", r, g, b))
		}
	}

	if len(codes) == 0 {
		return []rune{}
	}

	return []rune(fmt.Sprintf("\x1b[%sm", strings.Join(codes, ";")))
}

// === DEPRECATED METHODS ===

// CheckPTYOutput returns any pending PTY output for a session
// DEPRECATED: Use ProcessBatchedOutput with debouncer instead
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

// monitorPTYOutput reads from PTY output channel and sends to debouncer
func (s *AICoderService) monitorPTYOutput(sessionID string, ptySession *aicoder.PTYSession) {
	for {
		select {
		case output, ok := <-ptySession.OutputChan:
			if !ok {
				// Channel closed, session ended
				s.logSystemEvent(fmt.Sprintf("PTY output channel closed for session: %s", sessionID))
				return
			}

			// Send to debouncer for batching
			if s.debouncer != nil {
				s.debouncer.Add(sessionID, output)
			}

		case event, ok := <-ptySession.EventChan:
			if !ok {
				return
			}

			// Handle PTY events
			if event.Type == "error" {
				s.publishError(sessionID, fmt.Errorf("PTY error: %v", event.Data))
			}
		}
	}
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

// SendFormattedContent sends formatted content (error, proxy request, etc.) to an AI Coder session
// If sessionID is empty, sends to the most recently active session
func (s *AICoderService) SendFormattedContent(sessionID, contentType, title, content string) error {
	// Determine target session
	targetSessionID := sessionID
	if targetSessionID == "" {
		targetSessionID = s.GetActiveSession()
		if targetSessionID == "" {
			return fmt.Errorf("no active AI Coder session found")
		}
	}

	// Validate session exists and is active
	session, err := s.GetSession(targetSessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.Status != messages.SessionActive && session.Status != messages.SessionIdle {
		return fmt.Errorf("session is not active (status: %s)", session.Status)
	}

	// Format the content with a clear header
	var formattedContent strings.Builder

	formattedContent.WriteString("\n")
	formattedContent.WriteString("# ")
	formattedContent.WriteString(title)
	formattedContent.WriteString("\n")
	formattedContent.WriteString("# Type: ")
	formattedContent.WriteString(contentType)
	formattedContent.WriteString("\n\n")
	formattedContent.WriteString(content)
	formattedContent.WriteString("\n\n")

	// Send to AI Coder session
	inputBytes := []byte(formattedContent.String())
	if err := s.SendInput(targetSessionID, inputBytes); err != nil {
		return fmt.Errorf("failed to send content to AI Coder: %w", err)
	}

	// Log the activity
	s.logPTYActivity(targetSessionID,
		fmt.Sprintf("📋 Received %s: %s", contentType, title),
		false)

	return nil
}
