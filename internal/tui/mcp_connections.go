package tui

import (
	"fmt"
	"sort"
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
	m.mcpActivityMu.RLock()
	defer m.mcpActivityMu.RUnlock()

	items := make([]list.Item, 0, len(m.mcpConnections))
	
	// Convert connections to list items
	for _, conn := range m.mcpConnections {
		items = append(items, *conn)
	}

	// Sort by connection time (newest first)
	sort.Slice(items, func(i, j int) bool {
		connI := items[i].(mcpConnectionItem)
		connJ := items[j].(mcpConnectionItem)
		return connI.connectedAt.After(connJ.connectedAt)
	})

	m.mcpConnectionsList.SetItems(items)
}

// updateMCPActivityView updates the activity view for the selected client
func (m *Model) updateMCPActivityView() {
	if m.selectedMCPClient == "" {
		return
	}

	m.mcpActivityMu.RLock()
	defer m.mcpActivityMu.RUnlock()

	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	durationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	
	// Find the connection info
	conn, exists := m.mcpConnections[m.selectedMCPClient]
	if !exists {
		content.WriteString("Connection not found")
		m.mcpActivityViewport.SetContent(content.String())
		return
	}
	
	content.WriteString(titleStyle.Render(fmt.Sprintf("Activity Log - %s", conn.clientName)))
	content.WriteString("\n\n")

	// Get activities for this client
	activities, hasActivities := m.mcpActivities[m.selectedMCPClient]
	if !hasActivities || len(activities) == 0 {
		content.WriteString(timestampStyle.Render("No activity recorded yet"))
		m.mcpActivityViewport.SetContent(content.String())
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

	m.mcpActivityViewport.SetContent(content.String())
}