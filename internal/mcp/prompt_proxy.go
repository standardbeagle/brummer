package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PromptInfo represents information about a prompt from ListPrompts response
type PromptInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Arguments   []PromptArgument  `json:"arguments,omitempty"`
}

// PromptWithHandler extends Prompt with a handler function
type PromptWithHandler struct {
	Prompt
	Handler func(args map[string]interface{}) (interface{}, error)
}

// ProxyPrompt creates a proxy Prompt that forwards requests to a connected instance
func ProxyPrompt(instanceID string, promptInfo PromptInfo, connMgr *ConnectionManager) PromptWithHandler {
	// Prefix the name with instance ID
	prefixedName := fmt.Sprintf("%s/%s", instanceID, promptInfo.Name)
	
	return PromptWithHandler{
		Prompt: Prompt{
			Name:        prefixedName,
			Description: fmt.Sprintf("[%s] %s", instanceID, promptInfo.Description),
			Arguments:   promptInfo.Arguments,
		},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			// Get the client for this instance
			connections := connMgr.ListInstances()
			
			var activeClient *HubClient
			for _, conn := range connections {
				if conn.InstanceID == instanceID && conn.State == StateActive && conn.Client != nil {
					activeClient = conn.Client
					break
				}
			}
			
			if activeClient == nil {
				return nil, fmt.Errorf("instance %s is not connected or has no active client", instanceID)
			}
			
			// Forward the prompt request to the instance with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			result, err := activeClient.GetPrompt(ctx, promptInfo.Name, args)
			if err != nil {
				return nil, fmt.Errorf("prompt request failed: %w", err)
			}
			
			// Parse the result
			var response interface{}
			if err := json.Unmarshal(result, &response); err != nil {
				// Return raw result if parsing fails
				return result, nil
			}
			
			return response, nil
		},
	}
}

// ExtractInstanceAndPrompt parses a prefixed prompt name to get instance ID and original name
func ExtractInstanceAndPrompt(prefixedName string) (instanceID, promptName string, err error) {
	parts := strings.SplitN(prefixedName, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid prompt name format: expected 'instanceID/promptName', got '%s'", prefixedName)
	}
	return parts[0], parts[1], nil
}

// RegisterInstancePrompts fetches and registers prompts from a connected instance
func RegisterInstancePrompts(server *StreamableServer, connMgr *ConnectionManager, instanceID string) error {
	// Find the connection for the instance
	connections := connMgr.ListInstances()
	
	var activeClient *HubClient
	for _, conn := range connections {
		if conn.InstanceID == instanceID && conn.State == StateActive && conn.Client != nil {
			activeClient = conn.Client
			break
		}
	}
	
	if activeClient == nil {
		return fmt.Errorf("no active client available for instance %s", instanceID)
	}
	
	// List prompts from the instance
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	promptsData, err := activeClient.ListPrompts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list prompts from instance %s: %w", instanceID, err)
	}
	
	// Parse the prompts response
	var promptsResponse struct {
		Prompts []PromptInfo `json:"prompts"`
	}
	if err := json.Unmarshal(promptsData, &promptsResponse); err != nil {
		return fmt.Errorf("failed to parse prompts response: %w", err)
	}
	
	// Convert and register each prompt
	var prompts []PromptWithHandler
	for _, promptInfo := range promptsResponse.Prompts {
		proxyPrompt := ProxyPrompt(instanceID, promptInfo, connMgr)
		prompts = append(prompts, proxyPrompt)
	}
	
	// Register all prompts with the server
	return server.RegisterPromptsFromInstance(instanceID, prompts)
}

// UnregisterInstancePrompts removes all prompts from a disconnected instance
func UnregisterInstancePrompts(server *StreamableServer, instanceID string) error {
	return server.UnregisterPromptsFromInstance(instanceID)
}

// PromptTemplate represents a prompt template that can be filled with arguments
type PromptTemplate struct {
	Name        string
	Description string
	Template    string
	Arguments   []PromptArgument
}

// FillPromptTemplate fills a prompt template with the provided arguments
func FillPromptTemplate(template string, args map[string]interface{}) (string, error) {
	result := template
	
	// Simple template replacement - replaces {{key}} with value
	for key, value := range args {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}
	
	// Check for any remaining placeholders
	if strings.Contains(result, "{{") && strings.Contains(result, "}}") {
		return "", fmt.Errorf("template contains unfilled placeholders")
	}
	
	return result, nil
}