package components

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/commands"
	ui_services "github.com/standardbeagle/brummer/internal/tui/services"
)

// IntegratedShellCommandHandler implements CommandHandler for IntegratedShell
type IntegratedShellCommandHandler struct {
	shell        *IntegratedShell
	processMgr   *process.Manager
	logStore     *logs.Store
	slashContext *commands.SlashCommandContext
	aiService    ui_services.AICoderService
}

// NewIntegratedShellCommandHandler creates a new command handler
func NewIntegratedShellCommandHandler(shell *IntegratedShell, processMgr *process.Manager, logStore *logs.Store, aiService ui_services.AICoderService) *IntegratedShellCommandHandler {
	handler := &IntegratedShellCommandHandler{
		shell:      shell,
		processMgr: processMgr,
		logStore:   logStore,
		aiService:  aiService,
	}

	// Initialize slash command context
	showPattern := ""
	hidePattern := ""
	searchResults := []logs.LogEntry{}
	currentView := "processes"

	handler.slashContext = &commands.SlashCommandContext{
		ProcessManager: processMgr,
		LogStore:       logStore,
		UpdateChan:     nil, // We'll use tea.Cmd instead
		ShowPattern:    &showPattern,
		HidePattern:    &hidePattern,
		SearchResults:  &searchResults,
		CurrentView:    &currentView,

		// Callbacks
		UpdateLogsView: func() {
			// Trigger logs view update
			if shell.logsView != nil {
				shell.logsView.applyFilter()
			}
		},
		ClearLogs: func(target string) {
			// Clear logs - would need to implement Clear method
			// For now, just log the action
			logStore.Add("system", "System", fmt.Sprintf("🗑️ Clearing logs: %s", target), false)
		},
		SetProxyURL: func(urlStr string) {
			// TODO: Implement proxy URL setting
			logStore.Add("system", "System", fmt.Sprintf("🔗 Proxy URL set: %s", urlStr), false)
		},
		ListProxies: func() {
			// TODO: Implement proxy listing
			logStore.Add("system", "System", "📋 No proxies configured", false)
		},
		ToggleProxy: func() {
			// TODO: Implement proxy toggle
			logStore.Add("system", "System", "🔄 Proxy toggled", false)
		},
		StartAICoder: func(providerName string) {
			// TODO: Implement AI coder start
			logStore.Add("system", "System", fmt.Sprintf("🤖 Starting AI Coder: %s", providerName), false)
		},
		ShowTerminal: func() {
			// TODO: Implement terminal show
			logStore.Add("system", "System", "📟 Terminal view activated", false)
		},
	}

	return handler
}

// ExecuteCommand executes a command and returns a tea.Cmd
func (h *IntegratedShellCommandHandler) ExecuteCommand(command string) tea.Cmd {
	// Trim slash prefix if present
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, "/") {
		// Handle slash command via shared handler for side effects/logging
		commands.HandleSlashCommand(h.slashContext, command)

		// Special handling: /ai <provider> should start via service (non-blocking)
		if strings.HasPrefix(command, "/ai ") && h.aiService != nil {
			parts := strings.Fields(command)
			if len(parts) >= 2 {
				provider := parts[1]
				startSpec, _ := h.aiService.Start(provider)
				return tea.Batch(
					func() tea.Msg { return HideCommandPaletteMsg{} },
					func() tea.Msg { return ShowAIViewMsg{} },
					func() tea.Msg {
						return AICoderStartMsg{
							Name:         startSpec.Name,
							Model:        startSpec.Model,
							SystemPrompt: startSpec.SystemPrompt,
							WorkingDir:   startSpec.WorkingDir,
						}
					},
				)
			}
		}

		// Default: hide palette and trigger view updates when appropriate
		return tea.Batch(
			func() tea.Msg { return HideCommandPaletteMsg{} },
			func() tea.Msg {
				if strings.HasPrefix(command, "/show") || strings.HasPrefix(command, "/hide") {
					return LogsUpdateMsg{}
				}
				return nil
			},
		)
	}

	// Handle non-slash commands (like script names)
	if h.processMgr != nil {
		// Try to start it as a script
		if _, err := h.processMgr.StartScript(command); err == nil {
			h.logStore.Add("system", "System", fmt.Sprintf("✅ Started %s", command), false)
			return func() tea.Msg {
				return HideCommandPaletteMsg{}
			}
		}
	}

	// Unknown command
	h.logStore.Add("system", "System", fmt.Sprintf("❓ Unknown command: %s", command), true)
	return func() tea.Msg {
		return HideCommandPaletteMsg{}
	}
}

// GetSuggestions returns command suggestions based on prefix
func (h *IntegratedShellCommandHandler) GetSuggestions(prefix string) []string {
	p := strings.TrimSpace(prefix)
	lower := strings.ToLower(p)

	// Helper: collect script names from process manager
	getScripts := func() []string {
		if h.processMgr == nil {
			return nil
		}
		scriptsMap := h.processMgr.GetScripts()
		if len(scriptsMap) == 0 {
			return nil
		}
		names := make([]string, 0, len(scriptsMap))
		for name := range scriptsMap {
			names = append(names, name)
		}
		sort.Strings(names)
		return names
	}

	// Helper: collect AI providers via service or config fallback
	getProviders := func() []string {
		if h.aiService != nil {
			names := h.aiService.GetProviders()
			if len(names) > 0 {
				return names
			}
		}
		cfg, _ := config.Load()
		names := make([]string, 0)
		if cfg != nil && cfg.AICoders != nil && cfg.AICoders.Providers != nil {
			for name := range cfg.AICoders.Providers {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			defaults := config.GetDefaultProviderConfigs()
			for name := range defaults {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		return names
	}

	// Root slash commands (no decoration so Enter executes correctly)
	root := []string{"/show ", "/hide ", "/run ", "/restart ", "/stop ", "/clear ", "/proxy ", "/ai ", "/terminal", "/help"}

	var out []string

	if strings.HasPrefix(lower, "/") {
		parts := strings.Fields(p)
		cmd := parts[0]
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}

		switch {
		case strings.HasPrefix("/ai", strings.TrimPrefix(cmd, "/")) || cmd == "/ai":
			providers := getProviders()
			if len(providers) == 0 {
				return []string{"/ai "}
			}
			for _, prov := range providers {
				if arg == "" || strings.HasPrefix(strings.ToLower(prov), strings.ToLower(arg)) {
					out = append(out, "/ai "+prov)
				}
			}

		case strings.HasPrefix("/run", strings.TrimPrefix(cmd, "/")) || cmd == "/run":
			scripts := getScripts()
			for _, s := range scripts {
				if arg == "" || strings.HasPrefix(strings.ToLower(s), strings.ToLower(arg)) {
					out = append(out, "/run "+s)
				}
			}

		case strings.HasPrefix("/restart", strings.TrimPrefix(cmd, "/")) || cmd == "/restart":
			scripts := getScripts()
			for _, s := range scripts {
				if arg == "" || strings.HasPrefix(strings.ToLower(s), strings.ToLower(arg)) {
					out = append(out, "/restart "+s)
				}
			}

		case strings.HasPrefix("/stop", strings.TrimPrefix(cmd, "/")) || cmd == "/stop":
			scripts := getScripts()
			for _, s := range scripts {
				if arg == "" || strings.HasPrefix(strings.ToLower(s), strings.ToLower(arg)) {
					out = append(out, "/stop "+s)
				}
			}

		case cmd == "/proxy":
			// Suggest basic subcommands; could add active URLs later
			subs := []string{"/proxy ", "/proxy list ", "/proxy http://localhost:"}
			for _, sub := range subs {
				if strings.HasPrefix(strings.ToLower(sub), lower) {
					out = append(out, sub)
				}
			}

		default:
			// Filter root commands by prefix
			for _, r := range root {
				if strings.HasPrefix(strings.ToLower(r), lower) {
					out = append(out, r)
				}
			}
		}
	} else {
		// Bare input: suggest script names (to run directly)
		scripts := getScripts()
		for _, s := range scripts {
			if p == "" || strings.HasPrefix(strings.ToLower(s), lower) {
				out = append(out, s)
			}
		}
	}

	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

// Custom messages for command palette
type HideCommandPaletteMsg struct{}
type ShowCommandPaletteMsg struct{}
