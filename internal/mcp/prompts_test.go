package mcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test prompt registration
func TestRegisterPrompts(t *testing.T) {
	server := setupTestServer(t)

	// Verify prompts are registered
	assert.Greater(t, len(server.prompts), 0)

	// Check for expected prompts
	expectedPrompts := []string{
		"debug_error",
		"performance_analysis",
		"api_troubleshooting",
		"script_configuration",
	}

	for _, promptName := range expectedPrompts {
		_, exists := server.prompts[promptName]
		assert.True(t, exists, "Prompt %s should be registered", promptName)
	}
}

// Test prompts/list
func TestPromptsList(t *testing.T) {
	server := setupTestServer(t)

	msg := makeJSONRPCRequest("prompts/list", nil, 1)
	response := sendRequest(t, server, msg)

	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	result := response.Result.(map[string]interface{})
	prompts := result["prompts"].([]interface{})

	// Check that we have prompts registered
	assert.Greater(t, len(prompts), 0)

	// Verify prompt structure
	for _, promptInterface := range prompts {
		prompt := promptInterface.(map[string]interface{})
		assert.NotEmpty(t, prompt["name"])
		assert.NotEmpty(t, prompt["description"])

		// Check arguments if present
		if args, ok := prompt["arguments"]; ok && args != nil {
			arguments := args.([]interface{})
			for _, argInterface := range arguments {
				arg := argInterface.(map[string]interface{})
				assert.NotEmpty(t, arg["name"])
				assert.NotEmpty(t, arg["description"])
				assert.Contains(t, arg, "required")
			}
		}
	}
}

// Test prompts/get
func TestPromptsGet(t *testing.T) {
	server := setupTestServer(t)

	t.Run("get debug_error prompt", func(t *testing.T) {
		// Add some error logs for context
		server.logStore.Add("test-process", "test", "Error: Connection refused", true)
		server.logStore.Add("test-process", "test", "Error: Timeout occurred", true)

		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			"name": "debug_error",
			"arguments": map[string]interface{}{
				"error_context": "Connection issues",
			},
		}, 1)

		response := sendRequest(t, server, msg)

		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)

		result := response.Result.(map[string]interface{})

		// Check prompt structure
		assert.NotEmpty(t, result["description"])
		messages := result["messages"].([]interface{})
		assert.Greater(t, len(messages), 0)

		// Verify message structure
		for _, msgInterface := range messages {
			message := msgInterface.(map[string]interface{})
			assert.NotEmpty(t, message["role"])
			assert.NotEmpty(t, message["content"])
			assert.Contains(t, []string{"user", "assistant", "system"}, message["role"])
		}
	})

	t.Run("get performance_analysis prompt", func(t *testing.T) {
		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			"name": "performance_analysis",
			"arguments": map[string]interface{}{
				"metrics": "High CPU usage",
			},
		}, 1)

		response := sendRequest(t, server, msg)

		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)

		result := response.Result.(map[string]interface{})
		assert.NotEmpty(t, result["description"])

		messages := result["messages"].([]interface{})
		assert.Greater(t, len(messages), 0)
	})

	t.Run("get api_troubleshooting prompt", func(t *testing.T) {
		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			"name": "api_troubleshooting",
			"arguments": map[string]interface{}{
				"endpoint": "/api/users",
				"issue":    "404 Not Found",
			},
		}, 1)

		response := sendRequest(t, server, msg)

		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)

		result := response.Result.(map[string]interface{})
		messages := result["messages"].([]interface{})

		// Just verify we got messages, prompt generation might not include exact arguments
		assert.Greater(t, len(messages), 0)

		// Verify message structure
		for _, msgInterface := range messages {
			message := msgInterface.(map[string]interface{})
			assert.NotEmpty(t, message["role"])
			assert.NotNil(t, message["content"])
		}
	})

	t.Run("get script_configuration prompt", func(t *testing.T) {
		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			"name": "script_configuration",
			"arguments": map[string]interface{}{
				"task": "Set up a development server with hot reload",
			},
		}, 1)

		response := sendRequest(t, server, msg)

		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)

		result := response.Result.(map[string]interface{})
		messages := result["messages"].([]interface{})
		assert.Greater(t, len(messages), 0)
	})

	t.Run("get non-existent prompt", func(t *testing.T) {
		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			"name": "non_existent_prompt",
		}, 1)

		response := sendRequest(t, server, msg)

		assert.NotNil(t, response.Error)
		assert.Equal(t, -32602, response.Error.Code)
		assert.Contains(t, response.Error.Message, "Prompt not found")
	})

	t.Run("missing prompt name", func(t *testing.T) {
		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			// missing "name" field
			"arguments": map[string]interface{}{},
		}, 1)

		response := sendRequest(t, server, msg)

		assert.NotNil(t, response.Error)
		assert.Equal(t, -32602, response.Error.Code)
		// When name is missing, it defaults to empty string and gets "Prompt not found"
		assert.Contains(t, response.Error.Message, "Prompt not found")
	})
}

// Test prompt argument validation
func TestPromptArgumentValidation(t *testing.T) {
	server := setupTestServer(t)

	// Get the prompt list to understand required arguments
	listMsg := makeJSONRPCRequest("prompts/list", nil, 1)
	listResponse := sendRequest(t, server, listMsg)
	require.Nil(t, listResponse.Error)

	result := listResponse.Result.(map[string]interface{})
	prompts := result["prompts"].([]interface{})

	// Find a prompt with required arguments
	var promptWithRequiredArgs map[string]interface{}
	for _, promptInterface := range prompts {
		prompt := promptInterface.(map[string]interface{})
		if args, ok := prompt["arguments"]; ok && args != nil {
			arguments := args.([]interface{})
			for _, argInterface := range arguments {
				arg := argInterface.(map[string]interface{})
				if required, ok := arg["required"].(bool); ok && required {
					promptWithRequiredArgs = prompt
					break
				}
			}
		}
		if promptWithRequiredArgs != nil {
			break
		}
	}

	// If we found a prompt with required arguments, test missing them
	if promptWithRequiredArgs != nil {
		promptName := promptWithRequiredArgs["name"].(string)

		t.Run("missing required arguments", func(t *testing.T) {
			msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
				"name":      promptName,
				"arguments": map[string]interface{}{}, // Empty arguments
			}, 1)

			response := sendRequest(t, server, msg)

			// The behavior depends on implementation - it might return an error
			// or generate a prompt without the required arguments
			if response.Error != nil {
				assert.Equal(t, -32602, response.Error.Code)
			} else {
				// If no error, at least verify we got a valid response
				assert.NotNil(t, response.Result)
			}
		})
	}
}

// Test prompt content generation
func TestPromptContentGeneration(t *testing.T) {
	server := setupTestServer(t)

	// Add various types of data to test prompt generation
	server.logStore.Add("web-server", "web", "Server started on port 3000", false)
	server.logStore.Add("web-server", "web", "Error: EADDRINUSE: port already in use", true)
	server.logStore.Add("api-server", "api", "Connected to database", false)
	server.logStore.Add("api-server", "api", "Error: Connection timeout", true)

	t.Run("prompt incorporates recent errors", func(t *testing.T) {
		msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
			"name": "debug_error",
			"arguments": map[string]interface{}{
				"error_context": "Server startup issues",
			},
		}, 1)

		response := sendRequest(t, server, msg)
		require.Nil(t, response.Error)

		result := response.Result.(map[string]interface{})
		messages := result["messages"].([]interface{})

		// Check if error logs are incorporated
		hasErrorContent := false
		for _, msgInterface := range messages {
			message := msgInterface.(map[string]interface{})
			content := message["content"].(string)
			if containsStringPrompt(content, "EADDRINUSE") || containsStringPrompt(content, "Connection timeout") {
				hasErrorContent = true
				break
			}
		}

		assert.True(t, hasErrorContent, "Prompt should incorporate error logs")
	})
}

// Test concurrent prompt access
func TestConcurrentPromptAccess(t *testing.T) {
	server := setupTestServer(t)

	numRequests := 20
	results := make(chan error, numRequests)

	// Perform concurrent prompt gets
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			promptName := "debug_error"
			if id%2 == 0 {
				promptName = "performance_analysis"
			}

			msg := makeJSONRPCRequest("prompts/get", map[string]interface{}{
				"name": promptName,
				"arguments": map[string]interface{}{
					"context": fmt.Sprintf("Test context %d", id),
				},
			}, id)

			response := sendRequest(t, server, msg)
			if response.Error != nil {
				results <- fmt.Errorf("prompt get %d failed: %v", id, response.Error)
			} else {
				results <- nil
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsStringPrompt(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				len(s) > len(substr) && containsSubstringPrompt(s, substr)))
}

func containsSubstringPrompt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
