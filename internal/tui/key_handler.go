package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/services"
)

// KeyContext provides rich context about the current state for handlers
type KeyContext struct {
	View            View     // Current view (ViewProcesses, ViewAICoders, etc.)
	Context         string   // Additional context (e.g., "modal", "input", "pty")
	ActiveComponent interface{} // The active component (if needed for state checks)
	CursorPosition  int      // Cursor position in input (for slash command context)
	InputContent    string   // Current input content (for slash command context)
}

// KeyHandler defines the interface for all keyboard handlers
type KeyHandler interface {
	// CanHandle returns true if this handler should process this key
	// Now receives a richer context object instead of separate parameters
	CanHandle(key tea.KeyMsg, ctx KeyContext) bool

	// Handle processes the key and returns a command (or nil)
	Handle(key tea.KeyMsg, model *Model) tea.Cmd

	// Priority returns the handler priority (lower = higher priority)
	Priority() int

	// Name returns a descriptive name for debugging
	Name() string
}

// KeyHandlerRegistry manages keyboard handlers with clear precedence
type KeyHandlerRegistry struct {
	handlers []KeyHandler
}

// NewKeyHandlerRegistry creates a new registry
func NewKeyHandlerRegistry() *KeyHandlerRegistry {
	return &KeyHandlerRegistry{
		handlers: make([]KeyHandler, 0),
	}
}

// Register adds a handler and sorts by priority
func (r *KeyHandlerRegistry) Register(handler KeyHandler) {
	r.handlers = append(r.handlers, handler)

	// Sort by priority (lower number = higher priority)
	for i := len(r.handlers) - 1; i > 0; i-- {
		if r.handlers[i].Priority() < r.handlers[i-1].Priority() {
			r.handlers[i], r.handlers[i-1] = r.handlers[i-1], r.handlers[i]
		} else {
			break
		}
	}
}

// HandleKey routes a key to the appropriate handler
func (r *KeyHandlerRegistry) HandleKey(key tea.KeyMsg, ctx KeyContext, model *Model) tea.Cmd {
	for _, handler := range r.handlers {
		if handler.CanHandle(key, ctx) {
			cmd := handler.Handle(key, model)
			if cmd != nil {
				// Handler actually handled the key, return its command
				return cmd
			}
			// Handler matched but returned nil - continue to next handler
		}
	}

	return nil
}

// Helper function to convert key to string for logging
func keyToString(key tea.KeyMsg) string {
	switch key.Type {
	case tea.KeyRunes:
		return string(key.Runes)
	case tea.KeyTab:
		return "Tab"
	case tea.KeyEnter:
		return "Enter"
	case tea.KeyEsc:
		return "Esc"
	case tea.KeyCtrlC:
		return "Ctrl+C"
	default:
		return key.String()
	}
}

// --- Example Handler Implementations ---

// GlobalNavigationHandler handles view switching keys
// SIMPLIFIED: Now uses SimpleViewManager directly
type GlobalNavigationHandler struct {
	viewManager *SimpleViewManager
	debugMode   bool
}

func NewGlobalNavigationHandler(viewManager *SimpleViewManager, debugMode bool) KeyHandler {
	return &GlobalNavigationHandler{
		viewManager: viewManager,
		debugMode:   debugMode,
	}
}

func (h *GlobalNavigationHandler) Name() string {
	return "GlobalNavigation"
}

func (h *GlobalNavigationHandler) Priority() int {
	return 900 // LOWEST priority - navigation is fallback after all other handlers
}

func (h *GlobalNavigationHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	// Handle navigation keys globally, including in AI Coder view
	// This allows switching away from AI Coder view and prevents lockup
	// NOTE: AI Coder handler will still have priority when there's an active session

	// Handle number keys 1-7 and left/right arrow keys globally
	switch key.Type {
	case tea.KeyLeft, tea.KeyRight:
		return true
	case tea.KeyRunes:
		if len(key.Runes) == 1 {
			r := key.Runes[0]
			return r >= '1' && r <= '7'
		}
	}
	return false
}

func (h *GlobalNavigationHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	switch key.Type {
	case tea.KeyLeft:
		// Go to previous view
		prevView := CycleToPrevView(h.viewManager.GetCurrentView(), h.debugMode)
		return h.viewManager.SwitchToView(prevView)
	case tea.KeyRight:
		// Go to next view
		nextView := CycleToNextView(h.viewManager.GetCurrentView(), h.debugMode)
		return h.viewManager.SwitchToView(nextView)
	case tea.KeyRunes:
		if len(key.Runes) == 1 {
			// Try to switch by number key
			if view, ok := GetViewForNumberKey(key.Runes[0]); ok {
				return h.viewManager.SwitchToView(view)
			}
		}
	}
	return nil
}

// ViewSpecificHandler handles keys specific to certain views
type ViewSpecificKeyHandler struct {
	view    View
	handler func(tea.KeyMsg, *Model) tea.Cmd
}

func NewViewSpecificKeyHandler(view View, handler func(tea.KeyMsg, *Model) tea.Cmd) KeyHandler {
	return &ViewSpecificKeyHandler{view: view, handler: handler}
}

func (h *ViewSpecificKeyHandler) Name() string {
	return "ViewSpecific-" + string(h.view)
}

func (h *ViewSpecificKeyHandler) Priority() int {
	return 200 // After panels, before global navigation
}

func (h *ViewSpecificKeyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	return ctx.View == h.view
}

func (h *ViewSpecificKeyHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	return h.handler(key, model)
}

// QuitHandler handles quit commands with double-press protection
// Prevents accidental exits by requiring confirmation
type QuitHandler struct{}

func NewQuitHandler() KeyHandler {
	return &QuitHandler{}
}

func (h *QuitHandler) Name() string {
	return "Quit"
}

func (h *QuitHandler) Priority() int {
	return 50 // High priority - after terminal input, before panels/navigation
}

func (h *QuitHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	// Don't handle quit keys when in AI Coder view at all
	// AI Coder handler is absolute priority and handles all keys including Ctrl+C and Esc
	if ctx.View == ViewAICoders {
		return false
	}
	return key.Type == tea.KeyCtrlC || key.Type == tea.KeyEsc
}

func (h *QuitHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	// Double-press protection: require second press within 2 seconds
	now := time.Now()

	if model.awaitingQuitConfirmation {
		// Second press within timeout - actually quit
		timeSinceFirst := now.Sub(model.lastQuitAttemptTime)
		if timeSinceFirst < 2*time.Second {
			return tea.Quit
		}
		// Timeout expired, treat as first press again
		model.awaitingQuitConfirmation = true
		model.lastQuitAttemptTime = now
		return func() tea.Msg {
			return StatusMsg{Message: "Press ESC or Ctrl+C again to quit"}
		}
	}

	// First press - show warning
	model.awaitingQuitConfirmation = true
	model.lastQuitAttemptTime = now

	return func() tea.Msg {
		return StatusMsg{Message: "Press ESC or Ctrl+C again to quit"}
	}
}

// SlashCommandHandler handles the "/" key with context awareness
type SlashCommandHandler struct {
	openCommandPalette func() tea.Cmd
}

func NewSlashCommandHandler(openPalette func() tea.Cmd) KeyHandler {
	return &SlashCommandHandler{openCommandPalette: openPalette}
}

func (h *SlashCommandHandler) Name() string {
	return "SlashCommand"
}

func (h *SlashCommandHandler) Priority() int {
	return 100 // After exit keys, before view-specific
}

func (h *SlashCommandHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	// Only handle "/" key
	if key.Type != tea.KeyRunes || len(key.Runes) != 1 || key.Runes[0] != '/' {
		return false
	}

	// NEVER handle "/" when in AI Coder view - AI Coder handler has absolute priority
	if ctx.View == ViewAICoders {
		return false
	}

	// Don't handle "/" when in input field contexts:
	// - command_window: Command palette itself
	// - modal: Any modal dialog with input
	// - script_selector: Script selection input
	switch ctx.Context {
	case "command_window":
		// Don't intercept "/" in command palette - let user type it
		return false
	case "modal":
		// Don't intercept "/" in modal dialogs
		return false
	case "script_selector":
		// Don't intercept "/" in script selector
		return false
	}

	// In all other contexts (normal, file_browser, system_panel, etc),
	// always handle "/" key for command palette
	return true
}

func (h *SlashCommandHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	// Open the command palette
	if h.openCommandPalette != nil {
		return h.openCommandPalette()
	}

	// Fallback: return a message to open command palette
	return func() tea.Msg {
		return OpenCommandPaletteMsg{}
	}
}

// OpenCommandPaletteMsg signals that the command palette should be opened
type OpenCommandPaletteMsg struct{}

// AICoderKeyHandler handles special keyboard behavior in AI Coder view
type AICoderKeyHandler struct {
	aiCoderComponent interface{} // Reference to AICodersViewComponent
}

func NewAICoderKeyHandler(component interface{}) KeyHandler {
	return &AICoderKeyHandler{aiCoderComponent: component}
}

func (h *AICoderKeyHandler) Name() string {
	return "AICoderKeys"
}

func (h *AICoderKeyHandler) Priority() int {
	return 1 // ABSOLUTE HIGHEST priority - no other handlers can interfere
}

func (h *AICoderKeyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	// Only handle keys when in AI Coders view
	if ctx.View != ViewAICoders {
		return false
	}

	// When terminal is focused, handle ALL keys (they go to PTY)
	if ctx.Context == "terminal_focused" {
		return true
	}

	// When terminal is NOT focused but we're in AI Coder view,
	// we still need to handle specific control keys if there's a focused session
	// The Handle method will check if there's actually a focused session

	// Check for specific keys we want to handle when not in terminal focus
	switch key.Type {
	case tea.KeyEnter:
		// Enter key to refocus terminal (if there's a focused session)
		return true
	case tea.KeyCtrlT:
		// Ctrl+T to toggle terminal focus (if there's a focused session)
		return true
	case tea.KeyRunes:
		if len(key.Runes) == 1 && key.Runes[0] == 'q' {
			// 'q' to exit session (if there's a focused session)
			return true
		}
	}

	// Let all other keys pass through to navigation handlers
	// This allows 1-7, arrows, etc. to work normally
	return false
}


func (h *AICoderKeyHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	// Must be in AI Coders view to handle any keys
	if model.currentView() != ViewAICoders {
		return nil
	}

	// Validate we have the required state
	if model.masterData == nil || model.masterData.AICoderState == nil {
		return nil
	}

	aiState := model.masterData.AICoderState
	focusedSession := aiState.FocusedSession

	// Ctrl+T is special - it should work even without a focused session to exit AI Coder view
	if key.Type == tea.KeyCtrlT {
		if focusedSession != "" && aiState.TerminalFocused {
			// Has focused session and terminal is focused - release terminal focus
			model.masterData.AICoderState = aiState.SetTerminalFocus(false)
			return func() tea.Msg {
				return StatusMsg{Message: "Terminal focus released - arrows scroll viewport, Enter to refocus"}
			}
		} else if focusedSession != "" && !aiState.TerminalFocused {
			// Has focused session but terminal not focused - enable terminal focus
			model.masterData.AICoderState = aiState.SetTerminalFocus(true)
			return func() tea.Msg {
				return StatusMsg{Message: "Terminal focus enabled - keys go to terminal, Ctrl+T to release"}
			}
		} else {
			// No focused session - exit AI Coder view entirely
			return model.viewManager.SwitchToView(ViewProcesses)
		}
	}

	// For all other keys, must have a focused session to handle them
	if focusedSession == "" {
		return nil
	}

	
	// ==== HANDLE KEYS BASED ON TERMINAL FOCUS STATE ====
	if !aiState.TerminalFocused {
		// Terminal NOT focused - only handle specific control keys
		switch key.Type {
		case tea.KeyEnter:
			// Enter key when not in terminal focus mode - refocus terminal and send Enter
			model.masterData.AICoderState = aiState.SetTerminalFocus(true)

			// Also send the Enter key to the terminal
			if model.aiCoderService != nil {
				svc := model.aiCoderService.(*services.AICoderService)
				svc.SendInput(focusedSession, []byte("\r"))
			}

			return func() tea.Msg {
				return StatusMsg{Message: "Terminal focus enabled - keys go to terminal, Ctrl+T to release"}
			}

		case tea.KeyRunes:
			if len(key.Runes) == 1 && key.Runes[0] == 'q' {
				// Clear focused session completely - exit to session list
				model.masterData.AICoderState = aiState.SetTerminalFocus(false).SetFocusedSession("")

				return func() tea.Msg {
					return StatusMsg{Message: "Exited AI Coder session - back to session list"}
				}
			}
		}
		// For any other keys when not focused, return nil to let other handlers process them
		return nil
	}

	// ==== TERMINAL IS FOCUSED ====
	// When terminal is focused, send ALL keys to PTY using KeyToPTY converter
	// ESC key is NOT intercepted here so it can be used by AI coder tools (vim, nvim, etc.)
	// Only Ctrl+T can release terminal focus

	// Send keys to PTY
	input, err := KeyToPTY(key)
	if err != nil {
		// Log unsupported key but don't crash
		if model.logStore != nil {
			model.logStore.Add("ai_coder", "AI Coder",
				fmt.Sprintf("Unsupported key for PTY: %v", err), false)
		}
		return nil
	}

	// Send input to PTY via the service
	if model.aiCoderService != nil {
		// Use type assertion with interface check for test compatibility
		if svc, ok := model.aiCoderService.(interface{ SendInput(string, []byte) error }); ok {
			if err := svc.SendInput(focusedSession, input); err != nil {
				// Log error but don't crash
				if model.logStore != nil {
					model.logStore.Add("system", "System",
						fmt.Sprintf("Error sending input to PTY: %v", err), true)
				}
			}
		}
	}

	return nil
}

// Messages for AI Coder mode transitions
type ExitPTYModeMsg struct{}
type EnterPTYModeMsg struct{}


// GeneralKeyHandler handles common keys that aren't handled by other handlers
// This prevents "nothing happens" when users press common navigation/action keys
type GeneralKeyHandler struct{}

func NewGeneralKeyHandler() KeyHandler {
	return &GeneralKeyHandler{}
}

func (h *GeneralKeyHandler) Name() string     { return "GeneralKeys" }
func (h *GeneralKeyHandler) Priority() int    { return 900 } // Low priority, catch-all

func (h *GeneralKeyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	// NEVER handle keys when in AI Coder view - AI Coder handler has absolute priority
	if ctx.View == ViewAICoders {
		return false
	}

	// Don't handle keys in AI Coder PTY mode (let them pass through)
	if ctx.Context == "pty" {
		return false
	}

	// Handle common keys that might otherwise be ignored
	switch key.Type {
	case tea.KeyEnter, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete:
		return true
	case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight:
		return true
	case tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		return true
	case tea.KeyRunes:
		// Handle common single-letter keys that might be pressed
		if len(key.Runes) == 1 {
			r := key.Runes[0]
			// Letters that are commonly used for commands
			switch r {
			case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
				 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
				 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
				 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
				return true
			}
		}
	}
	return false
}

func (h *GeneralKeyHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	// Return a helpful message for common keys
	switch key.Type {
	case tea.KeyEnter:
		return func() tea.Msg {
			return StatusMsg{Message: "Enter key - select item or confirm action"}
		}
	case tea.KeyUp, tea.KeyDown:
		return func() tea.Msg {
			return StatusMsg{Message: "Use ↑/↓ or j/k to navigate, 1-7 to switch views"}
		}
	case tea.KeyLeft, tea.KeyRight:
		return func() tea.Msg {
			return StatusMsg{Message: "Use ←/→ to switch views, Tab for next view"}
		}
	case tea.KeyRunes:
		if len(key.Runes) == 1 {
			r := key.Runes[0]
			switch r {
			case 'q':
				return func() tea.Msg {
					return StatusMsg{Message: "Press Ctrl+C or Esc to quit"}
				}
			case 'h', '?':
				return func() tea.Msg {
					return StatusMsg{Message: "Press / for command palette, 1-7 for views"}
				}
			case 'f':
				return func() tea.Msg {
					return StatusMsg{Message: "Filter commands available in specific views"}
				}
			case 's':
				return func() tea.Msg {
					return StatusMsg{Message: "Stop commands available in Processes view"}
				}
			case 'r':
				return func() tea.Msg {
					return StatusMsg{Message: "Restart commands available in Processes view"}
				}
			case 'c':
				return func() tea.Msg {
					return StatusMsg{Message: "Clear commands available in Logs/Errors views"}
				}
			default:
				return func() tea.Msg {
					return StatusMsg{Message: fmt.Sprintf("Key '%c' not handled - try / for commands", r)}
				}
			}
		}
	}

	return nil
}

// StatusMsg displays a status message to the user
type StatusMsg struct {
	Message string
}

// ErrorMsg represents an error that occurred
type ErrorMsg struct {
	Error error
}