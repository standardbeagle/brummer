package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterHubTools registers all hub proxy tools that include an instance_id parameter
func RegisterHubTools(srv *server.MCPServer, connMgr *ConnectionManager) {
	// Script management tools
	registerHubScriptTools(srv, connMgr)
	
	// Log management tools  
	registerHubLogTools(srv, connMgr)
	
	// Proxy and telemetry tools
	registerHubProxyTools(srv, connMgr)
	
	// Browser automation tools
	registerHubBrowserTools(srv, connMgr)
	
	// REPL tool
	registerHubREPLTool(srv, connMgr)
}

// getHubClient gets the active client for an instance
func getHubClient(connMgr *ConnectionManager, instanceID string) (*HubClient, error) {
	connections := connMgr.ListInstances()
	for _, conn := range connections {
		if conn.InstanceID == instanceID && conn.State == StateActive && conn.Client != nil {
			return conn.Client, nil
		}
	}
	return nil, fmt.Errorf("instance %s is not connected or not active", instanceID)
}

// callInstanceTool calls a tool on a specific instance
func callInstanceTool(ctx context.Context, connMgr *ConnectionManager, instanceID, toolName string, args map[string]interface{}) (json.RawMessage, error) {
	client, err := getHubClient(connMgr, instanceID)
	if err != nil {
		return nil, err
	}
	
	return client.CallTool(ctx, toolName, args)
}

func registerHubScriptTools(srv *server.MCPServer, connMgr *ConnectionManager) {
	// hub_scripts_list - List scripts from a specific instance
	listTool := mcplib.NewTool("hub_scripts_list",
		mcplib.WithDescription("List all available npm scripts from a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance to list scripts from"),
		),
	)
	srv.AddTool(listTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "scripts_list", nil)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to list scripts: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
	
	// hub_scripts_run - Run a script on a specific instance
	runTool := mcplib.NewTool("hub_scripts_run",
		mcplib.WithDescription("Start a package.json script on a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance to run the script on"),
		),
		mcplib.WithString("name",
			mcplib.Required(),
			mcplib.Description("The name of the script to run"),
		),
	)
	srv.AddTool(runTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		scriptName, err := request.RequireString("name")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := map[string]interface{}{
			"name": scriptName,
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "scripts_run", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to run script: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
	
	// hub_scripts_stop - Stop a script on a specific instance
	stopTool := mcplib.NewTool("hub_scripts_stop",
		mcplib.WithDescription("Stop a running script process on a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("processId",
			mcplib.Required(),
			mcplib.Description("The process ID to stop"),
		),
	)
	srv.AddTool(stopTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		processID, err := request.RequireString("processId")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := map[string]interface{}{
			"processId": processID,
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "scripts_stop", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to stop script: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
	
	// hub_scripts_status - Check script status on a specific instance
	statusTool := mcplib.NewTool("hub_scripts_status",
		mcplib.WithDescription("Check the status of running scripts on a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("name",
			mcplib.Description("Optional script name to check"),
		),
	)
	srv.AddTool(statusTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if name := request.GetString("name", ""); name != "" {
			args["name"] = name
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "scripts_status", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to get status: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
}

func registerHubLogTools(srv *server.MCPServer, connMgr *ConnectionManager) {
	// hub_logs_stream - Stream logs from a specific instance
	streamTool := mcplib.NewTool("hub_logs_stream",
		mcplib.WithDescription("Stream real-time logs from a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("processId",
			mcplib.Description("Optional process ID to filter logs"),
		),
		mcplib.WithString("level",
			mcplib.Description("Log level filter (all, error, warn, info)"),
		),
		mcplib.WithBoolean("follow",
			mcplib.Description("Whether to stream new logs (default: true)"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Number of historical logs to return (default: 100)"),
		),
	)
	srv.AddTool(streamTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if processID := request.GetString("processId", ""); processID != "" {
			args["processId"] = processID
		}
		if level := request.GetString("level", ""); level != "" {
			args["level"] = level
		}
		if follow := request.GetBool("follow", true); follow {
			args["follow"] = follow
		}
		if limit := request.GetInt("limit", 0); limit > 0 {
			args["limit"] = limit
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "logs_stream", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to stream logs: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
	
	// hub_logs_search - Search logs on a specific instance
	searchTool := mcplib.NewTool("hub_logs_search",
		mcplib.WithDescription("Search through historical logs on a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("query",
			mcplib.Required(),
			mcplib.Description("Search query (text or regex)"),
		),
		mcplib.WithBoolean("regex",
			mcplib.Description("Whether query is a regex pattern"),
		),
		mcplib.WithString("level",
			mcplib.Description("Filter by log level (all, error, warn, info)"),
		),
		mcplib.WithString("processId",
			mcplib.Description("Filter by process ID"),
		),
		mcplib.WithString("since",
			mcplib.Description("Search logs since this time (RFC3339 format)"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum results to return (default: 100)"),
		),
	)
	srv.AddTool(searchTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		query, err := request.RequireString("query")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := map[string]interface{}{
			"query": query,
		}
		
		if regex := request.GetBool("regex", false); regex {
			args["regex"] = regex
		}
		if level := request.GetString("level", ""); level != "" {
			args["level"] = level
		}
		if processID := request.GetString("processId", ""); processID != "" {
			args["processId"] = processID
		}
		if since := request.GetString("since", ""); since != "" {
			args["since"] = since
		}
		if limit := request.GetInt("limit", 0); limit > 0 {
			args["limit"] = limit
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "logs_search", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to search logs: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
}

func registerHubProxyTools(srv *server.MCPServer, connMgr *ConnectionManager) {
	// hub_proxy_requests - Get HTTP requests from a specific instance
	requestsTool := mcplib.NewTool("hub_proxy_requests",
		mcplib.WithDescription("Get HTTP requests captured by the proxy on a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("processName",
			mcplib.Description("Filter by process name"),
		),
		mcplib.WithString("status",
			mcplib.Description("Filter by status (all, success, error)"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum requests to return (default: 100)"),
		),
	)
	srv.AddTool(requestsTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if processName := request.GetString("processName", ""); processName != "" {
			args["processName"] = processName
		}
		if status := request.GetString("status", ""); status != "" {
			args["status"] = status
		}
		if limit := request.GetInt("limit", 0); limit > 0 {
			args["limit"] = limit
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "proxy_requests", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to get proxy requests: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})

	// hub_telemetry_sessions - Get telemetry sessions from a specific instance
	sessionsTool := mcplib.NewTool("hub_telemetry_sessions",
		mcplib.WithDescription("Get browser telemetry sessions from a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithBoolean("active",
			mcplib.Description("Only show active sessions"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum sessions to return (default: 50)"),
		),
	)
	srv.AddTool(sessionsTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if active := request.GetBool("active", false); active {
			args["active"] = active
		}
		if limit := request.GetInt("limit", 0); limit > 0 {
			args["limit"] = limit
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "telemetry_sessions", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to get telemetry sessions: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})

	// hub_telemetry_events - Stream telemetry events from a specific instance
	eventsTool := mcplib.NewTool("hub_telemetry_events",
		mcplib.WithDescription("Stream real-time browser telemetry events from a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("sessionId",
			mcplib.Description("Filter by session ID"),
		),
		mcplib.WithString("eventType",
			mcplib.Description("Filter by event type"),
		),
		mcplib.WithBoolean("follow",
			mcplib.Description("Stream new events as they arrive (default: true)"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Number of historical events to return (default: 100)"),
		),
	)
	srv.AddTool(eventsTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if sessionID := request.GetString("sessionId", ""); sessionID != "" {
			args["sessionId"] = sessionID
		}
		if eventType := request.GetString("eventType", ""); eventType != "" {
			args["eventType"] = eventType
		}
		if follow := request.GetBool("follow", true); follow {
			args["follow"] = follow
		}
		if limit := request.GetInt("limit", 0); limit > 0 {
			args["limit"] = limit
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "telemetry_events", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to get telemetry events: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
}

func registerHubBrowserTools(srv *server.MCPServer, connMgr *ConnectionManager) {
	// hub_browser_open - Open URL in browser via a specific instance
	openTool := mcplib.NewTool("hub_browser_open",
		mcplib.WithDescription("Open a URL in the default browser via a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("url",
			mcplib.Required(),
			mcplib.Description("URL to open"),
		),
		mcplib.WithString("processName",
			mcplib.Description("Associate with this process name"),
		),
	)
	srv.AddTool(openTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		url, err := request.RequireString("url")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := map[string]interface{}{
			"url": url,
		}
		
		if processName := request.GetString("processName", ""); processName != "" {
			args["processName"] = processName
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "browser_open", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to open browser: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})

	// hub_browser_refresh - Refresh browser tabs via a specific instance
	refreshTool := mcplib.NewTool("hub_browser_refresh",
		mcplib.WithDescription("Refresh connected browser tabs via a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("sessionId",
			mcplib.Description("Specific session to refresh (optional)"),
		),
	)
	srv.AddTool(refreshTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if sessionID := request.GetString("sessionId", ""); sessionID != "" {
			args["sessionId"] = sessionID
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "browser_refresh", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to refresh browser: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})

	// hub_browser_navigate - Navigate browser tabs via a specific instance
	navigateTool := mcplib.NewTool("hub_browser_navigate",
		mcplib.WithDescription("Navigate browser tabs to a new URL via a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("url",
			mcplib.Required(),
			mcplib.Description("URL to navigate to"),
		),
		mcplib.WithString("sessionId",
			mcplib.Description("Specific session to navigate (defaults to most recent)"),
		),
	)
	srv.AddTool(navigateTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		url, err := request.RequireString("url")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := map[string]interface{}{
			"url": url,
		}
		
		if sessionID := request.GetString("sessionId", ""); sessionID != "" {
			args["sessionId"] = sessionID
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "browser_navigate", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to navigate browser: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})

	// hub_browser_screenshot - Take screenshot via a specific instance
	screenshotTool := mcplib.NewTool("hub_browser_screenshot",
		mcplib.WithDescription("Capture a screenshot of the current page via a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("sessionId",
			mcplib.Description("Specific session to screenshot (defaults to most recent)"),
		),
		mcplib.WithString("outputPath",
			mcplib.Description("Path to save screenshot (optional)"),
		),
	)
	srv.AddTool(screenshotTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := make(map[string]interface{})
		if sessionID := request.GetString("sessionId", ""); sessionID != "" {
			args["sessionId"] = sessionID
		}
		if outputPath := request.GetString("outputPath", ""); outputPath != "" {
			args["outputPath"] = outputPath
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "browser_screenshot", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to take screenshot: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
}

func registerHubREPLTool(srv *server.MCPServer, connMgr *ConnectionManager) {
	// hub_repl_execute - Execute JavaScript in browser context via a specific instance
	replTool := mcplib.NewTool("hub_repl_execute",
		mcplib.WithDescription("Execute JavaScript code in the browser context via a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance"),
		),
		mcplib.WithString("code",
			mcplib.Required(),
			mcplib.Description("JavaScript code to execute"),
		),
		mcplib.WithString("sessionId",
			mcplib.Description("Session to execute in (defaults to most recent)"),
		),
	)
	srv.AddTool(replTool, func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		instanceID, err := request.RequireString("instance_id")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		code, err := request.RequireString("code")
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		
		args := map[string]interface{}{
			"code": code,
		}
		
		if sessionID := request.GetString("sessionId", ""); sessionID != "" {
			args["sessionId"] = sessionID
		}
		
		result, err := callInstanceTool(ctx, connMgr, instanceID, "repl_execute", args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Failed to execute REPL: %v", err)), nil
		}
		
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	})
}