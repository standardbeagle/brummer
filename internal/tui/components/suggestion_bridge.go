package components

import (
	"fmt"
	"strings"

	"github.com/standardbeagle/brummer/internal/process"
)

// DynamicSuggestionProvider bridges to existing dynamic suggestion sources
type DynamicSuggestionProvider struct {
	processMgr *process.Manager
	scripts    map[string]string // script name -> description
}

// NewDynamicSuggestionProvider creates a provider that bridges to existing sources
func NewDynamicSuggestionProvider(
	processMgr *process.Manager,
	scripts map[string]string,
) *DynamicSuggestionProvider {
	return &DynamicSuggestionProvider{
		processMgr: processMgr,
		scripts:    scripts,
	}
}

// Suggest returns command suggestions from all dynamic sources
func (p *DynamicSuggestionProvider) Suggest(prefix string) []CommandSuggestion {
	var suggestions []CommandSuggestion

	// Remove leading slash if present
	cleanPrefix := strings.TrimPrefix(prefix, "/")
	lowerPrefix := strings.ToLower(cleanPrefix)

	// Static navigation commands
	staticCommands := []CommandSuggestion{
		{Command: "/processes", Description: "View running processes", Category: "Navigation", Shortcut: "1"},
		{Command: "/logs", Description: "View process logs", Category: "Navigation", Shortcut: "2"},
		{Command: "/errors", Description: "View error logs", Category: "Navigation", Shortcut: "3"},
		{Command: "/urls", Description: "View discovered URLs", Category: "Navigation", Shortcut: "4"},
		{Command: "/web", Description: "Open web viewer", Category: "Navigation", Shortcut: "5"},
		{Command: "/ai", Description: "Open AI Coders", Category: "Navigation", Shortcut: "6"},
		{Command: "/settings", Description: "View settings", Category: "Navigation", Shortcut: "7"},
		{Command: "/quit", Description: "Quit application", Category: "System", Shortcut: "q"},
		{Command: "/help", Description: "Show help", Category: "System", Shortcut: "?"},
	}

	// Add static commands that match
	for _, cmd := range staticCommands {
		if matchesPrefix(cmd.Command, "/"+lowerPrefix) {
			suggestions = append(suggestions, cmd)
		}
	}

	// Script commands (start/restart)
	if p.scripts != nil {
		for name, desc := range p.scripts {
			if matchesPrefix("start "+name, lowerPrefix) {
				suggestions = append(suggestions, CommandSuggestion{
					Command:     "/start " + name,
					Description: fmt.Sprintf("Start %s: %s", name, desc),
					Category:    "Process",
				})
			}
			if matchesPrefix("restart "+name, lowerPrefix) {
				suggestions = append(suggestions, CommandSuggestion{
					Command:     "/restart " + name,
					Description: fmt.Sprintf("Restart %s: %s", name, desc),
					Category:    "Process",
				})
			}
		}
	}

	// Running process commands (stop/restart)
	if p.processMgr != nil {
		processes := p.processMgr.GetAllProcesses()
		for _, proc := range processes {
			if proc.Status == "running" {
				if matchesPrefix("stop "+proc.Name, lowerPrefix) {
					suggestions = append(suggestions, CommandSuggestion{
						Command:     "/stop " + proc.Name,
						Description: fmt.Sprintf("Stop %s", proc.Name),
						Category:    "Process",
					})
				}
				if matchesPrefix("restart "+proc.Name, lowerPrefix) {
					suggestions = append(suggestions, CommandSuggestion{
						Command:     "/restart " + proc.Name,
						Description: fmt.Sprintf("Restart %s", proc.Name),
						Category:    "Process",
					})
				}
			}
		}
	}

	// AI provider commands - simplified for now
	if strings.HasPrefix(lowerPrefix, "ai") {
		// Add static AI provider suggestions
		aiProviders := []string{"openai", "claude", "gemini", "terminal"}
		for _, provider := range aiProviders {
			aiCmd := "ai " + provider
			if matchesPrefix(aiCmd, lowerPrefix) {
				suggestions = append(suggestions, CommandSuggestion{
					Command:     "/" + aiCmd,
					Description: fmt.Sprintf("Start AI Coder with %s", provider),
					Category:    "AI",
				})
			}
		}
	}

	// Proxy commands
	proxyCommands := []string{"proxy", "proxy on", "proxy off", "proxy toggle"}
	for _, cmd := range proxyCommands {
		if matchesPrefix(cmd, lowerPrefix) {
			desc := "Manage proxy settings"
			if strings.Contains(cmd, "on") {
				desc = "Enable proxy"
			} else if strings.Contains(cmd, "off") {
				desc = "Disable proxy"
			} else if strings.Contains(cmd, "toggle") {
				desc = "Toggle proxy state"
			}
			suggestions = append(suggestions, CommandSuggestion{
				Command:     "/" + cmd,
				Description: desc,
				Category:    "Settings",
			})
		}
	}

	return suggestions
}

// matchesPrefix checks if text starts with prefix (case-insensitive)
func matchesPrefix(text, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(text), strings.ToLower(prefix))
}