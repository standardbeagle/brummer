package aicoder

import (
	"bytes"
	"sync"
)

// RawPTYBuffer stores raw PTY output with ANSI codes intact
// This is designed for direct passthrough to Bubble Tea without terminal emulation
type RawPTYBuffer struct {
	mu sync.RWMutex

	// Circular buffer of lines for display
	lines      [][]byte
	maxLines   int
	currentLine []byte

	// Raw buffer for complete output
	rawBuffer  *bytes.Buffer
	maxRawSize int
}

// NewRawPTYBuffer creates a buffer for raw PTY output
func NewRawPTYBuffer(maxLines int) *RawPTYBuffer {
	return &RawPTYBuffer{
		lines:      make([][]byte, 0, maxLines),
		maxLines:   maxLines,
		rawBuffer:  &bytes.Buffer{},
		maxRawSize: 1024 * 1024, // 1MB max raw buffer
	}
}

// Write processes raw PTY output and stores it
func (b *RawPTYBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Add to raw buffer
	b.rawBuffer.Write(data)

	// Trim raw buffer if too large
	if b.rawBuffer.Len() > b.maxRawSize {
		// Keep last 80% of max size
		content := b.rawBuffer.Bytes()
		keep := int(float64(b.maxRawSize) * 0.8)
		b.rawBuffer.Reset()
		b.rawBuffer.Write(content[len(content)-keep:])
	}

	// Process data for line tracking
	for _, ch := range data {
		switch ch {
		case '\n':
			// New line - save current line and start new one
			if len(b.currentLine) > 0 || true { // Keep empty lines
				b.addLine(b.currentLine)
				b.currentLine = nil
			}
		case '\r':
			// Carriage return - might be followed by \n or used alone
			// Handle \r\n as single newline, \r alone as line overwrite
			// For now, ignore standalone \r
		default:
			// Add to current line (including ANSI escape sequences)
			b.currentLine = append(b.currentLine, ch)
		}
	}

	return len(data), nil
}

// addLine adds a line to the circular buffer
func (b *RawPTYBuffer) addLine(line []byte) {
	// Make a copy to avoid data races
	lineCopy := make([]byte, len(line))
	copy(lineCopy, line)

	b.lines = append(b.lines, lineCopy)

	// Trim old lines if we exceed max
	if len(b.lines) > b.maxLines {
		// Keep the most recent lines
		b.lines = b.lines[len(b.lines)-b.maxLines:]
	}
}

// GetLines returns the current lines for display
func (b *RawPTYBuffer) GetLines() [][]byte {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Return copy to avoid data races
	result := make([][]byte, len(b.lines))
	for i, line := range b.lines {
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)
		result[i] = lineCopy
	}

	// Add current line if not empty
	if len(b.currentLine) > 0 {
		lineCopy := make([]byte, len(b.currentLine))
		copy(lineCopy, b.currentLine)
		result = append(result, lineCopy)
	}

	return result
}

// GetRawOutput returns the complete raw output
func (b *RawPTYBuffer) GetRawOutput() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()

	output := make([]byte, b.rawBuffer.Len())
	copy(output, b.rawBuffer.Bytes())
	return output
}

// Clear resets the buffer
func (b *RawPTYBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines = b.lines[:0]
	b.currentLine = nil
	b.rawBuffer.Reset()
}

// GetScreenContent returns the visible content as a string
// This is for Bubble Tea to render directly with ANSI codes
func (b *RawPTYBuffer) GetScreenContent() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Return the raw buffer directly - let the terminal handle ALL ANSI codes
	// This includes cursor positioning, line updates, colors, etc.
	// The terminal knows how to handle \r for same-line updates
	return b.rawBuffer.String()
}

// GetFullContent returns all content with proper ANSI handling
// This version returns the full raw buffer for complete display
func (b *RawPTYBuffer) GetFullContent() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.rawBuffer.String()
}