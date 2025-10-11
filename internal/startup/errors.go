package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
)

// ErrorType categorizes different types of startup errors
type ErrorType int

const (
	ErrorTypeCritical ErrorType = iota
	ErrorTypeWarning
	ErrorTypeConfiguration
	ErrorTypeNetwork
	ErrorTypePermission
	ErrorTypeDependency
)

// StartupError represents a structured startup error with user-friendly messaging
type StartupError struct {
	Type        ErrorType
	Title       string
	Description string
	Cause       error
	Suggestions []string
	Context     map[string]interface{}
}

// ErrorReporter provides user-friendly error reporting
type ErrorReporter struct {
	output *os.File
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter() *ErrorReporter {
	return &ErrorReporter{
		output: os.Stderr,
	}
}

// ReportError displays a structured error to the user and optionally exits
func (r *ErrorReporter) ReportError(err *StartupError, exit bool) {
	// Print the error header
	r.printHeader(err.Type)

	// Print the main error message
	color.New(color.FgRed).Fprintf(r.output, "❌ %s\n", err.Title)

	// Print description
	if err.Description != "" {
		color.New(color.FgWhite).Fprintf(r.output, "   %s\n", err.Description)
	}

	// Print cause if available
	if err.Cause != nil {
		color.New(color.FgHiBlack).Fprintf(r.output, "   Details: %v\n", err.Cause)
	}

	// Print suggestions
	if len(err.Suggestions) > 0 {
		color.New(color.FgCyan).Fprintf(r.output, "\n💡 Suggestions:\n")
		for i, suggestion := range err.Suggestions {
			color.New(color.FgCyan).Fprintf(r.output, "   %d. %s\n", i+1, suggestion)
		}
	}

	// Print context if available
	if len(err.Context) > 0 {
		color.New(color.FgYellow).Fprintf(r.output, "\n📍 Context:\n")
		for key, value := range err.Context {
			color.New(color.FgYellow).Fprintf(r.output, "   %s: %v\n", key, value)
		}
	}

	// Add spacing
	fmt.Fprintln(r.output)

	if exit {
		r.printFooter(err.Type)
		os.Exit(1)
	}
}

// printHeader prints error type header
func (r *ErrorReporter) printHeader(errorType ErrorType) {
	switch errorType {
	case ErrorTypeCritical:
		color.New(color.BgRed, color.FgWhite).Fprintf(r.output, "\n🚨 CRITICAL ERROR 🚨\n")
	case ErrorTypeWarning:
		color.New(color.BgYellow, color.FgBlack).Fprintf(r.output, "\n⚠️  WARNING ⚠️\n")
	case ErrorTypeConfiguration:
		color.New(color.BgBlue, color.FgWhite).Fprintf(r.output, "\n⚙️  CONFIGURATION ⚙️\n")
	case ErrorTypeNetwork:
		color.New(color.BgMagenta, color.FgWhite).Fprintf(r.output, "\n🌐 NETWORK ERROR 🌐\n")
	case ErrorTypePermission:
		color.New(color.BgRed, color.FgWhite).Fprintf(r.output, "\n🔒 PERMISSION ERROR 🔒\n")
	case ErrorTypeDependency:
		color.New(color.BgCyan, color.FgBlack).Fprintf(r.output, "\n📦 DEPENDENCY ERROR 📦\n")
	}
	fmt.Fprintln(r.output)
}

// printFooter prints error type footer
func (r *ErrorReporter) printFooter(errorType ErrorType) {
	fmt.Fprintln(r.output)
	switch errorType {
	case ErrorTypeCritical:
		color.New(color.BgRed, color.FgWhite).Fprintf(r.output, "💀 Brummer cannot continue")
	case ErrorTypeWarning:
		color.New(color.BgYellow, color.FgBlack).Fprintf(r.output, "⚠️  Brummer will continue with reduced functionality")
	case ErrorTypeConfiguration:
		color.New(color.BgBlue, color.FgWhite).Fprintf(r.output, "⚙️  Check your configuration file")
	case ErrorTypeNetwork:
		color.New(color.BgMagenta, color.FgWhite).Fprintf(r.output, "🌐  Check your network connection")
	case ErrorTypePermission:
		color.New(color.BgRed, color.FgWhite).Fprintf(r.output, "🔒  Check file permissions")
	case ErrorTypeDependency:
		color.New(color.BgCyan, color.FgBlack).Fprintf(r.output, "📦  Install missing dependencies")
	}
	fmt.Fprintln(r.output)
}

// Create critical startup errors
func (r *ErrorReporter) CriticalWorkingDir(err error, workDir string) *StartupError {
	return &StartupError{
		Type:        ErrorTypeCritical,
		Title:       "Cannot access working directory",
		Description: fmt.Sprintf("Unable to access '%s'", workDir),
		Cause:       err,
		Suggestions: []string{
			"Check if the directory exists and is accessible",
			"Ensure you have read permissions for the directory",
			"Try running from a different directory",
			"Check if the directory path is correct",
		},
		Context: map[string]interface{}{
			"working_dir": workDir,
			"absolute_path": func() string {
				if abs, err := filepath.Abs(workDir); err == nil {
					return abs
				}
				return workDir
			}(),
			"user_permissions": func() string {
				if info, err := os.Stat(workDir); err == nil {
					return fmt.Sprintf("%v", info.Mode().Perm())
				}
				return "unknown"
			}(),
		},
	}
}

func (r *ErrorReporter) CriticalProcessManager(err error) *StartupError {
	return &StartupError{
		Type:        ErrorTypeCritical,
		Title:       "Failed to initialize process manager",
		Description: "Cannot start the background process management system",
		Cause:       err,
		Suggestions: []string{
			"Check if another instance of Brummer is running",
			"Ensure system has sufficient resources (memory, file descriptors)",
			"Try restarting your system",
			"Check for conflicting processes on the working directory",
		},
		Context: map[string]interface{}{
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
			"pid":  os.Getpid(),
		},
	}
}

func (r *ErrorReporter) CriticalMCPServer(err error, port int) *StartupError {
	return &StartupError{
		Type:        ErrorTypeCritical,
		Title:       "Failed to start MCP server",
		Description: fmt.Sprintf("Cannot start Model Context Protocol server on port %d", port),
		Cause:       err,
		Suggestions: []string{
			"Check if port " + fmt.Sprintf("%d", port) + " is already in use",
			"Try running with a different port: brum -p " + fmt.Sprintf("%d", port+1),
			"Disable MCP server with: brum --no-mcp",
			"Check firewall settings blocking the port",
			"Ensure no other MCP clients are using the same port",
		},
		Context: map[string]interface{}{
			"port":    port,
			"os":      runtime.GOOS,
			"timeout": "5 seconds",
		},
	}
}

func (r *ErrorReporter) CriticalTUI(err error) *StartupError {
	return &StartupError{
		Type:        ErrorTypeCritical,
		Title:       "Failed to start TUI",
		Description: "Cannot start the terminal user interface",
		Cause:       err,
		Suggestions: []string{
			"Check if your terminal supports TTY mode",
			"Try running in headless mode: brum --no-tui",
			"Ensure terminal is not redirected ( piping )",
			"Check for terminal emulator compatibility issues",
			"Try running in a different terminal or SSH session",
		},
		Context: map[string]interface{}{
			"terminal_tty": r.checkTTY(),
			"os":           runtime.GOOS,
			"terminal":     os.Getenv("TERM"),
		},
	}
}

func (r *ErrorReporter) WarningConfigLoad(err error) *StartupError {
	return &StartupError{
		Type:        ErrorTypeWarning,
		Title:       "Configuration loading failed",
		Description: "Cannot load configuration file, using defaults",
		Cause:       err,
		Suggestions: []string{
			"Brummer will use default configuration settings",
			"Check your .brum.toml file for syntax errors",
			"Create a configuration file: brum --settings > .brum.toml",
			"Ensure the configuration file is valid TOML format",
		},
		Context: map[string]interface{}{
			"config_files": []string{".brum.toml", "~/.brum.toml"},
		},
	}
}

func (r *ErrorReporter) WarningProxyServer(err error) *StartupError {
	return &StartupError{
		Type:        ErrorTypeWarning,
		Title:       "Proxy server failed to start",
		Description: "HTTP proxy functionality will be disabled",
		Cause:       err,
		Suggestions: []string{
			"Brummer will continue without proxy functionality",
			"URL detection will be disabled",
			"Manual proxy setup may be required",
			"Try different proxy port if there's a conflict",
		},
		Context: map[string]interface{}{
			"feature": "HTTP proxy server",
			"impact":  "reduced functionality",
		},
	}
}

func (r *ErrorReporter) WarningScriptStart(err error, scriptName string) *StartupError {
	return &StartupError{
		Type:        ErrorTypeWarning,
		Title:       "Script failed to start",
		Description: fmt.Sprintf("Cannot start script '%s'", scriptName),
		Cause:       err,
		Suggestions: []string{
			"Check if the script exists in package.json",
			"Ensure all dependencies are installed",
			"Try running the script manually to see detailed error",
			"Check the script syntax and file permissions",
		},
		Context: map[string]interface{}{
			"script":  scriptName,
			"command": fmt.Sprintf("%s %s", scriptName, err.Error()),
		},
	}
}

func (r *ErrorReporter) ConfigurationMissing(field string, defaultValue interface{}) *StartupError {
	return &StartupError{
		Type:        ErrorTypeConfiguration,
		Title:       "Configuration field missing",
		Description: fmt.Sprintf("Configuration field '%s' not found, using default value", field),
		Suggestions: []string{
			fmt.Sprintf("Using default value: %v", defaultValue),
			"Add the field to your .brum.toml configuration file",
			"Run 'brum --settings' to see current configuration",
			"See documentation for available configuration options",
		},
		Context: map[string]interface{}{
			"field":         field,
			"default_value": defaultValue,
			"config_file":   ".brum.toml",
		},
	}
}

// checkTTY checks if the current environment supports TTY
func (r *ErrorReporter) checkTTY() string {
	stdinTTY := false
	stdoutTTY := false

	// Check stdin
	if stat, err := os.Stdin.Stat(); err == nil {
		stdinTTY = (stat.Mode() & os.ModeCharDevice) != 0
	}

	// Check stdout
	if stat, err := os.Stdout.Stat(); err == nil {
		stdoutTTY = (stat.Mode() & os.ModeCharDevice) != 0
	}

	if stdinTTY && stdoutTTY {
		return "Yes (stdin/stdout are terminals)"
	} else if stdinTTY {
		return "Partial (stdin is terminal, stdout is redirected)"
	} else if stdoutTTY {
		return "Partial (stdout is terminal, stdin is redirected)"
	}
	return "No (stdin or stdout is redirected)"
}
