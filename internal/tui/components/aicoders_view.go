package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
)

// ViewMode represents the display mode of the AI coders view
type ViewMode string

const (
	ViewModeList ViewMode = "list" // Show session list
	ViewModePTY  ViewMode = "pty"  // Show active session PTY
)

// SessionViewState represents the UI state for session viewing
type SessionViewState struct {
	Mode             ViewMode
	FocusedSessionID string
	SelectedIndex    int
	ListScrollOffset int
	PTYScrollOffset  int
	ShowHelp         bool
	ViewportWidth    int
	ViewportHeight   int
}

// AICodersViewComponent displays AI coder sessions in tmux-style layout.
//
// Design principles:
// - Shows active coding sessions
// - Displays session status (active, idle, error)
// - Shows recent activity per session
// - Keyboard selection and switching
// - Integration with aicoder package
type AICodersViewComponent struct {
	BaseComponent

	// Service reference (T022)
	service interface{} // *services.AICoderService (interface{} to avoid import cycle)

	// View state (T022)
	viewState SessionViewState

	// Cached data (T022)
	cachedSessions []SessionSummary
	cachedTerminal *TerminalState

	// Legacy fields (to be migrated)
	viewport viewport.Model
	sessions []*SessionInfo

	selectedIndex int

	// Styles
	headerStyle   lipgloss.Style
	activeStyle   lipgloss.Style
	idleStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
}

// Type aliases for component use
type SessionSummary = messages.SessionSummary
type TerminalState = messages.TerminalState

// SessionInfo holds display information for an AI coder session
type SessionInfo struct {
	ID             string
	Name           string
	Status         string // "active", "idle", "error", "stopped"
	ProcessID      string
	StartTime      time.Time
	LastActivity   time.Time
	LinesExchanged int
	CurrentTask    string
	ErrorMessage   string
}

// NewAICodersViewComponent creates an AI coders view
// T022: Updated to accept service parameter
func NewAICodersViewComponent(service interface{}) *AICodersViewComponent {
	c := &AICodersViewComponent{
		BaseComponent: NewBaseComponent(),
		service:       service,
		viewState: SessionViewState{
			Mode:           ViewModeList,
			ViewportWidth:  80,
			ViewportHeight: 24,
		},
		cachedSessions: []SessionSummary{},
		viewport:       viewport.New(80, 24),
		sessions:       []*SessionInfo{},
		selectedIndex:  0,
		headerStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		activeStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		idleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		selectedStyle:  lipgloss.NewStyle().Background(lipgloss.Color("240")).Bold(true), // Gray background
		normalStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
	}

	c.updateViewport()
	return c
}

// Init initializes the component
func (c *AICodersViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *AICodersViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// T023: AI Coder Service Messages
	case messages.AICoderSessionCreatedMsg:
		// Fetch updated session list from service
		return c, c.fetchSessionList()

	case messages.SessionListMsg:
		// Update cached sessions
		c.cachedSessions = msg.Sessions
		c.viewState.SelectedIndex = 0 // Reset selection
		c.updateViewport()
		return c, nil

	case messages.AICoderSessionUpdatedMsg:
		// Update specific session in cache
		for i, sess := range c.cachedSessions {
			if sess.ID == msg.SessionID {
				c.cachedSessions[i].Status = msg.Status
				c.cachedSessions[i].LastActivity = msg.LastActivity
				break
			}
		}
		c.updateViewport()
		return c, nil

	case messages.AICoderSessionEndedMsg:
		// Remove from cached sessions
		for i, sess := range c.cachedSessions {
			if sess.ID == msg.SessionID {
				c.cachedSessions = append(c.cachedSessions[:i], c.cachedSessions[i+1:]...)
				// Adjust selection if needed
				if c.viewState.SelectedIndex >= len(c.cachedSessions) && c.viewState.SelectedIndex > 0 {
					c.viewState.SelectedIndex = len(c.cachedSessions) - 1
				}
				break
			}
		}
		// If we were viewing this session in PTY mode, return to list mode
		if c.viewState.Mode == ViewModePTY && c.viewState.FocusedSessionID == msg.SessionID {
			c.viewState.Mode = ViewModeList
			c.viewState.FocusedSessionID = ""
			c.cachedTerminal = nil
		}
		c.updateViewport()
		return c, nil

	case messages.AICoderPTYOutputMsg:
		// Fetch updated terminal state if we're viewing this session
		if c.viewState.Mode == ViewModePTY && c.viewState.FocusedSessionID == msg.SessionID {
			return c, c.fetchTerminalState(msg.SessionID)
		}
		return c, nil

	case messages.TerminalStateMsg:
		// Update cached terminal state
		c.cachedTerminal = msg.State
		c.updatePTYView()
		return c, nil

	case messages.AICoderErrorMsg:
		// Display error (for now, just update session status)
		// TODO: Consider adding notification system integration
		for i, sess := range c.cachedSessions {
			if sess.ID == msg.SessionID {
				c.cachedSessions[i].Status = messages.SessionError
				break
			}
		}
		c.updateViewport()
		return c, nil

	case tea.WindowSizeMsg:
		// T027: Resize handling with PTY notification
		c.viewState.ViewportWidth = msg.Width
		c.viewState.ViewportHeight = msg.Height
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height

		// If in PTY mode, resize the PTY
		if c.viewState.Mode == ViewModePTY && c.viewState.FocusedSessionID != "" {
			if c.service != nil {
				svc := c.service.(*services.AICoderService)
				_ = svc.ResizePTY(c.viewState.FocusedSessionID, msg.Width, msg.Height)
			}
		}

		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		// T024: View mode-specific keyboard routing
		switch c.viewState.Mode {
		case ViewModeList:
			switch msg.String() {
			case "j", "down":
				// Navigate down in session list
				if len(c.cachedSessions) > 0 {
					c.viewState.SelectedIndex = (c.viewState.SelectedIndex + 1) % len(c.cachedSessions)
					c.updateViewport()
				}
				return c, nil
			case "k", "up":
				// Navigate up in session list
				if len(c.cachedSessions) > 0 {
					c.viewState.SelectedIndex = (c.viewState.SelectedIndex - 1 + len(c.cachedSessions)) % len(c.cachedSessions)
					c.updateViewport()
				}
				return c, nil
			case "enter":
				// Switch to PTY mode for selected session
				if len(c.cachedSessions) > 0 && c.viewState.SelectedIndex < len(c.cachedSessions) {
					c.viewState.Mode = ViewModePTY
					c.viewState.FocusedSessionID = c.cachedSessions[c.viewState.SelectedIndex].ID
					return c, c.fetchTerminalState(c.viewState.FocusedSessionID)
				}
				return c, nil
			default:
				// Forward other keys to viewport for scrolling
				c.viewport, cmd = c.viewport.Update(msg)
				return c, cmd
			}

		case ViewModePTY:
			switch msg.String() {
			case "esc":
				// Return to list mode
				c.viewState.Mode = ViewModeList
				c.viewState.FocusedSessionID = ""
				c.cachedTerminal = nil
				c.updateViewport()
				return c, nil
			default:
				// Forward all other keys to PTY
				if c.viewState.FocusedSessionID != "" {
					return c, c.sendInput(c.viewState.FocusedSessionID, []byte(msg.String()))
				}
				return c, nil
			}
		}
	}

	return c, nil
}

// View renders the AI coders display
func (c *AICodersViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	// Ensure viewport content is up to date for current mode
	switch c.viewState.Mode {
	case ViewModeList:
		c.updateViewport()
	case ViewModePTY:
		c.updatePTYView()
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *AICodersViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content with formatted session list
func (c *AICodersViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render(fmt.Sprintf("AI Coder Sessions (%d active)", len(c.cachedSessions)))
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))
	lines = append(lines, "")

	if len(c.cachedSessions) == 0 {
		lines = append(lines, c.normalStyle.Render("No active AI coder sessions"))
		lines = append(lines, "")
		lines = append(lines, c.normalStyle.Render("Press 'a' to start a new AI coding session"))
		lines = append(lines, c.normalStyle.Render("AI coders provide tmux-style interactive coding assistance"))
	} else {
		// Session list using cached data
		for i, session := range c.cachedSessions {
			lines = append(lines, c.formatCachedSessionRow(session, i == c.viewState.SelectedIndex))
			lines = append(lines, "")
		}

		// Keyboard shortcuts
		lines = append(lines, strings.Repeat("─", c.viewport.Width))
		lines = append(lines, c.normalStyle.Render("j/k: Navigate  Enter: Attach  a: New  d: Detach  q: Stop"))
	}

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}

// formatCachedSessionRow formats a single cached session row
func (c *AICodersViewComponent) formatCachedSessionRow(session SessionSummary, selected bool) string {
	var lines []string

	// Session header with selection highlight
	statusStyle := c.normalStyle
	switch session.Status {
	case messages.SessionActive:
		statusStyle = c.activeStyle
	case messages.SessionIdle:
		statusStyle = c.idleStyle
	case messages.SessionError:
		statusStyle = c.errorStyle
	}

	statusIcon := "○"
	if session.Status == messages.SessionActive {
		statusIcon = "●"
	}

	// Truncate ID if longer than 8 characters
	sessionID := session.ID
	if len(sessionID) > 8 {
		sessionID = sessionID[:8]
	}

	sessionHeader := fmt.Sprintf("%s %s (%s)", statusIcon, session.Name, sessionID)
	statusText := fmt.Sprintf(" [%s]", session.Status)

	if selected {
		sessionHeader = c.selectedStyle.Render(sessionHeader + statusText)
	} else {
		sessionHeader = c.normalStyle.Render(sessionHeader) + statusStyle.Render(statusText)
	}

	lines = append(lines, sessionHeader)

	// Session details
	lastActivity := time.Since(session.LastActivity).Round(time.Second)
	details := fmt.Sprintf("  Status: %s  |  Last Activity: %s ago",
		session.Status, lastActivity)
	lines = append(lines, c.normalStyle.Render(details))

	return strings.Join(lines, "\n")
}

// formatSessionRow formats a single session row (legacy method for compatibility)
func (c *AICodersViewComponent) formatSessionRow(session *SessionInfo, selected bool) string {
	var lines []string

	// Session header with selection highlight
	statusStyle := c.normalStyle
	switch session.Status {
	case "active":
		statusStyle = c.activeStyle
	case "idle":
		statusStyle = c.idleStyle
	case "error":
		statusStyle = c.errorStyle
	}

	statusIcon := "○"
	if session.Status == "active" {
		statusIcon = "●"
	}

	// Truncate ID if longer than 8 characters
	sessionID := session.ID
	if len(sessionID) > 8 {
		sessionID = sessionID[:8]
	}

	sessionHeader := fmt.Sprintf("%s %s (%s)", statusIcon, session.Name, sessionID)
	statusText := fmt.Sprintf(" [%s]", session.Status)

	if selected {
		sessionHeader = c.selectedStyle.Render(sessionHeader + statusText)
	} else {
		sessionHeader = c.normalStyle.Render(sessionHeader) + statusStyle.Render(statusText)
	}

	lines = append(lines, sessionHeader)

	// Session details
	uptime := time.Since(session.StartTime).Round(time.Second)
	lastActivity := time.Since(session.LastActivity).Round(time.Second)

	details := fmt.Sprintf("  Uptime: %s  |  Last Activity: %s ago  |  Lines: %d",
		uptime, lastActivity, session.LinesExchanged)
	lines = append(lines, c.normalStyle.Render(details))

	// Current task
	if session.CurrentTask != "" {
		task := session.CurrentTask
		if len(task) > 70 {
			task = task[:67] + "..."
		}
		lines = append(lines, c.normalStyle.Render("  Task: "+task))
	}

	// Error message if any
	if session.ErrorMessage != "" {
		errMsg := session.ErrorMessage
		if len(errMsg) > 70 {
			errMsg = errMsg[:67] + "..."
		}
		lines = append(lines, c.errorStyle.Render("  Error: "+errMsg))
	}

	return strings.Join(lines, "\n")
}

// AddSession adds a new session to the view
func (c *AICodersViewComponent) AddSession(session *SessionInfo) {
	c.sessions = append(c.sessions, session)
	c.updateViewport()
}

// RemoveSession removes a session by ID
func (c *AICodersViewComponent) RemoveSession(sessionID string) {
	for i, session := range c.sessions {
		if session.ID == sessionID {
			c.sessions = append(c.sessions[:i], c.sessions[i+1:]...)
			if c.selectedIndex >= len(c.sessions) && c.selectedIndex > 0 {
				c.selectedIndex = len(c.sessions) - 1
			}
			c.updateViewport()
			break
		}
	}
}

// UpdateSession updates an existing session
func (c *AICodersViewComponent) UpdateSession(sessionID string, updater func(*SessionInfo)) {
	for _, session := range c.sessions {
		if session.ID == sessionID {
			updater(session)
			c.updateViewport()
			break
		}
	}
}

// GetSelectedSession returns the currently selected session
func (c *AICodersViewComponent) GetSelectedSession() *SessionInfo {
	if len(c.sessions) == 0 || c.selectedIndex >= len(c.sessions) {
		return nil
	}
	return c.sessions[c.selectedIndex]
}

// T026: Command Constructor Helpers
// These methods return tea.Cmd functions that call service methods

// fetchSessionList returns a command that fetches the session list from the service
func (c *AICodersViewComponent) fetchSessionList() tea.Cmd {
	return func() tea.Msg {
		if c.service == nil {
			return nil
		}
		svc := c.service.(*services.AICoderService)
		sessions := svc.ListSessions()
		return messages.SessionListMsg{Sessions: sessions}
	}
}

// fetchTerminalState returns a command that fetches terminal state for a session
func (c *AICodersViewComponent) fetchTerminalState(sessionID string) tea.Cmd {
	return func() tea.Msg {
		if c.service == nil {
			return nil
		}
		svc := c.service.(*services.AICoderService)
		state, err := svc.GetTerminalState(sessionID)
		if err != nil {
			return messages.AICoderErrorMsg{
				SessionID: sessionID,
				Error:     err,
				Severity:  "error",
			}
		}
		return messages.TerminalStateMsg{
			SessionID: sessionID,
			State:     state,
		}
	}
}

// sendInput returns a command that sends input to a session's PTY
func (c *AICodersViewComponent) sendInput(sessionID string, input []byte) tea.Cmd {
	return func() tea.Msg {
		if c.service == nil {
			return nil
		}
		svc := c.service.(*services.AICoderService)
		if err := svc.SendInput(sessionID, input); err != nil {
			return messages.AICoderErrorMsg{
				SessionID: sessionID,
				Error:     err,
				Severity:  "error",
			}
		}
		return nil
	}
}

// T025: PTY Rendering Helper
// updatePTYView updates the viewport with terminal content when in PTY mode
func (c *AICodersViewComponent) updatePTYView() {
	if c.viewState.Mode != ViewModePTY || c.cachedTerminal == nil {
		return
	}

	var lines []string

	// Header
	sessionName := "Unknown Session"
	for _, sess := range c.cachedSessions {
		if sess.ID == c.viewState.FocusedSessionID {
			sessionName = sess.Name
			break
		}
	}

	lines = append(lines, c.headerStyle.Render(fmt.Sprintf("Session: %s", sessionName)))
	lines = append(lines, strings.Repeat("─", c.viewState.ViewportWidth))

	// Terminal content with proper line wrapping and formatting
	termState := c.cachedTerminal
	if len(termState.Lines) > 0 {
		// Display terminal lines with proper handling for long lines
		for _, line := range termState.Lines {
			// Handle lines longer than viewport width
			if len(line) > c.viewState.ViewportWidth {
				// Simple word wrapping - can be enhanced later
				for len(line) > c.viewState.ViewportWidth {
					lines = append(lines, line[:c.viewState.ViewportWidth])
					line = line[c.viewState.ViewportWidth:]
				}
				if len(line) > 0 {
					lines = append(lines, line)
				}
			} else {
				lines = append(lines, line)
			}

			// Stop if we've filled the viewport (minus header/footer space)
			if len(lines) >= c.viewState.ViewportHeight-4 {
				break
			}
		}
	} else {
		// Show placeholder when no terminal content yet
		lines = append(lines, c.normalStyle.Render("Terminal session starting..."))
		lines = append(lines, c.normalStyle.Render("Waiting for output from AI coder"))
	}

	// Fill remaining space with empty lines if needed
	for len(lines) < c.viewState.ViewportHeight-3 {
		lines = append(lines, "")
	}

	// Cursor indicator at current position (if within viewport bounds)
	if termState.Cursor.Visible && termState.Cursor.Row < len(lines) {
		cursorLineIdx := termState.Cursor.Row
		if cursorLineIdx >= 0 && cursorLineIdx < len(lines) {
			// Add cursor indicator to the appropriate line
			line := lines[cursorLineIdx]
			if termState.Cursor.Col < len(line) {
				// Insert cursor marker at position
				lines[cursorLineIdx] = line[:termState.Cursor.Col] + "█" + line[termState.Cursor.Col+1:]
			} else {
				// Append cursor at end of line
				lines[cursorLineIdx] = line + "█"
			}
		}
	}

	// Footer
	lines = append(lines, strings.Repeat("─", c.viewState.ViewportWidth))
	lines = append(lines, c.normalStyle.Render("ESC=List | ?=Help | ↑↓=Scroll | Type to send input"))

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}
