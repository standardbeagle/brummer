package redux

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui_redux/commands"
	"github.com/standardbeagle/brummer/internal/tui_redux/messages"
	"github.com/standardbeagle/brummer/internal/tui_redux/model"
	"github.com/standardbeagle/brummer/internal/tui_redux/views"
)

// ReduxApp wraps the BubbleTea program with Redux/Elm architecture
type ReduxApp struct {
	program *tea.Program
}

// NewReduxApp creates a new Redux/Elm TUI application
func NewReduxApp() *ReduxApp {
	// Initialize the model with default state
	m := model.NewModel()

	// Create the program with BubbleTea
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	}

	program := tea.NewProgram(&reduxModel{Model: m}, opts...)

	return &ReduxApp{
		program: program,
	}
}

// Run starts the TUI application
func (a *ReduxApp) Run() error {
	_, err := a.program.Run()
	return err
}

// Quit gracefully shuts down the application
func (a *ReduxApp) Quit() {
	a.program.Quit()
}

// reduxModel wraps the model to implement the Update/View pattern without circular imports
type reduxModel struct {
	*model.Model
}

// Init initializes the model
func (m *reduxModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return messages.StatusUpdateMsg{
				Message: "Redux/Elm TUI initialized",
				Level:   "info",
			}
		},
		commands.LoadSettings(""),
	)
}

// Update handles messages using the Redux pattern
func (m *reduxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update performance metrics
	m.UpdateMetrics()

	// Route message to appropriate handler
	switch msg := msg.(type) {
	// View messages
	case messages.ViewSwitchedMsg:
		return handleViewSwitched(m, msg)
	case messages.WindowResizedMsg:
		return handleWindowResized(m, msg)
	case messages.FullscreenToggleMsg:
		return handleFullscreenToggle(m)

	// Process messages
	case messages.ProcessStartedMsg:
		return handleProcessStarted(m, msg)
	case messages.ProcessStoppedMsg:
		return handleProcessStopped(m, msg)
	case messages.ProcessSelectedMsg:
		return handleProcessSelected(m, msg)

	// Log messages
	case messages.LogFilterChangedMsg:
		return handleLogFilterChanged(m, msg)
	case messages.LogScrollMsg:
		return handleLogScroll(m, msg)

	// Settings messages
	case messages.SettingsUpdatedMsg:
		return handleSettingsUpdated(m, msg)
	case messages.StatusUpdateMsg:
		return handleStatusUpdate(m, msg)

	// System messages
	case messages.QuitMsg:
		return m, tea.Quit
	case messages.TickMsg:
		return handleTick(m, msg)

	// BubbleTea framework messages
	case tea.KeyMsg:
		return handleKeyMsg(m, msg)
	case tea.WindowSizeMsg:
		return handleWindowSizeMsg(m, msg)

	// Unknown message type - return model unchanged
	default:
		return m, nil
	}
}

// View renders the TUI
func (m *reduxModel) View() string {
	return views.Render(*m.Model)
}

// Message handlers

func handleViewSwitched(m *reduxModel, msg messages.ViewSwitchedMsg) (tea.Model, tea.Cmd) {
	m.SetCurrentView(msg.NewView)
	m.AddToViewHistory(msg.NewView)
	return m, nil
}

func handleWindowResized(m *reduxModel, msg messages.WindowResizedMsg) (tea.Model, tea.Cmd) {
	m.SetWindowSize(msg.Width, msg.Height)
	return m, nil
}

func handleFullscreenToggle(m *reduxModel) (tea.Model, tea.Cmd) {
	m.SetFullscreen(!m.IsFullscreen())
	return m, nil
}

func handleProcessStarted(m *reduxModel, msg messages.ProcessStartedMsg) (tea.Model, tea.Cmd) {
	m.SetProcess(msg.Process)
	return m, nil
}

func handleProcessStopped(m *reduxModel, msg messages.ProcessStoppedMsg) (tea.Model, tea.Cmd) {
	m.DeleteProcess(msg.ProcessID)
	return m, nil
}

func handleProcessSelected(m *reduxModel, msg messages.ProcessSelectedMsg) (tea.Model, tea.Cmd) {
	m.SetActiveProcessID(msg.ProcessID)
	return m, nil
}

func handleLogFilterChanged(m *reduxModel, msg messages.LogFilterChangedMsg) (tea.Model, tea.Cmd) {
	m.SetLogFilter(msg.Filter)
	return m, nil
}

func handleLogScroll(m *reduxModel, msg messages.LogScrollMsg) (tea.Model, tea.Cmd) {
	// Simple scroll implementation
	offset := m.GetScrollOffset()
	switch msg.Direction {
	case "up":
		offset = max(0, offset-msg.Amount)
	case "down":
		offset = offset + msg.Amount
	case "top":
		offset = 0
	case "bottom":
		// Set to max (will be calculated based on log count)
		offset = 1000
	}
	m.SetScrollOffset(offset)
	return m, nil
}

func handleSettingsUpdated(m *reduxModel, msg messages.SettingsUpdatedMsg) (tea.Model, tea.Cmd) {
	m.SetSettings(msg.Settings)
	return m, nil
}

func handleStatusUpdate(m *reduxModel, msg messages.StatusUpdateMsg) (tea.Model, tea.Cmd) {
	// For now, just log status updates
	// In a real implementation, these would be displayed in the UI
	return m, nil
}

func handleTick(m *reduxModel, msg messages.TickMsg) (tea.Model, tea.Cmd) {
	// Periodic updates could go here
	return m, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return messages.NewTickMsg()
	})
}

func handleKeyMsg(m *reduxModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Extract character input
	if len(msg.Runes) > 0 {
		switch msg.Runes[0] {
		case '1':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewProcesses))
		case '2':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewLogs))
		case '3':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewErrors))
		case '4':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewURLs))
		case '5':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewWeb))
		case '6':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewAICoders))
		case '7':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewSettings))
		case 'q':
			return m, tea.Quit
		}
	}

	// Handle special keys
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyTab:
		// Cycle to next view
		nextView := m.CurrentView() + 1
		if nextView > model.ViewSettings {
			nextView = model.ViewProcesses
		}
		return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), nextView))
	}

	return m, nil
}

func handleWindowSizeMsg(m *reduxModel, msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	windowMsg := messages.WindowResizedMsg{
		Width:  msg.Width,
		Height: msg.Height,
	}
	return handleWindowResized(m, windowMsg)
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Main entry point
func Main() error {
	app := NewReduxApp()

	if err := app.Run(); err != nil {
		return fmt.Errorf("failed to run Redux/Elm TUI: %w", err)
	}

	return nil
}