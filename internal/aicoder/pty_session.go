package aicoder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

var (
	// Debug logging for PTY output - set to true to enable
	debugPTYOutput = os.Getenv("DEBUG_PTY_OUTPUT") == "true"
	debugFile      *os.File
	debugMutex     sync.Mutex
)

// initDebugLogging initializes the debug log file if enabled
func initDebugLogging() {
	if debugPTYOutput && debugFile == nil {
		debugMutex.Lock()
		defer debugMutex.Unlock()

		if debugFile == nil {
			filename := fmt.Sprintf("pty-debug-%s.log", time.Now().Format("20060102-150405"))
			file, err := os.Create(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create debug log: %v\n", err)
				return
			}
			debugFile = file
			fmt.Fprintf(os.Stderr, "PTY debug logging enabled: %s\n", filename)

			// Write header
			debugFile.WriteString("=== PTY Debug Log ===\n")
			debugFile.WriteString(fmt.Sprintf("Started: %s\n", time.Now().Format(time.RFC3339)))
			debugFile.WriteString("Format: [timestamp] [hex] raw_bytes\n")
			debugFile.WriteString("==================\n\n")
		}
	}
}

// logPTYData logs raw PTY data to debug file
func logPTYData(data []byte, source string) {
	if debugPTYOutput && debugFile != nil {
		debugMutex.Lock()
		defer debugMutex.Unlock()

		timestamp := time.Now().Format("15:04:05.000")

		// Write timestamp and source
		debugFile.WriteString(fmt.Sprintf("[%s] [%s] ", timestamp, source))

		// Write hex representation
		for _, b := range data {
			debugFile.WriteString(fmt.Sprintf("%02X ", b))
		}
		debugFile.WriteString("\n")

		// Write ASCII representation (with control codes shown)
		debugFile.WriteString(fmt.Sprintf("[%s] [%s] ", timestamp, source))
		for _, b := range data {
			if b >= 32 && b < 127 {
				debugFile.WriteString(string(b))
			} else if b == '\n' {
				debugFile.WriteString("\\n")
			} else if b == '\r' {
				debugFile.WriteString("\\r")
			} else if b == '\t' {
				debugFile.WriteString("\\t")
			} else if b == 27 {
				debugFile.WriteString("\\e")
			} else {
				debugFile.WriteString(fmt.Sprintf("\\x%02X", b))
			}
		}
		debugFile.WriteString("\n\n")

		// Flush to ensure data is written
		debugFile.Sync()
	}
}

// PTYSession represents a pseudo-terminal session for an AI coder
type PTYSession struct {
	ID           string
	Name         string
	Command      *exec.Cmd
	PTY          *os.File
	Terminal     vt10x.Terminal // vt10x terminal emulator (DEPRECATED - use RawBuffer)
	RawBuffer    *RawPTYBuffer  // Raw PTY buffer for direct ANSI passthrough
	UseRawMode   bool           // If true, use RawBuffer instead of Terminal
	IsActive     bool
	IsFullScreen bool
	DebugMode    bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc

	// Terminal configuration state for cleanup
	terminalState *terminalState

	// Event channels - SIMPLIFIED: Reduced from 1000 to 100 for 1-3 sessions
	OutputChan chan []byte
	InputChan  chan []byte
	EventChan  chan PTYEvent

	// Brummer integration
	dataInjector *DataInjector

	// Stream JSON parsing for non-interactive mode
	isStreamJSON bool
	streamBuffer strings.Builder

	// Raw output history for scrollback with ANSI codes
	outputHistory []byte
	maxHistory    int
	historyMutex  sync.Mutex

	// Channel closing synchronization
	closeOnce sync.Once
	closed    bool

	// REMOVED: channelMemoryStats - not needed for 1-3 sessions
}

// IsAtStartOfLine returns true if the cursor is at the beginning of a line
// This helps determine if a "/" should trigger Brummer commands vs AI input
func (s *PTYSession) IsAtStartOfLine() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Terminal == nil {
		return true // Default to allowing Brummer commands
	}

	// Get cursor position
	cursor := s.Terminal.Cursor()
	return cursor.X == 0
}

// GetCurrentLineContent returns the content of the current line up to cursor
// This helps determine context for slash command routing
func (s *PTYSession) GetCurrentLineContent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Terminal == nil {
		return ""
	}

	cursor := s.Terminal.Cursor()
	content := s.Terminal.String()
	lines := strings.Split(content, "\n")

	if cursor.Y >= len(lines) {
		return ""
	}

	currentLine := lines[cursor.Y]
	if cursor.X >= len(currentLine) {
		return currentLine
	}

	return currentLine[:cursor.X]
}

// PTYEvent represents events that can occur in a PTY session
type PTYEvent struct {
	Type      PTYEventType
	SessionID string
	Data      interface{}
	Timestamp time.Time
}

type PTYEventType string

const (
	PTYEventOutput     PTYEventType = "output"
	PTYEventInput      PTYEventType = "input"
	PTYEventResize     PTYEventType = "resize"
	PTYEventClose      PTYEventType = "close"
	PTYEventDataInject PTYEventType = "data_inject"
)

// NewPTYSession creates a new PTY session for an AI coder
func NewPTYSession(id, name, command string, args []string) (*PTYSession, error) {
	return NewPTYSessionWithEnv(id, name, command, args, nil)
}

// NewPTYSessionWithEnv creates a new PTY session with additional environment variables
func NewPTYSessionWithEnv(id, name, command string, args []string, extraEnv map[string]string) (*PTYSession, error) {
	// Initialize debug logging if enabled
	initDebugLogging()

	// Detect if this is a stream-json session
	isStreamJSON := false
	for i, arg := range args {
		if arg == "--output-format" && i+1 < len(args) && args[i+1] == "stream-json" {
			isStreamJSON = true
			break
		}
	}
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()

	// Set up environment for proper terminal behavior
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, "COLORTERM=truecolor")
	cmd.Env = append(cmd.Env, "COLUMNS=80") // Set initial columns
	cmd.Env = append(cmd.Env, "LINES=24")   // Set initial lines

	// Add any extra environment variables (like BRUMMER_MCP_URL)
	for key, value := range extraEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Fix 1-second input delay: Configure PTY for immediate output (no line buffering)
	// This solves the issue where the last typed character has ~1 second delay
	terminalState, err := configurePTYForResponsiveness(ptmx)
	if err != nil {
		// PTY will still work with default buffering
		terminalState = nil // Ensure we don't try to restore invalid state
	}

	// Create terminal emulator - use vt10x for proper terminal state handling
	// Claude Code (Ink) does extensive screen rewriting that requires proper emulation
	useRawMode := false // Raw mode doesn't work with Ink's aggressive rewriting
	var terminal vt10x.Terminal
	var rawBuffer *RawPTYBuffer

	if useRawMode {
		// Raw PTY buffer - doesn't work well with Ink's screen rewriting
		rawBuffer = NewRawPTYBuffer(100)
		terminal = nil
	} else {
		// Use vt10x with standard 24-line viewport (no massive scrollback)
		terminal = vt10x.New(vt10x.WithSize(80, 24))
		rawBuffer = nil
	}

	session := &PTYSession{
		ID:            id,
		Name:          name,
		Command:       cmd,
		PTY:           ptmx,
		Terminal:      terminal,
		RawBuffer:     rawBuffer,
		UseRawMode:    useRawMode,
		IsActive:      true,
		IsFullScreen:  false,
		DebugMode:     false,
		ctx:           ctx,
		cancel:        cancel,
		OutputChan:    make(chan []byte, 100),  // SIMPLIFIED: Reduced from 1000 to 100
		InputChan:     make(chan []byte, 100),  // SIMPLIFIED: Reduced from 1000 to 100
		EventChan:     make(chan PTYEvent, 100), // SIMPLIFIED: Reduced from 1000 to 100
		dataInjector:  NewDataInjector(),
		isStreamJSON:  isStreamJSON,
		outputHistory: make([]byte, 0, 1024*1024), // Start with 1MB capacity
		maxHistory:    10 * 1024 * 1024,           // 10MB max history
		terminalState: terminalState,              // Store for cleanup
	}

	
	// Start I/O goroutines
	go session.readLoop()
	go session.writeLoop()

	// REMOVED: channelMonitorLoop - not needed for 1-3 sessions

	return session, nil
}

// readLoop continuously reads from the PTY and processes output
func (s *PTYSession) readLoop() {
	defer close(s.OutputChan)

	reader := bufio.NewReader(s.PTY)
	buffer := make([]byte, 1024) // Smaller buffer for more responsive updates

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					select {
					case s.EventChan <- PTYEvent{
						Type:      PTYEventClose,
						SessionID: s.ID,
						Data:      fmt.Sprintf("Read error: %v", err),
						Timestamp: time.Now(),
					}:
					case <-s.ctx.Done():
					}
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				// Debug log the raw PTY output
				logPTYData(data, "PTY_OUT")

				// Store raw output in history (with ANSI codes)
				s.historyMutex.Lock()
				s.outputHistory = append(s.outputHistory, data...)
				// Trim history if it exceeds max size
				if len(s.outputHistory) > s.maxHistory {
					// Keep the last maxHistory bytes
					s.outputHistory = s.outputHistory[len(s.outputHistory)-s.maxHistory:]
				}
				s.historyMutex.Unlock()

				// Feed data to terminal emulator or raw buffer
				if s.isStreamJSON {
					s.processStreamJSON(data)
				} else {
					s.mu.Lock()
					if s.UseRawMode && s.RawBuffer != nil {
						// Write to raw buffer for direct ANSI passthrough
						s.RawBuffer.Write(data)
					} else if s.Terminal != nil {
						// Write to vt10x terminal emulator in smaller chunks to prevent partial writes
						s.writeToTerminalInChunks(data)
					}
					s.mu.Unlock()
				}

				// Send to output channel
				select {
				case s.OutputChan <- data:
				case <-s.ctx.Done():
					return
				}

				// Emit output event
				select {
				case s.EventChan <- PTYEvent{
					Type:      PTYEventOutput,
					SessionID: s.ID,
					Data:      data,
					Timestamp: time.Now(),
				}:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

// writeLoop continuously writes input to the PTY
func (s *PTYSession) writeLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case data := <-s.InputChan:
			// Debug log the input being sent to PTY
			logPTYData(data, "PTY_IN")
			if _, err := s.PTY.Write(data); err != nil {
				// Check if session is closed before sending to channel
				s.mu.RLock()
				isClosed := s.closed
				s.mu.RUnlock()

				if !isClosed {
					select {
					case s.EventChan <- PTYEvent{
						Type:      PTYEventClose,
						SessionID: s.ID,
						Data:      fmt.Sprintf("Write error: %v", err),
						Timestamp: time.Now(),
					}:
					case <-s.ctx.Done():
					}
				}
				return
			}

			// Emit input event only if not closed
			s.mu.RLock()
			isClosed := s.closed
			s.mu.RUnlock()

			if !isClosed {
				select {
				case s.EventChan <- PTYEvent{
					Type:      PTYEventInput,
					SessionID: s.ID,
					Data:      data,
					Timestamp: time.Now(),
				}:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

// WriteInput sends input to the PTY session
func (s *PTYSession) WriteInput(data []byte) error {
	s.mu.RLock()
	if !s.IsActive {
		s.mu.RUnlock()
		return fmt.Errorf("session closed")
	}
	s.mu.RUnlock()

	select {
	case s.InputChan <- data:
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("session closed")
	default:
		return fmt.Errorf("input buffer full")
	}
}

// WriteInputNonBlocking sends input to PTY session without blocking for typing responsiveness.
//
// CRITICAL: This method writes directly to PTY bypassing InputChan to prevent typing lag.
// DO NOT mix with WriteInput() for the same data as it may cause duplication.
//
// Use cases:
//   - Interactive typing where responsiveness is critical
//   - Real-time character input where buffering causes lag
//
// Avoid when:
//   - Batch operations where ordering with other writes matters
//   - Programmatic input where WriteInput() ordering is needed
//
// This method ensures immediate PTY write while still emitting tracking events.
func (s *PTYSession) WriteInputNonBlocking(data []byte) error {
	s.mu.RLock()
	if !s.IsActive {
		s.mu.RUnlock()
		return fmt.Errorf("session closed")
	}
	s.mu.RUnlock()

	// Direct write to PTY for immediate response - bypasses InputChan entirely
	logPTYData(data, "PTY_IN_DIRECT")
	if _, err := s.PTY.Write(data); err != nil {
		return fmt.Errorf("direct write failed: %w", err)
	}

	// Emit input event for tracking without sending to InputChan (prevents duplication)
	s.mu.RLock()
	isClosed := s.closed
	s.mu.RUnlock()

	if !isClosed {
		select {
		case s.EventChan <- PTYEvent{
			Type:      PTYEventInput,
			SessionID: s.ID,
			Data:      data,
			Timestamp: time.Now(),
		}:
		case <-s.ctx.Done():
			// Session closed, ignore
		default:
			// Drop event if channel full - typing responsiveness is priority
		}
	}

	return nil
}

// WriteString sends a string to the PTY session
func (s *PTYSession) WriteString(text string) error {
	return s.WriteInput([]byte(text))
}

// InjectData injects Brummer data into the terminal session
func (s *PTYSession) InjectData(dataType DataInjectionType, data interface{}) error {
	injectedText, err := s.dataInjector.FormatData(dataType, data)
	if err != nil {
		return err
	}

	// Add a visual indicator for injected data
	formattedText := fmt.Sprintf("\n\n🔹 [BRUMMER] %s\n%s\n",
		s.dataInjector.GetDataTypeLabel(dataType),
		injectedText)

	if err := s.WriteString(formattedText); err != nil {
		return err
	}

	// Emit data injection event
	select {
	case s.EventChan <- PTYEvent{
		Type:      PTYEventDataInject,
		SessionID: s.ID,
		Data: map[string]interface{}{
			"type": dataType,
			"data": data,
			"text": formattedText,
		},
		Timestamp: time.Now(),
	}:
	case <-s.ctx.Done():
		return fmt.Errorf("session closed")
	}

	return nil
}

// Resize resizes the PTY
func (s *PTYSession) Resize(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := pty.Setsize(s.PTY, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	}); err != nil {
		return err
	}

	// Resize the vt10x terminal WIDTH to match display, but keep 1000 rows for scrollback
	if s.Terminal != nil {
		s.Terminal.Resize(width, 1000) // Match width, preserve scrollback height
	}

	select {
	case s.EventChan <- PTYEvent{
		Type:      PTYEventResize,
		SessionID: s.ID,
		Data: map[string]int{
			"width":  width,
			"height": height,
		},
		Timestamp: time.Now(),
	}:
	case <-s.ctx.Done():
		// Session closed, ignore
	}

	return nil
}

// SetFullScreen toggles full-screen mode
func (s *PTYSession) SetFullScreen(fullScreen bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsFullScreen = fullScreen
}

// IsFullScreenMode returns whether the session is in full-screen mode
func (s *PTYSession) IsFullScreenMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsFullScreen
}

// SetDebugMode enables/disables automatic event injection
func (s *PTYSession) SetDebugMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DebugMode = enabled
}

// IsDebugModeEnabled returns whether debug mode is enabled
func (s *PTYSession) IsDebugModeEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DebugMode
}

// GetTerminal returns the vt10x terminal emulator
func (s *PTYSession) GetTerminal() vt10x.Terminal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Terminal
}

// GetOutputHistory returns the raw output history with ANSI codes
func (s *PTYSession) GetOutputHistory() []byte {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	// Return a copy to prevent external modification
	history := make([]byte, len(s.outputHistory))
	copy(history, s.outputHistory)
	return history
}

// GetScreenContent returns the current terminal screen content
// In raw mode, this returns ANSI codes directly for Bubble Tea to render
func (s *PTYSession) GetScreenContent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.UseRawMode && s.RawBuffer != nil {
		// Return raw ANSI content for direct passthrough
		return s.RawBuffer.GetScreenContent()
	} else if s.Terminal != nil {
		// Fallback to vt10x String() method
		return s.Terminal.String()
	}

	return ""
}

// GetRawScreenLines returns the screen as lines of raw bytes
// This is useful for processing line by line with ANSI codes intact
func (s *PTYSession) GetRawScreenLines() [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.UseRawMode && s.RawBuffer != nil {
		return s.RawBuffer.GetLines()
	}

	// Fallback: convert terminal string to lines
	if s.Terminal != nil {
		content := s.Terminal.String()
		lines := strings.Split(content, "\n")
		result := make([][]byte, len(lines))
		for i, line := range lines {
			result[i] = []byte(line)
		}
		return result
	}

	return nil
}

// Close closes the PTY session
func (s *PTYSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.IsActive {
		return nil
	}

	s.IsActive = false
	s.closed = true

	// Use sync.Once to ensure we only close once
	s.closeOnce.Do(func() {
		// Restore terminal state before closing
		if s.terminalState != nil {
			restoreTerminalState(s.terminalState)
		}

		// Cancel context to signal goroutines to stop
		s.cancel()

		// Give goroutines a brief moment to exit cleanly
		// This prevents "send on closed channel" panics
		go func() {
			time.Sleep(100 * time.Millisecond)

			// Close channels after goroutines have had time to exit
			close(s.InputChan)
			close(s.EventChan)
		}()
	})

	// Terminate the process
	if s.Command != nil && s.Command.Process != nil {
		s.Command.Process.Kill()
	}

	// Close PTY
	if s.PTY != nil {
		s.PTY.Close()
	}

	return nil
}

// processStreamJSON processes streaming JSON output from Claude CLI
func (s *PTYSession) processStreamJSON(data []byte) {
	// Add data to stream buffer
	s.streamBuffer.Write(data)

	// Process complete lines (JSON objects are typically on separate lines)
	content := s.streamBuffer.String()
	lines := strings.Split(content, "\n")

	// Keep the last (potentially incomplete) line in the buffer
	if len(lines) > 0 {
		s.streamBuffer.Reset()
		s.streamBuffer.WriteString(lines[len(lines)-1])

		// Process complete lines
		for _, line := range lines[:len(lines)-1] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			s.parseStreamJSONLine(line)
		}
	}
}

// parseStreamJSONLine parses a single JSON line from Claude's stream output
func (s *PTYSession) parseStreamJSONLine(line string) {
	var streamEvent map[string]interface{}
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		// If not JSON, treat as regular output
		s.mu.Lock()
		s.Terminal.Write([]byte(line + "\n"))
		s.mu.Unlock()
		return
	}

	eventType, ok := streamEvent["type"].(string)
	if !ok {
		return
	}

	switch eventType {
	case "message_start":
		// Claude started responding
		s.addFormattedOutput("🤖 Claude is thinking...\n", "system")

	case "content_block_start":
		// New content block started
		s.addFormattedOutput("\n", "content")

	case "content_block_delta":
		// Incremental content - this is the main response text
		if delta, ok := streamEvent["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok {
				s.addFormattedOutput(text, "response")
			}
		}

	case "content_block_stop":
		// Content block finished
		s.addFormattedOutput("\n", "content")

	case "message_delta":
		// Message metadata update (like stop_reason)
		if delta, ok := streamEvent["delta"].(map[string]interface{}); ok {
			if stopReason, ok := delta["stop_reason"].(string); ok && stopReason != "" {
				s.addFormattedOutput(fmt.Sprintf("\n✅ Complete (%s)\n", stopReason), "system")
			}
		}

	case "message_stop":
		// Claude finished responding
		s.addFormattedOutput("\n🎯 Response complete\n", "system")

	case "error":
		// Handle errors in the stream
		if message, ok := streamEvent["message"].(string); ok {
			s.addFormattedOutput(fmt.Sprintf("\n❌ Error: %s\n", message), "error")
		}

	default:
		// For debugging: show unknown event types
		if s.DebugMode {
			s.addFormattedOutput(fmt.Sprintf("\n[DEBUG] %s: %s\n", eventType, line), "debug")
		}
	}
}

// addFormattedOutput adds formatted text to the terminal buffer
func (s *PTYSession) addFormattedOutput(text, outputType string) {
	// Add some basic formatting based on output type
	var formattedText string
	switch outputType {
	case "system":
		formattedText = fmt.Sprintf("\033[36m%s\033[0m", text) // Cyan
	case "error":
		formattedText = fmt.Sprintf("\033[31m%s\033[0m", text) // Red
	case "debug":
		formattedText = fmt.Sprintf("\033[90m%s\033[0m", text) // Gray
	case "response":
		formattedText = text // Regular text for Claude's response
	default:
		formattedText = text
	}

	s.mu.Lock()
	s.writeToTerminalInChunks([]byte(formattedText))
	s.mu.Unlock()
}

// writeToTerminalInChunks writes data to the terminal with lock-free optimization
func (s *PTYSession) writeToTerminalInChunks(data []byte) {
	// PERFORMANCE OPTIMIZATION: Single write attempt for better responsiveness
	// Chunking and retries cause lag - modern terminals handle large writes well
	if s.Terminal != nil {
		n, err := s.Terminal.Write(data)
		if err != nil {
			// Log terminal write errors for debugging but don't interrupt flow
			s.logTerminalError("terminal write failed", err, map[string]interface{}{
				"session_id":    s.ID,
				"data_length":   len(data),
				"bytes_written": n,
			})
		}
	} else {
		// Log missing terminal for debugging
		s.logTerminalError("terminal not available for write", nil, map[string]interface{}{
			"session_id":  s.ID,
			"data_length": len(data),
		})
	}
}

// logTerminalError logs terminal-related errors with context for debugging
func (s *PTYSession) logTerminalError(message string, err error, context map[string]interface{}) {
	// Only log if we have a way to emit events and session isn't closed
	s.mu.RLock()
	isClosed := s.closed
	s.mu.RUnlock()

	if !isClosed {
		// Create structured log data
		logData := make(map[string]interface{})
		logData["message"] = message
		logData["timestamp"] = time.Now()
		if err != nil {
			logData["error"] = err.Error()
		}
		for k, v := range context {
			logData[k] = v
		}

		// Emit as debug event for monitoring (non-blocking)
		select {
		case s.EventChan <- PTYEvent{
			Type:      PTYEventOutput, // Use output type for log events
			SessionID: s.ID,
			Data:      []byte(fmt.Sprintf("[PTY_LOG] %s", message)),
			Timestamp: time.Now(),
		}:
		default:
			// Drop if channel full - don't block on logging
		}
	}
}

// REMOVED: UpdateChannelStats - not needed for 1-3 sessions
// REMOVED: GetChannelStats - not needed for 1-3 sessions
// REMOVED: channelMonitorLoop - not needed for 1-3 sessions
