package components

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	ui_services "github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
	"golang.org/x/term"
)

// getTerminalSize attempts to get the actual terminal size
func getTerminalSize() (int, int, error) {
	// Try to get terminal size from stdout
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return width, height, nil
	}

	// Try stderr if stdout fails
	if width, height, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		return width, height, nil
	}

	// Try stdin as last resort
	if width, height, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		return width, height, nil
	}

	// Return error if all attempts fail
	return 0, 0, fmt.Errorf("unable to get terminal size")
}

// IntegratedShell combines all components into a working shell
//
// Performance optimizations:
// - Process name cache using sync.Map for O(1) non-blocking lookups
// - Event-driven log updates with debouncing instead of polling
// - Automatic terminal size detection on startup
type IntegratedShell struct {
	width  int
	height int
	quit   bool

	// View management
	currentView ViewType
	views       map[ViewType]*ViewInfo
	viewOrder   []ViewType

	// Components
	processView    *ProcessViewNew
	logsView       *LogsViewComponent
	errorsView     *ErrorsViewComponent
	urlsView       *URLsViewComponent
	webView        *URLsViewComponent // Reuse URLs component for now
	aiView         *AIViewComponent
	settingsView   *SettingsViewComponent // Using existing adapter
	commandPalette *CommandPalette

	// Command input state
	showCommandPalette bool

	// Pending errors to be sent to errors view
	pendingErrors []ErrorOccurredMsg

	// Pending URLs to be sent to URLs view
	pendingURLs []URLDetectedMsg

	// Data stores
	logStore         *logs.Store
	logSub           *DebouncedLogSubscription // Event-driven log updates with debouncing
	processNameCache *sync.Map                 // Cache process ID -> name for fast lookups

	// Process manager
	processMgr *process.Manager

	// Proxy server for URL mappings
	proxyServer ProxyServerInterface

	// Event system
	eventBus *events.EventBus

	// Width calculator
	widthCalc *TabWidthCalculator

	// Styles
	titleStyle       lipgloss.Style
	activeTabStyle   lipgloss.Style
	inactiveTabStyle lipgloss.Style
	errorBadgeStyle  lipgloss.Style
	warnBadgeStyle   lipgloss.Style
	infoBadgeStyle   lipgloss.Style
	separatorStyle   lipgloss.Style
}

// ProxyServerInterface defines the interface for proxy server operations
type ProxyServerInterface interface {
	GetURLMappings() []ProxyURLMapping
}

// ProxyURLMapping represents a URL mapping from the proxy server
type ProxyURLMapping struct {
	TargetURL   string
	ProxyPort   int
	ProxyURL    string
	ProcessName string
	Label       string
}

// NewIntegratedShell creates a shell with real components
func NewIntegratedShell(processMgr *process.Manager, eventBus *events.EventBus, proxyServer ProxyServerInterface) *IntegratedShell {
	// ALWAYS initialize colors first
	MustInitColors()

	// Create log store with reasonable defaults
	logStore := logs.NewStore(10000, nil)

	// Use terminal size if available, otherwise use reasonable defaults
	// Most modern terminals are at least 120x30
	initialWidth := 120
	initialHeight := 30

	// Try to get actual terminal size if possible
	// Note: This is just for a better initial size, the real size comes from WindowSizeMsg
	if width, height, err := getTerminalSize(); err == nil && width > 0 && height > 0 {
		initialWidth = width
		initialHeight = height
	}

	shell := &IntegratedShell{
		width:            initialWidth,
		height:           initialHeight,
		currentView:      ViewProcesses,
		views:            make(map[ViewType]*ViewInfo),
		processMgr:       processMgr,
		proxyServer:      proxyServer,
		logStore:         logStore,
		logSub:           NewDebouncedLogSubscription(logStore), // Event-driven subscription
		processNameCache: &sync.Map{},
		eventBus:         eventBus,
	}

	// Initialize components
	shell.processView = NewProcessViewNew(processMgr)
	shell.logsView = NewLogsViewComponent(logStore)
	shell.errorsView = NewErrorsViewComponent()
	shell.urlsView = NewURLsViewComponent(processMgr)
	shell.webView = NewURLsViewComponent(processMgr) // Reuse URLs component for web view
	shell.aiView = NewAIViewComponent()
	shell.settingsView = NewSettingsViewComponent()

	// Initialize services
	aiService := ui_services.NewAICoderService(nil, logStore)

	// Initialize command palette with handler
	commandHandler := NewIntegratedShellCommandHandler(shell, processMgr, logStore, aiService)
	shell.commandPalette = NewCommandPalette(commandHandler)

	// Populate initial process name cache
	if processMgr != nil {
		for _, proc := range processMgr.GetAllProcesses() {
			shell.processNameCache.Store(proc.ID, proc.Name)
		}
	}

	// Register log callback to capture process output
	if processMgr != nil {
		processMgr.AddLogCallback(func(processID, line string, isError bool) {
			// Get process name from cache (non-blocking)
			processName := "Unknown"
			if cached, ok := shell.processNameCache.Load(processID); ok {
				processName = cached.(string)
			}
			logStore.Add(processID, processName, line, isError)

			// If this is an error, send ErrorOccurredMsg to the errors view
			if isError && shell.errorsView != nil {
				// Send error message through Bubble Tea's message system
				// Note: We can't send messages directly from callback, need to use program.Send
				// For now, store errors in the errors view component
				shell.pendingErrors = append(shell.pendingErrors, ErrorOccurredMsg{
					ProcessID:   processID,
					ProcessName: processName,
					Error:       line,
					Timestamp:   time.Now(),
				})
			}

			// Detect URLs in the line
			urls := extractURLs(line)
			for _, url := range urls {
				// Store pending URL detection for sending to URLs view
				urlType := URLTypeHTTP // Default
				switch getURLType(url) {
				case "https":
					urlType = URLTypeHTTPS
				case "ws", "wss":
					urlType = URLTypeWebSocket
				}
				urlMsg := URLDetectedMsg{
					URL:       url,
					ProcessID: processID,
					Type:      urlType,
					Timestamp: time.Now(),
				}

				// Check if proxy server has a mapping for this URL
				if shell.proxyServer != nil {
					mappings := shell.proxyServer.GetURLMappings()
					for _, mapping := range mappings {
						if strings.Contains(url, fmt.Sprintf(":%d", mapping.ProxyPort)) ||
							strings.Contains(url, mapping.TargetURL) {
							urlMsg.HasProxy = true
							urlMsg.ProxyPort = mapping.ProxyPort
							urlMsg.ProxyURL = mapping.ProxyURL
							urlMsg.ProxyLabel = mapping.Label
							break
						}
					}
				}

				shell.pendingURLs = append(shell.pendingURLs, urlMsg)
			}

			// Trigger debounced update (non-blocking)
			shell.logSub.NotifyNewLogs()
		})
	}

	// Initialize views
	shell.initializeViews()

	// Initialize styles
	shell.initializeStyles()

	// Create width calculator
	shell.widthCalc = NewTabWidthCalculator(shell.width, len(shell.viewOrder))

	return shell
}

// initializeViews sets up all view metadata
func (s *IntegratedShell) initializeViews() {
	s.views[ViewProcesses] = &ViewInfo{
		Title:      "Processes",
		Icon:       "🏃",
		KeyBinding: "1",
	}

	s.views[ViewLogs] = &ViewInfo{
		Title:      "Logs",
		Icon:       "📄",
		KeyBinding: "2",
	}

	s.views[ViewErrors] = &ViewInfo{
		Title:      "Errors",
		Icon:       "❌",
		KeyBinding: "3",
	}

	s.views[ViewURLs] = &ViewInfo{
		Title:      "URLs",
		Icon:       "🔗",
		KeyBinding: "4",
	}

	s.views[ViewWeb] = &ViewInfo{
		Title:      "Web",
		Icon:       "🌐",
		KeyBinding: "5",
	}

	s.views[ViewAICoders] = &ViewInfo{
		Title:      "AI",
		Icon:       "🤖",
		KeyBinding: "6",
	}

	s.views[ViewSettings] = &ViewInfo{
		Title:      "Settings",
		Icon:       "⚙️",
		KeyBinding: "7",
	}

	s.viewOrder = []ViewType{
		ViewProcesses,
		ViewLogs,
		ViewErrors,
		ViewURLs,
		ViewWeb,
		ViewAICoders,
		ViewSettings,
	}
}

// initializeStyles sets up all styles
func (s *IntegratedShell) initializeStyles() {
	s.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	s.activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("236"))

	s.inactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	s.errorBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52"))

	s.warnBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Background(lipgloss.Color("94"))

	s.infoBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("17"))

	s.separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
}

// Init initializes the shell
func (s *IntegratedShell) Init() tea.Cmd {
	// DEBUG: Log init - write directly to file
	if debugFile, err := os.OpenFile("/tmp/brummer_debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(debugFile, "[DEBUG] IntegratedShell.Init() called\n")
		debugFile.Close()
	}

	var cmds []tea.Cmd

	// Don't send an initial window size - wait for Bubble Tea to send the actual size
	// This prevents the initial 80x24 hardcoded size from being used

	// Initialize components
	if s.processView != nil {
		cmds = append(cmds, s.processView.Init())
	}
	if s.logsView != nil {
		cmds = append(cmds, s.logsView.Init())
	}

	// Start log subscription FIRST (incremental updates)
	if s.logSub != nil {
		cmds = append(cmds, s.logSub.Subscribe())
	}

	// Focus the initial view (processes)
	if s.processView != nil {
		cmds = append(cmds, s.processView.Focus())
	}

	// Publish event that log handlers are ready
	// This signals that processes can start without losing logs
	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.LogHandlersReady,
			Data: map[string]interface{}{
				"component": "IntegratedShell",
			},
		})
		s.eventBus.Publish(events.Event{
			Type: events.ComponentsReady,
			Data: map[string]interface{}{
				"component": "IntegratedShell",
			},
		})
	}

	// Load any existing logs that might have been added before subscription
	// This handles edge cases where logs exist from previous runs
	if s.logStore != nil {
		existingLogs := s.logStore.GetAll()
		if len(existingLogs) > 0 {
			// Convert to pointers and send initial logs to view
			entries := make([]*logs.LogEntry, len(existingLogs))
			for i := range existingLogs {
				entry := existingLogs[i] // Copy to avoid reference issues
				entries[i] = &entry
			}

			// Send as a command that returns LogsUpdateMsg immediately
			cmds = append(cmds, func() tea.Msg {
				return LogsUpdateMsg{
					Entries: entries,
					Append:  false, // Replace, not append
				}
			})
		}
	}

	return tea.Batch(cmds...)
}

// Update handles messages (implements tea.Model interface)
func (s *IntegratedShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// DEBUG: Log window size
		if os.Getenv("BRUMMER_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[DEBUG] WindowSizeMsg: %dx%d\n", msg.Width, msg.Height)
		}
		s.width = msg.Width
		s.height = msg.Height
		s.widthCalc = NewTabWidthCalculator(s.width, len(s.viewOrder))

		// Update all components
		if s.processView != nil {
			_, cmd := s.processView.Update(msg)
			cmds = append(cmds, cmd)
		}
		if s.logsView != nil {
			_, cmd := s.logsView.Update(msg)
			cmds = append(cmds, cmd)
		}
		if s.errorsView != nil {
			_, cmd := s.errorsView.Update(msg)
			cmds = append(cmds, cmd)
		}
		if s.urlsView != nil {
			_, cmd := s.urlsView.Update(msg)
			cmds = append(cmds, cmd)
		}
		if s.webView != nil {
			_, cmd := s.webView.Update(msg)
			cmds = append(cmds, cmd)
		}
		if s.aiView != nil {
			_, cmd := s.aiView.Update(msg)
			cmds = append(cmds, cmd)
		}
		if s.settingsView != nil {
			_, cmd := s.settingsView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		// ALWAYS handle global exit keys first
		switch msg.String() {
		case "ctrl+c":
			s.quit = true
			return s, tea.Quit

		case "q":
			// Only quit if not in an input field
			if !s.isInputFocused() {
				s.quit = true
				return s, tea.Quit
			}

		case "1", "2", "3", "4", "5", "6", "7":
			// Number keys always switch views (unless in input field)
			if !s.isInputFocused() {
				for _, view := range s.viewOrder {
					if s.views[view].KeyBinding == msg.String() {
						cmd := s.switchToView(view)
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
						break
					}
				}
				return s, tea.Batch(cmds...)
			}

		case "left", "alt+left":
			// Previous view with arrow keys
			if !s.isInputFocused() {
				cmd := s.previousView()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return s, tea.Batch(cmds...)
			}

		case "right", "alt+right":
			// Next view with arrow keys
			if !s.isInputFocused() {
				cmd := s.nextView()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return s, tea.Batch(cmds...)
			}

		case "/":
			// Show command palette
			if !s.isInputFocused() && !s.showCommandPalette {
				s.showCommandPalette = true
				s.commandPalette.Show()
				return s, nil
			}
		}

		// Handle command palette input if shown
		if s.showCommandPalette && s.commandPalette != nil {
			_, cmd := s.commandPalette.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return s, tea.Batch(cmds...)
		}

		// Pass all other keys (including Tab) to current view component
		cmd := s.handleViewInput(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ProcessUpdateMsg:
		// Update process name cache when processes change
		if msg.Process != nil {
			switch msg.UpdateType {
			case ProcessStarted:
				// Add new process to cache
				s.processNameCache.Store(msg.Process.ID, msg.Process.Name)
			case ProcessStopped:
				// Optionally remove from cache (we keep it for historical logs)
				// s.processNameCache.Delete(msg.Process.ID)
			}
		}
		// Pass to process view
		cmd := s.updateActiveComponent(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case LogsUpdateMsg:
		// Forward to logs view regardless of which view is active
		if s.logsView != nil {
			_, cmd := s.logsView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Process any pending errors and send to errors view
		if len(s.pendingErrors) > 0 && s.errorsView != nil {
			// Send all pending errors to the errors view
			for _, errMsg := range s.pendingErrors {
				_, cmd := s.errorsView.Update(errMsg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			// Clear pending errors after sending
			s.pendingErrors = nil
		}

		// Process any pending URLs and send to URLs view
		if len(s.pendingURLs) > 0 && s.urlsView != nil {
			// Get proxy mappings if available
			var proxyMappings []ProxyURLMapping
			if s.proxyServer != nil {
				proxyMappings = s.proxyServer.GetURLMappings()
			}

			// Send all pending URLs to both URLs view and Web view
			for _, urlMsg := range s.pendingURLs {
				// Check if this URL has a proxy mapping
				for _, mapping := range proxyMappings {
					if strings.Contains(urlMsg.URL, mapping.TargetURL) {
						// Add proxy info to the message
						urlMsg.HasProxy = true
						urlMsg.ProxyPort = mapping.ProxyPort
						urlMsg.ProxyURL = mapping.ProxyURL
						urlMsg.ProxyLabel = mapping.Label
						break
					}
				}

				// Send to URLs view
				_, cmd := s.urlsView.Update(urlMsg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}

				// Also send to Web view (which handles proxy display)
				if s.webView != nil {
					_, cmd := s.webView.Update(urlMsg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
			// Clear pending URLs after sending
			s.pendingURLs = nil
		}

		// Re-subscribe for more updates
		if s.logSub != nil {
			cmds = append(cmds, s.logSub.Subscribe())
		}

	case ContinueLogSubscriptionMsg:
		// Re-subscribe without processing (no new logs)
		if s.logSub != nil {
			cmds = append(cmds, s.logSub.Subscribe())
		}

	case HideCommandPaletteMsg:
		// Hide the command palette
		s.showCommandPalette = false
		if s.commandPalette != nil {
			s.commandPalette.Hide()
		}
		return s, nil

	case ShowCommandPaletteMsg:
		// Show the command palette
		s.showCommandPalette = true
		if s.commandPalette != nil {
			s.commandPalette.Show()
		}
		return s, nil

	case ShowAIViewMsg:
		// Switch to AI view when requested (e.g., after /ai provider)
		cmd := s.switchToView(ViewAICoders)
		return s, cmd

	default:
		// Pass message to active component
		cmd := s.updateActiveComponent(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update notification counts
	s.updateNotifications()

	return s, tea.Batch(cmds...)
}

// View renders the shell
func (s *IntegratedShell) View() string {
	// DEBUG: Check if we're even getting called - write directly to file
	if debugFile, err := os.OpenFile("/tmp/brummer_debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(debugFile, "[DEBUG IntegratedShell.View called: size=%dx%d, view=%v]\n",
			s.width, s.height, s.currentView)
		debugFile.Close()
	}

	// Render header
	header := s.renderHeader()

	// Calculate dimensions
	headerHeight := lipgloss.Height(header)
	footerHeight := 1
	contentHeight := s.height - headerHeight - footerHeight - 2 // -2 for border
	contentWidth := CalculateAvailableWidth(s.width, true)      // true = has border

	// Render current view content
	content := s.renderCurrentView(contentWidth, contentHeight)

	// Apply border
	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	contentBox := contentStyle.Render(content)

	// Render footer
	footer := s.renderFooter()

	// Combine all parts
	fullView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		contentBox,
		footer,
	)

	// Render command palette as overlay if shown
	if s.showCommandPalette && s.commandPalette != nil {
		// Center the command palette
		paletteView := s.commandPalette.View()

		// Create overlay style
		overlayStyle := lipgloss.NewStyle().
			Width(s.width).
			Height(s.height).
			Align(lipgloss.Center, lipgloss.Center)

		// Overlay the palette on top of the main view
		fullView = overlayStyle.Render(paletteView)
	}

	// Always return at least the debug message
	if fullView == "" {
		return "[DEBUG: IntegratedShell.View() returning empty view]"
	}
	return fullView
}

// renderHeader renders the header with title and tabs
func (s *IntegratedShell) renderHeader() string {
	// Title
	titleText := "🎯 Brummer Process Manager"
	title := s.titleStyle.Render(titleText)

	titleLine := lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title)

	// Tabs - left aligned with padding
	tabs := s.renderTabs()
	tabLine := lipgloss.NewStyle().
		Width(s.width).
		PaddingLeft(2). // Small left padding for better appearance
		Render(tabs)

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, tabLine)
}

// renderTabs renders the tab bar with notifications
func (s *IntegratedShell) renderTabs() string {
	var tabs []string

	for _, viewType := range s.viewOrder {
		view := s.views[viewType]
		tab := s.renderTab(viewType, view)
		tabs = append(tabs, tab)
	}

	separator := s.separatorStyle.Render(" │ ")
	return strings.Join(tabs, separator)
}

// renderTab renders a single tab
func (s *IntegratedShell) renderTab(viewType ViewType, view *ViewInfo) string {
	// Build label based on width mode
	var label string

	switch s.widthCalc.Mode {
	case "minimal":
		label = view.KeyBinding
		if view.HasNotif {
			label += "•"
		}

	case "compact":
		label = fmt.Sprintf("%s %s", view.Icon, view.KeyBinding)
		if view.HasNotif && view.NotifIcon != "" {
			label += fmt.Sprintf(" %s", view.NotifIcon)
		}

	case "normal":
		abbrev := s.getAbbreviation(view.Title)
		label = fmt.Sprintf("%s %s.%s", view.Icon, view.KeyBinding, abbrev)
		if view.HasNotif && view.NotifCount > 0 {
			label += fmt.Sprintf(" %s %d", view.NotifIcon, view.NotifCount)
		}

	default: // "full"
		label = fmt.Sprintf("%s %s.%s", view.Icon, view.KeyBinding, view.Title)
		if view.HasNotif && view.NotifCount > 0 {
			if view.NotifCount > 99 {
				label += fmt.Sprintf(" %s 99+", view.NotifIcon)
			} else {
				label += fmt.Sprintf(" %s %d", view.NotifIcon, view.NotifCount)
			}
		}
	}

	// Add selection indicator or reserve space
	var indicator string
	if viewType == s.currentView {
		indicator = "▶"
	} else {
		indicator = " " // Reserve space for alignment
	}

	// Combine indicator and label (no extra spacing in indicator)
	fullLabel := indicator + label

	// Apply appropriate style
	if viewType == s.currentView {
		return s.activeTabStyle.Render(fullLabel)
	}

	return s.inactiveTabStyle.Render(fullLabel)
}

// renderCurrentView renders the content of the current view
func (s *IntegratedShell) renderCurrentView(contentWidth, contentHeight int) string {
	switch s.currentView {
	case ViewProcesses:
		if s.processView != nil {
			// Set size for the component
			s.processView.SetSize(contentWidth, contentHeight)
			return s.processView.View()
		}

	case ViewLogs:
		if s.logsView != nil {
			// Set size for the component
			s.logsView.SetSize(contentWidth, contentHeight)
			return s.logsView.View()
		}

	case ViewErrors:
		if s.errorsView != nil {
			// Set size for the component
			s.errorsView.SetSize(contentWidth, contentHeight)
			return s.errorsView.View()
		}
		return s.renderPlaceholder("Errors View", "❌ Errors will be displayed here", contentWidth, contentHeight)

	case ViewURLs:
		if s.urlsView != nil {
			// Set size for the component
			s.urlsView.SetSize(contentWidth, contentHeight)
			return s.urlsView.View()
		}
		return s.renderPlaceholder("URLs View", "🔗 Discovered URLs will be listed here", contentWidth, contentHeight)

	case ViewWeb:
		if s.webView != nil {
			// Set size for the component (reuse URLs view for proxy display)
			s.webView.SetSize(contentWidth, contentHeight)
			return s.webView.View()
		}
		return s.renderPlaceholder("Web View", "🌐 Browser automation controls", contentWidth, contentHeight)

	case ViewAICoders:
		if s.aiView != nil {
			// Set size for the component
			s.aiView.SetSize(contentWidth, contentHeight)
			return s.aiView.View()
		}
		return s.renderPlaceholder("AI Coders", "🤖 AI coding assistants", contentWidth, contentHeight)

	case ViewSettings:
		if s.settingsView != nil {
			// Set size for the component
			s.settingsView.SetSize(contentWidth, contentHeight)
			return s.settingsView.View()
		}
		return s.renderPlaceholder("Settings", "⚙️ Configuration settings", contentWidth, contentHeight)
	}

	return "Unknown view"
}

// renderPlaceholder renders a placeholder for unimplemented views
func (s *IntegratedShell) renderPlaceholder(title, description string, width, height int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center, lipgloss.Center).
		Width(width).
		Height(height)

	content := fmt.Sprintf("%s\n\n%s\n\n(Component not yet implemented)", title, description)
	return style.Render(content)
}

// renderFooter renders the footer with help text
func (s *IntegratedShell) renderFooter() string {
	help := "1-5: Switch views | ←→: Navigate | "

	// Add context-sensitive help
	switch s.currentView {
	case ViewProcesses:
		help += "r: Restart | s: Stop | R: Restart All | "
	}

	help += "q: Quit"

	// Truncate if too wide
	if SafeStringWidth(help) > s.width {
		help = SafeTruncateWithEllipsis(help, s.width-4)
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(s.width).
		Align(lipgloss.Center).
		Padding(0, 1)

	return footerStyle.Render(help)
}

// switchToView switches to a specific view
func (s *IntegratedShell) switchToView(view ViewType) tea.Cmd {
	oldView := s.currentView
	s.currentView = view

	// Clear notifications for the new view
	if info := s.views[view]; info.HasNotif {
		info.HasNotif = false
		info.NotifCount = 0
	}

	var cmds []tea.Cmd

	// Blur old component
	if cmd := s.blurComponent(oldView); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Focus new component
	if cmd := s.focusComponent(view); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// nextView switches to the next view
func (s *IntegratedShell) nextView() tea.Cmd {
	currentIndex := 0
	for i, v := range s.viewOrder {
		if v == s.currentView {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(s.viewOrder)
	return s.switchToView(s.viewOrder[nextIndex])
}

// previousView switches to the previous view
func (s *IntegratedShell) previousView() tea.Cmd {
	currentIndex := 0
	for i, v := range s.viewOrder {
		if v == s.currentView {
			currentIndex = i
			break
		}
	}

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(s.viewOrder) - 1
	}
	return s.switchToView(s.viewOrder[prevIndex])
}

// focusComponent focuses the component for a view
func (s *IntegratedShell) focusComponent(view ViewType) tea.Cmd {
	switch view {
	case ViewProcesses:
		if s.processView != nil {
			return s.processView.Focus()
		}
	case ViewLogs:
		if s.logsView != nil {
			return s.logsView.Focus()
		}
	case ViewErrors:
		if s.errorsView != nil {
			return s.errorsView.Focus()
		}
	case ViewURLs:
		if s.urlsView != nil {
			return s.urlsView.Focus()
		}
	case ViewWeb:
		if s.webView != nil {
			return s.webView.Focus()
		}
	case ViewAICoders:
		if s.aiView != nil {
			return s.aiView.Focus()
		}
	case ViewSettings:
		if s.settingsView != nil {
			return s.settingsView.Focus()
		}
	}
	return nil
}

// blurComponent blurs the component for a view
func (s *IntegratedShell) blurComponent(view ViewType) tea.Cmd {
	switch view {
	case ViewProcesses:
		if s.processView != nil {
			return s.processView.Blur()
		}
	case ViewLogs:
		if s.logsView != nil {
			return s.logsView.Blur()
		}
	case ViewErrors:
		if s.errorsView != nil {
			return s.errorsView.Blur()
		}
	case ViewURLs:
		if s.urlsView != nil {
			return s.urlsView.Blur()
		}
	case ViewWeb:
		if s.webView != nil {
			return s.webView.Blur()
		}
	case ViewAICoders:
		if s.aiView != nil {
			return s.aiView.Blur()
		}
	case ViewSettings:
		if s.settingsView != nil {
			return s.settingsView.Blur()
		}
	}
	return nil
}

// handleViewInput passes input to the current view component
func (s *IntegratedShell) handleViewInput(key tea.KeyMsg) tea.Cmd {
	switch s.currentView {
	case ViewProcesses:
		if s.processView != nil && s.processView.Focused() {
			_, cmd := s.processView.Update(key)
			return cmd
		}
	case ViewLogs:
		if s.logsView != nil && s.logsView.Focused() {
			_, cmd := s.logsView.Update(key)
			return cmd
		}
	case ViewErrors:
		if s.errorsView != nil && s.errorsView.Focused() {
			_, cmd := s.errorsView.Update(key)
			return cmd
		}
	case ViewURLs:
		if s.urlsView != nil && s.urlsView.Focused() {
			_, cmd := s.urlsView.Update(key)
			return cmd
		}
	case ViewWeb:
		if s.webView != nil && s.webView.Focused() {
			_, cmd := s.webView.Update(key)
			return cmd
		}
	case ViewAICoders:
		if s.aiView != nil && s.aiView.Focused() {
			_, cmd := s.aiView.Update(key)
			return cmd
		}
	case ViewSettings:
		if s.settingsView != nil && s.settingsView.Focused() {
			_, cmd := s.settingsView.Update(key)
			return cmd
		}
	}
	return nil
}

// updateActiveComponent updates the active component with a message
func (s *IntegratedShell) updateActiveComponent(msg tea.Msg) tea.Cmd {
	switch s.currentView {
	case ViewProcesses:
		if s.processView != nil {
			_, cmd := s.processView.Update(msg)
			return cmd
		}
	case ViewLogs:
		if s.logsView != nil {
			_, cmd := s.logsView.Update(msg)
			return cmd
		}
	case ViewErrors:
		if s.errorsView != nil {
			_, cmd := s.errorsView.Update(msg)
			return cmd
		}
	case ViewURLs:
		if s.urlsView != nil {
			_, cmd := s.urlsView.Update(msg)
			return cmd
		}
	case ViewWeb:
		if s.webView != nil {
			_, cmd := s.webView.Update(msg)
			return cmd
		}
	case ViewAICoders:
		if s.aiView != nil {
			_, cmd := s.aiView.Update(msg)
			return cmd
		}
	case ViewSettings:
		if s.settingsView != nil {
			_, cmd := s.settingsView.Update(msg)
			return cmd
		}
	}
	return nil
}

// updateNotifications updates notification counts from components
func (s *IntegratedShell) updateNotifications() {
	// Update process view notifications
	if s.processView != nil {
		errors, warnings, info := s.processView.GetNotificationCount()

		processInfo := s.views[ViewProcesses]
		if errors > 0 {
			processInfo.HasNotif = true
			processInfo.NotifCount = errors
			processInfo.NotifIcon = "🔴"
			processInfo.NotifSeverity = "error"
		} else if warnings > 0 {
			processInfo.HasNotif = true
			processInfo.NotifCount = warnings
			processInfo.NotifIcon = "🟡"
			processInfo.NotifSeverity = "warning"
		} else if info > 0 {
			processInfo.HasNotif = true
			processInfo.NotifCount = info
			processInfo.NotifIcon = "🔵"
			processInfo.NotifSeverity = "info"
		} else {
			processInfo.HasNotif = false
			processInfo.NotifCount = 0
		}

		// Don't show notifications for current view
		if s.currentView == ViewProcesses {
			processInfo.HasNotif = false
			processInfo.NotifCount = 0
		}
	}
}

// getAbbreviation returns abbreviated form of title
func (s *IntegratedShell) getAbbreviation(title string) string {
	abbreviations := map[string]string{
		"Processes": "Proc",
		"Logs":      "Log",
		"Errors":    "Err",
		"URLs":      "URL",
		"Web":       "Web",
	}

	if abbr, ok := abbreviations[title]; ok {
		return abbr
	}

	return SafeTruncate(title, 4)
}

// isInputFocused checks if an input field is currently focused
func (s *IntegratedShell) isInputFocused() bool {
	// Check if command input or any text input is focused
	return false
}

// Quit returns whether the shell should quit
func (s *IntegratedShell) Quit() bool {
	return s.quit
}

// SetSize sets the shell size
func (s *IntegratedShell) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.widthCalc = NewTabWidthCalculator(s.width, len(s.viewOrder))
}

// Focus sets the component focus (implements Component interface)
func (s *IntegratedShell) Focus() tea.Cmd {
	// Focus the current view's component
	return s.focusComponent(s.currentView)
}

// Blur removes focus (implements Component interface)
func (s *IntegratedShell) Blur() tea.Cmd {
	// Blur the current view's component
	return s.blurComponent(s.currentView)
}

// Focused returns if the shell is focused (implements Component interface)
func (s *IntegratedShell) Focused() bool {
	// Check if the current view's component is focused
	switch s.currentView {
	case ViewProcesses:
		if s.processView != nil {
			return s.processView.Focused()
		}
	case ViewLogs:
		if s.logsView != nil {
			return s.logsView.Focused()
		}
	}
	return false
}

// GetLogStore returns the log store (for testing)
func (s *IntegratedShell) GetLogStore() *logs.Store {
	return s.logStore
}

// GetProcessManager returns the process manager (for testing)
func (s *IntegratedShell) GetProcessManager() *process.Manager {
	return s.processMgr
}

// GetProcessNameCache returns a copy of the process name cache (for testing)
func (s *IntegratedShell) GetProcessNameCache() map[string]string {
	cache := make(map[string]string)
	s.processNameCache.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			if v, ok := value.(string); ok {
				cache[k] = v
			}
		}
		return true
	})
	return cache
}

// GetProcessNameCacheSync returns the actual sync.Map for testing concurrent operations
func (s *IntegratedShell) GetProcessNameCacheSync() *sync.Map {
	return s.processNameCache
}

// NotifyNewLogs triggers a log update notification (for testing)
func (s *IntegratedShell) NotifyNewLogs() {
	if s.logSub != nil {
		s.logSub.NotifyNewLogs()
	}
}

// RefreshLogsView safely triggers a view update (for testing)
// This should only be called from the main Bubble Tea update loop in production
func (s *IntegratedShell) RefreshLogsView() {
	// In tests, we can call this but need to be careful about concurrency
	// In production, this would normally be triggered by Bubble Tea messages
	if s.logsView != nil {
		// Note: This is still potentially unsafe if called from multiple goroutines
		// but safer than direct field access
		s.logsView.applyFilter()
	}
}

// Cleanup releases resources when the shell is shutting down
func (s *IntegratedShell) Cleanup() {
	if s.logSub != nil {
		s.logSub.Stop()
	}
}

// Removed streamLogs - now using subscription pattern for incremental updates
