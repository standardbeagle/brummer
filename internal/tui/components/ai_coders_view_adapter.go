package components

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AICodersViewComponent provides AI coder session management
type AICodersViewComponent struct {
	BaseComponent

	// Sessions
	sessions        []AICoderSessionInfo
	activeSessionID string
	selectedIndex   int

	// Current session view
	mode        AICoderViewMode
	messages    []AIMessage
	messageView viewport.Model
	inputArea   textarea.Model
	sessionList list.Model

	// Display options
	showTimestamps bool
	showTokenCount bool
	splitView      bool

	// Styles
	titleStyle     lipgloss.Style
	sessionStyle   lipgloss.Style
	userStyle      lipgloss.Style
	assistantStyle lipgloss.Style
	systemStyle    lipgloss.Style
	timestampStyle lipgloss.Style
	activeStyle    lipgloss.Style
	inactiveStyle  lipgloss.Style
	errorStyle     lipgloss.Style
}

// AICoderViewMode represents the display mode
type AICoderViewMode int

const (
	AICoderViewModeList AICoderViewMode = iota
	AICoderViewModeChat
	AICoderViewModeSplit
)

// AIMessage represents a message in the AI conversation
type AIMessage struct {
	ID        string
	SessionID string
	Role      string // "user", "assistant", "system"
	Content   string
	Timestamp time.Time
	Tokens    int
	Error     string
}

// NewAICodersViewComponent creates a new AI coders view component
func NewAICodersViewComponent() *AICodersViewComponent {
	a := &AICodersViewComponent{
		sessions:       []AICoderSessionInfo{},
		messages:       []AIMessage{},
		mode:           AICoderViewModeList,
		showTimestamps: true,
		showTokenCount: false,
		splitView:      true,
	}

	// Initialize styles
	a.initStyles()

	// Initialize viewport for messages
	a.messageView = viewport.New(80, 20)

	// Initialize input area
	a.inputArea = textarea.New()
	a.inputArea.Placeholder = "Type your message to AI..."
	a.inputArea.CharLimit = 4000
	a.inputArea.SetWidth(80)
	a.inputArea.SetHeight(4)

	// Initialize session list
	a.initializeSessionList()

	// Set visible by default
	a.visible = true

	return a
}

// initStyles initializes the component styles
func (a *AICodersViewComponent) initStyles() {
	a.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	a.sessionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))

	a.userStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	a.assistantStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	a.systemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	a.timestampStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	a.activeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	a.inactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	a.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
}

// initializeSessionList sets up the session list
func (a *AICodersViewComponent) initializeSessionList() {
	items := a.sessionsToListItems()

	a.sessionList = list.New(items, sessionItemDelegate{}, 40, 20)
	a.sessionList.Title = "AI Coder Sessions"
	a.sessionList.SetShowStatusBar(true)
	a.sessionList.SetFilteringEnabled(true)

	// Customize key bindings
	a.sessionList.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	a.sessionList.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}

// Update handles messages
// Init implements tea.Model interface
func (a *AICodersViewComponent) Init() tea.Cmd {
	return nil
}

func (a *AICodersViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle component-specific messages
	switch msg := msg.(type) {
	case AICoderStartMsg:
		// Start new AI coder session
		session := AICoderSessionInfo{
			ID:           generateSessionID(),
			Name:         msg.Name,
			Model:        msg.Model,
			State:        "active",
			MessageCount: 0,
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		}
		a.sessions = append(a.sessions, session)
		a.activeSessionID = session.ID
		a.updateSessionList()

		// Add system message
		a.messages = append(a.messages, AIMessage{
			ID:        generateMessageID(),
			SessionID: session.ID,
			Role:      "system",
			Content:   msg.SystemPrompt,
			Timestamp: time.Now(),
		})

		// Switch to chat mode
		a.mode = AICoderViewModeChat
		a.updateMessageView()

		return a, SendSystemNotification(NotificationSuccess,
			"AI Coder started", fmt.Sprintf("Session: %s", msg.Name))

	case AICoderStopMsg:
		// Stop AI coder session
		for i, session := range a.sessions {
			if session.ID == msg.SessionID {
				a.sessions[i].State = "stopped"
				break
			}
		}
		a.updateSessionList()

		if msg.SessionID == a.activeSessionID {
			a.activeSessionID = ""
			a.mode = AICoderViewModeList
		}

		return a, SendSystemNotification(NotificationInfo,
			"AI Coder stopped", fmt.Sprintf("Session: %s", msg.SessionID))

	case AICoderMessageMsg:
		// Add message to conversation
		message := AIMessage{
			ID:        generateMessageID(),
			SessionID: msg.SessionID,
			Role:      "user",
			Content:   msg.Message,
			Timestamp: time.Now(),
		}
		if !msg.IsUser {
			message.Role = "assistant"
		}

		a.messages = append(a.messages, message)

		// Update session activity
		for i, session := range a.sessions {
			if session.ID == msg.SessionID {
				a.sessions[i].MessageCount++
				a.sessions[i].LastActivity = time.Now()
				break
			}
		}

		a.updateMessageView()
		a.updateSessionList()

		// Clear input if it was user message
		if msg.IsUser {
			a.inputArea.SetValue("")
		}

		return a, nil

	case AICoderResponseMsg:
		// Handle AI response
		if msg.Error != nil {
			// Add error message
			a.messages = append(a.messages, AIMessage{
				ID:        generateMessageID(),
				SessionID: msg.SessionID,
				Role:      "system",
				Content:   fmt.Sprintf("Error: %v", msg.Error),
				Timestamp: time.Now(),
				Error:     msg.Error.Error(),
			})

			return a, SendSystemNotification(NotificationError,
				"AI Error", msg.Error.Error())
		}

		// Add or update assistant message
		if msg.Complete {
			a.messages = append(a.messages, AIMessage{
				ID:        generateMessageID(),
				SessionID: msg.SessionID,
				Role:      "assistant",
				Content:   msg.Response,
				Timestamp: time.Now(),
			})
		} else {
			// Streaming response - update last message
			for i := len(a.messages) - 1; i >= 0; i-- {
				if a.messages[i].SessionID == msg.SessionID && a.messages[i].Role == "assistant" {
					a.messages[i].Content += msg.Response
					break
				}
			}
		}

		a.updateMessageView()
		return a, nil

	case AICoderSessionUpdateMsg:
		// Update session list
		a.sessions = msg.Sessions
		a.updateSessionList()
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Update component sizes
		if a.splitView {
			a.sessionList.SetSize(msg.Width/3, msg.Height-4)
			a.messageView.Width = msg.Width*2/3 - 2
			a.messageView.Height = msg.Height - 10
		} else {
			a.sessionList.SetSize(msg.Width-4, msg.Height-4)
			a.messageView.Width = msg.Width - 4
			a.messageView.Height = msg.Height - 10
		}

		a.inputArea.SetWidth(msg.Width - 4)

	case tea.KeyMsg:
		// Handle key presses based on mode
		switch a.mode {
		case AICoderViewModeList:
			switch msg.String() {
			case "enter", " ":
				// Select session
				if item := a.sessionList.SelectedItem(); item != nil {
					if si, ok := item.(sessionItem); ok {
						a.activeSessionID = si.session.ID
						a.mode = AICoderViewModeChat
						a.loadSessionMessages(si.session.ID)
						return a, nil
					}
				}

			case "n":
				// New session
				return a, func() tea.Msg {
					return AICoderStartMsg{
						Name:         fmt.Sprintf("Session %d", len(a.sessions)+1),
						Model:        "claude-3",
						SystemPrompt: "You are a helpful AI coding assistant.",
						WorkingDir:   ".",
					}
				}

			case "d":
				// Delete/stop session
				if item := a.sessionList.SelectedItem(); item != nil {
					if si, ok := item.(sessionItem); ok {
						return a, func() tea.Msg {
							return AICoderStopMsg{
								SessionID: si.session.ID,
								Save:      true,
							}
						}
					}
				}

			case "s":
				// Toggle split view
				a.splitView = !a.splitView
				return a, nil

			case "tab":
				// Switch to chat if active session
				if a.activeSessionID != "" {
					a.mode = AICoderViewModeChat
					return a, nil
				}
			}

		case AICoderViewModeChat:
			// Check if input is focused
			if a.inputArea.Focused() {
				switch msg.String() {
				case "ctrl+c":
					// Cancel input
					a.inputArea.SetValue("")
					a.inputArea.Blur()
					return a, nil

				case "ctrl+enter":
					// Send message
					message := a.inputArea.Value()
					if message != "" && a.activeSessionID != "" {
						// Send message
						cmd := func() tea.Msg {
							return AICoderMessageMsg{
								SessionID: a.activeSessionID,
								Message:   message,
								IsUser:    true,
							}
						}
						a.inputArea.SetValue("")
						return a, cmd
					}

				case "esc":
					// Unfocus input
					a.inputArea.Blur()
					return a, nil

				default:
					// Update input
					var cmd tea.Cmd
					a.inputArea, cmd = a.inputArea.Update(msg)
					return a, cmd
				}
			} else {
				// Chat navigation
				switch msg.String() {
				case "i":
					// Focus input
					a.inputArea.Focus()
					return a, textarea.Blink

				case "tab", "l":
					// Switch to list
					a.mode = AICoderViewModeList
					return a, nil

				case "s":
					// Toggle split view
					a.splitView = !a.splitView
					return a, nil

				case "t":
					// Toggle timestamps
					a.showTimestamps = !a.showTimestamps
					a.updateMessageView()
					return a, nil

				case "c":
					// Clear messages (keep system prompt)
					var newMessages []AIMessage
					for _, msg := range a.messages {
						if msg.SessionID == a.activeSessionID && msg.Role == "system" {
							newMessages = append(newMessages, msg)
						}
					}
					a.messages = newMessages
					a.updateMessageView()
					return a, nil

				case "q":
					// Quit session
					a.activeSessionID = ""
					a.mode = AICoderViewModeList
					return a, nil
				}
			}

		case AICoderViewModeSplit:
			// Split view mode - handle both list and chat
			switch msg.String() {
			case "tab":
				// Toggle focus between list and chat
				// Implementation would depend on which side is focused
				return a, nil
			}
		}
	}

	// Update sub-components
	if a.visible {
		switch a.mode {
		case AICoderViewModeList:
			newList, cmd := a.sessionList.Update(msg)
			a.sessionList = newList
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

		case AICoderViewModeChat:
			// Update viewport
			newView, cmd := a.messageView.Update(msg)
			a.messageView = newView
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Update input if not handling keys above
			if a.inputArea.Focused() {
				newInput, cmd := a.inputArea.Update(msg)
				a.inputArea = newInput
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return a, tea.Batch(cmds...)
}

// View renders the component
func (a *AICodersViewComponent) View() string {
	if !a.visible {
		return ""
	}

	var b strings.Builder

	switch a.mode {
	case AICoderViewModeList:
		a.renderListView(&b)
	case AICoderViewModeChat:
		a.renderChatView(&b)
	case AICoderViewModeSplit:
		a.renderSplitView(&b)
	}

	return b.String()
}

// renderListView renders the session list
func (a *AICodersViewComponent) renderListView(b *strings.Builder) {
	// Title
	title := fmt.Sprintf("🤖 AI Coder Sessions (%d)", len(a.sessions))
	b.WriteString(a.titleStyle.Render(title) + "\n\n")

	// Session list
	b.WriteString(a.sessionList.View())

	// Footer hints
	b.WriteString("\n")
	hints := []string{
		"enter: open session",
		"n: new session",
		"d: stop session",
		"s: split view",
		"tab: switch to chat",
	}
	b.WriteString(a.timestampStyle.Render(strings.Join(hints, " | ")))
}

// renderChatView renders the chat interface
func (a *AICodersViewComponent) renderChatView(b *strings.Builder) {
	// Find active session
	var activeSession *AICoderSessionInfo
	for _, session := range a.sessions {
		if session.ID == a.activeSessionID {
			activeSession = &session
			break
		}
	}

	// Title bar
	if activeSession != nil {
		title := fmt.Sprintf("🤖 %s (%s) - %s", activeSession.Name, activeSession.Model, activeSession.State)
		b.WriteString(a.titleStyle.Render(title) + "\n")
	} else {
		b.WriteString(a.titleStyle.Render("🤖 AI Coder") + "\n")
	}

	// Message view
	b.WriteString(a.messageView.View())
	b.WriteString("\n")

	// Input area
	if a.activeSessionID != "" && activeSession != nil && activeSession.State == "active" {
		b.WriteString("─────────────────\n")
		b.WriteString(a.inputArea.View())
		b.WriteString("\n")

		// Input hints
		if a.inputArea.Focused() {
			hints := []string{
				"ctrl+enter: send",
				"ctrl+c: cancel",
				"esc: unfocus",
			}
			b.WriteString(a.timestampStyle.Render(strings.Join(hints, " | ")))
		} else {
			hints := []string{
				"i: input",
				"tab: list",
				"t: timestamps",
				"c: clear",
				"q: quit",
			}
			b.WriteString(a.timestampStyle.Render(strings.Join(hints, " | ")))
		}
	}
}

// renderSplitView renders split list and chat
func (a *AICodersViewComponent) renderSplitView(b *strings.Builder) {
	// Implementation would show both list and chat side by side
	// For now, just show chat
	a.renderChatView(b)
}

// Helper methods

func (a *AICodersViewComponent) updateSessionList() {
	items := a.sessionsToListItems()
	a.sessionList.SetItems(items)
	a.sessionList.Title = fmt.Sprintf("AI Coder Sessions (%d)", len(a.sessions))
}

func (a *AICodersViewComponent) sessionsToListItems() []list.Item {
	items := make([]list.Item, len(a.sessions))
	for i, session := range a.sessions {
		items[i] = sessionItem{session: session}
	}
	return items
}

func (a *AICodersViewComponent) loadSessionMessages(sessionID string) {
	// Filter messages for this session
	a.updateMessageView()
}

func (a *AICodersViewComponent) updateMessageView() {
	var content strings.Builder

	// Filter messages for active session
	for _, msg := range a.messages {
		if msg.SessionID != a.activeSessionID {
			continue
		}

		// Format message
		var roleStyle lipgloss.Style
		var rolePrefix string

		switch msg.Role {
		case "user":
			roleStyle = a.userStyle
			rolePrefix = "You"
		case "assistant":
			roleStyle = a.assistantStyle
			rolePrefix = "AI"
		case "system":
			roleStyle = a.systemStyle
			rolePrefix = "System"
		}

		// Add timestamp if enabled
		if a.showTimestamps {
			timestamp := msg.Timestamp.Format("15:04:05")
			content.WriteString(a.timestampStyle.Render(timestamp) + " ")
		}

		// Add role and content
		content.WriteString(roleStyle.Render(rolePrefix + ": "))

		if msg.Error != "" {
			content.WriteString(a.errorStyle.Render(msg.Content))
		} else {
			content.WriteString(msg.Content)
		}

		// Add token count if enabled
		if a.showTokenCount && msg.Tokens > 0 {
			content.WriteString(a.timestampStyle.Render(fmt.Sprintf(" [%d tokens]", msg.Tokens)))
		}

		content.WriteString("\n\n")
	}

	a.messageView.SetContent(content.String())
	a.messageView.GotoBottom()
}

// Focus sets focus to this component
func (a *AICodersViewComponent) Focus() tea.Cmd {
	a.focused = true
	if a.mode == AICoderViewModeChat {
		a.inputArea.Focus()
		return textarea.Blink
	}
	return nil
}

// Blur removes focus from this component
func (a *AICodersViewComponent) Blur() tea.Cmd {
	a.focused = false
	a.inputArea.Blur()
	return nil
}

// sessionItem implements list.Item for AI sessions
type sessionItem struct {
	session AICoderSessionInfo
}

func (i sessionItem) Title() string {
	icon := "🟢"
	if i.session.State != "active" {
		icon = "⚫"
	}

	return fmt.Sprintf("%s %s (%s)", icon, i.session.Name, i.session.Model)
}

func (i sessionItem) Description() string {
	parts := []string{}

	// Add message count
	parts = append(parts, fmt.Sprintf("%d messages", i.session.MessageCount))

	// Add duration
	duration := time.Since(i.session.StartTime)
	if duration < time.Hour {
		parts = append(parts, fmt.Sprintf("%dm", int(duration.Minutes())))
	} else {
		parts = append(parts, fmt.Sprintf("%dh %dm", int(duration.Hours()), int(duration.Minutes())%60))
	}

	// Add last activity
	lastActive := time.Since(i.session.LastActivity)
	if lastActive < time.Minute {
		parts = append(parts, "Active now")
	} else if lastActive < time.Hour {
		parts = append(parts, fmt.Sprintf("%dm ago", int(lastActive.Minutes())))
	} else {
		parts = append(parts, fmt.Sprintf("%dh ago", int(lastActive.Hours())))
	}

	return strings.Join(parts, " | ")
}

func (i sessionItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.session.Name, i.session.Model, i.session.State)
}

// sessionItemDelegate implements list.ItemDelegate
type sessionItemDelegate struct{}

func (d sessionItemDelegate) Height() int { return 2 }

func (d sessionItemDelegate) Spacing() int { return 1 }

func (d sessionItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d sessionItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Styles
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("170")).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		PaddingLeft(2)

	item, ok := listItem.(sessionItem)
	if !ok {
		return
	}

	// Format the item
	str := fmt.Sprintf("%s\n  %s", item.Title(), item.Description())

	if index == m.Index() {
		fmt.Fprintf(w, "%s", selectedStyle.Render("> "+str))
	} else {
		fmt.Fprintf(w, "%s", normalStyle.Render(str))
	}
}

// Helper functions

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}
