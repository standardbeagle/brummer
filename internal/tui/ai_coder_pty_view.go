package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hinshun/vt10x"

	"github.com/standardbeagle/brummer/internal/aicoder"
)

// AICoderPTYView manages the PTY-based AI coder interface
type AICoderPTYView struct {
	width  int
	height int

	// PTY management
	ptyManager     *aicoder.PTYManager
	currentSession *aicoder.PTYSession

	// View state
	isFullScreen    bool
	terminalFocused bool
	showHelp        bool
	statusMessage   string
	statusTime      time.Time

	// Scrollback history
	scrollbackBuffer []string
	scrollOffset     int
	maxScrollback    int
	useRawHistory    bool // Use raw output history with ANSI codes

	// Key bindings
	keyBindings []aicoder.KeyBinding

	// Styling
	borderStyle      lipgloss.Style
	fullScreenStyle  lipgloss.Style
	helpStyle        lipgloss.Style
	sessionInfoStyle lipgloss.Style
}

// PTYOutputMsg represents terminal output
type PTYOutputMsg struct {
	SessionID string
	Data      []byte
}

// PTYEventMsg represents PTY events
type PTYEventMsg struct {
	Event aicoder.PTYEvent
}

// NewAICoderPTYView creates a new AI coder PTY view
func NewAICoderPTYView(ptyManager *aicoder.PTYManager) *AICoderPTYView {
	view := &AICoderPTYView{
		ptyManager:       ptyManager,
		isFullScreen:     false,
		terminalFocused:  false,
		showHelp:         false,
		keyBindings:      aicoder.GetDefaultKeyBindings(),
		scrollbackBuffer: make([]string, 0),
		scrollOffset:     0,
		maxScrollback:    10000, // Keep 10k lines of history
		useRawHistory:    true,  // Use raw output with ANSI codes
	}

	view.setupStyles()
	return view
}

// setupStyles initializes the styling for the view
func (v *AICoderPTYView) setupStyles() {
	// Border style that doesn't override terminal colors
	v.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	v.fullScreenStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1)

	v.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)

	v.sessionInfoStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		Bold(true)
}

// Update handles messages for the PTY view
func (v *AICoderPTYView) Update(msg tea.Msg) (*AICoderPTYView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height

		// Resize current session if exists
		if v.currentSession != nil {
			termWidth, termHeight := v.getTerminalSize()
			v.currentSession.Resize(termWidth, termHeight)
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if v.currentSession != nil {
			switch msg.Type {
			case tea.MouseWheelUp:
				// Scroll up
				v.scrollOffset += 3 // Scroll 3 lines at a time
				// Limit scroll to available history
				if v.currentSession != nil {
					history := v.currentSession.GetOutputHistory()
					if len(history) > 0 {
						historyLines := len(strings.Split(string(history), "\n"))
						_, termHeight := v.getTerminalSize()
						maxScroll := historyLines - termHeight
						if maxScroll < 0 {
							maxScroll = 0
						}
						if v.scrollOffset > maxScroll {
							v.scrollOffset = maxScroll
						}
					}
				}
				return v, nil

			case tea.MouseWheelDown:
				// Scroll down
				v.scrollOffset -= 3
				if v.scrollOffset < 0 {
					v.scrollOffset = 0
				}
				return v, nil
			}
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyPress(msg)

	case PTYOutputMsg:
		// Terminal output received - add to scrollback buffer
		if v.currentSession != nil && msg.SessionID == v.currentSession.ID {
			v.addToScrollback(string(msg.Data))
			// Auto-scroll to bottom when new content arrives (unless user is actively scrolling)
			if v.scrollOffset > 0 && v.scrollOffset < 10 {
				v.scrollOffset = 0
			}
		}
		return v, nil

	case PTYEventMsg:
		return v.handlePTYEvent(msg.Event)
	}

	return v, nil
}

// handleKeyPress handles keyboard input
func (v *AICoderPTYView) handleKeyPress(msg tea.KeyMsg) (*AICoderPTYView, tea.Cmd) {
	// Handle global key bindings first
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("f11"))):
		cmd := v.toggleFullScreen()
		return v, cmd

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+h"))):
		v.showHelp = !v.showHelp
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+n"))):
		// Next session
		if session, err := v.ptyManager.NextSession(); err == nil {
			v.currentSession = session
		}
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+shift+p"))):
		// Previous session (using shift to avoid conflict with data injection)
		if session, err := v.ptyManager.PreviousSession(); err == nil {
			v.currentSession = session
		}
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
		// Detach from current session (but keep it running)
		v.currentSession = nil
		v.terminalFocused = false
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("f12"))):
		// Toggle debug mode for auto event forwarding
		if v.currentSession != nil {
			newState := !v.currentSession.IsDebugModeEnabled()
			v.currentSession.SetDebugMode(newState)

			// Show status message
			status := "disabled"
			if newState {
				status = "enabled"
			}
			v.statusMessage = fmt.Sprintf("[DEBUG MODE %s - Auto-forwarding Brummer events]", strings.ToUpper(status))
			v.statusTime = time.Now()
		}
		return v, nil
	}

	// Handle data injection key bindings
	for _, binding := range v.keyBindings {
		if key.Matches(msg, key.NewBinding(key.WithKeys(binding.Key))) {
			if err := v.ptyManager.InjectDataToCurrent(binding.DataType); err == nil {
				// Show brief feedback
				v.statusMessage = fmt.Sprintf("✅ Injected: %s", binding.Description)
				v.statusTime = time.Now()
				return v, v.showDataInjectionFeedback(binding.Description)
			} else if err != nil {
				// Show error feedback
				v.statusMessage = fmt.Sprintf("❌ Failed to inject data: %v", err)
				v.statusTime = time.Now()
			}
			return v, nil
		}
	}

	// If terminal is focused, send input to PTY
	if v.terminalFocused && v.currentSession != nil {
		// Check for Ctrl+Q to unfocus terminal (ESC is used by AI agents)
		if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+q"))) {
			v.terminalFocused = false
			return v, nil
		}

		// Convert key message to bytes and send to PTY
		input := v.keyMsgToBytes(msg)
		if len(input) > 0 {
			if err := v.currentSession.WriteInput(input); err != nil {
				// Session is closed or input buffer is full - silently handle
				v.terminalFocused = false
				return v, nil
			}
		}
		return v, nil
	}

	// Handle view-level commands when terminal is not focused
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Focus on terminal
		v.terminalFocused = true
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		// Unfocus terminal or exit full screen
		if v.isFullScreen {
			cmd := v.toggleFullScreen()
			return v, cmd
		} else {
			v.terminalFocused = false
		}
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
		// Scroll up
		if v.currentSession != nil {
			_, termHeight := v.getTerminalSize()
			v.scrollOffset += termHeight / 2
			// Limit scroll to available history
			history := v.currentSession.GetOutputHistory()
			if len(history) > 0 {
				historyLines := len(strings.Split(string(history), "\n"))
				maxScroll := historyLines - termHeight
				if maxScroll < 0 {
					maxScroll = 0
				}
				if v.scrollOffset > maxScroll {
					v.scrollOffset = maxScroll
				}
			}
		}
		return v, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
		// Scroll down
		if v.currentSession != nil {
			_, termHeight := v.getTerminalSize()
			v.scrollOffset -= termHeight / 2
			if v.scrollOffset < 0 {
				v.scrollOffset = 0
			}
		}
		return v, nil
	}

	return v, nil
}

// handlePTYEvent handles PTY-specific events
func (v *AICoderPTYView) handlePTYEvent(event aicoder.PTYEvent) (*AICoderPTYView, tea.Cmd) {
	switch event.Type {
	case aicoder.PTYEventClose:
		// Session closed, clear current session if it was this one
		if v.currentSession != nil && v.currentSession.ID == event.SessionID {
			v.currentSession = nil
			v.terminalFocused = false
		}
	}

	return v, nil
}

// toggleFullScreen toggles full-screen mode
func (v *AICoderPTYView) toggleFullScreen() tea.Cmd {
	v.isFullScreen = !v.isFullScreen

	if v.currentSession != nil {
		v.currentSession.SetFullScreen(v.isFullScreen)
		// Resize terminal to match new dimensions
		termWidth, termHeight := v.getTerminalSize()
		v.currentSession.Resize(termWidth, termHeight)
	}

	// Return command to enter/exit alternate screen
	if v.isFullScreen {
		return tea.EnterAltScreen
	}
	return tea.ExitAltScreen
}

// addToScrollback adds output to the scrollback buffer
func (v *AICoderPTYView) addToScrollback(data string) {
	// Split data into lines and add to buffer
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if line != "" || len(lines) > 1 {
			v.scrollbackBuffer = append(v.scrollbackBuffer, line)

			// Trim buffer if it exceeds max size
			if len(v.scrollbackBuffer) > v.maxScrollback {
				v.scrollbackBuffer = v.scrollbackBuffer[len(v.scrollbackBuffer)-v.maxScrollback:]
			}
		}
	}

	// Auto-scroll to bottom when new content arrives
	v.scrollOffset = 0
}

// getTerminalSize calculates the available terminal size based on view mode and constraints
//
// LAYOUT CONSTRAINTS DOCUMENTATION:
// This function is critical for proper terminal rendering and must account for all UI elements
// that consume screen space. The calculations ensure the PTY content fits within borders.
//
// CONSTRAINT BREAKDOWN:
// ┌─────────────────────────────────────────────────────────────────────────┐
// │                           WINDOWED MODE LAYOUT                         │
// ├─────────────────────────────────────────────────────────────────────────┤
// │ Session Info Header    (1 line)                                        │
// │ Status Message        (0-1 lines, conditional, 3s timeout)             │
// │ Blank Line            (1 line)                                         │
// │ ╭─────────────────────────────────────────────────────────────────────╮ │
// │ │ Terminal Content Area (calculated height)                          │ │
// │ │ ← Border (1 char) + Padding (1 char) = 2 chars per side           │ │
// │ │ ← Total horizontal deduction: 4 characters                         │ │
// │ ╰─────────────────────────────────────────────────────────────────────╯ │
// │ Controls/Help Footer  (2-10 lines, depends on showHelp)               │
// └─────────────────────────────────────────────────────────────────────────┘
//
// FULL SCREEN MODE LAYOUT:
// ┌─────────────────────────────────────────────────────────────────────────┐
// │ Session Info (Full Screen) (1 line)                                    │
// │ ┌─────────────────────────────────────────────────────────────────────┐ │
// │ │ Raw PTY Content (entire remaining space)                           │ │
// │ │ ← Border: 2 chars (left + right)                                   │ │
// │ │ ← Padding: 2 chars (left + right)                                  │ │
// │ │ ← Total deduction: 4 characters                                    │ │
// │ └─────────────────────────────────────────────────────────────────────┘ │
// │ Footer Controls       (1 line)                                         │
// └─────────────────────────────────────────────────────────────────────────┘
//
// CRITICAL CONSISTENCY REQUIREMENT:
// This function MUST return dimensions that match the rendering calculations in:
// - renderWindowed(): terminalWidth := v.width - 4
// - renderFullScreen(): borderedContent width calculation
// - renderTerminalWithBorder(): width parameter usage
//
// The "-4" deduction is used consistently across all rendering functions.
func (v *AICoderPTYView) getTerminalSize() (int, int) {
	// Ensure we have valid dimensions before any calculations
	if v.width <= 0 || v.height <= 0 {
		// Return sensible defaults if dimensions not set yet
		// These match common terminal defaults
		return 80, 24
	}

	// WINDOWED MODE HEADER/FOOTER CALCULATIONS:
	// These must match the actual UI elements rendered in renderWindowed()
	headerLines := 3 // Base: Session info (1) + blank line (1) + content start (1)
	if v.statusMessage != "" && time.Since(v.statusTime) < 3*time.Second {
		headerLines += 2 // Status message (1) + blank line (1)
	}
	footerLines := 2 // Base: blank line (1) + controls (1)
	if v.showHelp {
		footerLines = 10 // Extended help text takes ~8-10 lines
	}

	// CORE CONSTRAINT: BORDER + PADDING DEDUCTION
	// This is the most critical calculation that must remain consistent:
	//
	// Border rendering structure (see renderTerminalWithBorder):
	// "╭─────────╮"  ← Top border
	// "│ content │"  ← Left border (1) + left padding (1) + content + right padding (1) + right border (1)
	// "╰─────────╯"  ← Bottom border
	//
	// Total horizontal space consumed by border + padding:
	// - Left border: 1 character
	// - Left padding: 1 character
	// - Right padding: 1 character
	// - Right border: 1 character
	// - TOTAL: 4 characters
	//
	// This 4-character deduction is used in:
	// 1. This function (getTerminalSize)
	// 2. renderWindowed() terminal width calculation
	// 3. PTY session resize operations
	// 4. Border rendering width calculations
	const BORDER_AND_PADDING_WIDTH = 4

	if v.isFullScreen {
		// FULL SCREEN MODE:
		// Use maximum available space minus border/padding constraints
		// Height constraint is minimal (header + footer + border)
		width := v.width - BORDER_AND_PADDING_WIDTH
		height := v.height - 4 // Header (1) + footer (1) + top/bottom borders (2)

		// Safety bounds to prevent degenerate terminals
		if width <= 0 {
			width = 80 // Standard terminal width fallback
		}
		if height <= 0 {
			height = 24 // Standard terminal height fallback
		}
		return width, height
	} else {
		// WINDOWED MODE:
		// Account for all UI elements that consume vertical space
		width := v.width - BORDER_AND_PADDING_WIDTH - 2 // Additional 2 columns to fit properly in window
		height := v.height - headerLines - footerLines

		// Safety bounds to prevent degenerate terminals
		if width <= 0 {
			width = 80 // Standard terminal width fallback
		}
		if height <= 0 {
			height = 24 // Standard terminal height fallback
		}
		return width, height
	}
}

// keyMsgToBytes converts a tea.KeyMsg to bytes for PTY input
func (v *AICoderPTYView) keyMsgToBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyRunes:
		return []byte(msg.String())
	case tea.KeySpace:
		return []byte(" ")
	case tea.KeyEnter:
		return []byte("\r")
	case tea.KeyBackspace:
		return []byte("\b")
	case tea.KeyTab:
		return []byte("\t")
	case tea.KeyEsc:
		return []byte("\x1b")
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	case tea.KeyHome:
		return []byte("\x1b[H")
	case tea.KeyEnd:
		return []byte("\x1b[F")
	case tea.KeyPgUp:
		return []byte("\x1b[5~")
	case tea.KeyPgDown:
		return []byte("\x1b[6~")
	case tea.KeyDelete:
		return []byte("\x1b[3~")
	case tea.KeyCtrlC:
		return []byte("\x03")
	case tea.KeyCtrlD:
		return []byte("\x04")
	case tea.KeyCtrlZ:
		return []byte("\x1a")
	default:
		return []byte{}
	}
}

// showDataInjectionFeedback shows brief feedback for data injection
func (v *AICoderPTYView) showDataInjectionFeedback(description string) tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return struct{}{} // Clear feedback after 2 seconds
	})
}

// View renders the AI coder PTY view
func (v *AICoderPTYView) View() string {
	// Ensure we have valid dimensions before rendering
	if v.width <= 0 || v.height <= 0 {
		return fmt.Sprintf("Initializing AI Coder view... (dimensions: %dx%d)", v.width, v.height)
	}

	// Add debug info about current session
	if v.currentSession == nil {
		// Try to get current session from PTY manager
		if v.ptyManager != nil {
			if currentSession, exists := v.ptyManager.GetCurrentSession(); exists {
				v.currentSession = currentSession
				// Resize to current dimensions
				termWidth, termHeight := v.getTerminalSize()
				v.currentSession.Resize(termWidth, termHeight)
			}
		}

		// If still no session, show appropriate message
		if v.currentSession == nil {
			// Check if there are any sessions available
			sessions := []string{}
			hasPTYManager := v.ptyManager != nil
			sessionCount := 0
			if v.ptyManager != nil {
				allSessions := v.ptyManager.ListSessions()
				sessionCount = len(allSessions)
				for _, s := range allSessions {
					sessions = append(sessions, fmt.Sprintf("%s (Active: %v, Terminal: %v)", s.ID, s.IsActive, s.Terminal != nil))
				}
			}

			debugInfo := fmt.Sprintf("PTY Manager: %v, Session Count: %d", hasPTYManager, sessionCount)

			if len(sessions) > 0 {
				return fmt.Sprintf("No AI Coder session selected (dimensions: %dx%d)\n%s\nAvailable sessions: %v\n\nUse /ai <provider> to start a session", v.width, v.height, debugInfo, sessions)
			}
			return fmt.Sprintf("No AI Coder session active (dimensions: %dx%d)\n%s\n\nUse /ai <provider> to start a session", v.width, v.height, debugInfo)
		}
	}

	if v.isFullScreen {
		return v.renderFullScreen()
	} else {
		return v.renderWindowed()
	}
}

// renderFullScreen renders the full-screen terminal view
func (v *AICoderPTYView) renderFullScreen() string {
	var content strings.Builder

	// Full screen header
	header := v.sessionInfoStyle.Render(v.getSessionInfo() + " (Full Screen)")
	content.WriteString(header)
	content.WriteString("\n")

	// Terminal content
	if v.currentSession != nil {
		terminalContent := v.renderTerminalContent()
		// FULL SCREEN BORDER CONSTRAINT:
		// Width: v.width - 4 (BORDER_AND_PADDING_WIDTH from getTerminalSize)
		// Height: v.height - 4 (header + footer + top/bottom borders)
		// This MUST match the getTerminalSize() full screen calculation
		borderedContent := v.renderTerminalWithFullScreenBorder(terminalContent, v.width-4, v.height-4)
		content.WriteString(borderedContent)
	} else {
		noSessionMsg := "No active AI coder session\nPress Ctrl+N to create a new session"
		fullScreenStyle := v.fullScreenStyle.
			Width(v.width - 4).
			Height(v.height - 4).
			MaxWidth(v.width - 4).
			MaxHeight(v.height - 4)
		content.WriteString(fullScreenStyle.Render(noSessionMsg))
	}

	// Full screen footer
	footer := v.helpStyle.Render("ESC: Exit Full Screen | Ctrl+C: Interrupt | Ctrl+D: Detach")
	content.WriteString("\n")
	content.WriteString(footer)

	return content.String()
}

// renderWindowed renders the windowed terminal view
func (v *AICoderPTYView) renderWindowed() string {
	var content strings.Builder

	// Session info header
	sessionInfo := v.sessionInfoStyle.Render(v.getSessionInfo())
	content.WriteString(sessionInfo)
	content.WriteString("\n")

	// Status message if present
	if v.statusMessage != "" && time.Since(v.statusTime) < 3*time.Second {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Bold(true)
		content.WriteString(statusStyle.Render(v.statusMessage))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Terminal content - use more available space
	if v.currentSession != nil {
		terminalContent := v.renderTerminalContent()
		// Calculate height more precisely
		headerLines := 3 // Session info + blank line
		if v.statusMessage != "" && time.Since(v.statusTime) < 3*time.Second {
			headerLines += 2 // Status message + blank
		}
		footerLines := 2 // Controls line + padding
		if v.showHelp {
			footerLines = 10 // Full help text
		}
		terminalHeight := v.height - headerLines - footerLines
		if terminalHeight < 5 {
			terminalHeight = 5
		}
		// CONSISTENCY REQUIREMENT: This width calculation MUST match getTerminalSize()
		// Both functions use the same BORDER_AND_PADDING_WIDTH = 4 constraint
		// This ensures the rendered content fits exactly within the calculated terminal size
		terminalWidth := v.width - 4 // Match getTerminalSize() BORDER_AND_PADDING_WIDTH
		if terminalWidth < 20 {
			terminalWidth = 20
		}

		// Render terminal content with custom border to preserve ANSI codes
		borderedContent := v.renderTerminalWithBorder(terminalContent, terminalWidth, terminalHeight)
		content.WriteString(borderedContent)
	} else {
		noSessionMsg := "No active AI coder session\n\nTo start a new session:\n• Use: /ai <provider> (interactive)\n• Use: /ai <provider> <task> (with task)"
		terminalHeight := v.height - 6
		if terminalHeight < 5 {
			terminalHeight = 5
		}
		terminalWidth := v.width - 6
		if terminalWidth < 20 {
			terminalWidth = 20
		}

		// Also apply MaxWidth/MaxHeight here
		containerStyle := v.borderStyle.
			Width(terminalWidth).
			Height(terminalHeight).
			MaxWidth(terminalWidth).
			MaxHeight(terminalHeight)

		content.WriteString(containerStyle.Render(noSessionMsg))
	}

	content.WriteString("\n")

	// Help section
	if v.showHelp {
		content.WriteString(v.renderHelp())
	} else {
		// Brief controls
		controls := "F11: Full Screen | F12: Debug Mode | Ctrl+H: Help"
		if v.terminalFocused {
			controls = "Terminal Focused | Ctrl+Q: Unfocus | Ctrl+E/L/T/B/P/U/R: Inject Data | " + controls
		} else {
			controls = "Terminal Not Focused | Enter: Focus Terminal | / (start of line): Brummer Commands | " + controls
		}
		if v.scrollOffset > 0 {
			controls = fmt.Sprintf("[Scrolled ↑ %d] ", v.scrollOffset) + controls + " | Mouse/PgUp/PgDn: Scroll"
		} else {
			controls = controls + " | Mouse/PgUp/PgDn: Scroll"
		}
		content.WriteString(v.helpStyle.Render(controls))
	}

	return content.String()
}

// GetRawOutput returns the complete raw terminal output for full screen mode
func (v *AICoderPTYView) GetRawOutput() string {
	if v.currentSession == nil {
		return "\033[2J\033[H" + "No active AI coder session\r\nPress Ctrl+N to create a new session"
	}

	// In full screen mode, return the raw PTY output directly
	// This preserves all ANSI codes including cursor positioning, colors, etc.
	history := v.currentSession.GetOutputHistory()
	if len(history) == 0 {
		return "\033[2J\033[HWaiting for output..."
	}

	// Return the raw output as-is
	// The PTY program (like Claude Code) is responsible for screen management
	return string(history)
}

// renderTerminalContent renders the actual terminal content with full color support
//
// This function is the heart of our PTY rendering system. It reads from the vt10x
// terminal emulator buffer and reconstructs ANSI escape sequences to preserve
// colors and text attributes in the BubbleTea TUI.
//
// The key insight is that vt10x has already parsed all the ANSI sequences from
// the PTY output and maintains a cell-based buffer (like a real terminal). Our
// job is to read this buffer and reconstruct the necessary ANSI codes for display.
func (v *AICoderPTYView) renderTerminalContent() string {
	if v.currentSession == nil {
		return ""
	}

	// If we're scrolled up, render from history
	if v.scrollOffset > 0 {
		return v.renderScrolledView()
	}

	// Get the terminal emulator which has already parsed the PTY output
	terminal := v.currentSession.GetTerminal()
	if terminal == nil {
		return fmt.Sprintf("Waiting for terminal... (session: %s, active: %v, has output: %v)",
			v.currentSession.ID, v.currentSession.IsActive, len(v.currentSession.GetOutputHistory()) > 0)
	}

	// Build the visible content from the terminal buffer
	var content strings.Builder

	// Lock terminal while reading to ensure consistency
	terminal.Lock()
	defer terminal.Unlock()

	// Get our available rendering dimensions
	termWidth, termHeight := v.getTerminalSize()

	// Get the actual terminal buffer dimensions
	// The terminal may be larger than our display area
	width, height := terminal.Size()

	// Limit to our available display space
	if height > termHeight {
		height = termHeight
	}
	// Ensure content fits within our calculated display constraints
	// termWidth already accounts for border and padding, so use it directly
	if width > termWidth {
		width = termWidth
	}

	// Track the current text style as we scan across cells
	// This allows us to minimize ANSI code generation by only
	// emitting codes when the style changes
	currentFG := vt10x.DefaultFG
	currentBG := vt10x.DefaultBG
	var currentMode int16
	styleActive := false

	for y := 0; y < height; y++ {
		lineBuffer := strings.Builder{}

		for x := 0; x < width; x++ {
			// Get the cell at this position
			// Each cell contains: character, foreground color, background color, and text attributes
			cell := terminal.Cell(x, y)

			// Check if this cell's style differs from the current style
			// This optimization prevents generating redundant ANSI codes
			if cell.FG != currentFG || cell.BG != currentBG || cell.Mode != currentMode {
				// Reset previous style if one was active
				// This ensures a clean slate before applying new attributes
				if styleActive {
					lineBuffer.WriteString("\033[0m")
					styleActive = false
				}

				// Check if this cell needs any styling
				// DefaultFG/DefaultBG are special values meaning "use terminal default"
				if cell.FG != vt10x.DefaultFG || cell.BG != vt10x.DefaultBG || cell.Mode != 0 {
					// Build ANSI escape sequence codes
					codes := []string{}

					// Text attribute codes (Mode is a bitmask)
					// These must come before color codes in the ANSI sequence
					if cell.Mode&(1<<2) != 0 { // Bold (bit 2)
						codes = append(codes, "1")
					}
					if cell.Mode&(1<<1) != 0 { // Underline (bit 1)
						codes = append(codes, "4")
					}
					if cell.Mode&(1<<0) != 0 { // Reverse video (bit 0)
						codes = append(codes, "7")
					}
					if cell.Mode&(1<<5) != 0 { // Blink (bit 5)
						codes = append(codes, "5")
					}
					if cell.Mode&(1<<4) != 0 { // Italic (bit 4)
						codes = append(codes, "3")
					}

					// Foreground color handling
					// vt10x uses a clever Color encoding scheme:
					// - 0-7: Standard ANSI colors
					// - 8-15: Bright ANSI colors
					// - 16-255: 256-color palette
					// - 256-16777215: 24-bit true color (RGB packed)
					// - 16777216+: Special values (DefaultFG, DefaultBG)
					if cell.FG != vt10x.DefaultFG {
						if cell.FG < 8 {
							// Standard ANSI colors (black, red, green, yellow, blue, magenta, cyan, white)
							// Use codes 30-37
							codes = append(codes, fmt.Sprintf("3%d", cell.FG))
						} else if cell.FG < 16 {
							// Bright ANSI colors
							// Use codes 90-97 (bright black through bright white)
							codes = append(codes, fmt.Sprintf("9%d", cell.FG-8))
						} else if cell.FG < 256 {
							// 256-color palette
							// Use ESC[38;5;{n}m format
							codes = append(codes, fmt.Sprintf("38;5;%d", cell.FG))
						} else if cell.FG < vt10x.DefaultFG {
							// 24-bit true color (RGB)
							// vt10x packs RGB values into a single uint32:
							// Color = (R << 16) | (G << 8) | B
							// We need to extract and use ESC[38;2;{r};{g};{b}m format
							r := (cell.FG >> 16) & 0xFF
							g := (cell.FG >> 8) & 0xFF
							b := cell.FG & 0xFF
							codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
						}
					}

					// Background color handling (same scheme as foreground)
					if cell.BG != vt10x.DefaultBG {
						if cell.BG < 8 {
							// Standard ANSI background colors
							// Use codes 40-47
							codes = append(codes, fmt.Sprintf("4%d", cell.BG))
						} else if cell.BG < 16 {
							// Bright ANSI background colors
							// Use codes 100-107
							codes = append(codes, fmt.Sprintf("10%d", cell.BG-8))
						} else if cell.BG < 256 {
							// 256-color palette background
							// Use ESC[48;5;{n}m format
							codes = append(codes, fmt.Sprintf("48;5;%d", cell.BG))
						} else if cell.BG < vt10x.DefaultBG {
							// 24-bit true color background (RGB)
							// Use ESC[48;2;{r};{g};{b}m format
							r := (cell.BG >> 16) & 0xFF
							g := (cell.BG >> 8) & 0xFF
							b := cell.BG & 0xFF
							codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", r, g, b))
						}
					}

					// Generate the complete ANSI escape sequence
					if len(codes) > 0 {
						// ESC[{code1};{code2};...m format
						ansiCode := fmt.Sprintf("\033[%sm", strings.Join(codes, ";"))
						lineBuffer.WriteString(ansiCode)
						styleActive = true
					}
				}

				// Update our tracking variables
				currentFG = cell.FG
				currentBG = cell.BG
				currentMode = cell.Mode
			}

			// Write the character
			if cell.Char != 0 && cell.Char != '\n' && cell.Char != '\r' {
				lineBuffer.WriteRune(cell.Char)
			} else {
				lineBuffer.WriteRune(' ')
			}
		}

		// Reset style at end of line if needed
		if styleActive {
			lineBuffer.WriteString("\033[0m")
		}

		content.WriteString(lineBuffer.String())
		if y < height-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderScrolledView renders the terminal with scroll offset applied
func (v *AICoderPTYView) renderScrolledView() string {
	if v.currentSession == nil {
		return ""
	}

	// Get the output history
	history := v.currentSession.GetOutputHistory()
	if len(history) == 0 {
		return "No history available"
	}

	// Convert history to string and split into lines
	historyStr := string(history)
	allLines := strings.Split(historyStr, "\n")

	// Get terminal dimensions
	_, height := v.getTerminalSize()

	// Calculate which lines to show
	// We want to show 'height' lines, ending 'scrollOffset' lines from the bottom
	totalLines := len(allLines)
	endLine := totalLines - v.scrollOffset
	if endLine < 0 {
		endLine = 0
	}
	if endLine > totalLines {
		endLine = totalLines
	}

	startLine := endLine - height
	if startLine < 0 {
		startLine = 0
	}

	// Extract the visible lines
	var visibleLines []string
	if startLine < endLine && endLine <= totalLines {
		visibleLines = allLines[startLine:endLine]
	}

	// Join and return
	result := strings.Join(visibleLines, "\n")

	// Add scroll indicator
	if v.scrollOffset > 0 {
		scrollInfo := fmt.Sprintf(" [Scrolled up %d lines] ", v.scrollOffset)
		// Prepend to first line if there is one
		lines := strings.Split(result, "\n")
		if len(lines) > 0 {
			lines[0] = scrollInfo + lines[0]
			result = strings.Join(lines, "\n")
		} else {
			result = scrollInfo
		}
	}

	return result
}

// renderFromHistory renders content from the raw output history
func (v *AICoderPTYView) renderFromHistory() string {
	// For now, disable history scrolling as it's causing rendering issues
	// Just render the current terminal state
	terminal := v.currentSession.GetTerminal()
	if terminal == nil {
		return ""
	}

	var content strings.Builder

	// Lock terminal state while reading
	terminal.Lock()
	defer terminal.Unlock()

	// Get terminal dimensions
	width, height := terminal.Size()
	cursor := terminal.Cursor()
	cursorVisible := terminal.CursorVisible()

	// Render each line with color support
	for y := 0; y < height; y++ {
		lineBuffer := strings.Builder{}

		for x := 0; x < width; x++ {
			cell := terminal.Cell(x, y)

			// Skip style handling - we use raw output now

			// Handle cursor
			if v.terminalFocused && cursorVisible && x == cursor.X && y == cursor.Y {
				lineBuffer.WriteRune('█') // Block cursor
			} else if cell.Char != 0 {
				// Render the character
				lineBuffer.WriteRune(cell.Char)
			} else {
				lineBuffer.WriteRune(' ')
			}
		}

		// Reset at end of line
		lineBuffer.WriteString("\033[0m")

		content.WriteString(lineBuffer.String())

		// Add newline except for last line
		if y < height-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderTerminalWithBorder renders terminal content with a border while preserving ANSI codes
//
// TERMINAL SIZE ↔ BORDER RENDERING RELATIONSHIP:
// This function is the bridge between getTerminalSize() calculations and actual visual output.
// The relationship is critical for preventing content overflow and ensuring proper layout.
//
// CONSTRAINT FLOW:
// 1. getTerminalSize() calculates: terminalWidth = viewWidth - 4
// 2. PTY session is resized to these dimensions
// 3. renderTerminalContent() generates content fitting those dimensions
// 4. This function renders borders around content using the SAME width parameter
// 5. Final output fits exactly within the original view boundaries
//
// PARAMETER RELATIONSHIP:
// - width parameter: Should equal getTerminalSize() width return value
// - height parameter: Should equal getTerminalSize() height return value
// - Content: Must fit within (width-4) x (height-2) after accounting for borders
//
// BORDER STRUCTURE AND SPACE CONSUMPTION:
// ┌────────────────────────────────────────────────────┐
// │ ╭──────────────────────────────────────────────╮ │ ← width param
// │ │ content area (width-4 chars wide)                  │ │
// │ │ ^ left border (1) + left pad (1)                   │ │
// │ │   right pad (1) + right border (1) ^               │ │
// │ ╰──────────────────────────────────────────────╯ │
// └────────────────────────────────────────────────────┘
//
// CRITICAL INSIGHT: Using lipgloss.Style.Render() on content that contains ANSI codes
// will strip those codes! This is why we must use raw ANSI codes for the border itself
// and carefully preserve the content's ANSI codes.
//
// This function:
// 1. Draws a rounded border using Unicode box-drawing characters
// 2. Applies color to the border using raw ANSI codes (not lipgloss)
// 3. Preserves all ANSI codes in the content
// 4. Handles line wrapping and padding correctly
// 5. Ensures content fits exactly within calculated dimensions
func (v *AICoderPTYView) renderTerminalWithBorder(content string, width, height int) string {
	// Use rounded border characters for a modern look
	topLeft := "╭"
	topRight := "╮"
	bottomLeft := "╰"
	bottomRight := "╯"
	horizontal := "─"
	vertical := "│"

	// IMPORTANT: We use raw ANSI codes for border coloring
	// Using lipgloss.Style.Render() would strip ANSI codes from our content!
	borderColorCode := "\033[38;5;62m" // Color 62 (a nice blue-gray)
	resetCode := "\033[0m"

	var result strings.Builder

	// Top border
	result.WriteString(borderColorCode)
	result.WriteString(topLeft)
	for i := 0; i < width-2; i++ {
		result.WriteString(horizontal)
	}
	result.WriteString(topRight)
	result.WriteString(resetCode)
	result.WriteString("\n")

	// Content lines with side borders and padding
	lines := strings.Split(content, "\n")
	for i := 0; i < height-2; i++ {
		// Left border and padding
		result.WriteString(borderColorCode)
		result.WriteString(vertical)
		result.WriteString(resetCode)
		result.WriteString(" ") // left padding

		// Content line (preserving ANSI codes)
		if i < len(lines) {
			// The line already has ANSI codes, just write it
			result.WriteString(lines[i])

			// Calculate visible length (excluding ANSI codes) for right padding
			visibleLen := ansiLength(lines[i])
			if visibleLen < width-4 { // -4 for borders and padding
				// Add right padding
				for j := visibleLen; j < width-4; j++ {
					result.WriteString(" ")
				}
			}
		} else {
			// Empty line - fill with spaces
			for j := 0; j < width-4; j++ {
				result.WriteString(" ")
			}
		}

		// Right padding and border
		result.WriteString(" ") // right padding
		result.WriteString(borderColorCode)
		result.WriteString(vertical)
		result.WriteString(resetCode)
		result.WriteString("\n")
	}

	// Bottom border
	result.WriteString(borderColorCode)
	result.WriteString(bottomLeft)
	for i := 0; i < width-2; i++ {
		result.WriteString(horizontal)
	}
	result.WriteString(bottomRight)
	result.WriteString(resetCode)

	return result.String()
}

// ansiLength calculates the visible length of a string, excluding ANSI escape sequences
//
// BORDER RENDERING DEPENDENCY:
// This function is essential for the getTerminalSize() ↔ border rendering relationship.
// Without accurate visible length calculation, borders would be misaligned and content
// would overflow or have incorrect padding.
//
// WHY THIS MATTERS FOR CONSTRAINTS:
// 1. getTerminalSize() calculates: content should fit in (width-4) characters
// 2. PTY generates content with ANSI codes: "\033[31mHello\033[0m" (5 visible chars, 13 total bytes)
// 3. Border rendering needs visible length (5) to calculate correct padding
// 4. Without this, padding calculation would use total length (13) and overflow borders
//
// ANSI SEQUENCE EXAMPLES:
// - "\033[31mRed text\033[0m" → visible length: 8 ("Red text")
// - "\033[1;32mBold green\033[0m" → visible length: 10 ("Bold green")
// - "Hello \033[4munderlined\033[0m world" → visible length: 21 ("Hello underlined world")
//
// REGEX PATTERN BREAKDOWN:
// - \x1b (ESC character, same as \033)
// - \[ (literal bracket)
// - [0-9;]* (any sequence of digits and semicolons)
// - m (the terminating 'm')
//
// This covers all SGR (Select Graphic Rendition) sequences used for colors and text attributes.
// This is the most common type of ANSI sequence in terminal content.
func ansiLength(s string) int {
	// Remove all ANSI SGR sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cleaned := ansiRegex.ReplaceAllString(s, "")

	// Use rune length instead of byte length to properly handle Unicode
	// This ensures emojis and other multi-byte characters count as single visible characters
	return len([]rune(cleaned))
}

// renderTerminalWithFullScreenBorder renders terminal content with a full screen border
func (v *AICoderPTYView) renderTerminalWithFullScreenBorder(content string, width, height int) string {
	// Use normal border characters for full screen
	topLeft := "┌"
	topRight := "┐"
	bottomLeft := "└"
	bottomRight := "┘"
	horizontal := "─"
	vertical := "│"

	// Border characters with color (red for full screen)
	borderColorCode := "\033[38;5;196m" // Color 196 (red)
	resetCode := "\033[0m"

	var result strings.Builder

	// Top border
	result.WriteString(borderColorCode)
	result.WriteString(topLeft)
	for i := 0; i < width-2; i++ {
		result.WriteString(horizontal)
	}
	result.WriteString(topRight)
	result.WriteString(resetCode)
	result.WriteString("\n")

	// Content lines with side borders and padding
	lines := strings.Split(content, "\n")
	for i := 0; i < height-2; i++ {
		// Left border and padding
		result.WriteString(borderColorCode)
		result.WriteString(vertical)
		result.WriteString(resetCode)
		result.WriteString(" ") // left padding

		// Content line (preserving ANSI codes)
		if i < len(lines) {
			// The line already has ANSI codes, just write it
			result.WriteString(lines[i])

			// Calculate visible length (excluding ANSI codes) for right padding
			visibleLen := ansiLength(lines[i])
			if visibleLen < width-4 { // -4 for borders and padding
				// Add right padding
				for j := visibleLen; j < width-4; j++ {
					result.WriteString(" ")
				}
			}
		} else {
			// Empty line - fill with spaces
			for j := 0; j < width-4; j++ {
				result.WriteString(" ")
			}
		}

		// Right padding and border
		result.WriteString(" ") // right padding
		result.WriteString(borderColorCode)
		result.WriteString(vertical)
		result.WriteString(resetCode)
		result.WriteString("\n")
	}

	// Bottom border
	result.WriteString(borderColorCode)
	result.WriteString(bottomLeft)
	for i := 0; i < width-2; i++ {
		result.WriteString(horizontal)
	}
	result.WriteString(bottomRight)
	result.WriteString(resetCode)

	return result.String()
}

// renderHelp renders the help section
func (v *AICoderPTYView) renderHelp() string {
	var help strings.Builder

	help.WriteString(v.helpStyle.Render("🔥 AI Coder PTY Controls:\n"))
	help.WriteString(v.helpStyle.Render("Navigation: F11 (Full Screen) | F12 (Debug Mode) | Enter (Focus) | Ctrl+Q (Unfocus) | ESC (Exit Full Screen)\n"))
	help.WriteString(v.helpStyle.Render("Sessions: Ctrl+N (Next) | Ctrl+Shift+P (Previous) | Ctrl+D (Detach)\n"))
	help.WriteString(v.helpStyle.Render("\n🔹 Data Injection Keys:\n"))

	for _, binding := range v.keyBindings {
		help.WriteString(v.helpStyle.Render(fmt.Sprintf("  %s: %s\n",
			strings.ToUpper(binding.Key), binding.Description)))
	}

	help.WriteString(v.helpStyle.Render("\n🚨 Debug Mode (F12):\n"))
	help.WriteString(v.helpStyle.Render("When enabled, automatically forwards errors, test failures, and build failures to AI\n"))
	help.WriteString(v.helpStyle.Render("\n💡 Slash Commands:\n"))
	help.WriteString(v.helpStyle.Render("/ at start of line: Opens Brummer command palette\n"))
	help.WriteString(v.helpStyle.Render("/ mid-line: Sent to AI coder as regular input\n"))
	help.WriteString(v.helpStyle.Render("\nCtrl+H: Toggle this help"))

	return help.String()
}

// getSessionInfo returns information about the current session
func (v *AICoderPTYView) getSessionInfo() string {
	sessions := v.ptyManager.ListSessions()
	sessionCount := len(sessions)

	if v.currentSession == nil {
		if sessionCount == 0 {
			return "No AI Coder Sessions"
		} else {
			return fmt.Sprintf("AI Coder Sessions (%d) - None Selected", sessionCount)
		}
	}

	// Find current session index
	currentIndex := -1
	for i, session := range sessions {
		if session.ID == v.currentSession.ID {
			currentIndex = i + 1 // 1-based indexing for display
			break
		}
	}

	status := "Running"
	if !v.currentSession.IsActive {
		status = "Stopped"
	}

	debugMode := ""
	if v.currentSession.IsDebugModeEnabled() {
		debugMode = " [DEBUG]"
	}

	return fmt.Sprintf("Session %d/%d: %s (%s)%s",
		currentIndex, sessionCount, v.currentSession.Name, status, debugMode)
}

// SetCurrentSession sets the current session for the view
func (v *AICoderPTYView) SetCurrentSession(session *aicoder.PTYSession) {
	v.currentSession = session
	if session != nil {
		// Resize to current dimensions
		termWidth, termHeight := v.getTerminalSize()
		session.Resize(termWidth, termHeight)

		// Set status message to confirm session is set
		v.statusMessage = fmt.Sprintf("✅ AI Coder session started: %s", session.Name)
		v.statusTime = time.Now()

		// Auto-focus the terminal when a new session is set
		v.terminalFocused = true
	}
}

// IsTerminalFocused returns whether the terminal is currently focused for input
func (v *AICoderPTYView) IsTerminalFocused() bool {
	return v.terminalFocused
}

// UnfocusTerminal removes focus from the terminal
func (v *AICoderPTYView) UnfocusTerminal() {
	v.terminalFocused = false
}

// ShouldInterceptSlashCommand determines if "/" should open Brummer command palette
// Returns true if:
// - Terminal is not focused, OR
// - Cursor is at start of line (typically indicates new command)
func (v *AICoderPTYView) ShouldInterceptSlashCommand() bool {
	// If terminal not focused, always allow Brummer commands
	if !v.terminalFocused {
		return true
	}

	// If no current session, allow Brummer commands
	if v.currentSession == nil {
		return true
	}

	// When terminal is focused, slash commands should go to the AI agent
	// This allows AI agents to handle "/" as regular input
	return false
}

// GetCurrentLineContent returns the current line content for context
func (v *AICoderPTYView) GetCurrentLineContent() string {
	if v.currentSession == nil {
		return ""
	}
	return v.currentSession.GetCurrentLineContent()
}

// AttachToSession attaches the view to a specific session
func (v *AICoderPTYView) AttachToSession(sessionID string) error {
	if err := v.ptyManager.SetCurrentSession(sessionID); err != nil {
		return err
	}

	if session, exists := v.ptyManager.GetSession(sessionID); exists {
		v.SetCurrentSession(session)
		v.terminalFocused = true
		return nil
	}

	return fmt.Errorf("session not found")
}

// CreateSession creates a new PTY session
func (v *AICoderPTYView) CreateSession(name, command string, args []string) (*aicoder.PTYSession, error) {
	session, err := v.ptyManager.CreateSession(name, command, args)
	if err != nil {
		return nil, err
	}

	// Auto-attach to the new session
	v.SetCurrentSession(session)
	v.terminalFocused = true

	return session, nil
}
