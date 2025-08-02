package repl

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// LibraryManager manages the script library with caching and validation
type LibraryManager struct {
	mu             sync.RWMutex
	scripts        []Script
	lastLoadTime   time.Time
	cacheValidSecs int
}

// NewLibraryManager creates a new script library manager
func NewLibraryManager() *LibraryManager {
	return &LibraryManager{
		cacheValidSecs: DefaultCacheValidSecs,
	}
}

// LoadScripts loads all scripts from the filesystem with caching
func (lm *LibraryManager) LoadScripts() ([]Script, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Check if cache is still valid
	if time.Since(lm.lastLoadTime).Seconds() < float64(lm.cacheValidSecs) && lm.scripts != nil {
		return lm.scripts, nil
	}

	// Load scripts from filesystem
	scripts, err := loadAllScripts()
	if err != nil {
		return nil, err
	}

	// Sort scripts by name for consistent ordering
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	// Update cache
	lm.scripts = scripts
	lm.lastLoadTime = time.Now()

	return scripts, nil
}

// GetLibraryInfo returns information about the loaded library
func (lm *LibraryManager) GetLibraryInfo() (*LibraryInfo, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return nil, err
	}

	return &LibraryInfo{
		Scripts:  scripts,
		Count:    len(scripts),
		LoadedAt: lm.lastLoadTime,
	}, nil
}

// GetScript retrieves a specific script by name
func (lm *LibraryManager) GetScript(name string) (*Script, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return nil, err
	}

	for _, script := range scripts {
		if script.Name == name {
			return &script, nil
		}
	}

	return nil, fmt.Errorf("script not found: %s", name)
}

// ListScripts returns a list of all scripts with basic information
func (lm *LibraryManager) ListScripts() ([]map[string]interface{}, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(scripts))
	for i, script := range scripts {
		result[i] = map[string]interface{}{
			"name":        script.Name,
			"description": script.Metadata.Description,
			"category":    script.Metadata.Category,
			"tags":        script.Metadata.Tags,
			"author":      script.Metadata.Author,
			"version":     script.Metadata.Version,
			"updatedAt":   script.Metadata.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// SearchScripts searches for scripts by name, description, category, or tags
func (lm *LibraryManager) SearchScripts(query string) ([]map[string]interface{}, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var matches []map[string]interface{}

	for _, script := range scripts {
		// Search in name, description, category, and tags
		searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
			script.Name, script.Metadata.Description, script.Metadata.Category,
			strings.Join(script.Metadata.Tags, " ")))

		if strings.Contains(searchText, query) {
			matches = append(matches, map[string]interface{}{
				"name":        script.Name,
				"description": script.Metadata.Description,
				"category":    script.Metadata.Category,
				"tags":        script.Metadata.Tags,
				"author":      script.Metadata.Author,
				"version":     script.Metadata.Version,
				"updatedAt":   script.Metadata.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	return matches, nil
}

// AddScript adds a new script to the library
func (lm *LibraryManager) AddScript(name, code string, metadata ScriptMetadata) error {
	// Validate required fields
	if name == "" {
		return fmt.Errorf("script name is required")
	}
	if code == "" {
		return fmt.Errorf("script code is required")
	}
	if metadata.Description == "" {
		return fmt.Errorf("script description is required")
	}

	// Check if script already exists
	if _, err := lm.GetScript(name); err == nil {
		return fmt.Errorf("script %s already exists", name)
	}

	// Save script to filesystem
	if err := saveScript(name, code, metadata); err != nil {
		return err
	}

	// Invalidate cache
	lm.mu.Lock()
	lm.lastLoadTime = time.Time{}
	lm.mu.Unlock()

	return nil
}

// UpdateScript updates an existing script
func (lm *LibraryManager) UpdateScript(name, code string, metadata ScriptMetadata) error {
	// Validate required fields
	if name == "" {
		return fmt.Errorf("script name is required")
	}
	if code == "" {
		return fmt.Errorf("script code is required")
	}
	if metadata.Description == "" {
		return fmt.Errorf("script description is required")
	}

	// Check if script exists
	existingScript, err := lm.GetScript(name)
	if err != nil {
		return fmt.Errorf("script %s does not exist", name)
	}

	// Preserve creation time
	if !metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = existingScript.Metadata.CreatedAt
	}

	// Save updated script to filesystem
	if err := saveScript(name, code, metadata); err != nil {
		return err
	}

	// Invalidate cache
	lm.mu.Lock()
	lm.lastLoadTime = time.Time{}
	lm.mu.Unlock()

	return nil
}

// RemoveScript removes a script from the library
func (lm *LibraryManager) RemoveScript(name string) error {
	if name == "" {
		return fmt.Errorf("script name is required")
	}

	// Check if script exists
	if _, err := lm.GetScript(name); err != nil {
		return fmt.Errorf("script %s does not exist", name)
	}

	// Remove script from filesystem
	if err := removeScript(name); err != nil {
		return err
	}

	// Invalidate cache
	lm.mu.Lock()
	lm.lastLoadTime = time.Time{}
	lm.mu.Unlock()

	return nil
}

// GetCategories returns all unique categories
func (lm *LibraryManager) GetCategories() ([]string, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return nil, err
	}

	categorySet := make(map[string]bool)
	for _, script := range scripts {
		if script.Metadata.Category != "" {
			categorySet[script.Metadata.Category] = true
		}
	}

	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}

	sort.Strings(categories)
	return categories, nil
}

// GetScriptsByCategory returns all scripts in a specific category
func (lm *LibraryManager) GetScriptsByCategory(category string) ([]map[string]interface{}, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, script := range scripts {
		if script.Metadata.Category == category {
			result = append(result, map[string]interface{}{
				"name":        script.Name,
				"description": script.Metadata.Description,
				"category":    script.Metadata.Category,
				"tags":        script.Metadata.Tags,
				"author":      script.Metadata.Author,
				"version":     script.Metadata.Version,
				"updatedAt":   script.Metadata.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	return result, nil
}

// GenerateLibraryInjectionCode generates the JavaScript code to inject into the browser
func (lm *LibraryManager) GenerateLibraryInjectionCode() (string, error) {
	scripts, err := lm.LoadScripts()
	if err != nil {
		return "", err
	}

	return generateLibraryCode(scripts), nil
}

// GetScriptCode returns the raw code for a specific script
func (lm *LibraryManager) GetScriptCode(name string) (string, error) {
	script, err := lm.GetScript(name)
	if err != nil {
		return "", err
	}

	return script.Code, nil
}

// ValidateScript validates a script's syntax and metadata
func (lm *LibraryManager) ValidateScript(name, code string, metadata ScriptMetadata) error {
	// Validate name
	if name == "" {
		return fmt.Errorf("script name is required")
	}

	// Validate code
	if code == "" {
		return fmt.Errorf("script code is required")
	}

	// Validate metadata
	if metadata.Description == "" {
		return fmt.Errorf("script description is required")
	}

	// Basic JavaScript/TypeScript syntax validation
	// Check for balanced braces, parentheses, etc.
	if err := validateBasicSyntax(code); err != nil {
		return fmt.Errorf("syntax validation failed: %w", err)
	}

	return nil
}

// validateBasicSyntax performs basic syntax validation
func validateBasicSyntax(code string) error {
	// Simple validation for balanced braces and parentheses
	braceCount := 0
	parenCount := 0
	bracketCount := 0

	for _, char := range code {
		switch char {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}
	}

	if braceCount != 0 {
		return fmt.Errorf("unbalanced braces: %d", braceCount)
	}
	if parenCount != 0 {
		return fmt.Errorf("unbalanced parentheses: %d", parenCount)
	}
	if bracketCount != 0 {
		return fmt.Errorf("unbalanced brackets: %d", bracketCount)
	}

	return nil
}
