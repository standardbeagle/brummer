package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyToPTY converts Bubble Tea KeyMsg to raw PTY bytes
func KeyToPTY(msg tea.KeyMsg) ([]byte, error) {
	// Handle special keys first
	switch msg.Type {

	// Control characters
	case tea.KeyCtrlA:
		return []byte{0x01}, nil
	case tea.KeyCtrlB:
		return []byte{0x02}, nil
	case tea.KeyCtrlC:
		return []byte{0x03}, nil
	case tea.KeyCtrlD:
		return []byte{0x04}, nil
	case tea.KeyCtrlE:
		return []byte{0x05}, nil
	case tea.KeyCtrlF:
		return []byte{0x06}, nil
	case tea.KeyCtrlG:
		return []byte{0x07}, nil
	case tea.KeyCtrlH:
		return []byte{0x08}, nil // Same as backspace
	case tea.KeyCtrlJ:
		return []byte{0x0A}, nil // Same as LF
	case tea.KeyCtrlK:
		return []byte{0x0B}, nil
	case tea.KeyCtrlL:
		return []byte{0x0C}, nil
	case tea.KeyCtrlN:
		return []byte{0x0E}, nil
	case tea.KeyCtrlO:
		return []byte{0x0F}, nil
	case tea.KeyCtrlP:
		return []byte{0x10}, nil
	case tea.KeyCtrlQ:
		return []byte{0x11}, nil
	case tea.KeyCtrlR:
		return []byte{0x12}, nil
	case tea.KeyCtrlS:
		return []byte{0x13}, nil
	case tea.KeyCtrlT:
		return []byte{0x14}, nil
	case tea.KeyCtrlU:
		return []byte{0x15}, nil
	case tea.KeyCtrlV:
		return []byte{0x16}, nil
	case tea.KeyCtrlW:
		return []byte{0x17}, nil
	case tea.KeyCtrlX:
		return []byte{0x18}, nil
	case tea.KeyCtrlY:
		return []byte{0x19}, nil
	case tea.KeyCtrlZ:
		return []byte{0x1A}, nil
	case tea.KeyCtrlBackslash:
		return []byte{0x1C}, nil
	case tea.KeyCtrlCloseBracket:
		return []byte{0x1D}, nil
	case tea.KeyCtrlCaret:
		return []byte{0x1E}, nil
	case tea.KeyCtrlUnderscore:
		return []byte{0x1F}, nil

	// Basic keys
	case tea.KeyEnter:
		return []byte{'\r'}, nil // CR for PTY
	case tea.KeyTab:
		return []byte{'\t'}, nil
	case tea.KeyBackspace:
		return []byte{0x7F}, nil // DEL character
	case tea.KeyEscape:
		return []byte{0x1B}, nil
	case tea.KeySpace:
		return []byte{' '}, nil
	case tea.KeyDelete:
		return []byte("\x1b[3~"), nil

	// Arrow keys (ANSI escape sequences)
	case tea.KeyUp:
		return []byte("\x1b[A"), nil
	case tea.KeyDown:
		return []byte("\x1b[B"), nil
	case tea.KeyRight:
		return []byte("\x1b[C"), nil
	case tea.KeyLeft:
		return []byte("\x1b[D"), nil

	// Shift + Arrow keys
	case tea.KeyShiftUp:
		return []byte("\x1b[1;2A"), nil
	case tea.KeyShiftDown:
		return []byte("\x1b[1;2B"), nil
	case tea.KeyShiftRight:
		return []byte("\x1b[1;2C"), nil
	case tea.KeyShiftLeft:
		return []byte("\x1b[1;2D"), nil

	// Ctrl + Arrow keys
	case tea.KeyCtrlUp:
		return []byte("\x1b[1;5A"), nil
	case tea.KeyCtrlDown:
		return []byte("\x1b[1;5B"), nil
	case tea.KeyCtrlRight:
		return []byte("\x1b[1;5C"), nil
	case tea.KeyCtrlLeft:
		return []byte("\x1b[1;5D"), nil

	// Alt + Arrow keys (if Bubble Tea supports these)
	// Note: May need to check Bubble Tea version for these constants

	// Navigation keys
	case tea.KeyHome:
		return []byte("\x1b[H"), nil
	case tea.KeyEnd:
		return []byte("\x1b[F"), nil
	case tea.KeyPgUp:
		return []byte("\x1b[5~"), nil
	case tea.KeyPgDown:
		return []byte("\x1b[6~"), nil

	// Shift + Navigation
	case tea.KeyShiftHome:
		return []byte("\x1b[1;2H"), nil
	case tea.KeyShiftEnd:
		return []byte("\x1b[1;2F"), nil

	// Function keys
	case tea.KeyF1:
		return []byte("\x1bOP"), nil
	case tea.KeyF2:
		return []byte("\x1bOQ"), nil
	case tea.KeyF3:
		return []byte("\x1bOR"), nil
	case tea.KeyF4:
		return []byte("\x1bOS"), nil
	case tea.KeyF5:
		return []byte("\x1b[15~"), nil
	case tea.KeyF6:
		return []byte("\x1b[17~"), nil
	case tea.KeyF7:
		return []byte("\x1b[18~"), nil
	case tea.KeyF8:
		return []byte("\x1b[19~"), nil
	case tea.KeyF9:
		return []byte("\x1b[20~"), nil
	case tea.KeyF10:
		return []byte("\x1b[21~"), nil
	case tea.KeyF11:
		return []byte("\x1b[23~"), nil
	case tea.KeyF12:
		return []byte("\x1b[24~"), nil

	// Insert
	case tea.KeyInsert:
		return []byte("\x1b[2~"), nil

	// Shift + Tab
	case tea.KeyShiftTab:
		return []byte("\x1b[Z"), nil

	// Handle regular runes and Alt combinations
	case tea.KeyRunes:
		// Check for Alt combinations
		if msg.Alt {
			// Alt+key sends ESC followed by the key
			buf := []byte{0x1B} // ESC
			buf = append(buf, []byte(string(msg.Runes))...)
			return buf, nil
		}
		// Regular characters
		return []byte(string(msg.Runes)), nil
	}

	// Fallback for unhandled types - try to use the string representation
	str := msg.String()

	// Handle special string patterns that might not have specific types
	switch str {
	case "ctrl+space":
		return []byte{0x00}, nil
	case "ctrl+@":
		return []byte{0x00}, nil
	case "ctrl+[":
		return []byte{0x1B}, nil // Same as ESC
	case "ctrl+]":
		return []byte{0x1D}, nil
	case "ctrl+^":
		return []byte{0x1E}, nil
	case "ctrl+_":
		return []byte{0x1F}, nil
	}

	// Last resort - if we have runes, return them
	if len(msg.Runes) > 0 {
		return []byte(string(msg.Runes)), nil
	}

	// Truly unknown - return error
	return nil, fmt.Errorf("unsupported key type: %d", msg.Type)
}
