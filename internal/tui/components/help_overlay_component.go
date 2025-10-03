package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// HelpOverlayComponent displays comprehensive keyboard shortcuts help.
//
// Design principles:
// - Full-screen overlay
// - Organized by category
// - Scrollable content
// - Clear visual hierarchy
// - Context-aware help per view
type HelpOverlayComponent struct {
	BaseComponent

	viewport      viewport.Model
	active        bool
	currentView   string
	helpSections  []HelpSection

	// Styles
	overlayStyle   lipgloss.Style
	titleStyle     lipgloss.Style
	sectionStyle   lipgloss.Style
	keyStyle       lipgloss.Style
	descStyle      lipgloss.Style
	borderStyle    lipgloss.Style
}

// HelpSection represents a category of shortcuts
type HelpSection struct {
	Title     string
	Shortcuts []HelpShortcut
}

// HelpShortcut represents a single shortcut
type HelpShortcut struct {
	Key         string
	Description string
}

// NewHelpOverlayComponent creates a help overlay
func NewHelpOverlayComponent() *HelpOverlayComponent {
	c := &HelpOverlayComponent{
		BaseComponent: NewBaseComponent(),
		viewport:      viewport.New(80, 24),
		active:        false,
		currentView:   "logs",
		overlayStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("235")). // Very Dark Gray
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")). // Bright Blue
			Padding(1, 2),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")). // Bright Blue
			Underline(true),
		sectionStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")), // Bright Cyan
		keyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Bright Green
			Bold(true).
			Width(15),
		descStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White
		borderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}

	c.initHelpContent()
	c.updateViewport()

	return c
}

// initHelpContent sets up default help sections
func (c *HelpOverlayComponent) initHelpContent() {
	c.helpSections = []HelpSection{
		{
			Title: "Navigation",
			Shortcuts: []HelpShortcut{
				{Key: "1-9", Description: "Switch between views"},
				{Key: "Tab", Description: "Next view"},
				{Key: "Shift+Tab", Description: "Previous view"},
				{Key: "?", Description: "Show this help"},
				{Key: "Esc", Description: "Close overlays/Cancel"},
			},
		},
		{
			Title: "Global Commands",
			Shortcuts: []HelpShortcut{
				{Key: "Ctrl+P", Description: "Open command palette"},
				{Key: "/", Description: "Search/Filter"},
				{Key: "q", Description: "Quit application"},
				{Key: "Ctrl+C", Description: "Force quit"},
			},
		},
		{
			Title: "Logs View",
			Shortcuts: []HelpShortcut{
				{Key: "↑/↓", Description: "Scroll logs"},
				{Key: "Home/End", Description: "Jump to top/bottom"},
				{Key: "c", Description: "Clear all logs"},
				{Key: "/", Description: "Filter logs"},
			},
		},
		{
			Title: "Processes View",
			Shortcuts: []HelpShortcut{
				{Key: "↑/↓", Description: "Select process"},
				{Key: "Enter", Description: "View process details"},
				{Key: "r", Description: "Restart selected process"},
				{Key: "R", Description: "Restart all processes"},
				{Key: "s", Description: "Stop selected process"},
				{Key: "S", Description: "Stop all processes"},
			},
		},
		{
			Title: "Errors View",
			Shortcuts: []HelpShortcut{
				{Key: "j/k", Description: "Navigate errors (vim-style)"},
				{Key: "↑/↓", Description: "Navigate errors"},
				{Key: "Enter", Description: "View error details"},
				{Key: "c", Description: "Clear all errors"},
			},
		},
		{
			Title: "AI Coders View",
			Shortcuts: []HelpShortcut{
				{Key: "j/k", Description: "Navigate sessions"},
				{Key: "Enter", Description: "Attach to session"},
				{Key: "a", Description: "Create new session"},
				{Key: "d", Description: "Detach from session"},
				{Key: "q", Description: "Stop selected session"},
			},
		},
		{
			Title: "Settings View",
			Shortcuts: []HelpShortcut{
				{Key: "↑/↓", Description: "Scroll settings"},
				{Key: "e", Description: "Edit setting (future)"},
			},
		},
		{
			Title: "Command Palette",
			Shortcuts: []HelpShortcut{
				{Key: "Ctrl+P", Description: "Open palette"},
				{Key: "Type", Description: "Search commands"},
				{Key: "↑/↓", Description: "Navigate results"},
				{Key: "Enter", Description: "Execute command"},
				{Key: "Esc", Description: "Close palette"},
			},
		},
	}
}

// Init initializes the component
func (c *HelpOverlayComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *HelpOverlayComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		if !c.active {
			return c, nil
		}

		switch msg.String() {
		case "esc", "q", "?":
			c.Close()
			return c, nil

		default:
			// Forward to viewport for scrolling
			c.viewport, cmd = c.viewport.Update(msg)
			return c, cmd
		}
	}

	return c, nil
}

// View renders the help overlay
func (c *HelpOverlayComponent) View() string {
	if !c.IsVisible() || !c.active {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *HelpOverlayComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)

	// Make overlay take most of screen
	overlayWidth := width - 10
	overlayHeight := height - 4

	if overlayWidth < 60 {
		overlayWidth = 60
	}
	if overlayHeight < 20 {
		overlayHeight = 20
	}

	c.viewport.Width = overlayWidth
	c.viewport.Height = overlayHeight

	c.overlayStyle = c.overlayStyle.Width(overlayWidth).Height(overlayHeight)
	c.updateViewport()
}

// updateViewport updates the viewport content
func (c *HelpOverlayComponent) updateViewport() {
	var lines []string

	// Title
	title := c.titleStyle.Render("Brummer - Keyboard Shortcuts")
	lines = append(lines, title)
	lines = append(lines, "")

	// Render each section
	for _, section := range c.helpSections {
		// Section title
		sectionTitle := c.sectionStyle.Render(section.Title)
		lines = append(lines, sectionTitle)
		lines = append(lines, "")

		// Shortcuts in section
		for _, shortcut := range section.Shortcuts {
			key := c.keyStyle.Render("  " + shortcut.Key)
			desc := c.descStyle.Render(shortcut.Description)
			line := key + "  " + desc
			lines = append(lines, line)
		}

		lines = append(lines, "")
	}

	// Footer
	lines = append(lines, c.borderStyle.Render(strings.Repeat("─", c.viewport.Width)))
	lines = append(lines, "")
	footer := c.borderStyle.Render("Press ? or Esc to close help")
	lines = append(lines, footer)

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}

// Open shows the help overlay
func (c *HelpOverlayComponent) Open() {
	c.active = true
	c.viewport.GotoTop()
	c.updateViewport()
}

// Close hides the help overlay
func (c *HelpOverlayComponent) Close() {
	c.active = false
}

// IsActive returns whether the overlay is currently active
func (c *HelpOverlayComponent) IsActive() bool {
	return c.active
}

// SetCurrentView updates context for view-specific help
func (c *HelpOverlayComponent) SetCurrentView(viewName string) {
	c.currentView = viewName
	c.updateViewport()
}

// AddSection adds a custom help section
func (c *HelpOverlayComponent) AddSection(section HelpSection) {
	c.helpSections = append(c.helpSections, section)
	c.updateViewport()
}

// AddShortcut adds a shortcut to an existing section
func (c *HelpOverlayComponent) AddShortcut(sectionTitle string, shortcut HelpShortcut) {
	for i := range c.helpSections {
		if c.helpSections[i].Title == sectionTitle {
			c.helpSections[i].Shortcuts = append(c.helpSections[i].Shortcuts, shortcut)
			c.updateViewport()
			return
		}
	}

	// Section doesn't exist, create it
	c.AddSection(HelpSection{
		Title:     sectionTitle,
		Shortcuts: []HelpShortcut{shortcut},
	})
}

// GetShortcutSummary returns a brief summary for footer
func (c *HelpOverlayComponent) GetShortcutSummary() string {
	return "? Help • q Quit • 1-8 Views • Ctrl+P Commands"
}
