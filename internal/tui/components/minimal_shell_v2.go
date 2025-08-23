package components

import (
	"fmt"
	"strings"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MinimalShellV2 is an improved minimal shell using proper components
type MinimalShellV2 struct {
	BaseComponent
	
	// Current state
	currentView string
	views       map[string]Component
	quit        bool
	
	// Layout components
	mainLayout   *BoxLayout      // Vertical layout for header/content/footer
	header       Component
	contentArea  *BorderWindow   // Border window for main content
	viewSwitcher *ViewSwitcher   // Manages different views
	footer       Component
	
	// Tab bar
	tabBar       *TabBar
}

// TabBar component for view navigation
type TabBar struct {
	BaseComponent
	tabs        []string
	activeTab   int
	tabStyle    lipgloss.Style
	activeStyle lipgloss.Style
}

// NewTabBar creates a new tab bar
func NewTabBar(tabs []string) *TabBar {
	return &TabBar{
		BaseComponent: BaseComponent{visible: true},
		tabs:          tabs,
		activeTab:     0,
		tabStyle:      lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("241")),
		activeStyle:   lipgloss.NewStyle().Padding(0, 2).Bold(true).Foreground(lipgloss.Color("205")).Background(lipgloss.Color("235")),
	}
}

// SetActiveTab sets the active tab
func (t *TabBar) SetActiveTab(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeTab = index
	}
}

// GetActiveTab returns the active tab index
func (t *TabBar) GetActiveTab() int {
	return t.activeTab
}

// NextTab moves to next tab
func (t *TabBar) NextTab() {
	t.activeTab = (t.activeTab + 1) % len(t.tabs)
}

// PrevTab moves to previous tab
func (t *TabBar) PrevTab() {
	t.activeTab = (t.activeTab - 1 + len(t.tabs)) % len(t.tabs)
}

// Update handles tab bar updates
func (t *TabBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return t, nil
}

// View renders the tab bar
func (t *TabBar) View() string {
	if !t.visible {
		return ""
	}
	
	var tabs []string
	for i, tab := range t.tabs {
		style := t.tabStyle
		if i == t.activeTab {
			style = t.activeStyle
		}
		tabs = append(tabs, style.Render(fmt.Sprintf("[%d] %s", i+1, tab)))
	}
	
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// NewMinimalShellV2 creates an improved minimal shell
func NewMinimalShellV2() *MinimalShellV2 {
	shell := &MinimalShellV2{
		BaseComponent: BaseComponent{
			width:   80,
			height:  24,
			visible: true,
		},
		currentView: "processes",
		views:       make(map[string]Component),
	}
	
	// Initialize components
	shell.initializeComponents()
	
	return shell
}

// initializeComponents sets up all the components
func (m *MinimalShellV2) initializeComponents() {
	// Create main vertical layout
	m.mainLayout = NewBoxLayout(Vertical)
	m.mainLayout.SetSize(m.width, m.height)
	
	// Create header
	m.header = m.createHeader()
	
	// Create tab bar
	m.tabBar = NewTabBar([]string{"Processes", "Logs", "Errors", "URLs", "Web"})
	
	// Create view switcher with empty views
	m.viewSwitcher = NewViewSwitcher()
	m.createViews()
	
	// Create content area with border
	m.contentArea = NewBorderWindow(m.viewSwitcher)
	m.contentArea.SetBorderStyle(lipgloss.NormalBorder())
	m.contentArea.SetBorderColor(lipgloss.Color("240"))
	
	// Create footer
	m.footer = m.createFooter()
	
	// Add components to layout
	// Header: ~2 lines
	m.mainLayout.AddChild(m.header, 0)
	// Tab bar: ~1 line
	m.mainLayout.AddChild(m.tabBar, 0)
	// Content area: flexible (takes remaining space)
	m.mainLayout.AddChild(m.contentArea, 1.0)
	// Footer: ~1 line
	m.mainLayout.AddChild(m.footer, 0)
}

// createViews creates the empty view components
func (m *MinimalShellV2) createViews() {
	// Create simple text views for now
	views := map[string]string{
		"processes": "Process View\n\nNo processes running\n\nPress 's' to start a script",
		"logs":      "Logs View\n\nNo logs available\n\nStart a process to see logs",
		"errors":    "Errors View\n\nNo errors detected\n\n✓ All systems operational",
		"urls":      "URLs View\n\nNo URLs discovered\n\nURLs will appear when processes output them",
		"web":       "Web View\n\nNo web content\n\nPress 'o' to open a URL",
	}
	
	for name, content := range views {
		view := &SimpleTextView{
			BaseComponent: BaseComponent{visible: true},
			content:       content,
			name:          name,
		}
		m.views[name] = view
		m.viewSwitcher.AddView(name, view)
	}
	
	// Set initial view
	m.viewSwitcher.SwitchTo("processes")
}

// SimpleTextView is a basic text display component
type SimpleTextView struct {
	BaseComponent
	content  string
	name     string
	centered bool
	style    lipgloss.Style
}

func (s *SimpleTextView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s, nil
}

func (s *SimpleTextView) View() string {
	if !s.visible {
		return ""
	}
	
	content := s.content
	
	// Add debug info if we have a name
	if s.name != "" {
		debugInfo := fmt.Sprintf("\n\n---\nView: %s | Size: %dx%d", s.name, s.width, s.height)
		content = s.content + debugInfo
	}
	
	// Start with the component's style or create new
	style := s.style
	
	// Always set width to component width
	style = style.Width(s.width)
	
	// Center if requested
	if s.centered {
		style = style.Align(lipgloss.Center)
	}
	
	// Only set height and padding for content views
	if s.name != "" {
		style = style.Height(s.height).Padding(1)
	}
	
	return style.Render(content)
}

// createHeader creates the header component
func (m *MinimalShellV2) createHeader() Component {
	header := &SimpleTextView{
		BaseComponent: BaseComponent{
			visible: true,
			width:   m.width,
			height:  1,
		},
		content: "🎯 Brummer Process Manager",
	}
	header.centered = true
	header.style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("235"))
	return header
}

// createFooter creates the footer component
func (m *MinimalShellV2) createFooter() Component {
	help := "Tab/Shift+Tab: Navigate | 1-5: Jump to view | q: Quit | ?: Help"
	footer := &SimpleTextView{
		BaseComponent: BaseComponent{
			visible: true,
			width:   m.width,
			height:  1,
		},
		content: help,
	}
	footer.centered = true
	footer.style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235"))
	return footer
}

// Init initializes the shell
func (m *MinimalShellV2) Init() tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

// Update handles all messages
func (m *MinimalShellV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainLayout.SetSize(msg.Width, msg.Height)
		return m, nil
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
			
		case "tab":
			// Next tab
			m.tabBar.NextTab()
			m.switchToTab(m.tabBar.GetActiveTab())
			return m, nil
			
		case "shift+tab":
			// Previous tab
			m.tabBar.PrevTab()
			m.switchToTab(m.tabBar.GetActiveTab())
			return m, nil
			
		case "1", "2", "3", "4", "5":
			// Direct tab selection
			if index := int(msg.String()[0] - '1'); index >= 0 && index < 5 {
				m.tabBar.SetActiveTab(index)
				m.switchToTab(index)
			}
			return m, nil
		}
	}
	
	// Update main layout (which propagates to all children)
	if m.mainLayout != nil {
		newLayout, cmd := m.mainLayout.Update(msg)
		m.mainLayout = newLayout.(*BoxLayout)
		return m, cmd
	}
	
	return m, nil
}

// switchToTab switches to the specified tab index
func (m *MinimalShellV2) switchToTab(index int) {
	views := []string{"processes", "logs", "errors", "urls", "web"}
	if index >= 0 && index < len(views) {
		m.currentView = views[index]
		m.viewSwitcher.SwitchTo(m.currentView)
	}
}

// View renders the shell
func (m *MinimalShellV2) View() string {
	if m.quit {
		return "Goodbye!\n"
	}
	
	if m.mainLayout != nil {
		return m.mainLayout.View()
	}
	
	return "Initializing..."
}

// Utility function to center text
func centerText(text string, width int) string {
	if width <= 0 {
		return text
	}
	textLen := len(strings.TrimSpace(text))
	if textLen >= width {
		return text
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text
}