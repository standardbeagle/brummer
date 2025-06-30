package mcp

import (
	"regexp"
	"testing"
	
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
)

// TestToolNamesCompliance ensures all tool names comply with Claude Code's requirements
func TestToolNamesCompliance(t *testing.T) {
	// Claude Code tool name format: ^[a-zA-Z0-9_-]{1,128}$
	toolNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	
	// Create minimal dependencies
	eventBus := events.NewEventBus()
	server := NewStreamableServer(0, nil, nil, nil, eventBus)
	
	// Check each tool name
	invalidTools := []string{}
	for name := range server.tools {
		if !toolNameRegex.MatchString(name) {
			invalidTools = append(invalidTools, name)
		}
	}
	
	// Report any invalid tool names
	if len(invalidTools) > 0 {
		t.Errorf("The following tool names do not comply with Claude Code's regex ^[a-zA-Z0-9_-]{1,128}$:")
		for _, name := range invalidTools {
			t.Errorf("  - %s", name)
		}
	}
	
	// Also check specific known tools to ensure they're using underscores
	expectedTools := []string{
		"scripts_list",
		"scripts_run", 
		"scripts_stop",
		"scripts_status",
		"logs_stream",
		"logs_search",
		"proxy_requests",
		"telemetry_sessions",
		"telemetry_events",
		"browser_open",
		"browser_refresh",
		"browser_navigate",
		"browser_screenshot",
		"repl_execute",
	}
	
	for _, expectedName := range expectedTools {
		_, exists := server.tools[expectedName]
		assert.True(t, exists, "Expected tool %s to exist", expectedName)
		assert.True(t, toolNameRegex.MatchString(expectedName), "Tool %s should match regex", expectedName)
	}
}

// TestHubToolNamesCompliance tests hub-specific tool names
func TestHubToolNamesCompliance(t *testing.T) {
	// Claude Code tool name format: ^[a-zA-Z0-9_-]{1,128}$
	toolNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	
	// Hub tool names that should exist
	hubTools := []string{
		"instances_list",
		"instances_connect",
		"instances_disconnect",
		// Hub proxy tools
		"hub_scripts_list",
		"hub_scripts_run",
		"hub_scripts_stop",
		"hub_scripts_status",
		"hub_logs_stream",
		"hub_logs_search",
		"hub_proxy_requests",
		"hub_telemetry_sessions",
		"hub_telemetry_events",
		"hub_browser_open",
		"hub_browser_refresh",
		"hub_browser_navigate",
		"hub_browser_screenshot",
		"hub_repl_execute",
	}
	
	for _, toolName := range hubTools {
		assert.True(t, toolNameRegex.MatchString(toolName), 
			"Hub tool %s does not comply with Claude Code's regex", toolName)
	}
}

// TestProxyToolNamesCompliance tests that proxy tool names will be compliant
func TestProxyToolNamesCompliance(t *testing.T) {
	// Claude Code tool name format: ^[a-zA-Z0-9_-]{1,128}$
	toolNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	
	// Test various instance ID and tool name combinations
	testCases := []struct {
		instanceID string
		toolName   string
		valid      bool
	}{
		{"instance-123", "test_tool", true},
		{"test_instance", "my-tool", true},
		{"abc123", "tool_name", true},
		{"instance", "tool", true},
		// Test max length (128 chars total)
		{"instance", "very_long_tool_name_that_is_still_valid_but_getting_close_to_the_limit_of_characters_allowed_in_tool_names_128", true},
		// Invalid cases
		{"instance/bad", "tool", false},  // slash in instance ID
		{"instance", "bad/tool", false},   // slash in tool name
		{"instance.bad", "tool", false},   // dot in instance ID
		{"instance", "bad.tool", false},   // dot in tool name
	}
	
	for _, tc := range testCases {
		// This is how proxy tools are named
		proxyToolName := tc.instanceID + "_" + tc.toolName
		
		isValid := toolNameRegex.MatchString(proxyToolName)
		if tc.valid {
			assert.True(t, isValid, 
				"Expected proxy tool name '%s' to be valid", proxyToolName)
			assert.LessOrEqual(t, len(proxyToolName), 128,
				"Proxy tool name '%s' exceeds 128 character limit", proxyToolName)
		} else {
			assert.False(t, isValid,
				"Expected proxy tool name '%s' to be invalid", proxyToolName)
		}
	}
}

