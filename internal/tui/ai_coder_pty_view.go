package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	isFullScreen   bool
	terminalFocused bool
	showHelp       bool
	statusMessage  string
	statusTime     time.Time
	
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
		ptyManager:     ptyManager,
		isFullScreen:   false,
		terminalFocused: false,
		showHelp:       false,
		keyBindings:    aicoder.GetDefaultKeyBindings(),
	}
	
	view.setupStyles()
	return view
}

// setupStyles initializes the styling for the view
func (v *AICoderPTYView) setupStyles() {
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
		
	case tea.KeyMsg:
		return v.handleKeyPress(msg)
		
	case PTYOutputMsg:
		// Terminal output received - trigger re-render
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
		v.toggleFullScreen()
		return v, nil
		
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+h"))):
		v.showHelp = !v.showHelp
		return v, nil
		
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+n"))):
		// Next session
		if session, err := v.ptyManager.NextSession(); err == nil {
			v.currentSession = session
		}
		return v, nil
		
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+p"))):
		// Previous session  
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
				return v, v.showDataInjectionFeedback(binding.Description)
			}
			return v, nil
		}
	}
	
	// If terminal is focused, send input to PTY
	if v.terminalFocused && v.currentSession != nil {
		// Check for ESC key to unfocus terminal
		if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
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
			v.toggleFullScreen()
		} else {
			v.terminalFocused = false
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
func (v *AICoderPTYView) toggleFullScreen() {
	v.isFullScreen = !v.isFullScreen
	
	if v.currentSession != nil {
		v.currentSession.SetFullScreen(v.isFullScreen)
		// Resize terminal to match new dimensions
		termWidth, termHeight := v.getTerminalSize()
		v.currentSession.Resize(termWidth, termHeight)
	}
}

// getTerminalSize calculates the available terminal size
func (v *AICoderPTYView) getTerminalSize() (int, int) {
	// Ensure we have valid dimensions
	if v.width <= 0 || v.height <= 0 {
		// Return sensible defaults if dimensions not set yet
		return 80, 24
	}
	
	if v.isFullScreen {
		// Full screen: use almost all available space
		width := v.width - 4
		height := v.height - 4
		// Ensure positive values
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}
		return width, height
	} else {
		// Windowed: use portion of available space
		width := v.width - 6
		height := v.height - 8
		// Ensure positive values
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
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
		content.WriteString(v.fullScreenStyle.Width(v.width-2).Height(v.height-4).Render(terminalContent))
	} else {
		noSessionMsg := "No active AI coder session\nPress Ctrl+N to create a new session"
		content.WriteString(v.fullScreenStyle.Width(v.width-2).Height(v.height-4).Render(noSessionMsg))
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
	
	// Terminal content
	if v.currentSession != nil {
		terminalContent := v.renderTerminalContent()
		terminalHeight := v.height - 8 // Account for header, footer, help
		content.WriteString(v.borderStyle.Width(v.width-6).Height(terminalHeight).Render(terminalContent))
	} else {
		noSessionMsg := "No active AI coder session\n\nTo start a new session:\n• Use: /ai <provider> (interactive)\n• Use: /ai <provider> <task> (with task)"
		terminalHeight := v.height - 8
		content.WriteString(v.borderStyle.Width(v.width-6).Height(terminalHeight).Render(noSessionMsg))
	}
	
	content.WriteString("\n")
	
	// Help section
	if v.showHelp {
		content.WriteString(v.renderHelp())
	} else {
		// Brief controls
		controls := "F11: Full Screen | F12: Debug Mode | Ctrl+H: Help | Enter: Focus Terminal | Ctrl+N/P: Switch Session"
		if v.terminalFocused {
			controls = "ESC: Unfocus | " + controls
		}
		content.WriteString(v.helpStyle.Render(controls))
	}
	
	return content.String()
}

// renderTerminalContent renders the actual terminal content
func (v *AICoderPTYView) renderTerminalContent() string {
	if v.currentSession == nil {
		return ""
	}
	
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
	
	// Render each line
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := terminal.Cell(x, y)
			
			// Handle cursor
			if v.terminalFocused && cursorVisible && x == cursor.X && y == cursor.Y {
				content.WriteRune('█') // Block cursor
			} else if cell.Char != 0 {
				// Render the character
				content.WriteRune(cell.Char)
			} else {
				content.WriteRune(' ')
			}
		}
		
		// Add newline except for last line
		if y < height-1 {
			content.WriteString("\n")
		}
	}
	
	return content.String()
}

// renderHelp renders the help section
func (v *AICoderPTYView) renderHelp() string {
	var help strings.Builder
	
	help.WriteString(v.helpStyle.Render("🔥 AI Coder PTY Controls:\n"))
	help.WriteString(v.helpStyle.Render("Navigation: F11 (Full Screen) | F12 (Debug Mode) | Enter (Focus) | ESC (Unfocus/Exit)\n"))
	help.WriteString(v.helpStyle.Render("Sessions: Ctrl+N (Next) | Ctrl+P (Previous) | Ctrl+D (Detach)\n"))
	help.WriteString(v.helpStyle.Render("\n🔹 Data Injection Keys:\n"))
	
	for _, binding := range v.keyBindings {
		help.WriteString(v.helpStyle.Render(fmt.Sprintf("  %s: %s\n", 
			strings.ToUpper(binding.Key), binding.Description)))
	}
	
	help.WriteString(v.helpStyle.Render("\n🚨 Debug Mode (F12):\n"))
	help.WriteString(v.helpStyle.Render("When enabled, automatically forwards errors, test failures, and build failures to AI\n"))
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