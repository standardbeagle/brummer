package components

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// URLsViewComponent adapts the URLs view to the component system
type URLsViewComponent struct {
	BaseComponent

	// Component state
	urls []URLInfo
	list list.Model

	// Display options
	groupByProcess bool
	showTimestamp  bool

	// Styles
	titleStyle   lipgloss.Style
	urlStyle     lipgloss.Style
	processStyle lipgloss.Style
	helpStyle    lipgloss.Style
	activeStyle  lipgloss.Style
}

// NewURLsViewComponent creates a new URLs view component
func NewURLsViewComponent(processMgr interface{}) *URLsViewComponent {
	u := &URLsViewComponent{
		urls:           []URLInfo{},
		groupByProcess: false,
		showTimestamp:  true,
	}

	// Initialize styles
	u.initStyles()

	// Initialize list
	u.initializeList()

	// Set visible by default
	u.visible = true

	return u
}

// initStyles initializes the component styles
func (u *URLsViewComponent) initStyles() {
	u.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	u.urlStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Underline(true)

	u.processStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205"))

	u.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	u.activeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)
}

// initializeList sets up the bubbles list component
func (u *URLsViewComponent) initializeList() {
	items := u.urlsToListItems()

	u.list = list.New(items, urlItemDelegate{}, 80, 20)
	u.list.Title = "URLs"
	u.list.SetShowStatusBar(true)
	u.list.SetFilteringEnabled(true)
	u.list.SetShowHelp(true)

	// Customize key bindings
	u.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	u.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}

// Update handles messages
func (u *URLsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle component-specific messages
	switch msg := msg.(type) {
	case URLDetectedMsg:
		// Add or update URL
		found := false
		for i, url := range u.urls {
			if url.URL == msg.URL {
				// Update existing URL
				u.urls[i].LastSeen = msg.Timestamp
				u.urls[i].Count++
				if msg.ProcessID != "" {
					u.urls[i].ProcessID = msg.ProcessID
				}
				// Update proxy info if provided
				if msg.HasProxy {
					u.urls[i].HasProxy = true
					u.urls[i].ProxyPort = msg.ProxyPort
					u.urls[i].ProxyURL = msg.ProxyURL
					u.urls[i].ProxyLabel = msg.ProxyLabel
				}
				found = true
				break
			}
		}

		if !found {
			// Add new URL with proxy information if available
			u.urls = append(u.urls, URLInfo{
				URL:        msg.URL,
				ProcessID:  msg.ProcessID,
				Type:       msg.Type,
				FirstSeen:  msg.Timestamp,
				LastSeen:   msg.Timestamp,
				Count:      1,
				Active:     true,
				HasProxy:   msg.HasProxy,
				ProxyPort:  msg.ProxyPort,
				ProxyURL:   msg.ProxyURL,
				ProxyLabel: msg.ProxyLabel,
			})
		}

		u.updateList()

		// Send notification for new URL
		if !found {
			return u, SendSystemNotification(NotificationInfo,
				"URL Detected", msg.URL)
		}
		return u, nil

	case URLsUpdateMsg:
		// Update entire URL list
		u.urls = msg.URLs
		u.updateList()
		return u, nil

	case URLOpenMsg:
		// Open URL
		if msg.External {
			err := openURL(msg.URL)
			if err != nil {
				return u, SendSystemNotification(NotificationError,
					"Failed to open URL", err.Error())
			}
			return u, SendSystemNotification(NotificationSuccess,
				"URL opened", msg.URL)
		}
		// Internal opening would switch to web view
		cmds = append(cmds, SendViewChange("web", msg.URL))
		return u, tea.Batch(cmds...)

	case URLCopyMsg:
		// Copy URL to clipboard
		// This would need clipboard integration
		return u, SendSystemNotification(NotificationInfo,
			"URL copied", msg.URL)

	case tea.WindowSizeMsg:
		u.width = msg.Width
		u.height = msg.Height
		u.list.SetSize(msg.Width, msg.Height-4)

	case tea.KeyMsg:
		// Handle key presses
		switch msg.String() {
		case "enter", " ":
			// Open URL in browser
			if item := u.list.SelectedItem(); item != nil {
				if ui, ok := item.(urlItem); ok {
					return u, SendURLOpen(ui.url.URL, true)
				}
			}

		case "o":
			// Open URL externally
			if item := u.list.SelectedItem(); item != nil {
				if ui, ok := item.(urlItem); ok {
					return u, SendURLOpen(ui.url.URL, true)
				}
			}

		case "c":
			// Copy URL
			if item := u.list.SelectedItem(); item != nil {
				if ui, ok := item.(urlItem); ok {
					return u, SendURLCopy(ui.url.URL)
				}
			}

		case "w":
			// Open in web view
			if item := u.list.SelectedItem(); item != nil {
				if ui, ok := item.(urlItem); ok {
					return u, SendURLOpen(ui.url.URL, false)
				}
			}

		case "g":
			// Toggle grouping by process
			u.groupByProcess = !u.groupByProcess
			u.updateList()
			return u, nil

		case "t":
			// Toggle timestamp display
			u.showTimestamp = !u.showTimestamp
			u.updateList()
			return u, nil

		case "r":
			// Refresh/recheck URLs
			// This could ping URLs to check if they're still active
			return u, SendSystemNotification(NotificationInfo,
				"Refreshing URLs", "Checking URL availability...")
		}
	}

	// Update the list component
	if u.visible {
		newList, cmd := u.list.Update(msg)
		u.list = newList
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return u, tea.Batch(cmds...)
}

// View renders the component
func (u *URLsViewComponent) View() string {
	if !u.visible {
		return ""
	}

	// Use the list component for rendering
	return u.list.View()
}

// updateList updates the list items
func (u *URLsViewComponent) updateList() {
	items := u.urlsToListItems()
	u.list.SetItems(items)

	// Update title with count
	if len(u.urls) > 0 {
		u.list.Title = fmt.Sprintf("URLs (%d)", len(u.urls))
	} else {
		u.list.Title = "URLs"
	}
}

// urlsToListItems converts URLs to list items
func (u *URLsViewComponent) urlsToListItems() []list.Item {
	// Group by process if enabled
	if u.groupByProcess {
		// Create a map of process -> URLs
		grouped := make(map[string][]URLInfo)
		for _, url := range u.urls {
			process := url.ProcessID
			if process == "" {
				process = "unknown"
			}
			grouped[process] = append(grouped[process], url)
		}

		// Create list items grouped
		var items []list.Item
		for process, urls := range grouped {
			// Add process header
			items = append(items, urlGroupItem{
				process: process,
				count:   len(urls),
			})
			// Add URLs for this process
			for _, url := range urls {
				items = append(items, urlItem{
					url:           url,
					showTimestamp: u.showTimestamp,
					indented:      true,
				})
			}
		}
		return items
	}

	// Ungrouped list
	items := make([]list.Item, len(u.urls))
	for i, url := range u.urls {
		items[i] = urlItem{
			url:           url,
			showTimestamp: u.showTimestamp,
			indented:      false,
		}
	}
	return items
}

// Focus sets focus to this component
func (u *URLsViewComponent) Focus() tea.Cmd {
	u.focused = true
	return nil
}

// Blur removes focus from this component
func (u *URLsViewComponent) Blur() tea.Cmd {
	u.focused = false
	return nil
}

// urlItem implements list.Item for URLs
type urlItem struct {
	url           URLInfo
	showTimestamp bool
	indented      bool
}

func (i urlItem) Title() string {
	prefix := ""
	if i.indented {
		prefix = "  "
	}

	icon := "🔗"
	switch i.url.Type {
	case URLTypeHTTPS:
		icon = "🔒"
	case URLTypeWebSocket:
		icon = "🔌"
	case URLTypeFile:
		icon = "📁"
	}

	title := fmt.Sprintf("%s%s %s", prefix, icon, i.url.URL)

	// Add proxy badge if available
	if i.url.HasProxy {
		title += fmt.Sprintf(" 🚀 [Proxy: %s]", i.url.ProxyURL)
	}

	if i.url.Active {
		title += " ✓"
	}
	return title
}

func (i urlItem) Description() string {
	parts := []string{}

	// Add process if available
	if i.url.ProcessID != "" {
		parts = append(parts, fmt.Sprintf("Process: %s", i.url.ProcessID))
	}

	// Add proxy label if available
	if i.url.HasProxy && i.url.ProxyLabel != "" {
		parts = append(parts, fmt.Sprintf("Label: %s", i.url.ProxyLabel))
	}

	// Add proxy port if available
	if i.url.HasProxy && i.url.ProxyPort > 0 {
		parts = append(parts, fmt.Sprintf("Port: %d", i.url.ProxyPort))
	}

	// Add count if more than 1
	if i.url.Count > 1 {
		parts = append(parts, fmt.Sprintf("Seen: %dx", i.url.Count))
	}

	// Add timestamp if enabled
	if i.showTimestamp {
		timeAgo := time.Since(i.url.LastSeen)
		var timeStr string
		if timeAgo < time.Minute {
			timeStr = fmt.Sprintf("%ds ago", int(timeAgo.Seconds()))
		} else if timeAgo < time.Hour {
			timeStr = fmt.Sprintf("%dm ago", int(timeAgo.Minutes()))
		} else {
			timeStr = fmt.Sprintf("%dh ago", int(timeAgo.Hours()))
		}
		parts = append(parts, timeStr)
	}

	if i.indented {
		return "  " + strings.Join(parts, " | ")
	}
	return strings.Join(parts, " | ")
}

func (i urlItem) FilterValue() string {
	return i.url.URL
}

// urlGroupItem implements list.Item for process groups
type urlGroupItem struct {
	process string
	count   int
}

func (i urlGroupItem) Title() string {
	return fmt.Sprintf("📦 %s (%d URLs)", i.process, i.count)
}

func (i urlGroupItem) Description() string {
	return ""
}

func (i urlGroupItem) FilterValue() string {
	return i.process
}

// urlItemDelegate implements list.ItemDelegate
type urlItemDelegate struct{}

func (d urlItemDelegate) Height() int { return 2 }

func (d urlItemDelegate) Spacing() int { return 1 }

func (d urlItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d urlItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Create styles inline
	urlStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("170")).
		Bold(true)
	groupStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	var str string

	switch item := listItem.(type) {
	case urlItem:
		str = fmt.Sprintf("%s\n  %s", item.Title(), item.Description())
	case urlGroupItem:
		str = groupStyle.Render(item.Title())
	default:
		return
	}

	var fn func(...string) string
	if index == m.Index() {
		fn = func(strs ...string) string {
			s := strings.Join(strs, "")
			return selectedStyle.Render("> " + s)
		}
	} else {
		fn = urlStyle.Render
	}

	fmt.Fprintf(w, "%s", fn(str))
}

// Helper for copying URL
func SendURLCopy(url string) tea.Cmd {
	return func() tea.Msg {
		return URLCopyMsg{
			URL: url,
		}
	}
}

// openURL opens a URL in the default browser
func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}
