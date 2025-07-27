package logs

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrorContext represents a complete error with all its context
type ErrorContext struct {
	ID          string
	ProcessID   string
	ProcessName string
	Timestamp   time.Time
	Type        string   // e.g., "MongoError", "SyntaxError", "RuntimeError"
	Message     string   // Main error message
	Stack       []string // Stack trace lines
	Context     []string // Additional context lines
	Severity    string   // critical, error, warning
	Language    string   // js, go, python, java, etc.
	Raw         []string // All raw log lines that make up this error
}

// ErrorParser handles sophisticated multi-line error parsing
type ErrorParser struct {
	// Patterns for detecting error starts
	errorStartPatterns map[string]*regexp.Regexp

	// Patterns for stack trace detection
	stackPatterns map[string]*regexp.Regexp

	// Patterns for error continuation
	continuationPatterns []*regexp.Regexp

	// Active error contexts being built
	activeErrors map[string]*ErrorContext

	// Completed errors
	errors []ErrorContext

	// Maximum lines to look ahead for error context
	maxContextLines int
}

func NewErrorParser() *ErrorParser {
	return &ErrorParser{
		errorStartPatterns: map[string]*regexp.Regexp{
			// JavaScript/Node.js errors
			"js_unhandled":     regexp.MustCompile(`^\s*⨯\s*unhandled(?:Rejection)?:?\s*\[?(\w+(?:Error|Exception))\]?:?\s*(.+)`),
			"js_error_bracket": regexp.MustCompile(`^\[?(\w+(?:Error|Exception))\]?:\s*(.+)`),
			"js_error_simple":  regexp.MustCompile(`(?i)^(?:error:|fatal:|uncaught exception:)\s*(.+)`),
			"js_error_type":    regexp.MustCompile(`^(\w+Error):\s*(.+)`),
			"js_fetch_error":   regexp.MustCompile(`^(FetchError):\s*(.+)`),
			"js_rejection":     regexp.MustCompile(`^\s*(?:UnhandledPromiseRejectionWarning:|PromiseRejectionHandledWarning:)\s*(.+)`),

			// TypeScript/Build errors (higher priority)
			"ts_error":       regexp.MustCompile(`^(?:ERROR|Error)\s+in\s+(.+)`),
			"ts_specific":    regexp.MustCompile(`(TS\d+):\s*(.+)`),
			"ts_file_error":  regexp.MustCompile(`^(.+\.tsx?)\((\d+,\d+)\):\s+error\s+(TS\d+):\s*(.+)`),
			"build_error":    regexp.MustCompile(`^(?:Build Error|Compilation Error|ERROR):\s*(.+)`),
			"failed_compile": regexp.MustCompile(`(?i)failed\s+to\s+compile\.?`),
			"error_exit":     regexp.MustCompile(`^ERROR:\s*"([^"]+)"\s+exited\s+with\s+(\d+)\.?`),

			// React/JSX specific errors
			"react_jsx_key":          regexp.MustCompile(`(?i)missing\s+"key"\s+prop\s+for\s+element`),
			"react_jsx_adjacent":     regexp.MustCompile(`(?i)adjacent\s+jsx\s+elements\s+must\s+be\s+wrapped`),
			"react_hook_conditional": regexp.MustCompile(`(?i)react\s+hook\s+.+\s+is\s+called\s+conditionally`),
			"react_hook_dependency":  regexp.MustCompile(`(?i)react\s+hook\s+has\s+a\s+missing\s+dependency`),
			"react_invalid_child":    regexp.MustCompile(`(?i)(?:objects|functions)\s+are\s+not\s+valid\s+as\s+react\s+child`),

			// Go errors
			"go_panic": regexp.MustCompile(`^panic:\s*(.+)`),
			"go_error": regexp.MustCompile(`^(?:error:|Error:)\s*(.+)`),

			// Python errors
			"python_error":     regexp.MustCompile(`^(\w+(?:Error|Exception)):\s*(.+)`),
			"python_traceback": regexp.MustCompile(`^Traceback\s*\(most recent call last\):`),

			// Java errors
			"java_exception": regexp.MustCompile(`^(?:Exception in thread|Caused by:)\s*(.+)`),
			"java_error":     regexp.MustCompile(`^(\w+(?:Exception|Error)):\s*(.+)`),

			// Rust errors
			"rust_error": regexp.MustCompile(`^error(?:\[E\d+\])?:\s*(.+)`),

			// Vue specific errors
			"vue_template_error":    regexp.MustCompile(`(?i)template\s+compilation\s+error`),
			"vue_component_error":   regexp.MustCompile(`(?i)vue\s+component\s+error`),
			"vue_composition_error": regexp.MustCompile(`(?i)composition\s+api\s+error`),

			// Next.js specific errors
			"nextjs_build_error": regexp.MustCompile(`(?i)next\.js.*build.*error`),
			"nextjs_lint_error":  regexp.MustCompile(`^\s*\d+:\d+\s+Error:\s*(.+)`),

			// ESLint/Linting errors
			"eslint_error":    regexp.MustCompile(`^\s*\d+:\d+\s+error\s+(.+)`),
			"eslint_warning":  regexp.MustCompile(`^\s*\d+:\d+\s+warning\s+(.+)`),
			"lint_line_error": regexp.MustCompile(`^\s*\d+:\d+\s+Error:\s*(.+)`),

			// Database errors
			"mongo_error": regexp.MustCompile(`^(MongoError|MongoNetworkError|MongoTimeoutError):\s*(.+)`),

			// Generic errors
			"generic_failed":   regexp.MustCompile(`(?i)^.*(failed to|cannot|unable to|could not)\s+(.+)`),
			"generic_error":    regexp.MustCompile(`(?i)^\s*(?:⚠|❌|✖|ERROR|FAIL)\s+(.+)`),
			"generic_argument": regexp.MustCompile(`^Argument of type\s+.+\s+is not assignable to parameter`),
		},

		stackPatterns: map[string]*regexp.Regexp{
			// JavaScript stack traces
			"js_stack":          regexp.MustCompile(`^\s*at\s+.+\s*\(?.*:\d+:\d+\)?`),
			"js_stack_brackets": regexp.MustCompile(`^\s*\[.+\]\s+.+:\d+:\d+`),
			"js_stack_simple":   regexp.MustCompile(`^\s*at\s+.+\(.+:\d+:\d+\)`),
			"js_stack_file":     regexp.MustCompile(`^\s*at\s+.+\s+\(.+\.js:\d+:\d+\)`),
			"js_stack_webpack":  regexp.MustCompile(`^\s*at\s+webpack:///`),

			// Go stack traces
			"go_stack":     regexp.MustCompile(`^\s*.*\.go:\d+\s+.+`),
			"go_goroutine": regexp.MustCompile(`^goroutine\s+\d+`),

			// Python stack traces
			"python_stack": regexp.MustCompile(`^\s*File\s+"[^"]+",\s+line\s+\d+`),

			// Java stack traces
			"java_stack": regexp.MustCompile(`^\s*at\s+[\w\.$]+\(.+\)`),

			// Generic stack patterns
			"generic_stack": regexp.MustCompile(`^\s*#\d+\s+.+`),
		},

		continuationPatterns: []*regexp.Regexp{
			// Indented lines (common for multi-line errors)
			regexp.MustCompile(`^\s{2,}.+`),
			// Lines starting with special characters
			regexp.MustCompile(`^\s*[│├└─|]\s*.+`),
			// JSON-like object notation
			regexp.MustCompile(`^\s*\{`),
			regexp.MustCompile(`^\s*\}`),
			regexp.MustCompile(`^\s*\[`),
			regexp.MustCompile(`^\s*\]`),
			// Property notation
			regexp.MustCompile(`^\s*\w+:\s*.+`),
			// Numbered lists
			regexp.MustCompile(`^\s*\d+\.\s*.+`),
		},

		activeErrors:    make(map[string]*ErrorContext),
		errors:          make([]ErrorContext, 0),
		maxContextLines: 50,
	}
}

// ProcessLine processes a log line and updates error contexts
func (p *ErrorParser) ProcessLine(processID, processName, content string, timestamp time.Time) *ErrorContext {
	// Strip common log prefixes like timestamps [HH:MM:SS] and process names
	cleanContent := p.stripLogPrefixes(content)

	// Check if this line starts a new error
	if errorType, errorInfo := p.detectErrorStart(cleanContent); errorType != "" {
		// Create new error context
		errorCtx := &ErrorContext{
			ID:          fmt.Sprintf("%s-%d", processID, timestamp.UnixNano()),
			ProcessID:   processID,
			ProcessName: processName,
			Timestamp:   timestamp,
			Type:        errorInfo["type"],
			Message:     errorInfo["message"],
			Severity:    p.determineSeverity(content),
			Language:    p.detectLanguage(content),
			Raw:         []string{content},
		}

		// For certain error types, return immediately as they are single-line
		if p.isSingleLineError(errorType) {
			p.finalizeError(errorCtx)
			return errorCtx
		}

		// Store as active error for multi-line processing
		p.activeErrors[processID] = errorCtx

		return nil // Don't return yet, we're building the context
	}

	// Check if this line continues an active error
	if activeError, exists := p.activeErrors[processID]; exists {
		if p.isErrorContinuation(content, activeError) {
			activeError.Raw = append(activeError.Raw, content)

			// Check if it's a stack trace line
			if p.isStackTraceLine(content) {
				activeError.Stack = append(activeError.Stack, content)
			} else {
				activeError.Context = append(activeError.Context, content)
			}

			// Check if we've collected enough context
			if len(activeError.Raw) >= p.maxContextLines || p.isErrorEnd(content) {
				// Complete the error
				p.finalizeError(activeError)
				delete(p.activeErrors, processID)
				return activeError
			}

			return nil // Still building
		} else {
			// This line doesn't continue the error, finalize it
			p.finalizeError(activeError)
			delete(p.activeErrors, processID)
			return activeError
		}
	}

	// Check if this is a standalone error line
	if p.isStandaloneError(cleanContent) {
		return &ErrorContext{
			ID:          fmt.Sprintf("%s-%d", processID, timestamp.UnixNano()),
			ProcessID:   processID,
			ProcessName: processName,
			Timestamp:   timestamp,
			Type:        "Error",
			Message:     cleanContent,
			Severity:    p.determineSeverity(cleanContent),
			Language:    p.detectLanguage(cleanContent),
			Raw:         []string{content},
		}
	}

	return nil
}

// stripLogPrefixes removes common log line prefixes
func (p *ErrorParser) stripLogPrefixes(content string) string {
	// Remove timestamp patterns like [12:52:32], (12:52:32), 12:52:32
	timestampPatterns := []string{
		`^\[\d{1,2}:\d{2}:\d{2}\]\s*`,
		`^\(\d{1,2}:\d{2}:\d{2}\)\s*`,
		`^\d{1,2}:\d{2}:\d{2}\s+`,
		`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s*`,
	}

	cleaned := content
	for _, pattern := range timestampPatterns {
		re := regexp.MustCompile(pattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Remove process name patterns like [dev], (dev), dev: but preserve TypeScript errors
	processPatterns := []string{
		`^\[[\w-]+\]:\s*`,
		`^\([\w-]+\):\s*`,
	}

	// Only apply word: pattern if it's not a TypeScript error
	if !regexp.MustCompile(`^TS\d+:`).MatchString(cleaned) {
		processPatterns = append(processPatterns, `^[\w-]+:\s+`)
	}

	for _, pattern := range processPatterns {
		re := regexp.MustCompile(pattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	return cleaned
}

func (p *ErrorParser) detectErrorStart(content string) (string, map[string]string) {
	for patternName, pattern := range p.errorStartPatterns {
		if matches := pattern.FindStringSubmatch(content); matches != nil {
			info := make(map[string]string)

			switch {
			case strings.HasPrefix(patternName, "js_"):
				if patternName == "js_rejection" {
					// Handle promise rejection warnings
					if strings.Contains(content, "UnhandledPromiseRejectionWarning:") {
						info["type"] = "UnhandledPromiseRejectionWarning"
					} else {
						info["type"] = "PromiseRejectionHandledWarning"
					}
					if len(matches) > 1 {
						info["message"] = strings.TrimSpace(matches[1])
					}
				} else if (patternName == "js_error_type" || patternName == "js_fetch_error") && len(matches) > 1 {
					// Handle specific error types like TypeError, ReferenceError, FetchError, etc.
					errorType := matches[1]
					// Map FetchError to NetworkError for better categorization
					if errorType == "FetchError" {
						info["type"] = "NetworkError"
					} else {
						info["type"] = errorType
					}
					if len(matches) > 2 {
						info["message"] = strings.TrimSpace(matches[2])
					}
				} else if len(matches) > 1 {
					info["type"] = matches[1]
					if len(matches) > 2 {
						info["message"] = strings.TrimSpace(matches[2])
					} else {
						info["message"] = strings.TrimSpace(matches[1])
					}
				} else if strings.Contains(content, "unhandled") {
					info["type"] = "UnhandledRejection"
				}
			case strings.HasPrefix(patternName, "react_"):
				info["type"] = "ReactError"
				if len(matches) > 1 {
					info["message"] = strings.TrimSpace(matches[1])
				} else {
					info["message"] = content
				}
			case strings.HasPrefix(patternName, "ts_"):
				if patternName == "ts_file_error" && len(matches) >= 5 {
					info["type"] = matches[3] // TS error code
					info["message"] = fmt.Sprintf("%s: %s", matches[1], matches[4])
				} else if patternName == "ts_specific" && len(matches) >= 3 {
					info["type"] = matches[1] // TS error code
					info["message"] = matches[2]
				} else {
					info["type"] = "TypeScriptError"
					if len(matches) > 1 {
						info["message"] = matches[1]
					}
				}
			case patternName == "failed_compile":
				info["type"] = "CompilationError"
				info["message"] = content
			case patternName == "error_exit":
				info["type"] = "Error"
				if len(matches) > 1 {
					info["message"] = fmt.Sprintf("\"%s\" exited with %s", matches[1], matches[2])
				} else {
					info["message"] = content
				}
			case patternName == "lint_line_error":
				info["type"] = "LintError"
				if len(matches) > 1 {
					info["message"] = matches[1]
				} else {
					info["message"] = content
				}
			case strings.HasPrefix(patternName, "vue_"):
				info["type"] = "VueError"
				if len(matches) > 1 {
					info["message"] = matches[1]
				} else {
					info["message"] = content
				}
			case strings.HasPrefix(patternName, "nextjs_"):
				info["type"] = "NextJSError"
				if len(matches) > 1 {
					info["message"] = matches[1]
				} else {
					info["message"] = content
				}
			case strings.HasPrefix(patternName, "eslint_"):
				info["type"] = "ESLintError"
				if len(matches) > 1 {
					info["message"] = matches[1]
				} else {
					info["message"] = content
				}
			case strings.HasPrefix(patternName, "go_"):
				info["type"] = "GoError"
				if len(matches) > 1 {
					info["message"] = matches[1]
				}
			case strings.HasPrefix(patternName, "python_"):
				if patternName == "python_traceback" {
					info["type"] = "PythonError"
					info["message"] = "Python traceback"
				} else if len(matches) > 1 {
					info["type"] = matches[1]
					if len(matches) > 2 {
						info["message"] = matches[2]
					}
				}
			case strings.HasPrefix(patternName, "java_"):
				info["type"] = "JavaException"
				if len(matches) > 1 {
					info["message"] = matches[1]
				}
			case strings.HasPrefix(patternName, "rust_"):
				info["type"] = "RustError"
				if len(matches) > 1 {
					info["message"] = matches[1]
				}
			case patternName == "mongo_error" && len(matches) > 1:
				info["type"] = matches[1] // MongoError, MongoNetworkError, etc.
				if len(matches) > 2 {
					info["message"] = matches[2]
				}
			default:
				info["type"] = "Error"
				if len(matches) > 1 {
					info["message"] = matches[1]
				}
			}

			// Set defaults if not set
			if info["type"] == "" {
				info["type"] = "Error"
			}
			if info["message"] == "" {
				info["message"] = content
			}

			return patternName, info
		}
	}

	return "", nil
}

func (p *ErrorParser) isErrorContinuation(content string, activeError *ErrorContext) bool {
	// Empty lines within an error context
	if strings.TrimSpace(content) == "" && len(activeError.Raw) < 10 {
		return true
	}

	// Check continuation patterns
	for _, pattern := range p.continuationPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}

	// Check if it's a stack trace line
	if p.isStackTraceLine(content) {
		return true
	}

	// Language-specific continuations
	switch activeError.Language {
	case "javascript":
		// JS errors often have object notation
		if strings.HasPrefix(strings.TrimSpace(content), "{") ||
			strings.HasPrefix(strings.TrimSpace(content), "}") ||
			strings.HasPrefix(strings.TrimSpace(content), "[") ||
			strings.HasPrefix(strings.TrimSpace(content), "]") {
			return true
		}
	case "python":
		// Python errors have consistent indentation
		if strings.HasPrefix(content, "  ") || strings.HasPrefix(content, "\t") {
			return true
		}
	}

	return false
}

func (p *ErrorParser) isStackTraceLine(content string) bool {
	for _, pattern := range p.stackPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func (p *ErrorParser) isErrorEnd(content string) bool {
	// Common patterns that indicate error end
	trimmed := strings.TrimSpace(content)

	// Multiple closing braces often indicate end of error object
	if trimmed == "}" || trimmed == "}}" || trimmed == "}}}" {
		return true
	}

	// New timestamp patterns often indicate a new log entry
	if regexp.MustCompile(`^\d{1,2}:\d{2}:\d{2}`).MatchString(trimmed) {
		return true
	}

	// Success messages after errors
	if regexp.MustCompile(`(?i)(success|completed|done|finished)`).MatchString(trimmed) {
		return true
	}

	return false
}

func (p *ErrorParser) isStandaloneError(content string) bool {
	lower := strings.ToLower(content)

	// Simple error indicators
	errorKeywords := []string{
		"error:", "error ", "failed:", "failed ",
		"fatal:", "exception:", "panic:",
		"cannot ", "could not ", "unable to ",
	}

	// Also check for simple "Error: " at start of line
	if regexp.MustCompile(`(?i)^Error:\s+`).MatchString(content) {
		return true
	}

	for _, keyword := range errorKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

func (p *ErrorParser) determineSeverity(content string) string {
	lower := strings.ToLower(content)

	if strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "panic") ||
		strings.Contains(lower, "critical") {
		return "critical"
	}

	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "exception") {
		return "error"
	}

	if strings.Contains(lower, "warn") ||
		strings.Contains(lower, "warning") {
		return "warning"
	}

	return "info"
}

func (p *ErrorParser) detectLanguage(content string) string {
	// JavaScript/Node.js indicators (enhanced)
	if strings.Contains(content, "node_modules") ||
		strings.Contains(content, ".js:") ||
		strings.Contains(content, ".jsx:") ||
		strings.Contains(content, ".ts:") ||
		strings.Contains(content, ".tsx:") ||
		strings.Contains(content, "at Module.") ||
		strings.Contains(content, "webpack:///") ||
		strings.Contains(strings.ToLower(content), "react") ||
		strings.Contains(strings.ToLower(content), "vue") ||
		strings.Contains(strings.ToLower(content), "next") ||
		strings.Contains(content, "JSX") ||
		strings.Contains(content, "TypeScript") ||
		strings.Contains(content, "ESLint") ||
		strings.Contains(strings.ToLower(content), "failed to compile") ||
		strings.Contains(content, "UnhandledPromiseRejectionWarning") ||
		strings.Contains(content, "PromiseRejectionHandledWarning") ||
		regexp.MustCompile(`\w+Error:`).MatchString(content) ||
		regexp.MustCompile(`TS\d+:`).MatchString(content) {
		return "javascript"
	}

	// Go indicators
	if strings.Contains(content, ".go:") ||
		strings.Contains(content, "goroutine") ||
		strings.Contains(content, "panic:") {
		return "go"
	}

	// Python indicators
	if strings.Contains(content, ".py:") ||
		strings.Contains(content, "Traceback") ||
		strings.Contains(content, "File \"") {
		return "python"
	}

	// Java indicators
	if strings.Contains(content, ".java:") ||
		strings.Contains(content, "at com.") ||
		strings.Contains(content, "Exception") {
		return "java"
	}

	// Rust indicators
	if strings.Contains(content, ".rs:") ||
		strings.Contains(content, "error[E") {
		return "rust"
	}

	return "unknown"
}

// isSingleLineError determines if an error type is typically single-line
func (p *ErrorParser) isSingleLineError(errorType string) bool {
	singleLineTypes := []string{
		"ts_specific", "failed_compile", "react_jsx_key", "react_jsx_adjacent",
		"react_hook_conditional", "react_hook_dependency", "react_invalid_child",
		"react_compilation_failed", "vue_template_error", "vue_component_error",
		"vue_composition_error", "nextjs_build_error", "nextjs_lint_error",
		"eslint_error", "eslint_warning", "lint_line_error", "ts_error",
		"build_error", "generic_error",
	}

	for _, singleType := range singleLineTypes {
		if errorType == singleType {
			return true
		}
	}
	return false
}

func (p *ErrorParser) finalizeError(errorCtx *ErrorContext) {
	// Clean up the error message
	if errorCtx.Message == "" && len(errorCtx.Raw) > 0 {
		errorCtx.Message = errorCtx.Raw[0]
	}

	// Extract key information based on error type
	if strings.Contains(errorCtx.Type, "MongoError") {
		p.parseMongoError(errorCtx)
	}

	// Store the completed error
	p.errors = append(p.errors, *errorCtx)
}

func (p *ErrorParser) parseMongoError(errorCtx *ErrorContext) {
	// Extract hostname, error code, etc. from MongoDB errors
	for _, line := range errorCtx.Raw {
		if strings.Contains(line, "hostname:") {
			if match := regexp.MustCompile(`hostname:\s*'([^']+)'`).FindStringSubmatch(line); match != nil {
				errorCtx.Message = fmt.Sprintf("%s (hostname: %s)", errorCtx.Message, match[1])
			}
		}
		if strings.Contains(line, "code:") && strings.Contains(line, "ENOTFOUND") {
			errorCtx.Message = strings.ReplaceAll(errorCtx.Message, "getaddrinfo", "DNS lookup failed -")
		}
	}
}

// GetErrors returns all parsed errors
func (p *ErrorParser) GetErrors() []ErrorContext {
	// Include any active errors that haven't been finalized
	for _, activeError := range p.activeErrors {
		p.errors = append(p.errors, *activeError)
	}
	p.activeErrors = make(map[string]*ErrorContext)

	return p.errors
}

// ClearErrors clears the error history
func (p *ErrorParser) ClearErrors() {
	p.errors = make([]ErrorContext, 0)
	p.activeErrors = make(map[string]*ErrorContext)
}
