package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FooterComponent displays keyboard shortcuts and help information.
//
// Design principles:
// - Context-aware shortcuts (changes per view)
// - Compact, single-line display
// - Help hint always visible
// - Clear visual separation
type FooterComponent struct {
	BaseComponent

	globalShortcuts  []Shortcut
	contextShortcuts []Shortcut
	showHelp         bool

	// Styles
	keyStyle    lipgloss.Style
	labelStyle  lipgloss.Style
	helpStyle   lipgloss.Style
	dividerStyle lipgloss.Style
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Key   string
	Label string
}

// NewFooterComponent creates a footer shortcuts component
func NewFooterComponent() *FooterComponent {
	c := &FooterComponent{
		BaseComponent: NewBaseComponent(),
		showHelp:      true,
		globalShortcuts: []Shortcut{
			{Key: "1-9", Label: "Views"},
			{Key: "?", Label: "Help"},
			{Key: "q", Label: "Quit"},
		},
		contextShortcuts: []Shortcut{},
		keyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Bright Blue
			Bold(true),
		labelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Dark Gray
			Italic(true),
		dividerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			SetString(" • "),
	}

	return c
}

// Init initializes the component
func (c *FooterComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *FooterComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil
	}

	return c, nil
}

// View renders the footer
func (c *FooterComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	var parts []string

	// Context-specific shortcuts first
	for _, shortcut := range c.contextShortcuts {
		formatted := c.formatShortcut(shortcut)
		parts = append(parts, formatted)
	}

	// Global shortcuts
	for _, shortcut := range c.globalShortcuts {
		formatted := c.formatShortcut(shortcut)
		parts = append(parts, formatted)
	}

	// Help hint
	if c.showHelp {
		helpText := c.helpStyle.Render("Press ? for help")
		parts = append(parts, helpText)
	}

	// Join with dividers
	result := strings.Join(parts, c.dividerStyle.String())

	// Pad to full width if needed
	if c.width > 0 && len(result) < c.width {
		padding := strings.Repeat(" ", c.width-len(result))
		result = result + padding
	}

	return result
}

// formatShortcut formats a single shortcut
func (c *FooterComponent) formatShortcut(shortcut Shortcut) string {
	key := c.keyStyle.Render(shortcut.Key)
	label := c.labelStyle.Render(shortcut.Label)
	return fmt.Sprintf("%s %s", key, label)
}

// SetSize updates component dimensions
func (c *FooterComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
}

// SetContextShortcuts sets view-specific shortcuts
func (c *FooterComponent) SetContextShortcuts(shortcuts []Shortcut) {
	c.contextShortcuts = shortcuts
}

// ClearContextShortcuts removes all context shortcuts
func (c *FooterComponent) ClearContextShortcuts() {
	c.contextShortcuts = []Shortcut{}
}

// AddContextShortcut adds a single context shortcut
func (c *FooterComponent) AddContextShortcut(key, label string) {
	c.contextShortcuts = append(c.contextShortcuts, Shortcut{Key: key, Label: label})
}

// SetGlobalShortcuts sets always-visible shortcuts
func (c *FooterComponent) SetGlobalShortcuts(shortcuts []Shortcut) {
	c.globalShortcuts = shortcuts
}

// SetShowHelp toggles help hint visibility
func (c *FooterComponent) SetShowHelp(show bool) {
	c.showHelp = show
}

// GetDefaultLogsShortcuts returns default shortcuts for logs view
func GetDefaultLogsShortcuts() []Shortcut {
	return []Shortcut{
		{Key: "↑/↓", Label: "Scroll"},
		{Key: "/", Label: "Filter"},
		{Key: "c", Label: "Clear"},
	}
}

// GetDefaultProcessesShortcuts returns default shortcuts for processes view
func GetDefaultProcessesShortcuts() []Shortcut {
	return []Shortcut{
		{Key: "↑/↓", Label: "Select"},
		{Key: "r", Label: "Restart"},
		{Key: "s", Label: "Stop"},
		{Key: "Enter", Label: "Details"},
	}
}

// GetDefaultErrorsShortcuts returns default shortcuts for errors view
func GetDefaultErrorsShortcuts() []Shortcut {
	return []Shortcut{
		{Key: "j/k", Label: "Navigate"},
		{Key: "Enter", Label: "Details"},
		{Key: "c", Label: "Clear"},
	}
}

// GetDefaultAICodersShortcuts returns default shortcuts for AI coders view
func GetDefaultAICodersShortcuts() []Shortcut {
	return []Shortcut{
		{Key: "j/k", Label: "Navigate"},
		{Key: "Enter", Label: "Attach"},
		{Key: "a", Label: "New"},
		{Key: "d", Label: "Detach"},
	}
}
