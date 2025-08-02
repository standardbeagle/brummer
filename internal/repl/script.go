package repl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ScriptMetadata represents the front matter metadata for a TypeScript script
type ScriptMetadata struct {
	Description string            `json:"description"`
	Category    string            `json:"category,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Author      string            `json:"author,omitempty"`
	Version     string            `json:"version,omitempty"`
	Examples    []string          `json:"examples,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	ReturnType  string            `json:"returnType,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty"`
}

// Script represents a TypeScript script with metadata and code
type Script struct {
	Name     string         `json:"name"`
	Filename string         `json:"filename"`
	Metadata ScriptMetadata `json:"metadata"`
	Code     string         `json:"code"`
	FilePath string         `json:"filePath"`
	ModTime  time.Time      `json:"modTime"`
}

// LibraryInfo provides enumeration and metadata functions for the script library
type LibraryInfo struct {
	Scripts  []Script  `json:"scripts"`
	Count    int       `json:"count"`
	LoadedAt time.Time `json:"loadedAt"`
}

// Constants for validation and configuration
const (
	// MaxScriptNameLength is the maximum allowed length for script names
	MaxScriptNameLength = 64
	// MinScriptNameLength is the minimum allowed length for script names
	MinScriptNameLength = 1
	// DefaultCacheValidSecs is the default cache validity in seconds
	DefaultCacheValidSecs = 60
)

// Configurable timeout variables (can be set via environment variables)
var (
	// LibraryCheckTimeout is the timeout for checking library availability (configurable via BRUMMER_LIBRARY_CHECK_TIMEOUT)
	LibraryCheckTimeout = getTimeoutFromEnv("BRUMMER_LIBRARY_CHECK_TIMEOUT", 1*time.Second)
	// LibraryInjectTimeout is the timeout for library injection (configurable via BRUMMER_LIBRARY_INJECT_TIMEOUT)
	LibraryInjectTimeout = getTimeoutFromEnv("BRUMMER_LIBRARY_INJECT_TIMEOUT", 2*time.Second)
	// REPLResponseTimeout is the timeout for REPL responses (configurable via BRUMMER_REPL_RESPONSE_TIMEOUT)
	REPLResponseTimeout = getTimeoutFromEnv("BRUMMER_REPL_RESPONSE_TIMEOUT", 5*time.Second)
)

// getTimeoutFromEnv gets a timeout duration from environment variable or returns default
func getTimeoutFromEnv(envVar string, defaultTimeout time.Duration) time.Duration {
	if envValue := os.Getenv(envVar); envValue != "" {
		if seconds, err := strconv.Atoi(envValue); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultTimeout
}

// Pre-compiled regex patterns for better performance
var (
	// Front matter regex to match /*** ... ***/
	frontMatterRegex = regexp.MustCompile(`^/\*\*\*([\s\S]*?)\*\*\*/`)
	// Script name validation regex
	scriptNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// parseScriptFile parses a TypeScript file with front matter metadata
func parseScriptFile(filePath string) (*Script, error) {
	// Get file info first for better error context
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to access script file %s: %w", filePath, err)
	}

	// Add context to file size checks
	if fileInfo.Size() > 10*1024*1024 { // 10MB limit
		return nil, fmt.Errorf("script file %s too large: %d bytes (max 10MB)", filePath, fileInfo.Size())
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		context := fmt.Sprintf("path=%s, size=%d bytes, mode=%s", filePath, fileInfo.Size(), fileInfo.Mode())
		return nil, fmt.Errorf("failed to read script file (%s): %w", context, err)
	}

	contentStr := string(content)

	// Extract front matter
	matches := frontMatterRegex.FindStringSubmatch(contentStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("script file %s missing required front matter /*** ... ***/", filePath)
	}

	frontMatter := strings.TrimSpace(matches[1])

	// Parse JSON metadata from front matter
	var metadata ScriptMetadata
	if err := json.Unmarshal([]byte(frontMatter), &metadata); err != nil {
		return nil, fmt.Errorf("invalid JSON in front matter of %s: %w", filePath, err)
	}

	// Validate required description
	if metadata.Description == "" {
		return nil, fmt.Errorf("script file %s missing required 'description' in front matter", filePath)
	}

	// Extract code (everything after front matter)
	code := frontMatterRegex.ReplaceAllString(contentStr, "")
	code = strings.TrimLeft(code, "\n\r")

	// Get script name from filename (without extension)
	filename := filepath.Base(filePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Set timestamps if not provided
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = fileInfo.ModTime()
	}
	if metadata.UpdatedAt.IsZero() {
		metadata.UpdatedAt = fileInfo.ModTime()
	}

	return &Script{
		Name:     name,
		Filename: filename,
		Metadata: metadata,
		Code:     code,
		FilePath: filePath,
		ModTime:  fileInfo.ModTime(),
	}, nil
}

// getScriptsDirectory is a variable to allow overriding in tests
var getScriptsDirectory = defaultGetScriptsDirectory

// defaultGetScriptsDirectory returns the path to the scripts directory in .brum config folder
func defaultGetScriptsDirectory() (string, error) {
	// Try current directory first for project-specific scripts
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	projectScriptsDir := filepath.Join(currentDir, ".brum", "scripts")

	// Check if project scripts directory exists
	if _, err := os.Stat(projectScriptsDir); err == nil {
		return projectScriptsDir, nil
	}

	// Fall back to user's home directory for global scripts
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	globalScriptsDir := filepath.Join(homeDir, ".brum", "scripts")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(globalScriptsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create scripts directory %s: %w", globalScriptsDir, err)
	}

	return globalScriptsDir, nil
}

// loadAllScripts loads all TypeScript scripts from the scripts directory
func loadAllScripts() ([]Script, error) {
	scriptsDir, err := getScriptsDirectory()
	if err != nil {
		return nil, err
	}

	var scripts []Script

	// Walk through the scripts directory
	err = filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .ts files
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".ts") {
			script, parseErr := parseScriptFile(path)
			if parseErr != nil {
				// Log error but continue processing other scripts
				fmt.Printf("Warning: failed to parse script %s: %v\n", path, parseErr)
				return nil
			}
			scripts = append(scripts, *script)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load scripts from %s: %w", scriptsDir, err)
	}

	return scripts, nil
}

// validateScriptName validates a script name for security and format
func validateScriptName(name string) error {
	if len(name) < MinScriptNameLength || len(name) > MaxScriptNameLength {
		return fmt.Errorf("script name must be between %d and %d characters, got %d",
			MinScriptNameLength, MaxScriptNameLength, len(name))
	}

	// Additional security checks first (more specific error messages)
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("script name contains invalid path characters: %s", name)
	}

	if !scriptNameRegex.MatchString(name) {
		return fmt.Errorf("invalid script name: %s (only alphanumeric, underscore, and hyphen allowed)", name)
	}

	return nil
}

// secureFilePath ensures the file path is within the scripts directory
func secureFilePath(scriptsDir, filename string) (string, error) {
	// Check for directory traversal patterns
	if strings.Contains(filename, ".."+string(filepath.Separator)) ||
		strings.Contains(filename, string(filepath.Separator)+"..") ||
		filename == ".." ||
		strings.HasPrefix(filename, "..") && len(filename) > 2 && (filename[2] == '/' || filename[2] == '\\') {
		return "", fmt.Errorf("path traversal attempt detected: %s", filename)
	}

	// Check for absolute paths
	if filepath.IsAbs(filename) {
		return "", fmt.Errorf("path traversal attempt detected: %s", filename)
	}

	// Check for directory separators
	if strings.ContainsAny(filename, "/\\") {
		return "", fmt.Errorf("path traversal attempt detected: %s", filename)
	}

	// Clean the filename
	cleanName := filepath.Clean(filename)
	if cleanName != filename {
		return "", fmt.Errorf("invalid filename: %s", filename)
	}

	// Construct the full path
	fullPath := filepath.Join(scriptsDir, filename)

	// Resolve to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Ensure the path is within the scripts directory
	absScriptsDir, err := filepath.Abs(scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve scripts directory: %w", err)
	}

	// Normalize paths using filepath.Clean for consistent comparison
	absPath = filepath.Clean(absPath)
	absScriptsDir = filepath.Clean(absScriptsDir)

	// On Windows, perform case-insensitive comparison
	var pathsEqual, pathWithinDir bool
	if filepath.Separator == '\\' { // Windows
		pathsEqual = strings.EqualFold(absPath, absScriptsDir)
		pathWithinDir = strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(absScriptsDir+string(filepath.Separator)))
	} else { // Unix-like systems
		pathsEqual = absPath == absScriptsDir
		pathWithinDir = strings.HasPrefix(absPath, absScriptsDir+string(filepath.Separator))
	}

	// Must be within scripts directory or the directory itself
	if !pathWithinDir && !pathsEqual {
		return "", fmt.Errorf("path traversal attempt detected: %s (resolved to %s, expected within %s)", filename, absPath, absScriptsDir)
	}

	return absPath, nil
}

// saveScript saves a script to the scripts directory
func saveScript(name, code string, metadata ScriptMetadata) error {
	// Validate script name
	if err := validateScriptName(name); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	scriptsDir, err := getScriptsDirectory()
	if err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = now
	}
	metadata.UpdatedAt = now

	// Serialize metadata to JSON
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Create the complete file content
	content := fmt.Sprintf("/***\n%s\n***/\n\n%s", string(metadataJSON), code)

	// Write to file with secure path
	filename := name + ".ts"
	filePath, err := secureFilePath(scriptsDir, filename)
	if err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	// Get file info for error context
	fileInfo := fmt.Sprintf("path=%s, size=%d bytes", filePath, len(content))

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write script file (%s): %w", fileInfo, err)
	}

	return nil
}

// removeScript removes a script from the scripts directory
func removeScript(name string) error {
	// Validate script name
	if err := validateScriptName(name); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	scriptsDir, err := getScriptsDirectory()
	if err != nil {
		return err
	}

	filename := name + ".ts"
	filePath, err := secureFilePath(scriptsDir, filename)
	if err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	// Check if file exists and get info for error context
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("script %s does not exist", name)
	}
	if err != nil {
		return fmt.Errorf("failed to check script %s: %w", name, err)
	}

	// Additional context for debugging
	context := fmt.Sprintf("path=%s, size=%d bytes, modified=%s",
		filePath, fileInfo.Size(), fileInfo.ModTime().Format(time.RFC3339))

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove script %s (%s): %w", name, context, err)
	}

	return nil
}

// sanitizeJavaScript performs basic sanitization on JavaScript code
func sanitizeJavaScript(code string) string {
	// Remove any script tags that might be embedded (including content)
	code = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(code, "")

	// Remove any HTML comment markers that could break out of JS context
	code = regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(code, "")

	// Escape backticks in template literals to prevent injection (do this last)
	code = strings.ReplaceAll(code, "`", "\\`")

	return code
}

// escapeForJavaScript escapes a string for safe inclusion in JavaScript
func escapeForJavaScript(s string) string {
	// Escape backslashes first (must be done before other escapes)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Then escape quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "'", "\\'")
	// Escape control characters
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// generateLibraryCode generates JavaScript code that creates the global library object
func generateLibraryCode(scripts []Script) string {
	var codeBuilder strings.Builder

	codeBuilder.WriteString("// Brummer Script Library - Generated at " + time.Now().Format(time.RFC3339) + "\n")
	codeBuilder.WriteString("window.brummerLibrary = {\n")

	// Add individual scripts as functions
	for i, script := range scripts {
		// Sanitize and clean up the code
		funcCode := sanitizeJavaScript(strings.TrimSpace(script.Code))

		// If code doesn't start with function, wrap it
		if !strings.HasPrefix(funcCode, "function ") && !strings.HasPrefix(funcCode, "async function ") {
			// Check if it's an arrow function or expression
			if strings.Contains(funcCode, "=>") || !strings.Contains(funcCode, "function") {
				funcCode = fmt.Sprintf("function() {\n    return %s;\n  }", funcCode)
			}
		}

		// Escape description and category for safe inclusion in comments
		safeDesc := escapeForJavaScript(script.Metadata.Description)
		codeBuilder.WriteString(fmt.Sprintf("  // %s\n", safeDesc))
		if script.Metadata.Category != "" {
			safeCat := escapeForJavaScript(script.Metadata.Category)
			codeBuilder.WriteString(fmt.Sprintf("  // Category: %s\n", safeCat))
		}
		codeBuilder.WriteString(fmt.Sprintf("  %s: %s", script.Name, funcCode))

		if i < len(scripts)-1 {
			codeBuilder.WriteString(",\n\n")
		} else {
			codeBuilder.WriteString(",\n\n")
		}
	}

	// Add metadata and utility functions
	codeBuilder.WriteString("  // Library metadata and utility functions\n")
	codeBuilder.WriteString("  __meta: {\n")
	codeBuilder.WriteString(fmt.Sprintf("    loadedAt: new Date('%s'),\n", time.Now().Format(time.RFC3339)))
	codeBuilder.WriteString(fmt.Sprintf("    count: %d,\n", len(scripts)))
	codeBuilder.WriteString("    scripts: [\n")

	for i, script := range scripts {
		examples := "[]"
		if len(script.Metadata.Examples) > 0 {
			examplesJSON, _ := json.Marshal(script.Metadata.Examples)
			examples = string(examplesJSON)
		}

		tags := "[]"
		if len(script.Metadata.Tags) > 0 {
			tagsJSON, _ := json.Marshal(script.Metadata.Tags)
			tags = string(tagsJSON)
		}

		parameters := "{}"
		if len(script.Metadata.Parameters) > 0 {
			parametersJSON, _ := json.Marshal(script.Metadata.Parameters)
			parameters = string(parametersJSON)
		}

		codeBuilder.WriteString(fmt.Sprintf("      {\n"))
		codeBuilder.WriteString(fmt.Sprintf("        name: %q,\n", escapeForJavaScript(script.Name)))
		codeBuilder.WriteString(fmt.Sprintf("        description: %q,\n", escapeForJavaScript(script.Metadata.Description)))
		codeBuilder.WriteString(fmt.Sprintf("        category: %q,\n", escapeForJavaScript(script.Metadata.Category)))
		codeBuilder.WriteString(fmt.Sprintf("        tags: %s,\n", tags))
		codeBuilder.WriteString(fmt.Sprintf("        examples: %s,\n", examples))
		codeBuilder.WriteString(fmt.Sprintf("        parameters: %s,\n", parameters))
		codeBuilder.WriteString(fmt.Sprintf("        returnType: %q,\n", escapeForJavaScript(script.Metadata.ReturnType)))
		codeBuilder.WriteString(fmt.Sprintf("        author: %q,\n", escapeForJavaScript(script.Metadata.Author)))
		codeBuilder.WriteString(fmt.Sprintf("        version: %q\n", escapeForJavaScript(script.Metadata.Version)))

		if i < len(scripts)-1 {
			codeBuilder.WriteString("      },\n")
		} else {
			codeBuilder.WriteString("      }\n")
		}
	}

	codeBuilder.WriteString("    ]\n")
	codeBuilder.WriteString("  },\n\n")

	// Add utility functions
	codeBuilder.WriteString("  // Utility functions\n")
	codeBuilder.WriteString("  list: function() {\n")
	codeBuilder.WriteString("    return this.__meta.scripts.map(s => ({\n")
	codeBuilder.WriteString("      name: s.name,\n")
	codeBuilder.WriteString("      description: s.description,\n")
	codeBuilder.WriteString("      category: s.category\n")
	codeBuilder.WriteString("    }));\n")
	codeBuilder.WriteString("  },\n\n")

	codeBuilder.WriteString("  help: function(scriptName) {\n")
	codeBuilder.WriteString("    if (!scriptName) {\n")
	codeBuilder.WriteString("      console.table(this.list());\n")
	codeBuilder.WriteString("      return 'Available scripts listed above. Use brummerLibrary.help(\"scriptName\") for details.';\n")
	codeBuilder.WriteString("    }\n")
	codeBuilder.WriteString("    const script = this.__meta.scripts.find(s => s.name === scriptName);\n")
	codeBuilder.WriteString("    if (!script) return 'Script not found: ' + scriptName;\n")
	codeBuilder.WriteString("    console.log('Script:', script.name);\n")
	codeBuilder.WriteString("    console.log('Description:', script.description);\n")
	codeBuilder.WriteString("    if (script.category) console.log('Category:', script.category);\n")
	codeBuilder.WriteString("    if (script.examples.length) console.log('Examples:', script.examples);\n")
	codeBuilder.WriteString("    if (Object.keys(script.parameters).length) console.log('Parameters:', script.parameters);\n")
	codeBuilder.WriteString("    if (script.returnType) console.log('Returns:', script.returnType);\n")
	codeBuilder.WriteString("    return 'Help displayed above';\n")
	codeBuilder.WriteString("  },\n\n")

	codeBuilder.WriteString("  categories: function() {\n")
	codeBuilder.WriteString("    const cats = [...new Set(this.__meta.scripts.map(s => s.category).filter(Boolean))];\n")
	codeBuilder.WriteString("    return cats.sort();\n")
	codeBuilder.WriteString("  },\n\n")

	codeBuilder.WriteString("  byCategory: function(category) {\n")
	codeBuilder.WriteString("    return this.__meta.scripts.filter(s => s.category === category);\n")
	codeBuilder.WriteString("  }\n")

	codeBuilder.WriteString("};\n\n")
	codeBuilder.WriteString("// Add shorthand access\n")
	codeBuilder.WriteString("window.lib = window.brummerLibrary;\n")
	codeBuilder.WriteString("console.log('Brummer Script Library loaded with', window.brummerLibrary.__meta.count, 'scripts');\n")
	codeBuilder.WriteString("console.log('Use brummerLibrary.help() or lib.help() to see available scripts');\n")

	return codeBuilder.String()
}
