package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

// handleToolsList handles the tools/list request
func (s *StreamableServer) handleToolsList(msg *JSONRPCMessage) *JSONRPCMessage {
	tools := make([]map[string]interface{}, 0)
	for name, tool := range s.tools {
		toolInfo := map[string]interface{}{
			"name":        name,
			"description": tool.Description,
		}
		if tool.InputSchema != nil {
			var schema interface{}
			if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
				toolInfo["inputSchema"] = schema
			}
		}
		tools = append(tools, toolInfo)
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolCall handles the tools/call request
func (s *StreamableServer) handleToolCall(msg *JSONRPCMessage, w http.ResponseWriter, r *http.Request) (*JSONRPCMessage, bool) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.createErrorResponse(msg.ID, -32602, "Invalid params", err.Error()), false
	}

	tool, exists := s.tools[params.Name]
	if !exists {
		return s.createErrorResponse(msg.ID, -32602, "Tool not found", fmt.Sprintf("Tool '%s' not found", params.Name)), false
	}

	// Handle streaming tools
	if tool.Streaming && tool.StreamingHandler != nil {
		// Set up SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Streaming not fully implemented yet
		// For streaming tools, we need to handle them differently
		// This is a simplified version - for now just use regular handler
		result, err := tool.Handler(params.Arguments)
		if err != nil {
			return s.createErrorResponse(msg.ID, -32000, "Tool execution failed", err.Error()), false
		}

		return &JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      msg.ID,
			Result:  result,
		}, false
	}

	// Handle non-streaming tools
	result, err := tool.Handler(params.Arguments)
	if err != nil {
		return s.createErrorResponse(msg.ID, -32000, "Tool execution failed", err.Error()), false
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result:  result,
	}, false
}

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
	// scripts_list - List all available scripts
	s.tools["scripts_list"] = MCPTool{
		Name: "scripts_list",
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

	// scripts_run - Start a script
	s.tools["scripts_run"] = MCPTool{
		Name: "scripts_run",
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

			// Check if script is already running
			for _, proc := range s.processMgr.GetAllProcesses() {
				if proc.Name == params.Name && proc.Status == "running" {
					// Script is already running, send status and return
					send(map[string]interface{}{
						"type":      "duplicate",
						"processId": proc.ID,
						"name":      proc.Name,
						"status":    string(proc.Status),
						"message":   fmt.Sprintf("Script '%s' is already running with process ID: %s", proc.Name, proc.ID),
						"startTime": proc.StartTime.Format(time.RFC3339),
						"runtime":   time.Since(proc.StartTime).String(),
						"commands": map[string]string{
							"stop":    fmt.Sprintf("scripts/stop {\"processId\": \"%s\"}", proc.ID),
							"restart": "First stop the process, then run again",
						},
					})

					return map[string]interface{}{
						"duplicate": true,
						"processId": proc.ID,
						"status":    string(proc.Status),
						"message":   fmt.Sprintf("Script '%s' is already running", proc.Name),
					}, nil
				}
			}

			// Start the script
			process, err := s.processMgr.StartScript(params.Name)
			if err != nil {
				return nil, err
			}

			// Send initial status
			send(map[string]interface{}{
				"type":      "started",
				"processId": process.ID,
				"name":      process.Name,
				"script":    process.Script,
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
				"status":    process.Status,
				"exitCode":  process.ExitCode,
			}, nil
		},
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}

			// Check if script is already running
			for _, proc := range s.processMgr.GetAllProcesses() {
				if proc.Name == params.Name && proc.Status == "running" {
					// Script is already running, return its current state
					result := map[string]interface{}{
						"processId": proc.ID,
						"name":      proc.Name,
						"script":    proc.Script,
						"status":    string(proc.Status),
						"duplicate": true,
						"message":   fmt.Sprintf("Script '%s' is already running with process ID: %s", proc.Name, proc.ID),
						"startTime": proc.StartTime.Format(time.RFC3339),
						"runtime":   time.Since(proc.StartTime).String(),
						"commands": map[string]string{
							"stop":    fmt.Sprintf("scripts/stop {\"processId\": \"%s\"}", proc.ID),
							"restart": "First stop the process, then run again",
							"status":  "Check status with: scripts/status",
						},
					}

					// Get proxy URLs if available
					if s.proxyServer != nil {
						mappings := s.proxyServer.GetURLMappings()
						processUrls := make([]map[string]interface{}, 0)
						for _, m := range mappings {
							if m.ProcessName == proc.Name {
								processUrls = append(processUrls, map[string]interface{}{
									"targetUrl": m.TargetURL,
									"proxyUrl":  m.ProxyURL,
									"label":     m.Label,
								})
							}
						}
						if len(processUrls) > 0 {
							result["proxyUrls"] = processUrls
						}
					}

					return result, nil
				}
			}

			// Script not running, start it
			process, err := s.processMgr.StartScript(params.Name)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"processId": process.ID,
				"name":      process.Name,
				"script":    process.Script,
				"status":    string(process.Status),
				"duplicate": false,
			}, nil
		},
	}

	// scripts_stop - Stop a running script
	s.tools["scripts_stop"] = MCPTool{
		Name: "scripts_stop",
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
				"success":   true,
				"processId": params.ProcessID,
			}, nil
		},
	}

	// scripts_status - Check script status
	s.tools["scripts_status"] = MCPTool{
		Name: "scripts_status",
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

			// Get proxy mappings if available
			var proxyMappings []proxy.URLMapping
			if s.proxyServer != nil {
				proxyMappings = s.proxyServer.GetURLMappings()
			}

			if params.Name != "" {
				// Filter by name
				for _, p := range processes {
					if p.Name == params.Name {
						result := map[string]interface{}{
							"processId": p.ID,
							"name":      p.Name,
							"status":    string(p.Status),
							"startTime": p.StartTime,
							"uptime":    time.Since(p.StartTime).String(),
						}

						// Add proxy URLs for this process
						if len(proxyMappings) > 0 {
							processUrls := make([]map[string]interface{}, 0)
							for _, m := range proxyMappings {
								if m.ProcessName == p.Name {
									processUrls = append(processUrls, map[string]interface{}{
										"targetUrl": m.TargetURL,
										"proxyUrl":  m.ProxyURL,
										"label":     m.Label,
									})
								}
							}
							if len(processUrls) > 0 {
								result["proxyUrls"] = processUrls
							}
						}

						// Add commands for managing the process
						if p.Status == "running" {
							result["commands"] = map[string]string{
								"stop":    fmt.Sprintf("scripts/stop {\"processId\": \"%s\"}", p.ID),
								"restart": "First stop the process, then run again",
							}
						}

						return result, nil
					}
				}
				return map[string]interface{}{
					"name":   params.Name,
					"status": "not running",
				}, nil
			}

			// Return all processes
			result := make([]map[string]interface{}, 0, len(processes))
			for _, p := range processes {
				procInfo := map[string]interface{}{
					"processId": p.ID,
					"name":      p.Name,
					"status":    string(p.Status),
					"startTime": p.StartTime,
					"uptime":    time.Since(p.StartTime).String(),
				}

				// Add proxy URLs for each process
				if len(proxyMappings) > 0 {
					processUrls := make([]map[string]interface{}, 0)
					for _, m := range proxyMappings {
						if m.ProcessName == p.Name {
							processUrls = append(processUrls, map[string]interface{}{
								"targetUrl": m.TargetURL,
								"proxyUrl":  m.ProxyURL,
								"label":     m.Label,
							})
						}
					}
					if len(processUrls) > 0 {
						procInfo["proxyUrls"] = processUrls
					}
				}

				result = append(result, procInfo)
			}

			return result, nil
		},
	}
}

func (s *StreamableServer) registerLogTools() {
	// logs_stream - Stream real-time logs
	s.tools["logs_stream"] = MCPTool{
		Name: "logs_stream",
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
						"count":    count,
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

	// logs_search - Search historical logs
	s.tools["logs_search"] = MCPTool{
		Name: "logs_search",
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

			// Parse since time if provided
			var sinceTime time.Time
			if params.Since != "" {
				if t, err := time.Parse(time.RFC3339, params.Since); err == nil {
					sinceTime = t
				}
			}

			// Apply additional filters
			filtered := make([]interface{}, 0)
			for _, logEntry := range results {
				// Filter by level
				if params.Level != "" && params.Level != "all" {
					switch params.Level {
					case "error":
						if !logEntry.IsError {
							continue
						}
					case "warn":
						// Simple heuristic for warnings
						if !strings.Contains(strings.ToLower(logEntry.Content), "warn") {
							continue
						}
					case "info":
						if logEntry.IsError {
							continue
						}
					}
				}

				// Filter by processId
				if params.ProcessID != "" {
					if logEntry.ProcessID != params.ProcessID {
						continue
					}
				}

				// Filter by since time
				if !sinceTime.IsZero() {
					if logEntry.Timestamp.Before(sinceTime) {
						continue
					}
				}

				// Convert to interface format for JSON response
				filtered = append(filtered, map[string]interface{}{
					"id":          logEntry.ID,
					"processId":   logEntry.ProcessID,
					"processName": logEntry.ProcessName,
					"timestamp":   logEntry.Timestamp.Format(time.RFC3339),
					"message":     logEntry.Content,
					"isError":     logEntry.IsError,
					"tags":        logEntry.Tags,
					"priority":    logEntry.Priority,
				})
				if len(filtered) >= params.Limit {
					break
				}
			}

			return filtered, nil
		},
	}
}

func (s *StreamableServer) registerProxyTools() {
	// proxy_requests - Get HTTP requests
	s.tools["proxy_requests"] = MCPTool{
		Name: "proxy_requests",
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

	// telemetry_sessions - Get browser telemetry sessions
	s.tools["telemetry_sessions"] = MCPTool{
		Name: "telemetry_sessions",
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

	// telemetry_events - Stream telemetry events
	s.tools["telemetry_events"] = MCPTool{
		Name: "telemetry_events",
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
						"count":    count,
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
	// browser_open - Open URL in browser with proxy
	s.tools["browser_open"] = MCPTool{
		Name: "browser_open",
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
				"proxyUrl":    proxyURL,
				"opened":      true,
			}, nil
		},
	}

	// browser_refresh - Refresh browser tab
	s.tools["browser_refresh"] = MCPTool{
		Name: "browser_refresh",
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
					"action":    "refresh",
					"sessionId": params.SessionID,
				})
			}

			return map[string]interface{}{
				"sent": true,
			}, nil
		},
	}

	// browser_navigate - Navigate to URL
	s.tools["browser_navigate"] = MCPTool{
		Name: "browser_navigate",
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
					"action":    "navigate",
					"url":       params.URL,
					"sessionId": params.SessionID,
				})
			}

			return map[string]interface{}{
				"sent": true,
				"url":  params.URL,
			}, nil
		},
	}
}

func (s *StreamableServer) registerREPLTool() {
	// repl_execute - Execute JavaScript in browser context
	s.tools["repl_execute"] = MCPTool{
		Name: "repl_execute",
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

			// Create response ID and register channel
			responseID := fmt.Sprintf("repl-%d", time.Now().UnixNano())
			responseChan := s.registerREPLResponse(responseID)
			defer s.unregisterREPLResponse(responseID)

			// Send REPL command via WebSocket
			if s.proxyServer != nil {
				s.proxyServer.BroadcastToWebSockets("command", map[string]interface{}{
					"action":     "repl",
					"code":       params.Code,
					"sessionId":  params.SessionID,
					"responseId": responseID,
				})
			} else {
				return map[string]interface{}{
					"error": "proxy server not available",
				}, nil
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

	// browser_screenshot - Capture screenshot of browser tab
	s.tools["browser_screenshot"] = MCPTool{
		Name: "browser_screenshot",
		Description: `Capture a screenshot of the current browser tab.

Chain of thought: To capture the current state of your web application for debugging, documentation, or visual regression testing. Supports full page capture or visible viewport only.

Example usage:
- Capture visible viewport: {"format": "png"}
- Capture full page: {"fullPage": true, "format": "png"}
- Specific session: {"sessionId": "abc-123", "format": "jpeg", "quality": 85}
- With selector: {"selector": "#main-content", "format": "png"}`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"sessionId": {
					"type": "string",
					"description": "Session to capture screenshot from (defaults to most recent)"
				},
				"format": {
					"type": "string",
					"enum": ["png", "jpeg", "webp"],
					"default": "png",
					"description": "Image format for the screenshot"
				},
				"quality": {
					"type": "integer",
					"minimum": 0,
					"maximum": 100,
					"default": 90,
					"description": "Quality for jpeg/webp formats (0-100)"
				},
				"fullPage": {
					"type": "boolean",
					"default": false,
					"description": "Capture full page instead of just viewport"
				},
				"selector": {
					"type": "string",
					"description": "CSS selector to capture specific element"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				SessionID string `json:"sessionId"`
				Format    string `json:"format"`
				Quality   int    `json:"quality"`
				FullPage  bool   `json:"fullPage"`
				Selector  string `json:"selector"`
			}
			// Set defaults
			params.Format = "png"
			params.Quality = 90
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}

			// Create response ID and register channel
			responseID := fmt.Sprintf("screenshot-%d", time.Now().UnixNano())
			responseChan := s.registerREPLResponse(responseID)
			defer s.unregisterREPLResponse(responseID)

			// Build JavaScript code to capture screenshot
			var jsCode string
			if params.Selector != "" {
				// Capture specific element
				jsCode = fmt.Sprintf(`
					const element = document.querySelector('%s');
					if (!element) {
						throw new Error('Element not found: %s');
					}
					
					// Use html2canvas if available, otherwise use DOM-to-image approach
					if (typeof html2canvas !== 'undefined') {
						return html2canvas(element, {
							useCORS: true,
							allowTaint: true,
							scale: window.devicePixelRatio || 1
						}).then(canvas => {
							return canvas.toDataURL('image/%s', %f);
						});
					} else {
						// Fallback: try to use native browser APIs or return error
						return Promise.reject('Screenshot capture requires html2canvas library');
					}
				`, params.Selector, params.Selector, params.Format, float64(params.Quality)/100)
			} else if params.FullPage {
				// Capture full page
				jsCode = fmt.Sprintf(`
					// Store original scroll position
					const originalScrollX = window.scrollX;
					const originalScrollY = window.scrollY;
					
					// Get full document dimensions
					const fullHeight = Math.max(
						document.body.scrollHeight,
						document.body.offsetHeight,
						document.documentElement.clientHeight,
						document.documentElement.scrollHeight,
						document.documentElement.offsetHeight
					);
					const fullWidth = Math.max(
						document.body.scrollWidth,
						document.body.offsetWidth,
						document.documentElement.clientWidth,
						document.documentElement.scrollWidth,
						document.documentElement.offsetWidth
					);
					
					// Create canvas for full page
					const canvas = document.createElement('canvas');
					const ctx = canvas.getContext('2d');
					const scale = window.devicePixelRatio || 1;
					
					canvas.width = fullWidth * scale;
					canvas.height = fullHeight * scale;
					canvas.style.width = fullWidth + 'px';
					canvas.style.height = fullHeight + 'px';
					ctx.scale(scale, scale);
					
					// Use html2canvas for full page capture
					return new Promise((resolve, reject) => {
						// Dynamically load html2canvas if not already loaded
						if (typeof html2canvas === 'undefined') {
							const script = document.createElement('script');
							script.src = 'https://cdnjs.cloudflare.com/ajax/libs/html2canvas/1.4.1/html2canvas.min.js';
							script.onload = () => {
								// Capture full page
								html2canvas(document.body, {
									width: fullWidth,
									height: fullHeight,
									windowWidth: fullWidth,
									windowHeight: fullHeight,
									x: 0,
									y: 0,
									useCORS: true,
									allowTaint: true,
									scale: window.devicePixelRatio || 1,
									logging: false,
									scrollX: 0,
									scrollY: 0
								}).then(canvas => {
									// Restore scroll position
									window.scrollTo(originalScrollX, originalScrollY);
									resolve(canvas.toDataURL('image/%s', %f));
								}).catch(err => {
									window.scrollTo(originalScrollX, originalScrollY);
									reject(err);
								});
							};
							script.onerror = () => {
								window.scrollTo(originalScrollX, originalScrollY);
								reject('Failed to load html2canvas library');
							};
							document.head.appendChild(script);
						} else {
							// html2canvas already loaded
							html2canvas(document.body, {
								width: fullWidth,
								height: fullHeight,
								windowWidth: fullWidth,
								windowHeight: fullHeight,
								x: 0,
								y: 0,
								useCORS: true,
								allowTaint: true,
								scale: window.devicePixelRatio || 1,
								logging: false,
								scrollX: 0,
								scrollY: 0
							}).then(canvas => {
								window.scrollTo(originalScrollX, originalScrollY);
								resolve(canvas.toDataURL('image/%s', %f));
							}).catch(err => {
								window.scrollTo(originalScrollX, originalScrollY);
								reject(err);
							});
						}
					});
				`, params.Format, float64(params.Quality)/100, params.Format, float64(params.Quality)/100)
			} else {
				// Capture visible viewport using browser API if available
				jsCode = fmt.Sprintf(`
					// Try to use browser screenshot API if available (requires extension)
					if (window.__brummer_screenshot) {
						return window.__brummer_screenshot({
							format: '%s',
							quality: %d
						});
					}
					
					// Use html2canvas to capture the viewport
					return new Promise((resolve, reject) => {
						// Dynamically load html2canvas if not already loaded
						if (typeof html2canvas === 'undefined') {
							const script = document.createElement('script');
							script.src = 'https://cdnjs.cloudflare.com/ajax/libs/html2canvas/1.4.1/html2canvas.min.js';
							script.onload = () => {
								// Now html2canvas is available
								html2canvas(document.body, {
									width: window.innerWidth,
									height: window.innerHeight,
									x: window.scrollX,
									y: window.scrollY,
									useCORS: true,
									allowTaint: true,
									scale: window.devicePixelRatio || 1,
									logging: false
								}).then(canvas => {
									resolve(canvas.toDataURL('image/%s', %f));
								}).catch(reject);
							};
							script.onerror = () => reject('Failed to load html2canvas library');
							document.head.appendChild(script);
						} else {
							// html2canvas already loaded
							html2canvas(document.body, {
								width: window.innerWidth,
								height: window.innerHeight,
								x: window.scrollX,
								y: window.scrollY,
								useCORS: true,
								allowTaint: true,
								scale: window.devicePixelRatio || 1,
								logging: false
							}).then(canvas => {
								resolve(canvas.toDataURL('image/%s', %f));
							}).catch(reject);
						}
					});
				`, params.Format, params.Quality, params.Format, float64(params.Quality)/100, params.Format, float64(params.Quality)/100)
			}

			// Send screenshot command via WebSocket
			if s.proxyServer != nil {
				s.proxyServer.BroadcastToWebSockets("command", map[string]interface{}{
					"action":     "repl",
					"code":       jsCode,
					"sessionId":  params.SessionID,
					"responseId": responseID,
				})
			} else {
				return map[string]interface{}{
					"error": "proxy server not available",
				}, nil
			}

			// Wait for response with timeout
			select {
			case response := <-responseChan:
				// Check if response contains error
				if respMap, ok := response.(map[string]interface{}); ok {
					if errMsg, ok := respMap["error"].(string); ok {
						// Provide helpful guidance
						return map[string]interface{}{
							"error":      errMsg,
							"suggestion": "Screenshot capture is limited in browser context. Consider using: 1) Browser DevTools (F12 > Elements > right-click > 'Capture node screenshot'), 2) OS screenshot tools (Windows: Win+Shift+S, Mac: Cmd+Shift+4, Linux: varies), or 3) Browser extensions for full-page capture.",
						}, nil
					}
				}
				return response, nil
			case <-time.After(5 * time.Second):
				return map[string]interface{}{
					"error": "timeout waiting for screenshot response",
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
