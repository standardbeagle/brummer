package logs

import (
	"fmt"
	"strings"
	"time"
)

// ConfigurableErrorParser is a new error parser that uses TOML configuration
type ConfigurableErrorParser struct {
	config *ErrorParsingConfig

	// Active error contexts being built
	activeErrors map[string]*ErrorContext

	// Completed errors
	errors []ErrorContext
}

// NewConfigurableErrorParser creates a new configurable error parser
func NewConfigurableErrorParser(configPath string) (*ConfigurableErrorParser, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load error parsing config: %w", err)
	}

	return &ConfigurableErrorParser{
		config:       config,
		activeErrors: make(map[string]*ErrorContext),
		errors:       make([]ErrorContext, 0),
	}, nil
}

// NewDefaultConfigurableErrorParser creates a parser with default configuration
func NewDefaultConfigurableErrorParser() (*ConfigurableErrorParser, error) {
	return NewConfigurableErrorParser("")
}

// ProcessLine processes a log line and updates error contexts
func (p *ConfigurableErrorParser) ProcessLine(processID, processName, content string, timestamp time.Time) *ErrorContext {
	// Strip log prefixes based on configuration
	cleanContent := p.stripLogPrefixes(content)

	// Check if this line starts a new error
	if errorType, errorInfo := p.detectErrorStart(cleanContent); errorType != "" {
		// Extract language from errorType (format: "language.pattern_name")
		language := "unknown"
		if parts := strings.Split(errorType, "."); len(parts) == 2 {
			language = parts[0]
			// Map framework-specific languages to their base language
			switch language {
			case "react", "vue", "nextjs", "eslint":
				language = "javascript"
			// Keep typescript as typescript for TS-specific errors
			}
		}
		
		// Create new error context
		errorCtx := &ErrorContext{
			ID:          fmt.Sprintf("%s-%d", processID, timestamp.UnixNano()),
			ProcessID:   processID,
			ProcessName: processName,
			Timestamp:   timestamp,
			Type:        errorInfo["type"],
			Message:     errorInfo["message"],
			Severity:    p.determineSeverity(content, errorInfo["severity"]),
			Language:    language,
			Raw:         []string{content},
		}
		
		// If language is generic, try to detect from content
		if language == "generic" {
			detectedLang := p.detectLanguage(content)
			if detectedLang != "unknown" {
				errorCtx.Language = detectedLang
			}
		}

		// Check if this is a single-line error based on config
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
			if p.isStackTraceLine(content, activeError.Language) {
				activeError.Stack = append(activeError.Stack, content)
			} else {
				activeError.Context = append(activeError.Context, content)
			}

			// Check if we've collected enough context or reached error end
			maxLines := p.config.Settings.MaxContextLines
			if len(activeError.Raw) >= maxLines || p.isErrorEnd(content) {
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

	// Don't process standalone errors here - they should be caught by pattern detection above

	return nil
}

// stripLogPrefixes removes log prefixes based on configuration
func (p *ConfigurableErrorParser) stripLogPrefixes(content string) string {
	cleaned := content

	// Remove timestamp patterns
	for _, regex := range p.config.LogPrefixes.Timestamp.Regexes() {
		cleaned = regex.ReplaceAllString(cleaned, "")
	}

	// Remove process patterns
	for _, regex := range p.config.LogPrefixes.Process.Regexes() {
		cleaned = regex.ReplaceAllString(cleaned, "")
	}

	// Remove conditional process patterns (with exclusions)
	for _, regex := range p.config.LogPrefixes.ConditionalProcess.Regexes() {
		// Check if any exclude patterns match
		shouldExclude := false
		for _, excludeRegex := range p.config.LogPrefixes.ConditionalProcess.ExcludeRegexes() {
			if excludeRegex.MatchString(cleaned) {
				shouldExclude = true
				break
			}
		}

		if !shouldExclude {
			cleaned = regex.ReplaceAllString(cleaned, "")
		}
	}

	return cleaned
}

// detectErrorStart checks if content matches any configured error patterns
func (p *ConfigurableErrorParser) detectErrorStart(content string) (string, map[string]string) {
	// Check specific language patterns first (prioritize over generic)
	// Database patterns should be checked before javascript to catch MongoError, etc.
	languageOrder := []string{"database", "typescript", "react", "vue", "nextjs", "eslint", "javascript", "go", "python", "java", "rust"}

	for _, language := range languageOrder {
		if patterns, exists := p.config.ErrorPatterns[language]; exists {
			for patternName, pattern := range patterns {
				if matches := pattern.Regex().FindStringSubmatch(content); matches != nil {
					info := make(map[string]string)

					// Use configured type and severity
					info["type"] = pattern.Type
					info["severity"] = pattern.Severity

					// Extract message from regex groups
					if len(matches) > 1 {
						// Use the first capturing group as the message
						info["message"] = strings.TrimSpace(matches[1])
					} else {
						info["message"] = content
					}

					// For complex patterns with multiple groups, combine them
					if len(matches) > 2 {
						parts := make([]string, 0, len(matches)-1)
						for i := 1; i < len(matches); i++ {
							if matches[i] != "" {
								parts = append(parts, strings.TrimSpace(matches[i]))
							}
						}
						info["message"] = strings.Join(parts, ": ")
					}

					fullPatternName := fmt.Sprintf("%s.%s", language, patternName)
					return fullPatternName, info
				}
			}
		}
	}

	// Check generic patterns last
	if patterns, exists := p.config.ErrorPatterns["generic"]; exists {
		for patternName, pattern := range patterns {
			if matches := pattern.Regex().FindStringSubmatch(content); matches != nil {
				info := make(map[string]string)

				// Use configured type and severity
				info["type"] = pattern.Type
				info["severity"] = pattern.Severity

				// Extract message from regex groups
				if len(matches) > 1 {
					// Use the first capturing group as the message
					info["message"] = strings.TrimSpace(matches[1])
				} else {
					info["message"] = content
				}

				// For complex patterns with multiple groups, combine them
				if len(matches) > 2 {
					parts := make([]string, 0, len(matches)-1)
					for i := 1; i < len(matches); i++ {
						if matches[i] != "" {
							parts = append(parts, strings.TrimSpace(matches[i]))
						}
					}
					info["message"] = strings.Join(parts, ": ")
				}

				fullPatternName := fmt.Sprintf("generic.%s", patternName)
				return fullPatternName, info
			}
		}
	}

	return "", nil
}

// isErrorContinuation checks if a line continues an existing error
func (p *ConfigurableErrorParser) isErrorContinuation(content string, activeError *ErrorContext) bool {
	// Empty lines within reasonable limits
	if strings.TrimSpace(content) == "" && len(activeError.Raw) < 10 {
		return true
	}

	// Check general continuation patterns
	for _, regex := range p.config.ContinuationPatterns.General.Regexes() {
		if regex.MatchString(content) {
			return true
		}
	}

	// Check language-specific continuation patterns
	switch activeError.Language {
	case "javascript":
		for _, regex := range p.config.ContinuationPatterns.JavaScript.Regexes() {
			if regex.MatchString(content) {
				return true
			}
		}
	case "python":
		for _, regex := range p.config.ContinuationPatterns.Python.Regexes() {
			if regex.MatchString(content) {
				return true
			}
		}
	}

	// Check if it's a stack trace line
	if p.isStackTraceLine(content, activeError.Language) {
		return true
	}

	return false
}

// isStackTraceLine checks if content is a stack trace line
func (p *ConfigurableErrorParser) isStackTraceLine(content, language string) bool {
	// Check language-specific stack patterns first
	if stackConfig, exists := p.config.StackPatterns[language]; exists {
		for _, regex := range stackConfig.Regexes() {
			if regex.MatchString(content) {
				return true
			}
		}
	}

	// Check generic stack patterns
	if stackConfig, exists := p.config.StackPatterns["generic"]; exists {
		for _, regex := range stackConfig.Regexes() {
			if regex.MatchString(content) {
				return true
			}
		}
	}

	return false
}

// isErrorEnd checks if a line indicates the end of an error context
func (p *ConfigurableErrorParser) isErrorEnd(content string) bool {
	for _, regex := range p.config.EndPatterns.Regexes() {
		if regex.MatchString(content) {
			return true
		}
	}
	return false
}

// Note: isStandaloneError was removed as it was conflicting with pattern-based detection

// determineSeverity determines error severity based on config and content
func (p *ConfigurableErrorParser) determineSeverity(content, configSeverity string) string {
	// Use configured severity if provided
	if configSeverity != "" {
		return configSeverity
	}

	lower := strings.ToLower(content)

	// Check critical keywords from config
	for _, keyword := range p.config.Settings.CriticalKeywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return "critical"
		}
	}

	// Standard severity detection
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

// detectLanguage detects programming language from content
func (p *ConfigurableErrorParser) detectLanguage(content string) string {
	if !p.config.Settings.AutoDetectLanguage {
		return "unknown"
	}

	// Check each configured language
	for language, langConfig := range p.config.LanguageDetection {
		// Check file extensions
		for _, ext := range langConfig.FileExtensions {
			if strings.Contains(content, ext) {
				return language
			}
		}

		// Check stack patterns
		for _, pattern := range langConfig.StackPatterns {
			if strings.Contains(content, pattern) {
				return language
			}
		}

		// Check framework patterns
		lower := strings.ToLower(content)
		for _, pattern := range langConfig.FrameworkPatterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				return language
			}
		}

		// Check error patterns
		for _, pattern := range langConfig.ErrorPatterns {
			if strings.Contains(content, pattern) {
				return language
			}
		}
	}

	return "unknown"
}

// isSingleLineError checks if an error pattern is configured as single-line
func (p *ConfigurableErrorParser) isSingleLineError(errorType string) bool {
	// Parse the error type (format: "language.pattern_name")
	parts := strings.Split(errorType, ".")
	if len(parts) != 2 {
		return false
	}

	language, patternName := parts[0], parts[1]

	if patterns, exists := p.config.ErrorPatterns[language]; exists {
		if pattern, exists := patterns[patternName]; exists {
			return pattern.SingleLine
		}
	}

	return false
}

// finalizeError completes error processing and applies custom logic
func (p *ConfigurableErrorParser) finalizeError(errorCtx *ErrorContext) {
	// Set default message if empty
	if errorCtx.Message == "" && len(errorCtx.Raw) > 0 {
		errorCtx.Message = errorCtx.Raw[0]
	}

	// Apply custom error type processing
	p.applyCustomErrorProcessing(errorCtx)

	// Store the completed error
	p.errors = append(p.errors, *errorCtx)

	// Enforce memory limits
	if len(p.errors) > p.config.Limits.MaxErrorsInMemory {
		// Remove oldest errors
		removeCount := len(p.errors) - p.config.Limits.MaxErrorsInMemory
		p.errors = p.errors[removeCount:]
	}
}

// applyCustomErrorProcessing applies custom processing based on error type
func (p *ConfigurableErrorParser) applyCustomErrorProcessing(errorCtx *ErrorContext) {
	for _, customType := range p.config.CustomErrorTypes {
		// Check if this error matches any custom type patterns
		matched := false
		for _, regex := range customType.Regexes() {
			if regex.MatchString(errorCtx.Message) ||
				(len(errorCtx.Raw) > 0 && regex.MatchString(errorCtx.Raw[0])) {
				matched = true
				break
			}
		}

		if matched {
			// Update error type
			errorCtx.Type = customType.Type

			// Apply custom processing
			if customType.ExtractHostname && customType.HostnameRegex() != nil {
				p.extractHostname(errorCtx, customType)
			}

			// Apply DNS error replacements
			for old, new := range customType.DNSErrorReplacement {
				errorCtx.Message = strings.ReplaceAll(errorCtx.Message, old, new)
			}
		}
	}
}

// extractHostname extracts hostname information from error context
func (p *ConfigurableErrorParser) extractHostname(errorCtx *ErrorContext, customType CustomErrorType) {
	for _, line := range errorCtx.Raw {
		if matches := customType.HostnameRegex().FindStringSubmatch(line); matches != nil && len(matches) > 1 {
			hostname := matches[1]
			errorCtx.Message = fmt.Sprintf("%s (hostname: %s)", errorCtx.Message, hostname)
			break
		}
	}
}

// GetErrors returns all parsed errors
func (p *ConfigurableErrorParser) GetErrors() []ErrorContext {
	// Include any active errors that haven't been finalized
	for _, activeError := range p.activeErrors {
		p.errors = append(p.errors, *activeError)
	}
	p.activeErrors = make(map[string]*ErrorContext)

	return p.errors
}

// ClearErrors clears the error history
func (p *ConfigurableErrorParser) ClearErrors() {
	p.errors = make([]ErrorContext, 0)
	p.activeErrors = make(map[string]*ErrorContext)
}

// GetConfig returns the current configuration (for debugging/inspection)
func (p *ConfigurableErrorParser) GetConfig() *ErrorParsingConfig {
	return p.config
}

// ReloadConfig reloads the configuration from file
func (p *ConfigurableErrorParser) ReloadConfig(configPath string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	p.config = config
	return nil
}
