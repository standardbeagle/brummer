package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// LayoutController manages the overall TUI layout and core rendering
type LayoutController struct {
	// Dependencies
	processMgr  *process.Manager
	logStore    *logs.Store
	mcpServer   MCPServerInterface
	proxyServer *proxy.Server
	version     string
	workingDir  string

	// View state
	width             int
	height            int
	headerHeight      int
	footerHeight      int
	systemPanelHeight int
	showHelp          bool
	currentView       string
	systemPanelOpen   bool
	selectedProcess   string

	// Styles (initialized once)
	headerStyle          lipgloss.Style
	processSelectorStyle lipgloss.Style
	systemPanelStyle     lipgloss.Style
	helpStyle            lipgloss.Style
}

// NewLayoutController creates a new layout controller
func NewLayoutController(processMgr *process.Manager, logStore *logs.Store, mcpServer MCPServerInterface, proxyServer *proxy.Server, version, workingDir string) *LayoutController {
	lc := &LayoutController{
		processMgr:        processMgr,
		logStore:          logStore,
		mcpServer:         mcpServer,
		proxyServer:       proxyServer,
		version:           version,
		workingDir:        workingDir,
		headerHeight:      3,
		footerHeight:      1,
		systemPanelHeight: 10,
	}

	// Initialize styles
	lc.initStyles()

	return lc
}

// initStyles initializes the lipgloss styles
func (lc *LayoutController) initStyles() {
	lc.headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("53"))

	lc.processSelectorStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	lc.systemPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	lc.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

// UpdateSize updates the viewport dimensions
func (lc *LayoutController) UpdateSize(width, height int) {
	lc.width = width
	lc.height = height
}

// SetShowHelp sets whether to show help
func (lc *LayoutController) SetShowHelp(show bool) {
	lc.showHelp = show
}

// SetCurrentView sets the current view name
func (lc *LayoutController) SetCurrentView(view string) {
	lc.currentView = view
}

// SetSystemPanelOpen sets whether the system panel is open
func (lc *LayoutController) SetSystemPanelOpen(open bool) {
	lc.systemPanelOpen = open
}

// SetSelectedProcess sets the selected process ID
func (lc *LayoutController) SetSelectedProcess(processID string) {
	lc.selectedProcess = processID
}

// GetHeaderHeight returns the header height
func (lc *LayoutController) GetHeaderHeight() int {
	return lc.headerHeight
}

// GetFooterHeight returns the footer height
func (lc *LayoutController) GetFooterHeight() int {
	return lc.footerHeight
}

// GetSystemPanelHeight returns the system panel height
func (lc *LayoutController) GetSystemPanelHeight() int {
	if lc.systemPanelOpen {
		return lc.systemPanelHeight
	}
	return 0
}

// Note: RenderHeader is not used directly - the model.go renderHeader method
// handles the complex header rendering with notifications and unread indicators

// RenderFooter renders the application footer
func (lc *LayoutController) RenderFooter() string {
	helpKeys := []string{
		"Tab/←→: Switch View",
		"↑↓: Navigate",
		"Enter: Select",
		"q: Quit",
		"?: Help",
	}

	if lc.currentView == "Processes" {
		helpKeys = append([]string{"Space: Stop Process"}, helpKeys...)
	} else if lc.currentView == "Logs" {
		helpKeys = append([]string{"f: Filter", "c: Clear", "a: Auto-scroll"}, helpKeys...)
	}

	help := strings.Join(helpKeys, " • ")

	// Create footer with top border
	footerStyle := lc.helpStyle.
		Width(lc.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color("240"))

	return footerStyle.Render(help)
}

// RenderSystemPanel renders the system panel
func (lc *LayoutController) RenderSystemPanel() string {
	if !lc.systemPanelOpen {
		return ""
	}

	// Get system logs
	systemLogs := lc.logStore.GetByProcess("system")

	// Take last few logs that fit in panel
	maxLogs := lc.systemPanelHeight - 2 // Account for border
	startIdx := 0
	if len(systemLogs) > maxLogs {
		startIdx = len(systemLogs) - maxLogs
	}

	var content strings.Builder
	for _, log := range systemLogs[startIdx:] {
		style := lipgloss.NewStyle()
		if log.IsError {
			style = style.Foreground(lipgloss.Color("196"))
		}
		content.WriteString(style.Render(log.Content))
		content.WriteString("\n")
	}

	return lc.systemPanelStyle.
		Width(lc.width - 2).
		Height(lc.systemPanelHeight).
		Render(content.String())
}

// RenderHelpBar renders the help bar overlay
func (lc *LayoutController) RenderHelpBar() string {
	if !lc.showHelp {
		return ""
	}

	helpContent := `🔨 Brummer Help

Navigation:
  Tab, ←/→    Switch between views
  ↑/↓         Navigate lists
  Enter       Select item
  q           Quit application
  ?           Toggle this help

Process View:
  Space       Stop selected process
  r           Restart stopped process
  
Logs View:
  f           Filter logs
  c           Clear filter
  a           Toggle auto-scroll
  /           Search in logs
  
URLs View:
  Enter       Copy URL to clipboard
  o           Open URL in browser
  
Settings:
  Enter       Install/Copy selected item
  
Web View:
  ↑/↓         Navigate requests
  Enter       View request details
  Esc         Back to list
  
Press ? to close this help`

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60).
		Height(25).
		Render(helpContent)

	// Center the help box
	return lipgloss.Place(lc.width, lc.height, lipgloss.Center, lipgloss.Center, helpBox)
}

// RenderProcessSelector renders the process selector for logs view
func (lc *LayoutController) RenderProcessSelector() string {
	processes := lc.processMgr.GetAllProcesses()
	items := make([]string, 0, len(processes)+1)

	// Add "All" option
	allStyle := lc.processSelectorStyle
	if lc.selectedProcess == "" {
		allStyle = allStyle.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("229"))
	}
	items = append(items, allStyle.Render("All"))

	// Add process options
	for _, p := range processes {
		style := lc.processSelectorStyle
		if p.ID == lc.selectedProcess {
			style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("229"))
		}

		emoji := "🟢"
		if p.GetStatus() != process.StatusRunning {
			emoji = "🔴"
		}

		items = append(items, style.Render(fmt.Sprintf("%s %s", emoji, p.Name)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, items...)
}

// GetAvailableHeight calculates the available height for content
func (lc *LayoutController) GetAvailableHeight() int {
	totalHeight := lc.height
	usedHeight := lc.headerHeight + lc.footerHeight

	if lc.systemPanelOpen {
		usedHeight += lc.systemPanelHeight
	}

	availableHeight := totalHeight - usedHeight
	if availableHeight < 1 {
		availableHeight = 1
	}

	return availableHeight
}

// RenderMainView renders the complete view with layout
// Note: This is not currently used as the model.go handles the main layout
func (lc *LayoutController) RenderMainView(content string) string {
	var sections []string

	// Main content
	sections = append(sections, content)

	// System panel (if open)
	if lc.systemPanelOpen {
		sections = append(sections, lc.RenderSystemPanel())
	}

	// Footer
	sections = append(sections, lc.RenderFooter())

	// Combine all sections
	mainView := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Overlay help if shown
	if lc.showHelp {
		return lc.RenderHelpBar()
	}

	return mainView
}

// UpdateSizes recalculates component sizes based on terminal dimensions
func (lc *LayoutController) UpdateSizes(termWidth, termHeight int) {
	lc.width = termWidth
	lc.height = termHeight

	// Adjust system panel height if needed
	maxSystemPanelHeight := termHeight / 3
	if lc.systemPanelHeight > maxSystemPanelHeight {
		lc.systemPanelHeight = maxSystemPanelHeight
	}
}
