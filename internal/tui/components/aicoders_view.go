package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
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

	// ===== Layout (provided by centralized system) =====
	layoutRegions LayoutRegions // Single source of truth for dimensions

	// Viewport for scrollable terminal output
	viewport viewport.Model

	// Styles
	headerStyle   lipgloss.Style
	contentStyle  lipgloss.Style
	footerStyle   lipgloss.Style
	activeStyle   lipgloss.Style
	idleStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
}

// NewAICodersViewComponent creates an AI coders view
// T022: Updated to accept service parameter
func NewAICodersViewComponent(service interface{}) *AICodersViewComponent {
	// Initialize viewport for scrollable terminal output
	vp := viewport.New(80, 20) // Initial size, will be updated by SetLayoutRegions
	vp.MouseWheelEnabled = true
	vp.KeyMap = viewport.KeyMap{
		PageDown: viewport.DefaultKeyMap().PageDown,
		PageUp:   viewport.DefaultKeyMap().PageUp,
		HalfPageDown: viewport.DefaultKeyMap().HalfPageDown,
		HalfPageUp:   viewport.DefaultKeyMap().HalfPageUp,
		Down:     viewport.DefaultKeyMap().Down,
		Up:       viewport.DefaultKeyMap().Up,
	}

	c := &AICodersViewComponent{
		BaseComponent: NewBaseComponent(),
		service:       service,
		viewport:      vp,
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		contentStyle:  lipgloss.NewStyle().Padding(0, 1),                                 // Content padding
		footerStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),            // Gray footer
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

// UpdateViewport handles viewport scrolling for arrow keys, page up/down, and mouse events
// This should be called from the main model's Update() method when handling key messages
func (c *AICodersViewComponent) UpdateViewport(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	return cmd
}

// Update handles component updates including mouse events and auto-scroll
func (c *AICodersViewComponent) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle mouse wheel events for scrolling
		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			// Update viewport with mouse event
			return c.UpdateViewport(msg)
		}

	// Add other message types as needed
	default:
		// For other message types, update viewport if needed
		return c.UpdateViewport(msg)
	}

	return nil
}

// AutoScrollToBottom automatically scrolls the viewport to the bottom
// This should be called when new content is added to keep the latest output visible
func (c *AICodersViewComponent) AutoScrollToBottom() {
	if c.viewport.TotalLineCount() > c.viewport.Height {
		// Only auto-scroll if the viewport is already near the bottom (within 5 lines)
		// This prevents auto-scroll when user is manually scrolling up to read earlier content
		currentOffset := c.viewport.YOffset
		maxOffset := c.viewport.TotalLineCount() - c.viewport.Height

		// Auto-scroll if user is at bottom or close to it (within 5 lines)
		if maxOffset-currentOffset <= 5 {
			c.viewport.GotoBottom()
		}
	} else {
		// If content fits in viewport, always show the latest
		c.viewport.GotoBottom()
	}
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
// NOW: Uses centralized layout regions and viewport for scrolling
func (c *AICodersViewComponent) renderPTYViewWithState(aiState *state.AICoderState, sessionID string) string {
	if sessionID == "" {
		return c.normalStyle.Render("No session selected")
	}

	// Get content width from layout regions (or fallback to legacy width)
	contentWidth := c.layoutRegions.ContentWidth
	if contentWidth == 0 {
		contentWidth = c.width
	}

	// === HEADER (Fixed) ===
	sessionInfo, exists := aiState.GetSessionInfo(sessionID)
	if !exists {
		sessionInfo = state.SessionInfo{Name: "Unknown Session"}
	}

	headerLines := []string{
		c.headerStyle.Render(fmt.Sprintf("Session: %s (%s)", sessionInfo.Name, sessionInfo.Status)),
		strings.Repeat("─", contentWidth),
	}
	header := strings.Join(headerLines, "\n")

	// === VIEWPORT CONTENT (Scrollable) ===
	// Collect all content lines (viewport handles overflow and scrolling)
	var contentLines []string

	// Use screen buffer for rendering (optimized for 60 FPS with debouncer)
	if aiState.ScreenBuffer != nil && len(aiState.ScreenBuffer) > 0 {
		// Direct rendering from screen buffer - already processed by vt10x
		for _, runeSlice := range aiState.ScreenBuffer {
			line := string(runeSlice)
			// Add left padding for better readability
			contentLines = append(contentLines, "  "+line)
		}

		// Add cursor indicator if dirty (needs redraw)
		if aiState.ScreenDirty && len(contentLines) > 0 {
			lastIdx := len(contentLines) - 1
			contentLines[lastIdx] = contentLines[lastIdx] + "█"
		}
	} else {
		// No screen buffer yet - show waiting message
		contentLines = []string{
			c.normalStyle.Render(fmt.Sprintf("Terminal session starting for: %s", sessionInfo.Name)),
			c.normalStyle.Render("Waiting for PTY output from AI coder..."),
			"",
			c.footerStyle.Render("The terminal will appear here once the AI coder session starts."),
		}
	}

	// Set viewport content (viewport handles scrolling and overflow automatically)
	c.viewport.SetContent(strings.Join(contentLines, "\n"))

	// Auto-scroll to bottom if new content was added
	c.AutoScrollToBottom()

	// Get viewport rendering (with scroll position)
	viewportContent := c.viewport.View()

	// === ADD SCROLL INDICATOR (Right side) ===
	// Add a visual scrollbar on the right side if content is scrollable
	if c.viewport.TotalLineCount() > c.viewport.Height {
		viewportContent = c.addScrollbar(viewportContent)
	}

	// === FOOTER (Fixed) ===
	var footerLines []string
	footerLines = append(footerLines, strings.Repeat("─", contentWidth))

	// Show scroll position if content overflows
	scrollInfo := ""
	if c.viewport.TotalLineCount() > c.viewport.Height {
		scrollPercent := int(c.viewport.ScrollPercent() * 100)
		scrollInfo = fmt.Sprintf(" [%d%%]", scrollPercent)
	}

	// Show different footer based on terminal focus mode
	if aiState.TerminalFocused {
		footerLines = append(footerLines, c.activeStyle.Render("🔒 TERMINAL FOCUS MODE - Ctrl+T to release (ESC goes to terminal)"+scrollInfo))
	} else {
		footerLines = append(footerLines, c.normalStyle.Render("Enter or Ctrl+T=Focus | ESC=List | ↑↓=Scroll | ?=Help"+scrollInfo))
	}
	footer := strings.Join(footerLines, "\n")

	// Combine header, viewport content, and footer
	return lipgloss.JoinVertical(lipgloss.Left, header, viewportContent, footer)
}

// addScrollbar adds a visual scrollbar to the right side of the viewport content
func (c *AICodersViewComponent) addScrollbar(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Calculate scrollbar dimensions
	totalLines := c.viewport.TotalLineCount()
	visibleLines := c.viewport.Height
	scrollPercent := c.viewport.ScrollPercent()

	// Calculate scrollbar thumb position and size
	// Thumb size represents the visible portion relative to total content
	thumbSize := max(1, (visibleLines * visibleLines) / totalLines)
	trackSize := visibleLines
	thumbPosition := int(scrollPercent * float64(trackSize - thumbSize))

	// Build scrollbar for each line
	scrollbarChars := make([]string, len(lines))
	for i := range lines {
		if i >= thumbPosition && i < thumbPosition + thumbSize {
			scrollbarChars[i] = c.activeStyle.Render("█") // Thumb (bright color)
		} else {
			scrollbarChars[i] = c.footerStyle.Render("│") // Track (dimmed)
		}
	}

	// Add scrollbar to each line
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = line + " " + scrollbarChars[i]
	}

	return strings.Join(result, "\n")
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
// SetLayoutRegions updates the view's layout from centralized calculation (NEW)
func (c *AICodersViewComponent) SetLayoutRegions(regions LayoutRegions) {
	c.layoutRegions = regions

	// Update legacy fields for backward compatibility
	// Update viewport with calculated dimensions
	viewportWidth, viewportHeight := regions.GetViewportArea()

	// Account for header (3 lines) and footer (2 lines)
	headerHeight := 3
	footerHeight := 2
	contentHeight := viewportHeight - headerHeight - footerHeight

	// Ensure minimum size
	if contentHeight < 10 {
		contentHeight = 10
	}

	// Configure viewport
	c.viewport.Width = viewportWidth - 2  // Minus padding
	c.viewport.Height = contentHeight

	// Update styles with region dimensions
	c.contentStyle = c.contentStyle.
		Width(regions.ContentWidth).
		Height(contentHeight)
}

// GetLayoutRegions returns the current layout regions (for PTY sizing)
func (c *AICodersViewComponent) GetLayoutRegions() LayoutRegions {
	return c.layoutRegions
}

// SetSize is the legacy method (keep for backward compatibility)
func (c *AICodersViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)

	// If layoutRegions not set, create a basic one
	if c.layoutRegions.TerminalWidth == 0 {
		opts := DefaultLayoutOptions(width, height)
		c.layoutRegions = CalculateLayoutRegions(opts)

		// Update viewport
		c.viewport.Width = width - 4
		c.viewport.Height = height - 6 // Account for header/footer
	}
}

// processANSIFormatting processes common ANSI escape codes for terminal display
func processANSIFormatting(line string) string {
	// TEMPORARY: Return the line as-is to preserve ANSI color codes
	// This allows 24-bit color and formatting to pass through to the terminal
	// TODO: Implement proper ANSI rendering with lipgloss styles
	return line
}

