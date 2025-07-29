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
)

// PTYSession represents a pseudo-terminal session for an AI coder
type PTYSession struct {
	ID           string
	Name         string
	Command      *exec.Cmd
	PTY          *os.File
	Buffer       *TerminalBuffer
	IsActive     bool
	IsFullScreen bool
	DebugMode    bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	
	// Event channels
	OutputChan   chan []byte
	InputChan    chan []byte
	EventChan    chan PTYEvent
	
	// Brummer integration
	dataInjector *DataInjector
	
	// Stream JSON parsing for non-interactive mode
	isStreamJSON bool
	streamBuffer strings.Builder
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
	PTYEventOutput      PTYEventType = "output"
	PTYEventInput       PTYEventType = "input"
	PTYEventResize      PTYEventType = "resize"
	PTYEventClose       PTYEventType = "close"
	PTYEventDataInject  PTYEventType = "data_inject"
)

// TerminalBuffer manages the terminal screen state
type TerminalBuffer struct {
	Lines      []TerminalLine
	Width      int
	Height     int
	CursorX    int
	CursorY    int
	Scrollback []TerminalLine
	mu         sync.RWMutex
}

// TerminalLine represents a line in the terminal
type TerminalLine struct {
	Content string
	Style   TerminalStyle
}

// TerminalStyle represents text styling (colors, formatting)
type TerminalStyle struct {
	FgColor   int
	BgColor   int
	Bold      bool
	Italic    bool
	Underline bool
}

// NewPTYSession creates a new PTY session for an AI coder  
func NewPTYSession(id, name, command string, args []string) (*PTYSession, error) {
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
	
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	
	session := &PTYSession{
		ID:           id,
		Name:         name,
		Command:      cmd,
		PTY:          ptmx,
		Buffer:       NewTerminalBuffer(80, 24),
		IsActive:     true,
		IsFullScreen: false,
		DebugMode:    false,
		ctx:          ctx,
		cancel:       cancel,
		OutputChan:   make(chan []byte, 100),
		InputChan:    make(chan []byte, 100),
		EventChan:    make(chan PTYEvent, 100),
		dataInjector: NewDataInjector(),
		isStreamJSON: isStreamJSON,
	}
	
	// Start I/O goroutines
	go session.readLoop()
	go session.writeLoop()
	
	return session, nil
}

// NewTerminalBuffer creates a new terminal buffer
func NewTerminalBuffer(width, height int) *TerminalBuffer {
	lines := make([]TerminalLine, height)
	for i := range lines {
		lines[i] = TerminalLine{Content: "", Style: TerminalStyle{}}
	}
	
	return &TerminalBuffer{
		Lines:      lines,
		Width:      width,
		Height:     height,
		CursorX:    0,
		CursorY:    0,
		Scrollback: make([]TerminalLine, 0, 1000), // Keep 1000 lines of scrollback
	}
}

// readLoop continuously reads from the PTY and processes output
func (s *PTYSession) readLoop() {
	defer close(s.OutputChan)
	
	reader := bufio.NewReader(s.PTY)
	buffer := make([]byte, 4096)
	
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					s.EventChan <- PTYEvent{
						Type:      PTYEventClose,
						SessionID: s.ID,
						Data:      fmt.Sprintf("Read error: %v", err),
						Timestamp: time.Now(),
					}
				}
				return
			}
			
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				
				// Process the data through terminal buffer
				if s.isStreamJSON {
					s.processStreamJSON(data)
				} else {
					s.Buffer.ProcessOutput(data)
				}
				
				// Send to output channel
				select {
				case s.OutputChan <- data:
				case <-s.ctx.Done():
					return
				}
				
				// Emit output event
				s.EventChan <- PTYEvent{
					Type:      PTYEventOutput,
					SessionID: s.ID,
					Data:      data,
					Timestamp: time.Now(),
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
			if _, err := s.PTY.Write(data); err != nil {
				s.EventChan <- PTYEvent{
					Type:      PTYEventClose,
					SessionID: s.ID,
					Data:      fmt.Sprintf("Write error: %v", err),
					Timestamp: time.Now(),
				}
				return
			}
			
			// Emit input event
			s.EventChan <- PTYEvent{
				Type:      PTYEventInput,
				SessionID: s.ID,
				Data:      data,
				Timestamp: time.Now(),
			}
		}
	}
}

// WriteInput sends input to the PTY session
func (s *PTYSession) WriteInput(data []byte) error {
	select {
	case s.InputChan <- data:
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("session closed")
	default:
		return fmt.Errorf("input buffer full")
	}
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
	s.EventChan <- PTYEvent{
		Type:      PTYEventDataInject,
		SessionID: s.ID,
		Data: map[string]interface{}{
			"type": dataType,
			"data": data,
			"text": formattedText,
		},
		Timestamp: time.Now(),
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
	
	s.Buffer.Resize(width, height)
	
	s.EventChan <- PTYEvent{
		Type:      PTYEventResize,
		SessionID: s.ID,
		Data: map[string]int{
			"width":  width,
			"height": height,
		},
		Timestamp: time.Now(),
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

// GetBuffer returns the current terminal buffer (read-only)
func (s *PTYSession) GetBuffer() *TerminalBuffer {
	return s.Buffer
}

// Close closes the PTY session
func (s *PTYSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.IsActive {
		return nil
	}
	
	s.IsActive = false
	s.cancel()
	
	// Close channels
	close(s.InputChan)
	close(s.EventChan)
	
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

// ProcessOutput processes terminal output and updates the buffer
func (tb *TerminalBuffer) ProcessOutput(data []byte) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	// This is a simplified implementation
	// In a full implementation, we'd parse ANSI escape sequences
	text := string(data)
	
	// For now, just append to the current line
	if tb.CursorY < len(tb.Lines) {
		tb.Lines[tb.CursorY].Content += text
	}
	
	// Handle newlines
	for _, char := range text {
		if char == '\n' {
			tb.CursorY++
			tb.CursorX = 0
			
			// Scroll if needed
			if tb.CursorY >= tb.Height {
				// Move top line to scrollback
				if len(tb.Lines) > 0 {
					tb.Scrollback = append(tb.Scrollback, tb.Lines[0])
					if len(tb.Scrollback) > 1000 {
						tb.Scrollback = tb.Scrollback[1:]
					}
				}
				
				// Shift lines up
				copy(tb.Lines, tb.Lines[1:])
				tb.Lines[tb.Height-1] = TerminalLine{Content: "", Style: TerminalStyle{}}
				tb.CursorY = tb.Height - 1
			}
		}
	}
}

// Resize resizes the terminal buffer
func (tb *TerminalBuffer) Resize(width, height int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.Width = width
	tb.Height = height
	
	// Resize lines array
	if len(tb.Lines) < height {
		// Add more lines
		for i := len(tb.Lines); i < height; i++ {
			tb.Lines = append(tb.Lines, TerminalLine{Content: "", Style: TerminalStyle{}})
		}
	} else if len(tb.Lines) > height {
		// Move excess lines to scrollback
		excess := tb.Lines[height:]
		tb.Scrollback = append(tb.Scrollback, excess...)
		if len(tb.Scrollback) > 1000 {
			tb.Scrollback = tb.Scrollback[len(tb.Scrollback)-1000:]
		}
		tb.Lines = tb.Lines[:height]
	}
	
	// Adjust cursor position
	if tb.CursorY >= height {
		tb.CursorY = height - 1
	}
}

// GetLines returns the current terminal lines (read-only)
func (tb *TerminalBuffer) GetLines() []TerminalLine {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	
	lines := make([]TerminalLine, len(tb.Lines))
	copy(lines, tb.Lines)
	return lines
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
		s.Buffer.ProcessOutput([]byte(line + "\n"))
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
	
	s.Buffer.ProcessOutput([]byte(formattedText))
}