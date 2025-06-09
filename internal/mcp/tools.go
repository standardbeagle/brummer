package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
	
	"github.com/beagle/brummer/pkg/events"
	"github.com/google/uuid"
)

// registerTools registers all available MCP tools
func (s *StreamableServer) registerTools() {
	// Script management tools
	s.registerScriptTools()
	
	// Log management tools
	s.registerLogTools()
	
	// Proxy and telemetry tools
	s.registerProxyTools()
	
	// Browser automation tools
	s.registerBrowserTools()
	
	// REPL tool
	s.registerREPLTool()
}

func (s *StreamableServer) registerScriptTools() {
	// scripts/list - List all available scripts
	s.tools["scripts/list"] = MCPTool{
		Name: "scripts/list",
		Description: `List all available npm scripts from package.json.

Chain of thought: When you need to see what scripts are available to run, or when the user asks about available commands.

Example usage: {}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			scripts := s.processMgr.GetScripts()
			return map[string]interface{}{
				"scripts": scripts,
			}, nil
		},
	}
	
	// scripts/run - Start a script
	s.tools["scripts/run"] = MCPTool{
		Name: "scripts/run",
		Description: `Start a package.json script through Brummer.

Chain of thought: Whenever you want to run "npm run dev" or any other script, use this tool instead of running it directly. This ensures proper process management and log capturing.

Example usage:
- To start development server: {"name": "dev"}
- To run tests: {"name": "test"}
- To build project: {"name": "build"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "The name of the script to run"
				}
			},
			"required": ["name"]
		}`),
		Streaming: true,
		StreamingHandler: func(args json.RawMessage, send func(interface{})) (interface{}, error) {
			var params struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			// Start the script
			process, err := s.processMgr.StartScript(params.Name)
			if err != nil {
				return nil, err
			}
			
			// Send initial status
			send(map[string]interface{}{
				"type": "started",
				"processId": process.ID,
				"name": process.Name,
				"script": process.Script,
			})
			
			// Stream logs
			logChan := make(chan string, 100)
			s.processMgr.AddLogCallback(func(processID, line string, isError bool) {
				if processID == process.ID {
					select {
					case logChan <- line:
					default:
						// Channel full, skip
					}
				}
			})
			
			// Monitor process
			go func() {
				for {
					time.Sleep(100 * time.Millisecond)
					if process.Status != "running" {
						close(logChan)
						break
					}
				}
			}()
			
			// Stream logs
			for line := range logChan {
				send(map[string]interface{}{
					"type": "log",
					"line": line,
				})
			}
			
			return map[string]interface{}{
				"processId": process.ID,
				"status": process.Status,
				"exitCode": process.ExitCode,
			}, nil
		},
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			process, err := s.processMgr.StartScript(params.Name)
			if err != nil {
				return nil, err
			}
			
			return map[string]interface{}{
				"processId": process.ID,
				"name": process.Name,
				"script": process.Script,
				"status": string(process.Status),
			}, nil
		},
	}
	
	// scripts/stop - Stop a running script
	s.tools["scripts/stop"] = MCPTool{
		Name: "scripts/stop",
		Description: `Stop a running script process.

Chain of thought: When you need to stop a running process like a dev server, test runner, or build process.

Example usage: {"processId": "dev-1234567890"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"processId": {
					"type": "string",
					"description": "The process ID to stop"
				}
			},
			"required": ["processId"]
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				ProcessID string `json:"processId"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			if err := s.processMgr.StopProcess(params.ProcessID); err != nil {
				return nil, err
			}
			
			return map[string]interface{}{
				"success": true,
				"processId": params.ProcessID,
			}, nil
		},
	}
	
	// scripts/status - Check script status
	s.tools["scripts/status"] = MCPTool{
		Name: "scripts/status",
		Description: `Check the status of running scripts.

Chain of thought: Before starting a script, check if it's already running. Also useful to see what processes are currently active.

Example usage: {} or {"name": "dev"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "Optional script name to check"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				Name string `json:"name"`
			}
			json.Unmarshal(args, &params)
			
			processes := s.processMgr.GetAllProcesses()
			
			if params.Name != "" {
				// Filter by name
				for _, p := range processes {
					if p.Name == params.Name {
						return map[string]interface{}{
							"processId": p.ID,
							"name": p.Name,
							"status": string(p.Status),
							"startTime": p.StartTime,
							"uptime": time.Since(p.StartTime).String(),
						}, nil
					}
				}
				return map[string]interface{}{
					"name": params.Name,
					"status": "not running",
				}, nil
			}
			
			// Return all processes
			result := make([]map[string]interface{}, 0, len(processes))
			for _, p := range processes {
				result = append(result, map[string]interface{}{
					"processId": p.ID,
					"name": p.Name,
					"status": string(p.Status),
					"startTime": p.StartTime,
					"uptime": time.Since(p.StartTime).String(),
				})
			}
			
			return result, nil
		},
	}
}

func (s *StreamableServer) registerLogTools() {
	// logs/stream - Stream real-time logs
	s.tools["logs/stream"] = MCPTool{
		Name: "logs/stream",
		Description: `Stream real-time logs from running processes.

Chain of thought: When you want to monitor live output from a running process, or debug issues as they happen.

Example usage:
- All logs: {"follow": true}
- Specific process: {"processId": "dev-1234567890", "follow": true}
- Error logs only: {"level": "error", "follow": true}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"processId": {
					"type": "string",
					"description": "Optional process ID to filter logs"
				},
				"level": {
					"type": "string",
					"enum": ["all", "error", "warn", "info"],
					"description": "Log level filter"
				},
				"follow": {
					"type": "boolean",
					"default": true,
					"description": "Whether to stream new logs"
				},
				"limit": {
					"type": "integer",
					"default": 100,
					"description": "Number of historical logs to return"
				}
			}
		}`),
		Streaming: true,
		StreamingHandler: func(args json.RawMessage, send func(interface{})) (interface{}, error) {
			var params struct {
				ProcessID string `json:"processId"`
				Level     string `json:"level"`
				Follow    bool   `json:"follow"`
				Limit     int    `json:"limit"`
			}
			// Set defaults
			params.Follow = true
			params.Limit = 100
			json.Unmarshal(args, &params)
			
			// Send historical logs first
			var logs []interface{}
			if params.ProcessID != "" {
				logs = s.logStoreGetByProcessInterface(params.ProcessID)
			} else {
				logs = s.logStoreGetAllInterface()
			}
			
			// Apply limit
			if len(logs) > params.Limit {
				logs = logs[len(logs)-params.Limit:]
			}
			
			for _, log := range logs {
				send(map[string]interface{}{
					"type": "log",
					"data": log,
				})
			}
			
			if !params.Follow {
				return map[string]interface{}{
					"count": len(logs),
				}, nil
			}
			
			// Stream new logs
			logChan := make(chan interface{}, 100)
			stopChan := make(chan bool)
			
			// Subscribe to log events
			s.eventBus.Subscribe(events.LogLine, func(e events.Event) {
				if params.ProcessID == "" || e.ProcessID == params.ProcessID {
					select {
					case logChan <- e.Data:
					case <-stopChan:
						return
					default:
						// Channel full, skip
					}
				}
			})
			
			// Stream logs for 5 minutes max
			timeout := time.After(5 * time.Minute)
			count := len(logs)
			
			for {
				select {
				case log := <-logChan:
					send(map[string]interface{}{
						"type": "log",
						"data": log,
					})
					count++
				case <-timeout:
					close(stopChan)
					return map[string]interface{}{
						"count": count,
						"timedOut": true,
					}, nil
				}
			}
		},
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				ProcessID string `json:"processId"`
				Limit     int    `json:"limit"`
			}
			params.Limit = 100
			json.Unmarshal(args, &params)
			
			var logs []interface{}
			if params.ProcessID != "" {
				logs = s.logStoreGetByProcessInterface(params.ProcessID)
			} else {
				logs = s.logStoreGetAllInterface()
			}
			
			if len(logs) > params.Limit {
				logs = logs[len(logs)-params.Limit:]
			}
			
			return logs, nil
		},
	}
	
	// logs/search - Search historical logs
	s.tools["logs/search"] = MCPTool{
		Name: "logs/search",
		Description: `Search through historical logs using patterns or keywords.

Chain of thought: When debugging, search for specific errors, patterns, or messages in the logs. Useful for finding when an error first occurred or tracking down specific issues.

Example usage:
- Search for errors: {"query": "error", "level": "error"}
- Search with regex: {"query": "failed.*connection", "regex": true}
- Search in time range: {"query": "timeout", "since": "2024-01-09T10:00:00Z"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Search query (text or regex)"
				},
				"regex": {
					"type": "boolean",
					"default": false,
					"description": "Whether query is a regex pattern"
				},
				"level": {
					"type": "string",
					"enum": ["all", "error", "warn", "info"],
					"description": "Filter by log level"
				},
				"processId": {
					"type": "string",
					"description": "Filter by process ID"
				},
				"since": {
					"type": "string",
					"format": "date-time",
					"description": "Search logs since this time"
				},
				"limit": {
					"type": "integer",
					"default": 100,
					"description": "Maximum results to return"
				}
			},
			"required": ["query"]
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				Query     string `json:"query"`
				Regex     bool   `json:"regex"`
				Level     string `json:"level"`
				ProcessID string `json:"processId"`
				Since     string `json:"since"`
				Limit     int    `json:"limit"`
			}
			params.Limit = 100
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			results := s.logStore.Search(params.Query)
			
			// Apply additional filters
			filtered := make([]interface{}, 0)
			for _, result := range results {
				// TODO: Apply level, processId, and since filters
				filtered = append(filtered, result)
				if len(filtered) >= params.Limit {
					break
				}
			}
			
			return filtered, nil
		},
	}
}

func (s *StreamableServer) registerProxyTools() {
	// proxy/requests - Get HTTP requests
	s.tools["proxy/requests"] = MCPTool{
		Name: "proxy/requests",
		Description: `Get HTTP requests captured by the proxy.

Chain of thought: To see what API calls your app is making, check response times, status codes, or debug authentication issues.

Example usage:
- Recent requests: {"limit": 50}
- Requests for a process: {"processName": "dev"}
- Failed requests: {"status": "error"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"processName": {
					"type": "string",
					"description": "Filter by process name"
				},
				"status": {
					"type": "string",
					"enum": ["all", "success", "error"],
					"description": "Filter by status"
				},
				"limit": {
					"type": "integer",
					"default": 100,
					"description": "Maximum requests to return"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				ProcessName string `json:"processName"`
				Status      string `json:"status"`
				Limit       int    `json:"limit"`
			}
			params.Limit = 100
			json.Unmarshal(args, &params)
			
			if s.proxyServer == nil {
				return []interface{}{}, nil
			}
			
			var requests []interface{}
			if params.ProcessName != "" {
				reqs := s.proxyServer.GetRequestsForProcess(params.ProcessName)
				for _, req := range reqs {
					requests = append(requests, req)
				}
			} else {
				reqs := s.proxyServer.GetRequests()
				for _, req := range reqs {
					requests = append(requests, req)
				}
			}
			
			// Apply status filter
			if params.Status != "" && params.Status != "all" {
				filtered := make([]interface{}, 0)
				for _, req := range requests {
					if reqMap, ok := req.(map[string]interface{}); ok {
						isError := false
						if statusCode, ok := reqMap["StatusCode"].(int); ok {
							isError = statusCode >= 400
						}
						if (params.Status == "error" && isError) || (params.Status == "success" && !isError) {
							filtered = append(filtered, req)
						}
					}
				}
				requests = filtered
			}
			
			// Apply limit
			if len(requests) > params.Limit {
				requests = requests[len(requests)-params.Limit:]
			}
			
			return requests, nil
		},
	}
	
	// telemetry/sessions - Get browser telemetry sessions
	s.tools["telemetry/sessions"] = MCPTool{
		Name: "telemetry/sessions",
		Description: `Get browser telemetry sessions with performance metrics and errors.

Chain of thought: To monitor browser performance, JavaScript errors, memory usage, and user interactions. Useful for debugging client-side issues.

Example usage:
- All sessions: {}
- Sessions for a process: {"processName": "dev"}
- Recent session: {"limit": 1}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"processName": {
					"type": "string",
					"description": "Filter by process name"
				},
				"sessionId": {
					"type": "string",
					"description": "Get specific session by ID"
				},
				"limit": {
					"type": "integer",
					"default": 10,
					"description": "Maximum sessions to return"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				ProcessName string `json:"processName"`
				SessionID   string `json:"sessionId"`
				Limit       int    `json:"limit"`
			}
			params.Limit = 10
			json.Unmarshal(args, &params)
			
			if s.proxyServer == nil || s.proxyServer.GetTelemetryStore() == nil {
				return []interface{}{}, nil
			}
			
			telemetry := s.proxyServer.GetTelemetryStore()
			
			if params.SessionID != "" {
				session, exists := telemetry.GetSession(params.SessionID)
				if !exists {
					return nil, fmt.Errorf("session not found: %s", params.SessionID)
				}
				return session.GetMetricsSummary(), nil
			}
			
			var sessions []interface{}
			if params.ProcessName != "" {
				for _, session := range telemetry.GetSessionsForProcess(params.ProcessName) {
					sessions = append(sessions, session.GetMetricsSummary())
				}
			} else {
				for _, session := range telemetry.GetAllSessions() {
					sessions = append(sessions, session.GetMetricsSummary())
				}
			}
			
			// Apply limit
			if len(sessions) > params.Limit {
				sessions = sessions[:params.Limit]
			}
			
			return sessions, nil
		},
	}
	
	// telemetry/events - Stream telemetry events
	s.tools["telemetry/events"] = MCPTool{
		Name: "telemetry/events",
		Description: `Stream real-time telemetry events from the browser.

Chain of thought: To monitor browser activity in real-time, including console logs, errors, performance metrics, and user interactions.

Example usage:
- Stream all events: {"follow": true}
- Stream for specific session: {"sessionId": "abc-123", "follow": true}
- Get recent events: {"limit": 50, "follow": false}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"sessionId": {
					"type": "string",
					"description": "Filter by session ID"
				},
				"eventType": {
					"type": "string",
					"enum": ["all", "error", "console", "performance", "interaction"],
					"description": "Filter by event type"
				},
				"follow": {
					"type": "boolean",
					"default": true,
					"description": "Whether to stream new events"
				},
				"limit": {
					"type": "integer",
					"default": 50,
					"description": "Number of historical events"
				}
			}
		}`),
		Streaming: true,
		StreamingHandler: func(args json.RawMessage, send func(interface{})) (interface{}, error) {
			var params struct {
				SessionID string `json:"sessionId"`
				EventType string `json:"eventType"`
				Follow    bool   `json:"follow"`
				Limit     int    `json:"limit"`
			}
			params.Follow = true
			params.Limit = 50
			json.Unmarshal(args, &params)
			
			if s.proxyServer == nil || s.proxyServer.GetTelemetryStore() == nil {
				return map[string]interface{}{"error": "telemetry not available"}, nil
			}
			
			// Send historical events
			telemetry := s.proxyServer.GetTelemetryStore()
			count := 0
			
			if params.SessionID != "" {
				session, exists := telemetry.GetSession(params.SessionID)
				if exists && session != nil {
					events := session.Events
					start := 0
					if len(events) > params.Limit {
						start = len(events) - params.Limit
					}
					for i := start; i < len(events); i++ {
						send(map[string]interface{}{
							"type": "event",
							"data": events[i],
						})
						count++
					}
				}
			}
			
			if !params.Follow {
				return map[string]interface{}{"count": count}, nil
			}
			
			// Stream new events
			eventChan := make(chan interface{}, 100)
			stopChan := make(chan bool)
			
			// Subscribe to telemetry events
			s.eventBus.Subscribe(events.EventType("telemetry.received"), func(e events.Event) {
				select {
				case eventChan <- e.Data:
				case <-stopChan:
					return
				default:
				}
			})
			
			// Stream for 5 minutes max
			timeout := time.After(5 * time.Minute)
			
			for {
				select {
				case event := <-eventChan:
					send(map[string]interface{}{
						"type": "event",
						"data": event,
					})
					count++
				case <-timeout:
					close(stopChan)
					return map[string]interface{}{
						"count": count,
						"timedOut": true,
					}, nil
				}
			}
		},
		Handler: func(args json.RawMessage) (interface{}, error) {
			// Non-streaming version
			var params struct {
				SessionID string `json:"sessionId"`
				Limit     int    `json:"limit"`
			}
			params.Limit = 50
			json.Unmarshal(args, &params)
			
			if s.proxyServer == nil || s.proxyServer.GetTelemetryStore() == nil {
				return []interface{}{}, nil
			}
			
			telemetry := s.proxyServer.GetTelemetryStore()
			var events []interface{}
			
			if params.SessionID != "" {
				session, exists := telemetry.GetSession(params.SessionID)
				if exists && session != nil {
					for _, event := range session.Events {
						events = append(events, event)
					}
				}
			}
			
			// Apply limit
			if len(events) > params.Limit {
				events = events[len(events)-params.Limit:]
			}
			
			return events, nil
		},
	}
}

func (s *StreamableServer) registerBrowserTools() {
	// browser/open - Open URL in browser with proxy
	s.tools["browser/open"] = MCPTool{
		Name: "browser/open",
		Description: `Open a URL in the default browser with automatic proxy configuration.

Chain of thought: To test your web app with automatic proxy configuration for monitoring HTTP requests and telemetry. Works on Windows, Mac, Linux, and WSL2.

Example usage:
- Open detected URL: {"url": "http://localhost:3000"}
- Open with specific process: {"url": "http://localhost:3000", "processName": "dev"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "URL to open"
				},
				"processName": {
					"type": "string",
					"description": "Associate with this process name"
				}
			},
			"required": ["url"]
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				URL         string `json:"url"`
				ProcessName string `json:"processName"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			// Register URL with proxy if available
			proxyURL := params.URL
			if s.proxyServer != nil && s.proxyServer.IsRunning() {
				if params.ProcessName == "" {
					params.ProcessName = "browser"
				}
				proxyURL = s.proxyServer.RegisterURL(params.URL, params.ProcessName)
			}
			
			// Open browser
			if err := openBrowser(proxyURL); err != nil {
				return nil, fmt.Errorf("failed to open browser: %v", err)
			}
			
			return map[string]interface{}{
				"originalUrl": params.URL,
				"proxyUrl": proxyURL,
				"opened": true,
			}, nil
		},
	}
	
	// browser/refresh - Refresh browser tab
	s.tools["browser/refresh"] = MCPTool{
		Name: "browser/refresh",
		Description: `Send a refresh command to connected browser tabs.

Chain of thought: After making changes to your code, refresh the browser to see the updates without manually switching to the browser.

Example usage:
- Refresh all tabs: {}
- Refresh specific session: {"sessionId": "abc-123"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"sessionId": {
					"type": "string",
					"description": "Specific session to refresh"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				SessionID string `json:"sessionId"`
			}
			json.Unmarshal(args, &params)
			
			// Send refresh command via WebSocket to telemetry clients
			if s.proxyServer != nil {
				s.proxyServer.BroadcastToWebSockets("command", map[string]interface{}{
					"action": "refresh",
					"sessionId": params.SessionID,
				})
			}
			
			return map[string]interface{}{
				"sent": true,
			}, nil
		},
	}
	
	// browser/navigate - Navigate to URL
	s.tools["browser/navigate"] = MCPTool{
		Name: "browser/navigate",
		Description: `Navigate browser tabs to a different URL.

Chain of thought: To test different pages or routes in your app while maintaining the proxy connection for monitoring.

Example usage:
- Navigate all tabs: {"url": "/about"}
- Navigate specific session: {"url": "/dashboard", "sessionId": "abc-123"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "URL or path to navigate to"
				},
				"sessionId": {
					"type": "string",
					"description": "Specific session to navigate"
				}
			},
			"required": ["url"]
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				URL       string `json:"url"`
				SessionID string `json:"sessionId"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			// Send navigate command via WebSocket
			if s.proxyServer != nil {
				s.proxyServer.BroadcastToWebSockets("command", map[string]interface{}{
					"action": "navigate",
					"url": params.URL,
					"sessionId": params.SessionID,
				})
			}
			
			return map[string]interface{}{
				"sent": true,
				"url": params.URL,
			}, nil
		},
	}
}

func (s *StreamableServer) registerREPLTool() {
	// repl/execute - Execute JavaScript in browser context
	s.tools["repl/execute"] = MCPTool{
		Name: "repl/execute",
		Description: `Execute JavaScript code in the browser context.

Chain of thought: To debug or interact with your running web app, inspect variables, call functions, or modify the DOM. Supports async/await.

Example usage:
- Get value: {"code": "document.title"}
- Call function: {"code": "myApp.getUserData()"}
- Async code: {"code": "await fetch('/api/data').then(r => r.json())"}
- Multi-line: {"code": "const users = await getUsers();\nreturn users.length;"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"description": "JavaScript code to execute"
				},
				"sessionId": {
					"type": "string",
					"description": "Session to execute in (defaults to most recent)"
				}
			},
			"required": ["code"]
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				Code      string `json:"code"`
				SessionID string `json:"sessionId"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			
			// Create a response channel
			responseChan := make(chan map[string]interface{}, 1)
			responseID := fmt.Sprintf("repl-%d", time.Now().UnixNano())
			
			// Register response handler (this would need to be implemented)
			// s.registerREPLResponse(responseID, responseChan)
			
			// Send REPL command via WebSocket
			if s.proxyServer != nil {
				s.proxyServer.BroadcastToWebSockets("command", map[string]interface{}{
					"action": "repl",
					"code": params.Code,
					"sessionId": params.SessionID,
					"responseId": responseID,
				})
			}
			
			// Wait for response with timeout
			select {
			case response := <-responseChan:
				return response, nil
			case <-time.After(5 * time.Second):
				return map[string]interface{}{
					"error": "timeout waiting for response",
				}, nil
			}
		},
	}
}

// Helper functions

func openBrowser(url string) error {
	var cmd string
	var args []string
	
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		// Check if running in WSL
		if isWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	return exec.Command(cmd, args...).Start()
}

func isWSL() bool {
	// Check for WSL-specific environment variables or files
	releaseData, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}

// Tool list handler
func (s *StreamableServer) handleToolsList(msg *JSONRPCMessage) *JSONRPCMessage {
	tools := make([]map[string]interface{}, 0, len(s.tools))
	
	for name, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name": name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	
	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// Tool call handler
func (s *StreamableServer) handleToolCall(msg *JSONRPCMessage, w http.ResponseWriter, r *http.Request) (*JSONRPCMessage, bool) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.createErrorResponse(msg.ID, -32602, "Invalid params", nil), false
	}
	
	tool, ok := s.tools[params.Name]
	if !ok {
		return s.createErrorResponse(msg.ID, -32602, "Tool not found", nil), false
	}
	
	// Check if this tool supports streaming
	if tool.Streaming && r.Header.Get("Accept") == "text/event-stream" {
		// Set up streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		
		flusher, ok := w.(http.Flusher)
		if !ok {
			return s.createErrorResponse(msg.ID, -32603, "Streaming not supported", nil), false
		}
		
		// Create temporary session for streaming
		session := &ClientSession{
			ID:              uuid.New().String(),
			ResponseWriter:  w,
			Flusher:         flusher,
			StreamingActive: true,
		}
		
		// Execute tool with streaming
		go func() {
			defer func() {
				// Send final response
				s.sendSSEEvent(session, "done", map[string]interface{}{
					"id": msg.ID,
				})
			}()
			
			result, err := tool.StreamingHandler(params.Arguments, func(chunk interface{}) {
				// Send intermediate results
				s.sendSSEEvent(session, "message", JSONRPCMessage{
					Jsonrpc: "2.0",
					Method:  "tools/call/progress",
					Params:  mustMarshal(map[string]interface{}{
						"id":    msg.ID,
						"chunk": chunk,
					}),
				})
			})
			
			if err != nil {
				s.sendSSEEvent(session, "error", s.createErrorResponse(msg.ID, -32603, err.Error(), nil))
				return
			}
			
			// Send final result
			s.sendSSEEvent(session, "message", JSONRPCMessage{
				Jsonrpc: "2.0",
				ID:      msg.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": fmt.Sprintf("%v", result),
						},
					},
				},
			})
		}()
		
		return nil, true // Indicates streaming response
	}
	
	// Non-streaming execution
	result, err := tool.Handler(params.Arguments)
	if err != nil {
		return s.createErrorResponse(msg.ID, -32603, err.Error(), nil), false
	}
	
	// Format result based on type
	var content []map[string]interface{}
	switch v := result.(type) {
	case string:
		content = []map[string]interface{}{
			{
				"type": "text",
				"text": v,
			},
		}
	case map[string]interface{}, []interface{}:
		// For structured data, convert to JSON string
		jsonBytes, _ := json.Marshal(v)
		content = []map[string]interface{}{
			{
				"type": "text",
				"text": string(jsonBytes),
			},
		}
	default:
		// Fallback to string representation
		content = []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("%v", result),
			},
		}
	}
	
	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"content": content,
		},
	}, false
}