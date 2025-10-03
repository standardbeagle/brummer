package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TabsComponent displays a tab bar with icons, badges, and selection indicator.
//
// Design principles:
// - Shows all available views
// - Visual indicator for active tab
// - Icons for each view type
// - Optional badges (counts, status)
// - Keyboard shortcuts visible
type TabsComponent struct {
	BaseComponent

	tabs          []TabInfo
	activeIndex   int
	showShortcuts bool

	// Styles
	activeTabStyle   lipgloss.Style
	inactiveTabStyle lipgloss.Style
	badgeStyle       lipgloss.Style
}

// TabInfo holds display information for a tab
type TabInfo struct {
	ID       string
	Title    string
	Icon     string
	Shortcut string
	Badge    string // Optional count/status badge
	Enabled  bool
}

// NewTabsComponent creates a tab bar component
func NewTabsComponent() *TabsComponent {
	c := &TabsComponent{
		BaseComponent: NewBaseComponent(),
		activeIndex:   0,
		showShortcuts: true,
		tabs: []TabInfo{
			{ID: "logs", Title: "Logs", Icon: "📋", Shortcut: "1", Enabled: true},
			{ID: "processes", Title: "Processes", Icon: "⚙️", Shortcut: "2", Enabled: true},
			{ID: "urls", Title: "URLs", Icon: "🔗", Shortcut: "3", Enabled: true},
			{ID: "errors", Title: "Errors", Icon: "⚠️", Shortcut: "4", Enabled: true},
			{ID: "web", Title: "Web", Icon: "🌐", Shortcut: "5", Enabled: true},
			{ID: "aicoders", Title: "AI Coders", Icon: "🤖", Shortcut: "6", Enabled: true},
			{ID: "mcp", Title: "MCP", Icon: "🔌", Shortcut: "7", Enabled: true},
			{ID: "settings", Title: "Settings", Icon: "⚙", Shortcut: "8", Enabled: true},
			{ID: "diagnostic", Title: "Diagnostic", Icon: "🔍", Shortcut: "9", Enabled: true},
		},
		activeTabStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")). // Bright Blue
			Background(lipgloss.Color("236")). // Dark Gray
			Padding(0, 1),
		inactiveTabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")). // Light Gray
			Padding(0, 1),
		badgeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Bright Yellow
			Bold(true),
	}

	return c
}

// Init initializes the component
func (c *TabsComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *TabsComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil
	}

	return c, nil
}

// View renders the tab bar
func (c *TabsComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	var tabs []string

	for i, tab := range c.tabs {
		if !tab.Enabled {
			continue
		}

		// Build tab content
		var parts []string

		// Icon
		parts = append(parts, tab.Icon)

		// Title
		parts = append(parts, tab.Title)

		// Badge (if present)
		if tab.Badge != "" {
			badge := c.badgeStyle.Render(fmt.Sprintf("[%s]", tab.Badge))
			parts = append(parts, badge)
		}

		// Shortcut (if showing)
		if c.showShortcuts {
			parts = append(parts, fmt.Sprintf("(%s)", tab.Shortcut))
		}

		tabContent := strings.Join(parts, " ")

		// Apply style based on active state
		if i == c.activeIndex {
			tabs = append(tabs, c.activeTabStyle.Render(tabContent))
		} else {
			tabs = append(tabs, c.inactiveTabStyle.Render(tabContent))
		}
	}

	// Join tabs with separator
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(" │ ")

	return strings.Join(tabs, separator)
}

// SetSize updates component dimensions
func (c *TabsComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
}

// SetActiveTab sets the active tab by ID
func (c *TabsComponent) SetActiveTab(tabID string) {
	for i, tab := range c.tabs {
		if tab.ID == tabID {
			c.activeIndex = i
			break
		}
	}
}

// SetActiveTabByIndex sets the active tab by index
func (c *TabsComponent) SetActiveTabByIndex(index int) {
	if index >= 0 && index < len(c.tabs) {
		c.activeIndex = index
	}
}

// GetActiveTab returns the current active tab ID
func (c *TabsComponent) GetActiveTab() string {
	if c.activeIndex >= 0 && c.activeIndex < len(c.tabs) {
		return c.tabs[c.activeIndex].ID
	}
	return ""
}

// SetTabBadge sets a badge for a specific tab
func (c *TabsComponent) SetTabBadge(tabID string, badge string) {
	for i := range c.tabs {
		if c.tabs[i].ID == tabID {
			c.tabs[i].Badge = badge
			break
		}
	}
}

// ClearTabBadge removes the badge from a tab
func (c *TabsComponent) ClearTabBadge(tabID string) {
	c.SetTabBadge(tabID, "")
}

// SetShowShortcuts toggles shortcut display
func (c *TabsComponent) SetShowShortcuts(show bool) {
	c.showShortcuts = show
}

// EnableTab enables a tab by ID
func (c *TabsComponent) EnableTab(tabID string) {
	for i := range c.tabs {
		if c.tabs[i].ID == tabID {
			c.tabs[i].Enabled = true
			break
		}
	}
}

// DisableTab disables a tab by ID
func (c *TabsComponent) DisableTab(tabID string) {
	for i := range c.tabs {
		if c.tabs[i].ID == tabID {
			c.tabs[i].Enabled = false
			break
		}
	}
}
