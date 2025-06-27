package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ToolInfo represents information about a tool from ListTools response
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ProxyTool creates a proxy MCPTool that forwards calls to a connected instance
func ProxyTool(instanceID string, toolInfo ToolInfo, connMgr *ConnectionManager) MCPTool {
	return MCPTool{
		Name:        fmt.Sprintf("%s/%s", instanceID, toolInfo.Name),
		Description: fmt.Sprintf("[%s] %s", instanceID, toolInfo.Description),
		InputSchema: toolInfo.InputSchema,
		Handler: func(args json.RawMessage) (interface{}, error) {
			// Parse arguments from raw JSON
			var argMap map[string]interface{}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &argMap); err != nil {
					return nil, fmt.Errorf("failed to parse arguments: %w", err)
				}
			}
			
			// Get the client for this instance by checking all connections
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
			
			// Forward the tool call to the instance with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			result, err := activeClient.CallTool(ctx, toolInfo.Name, argMap)
			if err != nil {
				return nil, fmt.Errorf("tool call failed: %w", err)
			}
			
			// Parse the result
			var response interface{}
			if err := json.Unmarshal(result, &response); err != nil {
				// Return raw result if parsing fails
				return result, nil
			}
			
			return response, nil
		},
		// TODO: Add streaming support when original tool supports it
		Streaming: false,
	}
}

// ExtractInstanceAndTool parses a prefixed tool name to get instance ID and original tool name
func ExtractInstanceAndTool(prefixedName string) (instanceID, toolName string, err error) {
	parts := strings.SplitN(prefixedName, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tool name format: expected 'instanceID/toolName', got '%s'", prefixedName)
	}
	return parts[0], parts[1], nil
}

// RegisterInstanceTools fetches and registers tools from a connected instance
func RegisterInstanceTools(server *StreamableServer, connMgr *ConnectionManager, instanceID string) error {
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
	
	// List tools from the instance
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	toolsData, err := activeClient.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools from instance %s: %w", instanceID, err)
	}
	
	// Parse the tools response
	var toolsResponse struct {
		Tools []ToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(toolsData, &toolsResponse); err != nil {
		return fmt.Errorf("failed to parse tools response: %w", err)
	}
	
	// Convert and register each tool
	var mcpTools []MCPTool
	for _, toolInfo := range toolsResponse.Tools {
		proxyTool := ProxyTool(instanceID, toolInfo, connMgr)
		mcpTools = append(mcpTools, proxyTool)
	}
	
	// Register all tools with the server
	return server.RegisterToolsFromInstance(instanceID, mcpTools)
}

// UnregisterInstanceTools removes all tools from a disconnected instance
func UnregisterInstanceTools(server *StreamableServer, instanceID string) error {
	return server.UnregisterToolsFromInstance(instanceID)
}

