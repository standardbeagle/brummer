package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// mcpConnectionItem represents an MCP client connection
type mcpConnectionItem struct {
	clientID      string
	clientName    string
	connectedAt   time.Time
	lastActivity  time.Time
	requestCount  int
	isConnected   bool
}

func (i mcpConnectionItem) Title() string {
	status := "🟢"
	if !i.isConnected {
		status = "🔴"
	}
	return fmt.Sprintf("%s %s", status, i.clientName)
}

func (i mcpConnectionItem) Description() string {
	duration := time.Since(i.connectedAt).Round(time.Second)
	return fmt.Sprintf("ID: %s | Connected: %s | Requests: %d", i.clientID, duration, i.requestCount)
}

func (i mcpConnectionItem) FilterValue() string {
	return i.clientName + " " + i.clientID
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

// renderMCPConnections renders the MCP connections view
func (m Model) renderMCPConnections() string {
	if !m.debugMode {
		return m.renderSettings()
	}

	var content strings.Builder

	// Header with branding and description
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1).
		Width(m.width)

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
	headerHeight := 4 // header + subtitle + margins
	availableHeight := m.height - 6 - headerHeight // standard layout minus our header

	// Create left panel (connections list)
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 1

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
	m.mcpConnectionsList.SetWidth(leftWidth - 2)
	m.mcpConnectionsList.SetHeight(availableHeight - 2)

	// Update viewport dimensions
	m.mcpActivityViewport.Width = rightWidth - 2
	m.mcpActivityViewport.Height = availableHeight - 2

	leftPanel := connectionsStyle.Render(m.mcpConnectionsList.View())
	
	// Right panel content
	var rightContent string
	if m.selectedMCPClient != "" {
		rightContent = m.mcpActivityViewport.View()
	} else {
		noSelectionStyle := lipgloss.NewStyle().
			Width(rightWidth - 2).
			Height(availableHeight - 2).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("245"))
		rightContent = noSelectionStyle.Render("Select a connection to view activity")
	}
	
	rightPanel := activityStyle.Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	content.WriteString(panels)

	return content.String()
}

// updateMCPConnectionsList updates the list of MCP connections
func (m *Model) updateMCPConnectionsList() {
	// This would be called when MCP connections change
	// For now, we'll use mock data
	items := []list.Item{
		mcpConnectionItem{
			clientID:     "client-1",
			clientName:   "Claude Desktop",
			connectedAt:  time.Now().Add(-5 * time.Minute),
			lastActivity: time.Now().Add(-30 * time.Second),
			requestCount: 42,
			isConnected:  true,
		},
		mcpConnectionItem{
			clientID:     "client-2", 
			clientName:   "VS Code MCP",
			connectedAt:  time.Now().Add(-2 * time.Hour),
			lastActivity: time.Now().Add(-5 * time.Minute),
			requestCount: 156,
			isConnected:  true,
		},
		mcpConnectionItem{
			clientID:     "client-3",
			clientName:   "Test Client",
			connectedAt:  time.Now().Add(-1 * time.Hour),
			lastActivity: time.Now().Add(-45 * time.Minute),
			requestCount: 12,
			isConnected:  false,
		},
	}

	m.mcpConnectionsList.SetItems(items)
}

// updateMCPActivityView updates the activity view for the selected client
func (m *Model) updateMCPActivityView() {
	if m.selectedMCPClient == "" {
		return
	}

	// This would fetch real activity data for the selected client
	// For now, we'll use mock data
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	
	content.WriteString(titleStyle.Render(fmt.Sprintf("Activity Log - %s", m.selectedMCPClient)))
	content.WriteString("\n\n")

	// Mock activity entries
	activities := []MCPActivity{
		{
			Timestamp: time.Now().Add(-30 * time.Second),
			Method:    "tools/list",
			Params:    "{}",
			Response:  `{"tools": [...]}`,
			Duration:  15 * time.Millisecond,
		},
		{
			Timestamp: time.Now().Add(-25 * time.Second),
			Method:    "tools/call",
			Params:    `{"name": "scripts/list"}`,
			Response:  `{"scripts": {"dev": "npm run dev", "test": "npm test"}}`,
			Duration:  23 * time.Millisecond,
		},
		{
			Timestamp: time.Now().Add(-20 * time.Second),
			Method:    "tools/call",
			Params:    `{"name": "scripts/run", "arguments": {"script": "dev"}}`,
			Response:  `{"started": true, "pid": "dev-1234"}`,
			Duration:  45 * time.Millisecond,
		},
		{
			Timestamp: time.Now().Add(-15 * time.Second),
			Method:    "resources/list",
			Params:    "{}",
			Response:  `{"resources": [...]}`,
			Duration:  12 * time.Millisecond,
		},
		{
			Timestamp: time.Now().Add(-10 * time.Second),
			Method:    "resources/read",
			Params:    `{"uri": "logs://recent"}`,
			Error:     "Resource not found",
			Duration:  5 * time.Millisecond,
		},
	}

	for _, activity := range activities {
		content.WriteString(timestampStyle.Render(activity.Timestamp.Format("15:04:05")))
		content.WriteString(" ")
		content.WriteString(methodStyle.Render(activity.Method))
		content.WriteString(fmt.Sprintf(" (%dms)\n", activity.Duration.Milliseconds()))
		
		if activity.Params != "{}" {
			content.WriteString(fmt.Sprintf("  Params: %s\n", activity.Params))
		}
		
		if activity.Error != "" {
			content.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s\n", activity.Error)))
		} else if activity.Response != "" {
			response := activity.Response
			if len(response) > 100 {
				response = response[:97] + "..."
			}
			content.WriteString(fmt.Sprintf("  Response: %s\n", response))
		}
		
		content.WriteString("\n")
	}

	m.mcpActivityViewport.SetContent(content.String())
}