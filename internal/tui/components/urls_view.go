package components

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// URLsViewComponentNew is a stateless BubbleTea component for displaying discovered URLs
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type URLsViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	width  int
	height int

	// List model for display (minimal, stateless)
	list   list.Model

	// Styles
	headerStyle     lipgloss.Style
	sectionStyle    lipgloss.Style
	urlStyle         lipgloss.Style
	processStyle    lipgloss.Style
	timestampStyle  lipgloss.Style
	domainStyle     lipgloss.Style
	statsStyle      lipgloss.Style
	loadingStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	emptyStyle      lipgloss.Style
}

// NewURLsView creates a new stateless URLs view component
func NewURLsView() *URLsViewComponentNew {
	urlsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	urlsList.Title = "Discovered URLs"
	urlsList.SetShowStatusBar(false)

	return &URLsViewComponentNew{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,
		list:          urlsList,

		// Initialize styles
		headerStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // Bright Blue
		sectionStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13")), // Bright Magenta
		urlStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("14")),            // Bright Cyan
		processStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),            // Bright Yellow
		timestampStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),           // Gray
		domainStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")),             // Dark Gray
		statsStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")),            // Bright Green
		loadingStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),            // Bright Yellow
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")),           // Bright Red
		emptyStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),           // Gray
	}
}

// ViewWithState renders the URLs view using external state (Redux/Elm compliant)
//
// ⚠️  CRITICAL: This method MUST be pure and non-blocking!
// ❌ WRONG:  urls := urlService.DiscoverURLs()  // DON'T call services that might block!
// WHY: View() must return immediately. Blocking freezes the BubbleTea event loop.
// ✅ CORRECT: Only read from masterData (cached state).
//
// See: /docs/architecture/tui-deadlock-prevention.md for mandatory patterns
func (c *URLsViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.URLsState == nil {
		return c.renderErrorState("URLs state not available")
	}

	urlsState := masterData.URLsState

	// Update list with current state
	c.updateListFromState(urlsState)

	// Render the list
	return c.list.View()
}

// updateListFromState updates the URLs list using current state
func (c *URLsViewComponentNew) updateListFromState(urlsState *state.URLsState) {
	if urlsState.GetLoading() {
		// Show loading state
		c.list.SetItems([]list.Item{
			URLSectionItemWrapper{TitleText: "⏳ Loading URLs..."},
		})
		return
	}

	if error := urlsState.GetError(); error != "" {
		// Show error state
		c.list.SetItems([]list.Item{
			URLSectionItemWrapper{TitleText: "❌ Error Loading URLs"},
			URLInfoItemWrapper{
				URL:         "Error",
				Label:       error,
				ProcessName: "",
				Timestamp:   time.Now(),
			},
		})
		return
	}

	// Get filtered URLs
	filteredURLs := urlsState.GetFilteredURLs()

	if len(filteredURLs) == 0 {
		// Show empty state
		items := []list.Item{
			URLSectionItemWrapper{TitleText: "🔗 No URLs Discovered"},
		}

		if urlsState.GetFilterText() != "" {
			items = append(items, URLInfoItemWrapper{
				URL:         "No Results",
				Label:       fmt.Sprintf("No URLs match filter: %s", urlsState.GetFilterText()),
				ProcessName: "",
				Timestamp:   time.Now(),
			})
		} else {
			items = append(items, URLInfoItemWrapper{
				URL:         "Empty State",
				Label:       "Start processes to discover URLs from their output logs",
				ProcessName: "",
				Timestamp:   time.Now(),
			})
		}

		c.list.SetItems(items)
		return
	}

	// Show statistics header
	items := []list.Item{
		URLStatsItemWrapper{
			TotalURLs:     urlsState.GetTotalURLs(),
			UniqueDomains: urlsState.GetUniqueDomainCount(),
		},
	}

	// Group by process if enabled
	if urlsState.GetGroupByProcess() {
		groupedURLs := urlsState.GetURLsByProcess()
		sortedProcesses := make([]string, 0, len(groupedURLs))
		for process := range groupedURLs {
			sortedProcesses = append(sortedProcesses, process)
		}
		sort.Strings(sortedProcesses)

		for _, processName := range sortedProcesses {
			processURLs := groupedURLs[processName]
			if len(processURLs) > 0 {
				// Add process section header
				items = append(items, URLSectionItemWrapper{
					TitleText: fmt.Sprintf("🔗 %s (%d URLs)", processName, len(processURLs)),
				})

				// Add URLs for this process
				for _, url := range processURLs {
					items = append(items, URLInfoItemWrapper{
						URL:         url.URL,
						Label:       url.Label,
						ProcessName: url.ProcessName,
						Timestamp:   url.Timestamp,
					})
				}
			}
		}
	} else {
		// Show all URLs in one list
		items = append(items, URLSectionItemWrapper{TitleText: "🔗 All URLs"})
		for _, url := range filteredURLs {
			items = append(items, URLInfoItemWrapper{
				URL:         url.URL,
				Label:       url.Label,
				ProcessName: url.ProcessName,
				Timestamp:   url.Timestamp,
			})
		}
	}

	c.list.SetItems(items)
}

// renderErrorState renders an error state when URLs data is not available
func (c *URLsViewComponentNew) renderErrorState(message string) string {
	var content strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1).
		MarginBottom(1)

	content.WriteString(headerStyle.Render("🔗  Discovered URLs"))
	content.WriteString("\n")

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Margin(1)

	content.WriteString(errorStyle.Render("⚠️  "+message))
	content.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Margin(1)

	content.WriteString(helpStyle.Render("URL information will be available when processes are running and generating web requests"))

	return content.String()
}

// SetSize updates component dimensions
func (c *URLsViewComponentNew) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, height-6)
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *URLsViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, contentHeight)
}

// View renders the component (legacy method for compatibility)
func (c *URLsViewComponentNew) View() string {
	if !c.IsVisible() {
		return ""
	}
	return c.list.View()
}

// Update handles messages (minimal - stateless component)
func (c *URLsViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to list for selection
		c.list, cmd = c.list.Update(msg)
		return c, cmd
	}

	return c, nil
}

// Init initializes the component
func (c *URLsViewComponentNew) Init() tea.Cmd {
	return nil
}

// Wrapper types for list.Item interface compatibility

// URLSectionItemWrapper represents a section header in the URLs list
type URLSectionItemWrapper struct {
	TitleText string
}

func (i URLSectionItemWrapper) FilterValue() string { return "" }
func (i URLSectionItemWrapper) Title() string       { return i.TitleText }
func (i URLSectionItemWrapper) Description() string { return "" }

// URLStatsItemWrapper represents statistics in the URLs list
type URLStatsItemWrapper struct {
	TotalURLs     int
	UniqueDomains int
}

func (i URLStatsItemWrapper) FilterValue() string { return "statistics" }
func (i URLStatsItemWrapper) Title() string {
	return fmt.Sprintf("📊 Statistics: %d URLs from %d domains", i.TotalURLs, i.UniqueDomains)
}
func (i URLStatsItemWrapper) Description() string {
	return fmt.Sprintf("Total URLs discovered: %d | Unique domains: %d", i.TotalURLs, i.UniqueDomains)
}

// URLInfoItemWrapper represents a URL in the list
type URLInfoItemWrapper struct {
	URL         string
	Label       string
	ProcessName string
	Timestamp   time.Time
}

func (i URLInfoItemWrapper) FilterValue() string {
	return i.URL + " " + i.Label + " " + i.ProcessName
}

func (i URLInfoItemWrapper) Title() string {
	// Extract domain for display
	domain := "Unknown Domain"
	if parsed, err := url.Parse(i.URL); err == nil {
		domain = parsed.Host
		if domain == "" {
			domain = "Local"
		}
	}

	return fmt.Sprintf("🌐 %s", domain)
}

func (i URLInfoItemWrapper) Description() string {
	var parts []string

	if i.Label != "" && i.Label != i.URL {
		parts = append(parts, i.Label)
	}

	parts = append(parts, fmt.Sprintf("Process: %s", i.ProcessName))
	parts = append(parts, fmt.Sprintf("Time: %s", i.Timestamp.Format("15:04:05")))

	return strings.Join(parts, " | ")
}