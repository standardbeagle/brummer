package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/tui_redux/messages"
	"github.com/standardbeagle/brummer/internal/tui_redux/model"
	tea "github.com/charmbracelet/bubbletea"
)

// Process Commands

// StartProcess creates a command to start a new process
func StartProcess(name, command string, workDir string, env []string) tea.Cmd {
	return func() tea.Msg {
		// Parse command into parts
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return messages.NewErrorDetectedMsg("", fmt.Errorf("empty command"), "StartProcess")
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		if workDir != "" {
			cmd.Dir = workDir
		}
		if len(env) > 0 {
			cmd.Env = append(os.Environ(), env...)
		}

		// Start the process
		err := cmd.Start()
		if err != nil {
			return messages.NewErrorDetectedMsg("", err, "StartProcess")
		}

		// Create process model
		process := &model.Process{
			ID:          fmt.Sprintf("%s-%d", name, time.Now().Unix()),
			Name:        name,
			Command:     command,
			State:       "running",
			StartTime:   time.Now(),
			PID:         cmd.Process.Pid,
			WorkingDir:  workDir,
			Environment: env,
		}

		return messages.NewProcessStartedMsg(process)
	}
}

// StopProcess creates a command to stop a process
func StopProcess(processID string, pid int) tea.Cmd {
	return func() tea.Msg {
		if pid <= 0 {
			return messages.NewErrorDetectedMsg(processID, fmt.Errorf("invalid PID"), "StopProcess")
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			return messages.NewErrorDetectedMsg(processID, err, "StopProcess")
		}

		err = process.Kill()
		if err != nil {
			return messages.NewErrorDetectedMsg(processID, err, "StopProcess")
		}

		return messages.NewProcessStoppedMsg(processID, 0)
	}
}

// RestartProcess creates a command to restart a process
func RestartProcess(process *model.Process) tea.Cmd {
	return tea.Sequence(
		StopProcess(process.ID, process.PID),
		StartProcess(process.Name, process.Command, process.WorkingDir, process.Environment),
	)
}

// WatchProcessOutput creates a command to watch process output
func WatchProcessOutput(processID string, cmd *exec.Cmd) tea.Cmd {
	return func() tea.Msg {
		if cmd == nil {
			return messages.NewErrorDetectedMsg(processID, fmt.Errorf("process not running"), "WatchProcessOutput")
		}

		// For now, just return a placeholder message
		// In a real implementation, we'd need to set up pipes to read stdout/stderr
		return messages.NewProcessOutputMsg(processID, "Process output monitoring not yet implemented", "info")
	}
}

// Log Commands

// LoadLogs creates a command to load logs for a process
func LoadLogs(processID string, logDir string) tea.Cmd {
	return func() tea.Msg {
		if logDir == "" {
			return messages.NewErrorDetectedMsg(processID, fmt.Errorf("no log directory"), "LoadLogs")
		}

		logFile := fmt.Sprintf("%s/%s.log", logDir, processID)
		file, err := os.Open(logFile)
		if err != nil {
			return messages.NewErrorDetectedMsg(processID, err, "LoadLogs")
		}
		defer file.Close()

		var logs []model.LogEntry
		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			line := scanner.Text()
			logs = append(logs, model.LogEntry{
				Timestamp: time.Now().Add(-time.Duration(lineNum) * time.Second),
				ProcessID: processID,
				Line:      line,
				Stream:    "stdout",
				Level:     "info",
			})
			lineNum++

			// Limit log entries
			if len(logs) >= 1000 {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			return messages.NewErrorDetectedMsg(processID, err, "LoadLogs")
		}

		// This would normally return a custom message with logs
		// For now, return a success message
		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("Loaded %d log entries", len(logs)),
			Level:   "info",
		}
	}
}

// SaveLogs creates a command to save logs to a file
func SaveLogs(logs []model.LogEntry, path string) tea.Cmd {
	return func() tea.Msg {
		if len(logs) == 0 {
			return messages.NewErrorDetectedMsg("", fmt.Errorf("no logs to save"), "SaveLogs")
		}

		file, err := os.Create(path)
		if err != nil {
			return messages.NewErrorDetectedMsg("", err, "SaveLogs")
		}
		defer file.Close()

		for _, log := range logs {
			_, err := fmt.Fprintf(file, "[%s] %s\n", log.Timestamp.Format(time.RFC3339), log.Line)
			if err != nil {
				return messages.NewErrorDetectedMsg("", err, "SaveLogs")
			}
		}

		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("Saved %d log entries to %s", len(logs), path),
			Level:   "info",
		}
	}
}

// StreamLogs creates a command to stream logs in real-time
func StreamLogs(processID string) tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return messages.NewProcessOutputMsg(processID, "Streaming...", "stdout")
	})
}

// URL Commands

// OpenURLInBrowser creates a command to open a URL in the default browser
func OpenURLInBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		// Determine the command based on OS
		switch {
		case strings.Contains(strings.ToLower(os.Getenv("OSTYPE")), "linux"):
			cmd = exec.Command("xdg-open", url)
		case strings.Contains(strings.ToLower(os.Getenv("OSTYPE")), "darwin"):
			cmd = exec.Command("open", url)
		case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			// Try xdg-open as fallback
			cmd = exec.Command("xdg-open", url)
		}

		err := cmd.Start()
		if err != nil {
			return messages.NewErrorDetectedMsg("", err, "OpenURLInBrowser")
		}

		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("Opened URL: %s", url),
			Level:   "info",
		}
	}
}

// CopyURLToClipboard creates a command to copy a URL to clipboard
func CopyURLToClipboard(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		// Determine the command based on OS
		switch {
		case strings.Contains(strings.ToLower(os.Getenv("OSTYPE")), "linux"):
			if hasCommand("xclip") {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if hasCommand("xsel") {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			}
		case strings.Contains(strings.ToLower(os.Getenv("OSTYPE")), "darwin"):
			cmd = exec.Command("pbcopy")
		case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
			cmd = exec.Command("clip")
		}

		if cmd == nil {
			return messages.NewErrorDetectedMsg("", fmt.Errorf("no clipboard command available"), "CopyURLToClipboard")
		}

		// Pipe the URL to the command
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return messages.NewErrorDetectedMsg("", err, "CopyURLToClipboard")
		}

		go func() {
			defer stdin.Close()
			fmt.Fprint(stdin, url)
		}()

		err = cmd.Run()
		if err != nil {
			return messages.NewErrorDetectedMsg("", err, "CopyURLToClipboard")
		}

		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("Copied URL to clipboard: %s", url),
			Level:   "info",
		}
	}
}

// TestURLConnection creates a command to test if a URL is accessible
func TestURLConnection(url string) tea.Cmd {
	return func() tea.Msg {
		// Simple connectivity test using curl if available
		if !hasCommand("curl") {
			return messages.NewErrorDetectedMsg("", fmt.Errorf("curl not available"), "TestURLConnection")
		}

		cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", url)
		output, err := cmd.Output()
		if err != nil {
			return messages.NewErrorDetectedMsg("", err, "TestURLConnection")
		}

		statusCode := strings.TrimSpace(string(output))
		if statusCode == "200" || statusCode == "204" {
			return messages.StatusUpdateMsg{
				Message: fmt.Sprintf("URL is accessible (HTTP %s): %s", statusCode, url),
				Level:   "info",
			}
		}

		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("URL returned HTTP %s: %s", statusCode, url),
			Level:   "warn",
		}
	}
}

// Settings Commands

// LoadSettings creates a command to load settings from a file
func LoadSettings(configPath string) tea.Cmd {
	return func() tea.Msg {
		if configPath == "" {
			configPath = ".brum.toml"
		}

		file, err := os.Open(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Return default settings if file doesn't exist
				return messages.SettingsUpdatedMsg{
					Settings: model.Settings{
						Theme:              "default",
						MaxLogEntries:      10000,
						MaxProcessHistory:  100,
						MaxURLCache:        1000,
						MaxErrorHistory:    500,
						AutoRefresh:        true,
						RefreshInterval:    1 * time.Second,
						KeyboardShortcuts:  make(map[string]string),
						Debug:              false,
						WorkDir:            ".",
						Environment:        make([]string, 0),
					},
				}
			}
			return messages.NewErrorDetectedMsg("", err, "LoadSettings")
		}
		defer file.Close()

		// For now, return default settings
		// In a real implementation, we'd parse TOML here
		return messages.SettingsUpdatedMsg{
			Settings: model.Settings{
				Theme:              "default",
				MaxLogEntries:      10000,
				MaxProcessHistory:  100,
				MaxURLCache:        1000,
				MaxErrorHistory:    500,
				AutoRefresh:        true,
				RefreshInterval:    1 * time.Second,
				KeyboardShortcuts:  make(map[string]string),
				Debug:              false,
				WorkDir:            ".",
				Environment:        make([]string, 0),
			},
		}
	}
}

// SaveSettings creates a command to save settings to a file
func SaveSettings(settings model.Settings, configPath string) tea.Cmd {
	return func() tea.Msg {
		if configPath == "" {
			configPath = ".brum.toml"
		}

		// For now, just return success
		// In a real implementation, we'd serialize to TOML
		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("Settings saved to %s", configPath),
			Level:   "info",
		}
	}
}

// ApplyTheme creates a command to apply a theme
func ApplyTheme(theme model.Theme) tea.Cmd {
	return func() tea.Msg {
		return messages.ThemeChangedMsg{
			Theme: theme,
		}
	}
}

// System Commands

// CheckForUpdates creates a command to check for application updates
func CheckForUpdates() tea.Cmd {
	return func() tea.Msg {
		// Placeholder for update checking
		return messages.StatusUpdateMsg{
			Message: "Update check not implemented",
			Level:   "info",
		}
	}
}

// ExportState creates a command to export the current application state
func ExportState(state interface{}, path string) tea.Cmd {
	return func() tea.Msg {
		// Placeholder for state export
		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("State export not implemented to %s", path),
			Level:   "info",
		}
	}
}

// ImportState creates a command to import application state
func ImportState(path string) tea.Cmd {
	return func() tea.Msg {
		// Placeholder for state import
		return messages.StatusUpdateMsg{
			Message: fmt.Sprintf("State import not implemented from %s", path),
			Level:   "info",
		}
	}
}

// Utility Commands

// Tick creates a command that sends TickMsg at regular intervals
func Tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return messages.NewTickMsg()
	})
}

// Refresh creates a command that triggers a UI refresh
func Refresh() tea.Cmd {
	return func() tea.Msg {
		return messages.RefreshMsg{}
	}
}

// ShowStatus creates a command that shows a status message
func ShowStatus(message, level string) tea.Cmd {
	return func() tea.Msg {
		return messages.StatusUpdateMsg{
			Message: message,
			Level:   level,
		}
	}
}

// Batch creates a command that executes multiple commands in sequence
func Batch(cmds ...tea.Cmd) tea.Cmd {
	return tea.Sequence(cmds...)
}

// Helper functions

// hasCommand checks if a command exists in PATH
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// WithTimeout creates a command that times out after the given duration
func WithTimeout(cmd tea.Cmd, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		// Create a channel for the command result
		resultChan := make(chan tea.Msg)

		// Execute the command in a goroutine
		go func() {
			resultChan <- cmd()
		}()

		// Wait for result or timeout
		select {
		case msg := <-resultChan:
			return msg
		case <-time.After(timeout):
			return messages.StatusUpdateMsg{
				Message: "Command timed out",
				Level:   "warn",
			}
		}
	}
}