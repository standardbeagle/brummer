package tui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// mcpActivityMsg represents MCP activity message
type mcpActivityMsg struct {
	sessionId string
	activity  MCPActivity
}

// mcpConnectionMsg represents MCP connection message
type mcpConnectionMsg struct {
	sessionId      string
	clientInfo     string
	connected      bool
	connectedAt    time.Time
	connectionType string
	method         string
}

// MCPDebugController manages MCP debug view state and functionality
type MCPDebugController struct {
	// View components
	connectionsList  list.Model                    // List of MCP connections
	activityViewport viewport.Model                // Activity log for selected connection
	selectedClient   string                        // Selected MCP client ID
	connections      map[string]*mcpConnectionItem // sessionId -> connection
	activities       map[string][]MCPActivity      // sessionId -> activities
	mu               sync.RWMutex                  // Protects both connections and activities maps

	// Memory management
	maxConnections       int           // Maximum number of connections to keep
	maxActivitiesPerConn int           // Maximum activities per connection
	cleanupInterval      time.Duration // How often to cleanup old data
	lastCleanup          time.Time     // Last cleanup timestamp

	// Dependencies
	debugMode    bool
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewMCPDebugController creates a new MCP debug controller
func NewMCPDebugController(debugMode bool) *MCPDebugController {
	if !debugMode {
		return &MCPDebugController{debugMode: false}
	}

	// Initialize MCP connections list
	connectionsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	connectionsList.Title = "MCP Connections"
	connectionsList.SetShowStatusBar(false)

	return &MCPDebugController{
		connectionsList:  connectionsList,
		activityViewport: viewport.New(0, 0),
		connections:      make(map[string]*mcpConnectionItem),
		activities:       make(map[string][]MCPActivity),
		debugMode:        debugMode,
		// Memory management settings
		maxConnections:       100,         // Keep up to 100 connections
		maxActivitiesPerConn: 100,         // Keep up to 100 activities per connection (already implemented)
		cleanupInterval:      time.Minute, // Cleanup every minute
		lastCleanup:          time.Now(),
	}
}

// UpdateSize updates the controller dimensions
func (c *MCPDebugController) UpdateSize(width, height, headerHeight, footerHeight int) {
	if !c.debugMode {
		return
	}

	c.width = width
	c.height = height
	c.headerHeight = headerHeight
	c.footerHeight = footerHeight
}

// GetConnectionsList returns the connections list for direct manipulation
func (c *MCPDebugController) GetConnectionsList() *list.Model {
	return &c.connectionsList
}

// GetActivityViewport returns the activity viewport for direct manipulation
func (c *MCPDebugController) GetActivityViewport() *viewport.Model {
	return &c.activityViewport
}

// SetSelectedClient sets the currently selected MCP client
func (c *MCPDebugController) SetSelectedClient(clientID string) {
	c.selectedClient = clientID
	c.UpdateActivityView()
}

// GetSelectedClient returns the currently selected MCP client
func (c *MCPDebugController) GetSelectedClient() string {
	return c.selectedClient
}

// GetMemoryStats returns current memory usage statistics
func (c *MCPDebugController) GetMemoryStats() map[string]interface{} {
	if !c.debugMode {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	totalActivities := 0
	for _, activities := range c.activities {
		totalActivities += len(activities)
	}

	return map[string]interface{}{
		"connections":          len(c.connections),
		"maxConnections":       c.maxConnections,
		"totalActivities":      totalActivities,
		"maxActivitiesPerConn": c.maxActivitiesPerConn,
		"lastCleanup":          c.lastCleanup,
		"cleanupInterval":      c.cleanupInterval,
		"memoryUsagePercent":   float64(len(c.connections)) / float64(c.maxConnections) * 100,
	}
}

// cleanupOldData removes old connections and activities to manage memory usage
func (c *MCPDebugController) cleanupOldData() {
	if !c.debugMode {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Only cleanup if enough time has passed
	if now.Sub(c.lastCleanup) < c.cleanupInterval {
		return
	}
	c.lastCleanup = now

	// If we're under the connection limit, no need to cleanup
	if len(c.connections) <= c.maxConnections {
		return
	}

	// Create a slice of connections with their last activity times for sorting
	type connWithTime struct {
		sessionId    string
		lastActivity time.Time
		isConnected  bool
	}

	var connsWithTime []connWithTime
	for sessionId, conn := range c.connections {
		connsWithTime = append(connsWithTime, connWithTime{
			sessionId:    sessionId,
			lastActivity: conn.lastActivity,
			isConnected:  conn.isConnected,
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
	connectionsToRemove := len(c.connections) - c.maxConnections
	if connectionsToRemove > 0 {
		for i := 0; i < connectionsToRemove; i++ {
			sessionId := connsWithTime[i].sessionId
			delete(c.connections, sessionId)
			delete(c.activities, sessionId)

			// If we're removing the selected client, clear the selection
			if c.selectedClient == sessionId {
				c.selectedClient = ""
			}
		}
	}
}

// performMemoryManagement checks if cleanup is needed and performs it
func (c *MCPDebugController) performMemoryManagement() {
	if !c.debugMode {
		return
	}

	// Check if cleanup is needed (non-blocking check)
	c.mu.RLock()
	needsCleanup := len(c.connections) > c.maxConnections ||
		time.Since(c.lastCleanup) > c.cleanupInterval
	c.mu.RUnlock()

	if needsCleanup {
		c.cleanupOldData()
	}
}

// HandleConnection handles MCP connection events
func (c *MCPDebugController) HandleConnection(msg mcpConnectionMsg) {
	if !c.debugMode {
		return
	}

	// Perform memory management check before processing
	c.performMemoryManagement()

	c.mu.Lock()
	defer c.mu.Unlock()

	if msg.connected {
		// Determine client name based on client info
		clientName := "Unknown Client"
		if strings.Contains(msg.clientInfo, "Claude Code") {
			clientName = "Claude Code"
		} else if strings.Contains(msg.clientInfo, "VS Code") {
			clientName = "VS Code MCP"
		} else if msg.clientInfo != "" {
			if len(msg.clientInfo) > 20 {
				clientName = msg.clientInfo[:17] + "..."
			} else {
				clientName = msg.clientInfo
			}
		}

		c.connections[msg.sessionId] = &mcpConnectionItem{
			clientID:       msg.sessionId,
			clientName:     clientName,
			connectedAt:    msg.connectedAt,
			lastActivity:   msg.connectedAt,
			requestCount:   0,
			isConnected:    true,
			connectionType: msg.connectionType,
			method:         msg.method,
		}
		c.activities[msg.sessionId] = []MCPActivity{}
	} else {
		// Mark as disconnected
		if conn, exists := c.connections[msg.sessionId]; exists {
			conn.isConnected = false
			c.connections[msg.sessionId] = conn
		}
	}
}

// HandleActivity handles MCP activity events
func (c *MCPDebugController) HandleActivity(msg mcpActivityMsg) {
	if !c.debugMode {
		return
	}

	// Perform memory management check before processing
	c.performMemoryManagement()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update connection info if exists
	if conn, exists := c.connections[msg.sessionId]; exists {
		conn.lastActivity = msg.activity.Timestamp
		conn.requestCount++
		c.connections[msg.sessionId] = conn
	} else {
		// Create a connection entry for sessions that only have activity (e.g., POST requests)
		// This ensures all sessions are tracked even if they don't establish persistent connections
		c.connections[msg.sessionId] = &mcpConnectionItem{
			clientID:       msg.sessionId,
			clientName:     "HTTP Client",
			connectedAt:    msg.activity.Timestamp,
			lastActivity:   msg.activity.Timestamp,
			requestCount:   1,
			isConnected:    false, // HTTP-only sessions are not "connected" in the persistent sense
			connectionType: "HTTP",
			method:         "POST", // Default for activity-only sessions
		}
		c.activities[msg.sessionId] = []MCPActivity{}
	}

	// Add activity to the list
	activities := c.activities[msg.sessionId]

	// Limit activities to prevent memory issues
	if len(activities) >= c.maxActivitiesPerConn {
		activities = activities[1:] // Remove oldest
	}

	activities = append(activities, msg.activity)
	c.activities[msg.sessionId] = activities
}

// UpdateConnectionsList updates the list of MCP connections
func (c *MCPDebugController) UpdateConnectionsList() {
	if !c.debugMode {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make([]list.Item, 0, len(c.connections))

	// Convert connections to list items
	for _, conn := range c.connections {
		items = append(items, *conn)
	}

	// Sort by connection time (newest first)
	sort.Slice(items, func(i, j int) bool {
		connI := items[i].(mcpConnectionItem)
		connJ := items[j].(mcpConnectionItem)
		return connI.connectedAt.After(connJ.connectedAt)
	})

	c.connectionsList.SetItems(items)
}

// UpdateActivityView updates the activity view for the selected client
func (c *MCPDebugController) UpdateActivityView() {
	if !c.debugMode || c.selectedClient == "" {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	durationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Find the connection info
	conn, exists := c.connections[c.selectedClient]
	if !exists {
		content.WriteString("Connection not found")
		c.activityViewport.SetContent(content.String())
		return
	}

	content.WriteString(titleStyle.Render(fmt.Sprintf("Activity Log - %s", conn.clientName)))
	content.WriteString("\n\n")

	// Get activities for this client
	activities, hasActivities := c.activities[c.selectedClient]
	if !hasActivities || len(activities) == 0 {
		content.WriteString(timestampStyle.Render("No activity recorded yet"))
		c.activityViewport.SetContent(content.String())
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

	c.activityViewport.SetContent(content.String())
}

// Render renders the MCP connections view
func (c *MCPDebugController) Render(width, height, headerHeight, footerHeight int) string {
	if !c.debugMode {
		return ""
	}

	// Update dimensions if needed
	if c.width != width || c.height != height {
		c.UpdateSize(width, height, headerHeight, footerHeight)
	}

	var content strings.Builder

	// Header with branding and description
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1).
		Width(width)

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
	headerHeightLocal := 4                            // header + subtitle + margins
	availableHeight := height - 6 - headerHeightLocal // standard layout minus our header

	// Create left panel (connections list)
	leftWidth := width / 3
	rightWidth := width - leftWidth - 1

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
	c.connectionsList.SetWidth(leftWidth - 2)
	c.connectionsList.SetHeight(availableHeight - 2)

	// Update viewport dimensions
	c.activityViewport.Width = rightWidth - 2
	c.activityViewport.Height = availableHeight - 2

	leftPanel := connectionsStyle.Render(c.connectionsList.View())

	// Right panel content
	var rightContent string
	if c.selectedClient != "" {
		rightContent = c.activityViewport.View()
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
func (c *MCPDebugController) IsDebugMode() bool {
	return c.debugMode
}
