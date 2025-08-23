package components

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

// WidthHelpers provides safe width calculation utilities for components
// This centralizes width calculations to prevent common mistakes

// MustInitColors forces color output initialization
// Call this in component constructors to ensure consistent rendering
func MustInitColors() {
	lipgloss.SetColorProfile(termenv.ANSI256)
}

// SafeStringWidth calculates display width properly accounting for:
// - Multi-byte UTF-8 characters
// - Double-width emojis and CJK characters
// - ANSI escape codes (strips them first)
// 
// NEVER use len() for display width calculations!
func SafeStringWidth(s string) int {
	// First strip any ANSI codes
	cleaned := stripANSI(s)
	// Then calculate actual display width
	return runewidth.StringWidth(cleaned)
}

// SafeLipglossWidth uses lipgloss's width calculation which handles:
// - ANSI escape codes
// - Styled text
// - Emojis and wide characters
//
// Prefer this when working with styled text
func SafeLipglossWidth(s string) int {
	return lipgloss.Width(s)
}

// SafeTruncate truncates a string to fit within maxWidth display columns
// This properly handles emojis and wide characters
func SafeTruncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	
	// Strip ANSI first
	cleaned := stripANSI(s)
	
	// Calculate truncation point considering display width
	width := 0
	var result strings.Builder
	
	for _, r := range cleaned {
		rw := runewidth.RuneWidth(r)
		if width+rw > maxWidth {
			break
		}
		result.WriteRune(r)
		width += rw
	}
	
	return result.String()
}

// SafeTruncateWithEllipsis truncates and adds "..." if needed
func SafeTruncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 3 {
		if maxWidth <= 0 {
			return ""
		}
		return strings.Repeat(".", maxWidth)
	}
	
	if SafeStringWidth(s) <= maxWidth {
		return s
	}
	
	// Reserve 3 cells for "..."
	truncated := SafeTruncate(s, maxWidth-3)
	return truncated + "..."
}

// SafePadLeft pads string to reach targetWidth on the left
func SafePadLeft(s string, targetWidth int) string {
	currentWidth := SafeStringWidth(s)
	if currentWidth >= targetWidth {
		return s
	}
	
	padding := strings.Repeat(" ", targetWidth-currentWidth)
	return padding + s
}

// SafePadRight pads string to reach targetWidth on the right
func SafePadRight(s string, targetWidth int) string {
	currentWidth := SafeStringWidth(s)
	if currentWidth >= targetWidth {
		return s
	}
	
	padding := strings.Repeat(" ", targetWidth-currentWidth)
	return s + padding
}

// SafeCenter centers string within targetWidth
func SafeCenter(s string, targetWidth int) string {
	currentWidth := SafeStringWidth(s)
	if currentWidth >= targetWidth {
		return s
	}
	
	totalPadding := targetWidth - currentWidth
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding
	
	return strings.Repeat(" ", leftPadding) + s + strings.Repeat(" ", rightPadding)
}

// CalculateTabWidth calculates the display width needed for a tab
// including emoji, text, and notification badge
func CalculateTabWidth(icon, key, name, notifEmoji string, notifCount int) int {
	width := 0
	
	// Icon (emoji) is typically 2 cells
	if icon != "" {
		width += runewidth.StringWidth(icon)
	}
	
	// Space after icon
	if icon != "" && (key != "" || name != "") {
		width += 1 // space
	}
	
	// Key binding
	if key != "" {
		width += runewidth.StringWidth(key)
	}
	
	// Separator between key and name
	if key != "" && name != "" {
		width += 1 // dot or space
	}
	
	// Name
	if name != "" {
		width += runewidth.StringWidth(name)
	}
	
	// Notification badge
	if notifEmoji != "" && notifCount > 0 {
		width += 1 // space before notification
		width += runewidth.StringWidth(notifEmoji) // emoji is 2 cells
		width += 1 // space
		
		// Count digits
		if notifCount > 99 {
			width += 3 // "99+"
		} else if notifCount > 9 {
			width += 2 // two digits
		} else {
			width += 1 // single digit
		}
	}
	
	return width
}

// CalculateAvailableWidth returns usable width after accounting for borders
// For lipgloss borders: subtract 2 (left and right borders)
func CalculateAvailableWidth(totalWidth int, hasBorder bool) int {
	if hasBorder {
		return totalWidth - 2
	}
	return totalWidth
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip the '['
			continue
		}
		
		if inEscape {
			// Skip until we find the end of the escape sequence
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		
		result.WriteByte(s[i])
	}
	
	return result.String()
}

// EmojiWidth returns the display width of common emojis used in tabs
// Most emojis are 2 cells wide in terminals
func EmojiWidth(emoji string) int {
	return runewidth.StringWidth(emoji)
}

// Common emoji widths for quick reference (all are typically 2)
const (
	RunnerEmojiWidth = 2 // 🏃
	PageEmojiWidth   = 2 // 📄
	ErrorEmojiWidth  = 2 // ❌
	LinkEmojiWidth   = 2 // 🔗
	GlobeEmojiWidth  = 2 // 🌐
	RobotEmojiWidth  = 2 // 🤖
	GearEmojiWidth   = 2 // ⚙️
	
	// Notification emojis
	RedCircleWidth    = 2 // 🔴
	YellowCircleWidth = 2 // 🟡
	BlueCircleWidth   = 2 // 🔵
	GreenCircleWidth  = 2 // 🟢
	
	// UI elements
	TriangleWidth = 1 // ▶ (this is actually 1 cell)
	BoxLineWidth  = 1 // │ (box drawing character)
)

// ValidateComponentWidth checks if a component will fit within available space
func ValidateComponentWidth(componentWidth, availableWidth int) error {
	if componentWidth > availableWidth {
		return WidthOverflowError{
			Component: componentWidth,
			Available: availableWidth,
			Overflow:  componentWidth - availableWidth,
		}
	}
	return nil
}

// WidthOverflowError indicates a component is too wide for available space
type WidthOverflowError struct {
	Component int
	Available int
	Overflow  int
}

func (e WidthOverflowError) Error() string {
	return fmt.Sprintf("component width %d exceeds available %d by %d cells", 
		e.Component, e.Available, e.Overflow)
}

// TabWidthCalculator helps calculate responsive tab widths
type TabWidthCalculator struct {
	TerminalWidth int
	TabCount      int
	Mode          string // "minimal", "compact", "normal", "full"
}

// NewTabWidthCalculator creates a calculator for the given terminal width
func NewTabWidthCalculator(terminalWidth, tabCount int) *TabWidthCalculator {
	calc := &TabWidthCalculator{
		TerminalWidth: terminalWidth,
		TabCount:      tabCount,
	}
	calc.determineMode()
	return calc
}

// determineMode sets the appropriate rendering mode based on width
func (c *TabWidthCalculator) determineMode() {
	if c.TerminalWidth < 50 {
		c.Mode = "minimal"
	} else if c.TerminalWidth < 80 {
		c.Mode = "compact"
	} else if c.TerminalWidth < 120 {
		c.Mode = "normal"
	} else {
		c.Mode = "full"
	}
}

// AvailablePerTab returns the average width available per tab
func (c *TabWidthCalculator) AvailablePerTab() int {
	// Account for separators between tabs (3 cells each: " │ ")
	separatorWidth := (c.TabCount - 1) * 3
	
	// Some padding on sides
	padding := 4
	
	available := c.TerminalWidth - separatorWidth - padding
	if available < 0 {
		available = 0
	}
	
	if c.TabCount > 0 {
		return available / c.TabCount
	}
	return 0
}

// CanFitFullTabs checks if full tab names will fit
func (c *TabWidthCalculator) CanFitFullTabs(tabWidths []int) bool {
	total := 0
	for i, width := range tabWidths {
		total += width
		if i < len(tabWidths)-1 {
			total += 3 // separator
		}
	}
	return total <= c.TerminalWidth-4 // with some padding
}