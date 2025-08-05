package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
)

// URLsViewController manages the URLs view state and rendering
type URLsViewController struct {
	urlsViewport   viewport.Model
	ShowingMCPHelp bool // Exported for access from Model

	// Dependencies injected from parent Model
	logStore     *logs.Store
	mcpServer    MCPServerInterface
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewURLsViewController creates a new URLs view controller
func NewURLsViewController(logStore *logs.Store, mcpServer MCPServerInterface) *URLsViewController {
	return &URLsViewController{
		urlsViewport: viewport.New(0, 0),
		logStore:     logStore,
		mcpServer:    mcpServer,
	}
}

// UpdateSize updates the viewport dimensions
func (v *URLsViewController) UpdateSize(width, height, headerHeight, footerHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight
	v.urlsViewport.Width = width
	v.urlsViewport.Height = height - headerHeight - footerHeight
}

// ToggleMCPHelp toggles the MCP help display
func (v *URLsViewController) ToggleMCPHelp() {
	v.ShowingMCPHelp = !v.ShowingMCPHelp
}

// Render renders the URLs view
func (v *URLsViewController) Render() string {
	urls := v.logStore.GetURLs()

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
func (v *URLsViewController) renderSimple(urls []logs.URLEntry) string {
	var content strings.Builder

	// Header with count
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	content.WriteString(headerStyle.Render(fmt.Sprintf("🔗 Detected URLs (%d)", len(urls))) + "\n\n")

	if len(urls) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
		content.WriteString(emptyStyle.Render("No URLs detected yet. Start servers with /run <script>. Use /proxy or /toggle-proxy for URL management."))
	} else {
		content.WriteString(v.renderURLsList(urls))
	}

	v.urlsViewport.SetContent(content.String())
	return v.urlsViewport.View()
}

// renderURLsList renders a list of URLs with styling
func (v *URLsViewController) renderURLsList(urls []logs.URLEntry) string {
	var content strings.Builder

	// Group URLs by process
	urlsByProcess := make(map[string][]logs.URLEntry)
	for _, url := range urls {
		urlsByProcess[url.ProcessName] = append(urlsByProcess[url.ProcessName], url)
	}

	processStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Underline(true)
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
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

		// Deduplicate URLs for display
		seen := make(map[string]logs.URLEntry)
		for _, urlEntry := range processURLs {
			// Use the URL as key to deduplicate, keeping the first occurrence
			if _, exists := seen[urlEntry.URL]; !exists {
				seen[urlEntry.URL] = urlEntry
			}
		}

		// Display unique URLs
		for _, urlEntry := range seen {
			// Create clickable URL display
			clickable := fmt.Sprintf("   %s", urlStyle.Render(urlEntry.URL))
			content.WriteString(clickable)

			// Add context if available
			if urlEntry.Context != "" {
				content.WriteString(fmt.Sprintf(" %s", contextStyle.Render(fmt.Sprintf("(%s)", urlEntry.Context))))
			}

			// Add first seen time
			content.WriteString(fmt.Sprintf(" %s\n", timeStyle.Render(urlEntry.Timestamp.Format("15:04:05"))))
		}
	}

	return content.String()
}

// renderMCPConnectionBox renders the MCP connection information box
func (v *URLsViewController) renderMCPConnectionBox(mcpURLs []logs.URLEntry) string {
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

	if v.ShowingMCPHelp {
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
