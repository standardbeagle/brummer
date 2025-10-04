package components

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// IMPORTANT TTY TESTING WARNING:
// ========================================
// This component deals with WindowSizeMsg events which require proper TTY allocation.
//
// ⚠️  CRITICAL: Do NOT test this component using Bash tools that have TTY support disabled!
//
// AI assistants working on this code MUST:
// 1. Use proper terminal testing (run ./brum directly in real terminal)
// 2. Never use automated tools like 'bash' or 'timeout' for TUI testing
// 3. Test WindowSizeMsg functionality by resizing the terminal manually
// 4. Verify real-time updates by changing terminal size during testing
//
// The Bash tool in Claude Code runs in a non-TTY environment and will always fail
// with "could not open a new TTY: open /dev/tty: no such device or address"
// This does NOT indicate a problem with the code - it's a testing limitation.
//
// For proper testing, use:
// - Direct terminal execution: ./brum
// - Integration tests with TTY allocation
// - Manual terminal resizing to verify WindowSizeMsg events

// DiagnosticData holds all timing and sizing data atomically
// This struct should only be accessed via atomic operations
type DiagnosticData struct {
	// Terminal dimensions
	TerminalWidth  int32
	TerminalHeight int32
	ShellWidth     int32
	ShellHeight    int32

	// Component sizing info
	HeaderHeight  int32
	TabsHeight    int32
	FooterHeight  int32
	ContentHeight int32

	// WindowSizeMsg tracking
	WindowMsgCount int64
	LastWindowMsg  time.Time
}

// DiagnosticViewComponent shows detailed information about TUI sizing and calculations
// for debugging window/viewport issues.
// All diagnostic data is stored atomically in a single struct for consistency.
type DiagnosticViewComponent struct {
	BaseComponent

	// Store diagnostic information atomically in a single struct
	data atomic.Value // stores *DiagnosticData

	// Viewport for scrolling content
	viewport viewport.Model

	// Styles (immutable, no atomic needed)
	titleStyle   lipgloss.Style
	labelStyle   lipgloss.Style
	valueStyle   lipgloss.Style
	borderStyle  lipgloss.Style
	warningStyle lipgloss.Style
	successStyle lipgloss.Style
}

// NewDiagnosticViewComponent creates a new diagnostic view
func NewDiagnosticViewComponent() *DiagnosticViewComponent {
	c := &DiagnosticViewComponent{
		BaseComponent: NewBaseComponent(),
		viewport:     viewport.New(80, 24), // Default size, will be updated in SetSize

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")). // Yellow
			Bold(true).
			Background(lipgloss.Color("53")),

		labelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Bright Blue
			Bold(true),

		valueStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White

		borderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Dark Gray
			Bold(true),

		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")). // Orange
			Bold(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")). // Green
			Bold(true),
	}

	// Initialize atomic data with empty struct
	c.data.Store(&DiagnosticData{})

	return c
}

// getData returns the current diagnostic data atomically
// Always use this method to access diagnostic data
func (c *DiagnosticViewComponent) getData() *DiagnosticData {
	if dataInterface := c.data.Load(); dataInterface != nil {
		return dataInterface.(*DiagnosticData)
	}
	return &DiagnosticData{}
}

// updateData atomically updates the diagnostic data
// Use this method to safely update all fields atomically
func (c *DiagnosticViewComponent) updateData(updater func(*DiagnosticData)) {
	current := c.getData()
	newData := *current // Make a copy
	updater(&newData)
	c.data.Store(&newData)
}

// Init initializes the component
func (c *DiagnosticViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages and captures sizing information
func (c *DiagnosticViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update WindowSizeMsg tracking data
		c.updateData(func(data *DiagnosticData) {
			data.WindowMsgCount++
			data.LastWindowMsg = time.Now()

			// Store dimensions
			data.TerminalWidth = int32(msg.Width)
			data.TerminalHeight = int32(msg.Height)
		})

		// Fallback: if dimensions are zero, try to get terminal size directly
		if msg.Width == 0 || msg.Height == 0 {
			if w, h := c.getTerminalSize(); w > 0 && h > 0 {
				c.updateData(func(data *DiagnosticData) {
					data.TerminalWidth = int32(w)
					data.TerminalHeight = int32(h)
				})
			}
		}

		// Calculate expected component heights
		c.calculateExpectedHeights()

		// Force a re-render by returning a command that triggers a refresh
		return c, func() tea.Msg {
			return tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height}
		}

	case tea.KeyMsg:
		// Forward keyboard input to viewport for scrolling
		var cmd tea.Cmd
		c.viewport, cmd = c.viewport.Update(msg)
		return c, cmd

	case tea.MouseMsg:
		// Update on mouse activity - no logging needed
	}

	// Forward other messages to viewport
	c.viewport, _ = c.viewport.Update(msg)

	// Update base component size
	data := c.getData()
	c.BaseComponent.SetSize(int(data.TerminalWidth), int(data.TerminalHeight))

	return c, nil
}

// SetSize updates component dimensions
func (c *DiagnosticViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
}

// calculateExpectedHeights calculates expected heights for each component
func (c *DiagnosticViewComponent) calculateExpectedHeights() {
	data := c.getData()

	// These are the expected heights based on the actual component implementations
	headerHeight := int32(2)  // SimpleHeaderComponent uses 2 lines (header + divider)
	tabsHeight := int32(1)     // TabsComponent renders on 1 line
	footerHeight := int32(1)   // FooterComponent renders on 1 line

	contentHeight := data.TerminalHeight - headerHeight - tabsHeight - footerHeight

	// Ensure minimum content height
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Update data with new heights
	c.updateData(func(d *DiagnosticData) {
		d.HeaderHeight = headerHeight
		d.TabsHeight = tabsHeight
		d.FooterHeight = footerHeight
		d.ContentHeight = contentHeight
	})
}

// View renders the diagnostic information
func (c *DiagnosticViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	// Ensure terminal size is set - if still zero, try detection now
	data := c.getData()

	if data.TerminalWidth == 0 || data.TerminalHeight == 0 {
		// Try terminal size detection during rendering as a fallback
		if w, h := c.getTerminalSize(); w > 0 && h > 0 {
			c.updateData(func(d *DiagnosticData) {
				d.TerminalWidth = int32(w)
				d.TerminalHeight = int32(h)
			})
			c.calculateExpectedHeights()
		}
		// Don't apply false defaults - keep zeros to indicate detection failure
	}

	// Build diagnostic information
	var sections []string

	// Header
	header := c.titleStyle.Render("🔍 TUI DIAGNOSTIC PANEL")
	sections = append(sections, header)
	sections = append(sections, "")

	// Terminal Information
	terminalInfo := c.buildTerminalInfo()
	sections = append(sections, terminalInfo)

	// Component Height Calculations
	heightInfo := c.buildHeightCalculations()
	sections = append(sections, heightInfo)

	// Visual Calibration
	calibration := c.buildVisualCalibration()
	sections = append(sections, calibration)

	// Timing Information
	timingInfo := c.buildTimingInfo()
	sections = append(sections, timingInfo)

	// Recommendations
	recommendations := c.buildRecommendations()
	sections = append(sections, recommendations)

	// Join all sections
	content := strings.Join(sections, "\n")

	// Set content in viewport and let it handle scrolling
	c.viewport.SetContent(content)
	c.viewport.GotoTop() // Start at top for new content

	return c.viewport.View()
}

// buildTerminalInfo creates the terminal information section
func (c *DiagnosticViewComponent) buildTerminalInfo() string {
	var lines []string

	lines = append(lines, c.labelStyle.Render("📺 TERMINAL INFORMATION"))
	lines = append(lines, "")

	// Environment detection
	isWSL2 := c.isWSL2Environment()
	isWindowsTerminal := c.isWindowsTerminal()

	if isWSL2 && isWindowsTerminal {
		lines = append(lines, c.valueStyle.Render("  Environment: WSL2 + Windows Terminal"))
	} else if isWSL2 {
		lines = append(lines, c.valueStyle.Render("  Environment: WSL2"))
	} else if isWindowsTerminal {
		lines = append(lines, c.valueStyle.Render("  Environment: Windows Terminal"))
	} else {
		lines = append(lines, c.valueStyle.Render("  Environment: Traditional Linux"))
	}

	// Get atomic data
	data := c.getData()

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Width:  %s", c.valueStyle.Render(fmt.Sprintf("%d", data.TerminalWidth))))
	lines = append(lines, fmt.Sprintf("  Height: %s", c.valueStyle.Render(fmt.Sprintf("%d", data.TerminalHeight))))

	// Add WindowSizeMsg statistics
	lines = append(lines, fmt.Sprintf("  WindowSizeMsg Count: %s", c.valueStyle.Render(fmt.Sprintf("%d", data.WindowMsgCount))))
	if !data.LastWindowMsg.IsZero() {
		lines = append(lines, fmt.Sprintf("  Last WindowSizeMsg: %s", c.valueStyle.Render(data.LastWindowMsg.Format("15:04:05.000"))))
	}

	// Add debugging info
	if data.TerminalWidth == 0 || data.TerminalHeight == 0 {
		lines = append(lines, "")
		lines = append(lines, c.warningStyle.Render("⚠️  Terminal size detection failed"))
		lines = append(lines, c.valueStyle.Render("  All detection methods returned zero"))

		// Show key environment variables for debugging
		lines = append(lines, "")
		lines = append(lines, c.labelStyle.Render("🔍 DEBUG INFO:"))
		if wtSession := os.Getenv("WT_SESSION"); wtSession != "" {
			lines = append(lines, fmt.Sprintf("  WT_SESSION: %s", wtSession[:8]+"..."))
		}
		if wtProfile := os.Getenv("WT_PROFILE_ID"); wtProfile != "" {
			lines = append(lines, fmt.Sprintf("  WT_PROFILE_ID: %s", wtProfile[:8]+"..."))
		}
		if wslDistro := os.Getenv("WSL_DISTRO_NAME"); wslDistro != "" {
			lines = append(lines, fmt.Sprintf("  WSL_DISTRO_NAME: %s", wslDistro))
		}
	} else {
		// Check for reasonable dimensions
		if data.TerminalWidth < 80 || data.TerminalHeight < 24 {
			lines = append(lines, "")
			lines = append(lines, c.warningStyle.Render("⚠️  Terminal size below recommended minimum (80x24)"))
		} else {
			lines = append(lines, "")
			lines = append(lines, c.successStyle.Render("✅ Terminal size is adequate"))
		}
	}

	return strings.Join(lines, "\n")
}

// buildHeightCalculations creates the height calculations section
func (c *DiagnosticViewComponent) buildHeightCalculations() string {
	var lines []string

	// Get atomic data
	data := c.getData()

	lines = append(lines, c.labelStyle.Render("📐 COMPONENT HEIGHT CALCULATIONS"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Header Height:    %s", c.valueStyle.Render(fmt.Sprintf("%d", data.HeaderHeight))))
	lines = append(lines, fmt.Sprintf("  Tabs Height:      %s", c.valueStyle.Render(fmt.Sprintf("%d", data.TabsHeight))))
	lines = append(lines, fmt.Sprintf("  Footer Height:    %s", c.valueStyle.Render(fmt.Sprintf("%d", data.FooterHeight))))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Content Height:   %s", c.valueStyle.Render(fmt.Sprintf("%d", data.ContentHeight))))
	lines = append(lines, "")

	// Show calculation
	total := data.HeaderHeight + data.TabsHeight + data.FooterHeight + data.ContentHeight
	lines = append(lines, fmt.Sprintf("  Total Calculated: %s", c.valueStyle.Render(fmt.Sprintf("%d", total))))

	if total == data.TerminalHeight {
		lines = append(lines, c.successStyle.Render("✅ Height calculations match terminal height"))
	} else {
		lines = append(lines, c.warningStyle.Render(fmt.Sprintf("⚠️  Height mismatch: %d (calc) vs %d (terminal)", total, data.TerminalHeight)))
	}

	return strings.Join(lines, "\n")
}

// buildVisualCalibration creates the visual calibration section
func (c *DiagnosticViewComponent) buildVisualCalibration() string {
	var lines []string

	lines = append(lines, c.labelStyle.Render("🎨 VISUAL CALIBRATION"))
	lines = append(lines, "")
	lines = append(lines, c.valueStyle.Render("Lines should be numbered to verify vertical alignment:"))
	lines = append(lines, "")

	// Number lines for visual verification
	data := c.getData()
	terminalHeight := int(data.TerminalHeight)
	headerHeight := int(data.HeaderHeight)
	tabsHeight := int(data.TabsHeight)
	footerHeight := int(data.FooterHeight)

	maxLines := min(terminalHeight-4, 20) // Leave space for borders and other sections
	for i := 1; i <= maxLines; i++ {
		lineNum := fmt.Sprintf("%3d", i)

		// Add markers for key lines
		switch i {
		case headerHeight:
			lines = append(lines, fmt.Sprintf("%s | %s", lineNum, c.warningStyle.Render("← EXPECTED HEADER END")))
		case headerHeight + tabsHeight:
			lines = append(lines, fmt.Sprintf("%s | %s", lineNum, c.warningStyle.Render("← EXPECTED TABS END")))
		case terminalHeight - footerHeight:
			lines = append(lines, fmt.Sprintf("%s | %s", lineNum, c.warningStyle.Render("← EXPECTED FOOTER START")))
		default:
			lines = append(lines, fmt.Sprintf("%s |", lineNum))
		}
	}

	if maxLines < terminalHeight-4 {
		lines = append(lines, fmt.Sprintf("... (%d more lines)", terminalHeight-4-maxLines))
	}

	return strings.Join(lines, "\n")
}

// buildTimingInfo creates the timing information section
func (c *DiagnosticViewComponent) buildTimingInfo() string {
	var lines []string

	lines = append(lines, c.labelStyle.Render("⏰ TIMING INFORMATION"))
	lines = append(lines, "")

	// Get atomic data
	data := c.getData()

	if !data.LastWindowMsg.IsZero() {
		lines = append(lines, fmt.Sprintf("  Last Update: %s", c.valueStyle.Render(data.LastWindowMsg.Format("15:04:05.000"))))
		// Calculate time since last update
		elapsed := time.Since(data.LastWindowMsg)
		lines = append(lines, fmt.Sprintf("  Time Since: %s", c.valueStyle.Render(elapsed.String())))
	} else {
		lines = append(lines, c.valueStyle.Render("  Last Update: No WindowSizeMsg received"))
	}

	return strings.Join(lines, "\n")
}

// buildRecommendations creates troubleshooting recommendations
func (c *DiagnosticViewComponent) buildRecommendations() string {
	var lines []string

	lines = append(lines, c.labelStyle.Render("💡 TROUBLESHOOTING RECOMMENDATIONS"))
	lines = append(lines, "")

	data := c.getData()
	terminalHeight := int(data.TerminalHeight)
	contentHeight := int(data.ContentHeight)
	terminalWidth := int(data.TerminalWidth)

	if terminalHeight < 30 {
		lines = append(lines, c.warningStyle.Render("• Terminal height may be too small for optimal display"))
		lines = append(lines, c.valueStyle.Render("  Consider resizing terminal to at least 30 lines"))
	}

	if contentHeight < 10 {
		lines = append(lines, c.warningStyle.Render("• Content area may be too small"))
		lines = append(lines, c.valueStyle.Render("  This could indicate incorrect height calculations"))
	}

	if terminalWidth < 100 {
		lines = append(lines, c.warningStyle.Render("• Terminal width may limit component display"))
		lines = append(lines, c.valueStyle.Render("  Consider using wider terminal (100+ characters)"))
	}

	if len(lines) > 2 { // If we added recommendations
		lines = append(lines, "")
		lines = append(lines, c.successStyle.Render("✅ Use this panel to verify TUI sizing issues"))
	} else {
		lines = append(lines, c.successStyle.Render("✅ No sizing issues detected"))
	}

	return strings.Join(lines, "\n")
}

// min returns the minimum of two integers
// isWSL2Environment detects if we're running in WSL2
func (c *DiagnosticViewComponent) isWSL2Environment() bool {
	// Method 1: Check /proc/version for WSL2
	if versionBytes, err := os.ReadFile("/proc/version"); err == nil {
		version := string(versionBytes)
		if strings.Contains(version, "WSL2") || strings.Contains(version, "microsoft") {
			return true
		}
	}

	// Method 2: Check for WSL-specific environment variables
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSLENV") != "" {
		return true
	}

	return false
}

// isWindowsTerminal detects if we're running in Windows Terminal
func (c *DiagnosticViewComponent) isWindowsTerminal() bool {
	// Check for Windows Terminal specific environment variables
	if os.Getenv("WT_SESSION") != "" || os.Getenv("WT_PROFILE_ID") != "" {
		return true
	}

	// Check for Windows Terminal in terminal program name
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		if strings.Contains(strings.ToLower(termProgram), "vscode") ||
		   strings.Contains(strings.ToLower(termProgram), "windows terminal") {
			return true
		}
	}

	return false
}

// getTerminalSizeWindowsTerminal uses Windows Terminal specific detection
func (c *DiagnosticViewComponent) getTerminalSizeWindowsTerminal() (width, height int) {
	// Method 1: Try Windows Terminal specific environment variables
	// Note: WT_WIDTH and WT_HEIGHT aren't always set, but we check anyway
	if cols := os.Getenv("WT_WIDTH"); cols != "" {
		if w, err := strconv.Atoi(cols); err == nil && w > 0 {
			width = w
		}
	}
	if lines := os.Getenv("WT_HEIGHT"); lines != "" {
		if h, err := strconv.Atoi(lines); err == nil && h > 0 {
			height = h
		}
	}

	// Method 2: Try to get size from WSL environment forwarding
	// Windows Terminal forwards WT_SESSION and WT_PROFILE_ID via WSLENV
	// This indicates we're in Windows Terminal, but doesn't give size directly

	// Method 3: Try reading the Windows Terminal size via alternate methods
	// In Windows Terminal + WSL2, sometimes the size is available through different means

	// Method 4: Check for Windows Terminal's default sizes
	// Windows Terminal often starts with common default sizes
	if width == 0 || height == 0 {
		// Check tput first to see if we can get actual size
		if w, h := c.getTerminalSizeViaTput(); w > 0 && h > 0 {
			return w, h
		}

		// If tput fails, we might be in a non-interactive context
		// In that case, we return 0,0 to indicate detection failure
	}

	// Method 5: Try environment variables that might be set by Windows Terminal
	if width == 0 || height == 0 {
		if w, h := c.getTerminalSizeFromEnv(); w > 0 && h > 0 {
			return w, h
		}
	}

	return width, height
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getTerminalSize attempts to get terminal dimensions directly
func (c *DiagnosticViewComponent) getTerminalSize() (width, height int) {
	// Detect the environment first
	isWSL2 := c.isWSL2Environment()
	isWindowsTerminal := c.isWindowsTerminal()

	// Method 1: Windows Terminal + WSL2 specific approach
	if isWSL2 && isWindowsTerminal {
		if w, h := c.getTerminalSizeWindowsTerminal(); w > 0 && h > 0 {
			return w, h
		}
	}

	// Method 2: Try using tput (works well in WSL2 environments)
	if runtime.GOOS != "windows" {
		if w, h := c.getTerminalSizeViaTput(); w > 0 && h > 0 {
			return w, h
		}
	}

	// Method 3: Try environment variables (Windows Terminal sets these)
	if w, h := c.getTerminalSizeFromEnv(); w > 0 && h > 0 {
		return w, h
	}

	// Method 4: Try using stty (Unix) - less reliable in WSL2
	if runtime.GOOS != "windows" {
		if w, h := c.getTerminalSizeViaStty(); w > 0 && h > 0 {
			return w, h
		}
	}

	// Method 5: Fail fast - don't provide false defaults
	// Return zeros to indicate failure rather than misleading defaults
	return 0, 0
}

// getTerminalSizeViaTput uses tput to get terminal size
func (c *DiagnosticViewComponent) getTerminalSizeViaTput() (width, height int) {
	// Try tput cols
	if cmd := exec.Command("tput", "cols"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			if w, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && w > 0 {
				width = w
			}
		}
	}

	// Try tput lines
	if cmd := exec.Command("tput", "lines"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			if h, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && h > 0 {
				height = h
			}
		}
	}

	return width, height
}

// getTerminalSizeViaStty uses stty to get terminal size
func (c *DiagnosticViewComponent) getTerminalSizeViaStty() (width, height int) {
	cmd := exec.Command("stty", "size")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	// Parse stty size output: "speed 38400 baud; 80 columns; 24 lines;"
	parts := strings.Split(string(output), ";")
	for _, part := range parts {
		if strings.Contains(part, "columns") {
			fields := strings.Fields(part)
			for _, field := range fields {
				if w, err := strconv.Atoi(field); err == nil {
					width = w
					break
				}
			}
		}
		if strings.Contains(part, "lines") {
			fields := strings.Fields(part)
			for _, field := range fields {
				if h, err := strconv.Atoi(field); err == nil {
					height = h
					break
				}
			}
		}
	}

	return width, height
}

// getTerminalSizeFromEnv uses environment variables
func (c *DiagnosticViewComponent) getTerminalSizeFromEnv() (width, height int) {
	// Try LINES and COLUMNS environment variables
	if columns := os.Getenv("COLUMNS"); columns != "" {
		if w, err := strconv.Atoi(columns); err == nil {
			width = w
		}
	}
	if lines := os.Getenv("LINES"); lines != "" {
		if h, err := strconv.Atoi(lines); err == nil {
			height = h
		}
	}

	return width, height
}