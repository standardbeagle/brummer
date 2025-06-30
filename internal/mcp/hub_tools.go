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
		mcplib.WithDescription(`List all available npm scripts from a specific brummer instance via hub routing.

**When to use:**
- You're in hub mode coordinating multiple project instances
- User asks "what scripts are available in project X?"
- Need to see available scripts before using hub_scripts_run
- Working with multi-project development setup

**Hub workflow:**
1. Use instances_list to see available instances
2. Use instances_connect to establish session routing
3. Use hub_scripts_list to see scripts from connected instance
4. Use hub_scripts_run to execute specific scripts

**Instance ID discovery:**
Get instance_id from instances_list output, which shows:
- Instance ID (required parameter)
- Instance name and directory
- Connection state (must be "active")
- Port and process information

**Few-shot examples:**
1. User: "Show me scripts from the frontend project"
   → First: instances_list to find frontend instance ID
   → Then: hub_scripts_list with {"instance_id": "frontend-abc123"}

2. User: "What can I run in the backend instance?"
   → Use: hub_scripts_list with {"instance_id": "backend-def456"}

**vs. scripts_list:** Use hub_scripts_list when coordinating multiple instances through hub mode. Use scripts_list for single-instance local development.`),
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
		mcplib.WithDescription(`Start a package.json script on a specific brummer instance via hub routing with full process management.

**When to use:**
- User wants to start a script in a specific project: "start the frontend dev server", "run tests in the backend"
- Multi-project coordination: starting services across different instances
- Hub mode development where each project runs in its own instance
- Remote script execution in distributed development environments

**Hub workflow:**
1. instances_list to see available instances and their states
2. instances_connect to establish session routing (optional but recommended)
3. hub_scripts_list to see available scripts in target instance
4. hub_scripts_run to start the specific script
5. hub_logs_stream to monitor the started process

**Instance ID + Script coordination:**
- **instance_id**: Target brummer instance (get from instances_list)
- **name**: Script name from package.json (get from hub_scripts_list)
- Automatically handles process management on remote instance
- Returns process ID for use with hub_scripts_stop

**Few-shot examples:**
1. User: "Start the development server in the frontend project"
   → instances_list → find frontend instance
   → hub_scripts_run with {"instance_id": "frontend-abc123", "name": "dev"}

2. User: "Run tests in the backend instance"
   → hub_scripts_run with {"instance_id": "backend-def456", "name": "test"}

3. User: "Start all services" (multi-step)
   → hub_scripts_run for frontend: {"instance_id": "frontend-abc", "name": "dev"}
   → hub_scripts_run for backend: {"instance_id": "backend-def", "name": "start"}
   → hub_scripts_run for database: {"instance_id": "db-ghi", "name": "migrate"}

**Remote process management:**
- Starts script on target instance with full monitoring
- Handles duplicate detection (if script already running)
- Returns process ID for management operations
- Enables URL detection and proxy registration on target instance
- Supports streaming output via hub_logs_stream

**vs. scripts_run:** Use hub_scripts_run for multi-instance coordination. Use scripts_run for local single-instance development.`),
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
		mcplib.WithDescription(`Stop a running script process on a specific brummer instance via hub routing.

**When to use:**
- User wants to stop a specific service: "stop the frontend server", "kill the backend process"
- Process management across multiple instances
- Before restarting services in coordinated deployments
- Resource cleanup in multi-project environments
- Shutting down services before system maintenance

**Hub workflow:**
1. hub_scripts_status to see running processes and get process IDs
2. hub_scripts_stop with instance_id and processId to stop specific process
3. Optionally hub_scripts_run to restart with new configuration

**Required parameters:**
- **instance_id**: Target brummer instance (from instances_list)
- **processId**: Specific process to stop (from hub_scripts_status)

**Process ID discovery:**
Get processId from:
- hub_scripts_status output (shows all running processes per instance)
- hub_scripts_run output (when starting a process)
- Error messages mentioning already running processes

**Few-shot examples:**
1. User: "Stop the frontend development server"
   → hub_scripts_status to find frontend instance and process ID
   → hub_scripts_stop with {"instance_id": "frontend-abc123", "processId": "dev-1697123456"}

2. User: "Shutdown all services" (multi-step)
   → hub_scripts_status to get all running processes
   → hub_scripts_stop for each running process across instances

3. User: "Restart the backend API"
   → hub_scripts_stop with backend instance and process ID
   → hub_scripts_run to start again

**Remote process management:**
- Graceful shutdown with proper cleanup on target instance
- Handles process group termination for complex scripts
- Safe operation - no error if process already stopped
- Maintains instance connection for immediate restart if needed

**vs. scripts_stop:** Use hub_scripts_stop for multi-instance process management. Use scripts_stop for local single-instance development.`),
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
		mcplib.WithDescription(`Check the status of running scripts on a specific brummer instance with process details and proxy URLs.

**When to use:**
- User asks about running services: "what's running in the frontend?", "show me backend processes"
- Before starting scripts to avoid duplicates
- Getting process IDs for hub_scripts_stop operations
- Monitoring multi-instance development environment
- Finding proxy URLs for accessing services across instances

**Hub workflow:**
Central tool for cross-instance process monitoring:
1. instances_list to see available instances
2. hub_scripts_status to check what's running in each instance
3. Use process IDs with hub_scripts_stop if needed
4. Use proxy URLs to access running services

**Instance targeting:**
- **instance_id**: Required - which brummer instance to check
- **name**: Optional - specific script name to check

**Few-shot examples:**
1. User: "What's running in all my projects?"
   → instances_list to get all instances
   → hub_scripts_status for each active instance

2. User: "Is the frontend dev server running?"
   → hub_scripts_status with {"instance_id": "frontend-abc123", "name": "dev"}

3. User: "Show me all backend processes"
   → hub_scripts_status with {"instance_id": "backend-def456"}

4. User: "How do I access my services?"
   → hub_scripts_status to get proxy URLs for all running services

**Multi-instance process overview:**
- Shows running processes across distributed instances
- Provides process IDs needed for hub_scripts_stop
- Lists proxy URLs for web services (for team sharing)
- Displays uptime and health information
- Enables coordinated process management

**Return information per instance:**
- Process ID (required for hub_scripts_stop)
- Script name and current status
- Start time and uptime duration
- Proxy URLs for web services (if auto-detected)
- Management commands for process control

**vs. scripts_status:** Use hub_scripts_status for multi-instance monitoring. Use scripts_status for local single-instance development.`),
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
		mcplib.WithDescription(`Stream real-time logs from a specific brummer instance via hub routing for multi-project monitoring.

**When to use:**
- User wants to monitor specific instance: "watch the frontend logs", "monitor backend output"
- Multi-project debugging across distributed instances
- Real-time monitoring of services in different environments
- Coordinated development with team members on different projects
- Following logs after starting processes with hub_scripts_run

**Hub workflow:**
1. instances_list to see available instances
2. hub_scripts_run to start processes (optional)
3. hub_scripts_status to get process IDs
4. hub_logs_stream to monitor output from specific instance

**Instance targeting:**
- **instance_id**: Required - which brummer instance to stream from
- **processId**: Optional - specific process within that instance
- **level**: Optional - filter by log level (error, warn, info, all)
- **follow**: Optional - stream real-time (default: true)
- **limit**: Optional - historical logs to include (default: 100)

**Few-shot examples:**
1. User: "Watch the frontend development server logs"
   → hub_logs_stream with {"instance_id": "frontend-abc123", "processId": "dev-process-id"}

2. User: "Monitor all backend errors"
   → hub_logs_stream with {"instance_id": "backend-def456", "level": "error", "follow": true}

3. User: "Show recent activity from the API instance"
   → hub_logs_stream with {"instance_id": "api-ghi789", "limit": 50, "follow": false}

4. User: "Follow all logs from the database instance"
   → hub_logs_stream with {"instance_id": "db-jkl012", "follow": true}

**Multi-instance debugging:**
- Stream logs from multiple instances simultaneously by using multiple tool calls
- Filter by process ID to focus on specific services
- Use level filtering to focus on errors across instances
- Monitor distributed system behavior in real-time

**vs. logs_stream:** Use hub_logs_stream for multi-instance log monitoring. Use logs_stream for local single-instance development.`),
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
		mcplib.WithDescription(`Search through historical logs on a specific brummer instance via hub routing for distributed debugging.

**When to use:**
- User reports issues in specific instance: "find errors in the frontend", "search backend logs for timeouts"
- Cross-instance debugging: "which service is failing?", "find the root cause across projects"
- Historical analysis in multi-project environments
- Incident investigation across distributed development setup
- Pattern analysis in specific instance logs

**Hub workflow:**
1. instances_list to identify target instances
2. hub_logs_search in suspected instances to find issues
3. Use time-based filtering to correlate events across instances
4. hub_logs_stream for real-time monitoring after finding patterns

**Instance targeting:**
- **instance_id**: Required - which brummer instance to search
- **query**: Required - search pattern or keywords
- **regex**: Optional - use regular expressions
- **level**: Optional - filter by log level
- **processId**: Optional - search within specific process
- **since**: Optional - time-bounded search
- **limit**: Optional - maximum results (default: 100)

**Few-shot examples:**
1. User: "Find connection errors in the backend instance"
   → hub_logs_search with {"instance_id": "backend-def456", "query": "connection", "level": "error"}

2. User: "Search for API timeouts across the last hour"
   → hub_logs_search with {"instance_id": "api-ghi789", "query": "timeout", "since": "2024-01-15T10:00:00Z"}

3. User: "Find all database errors in the data service"
   → hub_logs_search with {"instance_id": "data-jkl012", "query": "database.*error", "regex": true}

4. User: "Which instance has authentication failures?"
   → Search multiple instances: hub_logs_search with "auth.*fail" across different instance_ids

**Multi-instance debugging strategy:**
1. **Distributed error tracking**: Search same error pattern across all instances
2. **Service correlation**: Find related errors in upstream/downstream services
3. **Timeline analysis**: Use "since" parameter to correlate events across instances
4. **Pattern analysis**: Use regex to find complex patterns in distributed logs

**vs. logs_search:** Use hub_logs_search for multi-instance log analysis. Use logs_search for local single-instance development.`),
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
		mcplib.WithDescription(`Get HTTP requests captured by the proxy on a specific brummer instance for distributed API debugging.

**When to use:**
- User asks about API traffic in specific service: "what requests is the frontend making?", "check backend API calls"
- Cross-service debugging: "which service is calling the failing API?", "trace request flow"
- Performance analysis across distributed services
- Authentication debugging in specific instances
- API integration testing in multi-service environments

**Hub workflow:**
1. instances_list to identify target instances
2. hub_proxy_requests to analyze HTTP traffic from specific instances
3. Cross-reference with hub_logs_search for complete debugging picture
4. Use hub_telemetry_sessions for frontend performance correlation

**Instance targeting:**
- **instance_id**: Required - which brummer instance to get requests from
- **processName**: Optional - filter by specific process within instance
- **status**: Optional - filter by success/error status
- **limit**: Optional - maximum requests to return (default: 100)

**Few-shot examples:**
1. User: "What API calls is the frontend making?"
   → hub_proxy_requests with {"instance_id": "frontend-abc123", "processName": "dev"}

2. User: "Find failed API requests in the backend service"
   → hub_proxy_requests with {"instance_id": "backend-def456", "status": "error"}

3. User: "Show recent API activity from the user service"
   → hub_proxy_requests with {"instance_id": "user-service-ghi789", "limit": 50}

4. User: "Debug the payment integration"
   → hub_proxy_requests with {"instance_id": "payment-jkl012", "processName": "api"}

**Distributed API debugging:**
- **Service isolation**: See requests from specific microservices
- **Cross-service tracing**: Follow request chains across instances
- **Performance bottlenecks**: Identify slow services in distributed architecture
- **Integration testing**: Verify API calls between services

**vs. proxy_requests:** Use hub_proxy_requests for multi-instance API monitoring. Use proxy_requests for local single-instance development.`),
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
		mcplib.WithDescription(`Get browser telemetry sessions from a specific brummer instance for distributed frontend performance monitoring.

**When to use:**
- User reports frontend performance issues in specific service: "my frontend is slow", "check the dashboard performance"
- Cross-service frontend monitoring in microservice architectures
- Comparing performance metrics across different frontend instances
- User experience analysis for specific applications
- Performance regression testing across distributed frontends

**Hub workflow:**
1. instances_list to identify frontend instances
2. hub_telemetry_sessions to get performance data from specific instances
3. hub_telemetry_events for real-time monitoring
4. Cross-reference with hub_proxy_requests for complete performance picture

**Instance targeting:**
- **instance_id**: Required - which brummer instance to get telemetry from
- **active**: Optional - only show active browser sessions
- **limit**: Optional - maximum sessions to return (default: 50)

**Few-shot examples:**
1. User: "How is the main app performing?"
   → hub_telemetry_sessions with {"instance_id": "main-app-abc123"}

2. User: "Check performance of the admin dashboard"
   → hub_telemetry_sessions with {"instance_id": "admin-def456", "active": true}

3. User: "Compare frontend performance across services"
   → hub_telemetry_sessions for each frontend instance
   → Compare Core Web Vitals across instances

4. User: "Show recent user sessions from the e-commerce site"
   → hub_telemetry_sessions with {"instance_id": "ecommerce-ghi789", "limit": 10}

**Distributed frontend monitoring:**
- **Service-specific performance**: Monitor individual frontend applications
- **Cross-application comparison**: Compare metrics across different UIs
- **User experience tracking**: Monitor real user performance per service
- **Performance regression detection**: Track performance changes per instance

**Multi-instance performance analysis:**
- Core Web Vitals (LCP, FID, CLS) per frontend service
- JavaScript error rates across different applications
- Memory usage patterns in distributed frontends
- User interaction patterns per service

**vs. telemetry_sessions:** Use hub_telemetry_sessions for multi-instance frontend monitoring. Use telemetry_sessions for local single-instance development.`),
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
		mcplib.WithDescription(`Stream real-time browser telemetry events from a specific brummer instance for distributed frontend debugging.

**When to use:**
- Real-time frontend debugging in specific service: "watch for errors in the admin panel", "monitor the checkout flow"
- Cross-service frontend monitoring during testing
- Live user experience tracking per application
- Performance regression detection during development
- Error monitoring across distributed frontend services

**Hub workflow:**
1. instances_list to identify frontend instances
2. hub_telemetry_events to stream real-time events from target instance
3. Filter by event type for focused monitoring
4. Correlate with hub_logs_stream for backend context

**Instance targeting:**
- **instance_id**: Required - which brummer instance to stream from
- **sessionId**: Optional - specific browser session within instance
- **eventType**: Optional - filter by event type (error, console, performance, interaction)
- **follow**: Optional - stream real-time events (default: true)
- **limit**: Optional - historical events to include (default: 100)

**Few-shot examples:**
1. User: "Watch for JavaScript errors in the main app"
   → hub_telemetry_events with {"instance_id": "main-app-abc123", "eventType": "error", "follow": true}

2. User: "Monitor console output from the admin dashboard"
   → hub_telemetry_events with {"instance_id": "admin-def456", "eventType": "console", "follow": true}

3. User: "Track performance events during load testing"
   → hub_telemetry_events with {"instance_id": "app-ghi789", "eventType": "performance", "follow": true}

4. User: "Monitor user interactions on the e-commerce site"
   → hub_telemetry_events with {"instance_id": "ecommerce-jkl012", "eventType": "interaction", "follow": true}

**Distributed frontend debugging:**
- **Service-specific error tracking**: Monitor errors per frontend application
- **Cross-application event correlation**: Compare events across different UIs
- **Real-time user experience monitoring**: Track interactions per service
- **Performance bottleneck identification**: Find slow operations per instance

**Event types for distributed monitoring:**
- **"error"**: JavaScript exceptions per frontend service
- **"console"**: Debug output from specific applications
- **"performance"**: Core Web Vitals and timing per service
- **"interaction"**: User behavior patterns per application

**vs. telemetry_events:** Use hub_telemetry_events for multi-instance frontend event streaming. Use telemetry_events for local single-instance development.`),
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
		mcplib.WithDescription(`Open a URL in the default browser via a specific brummer instance with distributed monitoring.

**When to use:**
- User wants to test specific service: "open the frontend app", "test the admin dashboard", "launch the API docs"
- Cross-service testing in distributed development environments
- Team collaboration with shareable URLs across instances
- Testing with monitoring enabled for specific services
- Multi-project development workflow coordination

**Hub workflow:**
1. instances_list to see available instances and their services
2. hub_scripts_status to get proxy URLs for running services
3. hub_browser_open to launch browser with monitoring for specific instance
4. hub_telemetry_events to monitor browser activity
5. hub_proxy_requests to track API calls

**Instance targeting:**
- **instance_id**: Required - which brummer instance to route through
- **url**: Required - URL to open (can be from hub_scripts_status)
- **processName**: Optional - associate with specific process for tracking

**Few-shot examples:**
1. User: "Open the frontend development server"
   → hub_scripts_status to get frontend proxy URL
   → hub_browser_open with {"instance_id": "frontend-abc123", "url": "http://localhost:20888"}

2. User: "Test the admin panel with monitoring"
   → hub_browser_open with {"instance_id": "admin-def456", "url": "http://localhost:3001/admin", "processName": "admin"}

3. User: "Launch the API documentation"
   → hub_browser_open with {"instance_id": "api-ghi789", "url": "http://localhost:8080/docs"}

4. User: "Open the e-commerce site for testing"
   → hub_browser_open with {"instance_id": "ecommerce-jkl012", "url": "http://localhost:3000", "processName": "dev"}

**Distributed browser automation:**
- **Service-specific monitoring**: Track browser activity per instance
- **Cross-service testing**: Open multiple services for integration testing
- **Team collaboration**: Share proxy URLs that route through specific instances
- **Multi-environment testing**: Test different versions/configs per instance

**Automatic features per instance:**
- Proxy configuration for request monitoring
- Telemetry collection setup
- Session tracking for performance analysis
- Request routing through specified instance

**vs. browser_open:** Use hub_browser_open for multi-instance browser coordination. Use browser_open for local single-instance development.`),
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
		mcplib.WithDescription(`Refresh connected browser tabs via a specific brummer instance for distributed development workflow.

**When to use:**
- User makes changes to specific service: "refresh the frontend", "reload the admin panel", "update the dashboard"
- Multi-service development with targeted updates
- Testing changes in specific instances without affecting others
- Coordinated refresh across distributed development environment
- Service-specific deployment testing

**Hub workflow:**
1. Make code changes to specific service
2. hub_browser_refresh to reload browser tabs connected to that instance
3. hub_telemetry_events to monitor reload performance
4. Verify changes without affecting other running services

**Instance targeting:**
- **instance_id**: Required - which brummer instance's browser connections to refresh
- **sessionId**: Optional - specific browser session within that instance

**Few-shot examples:**
1. User: "I updated the frontend CSS, refresh the browser"
   → hub_browser_refresh with {"instance_id": "frontend-abc123"}

2. User: "Reload the admin dashboard to see my changes"
   → hub_browser_refresh with {"instance_id": "admin-def456"}

3. User: "Refresh that specific testing session"
   → hub_browser_refresh with {"instance_id": "app-ghi789", "sessionId": "test-session-123"}

4. User: "Update all browser tabs for the user service"
   → hub_browser_refresh with {"instance_id": "user-service-jkl012"}

**Distributed refresh coordination:**
- **Service isolation**: Refresh only browsers connected to specific instance
- **Selective updates**: Update specific services without affecting others
- **Development efficiency**: Quick testing of changes per service
- **Team coordination**: Refresh shared environments per service

**Multi-instance development patterns:**
1. **Frontend changes**: hub_browser_refresh for frontend instance only
2. **Backend API updates**: Refresh only services that consume the API
3. **Component updates**: Refresh instances using specific components
4. **Configuration changes**: Refresh affected services only

**vs. browser_refresh:** Use hub_browser_refresh for multi-instance browser coordination. Use browser_refresh for local single-instance development.`),
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
		mcplib.WithDescription(`Navigate browser tabs to a new URL via a specific brummer instance for distributed testing workflows.

**When to use:**
- User wants to test specific pages in distributed services: "go to the user profile in the frontend", "test the admin settings page"
- Cross-service navigation testing in microservice architectures
- Multi-instance user flow testing
- Service-specific route testing and validation
- Coordinated navigation across distributed frontend applications

**Hub workflow:**
1. instances_list to identify target frontend instances
2. hub_browser_navigate to control browser navigation per instance
3. hub_telemetry_events to monitor navigation performance
4. hub_proxy_requests to track navigation-triggered API calls

**Instance targeting:**
- **instance_id**: Required - which brummer instance to route navigation through
- **url**: Required - URL or path to navigate to
- **sessionId**: Optional - specific browser session within instance

**Few-shot examples:**
1. User: "Test the user dashboard in the main app"
   → hub_browser_navigate with {"instance_id": "main-app-abc123", "url": "/dashboard"}

2. User: "Navigate to the admin settings page"
   → hub_browser_navigate with {"instance_id": "admin-def456", "url": "/admin/settings"}

3. User: "Test the checkout flow in the e-commerce instance"
   → hub_browser_navigate with {"instance_id": "ecommerce-ghi789", "url": "/checkout"}

4. User: "Go to the API documentation for the user service"
   → hub_browser_navigate with {"instance_id": "user-service-jkl012", "url": "/docs"}

**Distributed navigation testing:**
- **Service-specific routing**: Test routes within specific frontend instances
- **Cross-service user flows**: Navigate through different services in sequence
- **Multi-instance comparison**: Test same routes across different environments
- **Integration testing**: Navigate to pages that consume multiple services

**Multi-instance navigation patterns:**
1. **User flow testing**: Navigate through pages across different service instances
2. **Route validation**: Test routing in specific frontend applications
3. **Integration testing**: Navigate to pages that integrate multiple services
4. **Performance comparison**: Navigate same routes across different instances

**Monitoring during navigation:**
- Use hub_telemetry_events to track navigation performance per instance
- Use hub_proxy_requests to see API calls triggered by navigation
- Monitor service-specific performance metrics during route changes

**vs. browser_navigate:** Use hub_browser_navigate for multi-instance navigation coordination. Use browser_navigate for local single-instance development.`),
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
		mcplib.WithDescription(`Capture a screenshot of the current page via a specific brummer instance for distributed visual documentation.

**When to use:**
- User wants visual documentation of specific service: "screenshot the admin panel", "capture the frontend state"
- Cross-service visual regression testing
- Bug reporting with service-specific context
- Multi-instance UI comparison and validation
- Documentation generation for distributed applications

**Hub workflow:**
1. instances_list to identify target frontend instances
2. hub_browser_navigate to set up desired page state (if needed)
3. hub_browser_screenshot to capture visual state from specific instance
4. Use outputPath for organized screenshot management across instances

**Instance targeting:**
- **instance_id**: Required - which brummer instance to capture screenshot from
- **sessionId**: Optional - specific browser session within instance
- **outputPath**: Optional - where to save screenshot (useful for organizing by instance)

**Few-shot examples:**
1. User: "Take a screenshot of the main application"
   → hub_browser_screenshot with {"instance_id": "main-app-abc123"}

2. User: "Capture the admin dashboard for documentation"
   → hub_browser_screenshot with {"instance_id": "admin-def456", "outputPath": "admin-dashboard.png"}

3. User: "Screenshot the e-commerce checkout page"
   → hub_browser_screenshot with {"instance_id": "ecommerce-ghi789", "sessionId": "checkout-session"}

4. User: "Document the current state of all services" (multi-step)
   → hub_browser_screenshot for each instance with organized output paths

**Distributed visual testing:**
- **Service-specific capture**: Get screenshots from individual service instances
- **Cross-instance comparison**: Compare UI states across different services
- **Visual regression testing**: Track UI changes per service over time
- **Multi-environment documentation**: Capture different configurations per instance

**Screenshot organization patterns:**
1. **Service-based naming**: Use outputPath like "frontend-homepage.png", "admin-settings.png"
2. **Timestamp organization**: Include instance name and timestamp in filenames
3. **Feature documentation**: Capture specific features from relevant instances
4. **Comparison sets**: Take screenshots from multiple instances for side-by-side comparison

**Integration with instance workflow:**
- Capture screenshots after hub_scripts_run to document startup states
- Use with hub_browser_navigate to capture specific page states
- Combine with hub_telemetry_sessions for performance context

**vs. browser_screenshot:** Use hub_browser_screenshot for multi-instance visual documentation. Use browser_screenshot for local single-instance development.`),
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
		mcplib.WithDescription(`Execute JavaScript code in the browser context via a specific brummer instance for distributed debugging.

**When to use:**
- User wants to debug specific service: "check the state in the frontend", "test a function in the admin panel"
- Cross-service JavaScript debugging in distributed environments
- Service-specific state inspection and testing
- Multi-instance development with targeted code execution
- API testing from specific frontend contexts

**Hub workflow:**
1. instances_list to identify target frontend instances
2. hub_browser_open to establish browser connection (if needed)
3. hub_repl_execute to run JavaScript in specific instance context
4. Use results for debugging or state verification

**Instance targeting:**
- **instance_id**: Required - which brummer instance to execute code in
- **code**: Required - JavaScript code to execute
- **sessionId**: Optional - specific browser session within instance

**Few-shot examples:**
1. User: "Check the user state in the main app"
   → hub_repl_execute with {"instance_id": "main-app-abc123", "code": "window.currentUser || 'No user'"}

2. User: "Test the admin function in the admin panel"
   → hub_repl_execute with {"instance_id": "admin-def456", "code": "adminPanel.validatePermissions()"}

3. User: "Get cart data from the e-commerce instance"
   → hub_repl_execute with {"instance_id": "ecommerce-ghi789", "code": "await cart.getItems()"}

4. User: "Check API connectivity from the frontend"
   → hub_repl_execute with {"instance_id": "frontend-jkl012", "code": "await fetch('/api/health').then(r => r.status)"}

**Distributed debugging scenarios:**

*Cross-service state inspection:*
- Check different application states across instances
- Verify data consistency between services
- Debug service-specific issues

*API integration testing:*
- Test API calls from different frontend contexts
- Verify authentication across services
- Debug cross-origin issues per instance

*Service-specific functionality:*
- Test functions specific to each service
- Debug component behavior per instance
- Verify feature flags and configuration

**Multi-instance debugging patterns:**
1. **State comparison**: Execute same code across instances to compare results
2. **Service validation**: Test service-specific functionality
3. **Integration testing**: Verify cross-service communication
4. **Configuration debugging**: Check environment-specific settings

**Advanced distributed debugging examples:**
- Service health: "await fetch('/api/health').then(r => ({status: r.status, service: window.location.origin}))"
- Configuration check: "window.config || window.env || 'No config found'"
- API testing: "await myService.api.getData().catch(e => e.message)"

**vs. repl_execute:** Use hub_repl_execute for multi-instance JavaScript debugging. Use repl_execute for local single-instance development.`),
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