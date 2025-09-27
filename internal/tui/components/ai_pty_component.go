package components

// AIPTYComponent: AI tab that hosts an interactive PTY session for an AI provider.
// Purpose: When user runs "/ai <provider>", create a real PTY session using the
// provider's CLI, wire keyboard input -> PTY, stream output -> viewport, and keep
// it responsive. External deps: internal/aicoder for PTY, internal/config for
// provider CLI config, internal/logs for user-facing feedback.
// Notes:
// - Keeps UI non-blocking: session creation and output subscription are done via tea.Cmds
// - Minimal renderer: shows raw PTY output history preserving ANSI sequences
// - Resizes PTY on WindowSizeMsg
// - Input: forwards typed runes and basic keys to PTY using WriteInputNonBlocking

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
)

// AIPTYComponent renders an interactive PTY terminal for AI providers
type AIPTYComponent struct {
	BaseComponent

	logStore   *logs.Store
	session    *aicoder.PTYSession
	provider   string
	viewport   viewport.Model
	cmdSummary string // for diagnostics: "command args..."

	// instrumentation / diagnostics
	loggedFirstOutput bool
	isStarting        bool
	spinnerIndex      int
	statusMessage     string
	startTime         time.Time
	truncatedNotified bool // one-time notification when content exceeds viewport

	// debug diagnostics
	pendingSubscribe bool      // true right after we return a subscribe cmd
	outputCount      int       // number of output messages received
	lastOutputAt     time.Time // time of last output
	lastError        string    // last error encountered (shown in UI)
}

// internal spinner message
type aiptySpinnerMsg struct{}

// periodic refresh tick to keep viewport in sync with terminal
type aiptyRefreshTickMsg struct{}

func NewAIPTYComponent(logStore *logs.Store) *AIPTYComponent {
	c := &AIPTYComponent{logStore: logStore}
	c.viewport = viewport.New(80, 20)
	c.visible = true
	return c
}

func (c *AIPTYComponent) Focus() tea.Cmd { c.focused = true; return nil }
func (c *AIPTYComponent) Blur() tea.Cmd  { c.focused = false; return nil }
func (c *AIPTYComponent) SetSize(w, h int) {
	c.width, c.height = w, h
	// Defer viewport sizing to View/Update to avoid off-by-one due to status line
	if c.session != nil {
		_ = c.session.Resize(w, h)
	}
}

// subscribePTY returns a command that waits for next PTY output and emits AIPTYOutputMsg
func (c *AIPTYComponent) subscribePTY() tea.Cmd {
	if c.session == nil {
		return nil
	}
	s := c.session
	c.pendingSubscribe = true
	return func() tea.Msg {
		data, ok := <-s.OutputChan
		if !ok {
			return AIPTYEventMsg{Type: "closed", Msg: "PTY output channel closed"}
		}
		return AIPTYOutputMsg{SessionID: s.ID, Data: append([]byte(nil), data...)}
	}
}

// subscribePTYEvents listens for PTY EventChan messages (e.g., close/errors) and emits AIPTYEventMsg
func (c *AIPTYComponent) subscribePTYEvents() tea.Cmd {
	if c.session == nil {
		return nil
	}
	s := c.session
	return func() tea.Msg {
		ev, ok := <-s.EventChan
		if !ok {
			return AIPTYEventMsg{Type: "closed", Msg: "PTY event channel closed"}
		}
		// Surface key events to UI for diagnostics and fallback rendering
		switch ev.Type {
		case aicoder.PTYEventClose:
			msg := "PTY closed"
			if str, ok := ev.Data.(string); ok && str != "" {
				msg = str
			}
			return AIPTYEventMsg{Type: "closed", Msg: msg}
		case aicoder.PTYEventOutput:
			// Intentional no-op: avoid duplicating output via EventChan. OutputChan is the
			// single source of truth for UI rendering. We still keep the subscription alive.
			return AIPTYEventMsg{Type: "other", Msg: "event-output-ignored"}
		default:
			// Keep the loop alive even for non-output events
			return AIPTYEventMsg{Type: "other", Msg: "event-other"}
		}
	}

}

// startProviderSession creates the provider CLI PTY session based on config
func (c *AIPTYComponent) startProviderSession(provider string) tea.Cmd {
	return func() tea.Msg {
		cfg, _ := config.Load()
		mcpPort := 7777
		if cfg != nil {
			mcpPort = cfg.GetMCPPort()
		}

		var command string
		args := []string{}
		env := map[string]string{
			"BRUMMER_MCP_URL":  fmt.Sprintf("http://localhost:%d/mcp", mcpPort),
			"BRUMMER_MCP_PORT": fmt.Sprintf("%d", mcpPort),
			"TERM":             "xterm-256color",
		}
		if cfg != nil && cfg.AICoders != nil && cfg.AICoders.Providers != nil {
			if pcfg, ok := cfg.AICoders.Providers[provider]; ok && pcfg != nil && pcfg.CLITool != nil {
				if pcfg.CLITool.Command != nil {
					command = *pcfg.CLITool.Command
				}
				args = append(args, pcfg.CLITool.BaseArgs...)
				for i, a := range args {
					args[i] = strings.ReplaceAll(a, "${BRUMMER_MCP_URL}", env["BRUMMER_MCP_URL"])
				}
				for k, v := range pcfg.CLITool.Environment {
					env[k] = v
				}
			}
		}
		// Debug provider: /ai test -> emit a counter in the terminal
		if provider == "test" {
			command = "/bin/bash"
			args = []string{"-lc", "i=0; while true; do echo \"BRUMMER_TEST_COUNTER $i\"; i=$((i+1)); sleep 0.5; done"}
		}
		if provider == "terminal" || command == "" {
			command = "/bin/bash"
			args = []string{"-i"}
		}

		c.cmdSummary = fmt.Sprintf("%s %v", command, args)
		if c.logStore != nil {
			c.logStore.Add("ai-pty", "AI", fmt.Sprintf("Starting provider %q: %s", provider, c.cmdSummary), false)
		}

		// Validate the provider command exists on PATH (for immediate feedback)
		if _, lookErr := exec.LookPath(command); lookErr != nil {
			c.isStarting = false
			c.statusMessage = fmt.Sprintf("Command not found: %s", command)
			if c.logStore != nil {
				c.logStore.Add("ai-pty", "AI", c.statusMessage, true)
			}
			return nil
		}

		id := uuid.New().String()
		sess, err := aicoder.NewPTYSessionWithEnv(id, fmt.Sprintf("%s AI", provider), command, args, env)
		if err != nil {
			c.isStarting = false
			c.statusMessage = fmt.Sprintf("Failed to start: %v", err)
			if c.logStore != nil {
				c.logStore.Add("ai-pty", "AI", fmt.Sprintf("AI PTY failed: %v", err), true)
			}
			return nil
		}
		c.session = sess
		c.provider = provider
		c.loggedFirstOutput = false
		// Log PID for diagnostics
		if c.logStore != nil && c.session.Command != nil && c.session.Command.Process != nil {
			c.logStore.Add("ai-pty", "AI", fmt.Sprintf("PTY started: pid=%d", c.session.Command.Process.Pid), false)
		}
		// initial resize to the full available area (no internal frame)
		if c.width > 0 && c.height > 0 {
			_ = c.session.Resize(maxInt(20, c.width), maxInt(5, c.height))
		}

		// For terminal provider only, emit a readiness line so viewport shows content
		if provider == "terminal" {
			if err := c.session.WriteInputNonBlocking([]byte("echo BRUMMER_READY\n")); err != nil {
				c.lastError = fmt.Sprintf("readiness echo failed: %v", err)
				if c.logStore != nil {
					c.logStore.Add("ai-pty", "AI", c.lastError, true)
				}
			}
		}

		// Signal that session is ready; UI will begin subscriptions
		return AIPTYEventMsg{Type: "started", Msg: "pty-session-ready"}
	}
}

func (c *AIPTYComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(m.Width, m.Height)
		return c, nil

	case AIPTYStartMsg:
		c.isStarting = true
		c.spinnerIndex = 0
		c.statusMessage = fmt.Sprintf("Starting %q...", m.Provider)
		c.startTime = time.Now()
		if c.logStore != nil {
			c.logStore.Add("ai-pty", "AI", fmt.Sprintf("AIPTYStartMsg received for provider %q", m.Provider), false)
		}
		// Start provider PTY and spinner while waiting for first output
		return c, tea.Batch(c.startProviderSession(m.Provider), c.spinnerTickCmd())

	case AIPTYOutputMsg:
		// New data available; refresh viewport content and resubscribe for more
		if c.session != nil && m.SessionID == c.session.ID {
			c.outputCount++
			c.lastOutputAt = time.Now()
			c.pendingSubscribe = true // Keep subscribing
			if !c.loggedFirstOutput {
				c.loggedFirstOutput = true
				c.isStarting = false
				c.statusMessage = ""
				if c.logStore != nil {
					c.logStore.Add("ai-pty", "AI", fmt.Sprintf("First PTY output received (len=%d)", len(m.Data)), false)
				}
			}
			// Prefer current terminal screen from vt10x to support cursor navigation
			termStr := c.session.GetTerminal().String()
			c.viewport.SetContent(termStr)
			c.viewport.GotoBottom()
			return c, tea.Batch(c.subscribePTY(), c.subscribePTYEvents())
		}
		return c, nil

	case AIPTYEventMsg:
		// Lifecycle and output events via EventChan; always resubscribe unless closed
		switch m.Type {
		case "started":
			if c.session != nil {
				// Update terminal content immediately and start both refresh tick and subscriptions
				termStr := c.session.GetTerminal().String()
				c.viewport.SetContent(termStr)
				c.viewport.GotoBottom()
				return c, tea.Batch(c.refreshTickCmd(), c.subscribePTY(), c.subscribePTYEvents())
			}
		case "output":
			// Treat as output signal; refresh from terminal
			c.outputCount++
			c.lastOutputAt = time.Now()
			c.pendingSubscribe = true // Keep subscribing
			if !c.loggedFirstOutput {
				c.loggedFirstOutput = true
				c.isStarting = false
				c.statusMessage = ""
			}
			if c.session != nil {
				termStr := c.session.GetTerminal().String()
				c.viewport.SetContent(termStr)
				c.viewport.GotoBottom()
			}
			return c, tea.Batch(c.subscribePTY(), c.subscribePTYEvents())
		case "closed":
			c.isStarting = false
			c.pendingSubscribe = false
			c.lastError = ifElse(m.Msg != "", m.Msg, "PTY channel closed")
			c.statusMessage = c.lastError
			if c.logStore != nil {
				c.logStore.Add("ai-pty", "AI", c.lastError, true)
			}
			return c, nil
		}
		// Unknown/other events: keep listening
		c.pendingSubscribe = true
		return c, tea.Batch(c.subscribePTY(), c.subscribePTYEvents())

	case aiptySpinnerMsg:
		// Animate spinner while starting (until first output or failure)
		if c.isStarting && !c.loggedFirstOutput {
			c.spinnerIndex++
			// If startup takes too long, surface a hint
			if time.Since(c.startTime) > 10*time.Second && c.statusMessage == "" {
				c.statusMessage = "Still starting... verify provider CLI is installed and configured"
			}
			// If terminal already has content (history > 0), refresh view and clear spinner
			if c.session != nil && len(c.session.GetOutputHistory()) > 0 {
				c.loggedFirstOutput = true
				c.isStarting = false
				c.statusMessage = ""
				termStr := c.session.GetTerminal().String()
				c.viewport.SetContent(termStr)
				c.viewport.GotoBottom()
				return c, nil
			}
			// After 2s with no output, poke the PTY to coax a prompt
			if time.Since(c.startTime) > 2*time.Second && c.outputCount == 0 && c.session != nil {
				return c, tea.Batch(c.spinnerTickCmd(), c.pokePTYCmd())
			}
			return c, c.spinnerTickCmd()
		}
		return c, nil

	case aiptyRefreshTickMsg:
		// Periodic refresh: keep viewport in sync even if output events are missed
		if c.session != nil {
			termStr := c.session.GetTerminal().String()
			c.viewport.SetContent(termStr)
			c.viewport.GotoBottom()
			return c, c.refreshTickCmd()
		}
		return c, nil

	case tea.KeyMsg:
		if !c.focused || c.session == nil {
			return c, nil
		}
		// Basic key handling
		switch m.Type {
		case tea.KeyRunes:
			_ = c.session.WriteInputNonBlocking([]byte(string(m.Runes)))
			return c, nil
		case tea.KeyEnter:
			_ = c.session.WriteInputNonBlocking([]byte("\r"))
			return c, nil
		case tea.KeyBackspace:
			_ = c.session.WriteInputNonBlocking([]byte("\x7f"))
			return c, nil
		case tea.KeyCtrlC:
			_ = c.session.WriteInputNonBlocking([]byte("\x03"))
			return c, nil
		case tea.KeyUp:
			_ = c.session.WriteInputNonBlocking([]byte("\x1b[A"))
			return c, nil
		case tea.KeyDown:
			_ = c.session.WriteInputNonBlocking([]byte("\x1b[B"))
			return c, nil
		case tea.KeyLeft:
			_ = c.session.WriteInputNonBlocking([]byte("\x1b[D"))
			return c, nil
		case tea.KeyRight:
			_ = c.session.WriteInputNonBlocking([]byte("\x1b[C"))
			return c, nil
		}
	}
	return c, nil
}

func (c *AIPTYComponent) View() string {
	if !c.visible {
		return ""
	}
	// ensure viewport size (no internal frame subtract)
	c.viewport.Width = maxInt(20, c.width)
	c.viewport.Height = maxInt(5, c.height)

	// Build status/diagnostic overlay
	status := ""
	if c.isStarting && !c.loggedFirstOutput {
		spinner := []string{"|", "/", "-", "\\"}
		status = fmt.Sprintf("%s %s", spinner[c.spinnerIndex%len(spinner)], ifElse(c.statusMessage != "", c.statusMessage, "Starting..."))
	} else if c.statusMessage != "" {
		status = c.statusMessage
	}

	showDiag := c.isStarting || c.outputCount == 0 || c.lastError != ""
	diag := ""
	if showDiag {
		lastOut := "-"
		if !c.lastOutputAt.IsZero() {
			lastOut = c.lastOutputAt.Format(time.Kitchen)
		}
		var outSz, inSz, evSz int
		var lastChk time.Time
		pid := -1
		active := false
		histLen := 0
		if c.session != nil {
			// Get immediate channel stats snapshot
			c.session.UpdateChannelStats()
			outSz, inSz, evSz, lastChk = c.session.GetChannelStats()
			active = c.session.IsActive
			histLen = len(c.session.GetOutputHistory())
			if c.session.Command != nil && c.session.Command.Process != nil {
				pid = c.session.Command.Process.Pid
			}
		}
		lastChkStr := "-"
		if !lastChk.IsZero() {
			lastChkStr = lastChk.Format(time.Kitchen)
		}
		diag = fmt.Sprintf("[AI PTY] provider=%s subscribed=%t outputs=%d lastOut=%s size=%dx%d err=%s cmd=%s pid=%d active=%t hist=%d ch(o/i/e)=%d/%d/%d chk=%s",
			c.provider, c.pendingSubscribe, c.outputCount, lastOut, c.viewport.Width, c.viewport.Height, c.lastError, c.cmdSummary, pid, active, histLen, outSz, inSz, evSz, lastChkStr)
	}

	// if no session yet
	if c.session == nil {
		msg := "AI PTY: no active session. Use /ai <provider> to start."
		prefix := ""
		if status != "" {
			prefix = status
		}
		if diag != "" {
			if prefix != "" {
				prefix += "\n"
			}
			prefix += diag
		}
		if prefix != "" {
			msg = prefix + "\n\n" + msg
		}
		return msg
	}
	// content already set from output history
	content := c.viewport.View()

	// If content is empty and we have no status/diag, show a helpful message
	if content == "" && status == "" && diag == "" {
		content = "Waiting for AI provider output...\nPress any key to interact with the session."
	}

	prefix := ""
	if status != "" {
		prefix = status
	}
	if diag != "" {
		if prefix != "" {
			prefix += "\n"
		}
		prefix += diag
	}
	if prefix != "" {
		content = prefix + "\n" + content
	}
	return content
}

// helpers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *AIPTYComponent) refreshTickCmd() tea.Cmd {
	if c.session == nil {
		return nil
	}
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return aiptyRefreshTickMsg{} })
}

func (c *AIPTYComponent) spinnerTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return aiptySpinnerMsg{} })
}

// pokePTYCmd sends a gentle newline (and readiness echo for terminal) to coax output
func (c *AIPTYComponent) pokePTYCmd() tea.Cmd {
	if c.session == nil {
		return nil
	}
	provider := c.provider
	return func() tea.Msg {
		// Best-effort poke; ignore errors
		_ = c.session.WriteInputNonBlocking([]byte("\n"))
		if provider == "terminal" {
			_ = c.session.WriteInputNonBlocking([]byte("echo BRUMMER_READY\n"))
		}
		return nil
	}
}

func ifElse[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
