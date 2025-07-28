package mcp

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileOutputParameterValidation(t *testing.T) {
	// Test that all tools with file output have the correct output_file parameter
	server := &StreamableServer{
		tools: make(map[string]MCPTool),
	}

	// Register tools to test
	server.registerLogTools()
	server.registerProxyTools()

	toolsWithFileOutput := []string{
		"logs_stream",
		"logs_search",
		"proxy_requests",
		"telemetry_sessions",
		"telemetry_events",
	}

	for _, toolName := range toolsWithFileOutput {
		t.Run(toolName, func(t *testing.T) {
			tool, exists := server.tools[toolName]
			assert.True(t, exists, "Tool %s should exist", toolName)

			// Parse the input schema to check for output_file parameter
			var schema map[string]interface{}
			err := json.Unmarshal(tool.InputSchema, &schema)
			assert.NoError(t, err, "Should be able to parse schema for %s", toolName)

			properties, hasProps := schema["properties"].(map[string]interface{})
			assert.True(t, hasProps, "Schema should have properties for %s", toolName)

			outputFile, hasOutputFile := properties["output_file"].(map[string]interface{})
			assert.True(t, hasOutputFile, "Tool %s should have output_file parameter", toolName)

			// Check that output_file is string type
			paramType, hasType := outputFile["type"].(string)
			assert.True(t, hasType, "output_file should have type for %s", toolName)
			assert.Equal(t, "string", paramType, "output_file should be string type for %s", toolName)

			// Check that it has a description
			description, hasDesc := outputFile["description"].(string)
			assert.True(t, hasDesc, "output_file should have description for %s", toolName)
			assert.Contains(t, description, "file path", "Description should mention file path for %s", toolName)
		})
	}
}

func TestValidateOutputPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path allowed",
			path:    "",
			wantErr: false,
		},
		{
			name:    "normal file path",
			path:    "output.json",
			wantErr: false,
		},
		{
			name:    "path with directory",
			path:    "debug/output.json",
			wantErr: false,
		},
		{
			name:    "path traversal attack",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal not allowed",
		},
		{
			name:    "relative path with traversal",
			path:    "logs/../../../sensitive.txt",
			wantErr: true,
			errMsg:  "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "Expected error for path: %s", tt.path)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error for path: %s", tt.path)
			}
		})
	}
}

func TestFileOutputFunctionality(t *testing.T) {
	// Create temporary directory within current project for test files
	testOutputFile := "test_output.json"
	defer os.Remove(testOutputFile) // Clean up after test

	// Test validateOutputPath with our test file
	err := validateOutputPath(testOutputFile)
	assert.NoError(t, err, "Test output file path should be valid")

	// Test file creation (simulate what the tools do)
	testData := map[string]interface{}{
		"timestamp": "2025-07-27T23:00:00Z",
		"tool":      "test_tool",
		"count":     5,
		"results":   []string{"test1", "test2", "test3"},
	}

	data, err := json.MarshalIndent(testData, "", "  ")
	assert.NoError(t, err, "Should be able to marshal test data")

	err = os.WriteFile(testOutputFile, data, 0644)
	assert.NoError(t, err, "Should be able to write test file")

	// Verify file exists and contains valid JSON
	assert.FileExists(t, testOutputFile, "Output file should exist")

	// Read and verify content
	fileContent, err := os.ReadFile(testOutputFile)
	assert.NoError(t, err, "Should be able to read output file")

	var parsedData map[string]interface{}
	err = json.Unmarshal(fileContent, &parsedData)
	assert.NoError(t, err, "File content should be valid JSON")

	assert.Equal(t, "test_tool", parsedData["tool"], "Tool name should be preserved")
	assert.Equal(t, float64(5), parsedData["count"], "Count should be preserved")
}

func TestFileOutputDescriptions(t *testing.T) {
	// Test that all tools with file output mention it in their descriptions
	server := &StreamableServer{
		tools: make(map[string]MCPTool),
	}

	server.registerLogTools()
	server.registerProxyTools()

	toolsWithFileOutput := []string{
		"logs_stream",
		"logs_search",
		"proxy_requests",
		"telemetry_sessions",
		"telemetry_events",
	}

	for _, toolName := range toolsWithFileOutput {
		t.Run(toolName, func(t *testing.T) {
			tool, exists := server.tools[toolName]
			assert.True(t, exists, "Tool %s should exist", toolName)

			description := tool.Description
			assert.Contains(t, description, "file output", "Description should mention file output for %s", toolName)
		})
	}
}
