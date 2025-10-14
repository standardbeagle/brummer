package model

import (
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// View represents the different views in the application
type View int

const (
	ViewProcesses View = iota + 1
	ViewLogs
	ViewErrors
	ViewURLs
	ViewWeb
	ViewAICoders
	ViewSettings
)

// String returns the string representation of a View
func (v View) String() string {
	switch v {
	case ViewProcesses:
		return "Processes"
	case ViewLogs:
		return "Logs"
	case ViewErrors:
		return "Errors"
	case ViewURLs:
		return "URLs"
	case ViewWeb:
		return "Web"
	case ViewAICoders:
		return "AI Coders"
	case ViewSettings:
		return "Settings"
	default:
		return "Unknown"
	}
}

// Process represents a running process
type Process struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	State       string    `json:"state"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	ExitCode    int       `json:"exit_code"`
	PID         int       `json:"pid"`
	WorkingDir  string    `json:"working_dir"`
	Environment []string  `json:"environment"`
}

// IsRunning returns true if the process is currently running
func (p *Process) IsRunning() bool {
	return p.State == "running" || p.State == "starting"
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	ProcessID string    `json:"process_id"`
	Line      string    `json:"line"`
	Stream    string    `json:"stream"` // stdout, stderr
	Level     string    `json:"level"`  // info, warn, error
}

// ErrorEntry represents an error entry
type ErrorEntry struct {
	Timestamp time.Time `json:"timestamp"`
	ProcessID string    `json:"process_id"`
	Error     string    `json:"error"`
	Context   string    `json:"context"`
	Count     int       `json:"count"`
}

// URL represents a discovered URL
type URL struct {
	URL       string    `json:"url"`
	Label     string    `json:"label"`
	ProcessID string    `json:"process_id"`
	Detected  time.Time `json:"detected"`
	Hits      int       `json:"hits"`
}

// AISession represents an AI coder session
type AISession struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	State       string    `json:"state"`
	StartTime   time.Time `json:"start_time"`
	LastActive  time.Time `json:"last_active"`
	Model       string    `json:"model"`
	Context     []string  `json:"context"`
	MessageCount int      `json:"message_count"`
}

// Settings represents user configuration
type Settings struct {
	Theme              string            `json:"theme"`
	MaxLogEntries      int               `json:"max_log_entries"`
	MaxProcessHistory  int               `json:"max_process_history"`
	MaxURLCache        int               `json:"max_url_cache"`
	MaxErrorHistory    int               `json:"max_error_history"`
	AutoRefresh        bool              `json:"auto_refresh"`
	RefreshInterval    time.Duration     `json:"refresh_interval"`
	KeyboardShortcuts  map[string]string `json:"keyboard_shortcuts"`
	Debug              bool              `json:"debug"`
	WorkDir            string            `json:"work_dir"`
	Environment        []string          `json:"environment"`
}

// Theme represents UI theme settings
type Theme struct {
	Name        string `json:"name"`
	Primary     string `json:"primary"`
	Secondary   string `json:"secondary"`
	Background  string `json:"background"`
	Foreground  string `json:"foreground"`
	Error       string `json:"error"`
	Warning     string `json:"warning"`
	Success     string `json:"success"`
}

// ViewManager manages view state with atomic operations
type ViewManager struct {
	currentView atomic.Value // View
	viewHistory atomic.Value // []View
	maxHistory  int
}

// NewViewManager creates a new ViewManager
func NewViewManager() *ViewManager {
	vm := &ViewManager{
		maxHistory: 50,
	}
	vm.currentView.Store(ViewProcesses)
	vm.viewHistory.Store([]View{ViewProcesses})
	return vm
}

// SetCurrentView atomically sets the current view
func (vm *ViewManager) SetCurrentView(view View) []View {
	current := vm.currentView.Load().(View)
	if current == view {
		return vm.viewHistory.Load().([]View)
	}

	// Update history
	history := vm.viewHistory.Load().([]View)
	newHistory := append(history, view)

	// Limit history size
	if len(newHistory) > vm.maxHistory {
		newHistory = newHistory[1:]
	}

	vm.currentView.Store(view)
	vm.viewHistory.Store(newHistory)
	return newHistory
}

// GetCurrentView atomically gets the current view
func (vm *ViewManager) GetCurrentView() View {
	return vm.currentView.Load().(View)
}

// GetViewHistory atomically gets the view history
func (vm *ViewManager) GetViewHistory() []View {
	return vm.viewHistory.Load().([]View)
}

// CanGoBack returns true if we can navigate back in history
func (vm *ViewManager) CanGoBack() bool {
	history := vm.viewHistory.Load().([]View)
	return len(history) > 1
}

// GoBack navigates to the previous view in history
func (vm *ViewManager) GoBack() (View, bool) {
	history := vm.viewHistory.Load().([]View)
	if len(history) <= 1 {
		return vm.GetCurrentView(), false
	}

	// Remove current view from history
	newHistory := history[:len(history)-1]
	previousView := newHistory[len(newHistory)-1]

	vm.currentView.Store(previousView)
	vm.viewHistory.Store(newHistory)
	return previousView, true
}

// ProcessManager manages process state with atomic operations
type ProcessManager struct {
	processes       atomic.Value // map[string]*Process
	activeProcessID atomic.Value // string
	maxHistory      int
}

// NewProcessManager creates a new ProcessManager
func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		maxHistory: 100,
	}
	pm.processes.Store(make(map[string]*Process))
	pm.activeProcessID.Store("")
	return pm
}

// AddProcess atomically adds a process
func (pm *ProcessManager) AddProcess(process *Process) {
	processes := pm.processes.Load().(map[string]*Process)
	newProcesses := make(map[string]*Process)

	// Copy existing processes
	for k, v := range processes {
		newProcesses[k] = v
	}

	newProcesses[process.ID] = process

	// Limit history
	if len(newProcesses) > pm.maxHistory {
		// Remove oldest processes (simple FIFO)
		var oldestID string
		var oldestTime time.Time
		first := true

		for id, p := range newProcesses {
			if first || p.StartTime.Before(oldestTime) {
				oldestID = id
				oldestTime = p.StartTime
				first = false
			}
		}

		if oldestID != "" {
			delete(newProcesses, oldestID)
		}
	}

	pm.processes.Store(newProcesses)
}

// RemoveProcess atomically removes a process
func (pm *ProcessManager) RemoveProcess(id string) {
	processes := pm.processes.Load().(map[string]*Process)
	newProcesses := make(map[string]*Process)

	// Copy all processes except the one to remove
	for k, v := range processes {
		if k != id {
			newProcesses[k] = v
		}
	}

	pm.processes.Store(newProcesses)

	// Clear active process if it was removed
	if pm.GetActiveProcessID() == id {
		pm.SetActiveProcessID("")
	}
}

// GetProcesses atomically gets all processes
func (pm *ProcessManager) GetProcesses() map[string]*Process {
	processes := pm.processes.Load().(map[string]*Process)
	result := make(map[string]*Process)

	for k, v := range processes {
		result[k] = v
	}

	return result
}

// GetProcess atomically gets a specific process
func (pm *ProcessManager) GetProcess(id string) (*Process, bool) {
	processes := pm.processes.Load().(map[string]*Process)
	process, exists := processes[id]
	return process, exists
}

// SetActiveProcessID atomically sets the active process ID
func (pm *ProcessManager) SetActiveProcessID(id string) {
	pm.activeProcessID.Store(id)
}

// GetActiveProcessID atomically gets the active process ID
func (pm *ProcessManager) GetActiveProcessID() string {
	return pm.activeProcessID.Load().(string)
}

// GetActiveProcess atomically gets the active process
func (pm *ProcessManager) GetActiveProcess() (*Process, bool) {
	id := pm.GetActiveProcessID()
	if id == "" {
		return nil, false
	}
	return pm.GetProcess(id)
}

// Model is the central state store (Redux store)
type Model struct {
	// View Management
	viewManager *ViewManager

	// Process Management
	processManager *ProcessManager

	// Log State
	logs            atomic.Value // []LogEntry
	filteredLogs    atomic.Value // []LogEntry
	logFilter       atomic.Value // string
	scrollOffset    atomic.Value // int

	// Error State
	errors      atomic.Value // []ErrorEntry
	errorCounts atomic.Value // map[string]int

	// URL State
	urls      atomic.Value // map[string][]URL
	activeURL atomic.Value // string

	// AI Coder State
	aiSessions      atomic.Value // map[string]*AISession
	activeSessionID atomic.Value // string

	// Settings
	settings atomic.Value // Settings
	theme    atomic.Value // Theme

	// UI State
	windowWidth  atomic.Value // int
	windowHeight atomic.Value // int
	isFullscreen atomic.Value // bool

	// Command Palette
	paletteVisible atomic.Value // bool
	paletteQuery   atomic.Value // string
	paletteResults atomic.Value // []Command

	// Performance metrics
	lastUpdateTime time.Time
	updateCount    int64

	// Application state
	quitting      atomic.Bool
	initializing  atomic.Bool
}

// NewModel creates a new Model with default values
func NewModel() *Model {
	m := &Model{
		viewManager:    NewViewManager(),
		processManager: NewProcessManager(),
	}

	// Initialize atomic values
	m.logs.Store([]LogEntry{})
	m.filteredLogs.Store([]LogEntry{})
	m.logFilter.Store("")
	m.scrollOffset.Store(0)

	m.errors.Store([]ErrorEntry{})
	m.errorCounts.Store(make(map[string]int))

	m.urls.Store(make(map[string][]URL))
	m.activeURL.Store("")

	m.aiSessions.Store(make(map[string]*AISession))
	m.activeSessionID.Store("")

	// Default settings
	defaultSettings := Settings{
		Theme:              "default",
		MaxLogEntries:      10000,
		MaxProcessHistory:  100,
		MaxURLCache:        1000,
		MaxErrorHistory:    500,
		AutoRefresh:        true,
		RefreshInterval:    1 * time.Second,
		KeyboardShortcuts:  make(map[string]string),
		Debug:              false,
		WorkDir:            ".",
		Environment:        make([]string, 0),
	}
	m.settings.Store(defaultSettings)

	// Default theme
	defaultTheme := Theme{
		Name:       "default",
		Primary:    "#007ACC",
		Secondary:  "#6C757D",
		Background: "#282C34",
		Foreground: "#ABB2BF",
		Error:      "#F44747",
		Warning:    "#FFC107",
		Success:    "#28A745",
	}
	m.theme.Store(defaultTheme)

	// Default UI state
	m.windowWidth.Store(80)
	m.windowHeight.Store(24)
	m.isFullscreen.Store(false)

	// Command palette state
	m.paletteVisible.Store(false)
	m.paletteQuery.Store("")
	m.paletteResults.Store([]Command{})

	m.quitting.Store(false)
	m.initializing.Store(true)

	m.lastUpdateTime = time.Now()
	m.updateCount = 0

	return m
}

// View getters - thread-safe read access
func (m *Model) CurrentView() View {
	return m.viewManager.GetCurrentView()
}

func (m *Model) ViewHistory() []View {
	return m.viewManager.GetViewHistory()
}

func (m *Model) WindowWidth() int {
	return m.windowWidth.Load().(int)
}

func (m *Model) WindowHeight() int {
	return m.windowHeight.Load().(int)
}

func (m *Model) IsFullscreen() bool {
	return m.isFullscreen.Load().(bool)
}

// Process getters - thread-safe read access
func (m *Model) Processes() map[string]*Process {
	return m.processManager.GetProcesses()
}

func (m *Model) Process(id string) (*Process, bool) {
	return m.processManager.GetProcess(id)
}

func (m *Model) ActiveProcessID() string {
	return m.processManager.GetActiveProcessID()
}

func (m *Model) ActiveProcess() (*Process, bool) {
	return m.processManager.GetActiveProcess()
}

// Log getters - thread-safe read access
func (m *Model) Logs() []LogEntry {
	return m.logs.Load().([]LogEntry)
}

func (m *Model) FilteredLogs() []LogEntry {
	return m.filteredLogs.Load().([]LogEntry)
}

func (m *Model) LogFilter() string {
	return m.logFilter.Load().(string)
}

func (m *Model) ScrollOffset() int {
	return m.scrollOffset.Load().(int)
}

// Settings getters - thread-safe read access
func (m *Model) Settings() Settings {
	return m.settings.Load().(Settings)
}

func (m *Model) Theme() Theme {
	return m.theme.Load().(Theme)
}

// Additional getter methods for views
func (m *Model) Errors() []ErrorEntry {
	return m.errors.Load().([]ErrorEntry)
}

func (m *Model) IsInitializing() bool {
	return m.initializing.Load()
}

func (m *Model) SetInitializing(initializing bool) {
	m.initializing.Store(initializing)
}

// BubbleTea interface methods
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update will be implemented by the main application
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This will be handled by the main application to avoid circular imports
	return m, nil
}

// View will be implemented by the main application
func (m *Model) View() string {
	// This will be handled by the main application to avoid circular imports
	return "Redux/Elm TUI - Loading..."
}

// Command represents a command palette command
type Command struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Shortcut    string `json:"shortcut"`
	Action      func() tea.Msg `json:"-"`
}

// Setter methods for Model - needed by update handlers

// SetCurrentView sets the current view
func (m *Model) SetCurrentView(view View) {
	m.viewManager.SetCurrentView(view)
}

// AddToViewHistory adds a view to the history
func (m *Model) AddToViewHistory(view View) {
	m.viewManager.SetCurrentView(view)
}

// SetWindowSize sets the window dimensions
func (m *Model) SetWindowSize(width, height int) {
	m.windowWidth.Store(width)
	m.windowHeight.Store(height)
}

// SetFullscreen sets the fullscreen state
func (m *Model) SetFullscreen(fullscreen bool) {
	m.isFullscreen.Store(fullscreen)
}

// UpdateMetrics updates performance metrics
func (m *Model) UpdateMetrics() {
	m.updateCount++
	m.lastUpdateTime = time.Now()
}

// SetProcess adds or updates a process in the model
func (m *Model) SetProcess(process *Process) {
	m.processManager.AddProcess(process)
}

// DeleteProcess removes a process from the model
func (m *Model) DeleteProcess(id string) {
	m.processManager.RemoveProcess(id)
}

// SetActiveProcessID sets the active process ID
func (m *Model) SetActiveProcessID(id string) {
	m.processManager.SetActiveProcessID(id)
}

// SetLogFilter sets the log filter
func (m *Model) SetLogFilter(filter string) {
	m.logFilter.Store(filter)
}

// GetScrollOffset gets the current scroll offset
func (m *Model) GetScrollOffset() int {
	return m.scrollOffset.Load().(int)
}

// SetScrollOffset sets the scroll offset
func (m *Model) SetScrollOffset(offset int) {
	m.scrollOffset.Store(offset)
}

// SetSettings updates the settings
func (m *Model) SetSettings(settings Settings) {
	m.settings.Store(settings)
}