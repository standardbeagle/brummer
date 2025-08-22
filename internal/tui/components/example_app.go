// Package components provides an example application demonstrating the component system
package components

import (
	"fmt"
	"os"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/logs"
)

// ExampleApp demonstrates how to use the component system
type ExampleApp struct {
	// Layout manager
	layout *LayoutComponent
	
	// View components
	viewSwitcher    *ViewSwitcher
	processView     *ProcessViewComponent
	logsView        *LogsViewComponent
	errorsView      *ErrorsViewComponent
	urlsView        *URLsViewComponent
	webView         *WebViewComponent
	aiCodersView    *AICodersViewComponent
	settingsView    *SettingsViewComponent
	
	// UI components
	notifications   *NotificationComponent
	commandInput    *CommandInputComponent
	scriptSelector  *ScriptSelectorComponent
	
	// Header and footer
	header          *ExampleHeaderComponent
	footer          *ExampleFooterComponent
	
	// Application state
	width           int
	height          int
	ready           bool
	quitting        bool
}

// ExampleHeaderComponent displays application header for the example app
type ExampleHeaderComponent struct {
	BaseComponent
	title    string
	subtitle string
	style    lipgloss.Style
}

// ExampleFooterComponent displays status bar for the example app
type ExampleFooterComponent struct {
	BaseComponent
	status      string
	keybindings string
	style       lipgloss.Style
}

// NewExampleApp creates a new example application
func NewExampleApp() *ExampleApp {
	app := &ExampleApp{}
	
	// Initialize components
	app.initializeComponents()
	
	// Set up layout
	app.setupLayout()
	
	return app
}

// initializeComponents creates all components
func (app *ExampleApp) initializeComponents() {
	// Create process manager and log store
	processMgr, _ := process.NewManager(".", nil, false)
	logStore := logs.NewStore(1000, nil)
	
	// Create view components
	app.processView = NewProcessViewComponent(processMgr)
	app.logsView = NewLogsViewComponent(logStore)
	app.errorsView = NewErrorsViewComponent()
	app.urlsView = NewURLsViewComponent()
	app.webView = NewWebViewComponent()
	app.aiCodersView = NewAICodersViewComponent()
	app.settingsView = NewSettingsViewComponent()
	
	// Create UI components
	app.notifications = NewNotificationComponent(nil)
	app.commandInput = NewCommandInputComponent()
	app.scriptSelector = NewScriptSelectorComponent(processMgr, nil)
	
	// Create header and footer
	app.header = &ExampleHeaderComponent{
		title:    "🚀 Brummer Component System Demo",
		subtitle: "Modern TUI with Modular Components",
		style:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
	}
	app.header.visible = true
	
	app.footer = &ExampleFooterComponent{
		status:      "Ready",
		keybindings: "Tab: Switch View | Ctrl+P: Command | Ctrl+N: Notifications | Q: Quit",
		style:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}
	app.footer.visible = true
	
	// Create view switcher
	app.viewSwitcher = NewViewSwitcher()
	app.viewSwitcher.AddView("processes", app.processView)
	app.viewSwitcher.AddView("logs", app.logsView)
	app.viewSwitcher.AddView("errors", app.errorsView)
	app.viewSwitcher.AddView("urls", app.urlsView)
	app.viewSwitcher.AddView("web", app.webView)
	app.viewSwitcher.AddView("ai", app.aiCodersView)
	app.viewSwitcher.AddView("settings", app.settingsView)
	app.viewSwitcher.AddView("scripts", app.scriptSelector)
	
	// Set initial view
	app.viewSwitcher.SwitchTo("processes")
}

// setupLayout configures the layout
func (app *ExampleApp) setupLayout() {
	// Create layout component
	app.layout = NewLayoutComponent()
	app.layout.SetMode(LayoutModeHeaderFooter)
	
	// Set components in layout
	app.layout.SetHeader(app.header)
	app.layout.SetFooter(app.footer)
	app.layout.SetMain(app.viewSwitcher)
	app.layout.SetNotifications(app.notifications)
	app.layout.SetCommandPalette(app.commandInput)
}

// Init initializes the application
func (app *ExampleApp) Init() tea.Cmd {
	// Send initial welcome notification
	return tea.Batch(
		tea.EnterAltScreen,
		SendSystemNotification(NotificationSuccess,
			"Welcome!",
			"Component System Initialized"),
		app.simulateActivity(),
	)
}

// Update handles messages
func (app *ExampleApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle global keys first
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if !app.commandInput.Focused() {
				app.quitting = true
				return app, tea.Quit
			}
			
		case "tab":
			// Cycle through views
			views := []string{"processes", "logs", "errors", "urls", "web", "ai", "settings", "scripts"}
			// Since we don't have CurrentViewName, just cycle through
			// This is a simplified implementation
			app.viewSwitcher.SwitchTo(views[0])
			app.footer.status = fmt.Sprintf("View: %s", views[0])
			return app, nil
			
		case "ctrl+p":
			// Toggle command palette visibility
			if app.commandInput.visible {
				app.commandInput.visible = false
			} else {
				app.commandInput.visible = true
				return app, app.commandInput.Focus()
			}
			
		case "ctrl+n":
			// Show test notification
			return app, SendSystemNotification(NotificationInfo,
				"Test Notification",
				fmt.Sprintf("Time: %s", time.Now().Format("15:04:05")))
			
		case "ctrl+l":
			// Cycle layout modes
			modes := []LayoutMode{
				LayoutModeFull,
				LayoutModeSidebar,
				LayoutModeHeaderFooter,
				LayoutModeComplete,
			}
			currentMode := app.layout.mode
			nextMode := modes[(int(currentMode)+1)%len(modes)]
			app.layout.SetMode(nextMode)
			
			modeName := []string{"Full", "Sidebar", "Header/Footer", "Complete"}[nextMode]
			return app, SendSystemNotification(NotificationInfo,
				"Layout Changed",
				fmt.Sprintf("Mode: %s", modeName))
			
		case "1", "2", "3", "4", "5", "6", "7", "8":
			// Quick view switching
			views := []string{"processes", "logs", "errors", "urls", "web", "ai", "settings", "scripts"}
			index := int(msg.String()[0] - '1')
			if index < len(views) {
				app.viewSwitcher.SwitchTo(views[index])
				app.footer.status = fmt.Sprintf("View: %s", views[index])
			}
			return app, nil
		}
		
	case tea.WindowSizeMsg:
		app.width = msg.Width
		app.height = msg.Height
		app.ready = true
		
	case ViewChangeMsg:
		// Handle view change from components
		app.viewSwitcher.SwitchTo(msg.ViewID)
		app.footer.status = fmt.Sprintf("View: %s", msg.ViewID)
		return app, nil
		
	case SystemNotificationMsg:
		// Forward to notification component
		if app.notifications != nil {
			newNotif, cmd := app.notifications.Update(msg)
			app.notifications = newNotif.(*NotificationComponent)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	// Update layout (which updates all child components)
	newLayout, cmd := app.layout.Update(msg)
	app.layout = newLayout.(*LayoutComponent)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	
	return app, tea.Batch(cmds...)
}

// View renders the application
func (app *ExampleApp) View() string {
	if !app.ready {
		return "Initializing..."
	}
	
	if app.quitting {
		return "Goodbye! 👋\n"
	}
	
	return app.layout.View()
}

// simulateActivity generates demo activity
func (app *ExampleApp) simulateActivity() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		// Simulate various activities
		activities := []tea.Msg{
			ProcessUpdateMsg{
				Process: &process.Process{
					ID:     "demo-" + fmt.Sprintf("%d", t.Unix()),
					Name:   "Demo Process",
					Status: process.StatusRunning,
				},
				UpdateType: ProcessStarted,
			},
			LogsUpdateMsg{
				Entries: []*logs.LogEntry{
					{
						Timestamp: t,
						Level:     logs.LogLevel(1), // Info level
						Content:   fmt.Sprintf("Demo log entry at %s", t.Format("15:04:05")),
						ProcessID: "demo",
					},
				},
				Append: true,
			},
			URLDetectedMsg{
				URL:       fmt.Sprintf("http://localhost:%d", 3000+t.Second()%10),
				ProcessID: "demo",
				Type:      URLTypeHTTP,
				Timestamp: t,
			},
			ErrorOccurredMsg{
				ProcessID:   "demo",
				ProcessName: "Demo Process",
				Error:       fmt.Sprintf("Demo error at %s", t.Format("15:04:05")),
				Timestamp:   t,
			},
		}
		
		// Return a random activity
		return activities[t.Second()%len(activities)]
	})
}

// Header component implementation

func (h *ExampleHeaderComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
	}
	return h, nil
}

func (h *ExampleHeaderComponent) View() string {
	if !h.visible {
		return ""
	}
	
	title := h.style.Render(h.title)
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(h.subtitle)
	
	return fmt.Sprintf("%s\n%s", title, subtitle)
}

func (h *ExampleHeaderComponent) Focus() tea.Cmd {
	h.focused = true
	return nil
}

func (h *ExampleHeaderComponent) Blur() tea.Cmd {
	h.focused = false
	return nil
}

func (h *ExampleHeaderComponent) Focused() bool {
	return h.focused
}

// Footer component implementation

func (f *ExampleFooterComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
	}
	return f, nil
}

func (f *ExampleFooterComponent) View() string {
	if !f.visible {
		return ""
	}
	
	// Create a status bar
	statusWidth := f.width / 3
	keysWidth := f.width - statusWidth - 1
	
	status := f.style.Render(fmt.Sprintf("%-*s", statusWidth, f.status))
	keys := f.style.Render(fmt.Sprintf("%*s", keysWidth, f.keybindings))
	
	return status + " " + keys
}

func (f *ExampleFooterComponent) Focus() tea.Cmd {
	f.focused = true
	return nil
}

func (f *ExampleFooterComponent) Blur() tea.Cmd {
	f.focused = false
	return nil
}

func (f *ExampleFooterComponent) Focused() bool {
	return f.focused
}

// RunExampleApp runs the example application
func RunExampleApp() error {
	app := NewExampleApp()
	
	p := tea.NewProgram(app, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running example app: %w", err)
	}
	
	return nil
}

// main function for standalone execution
func main() {
	if err := RunExampleApp(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}