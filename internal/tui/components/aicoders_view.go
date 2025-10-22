package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// ViewMode represents the display mode of the AI coders view
type ViewMode string

const (
	ViewModeList ViewMode = "list"
	ViewModePTY  ViewMode = "pty"
)

// AICodersViewComponent renders AI coder sessions and their PTY output
// T022: Updated to accept service parameter
// T056-T061: Enhanced with WebCompanion integration for AI interactions
// NOTE: This component is now stateless - state is managed by AICoderState sub-model
type AICodersViewComponent struct {
	BaseComponent

	// Service reference (T022)
	service interface{} // *services.AICoderService (interface{} to avoid import cycle)

	// View rendering fields (stateless - only for UI configuration)
	width  int
	height int

	
	// Styles
	headerStyle   lipgloss.Style
	activeStyle   lipgloss.Style
	idleStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
}

// NewAICodersViewComponent creates an AI coders view
// T022: Updated to accept service parameter
func NewAICodersViewComponent(service interface{}) *AICodersViewComponent {
	c := &AICodersViewComponent{
		BaseComponent: NewBaseComponent(),
		service:       service,
		width:         80,
		height:        24,
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		activeStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		idleStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("240")).Bold(true), // Gray background
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
	}

	return c
}

// Init initializes the component
func (c *AICodersViewComponent) Init() tea.Cmd {
	return nil
}



// ViewWithState renders the component using external state (stateless/Elm-compliant)
func (c *AICodersViewComponent) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() || masterData == nil || masterData.AICoderState == nil {
		return ""
	}

	aiState := masterData.AICoderState

	// Determine view mode based on whether there's a focused session
	var viewMode ViewMode
	focusedSessionID := aiState.FocusedSession

	if focusedSessionID != "" && aiState.HasActiveSessions() {
		viewMode = ViewModePTY
	} else {
		viewMode = ViewModeList
	}

	// Render based on mode
	switch viewMode {
	case ViewModePTY:
		return c.renderPTYViewWithState(aiState, focusedSessionID)
	case ViewModeList:
		return c.renderListViewWithState(aiState)
	default:
		return c.normalStyle.Render("Invalid view mode")
	}
}

// renderPTYViewWithState renders PTY terminal content using state from sub-model
func (c *AICodersViewComponent) renderPTYViewWithState(aiState *state.AICoderState, sessionID string) string {
	if sessionID == "" {
		return c.normalStyle.Render("No session selected")
	}

	var lines []string

	// Header
	sessionInfo, exists := aiState.GetSessionInfo(sessionID)
	if !exists {
		sessionInfo = state.SessionInfo{Name: "Unknown Session"}
	}

	lines = append(lines, c.headerStyle.Render(fmt.Sprintf("Session: %s", sessionInfo.Name)))
	lines = append(lines, strings.Repeat("─", c.width))

	// NEW: Use screen buffer for rendering (optimized for 60 FPS)
	if aiState.ScreenBuffer != nil && len(aiState.ScreenBuffer) > 0 {
		// Direct rendering from screen buffer - already processed by vt10x
		for _, runeSlice := range aiState.ScreenBuffer {
			line := string(runeSlice)
			lines = append(lines, line)
		}

		// Show cursor position if dirty (needs redraw)
		if aiState.ScreenDirty {
			// Could add cursor indicator here if needed
		}
	} else {
		// Fallback: Use deprecated methods during migration
		ptyLines := aiState.GetPTYOutput(sessionID)
		terminalState := aiState.GetTerminalState(sessionID)

		// Prefer VT10x terminal state if available, otherwise use PTY buffer
		var contentLines []string
		if terminalState != nil && len(terminalState.Lines) > 0 {
			contentLines = terminalState.Lines
		} else if len(ptyLines) > 0 {
			contentLines = ptyLines
		} else {
			contentLines = []string{
				c.normalStyle.Render(fmt.Sprintf("Terminal session starting for: %s", sessionInfo.Name)),
				c.normalStyle.Render("Waiting for PTY output from AI coder..."),
			}
		}

		// Display fallback content with proper width handling
		for _, line := range contentLines {
			cleanLine := processANSIFormatting(line)

			// Handle long lines with word wrapping
			if len(cleanLine) > c.width {
				for len(cleanLine) > c.width {
					lines = append(lines, cleanLine[:c.width])
					cleanLine = cleanLine[c.width:]
				}
				if len(cleanLine) > 0 {
					lines = append(lines, cleanLine)
				}
			} else {
				lines = append(lines, cleanLine)
			}

			// Stop if we've filled the viewport (minus header/footer space)
			if len(lines) >= c.height-4 {
				break
			}
		}
	}

	// Fill remaining space with empty lines if needed
	for len(lines) < c.height-3 {
		lines = append(lines, "")
	}

	// Simple cursor indicator
	if len(lines) > 2 {
		lastLineIdx := len(lines) - 3
		if lastLineIdx >= 0 && lastLineIdx < len(lines) {
			line := lines[lastLineIdx]
			if len(line) < c.width-1 {
				lines[lastLineIdx] = line + "█"
			}
		}
	}

	// Footer
	lines = append(lines, strings.Repeat("─", c.width))
	lines = append(lines, c.normalStyle.Render("ESC=List | ?=Help | ↑↓=Scroll | Type to send input"))

	return strings.Join(lines, "\n")
}

// renderListViewWithState renders the session list using state from sub-model
func (c *AICodersViewComponent) renderListViewWithState(aiState *state.AICoderState) string {
	var lines []string

	// Header
	lines = append(lines, c.headerStyle.Render("AI Coder Sessions"))
	lines = append(lines, strings.Repeat("═", c.width))

	if !aiState.HasActiveSessions() {
		lines = append(lines, "")
		lines = append(lines, c.normalStyle.Render("No active AI coder sessions"))
		lines = append(lines, c.normalStyle.Render("Press 'n' to create a new session"))
		return strings.Join(lines, "\n")
	}

	// Session list
	sessionIDs := aiState.GetActiveSessionIDs()
	for _, sessionID := range sessionIDs {
		sessionInfo, _ := aiState.GetSessionInfo(sessionID)
		selected := sessionID == aiState.FocusedSession

		line := c.formatSessionRowWithInfo(sessionInfo, selected)
		lines = append(lines, line)
	}

	// Footer
	lines = append(lines, strings.Repeat("═", c.width))
	lines = append(lines, c.normalStyle.Render("n=New | Enter=PTY | Del=Stop | ?=Help"))

	return strings.Join(lines, "\n")
}

// formatSessionRowWithInfo formats a session row for display (stateless version)
func (c *AICodersViewComponent) formatSessionRowWithInfo(session state.SessionInfo, selected bool) string {
	var prefix string
	if selected {
		prefix = c.selectedStyle.Render("▶ ")
	} else {
		prefix = "  "
	}

	var statusStyle lipgloss.Style
	var statusIcon string

	switch session.Status {
	case "active":
		statusStyle = c.activeStyle
		statusIcon = "●"
	case "idle":
		statusStyle = c.idleStyle
		statusIcon = "○"
	default:
		statusStyle = c.errorStyle
		statusIcon = "✗"
	}

	statusText := statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, session.Status))
	sessionText := fmt.Sprintf("%s%s (ID: %s)", prefix, session.Name, session.ID[:8])

	if selected {
		return c.selectedStyle.Render(sessionText) + " " + statusText
	}
	return sessionText + " " + statusText
}

// SetSize updates component dimensions
func (c *AICodersViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.width = width
	c.height = height
}

// processANSIFormatting processes common ANSI escape codes for terminal display
func processANSIFormatting(line string) string {
	// TEMPORARY: Return the line as-is to preserve ANSI color codes
	// This allows 24-bit color and formatting to pass through to the terminal
	// TODO: Implement proper ANSI rendering with lipgloss styles
	return line
}

