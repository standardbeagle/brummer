package components

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MinimalShell is a clean, simple component system shell for testing
type MinimalShell struct {
	width  int
	height int
	currentView string
	quit   bool
}

// NewMinimalShell creates a minimal working shell
func NewMinimalShell() *MinimalShell {
	return &MinimalShell{
		width:  80,
		height: 24,
		currentView: "processes",
	}
}

// Init initializes the shell
func (m *MinimalShell) Init() tea.Cmd {
	// Send initial window size
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: 80, Height: 24}
	}
}

// Update handles all messages
func (m *MinimalShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "1":
			m.currentView = "processes"
		case "2":
			m.currentView = "logs"
		case "3":
			m.currentView = "errors"
		case "4":
			m.currentView = "urls"
		case "5":
			m.currentView = "web"
		case "tab":
			// Cycle to next view
			views := []string{"processes", "logs", "errors", "urls", "web"}
			for i, v := range views {
				if v == m.currentView {
					m.currentView = views[(i+1)%len(views)]
					break
				}
			}
		case "shift+tab":
			// Cycle to previous view
			views := []string{"processes", "logs", "errors", "urls", "web"}
			for i, v := range views {
				if v == m.currentView {
					m.currentView = views[(i-1+len(views))%len(views)]
					break
				}
			}
		}
		return m, nil
	}
	
	return m, nil
}

// View renders the shell
func (m *MinimalShell) View() string {
	if m.quit {
		return "Goodbye!\n"
	}
	
	// Header
	header := m.renderHeader()
	
	// Main content area
	content := m.renderContent()
	
	// Footer
	footer := m.renderFooter()
	
	// Calculate heights
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight - 1 // -1 for safety
	
	// Ensure content fills the space
	// IMPORTANT: Width must be terminal width - 2 for the border characters
	contentStyle := lipgloss.NewStyle().
		Width(m.width - 2).  // Subtract 2 for left and right border
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	
	styledContent := contentStyle.Render(content)
	
	// Join all parts
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		styledContent,
		footer,
	)
}

func (m *MinimalShell) renderHeader() string {
	// Title line
	title := "🎯 Brummer Process Manager"
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Align(lipgloss.Center)
	
	// Tab indicators
	tabs := []string{"[1] Processes", "[2] Logs", "[3] Errors", "[4] URLs", "[5] Web"}
	viewNames := []string{"processes", "logs", "errors", "urls", "web"}
	
	var tabLine []string
	for i, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if viewNames[i] == m.currentView {
			// Active tab
			style = style.Bold(true).
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("236"))
		} else {
			// Inactive tab
			style = style.Foreground(lipgloss.Color("241"))
		}
		tabLine = append(tabLine, style.Render(tab))
	}
	
	tabsStr := lipgloss.JoinHorizontal(lipgloss.Top, tabLine...)
	tabsStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center)
	
	// Join title and tabs
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		tabsStyle.Render(tabsStr),
	)
}

func (m *MinimalShell) renderContent() string {
	// Render empty panel based on current view
	var content string
	
	switch m.currentView {
	case "processes":
		content = "Process View (Empty)\n\nNo processes running"
	case "logs":
		content = "Logs View (Empty)\n\nNo logs available"
	case "errors":
		content = "Errors View (Empty)\n\nNo errors detected"
	case "urls":
		content = "URLs View (Empty)\n\nNo URLs discovered"
	case "web":
		content = "Web View (Empty)\n\nNo web content"
	default:
		content = "Unknown view: " + m.currentView
	}
	
	// Add some debug info
	content += fmt.Sprintf("\n\nWindow: %dx%d", m.width, m.height)
	content += fmt.Sprintf("\nCurrent View: %s", m.currentView)
	
	return content
}

func (m *MinimalShell) renderFooter() string {
	help := "1-5: Switch views | q: Quit | Tab: Navigate"
	
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 1)
	
	return footerStyle.Render(help)
}