package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
func NewAICodersViewComponent() *AICodersViewComponent {
	c := &AICodersViewComponent{
		BaseComponent: NewBaseComponent(),
		viewport:      viewport.New(80, 24),
		sessions:      []*SessionInfo{},
		selectedIndex: 0,
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		activeStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		idleStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("240")).Bold(true), // Gray background
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
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
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height
		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			// Navigate down
			if len(c.sessions) > 0 {
				c.selectedIndex = (c.selectedIndex + 1) % len(c.sessions)
				c.updateViewport()
			}
			return c, nil
		case "k", "up":
			// Navigate up
			if len(c.sessions) > 0 {
				c.selectedIndex = (c.selectedIndex - 1 + len(c.sessions)) % len(c.sessions)
				c.updateViewport()
			}
			return c, nil
		default:
			// Forward other keys to viewport for scrolling
			c.viewport, cmd = c.viewport.Update(msg)
			return c, cmd
		}
	}

	return c, nil
}

// View renders the AI coders display
func (c *AICodersViewComponent) View() string {
	if !c.IsVisible() {
		return ""
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
	header := c.headerStyle.Render(fmt.Sprintf("AI Coder Sessions (%d active)", len(c.sessions)))
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))
	lines = append(lines, "")

	if len(c.sessions) == 0 {
		lines = append(lines, c.normalStyle.Render("No active AI coder sessions"))
		lines = append(lines, "")
		lines = append(lines, c.normalStyle.Render("Press 'a' to start a new AI coding session"))
		lines = append(lines, c.normalStyle.Render("AI coders provide tmux-style interactive coding assistance"))
	} else {
		// Session list
		for i, session := range c.sessions {
			lines = append(lines, c.formatSessionRow(session, i == c.selectedIndex))
			lines = append(lines, "")
		}

		// Keyboard shortcuts
		lines = append(lines, strings.Repeat("─", c.viewport.Width))
		lines = append(lines, c.normalStyle.Render("j/k: Navigate  Enter: Attach  a: New  d: Detach  q: Stop"))
	}

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}

// formatSessionRow formats a single session row
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
