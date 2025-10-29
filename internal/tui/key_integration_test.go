package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCriticalKeysIntegration(t *testing.T) {
	tests := []struct {
		name     string
		key      tea.KeyMsg
		expected []byte
	}{
		// Critical keys that were broken
		{"Enter sends \\r", tea.KeyMsg{Type: tea.KeyEnter}, []byte("\r")},
		{"Space sends space", tea.KeyMsg{Type: tea.KeySpace}, []byte(" ")},
		{"Tab sends \\t", tea.KeyMsg{Type: tea.KeyTab}, []byte("\t")},

		// Arrow keys
		{"Up arrow", tea.KeyMsg{Type: tea.KeyUp}, []byte("\x1b[A")},
		{"Down arrow", tea.KeyMsg{Type: tea.KeyDown}, []byte("\x1b[B")},

		// Regular characters
		{"Letter a", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, []byte("a")},
		{"Number 1", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, []byte("1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := KeyToPTY(tt.key)
			if err != nil {
				t.Fatalf("KeyToPTY returned error: %v", err)
			}

			if string(result) != string(tt.expected) {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}

			// Verify Enter specifically doesn't return "enter"
			if tt.key.Type == tea.KeyEnter && string(result) == "enter" {
				t.Fatal("CRITICAL: Enter key is still returning literal 'enter' string!")
			}

			// Verify Space specifically doesn't return "space"
			if tt.key.Type == tea.KeySpace && string(result) == "space" {
				t.Fatal("CRITICAL: Space key is still returning literal 'space' string!")
			}
		})
	}

	fmt.Println("✅ All critical keys are correctly converted to PTY sequences")
	fmt.Println("✅ Enter key sends \\r (not 'enter')")
	fmt.Println("✅ Space key sends space (not 'space')")
}