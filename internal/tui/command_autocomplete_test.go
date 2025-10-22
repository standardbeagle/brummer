package tui

import (
	"testing"

	"github.com/standardbeagle/brummer/internal/config"
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/stretchr/testify/assert"
)

// TestCommandAutocompleteInitialization tests that autocomplete initializes with "/" correctly
func TestCommandAutocompleteInitialization(t *testing.T) {
	scripts := map[string]string{
		"dev":  "npm run dev",
		"test": "npm test",
	}

	// Create config with AI providers
	cfg := &config.Config{
		AICoders: &config.AICoderConfig{
			Providers: map[string]*config.ProviderConfig{
				"claude": {},
				"bash":   {},
				"gemini": {},
			},
		},
	}

	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, nil, cfg)
	autocomplete.SetValue("/")

	suggestions := autocomplete.GetSuggestions()
	assert.NotEmpty(t, suggestions, "Should have suggestions after setting value to '/'")

	// Verify root commands are suggested
	rootCommands := []string{"/ai", "/run", "/restart", "/stop", "/clear"}
	for _, cmd := range rootCommands {
		found := false
		for _, suggestion := range suggestions {
			if suggestion == cmd {
				found = true
				break
			}
		}
		assert.True(t, found, "Should suggest command: %s", cmd)
	}
}

// TestAIProviderValidation tests that AI provider validation works with config
func TestAIProviderValidation(t *testing.T) {
	scripts := map[string]string{}

	// Test with config providers
	cfg := &config.Config{
		AICoders: &config.AICoderConfig{
			Providers: map[string]*config.ProviderConfig{
				"claude": {},
				"bash":   {},
				"gemini": {},
				"opus":   {},
			},
		},
	}

	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, nil, cfg)

	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantError string
	}{
		{
			name:      "Valid Claude provider",
			input:     "/ai claude",
			wantValid: true,
		},
		{
			name:      "Valid Bash provider",
			input:     "/ai bash",
			wantValid: true,
		},
		{
			name:      "Valid Gemini provider",
			input:     "/ai gemini",
			wantValid: true,
		},
		{
			name:      "Valid Opus provider",
			input:     "/ai opus",
			wantValid: true,
		},
		{
			name:      "Invalid provider",
			input:     "/ai nonexistent",
			wantValid: false,
			wantError: "Invalid AI provider",
		},
		{
			name:      "Missing provider name",
			input:     "/ai",
			wantValid: false,
			wantError: "Please specify an AI provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autocomplete.SetValue(tt.input)
			valid, errMsg := autocomplete.ValidateInput()

			assert.Equal(t, tt.wantValid, valid, "Validation result mismatch for: %s", tt.input)
			if !tt.wantValid {
				assert.Contains(t, errMsg, tt.wantError, "Error message should contain: %s", tt.wantError)
			}
		})
	}
}

// TestAIProviderValidationFallback tests validation with no config (uses fallback list)
func TestAIProviderValidationFallback(t *testing.T) {
	scripts := map[string]string{}

	// No config - should use fallback list
	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, nil, nil)

	fallbackProviders := []string{
		"aider", "bash", "claude", "claude-secure", "copilot", "copilot-secure",
		"fish", "gemini", "gemini-secure", "local", "opencode", "opus", "sonnet", "zsh",
	}

	for _, provider := range fallbackProviders {
		t.Run("Fallback_"+provider, func(t *testing.T) {
			autocomplete.SetValue("/ai " + provider)
			valid, errMsg := autocomplete.ValidateInput()

			assert.True(t, valid, "Provider '%s' should be valid in fallback list", provider)
			assert.Empty(t, errMsg, "Should have no error for valid provider: %s", provider)
		})
	}
}

// TestAIProviderSuggestions tests that AI provider suggestions come from config
func TestAIProviderSuggestions(t *testing.T) {
	scripts := map[string]string{}

	cfg := &config.Config{
		AICoders: &config.AICoderConfig{
			Providers: map[string]*config.ProviderConfig{
				"claude":  {},
				"bash":    {},
				"gemini":  {},
				"custom1": {},
				"custom2": {},
			},
		},
	}

	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, nil, cfg)
	autocomplete.SetValue("/ai ")

	suggestions := autocomplete.GetSuggestions()
	assert.NotEmpty(t, suggestions, "Should have AI provider suggestions")

	// Verify configured providers are in suggestions
	configuredProviders := []string{"claude", "bash", "gemini", "custom1", "custom2"}
	for _, provider := range configuredProviders {
		found := false
		for _, suggestion := range suggestions {
			if suggestion == provider {
				found = true
				break
			}
		}
		assert.True(t, found, "Should suggest configured provider: %s", provider)
	}
}

// TestRunCommandValidation tests /run command validation
func TestRunCommandValidation(t *testing.T) {
	scripts := map[string]string{
		"dev":   "npm run dev",
		"test":  "npm test",
		"build": "npm run build",
	}

	// Create mock process manager (no eventBus needed for this test)
	pm, err := processes.NewManager(".", nil, false)
	assert.NoError(t, err, "Should create process manager without error")

	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, pm, nil)

	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantError string
	}{
		{
			name:      "Valid run command",
			input:     "/run dev",
			wantValid: true,
		},
		{
			name:      "Valid run command with test",
			input:     "/run test",
			wantValid: true,
		},
		{
			name:      "Invalid script name",
			input:     "/run nonexistent",
			wantValid: false,
			wantError: "Script 'nonexistent' not found",
		},
		{
			name:      "Missing script name",
			input:     "/run",
			wantValid: false,
			wantError: "Please specify a script name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autocomplete.SetValue(tt.input)
			valid, errMsg := autocomplete.ValidateInput()

			assert.Equal(t, tt.wantValid, valid, "Validation result mismatch for: %s", tt.input)
			if !tt.wantValid {
				assert.Contains(t, errMsg, tt.wantError, "Error message should contain: %s", tt.wantError)
			}
		})
	}
}

// TestClearCommandValidation tests /clear command validation
func TestClearCommandValidation(t *testing.T) {
	scripts := map[string]string{
		"dev": "npm run dev",
	}

	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, nil, nil)

	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantError string
	}{
		{
			name:      "Clear all",
			input:     "/clear all",
			wantValid: true,
		},
		{
			name:      "Clear logs",
			input:     "/clear logs",
			wantValid: true,
		},
		{
			name:      "Clear errors",
			input:     "/clear errors",
			wantValid: true,
		},
		{
			name:      "Clear web",
			input:     "/clear web",
			wantValid: true,
		},
		{
			name:      "Clear script logs",
			input:     "/clear dev",
			wantValid: true,
		},
		{
			name:      "Invalid clear target",
			input:     "/clear invalid",
			wantValid: false,
			wantError: "Invalid clear target",
		},
		{
			name:      "Clear without target defaults to all",
			input:     "/clear",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autocomplete.SetValue(tt.input)
			valid, errMsg := autocomplete.ValidateInput()

			assert.Equal(t, tt.wantValid, valid, "Validation result mismatch for: %s", tt.input)
			if !tt.wantValid {
				assert.Contains(t, errMsg, tt.wantError, "Error message should contain: %s", tt.wantError)
			}
		})
	}
}

// TestProxyCommandValidation tests /proxy command validation
func TestProxyCommandValidation(t *testing.T) {
	autocomplete := NewCommandAutocompleteWithProcessManager(map[string]string{}, nil, nil)

	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantError string
	}{
		{
			name:      "Proxy list",
			input:     "/proxy list",
			wantValid: true,
		},
		{
			name:      "Proxy with http URL",
			input:     "/proxy http://localhost:3000",
			wantValid: true,
		},
		{
			name:      "Proxy with https URL",
			input:     "/proxy https://example.com",
			wantValid: true,
		},
		{
			name:      "Proxy without scheme (auto-adds http)",
			input:     "/proxy localhost:3000",
			wantValid: true,
		},
		{
			name:      "Missing URL or subcommand",
			input:     "/proxy",
			wantValid: false,
			wantError: "Please specify a subcommand or URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autocomplete.SetValue(tt.input)
			valid, errMsg := autocomplete.ValidateInput()

			assert.Equal(t, tt.wantValid, valid, "Validation result mismatch for: %s", tt.input)
			if !tt.wantValid {
				assert.Contains(t, errMsg, tt.wantError, "Error message should contain: %s", tt.wantError)
			}
		})
	}
}

// TestArbitraryCommandValidation tests that commands without "/" are always valid
func TestArbitraryCommandValidation(t *testing.T) {
	autocomplete := NewCommandAutocompleteWithProcessManager(map[string]string{}, nil, nil)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Simple command",
			input: "ls -la",
		},
		{
			name:  "Node command",
			input: "node server.js",
		},
		{
			name:  "Git command",
			input: "git status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autocomplete.SetValue(tt.input)
			valid, errMsg := autocomplete.ValidateInput()

			assert.True(t, valid, "Arbitrary command should be valid: %s", tt.input)
			assert.Empty(t, errMsg, "Should have no error for arbitrary command: %s", tt.input)
		})
	}
}

// TestSuggestionFiltering tests that suggestions are filtered based on input
func TestSuggestionFiltering(t *testing.T) {
	scripts := map[string]string{
		"dev":   "npm run dev",
		"test":  "npm test",
		"build": "npm run build",
	}

	autocomplete := NewCommandAutocompleteWithProcessManager(scripts, nil, nil)

	tests := []struct {
		name            string
		input           string
		expectedContain []string
		expectedNotHave []string
	}{
		{
			name:            "Root slash shows all root commands",
			input:           "/",
			expectedContain: []string{"/ai", "/run", "/restart", "/stop", "/clear", "/proxy"},
		},
		{
			name:            "Run command shows scripts",
			input:           "/run ",
			expectedContain: []string{"dev", "test", "build"},
		},
		{
			name:            "Partial run command filters scripts",
			input:           "/run d",
			expectedContain: []string{"dev"},
			expectedNotHave: []string{"test", "build"},
		},
		{
			name:            "Clear shows targets",
			input:           "/clear ",
			expectedContain: []string{"all", "logs", "errors", "web"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autocomplete.SetValue(tt.input)
			suggestions := autocomplete.GetSuggestions()

			for _, expected := range tt.expectedContain {
				found := false
				for _, suggestion := range suggestions {
					if suggestion == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "Should suggest: %s for input: %s", expected, tt.input)
			}

			for _, notExpected := range tt.expectedNotHave {
				found := false
				for _, suggestion := range suggestions {
					if suggestion == notExpected {
						found = true
						break
					}
				}
				assert.False(t, found, "Should NOT suggest: %s for input: %s", notExpected, tt.input)
			}
		})
	}
}
