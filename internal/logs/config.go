package logs

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

//go:embed error_parsing.toml
var defaultConfigFS embed.FS

// ErrorParsingConfig represents the complete TOML configuration structure
type ErrorParsingConfig struct {
	Settings             Settings                      `toml:"settings"`
	LanguageDetection    map[string]LanguageConfig     `toml:"language_detection"`
	ErrorPatterns        map[string]map[string]Pattern `toml:"error_patterns"`
	StackPatterns        map[string]StackConfig        `toml:"stack_patterns"`
	ContinuationPatterns ContinuationConfig            `toml:"continuation_patterns"`
	EndPatterns          EndConfig                     `toml:"end_patterns"`
	LogPrefixes          LogPrefixConfig               `toml:"log_prefixes"`
	CustomErrorTypes     map[string]CustomErrorType    `toml:"custom_error_types"`
	Frameworks           map[string]FrameworkConfig    `toml:"frameworks"`
	Limits               Limits                        `toml:"limits"`
}

type Settings struct {
	MaxContextLines       int      `toml:"max_context_lines"`
	MaxContextWaitSeconds int      `toml:"max_context_wait_seconds"`
	AutoDetectLanguage    bool     `toml:"auto_detect_language"`
	CriticalKeywords      []string `toml:"critical_keywords"`
	ContinuationKeywords  []string `toml:"continuation_keywords"`
}

type LanguageConfig struct {
	FileExtensions    []string `toml:"file_extensions"`
	StackPatterns     []string `toml:"stack_patterns"`
	FrameworkPatterns []string `toml:"framework_patterns"`
	ErrorPatterns     []string `toml:"error_patterns"`
}

type Pattern struct {
	Pattern     string `toml:"pattern"`
	Type        string `toml:"type"`
	Severity    string `toml:"severity"`
	SingleLine  bool   `toml:"single_line"`
	Description string `toml:"description"`

	// Compiled regex (not in TOML)
	regex *regexp.Regexp
}

type StackConfig struct {
	Patterns []string `toml:"patterns"`

	// Compiled regexes (not in TOML)
	regexes []*regexp.Regexp
}

type ContinuationConfig struct {
	General    PatternList `toml:"general"`
	JavaScript PatternList `toml:"javascript"`
	Python     PatternList `toml:"python"`
}

type PatternList struct {
	Patterns []string `toml:"patterns"`

	// Compiled regexes (not in TOML)
	regexes []*regexp.Regexp
}

type EndConfig struct {
	Patterns []string `toml:"patterns"`

	// Compiled regexes (not in TOML)
	regexes []*regexp.Regexp
}

type LogPrefixConfig struct {
	Timestamp          PatternList              `toml:"timestamp"`
	Process            PatternList              `toml:"process"`
	ConditionalProcess ConditionalProcessConfig `toml:"conditional_process"`
}

type ConditionalProcessConfig struct {
	Patterns         []string `toml:"patterns"`
	ExcludeIfMatches []string `toml:"exclude_if_matches"`

	// Compiled regexes (not in TOML)
	regexes        []*regexp.Regexp
	excludeRegexes []*regexp.Regexp
}

type CustomErrorType struct {
	Type                string            `toml:"type"`
	Patterns            []string          `toml:"patterns"`
	ExtractHostname     bool              `toml:"extract_hostname"`
	HostnamePattern     string            `toml:"hostname_pattern"`
	DNSErrorReplacement map[string]string `toml:"dns_error_replacement"`

	// Compiled regexes (not in TOML)
	regexes       []*regexp.Regexp
	hostnameRegex *regexp.Regexp
}

type FrameworkConfig struct {
	HookErrors           []string `toml:"hook_errors"`
	JSXErrors            []string `toml:"jsx_errors"`
	BuildErrorContext    int      `toml:"build_error_context"`
	LintIntegration      bool     `toml:"lint_integration"`
	TemplateErrorContext int      `toml:"template_error_context"`
	CompositionAPIErrors bool     `toml:"composition_api_errors"`
	Patterns             []string `toml:"patterns"`
}

type Limits struct {
	MaxErrorsInMemory        int  `toml:"max_errors_in_memory"`
	MaxErrorSizeBytes        int  `toml:"max_error_size_bytes"`
	ErrorCompletionTimeoutMs int  `toml:"error_completion_timeout_ms"`
	MaxStackTraceLines       int  `toml:"max_stack_trace_lines"`
	DebugLogging             bool `toml:"debug_logging"`
}

// LoadConfig loads the error parsing configuration from TOML file
func LoadConfig(configPath string) (*ErrorParsingConfig, error) {
	var config ErrorParsingConfig

	// Use default config if no path provided or file doesn't exist
	var configData []byte
	var err error

	if configPath == "" || !fileExists(configPath) {
		// Load embedded default configuration
		configData, err = defaultConfigFS.ReadFile("error_parsing.toml")
		if err != nil {
			return nil, fmt.Errorf("failed to load default config: %w", err)
		}
	} else {
		// Load user-provided configuration
		configData, err = os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	}

	// Parse TOML
	if err := toml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config: %w", err)
	}

	// Compile all regex patterns
	if err := compileRegexes(&config); err != nil {
		return nil, fmt.Errorf("failed to compile regex patterns: %w", err)
	}

	// Set defaults if not specified
	setDefaults(&config)

	return &config, nil
}

// LoadDefaultConfig loads the embedded default configuration
func LoadDefaultConfig() (*ErrorParsingConfig, error) {
	return LoadConfig("")
}

// GetUserConfigPath returns the path where user config should be stored
func GetUserConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".brummer", "error_parsing.toml")
}

// CreateUserConfig creates a user configuration file with the default settings
func CreateUserConfig() error {
	configPath := GetUserConfigPath()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read default config
	configData, err := defaultConfigFS.ReadFile("error_parsing.toml")
	if err != nil {
		return fmt.Errorf("failed to read default config: %w", err)
	}

	// Write to user config path
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write user config: %w", err)
	}

	return nil
}

func compileRegexes(config *ErrorParsingConfig) error {
	// Compile error patterns
	for language, patterns := range config.ErrorPatterns {
		compiledPatterns := make(map[string]Pattern)
		for name, pattern := range patterns {
			regex, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				return fmt.Errorf("failed to compile error pattern %s.%s: %w", language, name, err)
			}
			pattern.regex = regex
			compiledPatterns[name] = pattern
		}
		config.ErrorPatterns[language] = compiledPatterns
	}

	// Compile stack patterns
	for language, stackConfig := range config.StackPatterns {
		regexes := make([]*regexp.Regexp, len(stackConfig.Patterns))
		for i, pattern := range stackConfig.Patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("failed to compile stack pattern %s[%d]: %w", language, i, err)
			}
			regexes[i] = regex
		}
		stackConfig.regexes = regexes
		config.StackPatterns[language] = stackConfig
	}

	// Compile continuation patterns
	if err := compilePatternList(&config.ContinuationPatterns.General); err != nil {
		return fmt.Errorf("failed to compile general continuation patterns: %w", err)
	}
	if err := compilePatternList(&config.ContinuationPatterns.JavaScript); err != nil {
		return fmt.Errorf("failed to compile javascript continuation patterns: %w", err)
	}
	if err := compilePatternList(&config.ContinuationPatterns.Python); err != nil {
		return fmt.Errorf("failed to compile python continuation patterns: %w", err)
	}

	// Compile end patterns
	regexes := make([]*regexp.Regexp, len(config.EndPatterns.Patterns))
	for i, pattern := range config.EndPatterns.Patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile end pattern[%d]: %w", i, err)
		}
		regexes[i] = regex
	}
	config.EndPatterns.regexes = regexes

	// Compile log prefix patterns
	if err := compilePatternList(&config.LogPrefixes.Timestamp); err != nil {
		return fmt.Errorf("failed to compile timestamp patterns: %w", err)
	}
	if err := compilePatternList(&config.LogPrefixes.Process); err != nil {
		return fmt.Errorf("failed to compile process patterns: %w", err)
	}

	// Compile conditional process patterns
	regexes = make([]*regexp.Regexp, len(config.LogPrefixes.ConditionalProcess.Patterns))
	for i, pattern := range config.LogPrefixes.ConditionalProcess.Patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile conditional process pattern[%d]: %w", i, err)
		}
		regexes[i] = regex
	}
	config.LogPrefixes.ConditionalProcess.regexes = regexes

	excludeRegexes := make([]*regexp.Regexp, len(config.LogPrefixes.ConditionalProcess.ExcludeIfMatches))
	for i, pattern := range config.LogPrefixes.ConditionalProcess.ExcludeIfMatches {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile conditional exclude pattern[%d]: %w", i, err)
		}
		excludeRegexes[i] = regex
	}
	config.LogPrefixes.ConditionalProcess.excludeRegexes = excludeRegexes

	// Compile custom error type patterns
	for name, customType := range config.CustomErrorTypes {
		regexes := make([]*regexp.Regexp, len(customType.Patterns))
		for i, pattern := range customType.Patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("failed to compile custom error type %s pattern[%d]: %w", name, i, err)
			}
			regexes[i] = regex
		}
		customType.regexes = regexes

		if customType.HostnamePattern != "" {
			hostnameRegex, err := regexp.Compile(customType.HostnamePattern)
			if err != nil {
				return fmt.Errorf("failed to compile hostname pattern for %s: %w", name, err)
			}
			customType.hostnameRegex = hostnameRegex
		}

		config.CustomErrorTypes[name] = customType
	}

	return nil
}

func compilePatternList(patternList *PatternList) error {
	regexes := make([]*regexp.Regexp, len(patternList.Patterns))
	for i, pattern := range patternList.Patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile pattern[%d]: %w", i, err)
		}
		regexes[i] = regex
	}
	patternList.regexes = regexes
	return nil
}

func setDefaults(config *ErrorParsingConfig) {
	if config.Settings.MaxContextLines == 0 {
		config.Settings.MaxContextLines = 50
	}
	if config.Settings.MaxContextWaitSeconds == 0 {
		config.Settings.MaxContextWaitSeconds = 2
	}
	if config.Limits.MaxErrorsInMemory == 0 {
		config.Limits.MaxErrorsInMemory = 1000
	}
	if config.Limits.MaxErrorSizeBytes == 0 {
		config.Limits.MaxErrorSizeBytes = 10240
	}
	if config.Limits.ErrorCompletionTimeoutMs == 0 {
		config.Limits.ErrorCompletionTimeoutMs = 2000
	}
	if config.Limits.MaxStackTraceLines == 0 {
		config.Limits.MaxStackTraceLines = 50
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper methods for accessing compiled regexes

func (p *Pattern) Regex() *regexp.Regexp {
	return p.regex
}

func (s *StackConfig) Regexes() []*regexp.Regexp {
	return s.regexes
}

func (p *PatternList) Regexes() []*regexp.Regexp {
	return p.regexes
}

func (e *EndConfig) Regexes() []*regexp.Regexp {
	return e.regexes
}

func (c *ConditionalProcessConfig) Regexes() []*regexp.Regexp {
	return c.regexes
}

func (c *ConditionalProcessConfig) ExcludeRegexes() []*regexp.Regexp {
	return c.excludeRegexes
}

func (c *CustomErrorType) Regexes() []*regexp.Regexp {
	return c.regexes
}

func (c *CustomErrorType) HostnameRegex() *regexp.Regexp {
	return c.hostnameRegex
}
