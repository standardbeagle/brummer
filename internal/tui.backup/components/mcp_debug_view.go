package components

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MCPConnectionItem represents an MCP client connection
type MCPConnectionItem struct {
	ClientID       string
	ClientName     string
	ConnectedAt    time.Time
	LastActivity   time.Time
	RequestCount   int
	IsConnected    bool
	ConnectionType string // "SSE", "HTTP", "HTTP+SSE", "Legacy"
	Method         string // "GET", "POST"
}

func (i MCPConnectionItem) Title() string {
	status := "🟢"
	if !i.IsConnected {
		status = "🔴"
	}

	// Add connection type icon
	typeIcon := ""
	switch i.ConnectionType {
	case "SSE":
		typeIcon = "📡" // Streaming connection
	case "HTTP":
		typeIcon = "🌐" // HTTP request
	case "HTTP+SSE":
		typeIcon = "🔄" // HTTP with SSE response
	default:
		typeIcon = "❓" // Legacy or unknown
	}

	return fmt.Sprintf("%s %s %s", status, typeIcon, i.ClientName)
}

func (i MCPConnectionItem) Description() string {
	duration := time.Since(i.ConnectedAt).Round(time.Second)
	return fmt.Sprintf("ID: %s | %s %s | Connected: %s | Requests: %d",
		i.ClientID, i.ConnectionType, i.Method, duration, i.RequestCount)
}

func (i MCPConnectionItem) FilterValue() string {
	return i.ClientName + " " + i.ClientID
}

// MCPActivity represents an activity log entry for an MCP connection
type MCPActivity struct {
	Timestamp time.Time
	Method    string
	Params    string
	Response  string
	Error     string
	Duration  time.Duration
}

// MCPDebugView is a BubbleTea component for displaying MCP debug information
// Migrated from mcp_debug_controller.go (T028-T031)
// Architecture: Component with internal state management (no service dependency)
type MCPDebugView struct {
	// View components
	connectionsList  list.Model                    // List of MCP connections
	activityViewport viewport.Model                // Activity log for selected connection
	selectedClient   string                        // Selected MCP client ID
	connections      map[string]*MCPConnectionItem // sessionId -> connection
	activities       map[string][]MCPActivity      // sessionId -> activities
	mu               sync.RWMutex                  // Protects both connections and activities maps

	// Memory management
	maxConnections       int           // Maximum number of connections to keep
	maxActivitiesPerConn int           // Maximum activities per connection
	cleanupInterval      time.Duration // How often to cleanup old data
	lastCleanup          time.Time     // Last cleanup timestamp

	// Dependencies
	debugMode bool
	width     int
	height    int
}

// NewMCPDebugView creates a new MCP debug view component
func NewMCPDebugView(debugMode bool) *MCPDebugView {
	if !debugMode {
		return &MCPDebugView{debugMode: false}
	}

	// Initialize MCP connections list
	connectionsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	connectionsList.Title = "MCP Connections"
	connectionsList.SetShowStatusBar(false)

	return &MCPDebugView{
		connectionsList:  connectionsList,
		activityViewport: viewport.New(0, 0),
		connections:      make(map[string]*MCPConnectionItem),
		activities:       make(map[string][]MCPActivity),
		debugMode:        debugMode,
		// Memory management settings
		maxConnections:       100,         // Keep up to 100 connections
		maxActivitiesPerConn: 100,         // Keep up to 100 activities per connection
		cleanupInterval:      time.Minute, // Cleanup every minute
		lastCleanup:          time.Now(),
	}
}

// Init initializes the component (BubbleTea interface)
func (v *MCPDebugView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *MCPDebugView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}

	// Update list model
	v.connectionsList, cmd = v.connectionsList.Update(msg)

	return v, cmd
}

// View renders the component (BubbleTea interface)
func (v *MCPDebugView) View() string {
	return v.Render()
}

// SetSize updates the component's dimensions (Component interface)
func (v *MCPDebugView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Focus gives keyboard focus to the component (Component interface)
func (v *MCPDebugView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *MCPDebugView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *MCPDebugView) GetSize() (width, height int) {
	return v.width, v.height
}

// GetConnectionsList returns the connections list for direct manipulation
func (v *MCPDebugView) GetConnectionsList() *list.Model {
	return &v.connectionsList
}

// GetActivityViewport returns the activity viewport for direct manipulation
func (v *MCPDebugView) GetActivityViewport() *viewport.Model {
	return &v.activityViewport
}

// SetSelectedClient sets the currently selected MCP client
func (v *MCPDebugView) SetSelectedClient(clientID string) {
	v.selectedClient = clientID
	v.UpdateActivityView()
}

// GetSelectedClient returns the currently selected MCP client
func (v *MCPDebugView) GetSelectedClient() string {
	return v.selectedClient
}

// GetMemoryStats returns current memory usage statistics
func (v *MCPDebugView) GetMemoryStats() map[string]interface{} {
	if !v.debugMode {
		return nil
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	totalActivities := 0
	for _, activities := range v.activities {
		totalActivities += len(activities)
	}

	return map[string]interface{}{
		"connections":          len(v.connections),
		"maxConnections":       v.maxConnections,
		"totalActivities":      totalActivities,
		"maxActivitiesPerConn": v.maxActivitiesPerConn,
		"lastCleanup":          v.lastCleanup,
		"cleanupInterval":      v.cleanupInterval,
		"memoryUsagePercent":   float64(len(v.connections)) / float64(v.maxConnections) * 100,
	}
}

// cleanupOldData removes old connections and activities to manage memory usage
func (v *MCPDebugView) cleanupOldData() {
	if !v.debugMode {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()

	// Only cleanup if enough time has passed
	if now.Sub(v.lastCleanup) < v.cleanupInterval {
		return
	}
	v.lastCleanup = now

	// If we're under the connection limit, no need to cleanup
	if len(v.connections) <= v.maxConnections {
		return
	}

	// Create a slice of connections with their last activity times for sorting
	type connWithTime struct {
		sessionId    string
		lastActivity time.Time
		isConnected  bool
	}

	var connsWithTime []connWithTime
	for sessionId, conn := range v.connections {
		connsWithTime = append(connsWithTime, connWithTime{
			sessionId:    sessionId,
			lastActivity: conn.LastActivity,
			isConnected:  conn.IsConnected,
		})
	}

	// Sort by last activity time (oldest first), but prioritize disconnected sessions for removal
	sort.Slice(connsWithTime, func(i, j int) bool {
		// If one is connected and other is not, prioritize removing disconnected
		if connsWithTime[i].isConnected != connsWithTime[j].isConnected {
			return !connsWithTime[i].isConnected && connsWithTime[j].isConnected
		}
		// Otherwise sort by last activity time (oldest first)
		return connsWithTime[i].lastActivity.Before(connsWithTime[j].lastActivity)
	})

	// Remove excess connections (keep the most recent maxConnections)
	connectionsToRemove := len(v.connections) - v.maxConnections
	if connectionsToRemove > 0 {
		for i := 0; i < connectionsToRemove; i++ {
			sessionId := connsWithTime[i].sessionId
			delete(v.connections, sessionId)
			delete(v.activities, sessionId)

			// If we're removing the selected client, clear the selection
			if v.selectedClient == sessionId {
				v.selectedClient = ""
			}
		}
	}
}

// performMemoryManagement checks if cleanup is needed and performs it
func (v *MCPDebugView) performMemoryManagement() {
	if !v.debugMode {
		return
	}

	// Check if cleanup is needed (non-blocking check)
	v.mu.RLock()
	needsCleanup := len(v.connections) > v.maxConnections ||
		time.Since(v.lastCleanup) > v.cleanupInterval
	v.mu.RUnlock()

	if needsCleanup {
		v.cleanupOldData()
	}
}

// HandleConnection handles MCP connection events
func (v *MCPDebugView) HandleConnection(sessionId, clientInfo string, connected bool, connectedAt time.Time, connectionType, method string) {
	if !v.debugMode {
		return
	}

	// Perform memory management check before processing
	v.performMemoryManagement()

	v.mu.Lock()
	defer v.mu.Unlock()

	if connected {
		// Determine client name based on client info
		clientName := "Unknown Client"
		if strings.Contains(clientInfo, "Claude Code") {
			clientName = "Claude Code"
		} else if strings.Contains(clientInfo, "VS Code") {
			clientName = "VS Code MCP"
		} else if clientInfo != "" {
			if len(clientInfo) > 20 {
				clientName = clientInfo[:17] + "..."
			} else {
				clientName = clientInfo
			}
		}

		v.connections[sessionId] = &MCPConnectionItem{
			ClientID:       sessionId,
			ClientName:     clientName,
			ConnectedAt:    connectedAt,
			LastActivity:   connectedAt,
			RequestCount:   0,
			IsConnected:    true,
			ConnectionType: connectionType,
			Method:         method,
		}
		v.activities[sessionId] = []MCPActivity{}
	} else {
		// Mark as disconnected
		if conn, exists := v.connections[sessionId]; exists {
			conn.IsConnected = false
			v.connections[sessionId] = conn
		}
	}
}

// HandleActivity handles MCP activity events
func (v *MCPDebugView) HandleActivity(sessionId string, activity MCPActivity) {
	if !v.debugMode {
		return
	}

	// Perform memory management check before processing
	v.performMemoryManagement()

	v.mu.Lock()
	defer v.mu.Unlock()

	// Update connection info if exists
	if conn, exists := v.connections[sessionId]; exists {
		conn.LastActivity = activity.Timestamp
		conn.RequestCount++
		v.connections[sessionId] = conn
	} else {
		// Create a connection entry for sessions that only have activity (e.g., POST requests)
		v.connections[sessionId] = &MCPConnectionItem{
			ClientID:       sessionId,
			ClientName:     "HTTP Client",
			ConnectedAt:    activity.Timestamp,
			LastActivity:   activity.Timestamp,
			RequestCount:   1,
			IsConnected:    false,
			ConnectionType: "HTTP",
			Method:         "POST",
		}
		v.activities[sessionId] = []MCPActivity{}
	}

	// Add activity to the list
	activities := v.activities[sessionId]

	// Limit activities to prevent memory issues
	if len(activities) >= v.maxActivitiesPerConn {
		activities = activities[1:] // Remove oldest
	}

	activities = append(activities, activity)
	v.activities[sessionId] = activities
}

// UpdateConnectionsList updates the list of MCP connections
func (v *MCPDebugView) UpdateConnectionsList() {
	if !v.debugMode {
		return
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	items := make([]list.Item, 0, len(v.connections))

	// Convert connections to list items
	for _, conn := range v.connections {
		items = append(items, *conn)
	}

	// Sort by connection time (newest first)
	sort.Slice(items, func(i, j int) bool {
		connI := items[i].(MCPConnectionItem)
		connJ := items[j].(MCPConnectionItem)
		return connI.ConnectedAt.After(connJ.ConnectedAt)
	})

	v.connectionsList.SetItems(items)
}

// UpdateActivityView updates the activity view for the selected client
func (v *MCPDebugView) UpdateActivityView() {
	if !v.debugMode || v.selectedClient == "" {
		return
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	durationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Find the connection info
	conn, exists := v.connections[v.selectedClient]
	if !exists {
		content.WriteString("Connection not found")
		v.activityViewport.SetContent(content.String())
		return
	}

	content.WriteString(titleStyle.Render(fmt.Sprintf("Activity Log - %s", conn.ClientName)))
	content.WriteString("\n\n")

	// Get activities for this client
	activities, hasActivities := v.activities[v.selectedClient]
	if !hasActivities || len(activities) == 0 {
		content.WriteString(timestampStyle.Render("No activity recorded yet"))
		v.activityViewport.SetContent(content.String())
		return
	}

	// Show activities in reverse order (newest first)
	for i := len(activities) - 1; i >= 0; i-- {
		activity := activities[i]

		content.WriteString(timestampStyle.Render(activity.Timestamp.Format("15:04:05.000")))
		content.WriteString(" ")
		content.WriteString(methodStyle.Render(activity.Method))
		content.WriteString(" ")
		content.WriteString(durationStyle.Render(fmt.Sprintf("(%dms)", activity.Duration.Milliseconds())))
		content.WriteString("\n")

		if activity.Params != "" && activity.Params != "{}" && activity.Params != "null" {
			params := activity.Params
			if len(params) > 150 {
				params = params[:147] + "..."
			}
			content.WriteString(fmt.Sprintf("  → Params: %s\n", params))
		}

		if activity.Error != "" {
			content.WriteString(errorStyle.Render(fmt.Sprintf("  ✗ Error: %s\n", activity.Error)))
		} else if activity.Response != "" {
			response := activity.Response
			if len(response) > 150 {
				response = response[:147] + "..."
			}
			content.WriteString(fmt.Sprintf("  ← Response: %s\n", response))
		}

		content.WriteString("\n")
	}

	v.activityViewport.SetContent(content.String())
}

// Render renders the MCP connections view
func (v *MCPDebugView) Render() string {
	if !v.debugMode {
		return ""
	}

	var content strings.Builder

	// Header with branding and description
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1).
		Width(v.width)

	content.WriteString(headerStyle.Render("🔌  MCP Connections"))
	content.WriteString("\n")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		MarginBottom(1)

	content.WriteString(subtitleStyle.Render("Monitor active MCP client connections and their activity"))
	content.WriteString("\n")

	// Calculate available height
	headerHeightLocal := 4                              // header + subtitle + margins
	availableHeight := v.height - 6 - headerHeightLocal // standard layout minus our header

	// Create left panel (connections list)
	leftWidth := v.width / 3
	rightWidth := v.width - leftWidth - 1

	connectionsStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	activityStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	// Update list dimensions
	v.connectionsList.SetWidth(leftWidth - 2)
	v.connectionsList.SetHeight(availableHeight - 2)

	// Update viewport dimensions
	v.activityViewport.Width = rightWidth - 2
	v.activityViewport.Height = availableHeight - 2

	leftPanel := connectionsStyle.Render(v.connectionsList.View())

	// Right panel content
	var rightContent string
	if v.selectedClient != "" {
		rightContent = v.activityViewport.View()
	} else {
		noSelectionStyle := lipgloss.NewStyle().
			Width(rightWidth-2).
			Height(availableHeight-2).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("245"))
		rightContent = noSelectionStyle.Render("Select a connection to view activity")
	}

	rightPanel := activityStyle.Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	content.WriteString(panels)

	return content.String()
}

// IsDebugMode returns whether debug mode is enabled
func (v *MCPDebugView) IsDebugMode() bool {
	return v.debugMode
}
