package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/stretchr/testify/assert"
)

// MockAICoderService captures PTY input for testing
type MockAICoderService struct {
	lastInput []byte
}

func (m *MockAICoderService) SendInput(sessionID string, input []byte) error {
	m.lastInput = input
	return nil
}

// TestKeyToPTYConverter tests the centralized key-to-PTY conversion
func TestKeyToPTYConverter(t *testing.T) {
	tests := []struct {
		name        string
		key         tea.KeyMsg
		expected    []byte
		shouldError bool
	}{
		// Critical keys that were broken
		{"Enter key", tea.KeyMsg{Type: tea.KeyEnter}, []byte("\r"), false},
		{"Space key", tea.KeyMsg{Type: tea.KeySpace}, []byte(" "), false},

		// Navigation keys
		{"Tab key", tea.KeyMsg{Type: tea.KeyTab}, []byte("\t"), false},
		{"Shift+Tab", tea.KeyMsg{Type: tea.KeyShiftTab}, []byte("\x1b[Z"), false},
		{"Up arrow", tea.KeyMsg{Type: tea.KeyUp}, []byte("\x1b[A"), false},
		{"Down arrow", tea.KeyMsg{Type: tea.KeyDown}, []byte("\x1b[B"), false},
		{"Left arrow", tea.KeyMsg{Type: tea.KeyLeft}, []byte("\x1b[D"), false},
		{"Right arrow", tea.KeyMsg{Type: tea.KeyRight}, []byte("\x1b[C"), false},

		// Editing keys
		{"Backspace", tea.KeyMsg{Type: tea.KeyBackspace}, []byte{0x7f}, false},
		{"Delete", tea.KeyMsg{Type: tea.KeyDelete}, []byte("\x1b[3~"), false},
		{"Insert", tea.KeyMsg{Type: tea.KeyInsert}, []byte("\x1b[2~"), false},
		{"Home", tea.KeyMsg{Type: tea.KeyHome}, []byte("\x1b[H"), false},
		{"End", tea.KeyMsg{Type: tea.KeyEnd}, []byte("\x1b[F"), false},
		{"Page Up", tea.KeyMsg{Type: tea.KeyPgUp}, []byte("\x1b[5~"), false},
		{"Page Down", tea.KeyMsg{Type: tea.KeyPgDown}, []byte("\x1b[6~"), false},

		// Control keys
		{"Ctrl+C", tea.KeyMsg{Type: tea.KeyCtrlC}, []byte{0x03}, false},
		{"Ctrl+D", tea.KeyMsg{Type: tea.KeyCtrlD}, []byte{0x04}, false},
		{"Ctrl+Z", tea.KeyMsg{Type: tea.KeyCtrlZ}, []byte{0x1A}, false},
		{"Ctrl+L", tea.KeyMsg{Type: tea.KeyCtrlL}, []byte{0x0C}, false},
		{"Ctrl+A", tea.KeyMsg{Type: tea.KeyCtrlA}, []byte{0x01}, false},
		{"Ctrl+E", tea.KeyMsg{Type: tea.KeyCtrlE}, []byte{0x05}, false},
		{"Ctrl+U", tea.KeyMsg{Type: tea.KeyCtrlU}, []byte{0x15}, false},
		{"Ctrl+K", tea.KeyMsg{Type: tea.KeyCtrlK}, []byte{0x0B}, false},
		{"Ctrl+W", tea.KeyMsg{Type: tea.KeyCtrlW}, []byte{0x17}, false},

		// Function keys
		{"F1", tea.KeyMsg{Type: tea.KeyF1}, []byte("\x1bOP"), false},
		{"F2", tea.KeyMsg{Type: tea.KeyF2}, []byte("\x1bOQ"), false},
		{"F3", tea.KeyMsg{Type: tea.KeyF3}, []byte("\x1bOR"), false},
		{"F4", tea.KeyMsg{Type: tea.KeyF4}, []byte("\x1bOS"), false},
		{"F5", tea.KeyMsg{Type: tea.KeyF5}, []byte("\x1b[15~"), false},

		// Regular characters
		{"Letter 'a'", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, []byte("a"), false},
		{"Number '1'", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, []byte("1"), false},
		{"Symbol '!'", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}}, []byte("!"), false},

		// Unknown key should return error
		{"Unknown key", tea.KeyMsg{Type: 9999}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := KeyToPTY(tt.key)

			if tt.shouldError {
				assert.Error(t, err, "%s should return an error", tt.name)
				assert.Nil(t, result, "%s should return nil bytes on error", tt.name)
			} else {
				assert.NoError(t, err, "%s should not return an error", tt.name)
				assert.Equal(t, tt.expected, result, "%s should convert to correct bytes", tt.name)
			}
		})
	}
}

// TestPTYKeyConversion tests that keys are converted correctly to PTY byte sequences
// using the handler with the new centralized converter
func TestPTYKeyConversion(t *testing.T) {
	tests := []struct {
		name     string
		key      tea.KeyMsg
		expected []byte
	}{
		// Critical keys that were broken
		{"Enter key", tea.KeyMsg{Type: tea.KeyEnter}, []byte("\r")},
		{"Space key", tea.KeyMsg{Type: tea.KeySpace}, []byte(" ")},

		// Navigation keys
		{"Tab key", tea.KeyMsg{Type: tea.KeyTab}, []byte("\t")},
		{"Shift+Tab", tea.KeyMsg{Type: tea.KeyShiftTab}, []byte("\x1b[Z")},
		{"Up arrow", tea.KeyMsg{Type: tea.KeyUp}, []byte("\x1b[A")},
		{"Down arrow", tea.KeyMsg{Type: tea.KeyDown}, []byte("\x1b[B")},
		{"Left arrow", tea.KeyMsg{Type: tea.KeyLeft}, []byte("\x1b[D")},
		{"Right arrow", tea.KeyMsg{Type: tea.KeyRight}, []byte("\x1b[C")},

		// Editing keys
		{"Backspace", tea.KeyMsg{Type: tea.KeyBackspace}, []byte{0x7f}},
		{"Delete", tea.KeyMsg{Type: tea.KeyDelete}, []byte("\x1b[3~")},
		{"Insert", tea.KeyMsg{Type: tea.KeyInsert}, []byte("\x1b[2~")},
		{"Home", tea.KeyMsg{Type: tea.KeyHome}, []byte("\x1b[H")},
		{"End", tea.KeyMsg{Type: tea.KeyEnd}, []byte("\x1b[F")},
		{"Page Up", tea.KeyMsg{Type: tea.KeyPgUp}, []byte("\x1b[5~")},
		{"Page Down", tea.KeyMsg{Type: tea.KeyPgDown}, []byte("\x1b[6~")},

		// Control keys
		{"Ctrl+C", tea.KeyMsg{Type: tea.KeyCtrlC}, []byte{0x03}},
		{"Ctrl+D", tea.KeyMsg{Type: tea.KeyCtrlD}, []byte{0x04}},
		{"Ctrl+Z", tea.KeyMsg{Type: tea.KeyCtrlZ}, []byte{0x1A}},
		{"Ctrl+L", tea.KeyMsg{Type: tea.KeyCtrlL}, []byte{0x0C}},
		{"Ctrl+A", tea.KeyMsg{Type: tea.KeyCtrlA}, []byte{0x01}},
		{"Ctrl+E", tea.KeyMsg{Type: tea.KeyCtrlE}, []byte{0x05}},
		{"Ctrl+U", tea.KeyMsg{Type: tea.KeyCtrlU}, []byte{0x15}},
		{"Ctrl+K", tea.KeyMsg{Type: tea.KeyCtrlK}, []byte{0x0B}},
		{"Ctrl+W", tea.KeyMsg{Type: tea.KeyCtrlW}, []byte{0x17}},

		// Function keys
		{"F1", tea.KeyMsg{Type: tea.KeyF1}, []byte("\x1bOP")},
		{"F2", tea.KeyMsg{Type: tea.KeyF2}, []byte("\x1bOQ")},
		{"F3", tea.KeyMsg{Type: tea.KeyF3}, []byte("\x1bOR")},
		{"F4", tea.KeyMsg{Type: tea.KeyF4}, []byte("\x1bOS")},
		{"F5", tea.KeyMsg{Type: tea.KeyF5}, []byte("\x1b[15~")},

		// Regular characters
		{"Letter 'a'", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, []byte("a")},
		{"Number '1'", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, []byte("1")},
		{"Symbol '!'", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}}, []byte("!")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup model with AI Coder session and terminal focused
			model := &Model{
				masterData: state.NewMasterData(),
			}
			model.viewManager = NewSimpleViewManager(ViewAICoders)
			model.masterData.AICoderState = state.NewAICoderState()
			model.masterData.AICoderState = model.masterData.AICoderState.
				AddSession("test-session", "Test Session").
				SetFocusedSession("test-session").
				SetTerminalFocus(true) // Terminal IS focused

			// Use mock service to capture PTY input
			mockService := &MockAICoderService{}
			model.aiCoderService = mockService

			// Create handler and process key
			handler := NewAICoderKeyHandler(nil).(*AICoderKeyHandler)
			ctx := KeyContext{
				View:    ViewAICoders,
				Context: "terminal_focused",
			}

			// Verify handler can handle the key
			canHandle := handler.CanHandle(tt.key, ctx)
			assert.True(t, canHandle, "%s should be handled when terminal is focused", tt.name)

			// Handle the key
			cmd := handler.Handle(tt.key, model)
			// Most keys should return nil (no command) as they just send to PTY
			// ESC and Ctrl+T are special and return commands
			if tt.key.Type != tea.KeyEsc && tt.key.Type != tea.KeyCtrlT {
				assert.Nil(t, cmd, "%s should not return a command", tt.name)
			}

			// Verify correct bytes were sent to PTY
			assert.Equal(t, tt.expected, mockService.lastInput,
				"%s should send correct bytes to PTY", tt.name)
		})
	}
}

// TestUnknownKeyHandling tests that unknown keys are handled safely
func TestUnknownKeyHandling(t *testing.T) {
	// Setup model
	model := &Model{
		masterData: state.NewMasterData(),
	}
	model.viewManager = NewSimpleViewManager(ViewAICoders)
	model.masterData.AICoderState = state.NewAICoderState()
	model.masterData.AICoderState = model.masterData.AICoderState.
		AddSession("test-session", "Test Session").
		SetFocusedSession("test-session").
		SetTerminalFocus(true)

	mockService := &MockAICoderService{}
	model.aiCoderService = mockService

	handler := NewAICoderKeyHandler(nil).(*AICoderKeyHandler)
	ctx := KeyContext{
		View:    ViewAICoders,
		Context: "terminal_focused",
	}

	// Create an unknown key type (using a value that's not defined)
	unknownKey := tea.KeyMsg{Type: 9999} // Invalid type

	// Should handle it (returns true for all keys when terminal focused)
	canHandle := handler.CanHandle(unknownKey, ctx)
	assert.True(t, canHandle, "Should handle unknown key when terminal focused")

	// But should NOT send anything to PTY
	cmd := handler.Handle(unknownKey, model)
	assert.Nil(t, cmd, "Unknown key should return nil")
	assert.Nil(t, mockService.lastInput, "Unknown key should not send anything to PTY")
}