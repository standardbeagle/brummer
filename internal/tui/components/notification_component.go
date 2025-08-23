package components

import (
	"fmt"
	"strings"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/pkg/events"
)

// NotificationComponent displays system notifications
type NotificationComponent struct {
	BaseComponent
	
	// Notifications queue
	notifications []SystemNotificationMsg
	maxNotifications int
	
	// Display settings
	position      NotificationPosition
	showDuration  time.Duration
	fadeOut       bool
	
	// Styles
	successStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	warningStyle  lipgloss.Style
	infoStyle     lipgloss.Style
	borderStyle   lipgloss.Style
	titleStyle    lipgloss.Style
	messageStyle  lipgloss.Style
	actionStyle   lipgloss.Style
	
	// EventBus integration (simplified for now)
	eventBus     *events.EventBus
}

// NotificationPosition defines where notifications appear
type NotificationPosition int

const (
	NotificationTopRight NotificationPosition = iota
	NotificationTopLeft
	NotificationBottomRight
	NotificationBottomLeft
	NotificationTop
	NotificationBottom
)

// NewNotificationComponent creates a new notification component
func NewNotificationComponent(eventBus *events.EventBus) *NotificationComponent {
	n := &NotificationComponent{
		notifications:    []SystemNotificationMsg{},
		maxNotifications: 5,
		position:        NotificationTopRight,
		showDuration:    5 * time.Second,
		fadeOut:         true,
		eventBus:        eventBus,
	}
	
	// Initialize styles
	n.initStyles()
	
	// Subscribe to EventBus notifications if available
	if eventBus != nil {
		n.subscribeToEventBus()
	}
	
	// Set visible by default
	n.visible = true
	
	return n
}

// initStyles initializes the component styles
func (n *NotificationComponent) initStyles() {
	// Base border style
	n.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		MarginBottom(1)
	
	// Level-specific styles
	n.successStyle = n.borderStyle.Copy().
		BorderForeground(lipgloss.Color("42")).
		Foreground(lipgloss.Color("42"))
	
	n.errorStyle = n.borderStyle.Copy().
		BorderForeground(lipgloss.Color("196")).
		Foreground(lipgloss.Color("196"))
	
	n.warningStyle = n.borderStyle.Copy().
		BorderForeground(lipgloss.Color("214")).
		Foreground(lipgloss.Color("214"))
	
	n.infoStyle = n.borderStyle.Copy().
		BorderForeground(lipgloss.Color("86")).
		Foreground(lipgloss.Color("86"))
	
	// Content styles
	n.titleStyle = lipgloss.NewStyle().
		Bold(true).
		MarginBottom(1)
	
	n.messageStyle = lipgloss.NewStyle()
	
	n.actionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Underline(true).
		MarginTop(1)
}

// subscribeToEventBus subscribes to EventBus notifications
func (n *NotificationComponent) subscribeToEventBus() {
	// Simplified for now - EventBus integration can be added later
	// when the EventBus supports system notifications
}

// mapEventLevel maps EventBus notification level to component level
// This is a placeholder for future EventBus integration
func (n *NotificationComponent) mapEventLevel(level string) NotificationLevel {
	switch level {
	case "error":
		return NotificationError
	case "warning":
		return NotificationWarning
	case "success":
		return NotificationSuccess
	default:
		return NotificationInfo
	}
}

// Update handles messages
func (n *NotificationComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case SystemNotificationMsg:
		// Add new notification
		n.addNotification(msg)
		
		// Schedule auto-dismiss if needed
		if msg.AutoDismiss {
			duration := msg.Duration
			if duration == 0 {
				duration = n.showDuration
			}
			cmds = append(cmds, n.scheduleDismiss(msg.ID, duration))
		}
		return n, tea.Batch(cmds...)
		
	case NotificationMsg:
		// Handle old-style notification (convert to SystemNotificationMsg)
		n.addNotification(SystemNotificationMsg{
			ID:          generateID(),
			Level:       msg.Level,
			Title:       msg.Title,
			Message:     msg.Message,
			Timestamp:   time.Now(),
			Actions:     msg.Actions,
			AutoDismiss: msg.Level != NotificationError,
			Duration:    n.showDuration,
		})
		
		if msg.Level != NotificationError {
			cmds = append(cmds, n.scheduleDismiss(n.notifications[len(n.notifications)-1].ID, n.showDuration))
		}
		return n, tea.Batch(cmds...)
		
	case NotificationDismissMsg:
		// Dismiss notification(s)
		if msg.All {
			n.notifications = []SystemNotificationMsg{}
		} else {
			n.removeNotification(msg.ID)
		}
		return n, nil
		
	case NotificationActionMsg:
		// Execute notification action
		for _, notif := range n.notifications {
			if notif.ID == msg.NotificationID && msg.ActionIndex < len(notif.Actions) {
				cmd := notif.Actions[msg.ActionIndex].Command
				// Remove notification after action
				n.removeNotification(notif.ID)
				return n, cmd
			}
		}
		return n, nil
		
	case dismissTimerMsg:
		// Auto-dismiss notification
		n.removeNotification(msg.notificationID)
		return n, nil
		
	case tea.KeyMsg:
		// Handle key presses for visible notifications
		if len(n.notifications) > 0 {
			switch msg.String() {
			case "esc":
				// Dismiss top notification
				if len(n.notifications) > 0 {
					n.removeNotification(n.notifications[0].ID)
				}
				return n, nil
				
			case "ctrl+n":
				// Dismiss all notifications
				n.notifications = []SystemNotificationMsg{}
				return n, nil
			}
			
			// Check for action keys (1-9)
			if len(n.notifications) > 0 && len(n.notifications[0].Actions) > 0 {
				for i, action := range n.notifications[0].Actions {
					if msg.String() == fmt.Sprintf("%d", i+1) {
						cmd := action.Command
						n.removeNotification(n.notifications[0].ID)
						return n, cmd
					}
				}
			}
		}
	}
	
	return n, tea.Batch(cmds...)
}

// View renders the component
func (n *NotificationComponent) View() string {
	if !n.visible || len(n.notifications) == 0 {
		return ""
	}
	
	var notifications []string
	
	// Render each notification
	count := len(n.notifications)
	if count > n.maxNotifications {
		count = n.maxNotifications
	}
	
	for i := 0; i < count; i++ {
		notif := n.notifications[i]
		notifications = append(notifications, n.renderNotification(notif, i == 0))
	}
	
	// Position notifications
	content := strings.Join(notifications, "\n")
	
	// Apply positioning
	switch n.position {
	case NotificationTopRight:
		// Align to top-right
		lines := strings.Split(content, "\n")
		maxWidth := 0
		for _, line := range lines {
			width := lipgloss.Width(line)
			if width > maxWidth {
				maxWidth = width
			}
		}
		
		aligned := []string{}
		for _, line := range lines {
			width := lipgloss.Width(line)
			padding := strings.Repeat(" ", n.width-width-2)
			aligned = append(aligned, padding+line)
		}
		content = strings.Join(aligned, "\n")
		
	case NotificationTopLeft:
		// Already left-aligned by default
		
	case NotificationBottomRight:
		// Would need terminal height to position at bottom
		// For now, align right
		lines := strings.Split(content, "\n")
		maxWidth := 0
		for _, line := range lines {
			width := lipgloss.Width(line)
			if width > maxWidth {
				maxWidth = width
			}
		}
		
		aligned := []string{}
		for _, line := range lines {
			width := lipgloss.Width(line)
			padding := strings.Repeat(" ", n.width-width-2)
			aligned = append(aligned, padding+line)
		}
		content = strings.Join(aligned, "\n")
		
	case NotificationTop:
		// Center horizontally
		lines := strings.Split(content, "\n")
		centered := []string{}
		for _, line := range lines {
			width := lipgloss.Width(line)
			leftPadding := (n.width - width) / 2
			if leftPadding > 0 {
				centered = append(centered, strings.Repeat(" ", leftPadding)+line)
			} else {
				centered = append(centered, line)
			}
		}
		content = strings.Join(centered, "\n")
	}
	
	return content
}

// renderNotification renders a single notification
func (n *NotificationComponent) renderNotification(notif SystemNotificationMsg, isTop bool) string {
	// Select style based on level
	var style lipgloss.Style
	var icon string
	
	switch notif.Level {
	case NotificationSuccess:
		style = n.successStyle
		icon = "✓"
	case NotificationError:
		style = n.errorStyle
		icon = "✗"
	case NotificationWarning:
		style = n.warningStyle
		icon = "⚠"
	default:
		style = n.infoStyle
		icon = "ℹ"
	}
	
	// Build content
	var content strings.Builder
	
	// Title with icon
	if notif.Title != "" {
		title := fmt.Sprintf("%s %s", icon, notif.Title)
		content.WriteString(n.titleStyle.Render(title))
		if notif.Message != "" {
			content.WriteString("\n")
		}
	}
	
	// Message
	if notif.Message != "" {
		// Wrap message if too long
		wrapped := n.wrapText(notif.Message, 50)
		content.WriteString(n.messageStyle.Render(wrapped))
	}
	
	// Details (if expanded and is top notification)
	if isTop && notif.Details != "" {
		content.WriteString("\n\n")
		content.WriteString(n.messageStyle.Render(n.wrapText(notif.Details, 50)))
	}
	
	// Actions (only for top notification)
	if isTop && len(notif.Actions) > 0 {
		content.WriteString("\n")
		actions := []string{}
		for i, action := range notif.Actions {
			actionText := fmt.Sprintf("[%d] %s", i+1, action.Label)
			actions = append(actions, n.actionStyle.Render(actionText))
		}
		content.WriteString(strings.Join(actions, "  "))
	}
	
	// Help text for top notification
	if isTop {
		help := "[esc] dismiss"
		if len(n.notifications) > 1 {
			help += fmt.Sprintf(" | %d more", len(n.notifications)-1)
		}
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(help))
	}
	
	// Apply border style
	return style.Render(content.String())
}

// addNotification adds a notification to the queue
func (n *NotificationComponent) addNotification(notif SystemNotificationMsg) {
	// Check for duplicate
	for i, existing := range n.notifications {
		if existing.Title == notif.Title && existing.Message == notif.Message {
			// Update timestamp and move to front
			notif.Count = existing.Count + 1
			n.notifications = append([]SystemNotificationMsg{notif}, 
				append(n.notifications[:i], n.notifications[i+1:]...)...)
			return
		}
	}
	
	// Add to front of queue
	n.notifications = append([]SystemNotificationMsg{notif}, n.notifications...)
	
	// Limit queue size
	if len(n.notifications) > n.maxNotifications*2 {
		n.notifications = n.notifications[:n.maxNotifications*2]
	}
}

// removeNotification removes a notification by ID
func (n *NotificationComponent) removeNotification(id string) {
	filtered := []SystemNotificationMsg{}
	for _, notif := range n.notifications {
		if notif.ID != id {
			filtered = append(filtered, notif)
		}
	}
	n.notifications = filtered
}

// scheduleDismiss schedules auto-dismissal of a notification
func (n *NotificationComponent) scheduleDismiss(id string, duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return dismissTimerMsg{notificationID: id}
	})
}

// wrapText wraps text to fit within maxWidth
func (n *NotificationComponent) wrapText(text string, maxWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	
	var lines []string
	currentLine := words[0]
	
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= maxWidth {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return strings.Join(lines, "\n")
}

// SetPosition sets the notification position
func (n *NotificationComponent) SetPosition(pos NotificationPosition) {
	n.position = pos
}

// SetMaxNotifications sets the maximum number of visible notifications
func (n *NotificationComponent) SetMaxNotifications(max int) {
	n.maxNotifications = max
}

// SetShowDuration sets the auto-dismiss duration
func (n *NotificationComponent) SetShowDuration(duration time.Duration) {
	n.showDuration = duration
}

// Focus sets focus to this component
func (n *NotificationComponent) Focus() tea.Cmd {
	n.focused = true
	return nil
}

// Blur removes focus from this component
func (n *NotificationComponent) Blur() tea.Cmd {
	n.focused = false
	return nil
}

// Cleanup unsubscribes from EventBus
func (n *NotificationComponent) Cleanup() {
	// Simplified for now - no subscription to clean up
}

// dismissTimerMsg is sent when a notification should be auto-dismissed
type dismissTimerMsg struct {
	notificationID string
}