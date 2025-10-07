package components

// layout_manager.go - Pure function utilities for TUI layout calculations
// Migrated from layout_controller.go (T019) - Contains only pure layout math
// No dependencies on services or state - just calculations

// LayoutDimensions represents the calculated dimensions for TUI layout
type LayoutDimensions struct {
	Width              int
	Height             int
	HeaderHeight       int
	FooterHeight       int
	SystemPanelHeight  int
	AvailableHeight    int
	ProcessSelectorRow int
}

// LayoutConfig holds configuration for layout calculations
type LayoutConfig struct {
	TermWidth          int
	TermHeight         int
	HeaderHeight       int  // Default: 3
	FooterHeight       int  // Default: 1
	SystemPanelHeight  int  // Default: 10
	SystemPanelOpen    bool // Whether system panel is visible
	ShowProcessBar     bool // Whether to show process selector bar
}

// DefaultLayoutConfig returns the default layout configuration
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		HeaderHeight:      3,
		FooterHeight:      1,
		SystemPanelHeight: 10,
		SystemPanelOpen:   false,
		ShowProcessBar:    false,
	}
}

// CalculateDimensions performs pure layout calculations
// Returns all dimensions needed for rendering the TUI
func CalculateDimensions(config LayoutConfig) LayoutDimensions {
	dims := LayoutDimensions{
		Width:             config.TermWidth,
		Height:            config.TermHeight,
		HeaderHeight:      config.HeaderHeight,
		FooterHeight:      config.FooterHeight,
		SystemPanelHeight: 0, // Only shown when open
	}

	// Calculate system panel height (only if open)
	if config.SystemPanelOpen {
		// Limit system panel to 1/3 of terminal height
		maxSystemPanelHeight := config.TermHeight / 3
		if config.SystemPanelHeight > maxSystemPanelHeight {
			dims.SystemPanelHeight = maxSystemPanelHeight
		} else {
			dims.SystemPanelHeight = config.SystemPanelHeight
		}
	}

	// Calculate available height for main content
	usedHeight := dims.HeaderHeight + dims.FooterHeight + dims.SystemPanelHeight
	if config.ShowProcessBar {
		usedHeight += 1 // Process selector bar height
		dims.ProcessSelectorRow = dims.HeaderHeight
	}

	availableHeight := config.TermHeight - usedHeight
	if availableHeight < 1 {
		availableHeight = 1
	}
	dims.AvailableHeight = availableHeight

	return dims
}

// CalculateContentArea calculates the dimensions for the main content area
// Returns (width, height, topOffset)
func CalculateContentArea(dims LayoutDimensions, showProcessBar bool) (width, height, topOffset int) {
	width = dims.Width
	height = dims.AvailableHeight
	topOffset = dims.HeaderHeight

	if showProcessBar {
		topOffset += 1 // Process bar takes 1 row
		height -= 1    // Reduce available height
	}

	return width, height, topOffset
}

// CalculateSystemPanelArea calculates the dimensions for the system panel
// Returns (width, height, isEmpty)
func CalculateSystemPanelArea(dims LayoutDimensions) (width, height int, isEmpty bool) {
	if dims.SystemPanelHeight == 0 {
		return 0, 0, true
	}

	// Account for border (2 characters width)
	width = dims.Width - 2
	if width < 1 {
		width = 1
	}

	height = dims.SystemPanelHeight
	return width, height, false
}

// CalculateHelpBoxDimensions calculates centered help box dimensions
// Returns (width, height, centerX, centerY)
func CalculateHelpBoxDimensions(termWidth, termHeight int) (width, height, centerX, centerY int) {
	width = 60
	height = 25

	// Ensure help box fits in terminal
	if width > termWidth-4 {
		width = termWidth - 4
	}
	if height > termHeight-4 {
		height = termHeight - 4
	}

	// Calculate center position
	centerX = termWidth / 2
	centerY = termHeight / 2

	return width, height, centerX, centerY
}

// CalculateMaxSystemLogs calculates how many system logs fit in the panel
func CalculateMaxSystemLogs(systemPanelHeight int) int {
	// Account for border (2 lines)
	maxLogs := systemPanelHeight - 2
	if maxLogs < 1 {
		maxLogs = 1
	}
	return maxLogs
}

// CalculateViewportHeight calculates the height for a viewport widget
func CalculateViewportHeight(dims LayoutDimensions) int {
	return dims.AvailableHeight
}

// ShouldAdjustSystemPanel determines if system panel needs size adjustment
func ShouldAdjustSystemPanel(systemPanelHeight, termHeight int) (shouldAdjust bool, newHeight int) {
	maxHeight := termHeight / 3
	if systemPanelHeight > maxHeight {
		return true, maxHeight
	}
	return false, systemPanelHeight
}
