package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// formatBytes formats byte counts into human-readable format (simple version)
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

// formatSize formats byte counts into human-readable format (full version)
func formatSize(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// renderExitScreen renders the exit screen with Brummer bee logo
func renderExitScreen() string {
	bee := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Render(`
    ╭─╮
   ╱   ╲
  ╱ ● ● ╲    🐝 Thanks for using Brummer!
 ╱   ◡   ╲   
╱  ╲   ╱  ╲   Happy scripting! 
╲   ╲ ╱   ╱  
 ╲   ╱   ╱
  ╲ ─── ╱
   ╲___╱

`)
	return bee
}

// copyToClipboard copies text to the system clipboard across platforms
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if exec.Command("which", "xclip").Run() == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if exec.Command("which", "xsel").Run() == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getTerminalSize returns the terminal width and height
// This uses golang.org/x/term to get the size directly from the terminal
func getTerminalSize() (width, height int, err error) {
	// Try stdout first
	fd := int(os.Stdout.Fd())
	width, height, err = term.GetSize(fd)
	if err == nil {
		return width, height, nil
	}
	stdoutErr := fmt.Errorf("stdout: %w", err)

	// Fallback to stderr
	fd = int(os.Stderr.Fd())
	width, height, err = term.GetSize(fd)
	if err == nil {
		return width, height, nil
	}
	stderrErr := fmt.Errorf("stderr: %w", err)

	// Fallback to stdin
	fd = int(os.Stdin.Fd())
	width, height, err = term.GetSize(fd)
	if err != nil {
		// Return error with context about all attempts
		return 0, 0, fmt.Errorf("failed to get terminal size from any file descriptor: %v, %v, stdin: %w",
			stdoutErr, stderrErr, err)
	}
	return width, height, nil
}
