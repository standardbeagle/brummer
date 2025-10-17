// Package tui provides PTY terminal emulation for realistic terminal simulation.
// This file contains terminal emulation capabilities that enhance the PTY integration
// with features like scrollback buffer, cursor tracking, and terminal state management.
package tui

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/standardbeagle/brummer/test/tui/pty"
)

// PTYTerminalEmulator provides advanced terminal emulation for PTY integration
type PTYTerminalEmulator struct {
	mu sync.RWMutex

	// PTY integration
	ptyLayer        *pty.PTYLayerImpl
	ptyIntegration  *PTYIntegration

	// Terminal state
	terminalState   TerminalState
	cursorPosition  CursorPosition
	scrollbackBuffer ScrollbackBuffer

	// Configuration
	config TerminalEmulatorConfig

	// Content management
	screenBuffer     [][]rune
	lineAttributes   []LineAttributes
	scrollbackLines  []string

	// History tracking
	inputHistory     []string
	outputHistory    []string
	maxHistorySize   int

	// Performance tracking
	lastUpdateTime   time.Time
	updateCount      int
	renderCount      int

	// Event handling
	eventCallbacks   map[EventType][]func(TerminalEvent)
}

// TerminalState represents the current terminal emulation state
type TerminalState struct {
	Mode           TerminalMode
	ForegroundColor string
	BackgroundColor string
	Bold           bool
	Underline      bool
	Reverse        bool
	Blink          bool
}

// TerminalMode represents terminal operation modes
type TerminalMode int

const (
	ModeNormal TerminalMode = iota
	ModeApplication
	ModeAlternateScreen
	ModeInsert
	ModeOrigin
)

// CursorPosition represents cursor position and attributes
type CursorPosition struct {
	Row     int
	Col     int
	Visible bool
	Blink   bool
	Shape   CursorShape
}

// CursorShape represents cursor appearance
type CursorShape int

const (
	CursorBlock CursorShape = iota
	CursorUnderline
	CursorBar
)

// LineAttributes represents styling attributes for a line
type LineAttributes struct {
	Bold      bool
	Underline bool
	Reverse   bool
	Blink     bool
	FGColor   string
	BGColor   string
}

// ScrollbackBuffer manages terminal scrollback history
type ScrollbackBuffer struct {
	Lines       []string
	MaxLines    int
	CurrentLine int
	ScrollOffset int
}

// TerminalEmulatorConfig contains configuration for terminal emulation
type TerminalEmulatorConfig struct {
	// Terminal dimensions
	Rows    int
	Cols    int

	// Scrollback buffer
	MaxScrollbackLines int
	EnableScrollback   bool

	// Cursor settings
	DefaultCursorShape CursorShape
	CursorBlinkInterval time.Duration
	ShowCursor         bool

	// Performance settings
	UpdateInterval     time.Duration
	MaxHistorySize     int
	EnableHistory      bool

	// Rendering options
	EnableLineWrapping  bool
	EnableANSIColors    bool
	EnableUTF8          bool
	TabSize            int

	// Debugging
	EnableDebug         bool
	TraceANSISequences  bool
}

// EventType represents types of terminal events
type EventType int

const (
	EventCursorMove EventType = iota
	EventContentChange
	EventScroll
	EventModeChange
	EventResize
	EventBell
)

// TerminalEvent represents a terminal emulation event
type TerminalEvent struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// DefaultTerminalEmulatorConfig returns a default configuration
func DefaultTerminalEmulatorConfig() TerminalEmulatorConfig {
	return TerminalEmulatorConfig{
		Rows:                24,
		Cols:                80,
		MaxScrollbackLines:  1000,
		EnableScrollback:    true,
		DefaultCursorShape:  CursorBlock,
		CursorBlinkInterval: 500 * time.Millisecond,
		ShowCursor:          true,
		UpdateInterval:      16 * time.Millisecond, // ~60 FPS
		MaxHistorySize:      10000,
		EnableHistory:       true,
		EnableLineWrapping:  true,
		EnableANSIColors:    true,
		EnableUTF8:          true,
		TabSize:            4,
		EnableDebug:         false,
		TraceANSISequences:  false,
	}
}

// NewPTYTerminalEmulator creates a new PTY terminal emulator
func NewPTYTerminalEmulator(ptyLayer *pty.PTYLayerImpl, config TerminalEmulatorConfig) *PTYTerminalEmulator {
	te := &PTYTerminalEmulator{
		ptyLayer:       ptyLayer,
		config:         config,
		terminalState:  TerminalState{Mode: ModeNormal},
		cursorPosition: CursorPosition{
			Row:     0,
			Col:     0,
			Visible: config.ShowCursor,
			Blink:   true,
			Shape:   config.DefaultCursorShape,
		},
		scrollbackBuffer: ScrollbackBuffer{
			MaxLines:    config.MaxScrollbackLines,
			CurrentLine: 0,
			ScrollOffset: 0,
		},
		screenBuffer:    make([][]rune, config.Rows),
		lineAttributes:  make([]LineAttributes, config.Rows),
		inputHistory:    make([]string, 0),
		outputHistory:   make([]string, 0),
		maxHistorySize:  config.MaxHistorySize,
		eventCallbacks:  make(map[EventType][]func(TerminalEvent)),
		lastUpdateTime:  time.Now(),
	}

	// Initialize screen buffer
	for i := range te.screenBuffer {
		te.screenBuffer[i] = make([]rune, config.Cols)
		for j := range te.screenBuffer[i] {
			te.screenBuffer[i][j] = ' '
		}
		te.lineAttributes[i] = LineAttributes{}
	}

	return te
}

// SetPTYIntegration sets the PTY integration for advanced control
func (te *PTYTerminalEmulator) SetPTYIntegration(ptyInt *PTYIntegration) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.ptyIntegration = ptyInt
}

// ProcessInput processes input through the terminal emulator
func (te *PTYTerminalEmulator) ProcessInput(input []byte) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	// Add to input history if enabled
	if te.config.EnableHistory {
		te.addToInputHistory(string(input))
	}

	// Process ANSI escape sequences and regular input
	te.processANSIInput(input)

	// Update cursor position and terminal state
	te.updateTerminalState()

	// Generate events
	te.generateEvent(EventContentChange, TerminalEvent{
		Type:      EventContentChange,
		Timestamp: time.Now(),
		Data:      input,
	})

	te.lastUpdateTime = time.Now()
	te.updateCount++

	return nil
}

// processANSIInput processes ANSI escape sequences and regular character input
func (te *PTYTerminalEmulator) processANSIInput(input []byte) {
	i := 0
	for i < len(input) {
		char := input[i]

		switch char {
		case 0x1B: // ESC
			if i+1 < len(input) && input[i+1] == '[' {
				// CSI sequence
				i, _ = te.processCSISequence(input, i+2)
			} else if i+1 < len(input) && input[i+1] == ']' {
				// OSC sequence
				i, _ = te.processOSCSequence(input, i+2)
			} else {
				// Simple escape
				i++
			}

		case '\r': // Carriage return
			te.cursorPosition.Col = 0
			i++

		case '\n': // Line feed
			te.newLine()
			i++

		case '\t': // Tab
			te.cursorPosition.Col += te.config.TabSize
			if te.cursorPosition.Col >= te.config.Cols {
				te.cursorPosition.Col = te.config.Cols - 1
			}
			i++

		case '\x7F': // Backspace
			if te.cursorPosition.Col > 0 {
				te.cursorPosition.Col--
				te.screenBuffer[te.cursorPosition.Row][te.cursorPosition.Col] = ' '
			}
			i++

		case '\b': // Backspace (alternative)
			if te.cursorPosition.Col > 0 {
				te.cursorPosition.Col--
				te.screenBuffer[te.cursorPosition.Row][te.cursorPosition.Col] = ' '
			}
			i++

		case '\a': // Bell
			te.generateEvent(EventBell, TerminalEvent{
				Type:      EventBell,
				Timestamp: time.Now(),
			})
			i++

		default:
			// Regular character
			if char >= 32 && char <= 126 {
				te.writeChar(rune(char))
			}
			i++
		}
	}
}

// processCSISequence processes CSI (Control Sequence Introducer) sequences
func (te *PTYTerminalEmulator) processCSISequence(input []byte, start int) (int, error) {
	if start >= len(input) {
		return start, nil
	}

	// Parse parameters and final character
	params := make([]int, 0)
	currentParam := 0
	haveParam := false

	i := start
	for i < len(input) {
		char := input[i]

		if char >= '0' && char <= '9' {
			currentParam = currentParam*10 + int(char-'0')
			haveParam = true
		} else if char == ';' {
			if haveParam {
				params = append(params, currentParam)
			} else {
				params = append(params, 0)
			}
			currentParam = 0
			haveParam = false
		} else {
			// End of CSI sequence
			if haveParam {
				params = append(params, currentParam)
			}

			// Handle the CSI command
			te.handleCSICommand(char, params)

			if te.config.TraceANSISequences {
				fmt.Printf("CSI: ESC[ %c %v\n", char, params)
			}

			return i + 1, nil
		}
		i++
	}

	return i, nil
}

// processOSCSequence processes OSC (Operating System Command) sequences
func (te *PTYTerminalEmulator) processOSCSequence(input []byte, start int) (int, error) {
	// Simple OSC implementation - just skip until ST (String Terminator)
	i := start
	for i < len(input) {
		if input[i] == 0x1B && i+1 < len(input) && input[i+1] == '\\' {
			return i + 2, nil
		}
		if input[i] == '\x9c' { // ST in 8-bit form
			return i + 1, nil
		}
		i++
	}
	return i, nil
}

// handleCSICommand handles specific CSI commands
func (te *PTYTerminalEmulator) handleCSICommand(cmd byte, params []int) {
	switch cmd {
	case 'A': // Cursor Up
		rows := 1
		if len(params) > 0 && params[0] > 0 {
			rows = params[0]
		}
		te.cursorPosition.Row = max(0, te.cursorPosition.Row-rows)
		te.generateEvent(EventCursorMove, TerminalEvent{
			Type: EventCursorMove,
			Data: te.cursorPosition,
		})

	case 'B': // Cursor Down
		rows := 1
		if len(params) > 0 && params[0] > 0 {
			rows = params[0]
		}
		te.cursorPosition.Row = min(te.config.Rows-1, te.cursorPosition.Row+rows)
		te.generateEvent(EventCursorMove, TerminalEvent{
			Type: EventCursorMove,
			Data: te.cursorPosition,
		})

	case 'C': // Cursor Right
		cols := 1
		if len(params) > 0 && params[0] > 0 {
			cols = params[0]
		}
		te.cursorPosition.Col = min(te.config.Cols-1, te.cursorPosition.Col+cols)
		te.generateEvent(EventCursorMove, TerminalEvent{
			Type: EventCursorMove,
			Data: te.cursorPosition,
		})

	case 'D': // Cursor Left
		cols := 1
		if len(params) > 0 && params[0] > 0 {
			cols = params[0]
		}
		te.cursorPosition.Col = max(0, te.cursorPosition.Col-cols)
		te.generateEvent(EventCursorMove, TerminalEvent{
			Type: EventCursorMove,
			Data: te.cursorPosition,
		})

	case 'H', 'f': // Cursor Position
		row := 1
		col := 1
		if len(params) > 0 && params[0] > 0 {
			row = params[0]
		}
		if len(params) > 1 && params[1] > 0 {
			col = params[1]
		}
		te.cursorPosition.Row = min(te.config.Rows-1, max(0, row-1))
		te.cursorPosition.Col = min(te.config.Cols-1, max(0, col-1))
		te.generateEvent(EventCursorMove, TerminalEvent{
			Type: EventCursorMove,
			Data: te.cursorPosition,
		})

	case 'J': // Erase in Display
		eraseMode := 0
		if len(params) > 0 {
			eraseMode = params[0]
		}
		te.eraseDisplay(eraseMode)

	case 'K': // Erase in Line
		eraseMode := 0
		if len(params) > 0 {
			eraseMode = params[0]
		}
		te.eraseLine(eraseMode)

	case 'm': // Select Graphic Rendition (SGR)
		te.handleSGR(params)

	case 's': // Save Cursor Position
		// Implementation would save current cursor position

	case 'u': // Restore Cursor Position
		// Implementation would restore saved cursor position

	case '?': // Private mode commands
		// Handle private mode commands like ?25 for cursor visibility
		if len(params) > 0 {
			switch params[0] {
			case 25:
				te.cursorPosition.Visible = true
			case 0: // DEC private mode reset
				if len(params) > 1 && params[1] == 25 {
					te.cursorPosition.Visible = true
				}
			}
		}
	}
}

// handleSGR handles Select Graphic Rendition (color and formatting)
func (te *PTYTerminalEmulator) handleSGR(params []int) {
	if len(params) == 0 {
		// Reset all attributes
		te.terminalState = TerminalState{
			Mode: te.terminalState.Mode,
		}
		return
	}

	for _, param := range params {
		switch param {
		case 0: // Reset
			te.terminalState.Bold = false
			te.terminalState.Underline = false
			te.terminalState.Reverse = false
			te.terminalState.Blink = false
			te.terminalState.ForegroundColor = ""
			te.terminalState.BackgroundColor = ""

		case 1: // Bold
			te.terminalState.Bold = true

		case 4: // Underline
			te.terminalState.Underline = true

		case 5: // Blink
			te.terminalState.Blink = true

		case 7: // Reverse
			te.terminalState.Reverse = true

		case 30, 31, 32, 33, 34, 35, 36, 37: // Set foreground color
			colors := []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}
			te.terminalState.ForegroundColor = colors[param-30]

		case 40, 41, 42, 43, 44, 45, 46, 47: // Set background color
			colors := []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}
			te.terminalState.BackgroundColor = colors[param-40]
		}
	}
}

// writeChar writes a character to the screen buffer at the current cursor position
func (te *PTYTerminalEmulator) writeChar(char rune) {
	if te.cursorPosition.Row >= te.config.Rows || te.cursorPosition.Col >= te.config.Cols {
		return
	}

	te.screenBuffer[te.cursorPosition.Row][te.cursorPosition.Col] = char

	// Apply current terminal state to line attributes
	te.lineAttributes[te.cursorPosition.Row] = LineAttributes{
		Bold:   te.terminalState.Bold,
		Underline: te.terminalState.Underline,
		Reverse: te.terminalState.Reverse,
		Blink:  te.terminalState.Blink,
		FGColor: te.terminalState.ForegroundColor,
		BGColor: te.terminalState.BackgroundColor,
	}

	// Move cursor
	te.cursorPosition.Col++
	if te.cursorPosition.Col >= te.config.Cols {
		if te.config.EnableLineWrapping {
			te.cursorPosition.Col = 0
			te.newLine()
		} else {
			te.cursorPosition.Col = te.config.Cols - 1
		}
	}
}

// newLine creates a new line, handling scrollback
func (te *PTYTerminalEmulator) newLine() {
	// Save current line to scrollback buffer if enabled
	if te.config.EnableScrollback {
		currentLine := string(te.screenBuffer[te.cursorPosition.Row])
		te.addToScrollback(currentLine)
	}

	te.cursorPosition.Row++
	if te.cursorPosition.Row >= te.config.Rows {
		// Scroll up
		te.scrollUp()
		te.cursorPosition.Row = te.config.Rows - 1
	}

	// Clear the new line
	for i := range te.screenBuffer[te.cursorPosition.Row] {
		te.screenBuffer[te.cursorPosition.Row][i] = ' '
	}
}

// scrollUp scrolls the screen buffer up by one line
func (te *PTYTerminalEmulator) scrollUp() {
	// Move all lines up
	for i := 0; i < te.config.Rows-1; i++ {
		copy(te.screenBuffer[i], te.screenBuffer[i+1])
		te.lineAttributes[i] = te.lineAttributes[i+1]
	}

	// Clear the last line
	for i := range te.screenBuffer[te.config.Rows-1] {
		te.screenBuffer[te.config.Rows-1][i] = ' '
	}
	te.lineAttributes[te.config.Rows-1] = LineAttributes{}

	te.generateEvent(EventScroll, TerminalEvent{
		Type: EventScroll,
		Data: 1,
	})
}

// eraseDisplay erases parts of the display
func (te *PTYTerminalEmulator) eraseDisplay(mode int) {
	switch mode {
	case 0: // From cursor to end of screen
		for row := te.cursorPosition.Row; row < te.config.Rows; row++ {
			startCol := 0
			if row == te.cursorPosition.Row {
				startCol = te.cursorPosition.Col
			}
			for col := startCol; col < te.config.Cols; col++ {
				te.screenBuffer[row][col] = ' '
			}
		}

	case 1: // From beginning to cursor
		for row := 0; row <= te.cursorPosition.Row; row++ {
			endCol := te.config.Cols
			if row == te.cursorPosition.Row {
				endCol = te.cursorPosition.Col + 1
			}
			for col := 0; col < endCol; col++ {
				te.screenBuffer[row][col] = ' '
			}
		}

	case 2: // Entire screen
		for row := 0; row < te.config.Rows; row++ {
			for col := 0; col < te.config.Cols; col++ {
				te.screenBuffer[row][col] = ' '
			}
		}
	}
}

// eraseLine erases parts of the current line
func (te *PTYTerminalEmulator) eraseLine(mode int) {
	switch mode {
	case 0: // From cursor to end of line
		for col := te.cursorPosition.Col; col < te.config.Cols; col++ {
			te.screenBuffer[te.cursorPosition.Row][col] = ' '
		}

	case 1: // From beginning to cursor
		for col := 0; col <= te.cursorPosition.Col; col++ {
			te.screenBuffer[te.cursorPosition.Row][col] = ' '
		}

	case 2: // Entire line
		for col := 0; col < te.config.Cols; col++ {
			te.screenBuffer[te.cursorPosition.Row][col] = ' '
		}
	}
}

// addToScrollback adds a line to the scrollback buffer
func (te *PTYTerminalEmulator) addToScrollback(line string) {
	te.scrollbackBuffer.Lines = append(te.scrollbackBuffer.Lines, line)
	if len(te.scrollbackBuffer.Lines) > te.scrollbackBuffer.MaxLines {
		te.scrollbackBuffer.Lines = te.scrollbackBuffer.Lines[1:]
	}
}

// addToInputHistory adds input to the history
func (te *PTYTerminalEmulator) addToInputHistory(input string) {
	te.inputHistory = append(te.inputHistory, input)
	if len(te.inputHistory) > te.maxHistorySize {
		te.inputHistory = te.inputHistory[1:]
	}
}

// addToOutputHistory adds output to the history
func (te *PTYTerminalEmulator) addToOutputHistory(output string) {
	te.outputHistory = append(te.outputHistory, output)
	if len(te.outputHistory) > te.maxHistorySize {
		te.outputHistory = te.outputHistory[1:]
	}
}

// updateTerminalState updates internal terminal state
func (te *PTYTerminalEmulator) updateTerminalState() {
	// Update performance counters
	te.updateCount++
}

// generateEvent generates a terminal event
func (te *PTYTerminalEmulator) generateEvent(eventType EventType, event TerminalEvent) {
	if callbacks, exists := te.eventCallbacks[eventType]; exists {
		for _, callback := range callbacks {
			// NO MORE GOROUTINES - Event callbacks should be command-driven
			// through BubbleTea Update() cycle instead of background goroutine
			// Store events for later processing in UI update loop
			// For now, execute synchronously to prevent race conditions
			callback(event)
		}
	}
}

// Render renders the terminal content as a string
func (te *PTYTerminalEmulator) Render() string {
	te.mu.RLock()
	defer te.mu.RUnlock()

	var buffer bytes.Buffer

	for row := 0; row < te.config.Rows; row++ {
		line := string(te.screenBuffer[row])

		// Apply line attributes if ANSI colors are enabled
		if te.config.EnableANSIColors && te.lineAttributes[row] != (LineAttributes{}) {
			line = te.applyLineAttributes(line, te.lineAttributes[row])
		}

		buffer.WriteString(line)

		// Add cursor if at this position and visible
		if te.cursorPosition.Visible && te.cursorPosition.Row == row {
			cursorLine := strings.Repeat(" ", te.cursorPosition.Col) + "█"
			if te.cursorPosition.Col < te.config.Cols {
				buffer.WriteString(cursorLine)
			}
		}

		if row < te.config.Rows-1 {
			buffer.WriteString("\n")
		}
	}

	te.renderCount++
	return buffer.String()
}

// applyLineAttributes applies styling attributes to a line
func (te *PTYTerminalEmulator) applyLineAttributes(line string, attrs LineAttributes) string {
	// This is a simplified implementation
	// In a full implementation, you'd generate ANSI escape sequences
	if attrs.Bold {
		line = fmt.Sprintf("\x1b[1m%s\x1b[22m", line)
	}
	if attrs.Underline {
		line = fmt.Sprintf("\x1b[4m%s\x1b[24m", line)
	}
	if attrs.Reverse {
		line = fmt.Sprintf("\x1b[7m%s\x1b[27m", line)
	}
	if attrs.Blink {
		line = fmt.Sprintf("\x1b[5m%s\x1b[25m", line)
	}
	if attrs.FGColor != "" {
		colorMap := map[string]int{
			"black":   30,
			"red":     31,
			"green":   32,
			"yellow":  33,
			"blue":    34,
			"magenta": 35,
			"cyan":    36,
			"white":   37,
		}
		if color, exists := colorMap[attrs.FGColor]; exists {
			line = fmt.Sprintf("\x1b[%dm%s\x1b[39m", color, line)
		}
	}

	return line
}

// Resize resizes the terminal emulator
func (te *PTYTerminalEmulator) Resize(rows, cols int) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	if rows < 1 || cols < 1 {
		return fmt.Errorf("invalid terminal size: %dx%d", rows, cols)
	}

	// Create new buffers
	newScreenBuffer := make([][]rune, rows)
	newLineAttributes := make([]LineAttributes, rows)

	// Copy existing content
	for i := 0; i < min(rows, len(te.screenBuffer)); i++ {
		newScreenBuffer[i] = make([]rune, cols)
		newLineAttributes[i] = te.lineAttributes[i]

		for j := 0; j < min(cols, len(te.screenBuffer[i])); j++ {
			newScreenBuffer[i][j] = te.screenBuffer[i][j]
		}

		// Fill remaining columns with spaces
		for j := len(te.screenBuffer[i]); j < cols; j++ {
			newScreenBuffer[i][j] = ' '
		}
	}

	// Fill remaining rows with empty lines
	for i := len(te.screenBuffer); i < rows; i++ {
		newScreenBuffer[i] = make([]rune, cols)
		for j := range newScreenBuffer[i] {
			newScreenBuffer[i][j] = ' '
		}
		newLineAttributes[i] = LineAttributes{}
	}

	te.screenBuffer = newScreenBuffer
	te.lineAttributes = newLineAttributes
	te.config.Rows = rows
	te.config.Cols = cols

	// Adjust cursor position if necessary
	te.cursorPosition.Row = min(te.cursorPosition.Row, rows-1)
	te.cursorPosition.Col = min(te.cursorPosition.Col, cols-1)

	// Update PTY layer if available
	if te.ptyLayer != nil {
		size := pty.PTYSize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		}
		te.ptyLayer.SetSize(size)
	}

	te.generateEvent(EventResize, TerminalEvent{
		Type: EventResize,
		Data: struct{ Rows, Cols int }{rows, cols},
	})

	if te.config.EnableDebug {
		fmt.Printf("Terminal resized to %dx%d\n", rows, cols)
	}

	return nil
}

// GetCursorPosition returns the current cursor position
func (te *PTYTerminalEmulator) GetCursorPosition() CursorPosition {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.cursorPosition
}

// GetTerminalState returns the current terminal state
func (te *PTYTerminalEmulator) GetTerminalState() TerminalState {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.terminalState
}

// GetScrollbackBuffer returns the scrollback buffer
func (te *PTYTerminalEmulator) GetScrollbackBuffer() []string {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return append([]string{}, te.scrollbackBuffer.Lines...)
}

// GetInputHistory returns the input history
func (te *PTYTerminalEmulator) GetInputHistory() []string {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return append([]string{}, te.inputHistory...)
}

// GetOutputHistory returns the output history
func (te *PTYTerminalEmulator) GetOutputHistory() []string {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return append([]string{}, te.outputHistory...)
}

// GetPerformanceStats returns performance statistics
func (te *PTYTerminalEmulator) GetPerformanceStats() map[string]interface{} {
	te.mu.RLock()
	defer te.mu.RUnlock()

	return map[string]interface{}{
		"update_count":    te.updateCount,
		"render_count":    te.renderCount,
		"last_update":     te.lastUpdateTime,
		"uptime_seconds":  time.Since(te.lastUpdateTime).Seconds(),
		"screen_rows":     te.config.Rows,
		"screen_cols":     te.config.Cols,
		"scrollback_size": len(te.scrollbackBuffer.Lines),
		"input_history":   len(te.inputHistory),
		"output_history":  len(te.outputHistory),
	}
}

// AddEventCallback adds a callback for terminal events
func (te *PTYTerminalEmulator) AddEventCallback(eventType EventType, callback func(TerminalEvent)) {
	te.mu.Lock()
	defer te.mu.Unlock()

	if te.eventCallbacks[eventType] == nil {
		te.eventCallbacks[eventType] = make([]func(TerminalEvent), 0)
	}
	te.eventCallbacks[eventType] = append(te.eventCallbacks[eventType], callback)
}

// ClearScreen clears the entire screen
func (te *PTYTerminalEmulator) ClearScreen() {
	te.mu.Lock()
	defer te.mu.Unlock()

	for row := 0; row < te.config.Rows; row++ {
		for col := 0; col < te.config.Cols; col++ {
			te.screenBuffer[row][col] = ' '
		}
		te.lineAttributes[row] = LineAttributes{}
	}

	te.cursorPosition.Row = 0
	te.cursorPosition.Col = 0
}

// Reset resets the terminal to initial state
func (te *PTYTerminalEmulator) Reset() {
	te.mu.Lock()
	defer te.mu.Unlock()

	te.terminalState = TerminalState{Mode: ModeNormal}
	te.cursorPosition = CursorPosition{
		Row:     0,
		Col:     0,
		Visible: te.config.ShowCursor,
		Blink:   true,
		Shape:   te.config.DefaultCursorShape,
	}

	te.ClearScreen()
	te.scrollbackBuffer.Lines = te.scrollbackBuffer.Lines[:0]
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}