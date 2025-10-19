package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// MCPDebugViewComponentNew is a stateless BubbleTea component for displaying MCP debug information
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type MCPDebugViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	width   int
	height  int
	debugMode bool

	// Components for rendering (minimal, stateless)
	connectionsList list.Model
	activityViewport viewport.Model

	// Styles
	headerStyle      lipgloss.Style
	subtitleStyle    lipgloss.Style
	connectionsStyle lipgloss.Style
	activityStyle    lipgloss.Style
	noSelectionStyle lipgloss.Style
	titleStyle       lipgloss.Style
	methodStyle      lipgloss.Style
	timestampStyle   lipgloss.Style
	errorStyle       lipgloss.Style
	durationStyle    lipgloss.Style
}

// NewMCPDebugView creates a new stateless MCP debug view component
func NewMCPDebugView(debugMode bool) *MCPDebugViewComponentNew {
	if !debugMode {
		return &MCPDebugViewComponentNew{debugMode: false}
	}

	// Initialize MCP connections list
	connectionsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	connectionsList.Title = "MCP Connections"
	connectionsList.SetShowStatusBar(false)

	return &MCPDebugViewComponentNew{
		BaseComponent:     NewBaseComponent(),
		width:            80,
		height:           24,
		debugMode:        debugMode,
		connectionsList:  connectionsList,
		activityViewport: viewport.New(0, 0),

		// Initialize styles
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Padding(0, 2).
			MarginBottom(1),
		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true).
			MarginBottom(1),
		connectionsStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		activityStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		noSelectionStyle: lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("245")),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")),
		methodStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		timestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		durationStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
	}
}

// ViewWithState renders the MCP debug view using external state (Redux/Elm compliant)
func (c *MCPDebugViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.debugMode || !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.MCPDebugState == nil {
		return c.renderErrorState("MCP debug state not available")
	}

	debugState := masterData.MCPDebugState

	// Update components with current state
	c.updateComponents(debugState)

	// Generate content based on current state
	return c.renderContent(debugState)
}

// updateComponents updates the list and viewport with current state
func (c *MCPDebugViewComponentNew) updateComponents(debugState *state.MCPDebugState) {
	// Update connections list
	sortedConnections := debugState.GetSortedConnections()
	items := make([]list.Item, len(sortedConnections))
	for i, conn := range sortedConnections {
		items[i] = MCPConnectionItemWrapper{conn: &conn}
	}
	c.connectionsList.SetItems(items)

	// Update activity viewport for selected client
	selectedClient := debugState.GetSelectedClient()
	if selectedClient != "" {
		c.updateActivityView(debugState, selectedClient)
	} else {
		c.activityViewport.SetContent("")
	}
}

// updateActivityView updates the activity viewport content
func (c *MCPDebugViewComponentNew) updateActivityView(debugState *state.MCPDebugState, selectedClient string) {
	var content strings.Builder

	// Find the connection info
	conn := debugState.GetConnectionByID(selectedClient)
	if conn == nil {
		content.WriteString("Connection not found")
		c.activityViewport.SetContent(content.String())
		return
	}

	content.WriteString(c.titleStyle.Render(fmt.Sprintf("Activity Log - %s", conn.ClientName)))
	content.WriteString("\n\n")

	// Get activities for this client
	activities := debugState.GetActivitiesForClient(selectedClient)
	if len(activities) == 0 {
		content.WriteString(c.timestampStyle.Render("No activity recorded yet"))
		c.activityViewport.SetContent(content.String())
		return
	}

	// Show activities in reverse order (newest first)
	for i := len(activities) - 1; i >= 0; i-- {
		activity := activities[i]

		content.WriteString(c.timestampStyle.Render(activity.Timestamp.Format("15:04:05.000")))
		content.WriteString(" ")
		content.WriteString(c.methodStyle.Render(activity.Method))
		content.WriteString(" ")
		content.WriteString(c.durationStyle.Render(fmt.Sprintf("(%dms)", activity.Duration.Milliseconds())))
		content.WriteString("\n")

		if activity.Params != "" && activity.Params != "{}" && activity.Params != "null" {
			params := activity.Params
			if len(params) > 150 {
				params = params[:147] + "..."
			}
			content.WriteString(fmt.Sprintf("  → Params: %s\n", params))
		}

		if activity.Error != "" {
			content.WriteString(c.errorStyle.Render(fmt.Sprintf("  ✗ Error: %s\n", activity.Error)))
		} else if activity.Response != "" {
			response := activity.Response
			if len(response) > 150 {
				response = response[:147] + "..."
			}
			content.WriteString(fmt.Sprintf("  ← Response: %s\n", response))
		}

		content.WriteString("\n")
	}

	c.activityViewport.SetContent(content.String())
}

// renderContent generates the main content based on debug state
func (c *MCPDebugViewComponentNew) renderContent(debugState *state.MCPDebugState) string {
	var content strings.Builder

	// Header with branding and description
	content.WriteString(c.headerStyle.Render("🔌  MCP Connections"))
	content.WriteString("\n")
	content.WriteString(c.subtitleStyle.Render("Monitor active MCP client connections and their activity"))
	content.WriteString("\n")

	// Calculate available height
	headerHeightLocal := 4                              // header + subtitle + margins
	availableHeight := c.height - 6 - headerHeightLocal // standard layout minus our header

	// Create left panel (connections list)
	leftWidth := c.width / 3
	rightWidth := c.width - leftWidth - 1

	// Update styles with dimensions
	c.connectionsStyle = c.connectionsStyle.Width(leftWidth).Height(availableHeight)
	c.activityStyle = c.activityStyle.Width(rightWidth).Height(availableHeight)
	c.noSelectionStyle = c.noSelectionStyle.Width(rightWidth-2).Height(availableHeight-2)

	// Update component dimensions
	c.connectionsList.SetWidth(leftWidth - 2)
	c.connectionsList.SetHeight(availableHeight - 2)
	c.activityViewport.Width = rightWidth - 2
	c.activityViewport.Height = availableHeight - 2

	leftPanel := c.connectionsStyle.Render(c.connectionsList.View())

	// Right panel content
	var rightContent string
	selectedClient := debugState.GetSelectedClient()
	if selectedClient != "" {
		rightContent = c.activityViewport.View()
	} else {
		rightContent = c.noSelectionStyle.Render("Select a connection to view activity")
	}

	rightPanel := c.activityStyle.Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	content.WriteString(panels)

	return content.String()
}

// renderErrorState renders an error state when debug data is not available
func (c *MCPDebugViewComponentNew) renderErrorState(message string) string {
	if !c.debugMode {
		return ""
	}

	var content strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1).
		Width(c.width)

	content.WriteString(headerStyle.Render("🔌  MCP Connections"))
	content.WriteString("\n")

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Margin(1)

	content.WriteString(errorStyle.Render("⚠️  "+message))
	content.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Margin(1)

	content.WriteString(helpStyle.Render("Enable debug mode with --debug flag to see MCP connection details"))

	return content.String()
}

// SetSize updates component dimensions
func (c *MCPDebugViewComponentNew) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.headerStyle = c.headerStyle.Width(width)
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *MCPDebugViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.headerStyle = c.headerStyle.Width(width)
}

// View renders the component (legacy method for compatibility)
func (c *MCPDebugViewComponentNew) View() string {
	if !c.debugMode || !c.IsVisible() {
		return ""
	}
	return c.renderContent(nil) // This won't work well, but maintains compatibility
}

// Update handles messages (minimal - stateless component)
func (c *MCPDebugViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !c.debugMode {
		return c, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to list for selection
		c.connectionsList, cmd = c.connectionsList.Update(msg)
		return c, cmd
	}

	return c, nil
}

// Init initializes the component
func (c *MCPDebugViewComponentNew) Init() tea.Cmd {
	return nil
}

// MCPConnectionItemWrapper wraps the state MCPConnectionItem for list.Item interface
type MCPConnectionItemWrapper struct {
	conn *state.MCPConnectionItem
}

func (i MCPConnectionItemWrapper) Title() string {
	status := "🟢"
	if !i.conn.IsConnected {
		status = "🔴"
	}

	// Add connection type icon
	typeIcon := ""
	switch i.conn.ConnectionType {
	case "SSE":
		typeIcon = "📡" // Streaming connection
	case "HTTP":
		typeIcon = "🌐" // HTTP request
	case "HTTP+SSE":
		typeIcon = "🔄" // HTTP with SSE response
	default:
		typeIcon = "❓" // Legacy or unknown
	}

	return fmt.Sprintf("%s %s %s", status, typeIcon, i.conn.ClientName)
}

func (i MCPConnectionItemWrapper) Description() string {
	duration := time.Since(i.conn.ConnectedAt).Round(time.Second)
	return fmt.Sprintf("ID: %s | %s %s | Connected: %s | Requests: %d",
		i.conn.ClientID, i.conn.ConnectionType, i.conn.Method, duration, i.conn.RequestCount)
}

func (i MCPConnectionItemWrapper) FilterValue() string {
	return i.conn.ClientName + " " + i.conn.ClientID
}