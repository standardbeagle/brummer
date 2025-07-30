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

// PTYSession represents a pseudo-terminal session for an AI coder
type PTYSession struct {
	ID           string
	Name         string
	Command      *exec.Cmd
	PTY          *os.File
	Terminal     vt10x.Terminal    // vt10x terminal emulator
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
	
	// Create vt10x terminal emulator
	terminal := vt10x.New(vt10x.WithSize(80, 24))
	
	session := &PTYSession{
		ID:           id,
		Name:         name,
		Command:      cmd,
		PTY:          ptmx,
		Terminal:     terminal,
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
				
				// Feed data to terminal emulator
				if s.isStreamJSON {
					s.processStreamJSON(data)
				} else {
					// Write to vt10x terminal
					s.mu.Lock()
					s.Terminal.Write(data)
					s.mu.Unlock()
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
	
	// Resize vt10x terminal
	s.Terminal.Resize(width, height)
	
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

// GetTerminal returns the vt10x terminal emulator
func (s *PTYSession) GetTerminal() vt10x.Terminal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Terminal
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
	s.Terminal.Write([]byte(formattedText))
	s.mu.Unlock()
}