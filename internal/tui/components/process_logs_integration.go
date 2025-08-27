// Package components provides a shell that integrates process and logs views
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

// TabInfo represents a tab in the shell
type TabInfo struct {
	Icon      string
	Name      string
	ShortName string
}

// ProcessLogsShell integrates process and logs views with proper communication
type ProcessLogsShell struct {
	// Components
	processView *ProcessViewNew
	logsView    *LogsViewComponent

	// State
	activeTab int
	tabs      []TabInfo
	width     int
	height    int

	// Dependencies
	processMgr *process.Manager
	logStore   *logs.Store
	logSub     *LogSubscription

	// Styles
	tabBarStyle    lipgloss.Style
	activeTabStyle lipgloss.Style
	tabStyle       lipgloss.Style
	contentStyle   lipgloss.Style
}

// NewProcessLogsShell creates a shell with integrated process and logs views
func NewProcessLogsShell(processMgr *process.Manager, logStore *logs.Store) *ProcessLogsShell {
	// Initialize colors
	MustInitColors()

	// Create components
	processView := NewProcessViewNew(processMgr)
	logsView := NewLogsViewComponent(logStore)
	logSub := NewLogSubscription(logStore)

	shell := &ProcessLogsShell{
		processView: processView,
		logsView:    logsView,
		processMgr:  processMgr,
		logStore:    logStore,
		logSub:      logSub,
		activeTab:   0,
		tabs: []TabInfo{
			{Icon: "🏃", Name: "Processes", ShortName: "Proc"},
			{Icon: "📄", Name: "Logs", ShortName: "Logs"},
		},
	}

	shell.initStyles()

	// Set initial sizes
	shell.SetSize(80, 24)

	// Register log callback to capture process output
	if processMgr != nil {
		processMgr.AddLogCallback(func(processID, line string, isError bool) {
			// Add to log store
			if logStore != nil {
				// Get process name
				processName := "Unknown"
				for _, proc := range processMgr.GetAllProcesses() {
					if proc.ID == processID {
						processName = proc.Name
						break
					}
				}
				logStore.Add(processID, processName, line, isError)
			}
		})
	}

	return shell
}

func (s *ProcessLogsShell) initStyles() {
	s.tabBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	s.activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("236")).
		Padding(0, 2)

	s.tabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 2)

	s.contentStyle = lipgloss.NewStyle()
}

// Init initializes the shell
func (s *ProcessLogsShell) Init() tea.Cmd {
	return tea.Batch(
		s.processView.Init(),
		s.logsView.Init(),
		s.logSub.Subscribe(), // Use subscription instead of polling
	)
}

// Update handles messages
func (s *ProcessLogsShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Debug mode validation
	Debug().CheckState("ProcessLogsShell", func() error {
		return ValidateShell(s)
	})

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.SetSize(msg.Width, msg.Height)

		// Update components
		s.processView.Update(msg)
		s.logsView.Update(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Switch to next tab
			s.activeTab = (s.activeTab + 1) % len(s.tabs)
			s.updateFocus()

		case "shift+tab":
			// Switch to previous tab
			s.activeTab = s.activeTab - 1
			if s.activeTab < 0 {
				s.activeTab = len(s.tabs) - 1
			}
			s.updateFocus()

		case "ctrl+c", "q":
			return s, tea.Quit

		default:
			// Forward to active component
			if s.activeTab == 0 {
				m, cmd := s.processView.Update(msg)
				s.processView = m.(*ProcessViewNew)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else if s.activeTab == 1 {
				m, cmd := s.logsView.Update(msg)
				s.logsView = m.(*LogsViewComponent)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case SwitchToLogsMsg:
		// Switch to logs view and filter by process
		s.activeTab = 1
		s.updateFocus()

		// Apply process filter to logs
		cmds = append(cmds, func() tea.Msg {
			return LogsFilterMsg{
				ProcessID: msg.ProcessID,
			}
		})

		// Update logs view with the filter
		m, cmd := s.logsView.Update(LogsFilterMsg{ProcessID: msg.ProcessID})
		s.logsView = m.(*LogsViewComponent)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case LogsUpdateMsg:
		// Handle incremental log updates from subscription
		m, cmd := s.logsView.Update(msg)
		s.logsView = m.(*LogsViewComponent)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Continue subscription
		cmds = append(cmds, s.logSub.Subscribe())

	default:
		// Update both components with other messages
		m1, cmd1 := s.processView.Update(msg)
		s.processView = m1.(*ProcessViewNew)
		if cmd1 != nil {
			cmds = append(cmds, cmd1)
		}

		m2, cmd2 := s.logsView.Update(msg)
		s.logsView = m2.(*LogsViewComponent)
		if cmd2 != nil {
			cmds = append(cmds, cmd2)
		}
	}

	return s, tea.Batch(cmds...)
}

// View renders the shell
func (s *ProcessLogsShell) View() string {
	return Debug().TrackRender("ProcessLogsShell", func() string {
		var b strings.Builder

		// Render tab bar
		b.WriteString(s.renderTabBar())
		b.WriteString("\n")

		// Calculate content area
		tabBarHeight := 1
		debugOverlayHeight := 0

		// Reserve space for debug overlay if enabled
		if Debug().Enabled() {
			debugOverlayHeight = 15 // Approximate overlay height
		}

		contentHeight := s.height - tabBarHeight - debugOverlayHeight
		if contentHeight < 5 {
			contentHeight = 5 // Minimum usable height
		}

		// Render active component
		var content string
		switch s.activeTab {
		case 0:
			s.processView.SetSize(s.width, contentHeight)
			content = s.processView.View()
		case 1:
			s.logsView.SetSize(s.width, contentHeight)
			content = s.logsView.View()
		}

		// Ensure content fills the width
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if lipgloss.Width(line) < s.width {
				lines[i] = line + strings.Repeat(" ", s.width-lipgloss.Width(line))
			}
		}
		content = strings.Join(lines, "\n")

		b.WriteString(content)

		// Add debug overlay if enabled
		if Debug().Enabled() {
			// Position overlay in top-right corner
			debugOverlay := Debug().RenderOverlay(s.width, s.height)
			if debugOverlay != "" {
				// Split content into lines for overlay positioning
				contentLines := strings.Split(b.String(), "\n")
				overlayLines := strings.Split(debugOverlay, "\n")

				// Position overlay in top-right
				for i, overlayLine := range overlayLines {
					if i < len(contentLines) && len(overlayLines) > 0 {
						// Calculate position to right-align overlay
						overlayWidth := lipgloss.Width(overlayLine)
						if s.width > overlayWidth {
							contentLineWidth := lipgloss.Width(contentLines[i])
							availableWidth := s.width - overlayWidth

							if contentLineWidth < availableWidth {
								// Overlay on empty space
								padding := strings.Repeat(" ", availableWidth-contentLineWidth)
								contentLines[i] = contentLines[i] + padding + overlayLine
							} else {
								// Overlay over content (debug priority)
								truncated := contentLines[i]
								if len(truncated) > availableWidth {
									truncated = truncated[:availableWidth]
								}
								contentLines[i] = truncated + overlayLine
							}
						}
					}
				}

				// Add remaining overlay lines
				for i := len(contentLines); i < len(overlayLines); i++ {
					overlayLine := overlayLines[i]
					overlayWidth := lipgloss.Width(overlayLine)
					if s.width > overlayWidth {
						padding := strings.Repeat(" ", s.width-overlayWidth)
						contentLines = append(contentLines, padding+overlayLine)
					} else {
						contentLines = append(contentLines, overlayLine)
					}
				}

				return strings.Join(contentLines, "\n")
			}
		}

		return b.String()
	})
}

func (s *ProcessLogsShell) renderTabBar() string {
	var tabs []string

	for i, tab := range s.tabs {
		// Get notification counts for the tab
		var notifications string
		if i == 0 {
			// Process tab notifications
			errors, warnings, info := s.processView.GetNotificationCount()
			if errors > 0 {
				notifications = fmt.Sprintf(" 🔴%d", errors)
			} else if warnings > 0 {
				notifications = fmt.Sprintf(" 🟡%d", warnings)
			} else if info > 0 {
				notifications = fmt.Sprintf(" 🔵%d", info)
			}
		}

		// Format tab
		tabText := fmt.Sprintf("%s %s%s", tab.Icon, tab.Name, notifications)

		if i == s.activeTab {
			tabs = append(tabs, s.activeTabStyle.Render(tabText))
		} else {
			tabs = append(tabs, s.tabStyle.Render(tabText))
		}
	}

	tabBar := strings.Join(tabs, " ")

	// Add help text
	help := "Tab/Shift+Tab: Switch tabs | q: Quit"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	// Calculate spacing
	tabBarWidth := lipgloss.Width(tabBar)
	helpWidth := lipgloss.Width(help)
	spacing := s.width - tabBarWidth - helpWidth - 4
	if spacing < 0 {
		spacing = 0
	}

	return s.tabBarStyle.Width(s.width).Render(
		tabBar + strings.Repeat(" ", spacing) + helpStyle.Render(help),
	)
}

func (s *ProcessLogsShell) updateFocus() {
	// Blur all components
	s.processView.Blur()
	s.logsView.Blur()

	// Focus active component
	switch s.activeTab {
	case 0:
		s.processView.Focus()
	case 1:
		s.logsView.Focus()
	}
}

// SetSize updates the shell size
func (s *ProcessLogsShell) SetSize(width, height int) {
	s.width = width
	s.height = height

	// Update component sizes
	contentHeight := height - 1 // Account for tab bar
	s.processView.SetSize(width, contentHeight)
	s.logsView.SetSize(width, contentHeight)
}

// Removed streamLogs and logStreamMsg - now using subscription pattern

// Example usage function
func ExampleProcessLogsIntegration() {
	// This would be called from main or test code
	// processMgr := process.NewManager(...)
	// logStore := logs.NewStore(...)
	//
	// shell := NewProcessLogsShell(processMgr, logStore)
	// p := tea.NewProgram(shell, tea.WithAltScreen())
	// if _, err := p.Run(); err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// }
}
