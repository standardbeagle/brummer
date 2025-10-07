package components

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
)

// URLsView is a BubbleTea component for displaying discovered URLs
// Migrated from urls_view_controller.go (T023-T025)
// Architecture: Component → Service (direct logStore access)
type URLsView struct {
	viewport       viewport.Model
	showingMCPHelp bool

	// Direct service access (no controller indirection)
	logStore *logs.Store

	// Layout dimensions
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewURLsView creates a new URLs view component
func NewURLsView(logStore *logs.Store) *URLsView {
	return &URLsView{
		viewport:     viewport.New(0, 0),
		logStore:     logStore,
		headerHeight: 3,
		footerHeight: 1,
	}
}

// Init initializes the component (BubbleTea interface)
func (v *URLsView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *URLsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		contentHeight := v.height - v.headerHeight - v.footerHeight
		v.viewport.Width = msg.Width
		v.viewport.Height = contentHeight
	}

	return v, nil
}

// View renders the component (BubbleTea interface)
func (v *URLsView) View() string {
	return v.Render()
}

// SetSize updates the component's dimensions (Component interface)
func (v *URLsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	contentHeight := height - v.headerHeight - v.footerHeight
	v.viewport.Width = width
	v.viewport.Height = contentHeight
}

// UpdateSize updates the viewport dimensions with pre-calculated content height
func (v *URLsView) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight
	v.viewport.Width = width
	v.viewport.Height = contentHeight
}

// Focus gives keyboard focus to the component (Component interface)
func (v *URLsView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *URLsView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *URLsView) GetSize() (width, height int) {
	return v.width, v.height
}

// ToggleMCPHelp toggles the MCP help display
func (v *URLsView) ToggleMCPHelp() {
	v.showingMCPHelp = !v.showingMCPHelp
}

// IsShowingMCPHelp returns whether MCP help is displayed
func (v *URLsView) IsShowingMCPHelp() bool {
	return v.showingMCPHelp
}

// Render renders the URLs view with direct logStore access
func (v *URLsView) Render() string {
	// Direct service call - no controller indirection (T025)
	urls := v.logStore.GetURLs()

	// Apply stable sort to ensure consistent ordering in UI
	// Sort by: ProcessName first, then URL, then Timestamp
	sort.Slice(urls, func(i, j int) bool {
		if urls[i].ProcessName != urls[j].ProcessName {
			return urls[i].ProcessName < urls[j].ProcessName
		}
		if urls[i].URL != urls[j].URL {
			return urls[i].URL < urls[j].URL
		}
		return urls[i].Timestamp.Before(urls[j].Timestamp)
	})

	// Separate MCP URLs from regular URLs
	var mcpURLs []logs.URLEntry
	var regularURLs []logs.URLEntry

	for _, urlEntry := range urls {
		if urlEntry.ProcessName == "MCP" || urlEntry.ProcessName == "mcp-server" {
			mcpURLs = append(mcpURLs, urlEntry)
		} else {
			regularURLs = append(regularURLs, urlEntry)
		}
	}

	// Split layout: regular URLs on left, MCP connection box on right
	if v.width < 100 {
		// For narrow screens, use simple single column
		return v.renderSimple(urls)
	}

	leftWidth := v.width * 2 / 3
	rightWidth := v.width - leftWidth - 3
	contentHeight := v.height - v.headerHeight - v.footerHeight

	// Create left panel content (regular URLs)
	var leftContent strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	leftContent.WriteString(headerStyle.Render(fmt.Sprintf("🔗 Application URLs (%d)", len(regularURLs))) + "\n\n")

	if len(regularURLs) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
		leftContent.WriteString(emptyStyle.Render("No application URLs detected yet.\nStart servers with /run <script>."))
	} else {
		leftContent.WriteString(v.renderURLsList(regularURLs))
	}

	// Create right panel content (MCP connection box)
	rightContent := v.renderMCPConnectionBox(mcpURLs)

	// Create bordered panels
	leftPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(leftWidth - 2).
		Height(contentHeight - 2).
		Padding(1).
		Render(leftContent.String())

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("75")).
		Width(rightWidth - 2).
		Height(contentHeight - 2).
		Padding(1).
		Render(rightContent)

	// Combine panels side by side
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "   ", rightPanel)
}

// renderSimple renders a simple single-column view for narrow screens
func (v *URLsView) renderSimple(urls []logs.URLEntry) string {
	var content strings.Builder

	// URLs are already sorted by the Render method, no need to sort again

	// Header with count
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	content.WriteString(headerStyle.Render(fmt.Sprintf("🔗 Detected URLs (%d)", len(urls))) + "\n\n")

	if len(urls) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
		content.WriteString(emptyStyle.Render("No URLs detected yet. Start servers with /run <script>. Use /proxy or /toggle-proxy for URL management."))
	} else {
		content.WriteString(v.renderURLsList(urls))
	}

	v.viewport.SetContent(content.String())
	return v.viewport.View()
}

// renderURLsList renders a list of URLs with styling
func (v *URLsView) renderURLsList(urls []logs.URLEntry) string {
	var content strings.Builder

	// Group URLs by process
	urlsByProcess := make(map[string][]logs.URLEntry)
	for _, url := range urls {
		urlsByProcess[url.ProcessName] = append(urlsByProcess[url.ProcessName], url)
	}

	processStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Underline(true)
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("102"))

	// Instructions
	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	content.WriteString(instructionStyle.Render("Press Enter on URL to copy to clipboard") + "\n\n")

	isFirst := true
	for processName, processURLs := range urlsByProcess {
		if !isFirst {
			content.WriteString("\n")
		}
		isFirst = false

		content.WriteString(processStyle.Render(fmt.Sprintf("📦 %s", processName)) + "\n")

		// Deduplicate URLs while maintaining order from sorted input
		seen := make(map[string]bool)
		for _, urlEntry := range processURLs {
			// Skip if we've already seen this URL
			if seen[urlEntry.URL] {
				continue
			}
			seen[urlEntry.URL] = true

			// Create clickable URL display
			clickable := fmt.Sprintf("   %s", urlStyle.Render(urlEntry.URL))
			content.WriteString(clickable)

			// Add proxy URL if available (instead of context)
			if urlEntry.ProxyURL != "" {
				proxyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Italic(true)
				content.WriteString(fmt.Sprintf(" → %s", proxyStyle.Render(urlEntry.ProxyURL)))
			}

			// Add first seen time
			content.WriteString(fmt.Sprintf(" %s\n", timeStyle.Render(urlEntry.Timestamp.Format("15:04:05"))))
		}
	}

	return content.String()
}

// renderMCPConnectionBox renders the MCP connection information box
func (v *URLsView) renderMCPConnectionBox(mcpURLs []logs.URLEntry) string {
	var content strings.Builder

	// MCP Connection box styling
	boxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("75")).
		Align(lipgloss.Center)

	content.WriteString(titleStyle.Render("🔌 MCP Connection") + "\n\n")

	if len(mcpURLs) > 0 {
		// Show MCP is running
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true).
			Align(lipgloss.Center)

		content.WriteString(statusStyle.Render("✓ MCP Server Active") + "\n\n")

		// Show the MCP URL
		urlStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Underline(true)

		for _, url := range mcpURLs {
			content.WriteString(boxStyle.Render("Connect at:") + "\n")
			content.WriteString(urlStyle.Render(url.URL) + "\n\n")
			break // Only show the first one
		}
	} else {
		// MCP not running
		content.WriteString(boxStyle.Render("MCP Server Not Active") + "\n\n")
	}

	// Add connection instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Width(30).
		Align(lipgloss.Left)

	if v.showingMCPHelp {
		// Show detailed help
		content.WriteString(instructionStyle.Render("Claude.ai Configuration:") + "\n\n")

		codeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

		content.WriteString(codeStyle.Render(`{
  "mcpServers": {
    "brummer": {
      "command": "brum",
      "args": ["--mcp"],
      "env": {}
    }
  }
}`) + "\n\n")

		content.WriteString(instructionStyle.Render("Save this to:") + "\n")
		content.WriteString(codeStyle.Render("~/Library/Application Support/Claude/claude_desktop_config.json") + "\n\n")

		linkStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
		content.WriteString(linkStyle.Render("Press ? to hide help"))
	} else {
		content.WriteString(instructionStyle.Render("Configure Claude.ai to connect\nto this MCP server.") + "\n\n")

		linkStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)
		content.WriteString(linkStyle.Render("Press ? for setup help"))
	}

	return content.String()
}
