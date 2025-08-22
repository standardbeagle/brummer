package components

import (
	"fmt"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LayoutComponent manages the overall layout and composition of components
type LayoutComponent struct {
	BaseComponent
	
	// Layout configuration
	mode           LayoutMode
	splitRatio     float32 // For split layouts (0.0 to 1.0)
	
	// Child components
	header         Component
	footer         Component
	sidebar        Component
	main           Component
	notifications  Component
	commandPalette Component
	
	// Layout state
	sidebarVisible bool
	footerVisible  bool
	headerVisible  bool
	
	// Styles
	borderStyle    lipgloss.Style
	headerStyle    lipgloss.Style
	footerStyle    lipgloss.Style
	sidebarStyle   lipgloss.Style
	mainStyle      lipgloss.Style
}

// LayoutMode represents different layout configurations
type LayoutMode int

const (
	LayoutModeFull LayoutMode = iota      // Full screen single component
	LayoutModeSidebar                      // Sidebar + Main
	LayoutModeHeaderFooter                 // Header + Main + Footer
	LayoutModeComplete                     // Header + Sidebar + Main + Footer
	LayoutModeSplit                        // Two components side by side
	LayoutModeGrid                         // Grid layout
)

// NewLayoutComponent creates a new layout component
func NewLayoutComponent() *LayoutComponent {
	l := &LayoutComponent{
		mode:           LayoutModeFull,
		splitRatio:     0.3,
		sidebarVisible: true,
		footerVisible:  true,
		headerVisible:  true,
	}
	
	// Initialize styles
	l.initStyles()
	
	// Set visible by default
	l.visible = true
	
	return l
}

// initStyles initializes the component styles
func (l *LayoutComponent) initStyles() {
	l.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241"))
	
	l.headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(0, 1)
	
	l.footerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)
	
	l.sidebarStyle = lipgloss.NewStyle().
		Padding(0, 1)
	
	l.mainStyle = lipgloss.NewStyle().
		Padding(0, 1)
}

// SetMode sets the layout mode
func (l *LayoutComponent) SetMode(mode LayoutMode) {
	l.mode = mode
}

// SetHeader sets the header component
func (l *LayoutComponent) SetHeader(c Component) {
	l.header = c
}

// SetFooter sets the footer component
func (l *LayoutComponent) SetFooter(c Component) {
	l.footer = c
}

// SetSidebar sets the sidebar component
func (l *LayoutComponent) SetSidebar(c Component) {
	l.sidebar = c
}

// SetMain sets the main component
func (l *LayoutComponent) SetMain(c Component) {
	l.main = c
}

// SetNotifications sets the notifications component
func (l *LayoutComponent) SetNotifications(c Component) {
	l.notifications = c
}

// SetCommandPalette sets the command palette component
func (l *LayoutComponent) SetCommandPalette(c Component) {
	l.commandPalette = c
}

// Update handles messages
func (l *LayoutComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle layout-specific messages
	switch msg := msg.(type) {
	case LayoutModeMsg:
		l.mode = msg.Mode
		return l, nil
		
	case LayoutToggleSidebarMsg:
		l.sidebarVisible = !l.sidebarVisible
		return l, nil
		
	case LayoutToggleFooterMsg:
		l.footerVisible = !l.footerVisible
		return l, nil
		
	case LayoutToggleHeaderMsg:
		l.headerVisible = !l.headerVisible
		return l, nil
		
	case LayoutSetSplitRatioMsg:
		l.splitRatio = msg.Ratio
		if l.splitRatio < 0.1 {
			l.splitRatio = 0.1
		}
		if l.splitRatio > 0.9 {
			l.splitRatio = 0.9
		}
		return l, nil
		
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
		
		// Propagate to child components with adjusted sizes
		l.updateChildSizes(msg)
		
	case tea.KeyMsg:
		// Handle layout-specific shortcuts
		switch msg.String() {
		case "ctrl+b":
			// Toggle sidebar
			l.sidebarVisible = !l.sidebarVisible
			return l, nil
			
		case "ctrl+f":
			// Toggle footer
			l.footerVisible = !l.footerVisible
			return l, nil
			
		case "ctrl+h":
			// Toggle header
			l.headerVisible = !l.headerVisible
			return l, nil
			
		case "ctrl+l":
			// Cycle layout modes
			l.mode = LayoutMode((int(l.mode) + 1) % 6)
			return l, nil
		}
	}
	
	// Update child components based on layout mode
	switch l.mode {
	case LayoutModeFull:
		if l.main != nil {
			newMain, cmd := l.main.Update(msg)
			l.main = newMain
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		
	case LayoutModeSidebar:
		if l.sidebar != nil && l.sidebarVisible {
			newSidebar, cmd := l.sidebar.Update(msg)
			l.sidebar = newSidebar
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if l.main != nil {
			newMain, cmd := l.main.Update(msg)
			l.main = newMain
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		
	case LayoutModeHeaderFooter:
		if l.header != nil && l.headerVisible {
			newHeader, cmd := l.header.Update(msg)
			l.header = newHeader
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if l.main != nil {
			newMain, cmd := l.main.Update(msg)
			l.main = newMain
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if l.footer != nil && l.footerVisible {
			newFooter, cmd := l.footer.Update(msg)
			l.footer = newFooter
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		
	case LayoutModeComplete:
		// Update all visible components
		if l.header != nil && l.headerVisible {
			newHeader, cmd := l.header.Update(msg)
			l.header = newHeader
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if l.sidebar != nil && l.sidebarVisible {
			newSidebar, cmd := l.sidebar.Update(msg)
			l.sidebar = newSidebar
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if l.main != nil {
			newMain, cmd := l.main.Update(msg)
			l.main = newMain
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if l.footer != nil && l.footerVisible {
			newFooter, cmd := l.footer.Update(msg)
			l.footer = newFooter
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	// Always update notifications and command palette if visible
	if l.notifications != nil {
		if visible, ok := l.notifications.(Visible); !ok || visible.IsVisible() {
			newNotif, cmd := l.notifications.Update(msg)
			l.notifications = newNotif
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	if l.commandPalette != nil {
		if visible, ok := l.commandPalette.(Visible); !ok || visible.IsVisible() {
			newCmd, cmd := l.commandPalette.Update(msg)
			l.commandPalette = newCmd
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	return l, tea.Batch(cmds...)
}

// View renders the component
func (l *LayoutComponent) View() string {
	if !l.visible {
		return ""
	}
	
	switch l.mode {
	case LayoutModeFull:
		return l.renderFullLayout()
	case LayoutModeSidebar:
		return l.renderSidebarLayout()
	case LayoutModeHeaderFooter:
		return l.renderHeaderFooterLayout()
	case LayoutModeComplete:
		return l.renderCompleteLayout()
	case LayoutModeSplit:
		return l.renderSplitLayout()
	case LayoutModeGrid:
		return l.renderGridLayout()
	default:
		return l.renderFullLayout()
	}
}

// renderFullLayout renders a single component full screen
func (l *LayoutComponent) renderFullLayout() string {
	if l.main == nil {
		return ""
	}
	
	// Main content
	content := l.main.View()
	
	// Overlay notifications
	if l.notifications != nil {
		if visible, ok := l.notifications.(Visible); !ok || visible.IsVisible() {
			content = l.overlayNotifications(content)
		}
	}
	
	// Overlay command palette
	if l.commandPalette != nil {
		if visible, ok := l.commandPalette.(Visible); !ok || visible.IsVisible() {
			content = l.overlayCommandPalette(content)
		}
	}
	
	return content
}

// renderSidebarLayout renders sidebar + main layout
func (l *LayoutComponent) renderSidebarLayout() string {
	if l.main == nil {
		return l.renderFullLayout()
	}
	
	var content string
	
	if l.sidebar != nil && l.sidebarVisible {
		// Calculate widths
		sidebarWidth := int(float32(l.width) * l.splitRatio)
		// mainWidth := l.width - sidebarWidth - 1 // -1 for separator (unused but kept for reference)
		
		// Render side by side
		sidebarView := l.sidebar.View()
		mainView := l.main.View()
		
		sidebarLines := strings.Split(sidebarView, "\n")
		mainLines := strings.Split(mainView, "\n")
		
		maxLines := max(len(sidebarLines), len(mainLines))
		
		var b strings.Builder
		for i := 0; i < maxLines; i++ {
			// Sidebar
			if i < len(sidebarLines) {
				line := sidebarLines[i]
				if len(line) > sidebarWidth {
					line = line[:sidebarWidth]
				}
				b.WriteString(fmt.Sprintf("%-*s", sidebarWidth, line))
			} else {
				b.WriteString(strings.Repeat(" ", sidebarWidth))
			}
			
			// Separator
			b.WriteString("│")
			
			// Main
			if i < len(mainLines) {
				b.WriteString(mainLines[i])
			}
			
			if i < maxLines-1 {
				b.WriteString("\n")
			}
		}
		
		content = b.String()
	} else {
		content = l.main.View()
	}
	
	// Overlay components
	content = l.overlayComponents(content)
	
	return content
}

// renderHeaderFooterLayout renders header + main + footer layout
func (l *LayoutComponent) renderHeaderFooterLayout() string {
	var parts []string
	
	// Header
	if l.header != nil && l.headerVisible {
		headerView := l.header.View()
		if headerView != "" {
			parts = append(parts, headerView)
			parts = append(parts, strings.Repeat("─", l.width))
		}
	}
	
	// Main
	if l.main != nil {
		parts = append(parts, l.main.View())
	}
	
	// Footer
	if l.footer != nil && l.footerVisible {
		footerView := l.footer.View()
		if footerView != "" {
			parts = append(parts, strings.Repeat("─", l.width))
			parts = append(parts, footerView)
		}
	}
	
	content := strings.Join(parts, "\n")
	
	// Overlay components
	content = l.overlayComponents(content)
	
	return content
}

// renderCompleteLayout renders the complete layout with all components
func (l *LayoutComponent) renderCompleteLayout() string {
	var b strings.Builder
	
	// Header
	headerHeight := 0
	if l.header != nil && l.headerVisible {
		headerView := l.header.View()
		if headerView != "" {
			b.WriteString(headerView)
			b.WriteString("\n")
			b.WriteString(strings.Repeat("─", l.width))
			b.WriteString("\n")
			headerHeight = strings.Count(headerView, "\n") + 2
		}
	}
	
	// Calculate remaining height
	footerHeight := 0
	if l.footer != nil && l.footerVisible {
		footerView := l.footer.View()
		if footerView != "" {
			footerHeight = strings.Count(footerView, "\n") + 2
		}
	}
	
	middleHeight := l.height - headerHeight - footerHeight
	
	// Middle section (sidebar + main)
	if middleHeight > 0 {
		if l.sidebar != nil && l.sidebarVisible {
			// With sidebar
			sidebarWidth := int(float32(l.width) * l.splitRatio)
			// mainWidth := l.width - sidebarWidth - 1 (unused)
			
			sidebarView := ""
			if l.sidebar != nil {
				sidebarView = l.sidebar.View()
			}
			
			mainView := ""
			if l.main != nil {
				mainView = l.main.View()
			}
			
			sidebarLines := strings.Split(sidebarView, "\n")
			mainLines := strings.Split(mainView, "\n")
			
			for i := 0; i < middleHeight; i++ {
				// Sidebar
				if i < len(sidebarLines) {
					line := sidebarLines[i]
					if len(line) > sidebarWidth {
						line = line[:sidebarWidth]
					}
					b.WriteString(fmt.Sprintf("%-*s", sidebarWidth, line))
				} else {
					b.WriteString(strings.Repeat(" ", sidebarWidth))
				}
				
				// Separator
				b.WriteString("│")
				
				// Main
				if i < len(mainLines) {
					b.WriteString(mainLines[i])
				}
				
				b.WriteString("\n")
			}
		} else {
			// No sidebar
			if l.main != nil {
				b.WriteString(l.main.View())
				b.WriteString("\n")
			}
		}
	}
	
	// Footer
	if l.footer != nil && l.footerVisible {
		footerView := l.footer.View()
		if footerView != "" {
			b.WriteString(strings.Repeat("─", l.width))
			b.WriteString("\n")
			b.WriteString(footerView)
		}
	}
	
	content := b.String()
	
	// Overlay components
	content = l.overlayComponents(content)
	
	return content
}

// renderSplitLayout renders two components side by side
func (l *LayoutComponent) renderSplitLayout() string {
	// Similar to sidebar layout but with equal priority
	return l.renderSidebarLayout()
}

// renderGridLayout renders components in a grid
func (l *LayoutComponent) renderGridLayout() string {
	// For now, default to complete layout
	// Grid implementation would divide screen into cells
	return l.renderCompleteLayout()
}

// Helper methods

func (l *LayoutComponent) overlayComponents(base string) string {
	content := base
	
	// Overlay notifications
	if l.notifications != nil {
		if visible, ok := l.notifications.(Visible); !ok || visible.IsVisible() {
			content = l.overlayNotifications(content)
		}
	}
	
	// Overlay command palette
	if l.commandPalette != nil {
		if visible, ok := l.commandPalette.(Visible); !ok || visible.IsVisible() {
			content = l.overlayCommandPalette(content)
		}
	}
	
	return content
}

func (l *LayoutComponent) overlayNotifications(base string) string {
	notifView := l.notifications.View()
	if notifView == "" {
		return base
	}
	
	// Overlay notifications at the top-right
	baseLines := strings.Split(base, "\n")
	notifLines := strings.Split(notifView, "\n")
	
	for i := 0; i < len(notifLines) && i < len(baseLines); i++ {
		// Calculate position for right alignment
		notifWidth := len(notifLines[i])
		if notifWidth > 0 && l.width > notifWidth {
			// Overlay on the right side
			baseLine := baseLines[i]
			if len(baseLine) < l.width {
				baseLine += strings.Repeat(" ", l.width-len(baseLine))
			}
			
			startPos := l.width - notifWidth
			if startPos > 0 && startPos < len(baseLine) {
				baseLines[i] = baseLine[:startPos] + notifLines[i]
			}
		}
	}
	
	return strings.Join(baseLines, "\n")
}

func (l *LayoutComponent) overlayCommandPalette(base string) string {
	cmdView := l.commandPalette.View()
	if cmdView == "" {
		return base
	}
	
	// Overlay command palette at the bottom
	baseLines := strings.Split(base, "\n")
	cmdLines := strings.Split(cmdView, "\n")
	
	// Calculate start position for overlay
	startLine := len(baseLines) - len(cmdLines) - 2
	if startLine < 0 {
		startLine = 0
	}
	
	for i, cmdLine := range cmdLines {
		lineIdx := startLine + i
		if lineIdx < len(baseLines) {
			baseLines[lineIdx] = cmdLine
		}
	}
	
	return strings.Join(baseLines, "\n")
}

func (l *LayoutComponent) updateChildSizes(msg tea.WindowSizeMsg) {
	// Calculate sizes based on layout mode
	switch l.mode {
	case LayoutModeFull:
		if l.main != nil {
			l.main.Update(msg)
		}
		
	case LayoutModeSidebar:
		sidebarWidth := int(float32(msg.Width) * l.splitRatio)
		mainWidth := msg.Width - sidebarWidth - 1
		
		if l.sidebar != nil {
			l.sidebar.Update(tea.WindowSizeMsg{
				Width:  sidebarWidth,
				Height: msg.Height,
			})
		}
		if l.main != nil {
			l.main.Update(tea.WindowSizeMsg{
				Width:  mainWidth,
				Height: msg.Height,
			})
		}
		
	case LayoutModeHeaderFooter, LayoutModeComplete:
		headerHeight := 3
		footerHeight := 2
		mainHeight := msg.Height - headerHeight - footerHeight
		
		if l.header != nil {
			l.header.Update(tea.WindowSizeMsg{
				Width:  msg.Width,
				Height: headerHeight,
			})
		}
		if l.main != nil {
			l.main.Update(tea.WindowSizeMsg{
				Width:  msg.Width,
				Height: mainHeight,
			})
		}
		if l.footer != nil {
			l.footer.Update(tea.WindowSizeMsg{
				Width:  msg.Width,
				Height: footerHeight,
			})
		}
	}
}

// Focus sets focus to this component
func (l *LayoutComponent) Focus() tea.Cmd {
	l.focused = true
	// Focus the main component by default
	if l.main != nil {
		return l.main.Focus()
	}
	return nil
}

// Blur removes focus from this component
func (l *LayoutComponent) Blur() tea.Cmd {
	l.focused = false
	// Blur all child components
	var cmds []tea.Cmd
	if l.main != nil {
		if cmd := l.main.Blur(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if l.sidebar != nil {
		if cmd := l.sidebar.Blur(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if l.header != nil {
		if cmd := l.header.Blur(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if l.footer != nil {
		if cmd := l.footer.Blur(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Layout-specific messages

type LayoutModeMsg struct {
	Mode LayoutMode
}

type LayoutToggleSidebarMsg struct{}

type LayoutToggleFooterMsg struct{}

type LayoutToggleHeaderMsg struct{}

type LayoutSetSplitRatioMsg struct {
	Ratio float32
}