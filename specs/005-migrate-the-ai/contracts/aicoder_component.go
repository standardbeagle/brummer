// Package contracts defines the component interfaces for AI coder functionality.
// This is a design contract - not executable code.
//
// Location (when implemented): internal/tui/components/aicoders_view.go (enhance existing)
package contracts

import (
	tea "github.com/charmbracelet/bubbletea"
)

// AICodersViewComponent implements the BubbleTea Model interface for AI coder UI.
//
// Responsibilities:
//   - Render session list and PTY terminal views
//   - Handle keyboard input (navigation, selection, input forwarding)
//   - Manage view state (list vs PTY mode, scroll positions)
//   - Subscribe to service events and update UI
//
// Architecture:
//   - Extends BaseComponent (internal/tui/components/base.go)
//   - Calls AICoderService for all business logic
//   - Receives updates via BubbleTea messages
//   - Maintains UI-only state (scroll, selection, viewport)
//
// Thread Safety:
//   - NOT thread-safe (BubbleTea guarantees serial Update() calls)
//   - All state modifications must occur in Update() method
//   - Do NOT mutate state from Init() or View()
type AICodersViewComponent interface {
	// BubbleTea Model Interface

	// Init initializes the component and returns initial commands.
	//
	// Returns:
	//   tea.Cmd: Initial commands (typically FetchSessionListCmd)
	//
	// Post-conditions:
	//   - Component initialized with default view state
	//   - Service reference set
	//   - Viewport created with default dimensions
	//
	// Example:
	//   component := NewAICodersViewComponent(service)
	//   cmd := component.Init()
	Init() tea.Cmd

	// Update handles BubbleTea messages and updates component state.
	//
	// Parameters:
	//   msg: Message to process (KeyMsg, WindowSizeMsg, service events, etc.)
	//
	// Returns:
	//   tea.Model: Updated component (typically self)
	//   tea.Cmd: Commands to execute (may be nil)
	//
	// Message Types Handled:
	//   - tea.KeyMsg: Keyboard input (navigation, selection, PTY input)
	//   - tea.WindowSizeMsg: Resize viewport
	//   - AICoderSessionCreatedMsg: Add session to list
	//   - AICoderSessionUpdatedMsg: Update session in list
	//   - AICoderSessionEndedMsg: Remove session from list
	//   - AICoderPTYOutputMsg: Trigger terminal refresh
	//   - AICoderErrorMsg: Display error notification
	//
	// State Modifications:
	//   - viewState updated based on message
	//   - sessions list cached from service
	//   - terminal state refreshed on PTY output
	//   - viewport resized on WindowSizeMsg
	//
	// Example:
	//   model, cmd := component.Update(msg)
	//   return model, cmd
	Update(tea.Msg) (tea.Model, tea.Cmd)

	// View renders the component to a string.
	//
	// Returns:
	//   string: Rendered UI (ANSI escape codes for styling)
	//
	// Rendering Logic:
	//   - ViewModeList: Render session list with selection
	//   - ViewModePTY: Render terminal view with PTY output
	//   - Viewport applied for scrolling
	//   - Styles applied for visual hierarchy
	//
	// Performance:
	//   - Must be fast (<16ms for 60fps rendering)
	//   - No I/O operations (read-only state access)
	//   - Cached terminal state (no service calls)
	//
	// Example:
	//   output := component.View()
	//   fmt.Print(output)
	View() string

	// BaseComponent Interface (from components/base.go)

	// SetSize updates viewport dimensions.
	//
	// Parameters:
	//   width: New viewport width
	//   height: New viewport height
	//
	// Post-conditions:
	//   - Viewport resized
	//   - Terminal resized (if in PTY mode)
	//   - ResizePTYCmd issued for active session
	//
	// Example:
	//   component.SetSize(120, 40)
	SetSize(width, height int)

	// GetSize returns current viewport dimensions.
	//
	// Returns:
	//   width: Current viewport width
	//   height: Current viewport height
	//
	// Example:
	//   w, h := component.GetSize()
	GetSize() (width, height int)

	// Focus sets component focus state.
	//
	// Parameters:
	//   focused: True to focus, false to blur
	//
	// Post-conditions:
	//   - Focus state updated
	//   - Keyboard input enabled/disabled
	//   - Visual focus indicator updated
	//
	// Example:
	//   component.Focus(true)  // Enable keyboard input
	Focus(focused bool)

	// IsFocused returns focus state.
	//
	// Returns:
	//   bool: True if focused, false otherwise
	//
	// Example:
	//   if component.IsFocused() {
	//       // Handle keyboard input
	//   }
	IsFocused() bool
}

// View Modes

const (
	// ViewModeList shows session list (navigation, selection)
	ViewModeList ViewMode = "list"

	// ViewModePTY shows active session terminal (full PTY view)
	ViewModePTY ViewMode = "pty"
)

type ViewMode string

// Key Bindings

// Key bindings follow Vim conventions where applicable:
//
// List Mode:
//   j / down    - Navigate down
//   k / up      - Navigate up
//   enter       - Open selected session (switch to PTY mode)
//   n           - Create new session (prompt for name/command)
//   d           - Delete selected session
//   r           - Refresh session list
//   ?           - Toggle help overlay
//
// PTY Mode:
//   esc         - Return to list mode
//   ctrl+c      - Send interrupt to PTY
//   ctrl+d      - Send EOF to PTY
//   pgup/pgdn   - Scroll terminal history
//   ?           - Toggle help overlay
//   [Any Key]   - Forward to PTY (typing responsiveness)
//
// Global:
//   /           - Open command palette (if terminal not focused)
//   q           - Quit (with confirmation)

// Command Constructors

// These functions construct BubbleTea commands for async operations.
// Actual implementation will be in the component file.

// FetchSessionListCmd fetches session list from service.
//
// Returns:
//   tea.Cmd: Command that fetches sessions and returns SessionListMsg
//
// Example:
//   cmd := FetchSessionListCmd(service)
//   return component, cmd
//
// Implementation:
//   return func() tea.Msg {
//       sessions := service.ListSessions()
//       return SessionListMsg{Sessions: sessions}
//   }
func FetchSessionListCmd(service AICoderService) tea.Cmd

// FetchTerminalStateCmd fetches terminal state for a session.
//
// Parameters:
//   service: Service to query
//   sessionID: Session to fetch state for
//
// Returns:
//   tea.Cmd: Command that fetches terminal state and returns TerminalStateMsg
//
// Example:
//   cmd := FetchTerminalStateCmd(service, sessionID)
//   return component, cmd
//
// Implementation:
//   return func() tea.Msg {
//       state, err := service.GetTerminalState(sessionID)
//       if err != nil {
//           return AICoderErrorMsg{SessionID: sessionID, Error: err}
//       }
//       return TerminalStateMsg{SessionID: sessionID, State: state}
//   }
func FetchTerminalStateCmd(service AICoderService, sessionID string) tea.Cmd

// StartSessionCmd starts a new AI coder session.
//
// Parameters:
//   service: Service to call
//   name: Session name
//   command: Command to execute
//   workingDir: Working directory
//   width: Terminal width
//   height: Terminal height
//
// Returns:
//   tea.Cmd: Command that creates and starts session
//
// Example:
//   cmd := StartSessionCmd(service, "Debug", "npx code", ".", 80, 24)
//   return component, cmd
//
// Implementation:
//   return func() tea.Msg {
//       sessionID, err := service.CreateSession(ctx, name, command, workingDir)
//       if err != nil {
//           return AICoderErrorMsg{Error: err}
//       }
//       if err := service.StartSession(ctx, sessionID, width, height); err != nil {
//           return AICoderErrorMsg{SessionID: sessionID, Error: err}
//       }
//       return AICoderSessionCreatedMsg{SessionID: sessionID, Name: name}
//   }
func StartSessionCmd(service AICoderService, name, command, workingDir string, width, height int) tea.Cmd

// StopSessionCmd stops a session.
//
// Parameters:
//   service: Service to call
//   sessionID: Session to stop
//
// Returns:
//   tea.Cmd: Command that stops session
//
// Example:
//   cmd := StopSessionCmd(service, sessionID)
//   return component, cmd
//
// Implementation:
//   return func() tea.Msg {
//       if err := service.StopSession(ctx, sessionID); err != nil {
//           return AICoderErrorMsg{SessionID: sessionID, Error: err}
//       }
//       return AICoderSessionEndedMsg{SessionID: sessionID, Reason: "stopped"}
//   }
func StopSessionCmd(service AICoderService, sessionID string) tea.Cmd

// SendInputCmd sends keyboard input to PTY.
//
// Parameters:
//   service: Service to call
//   sessionID: Session to send input to
//   input: Raw bytes to send
//
// Returns:
//   tea.Cmd: Command that sends input (may return error msg)
//
// Example:
//   cmd := SendInputCmd(service, sessionID, []byte("ls\n"))
//   return component, cmd
//
// Implementation:
//   return func() tea.Msg {
//       if err := service.SendInput(sessionID, input); err != nil {
//           return AICoderErrorMsg{SessionID: sessionID, Error: err}
//       }
//       return nil  // Silent success
//   }
func SendInputCmd(service AICoderService, sessionID string, input []byte) tea.Cmd

// Internal Message Types (Component-only)

// SessionListMsg contains fetched session list.
type SessionListMsg struct {
	Sessions []SessionSummary
}

// TerminalStateMsg contains fetched terminal state.
type TerminalStateMsg struct {
	SessionID string
	State     *TerminalState
}

// Rendering Helpers

// RenderSessionList renders the session list view.
//
// Parameters:
//   sessions: Sessions to display
//   selectedIndex: Currently selected index
//   width: Viewport width
//   height: Viewport height
//   styles: Styling configuration
//
// Returns:
//   string: Rendered session list
//
// Format:
//   Active Sessions (3)
//   ───────────────────
//   ▸ Debug API      [ACTIVE]   2m ago
//     Test Runner    [IDLE]     5m ago
//     Build Watch    [ERROR]    10m ago
//
// Example:
//   output := RenderSessionList(sessions, 0, 80, 24, styles)
func RenderSessionList(sessions []SessionSummary, selectedIndex, width, height int, styles AICoderStyles) string

// RenderPTYView renders the terminal view.
//
// Parameters:
//   state: Terminal state to render
//   width: Viewport width
//   height: Viewport height
//   styles: Styling configuration
//
// Returns:
//   string: Rendered terminal view
//
// Format:
//   Session: Debug API
//   ────────────────────────────────────
//   [Terminal Output Lines]
//   ...
//   [Cursor Indicator]
//   ────────────────────────────────────
//   ESC=List | ?=Help | ↑↓=Scroll
//
// Example:
//   output := RenderPTYView(state, 80, 24, styles)
func RenderPTYView(state *TerminalState, width, height int, styles AICoderStyles) string

// Performance Requirements

// The component must meet these performance targets:
//
// Rendering:
//   - View() must complete in <16ms (60fps)
//   - No blocking I/O in View() (use cached state)
//   - Terminal rendering optimized for 1000+ lines
//
// Responsiveness:
//   - Keyboard input forwarded to PTY in <50ms
//   - View mode switching in <100ms
//   - Session list updates in <50ms
//
// Memory:
//   - Terminal state caching limited to active session
//   - Session list cached (lightweight summaries)
//   - No unbounded growth (cleanup on session end)
