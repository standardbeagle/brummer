package tui

// UI Layout Constants
const (
	// Terminal size constraints
	MinTerminalWidth  = 20
	MinTerminalHeight = 10
	
	// Command window dimensions
	DefaultCommandWindowWidth = 60
	CommandWindowPadding      = 10
	
	// Script selector dimensions
	ScriptSelectorMinWidth  = 40
	ScriptSelectorMaxWidth  = 120
	ScriptSelectorMargin    = 20
	
	// Dropdown display limits
	MaxDropdownSuggestions      = 10
	MaxDropdownSuggestionsSmall = 5
	SmallTerminalHeightThreshold = 20
	
	// Centering calculations
	TitleAndHelpTextHeight = 4
	
	// System panel dimensions
	SystemPanelMaxMessages = 100
	
	// Update channel buffer size
	UpdateChannelBufferSize = 100
)

// Time constants
const (
	// Tick intervals
	DefaultTickInterval = "1s"
)

// Error messages
const (
	ErrProcessManagerNotInitialized = "process manager not initialized"
	ErrFailedToStartCommand        = "Failed to start command '%s' with args %v: %v"
	ErrFailedToStartScript         = "Failed to start script '%s': %v"
	ErrTerminalTooSmall           = "Terminal too small"
)