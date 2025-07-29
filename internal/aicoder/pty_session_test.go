package aicoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalBuffer_Resize_HandlesNegativeDimensions(t *testing.T) {
	tests := []struct {
		name         string
		inputWidth   int
		inputHeight  int
		wantWidth    int
		wantHeight   int
		initialLines int
	}{
		{
			name:         "negative width and height",
			inputWidth:   -10,
			inputHeight:  -5,
			wantWidth:    80,
			wantHeight:   24,
			initialLines: 10,
		},
		{
			name:         "zero width and height",
			inputWidth:   0,
			inputHeight:  0,
			wantWidth:    80,
			wantHeight:   24,
			initialLines: 10,
		},
		{
			name:         "valid dimensions",
			inputWidth:   100,
			inputHeight:  40,
			wantWidth:    100,
			wantHeight:   40,
			initialLines: 10,
		},
		{
			name:         "resize to smaller height moves lines to scrollback",
			inputWidth:   80,
			inputHeight:  5,
			wantWidth:    80,
			wantHeight:   5,
			initialLines: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer with initial lines
			buffer := &TerminalBuffer{
				Lines: make([]TerminalLine, tt.initialLines),
			}
			
			// Initialize lines with content
			for i := range buffer.Lines {
				buffer.Lines[i] = TerminalLine{
					Content: "Test line",
					Style:   TerminalStyle{},
				}
			}

			// Should not panic even with negative dimensions
			require.NotPanics(t, func() {
				buffer.Resize(tt.inputWidth, tt.inputHeight)
			})

			// Check dimensions were set correctly
			assert.Equal(t, tt.wantWidth, buffer.Width)
			assert.Equal(t, tt.wantHeight, buffer.Height)
			
			// Check lines array was resized correctly
			assert.LessOrEqual(t, len(buffer.Lines), buffer.Height)
			
			// If we resized to smaller, check scrollback
			if tt.initialLines > tt.wantHeight && tt.wantHeight > 0 {
				expectedScrollback := tt.initialLines - tt.wantHeight
				assert.Equal(t, expectedScrollback, len(buffer.Scrollback))
			}
		})
	}
}

func TestPTYSession_Resize_HandlesInvalidDimensions(t *testing.T) {
	// Create a PTY session with a buffer
	session := &PTYSession{
		ID:       "test-session",
		Name:     "Test Session",
		IsActive: true,
		Buffer:   &TerminalBuffer{Lines: make([]TerminalLine, 10)},
	}

	// Test cases that would have caught the panic
	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"negative dimensions", -10, -5},
		{"zero dimensions", 0, 0},
		{"negative width only", -5, 24},
		{"negative height only", 80, -10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				session.Resize(tc.width, tc.height)
			})
			
			// Verify buffer has valid dimensions
			assert.Greater(t, session.Buffer.Width, 0)
			assert.Greater(t, session.Buffer.Height, 0)
		})
	}
}

func TestTerminalBuffer_SliceBoundsCheck(t *testing.T) {
	// This specifically tests the slice bounds issue that caused the panic
	buffer := &TerminalBuffer{
		Lines: make([]TerminalLine, 10),
	}
	
	// Fill with some content
	for i := range buffer.Lines {
		buffer.Lines[i] = TerminalLine{Content: "Line content"}
	}

	// Try to resize to a negative height (which gets corrected to 24)
	// This was the exact scenario: tb.Lines[height:] with negative height
	require.NotPanics(t, func() {
		buffer.Resize(80, -8) // Would have caused: slice bounds out of range [-8:]
	})
	
	// Buffer should have been resized to default height
	assert.Equal(t, 24, buffer.Height)
	assert.Equal(t, 24, len(buffer.Lines))
}