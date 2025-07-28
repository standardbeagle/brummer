package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

// validateOutputPath validates file output paths to prevent security issues like directory traversal
func validateOutputPath(path string) error {
	if path == "" {
		return nil // Empty path is allowed (no file output)
	}

	// Prevent directory traversal attacks
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed in output file path")
	}

	// Convert to absolute path for validation
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to determine current directory: %w", err)
	}

	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("unable to resolve current directory: %w", err)
	}

	// Ensure output path is within current project directory or a subdirectory
	if !strings.HasPrefix(absPath, cwdAbs) {
		return fmt.Errorf("output file must be within current project directory")
	}

	// Ensure parent directory exists or can be created
	parentDir := filepath.Dir(absPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("unable to create parent directory for output file: %w", err)
	}

	return nil
}

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

	// Utility tools
	s.registerUtilityTools()
}

func (s *StreamableServer) registerScriptTools() {
	// scripts_list - List all available scripts
	s.tools["scripts_list"] = MCPTool{
		Name: "scripts_list",
		Description: `List all available npm/yarn/pnpm/bun scripts from package.json.

Use this to see what scripts are available before running them with scripts_run.

For detailed documentation and examples, use: about tool="scripts_list"`,
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
		Description: `Start a package.json script with full process management, log capture, and URL detection.

Automatically captures output, detects URLs for proxy setup, and handles duplicate prevention.

For detailed documentation and examples, use: about tool="scripts_run"`,
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
		Description: `Stop a running script process gracefully with proper cleanup.

Requires process ID from scripts_status. Use for stopping development servers or freeing up resources.

For detailed documentation and examples, use: about tool="scripts_stop"`,
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
		Description: `Check the status of running scripts with process information and proxy URLs.

Shows what's running, provides process IDs for scripts_stop, and displays proxy URLs for web access.

For detailed documentation and examples, use: about tool="scripts_status"`,
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
		Description: `Stream real-time logs from running processes with filtering and historical context.

Supports filtering by process, log level, and includes file output option for saving large datasets.

For detailed documentation and examples, use: about tool="logs_stream"`,
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
				},
				"output_file": {
					"type": "string",
					"description": "Optional file path to write log data (e.g., 'logs.json', 'debug/stream-logs.json')"
				}
			}
		}`),
		Streaming: true,
		StreamingHandler: func(args json.RawMessage, send func(interface{})) (interface{}, error) {
			var params struct {
				ProcessID  string `json:"processId"`
				Level      string `json:"level"`
				Follow     bool   `json:"follow"`
				Limit      int    `json:"limit"`
				OutputFile string `json:"output_file"`
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
				ProcessID  string `json:"processId"`
				Limit      int    `json:"limit"`
				OutputFile string `json:"output_file"`
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

			result := map[string]interface{}{
				"logs": logs,
			}

			// Write to file if requested
			if params.OutputFile != "" {
				// Validate output path for security
				if err := validateOutputPath(params.OutputFile); err != nil {
					result["error"] = fmt.Sprintf("Invalid output file path: %v", err)
					return result, nil
				}

				logData := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"tool":      "logs_stream",
					"parameters": map[string]interface{}{
						"processId": params.ProcessID,
						"limit":     params.Limit,
					},
					"count": len(logs),
					"logs":  logs,
				}

				data, err := json.MarshalIndent(logData, "", "  ")
				if err != nil {
					result["error"] = fmt.Sprintf("Failed to marshal logs data: %v", err)
					return result, nil
				}

				if err := os.WriteFile(params.OutputFile, data, 0644); err != nil {
					absPath, _ := filepath.Abs(params.OutputFile)
					cwd, _ := os.Getwd()
					result["error"] = fmt.Sprintf("Failed to write logs to file: %v\nFile path: %s\nAbsolute path: %s\nWorking directory: %s\nData size: %d bytes", err, params.OutputFile, absPath, cwd, len(data))
					return result, nil
				}
				result["file_written"] = params.OutputFile
				result["message"] = fmt.Sprintf("Logs written to %s (%d entries)", params.OutputFile, len(logs))
			}

			// For backward compatibility, return logs directly if no file output
			if params.OutputFile == "" {
				return logs, nil
			}

			return result, nil
		},
	}

	// logs_search - Search historical logs
	s.tools["logs_search"] = MCPTool{
		Name: "logs_search",
		Description: `Search through historical logs using text patterns, regex, and advanced filtering.

Supports time-range filtering, level filtering, and file output for saving search results.

For detailed documentation and examples, use: about tool="logs_search"`,
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
				},
				"output_file": {
					"type": "string",
					"description": "Optional file path to write search results (e.g., 'search-results.json', 'debug/logs-search.json')"
				}
			},
			"required": ["query"]
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				Query      string `json:"query"`
				Regex      bool   `json:"regex"`
				Level      string `json:"level"`
				ProcessID  string `json:"processId"`
				Since      string `json:"since"`
				Limit      int    `json:"limit"`
				OutputFile string `json:"output_file"`
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

			result := map[string]interface{}{
				"results": filtered,
			}

			// Write to file if requested
			if params.OutputFile != "" {
				// Validate output path for security
				if err := validateOutputPath(params.OutputFile); err != nil {
					result["error"] = fmt.Sprintf("Invalid output file path: %v", err)
					return result, nil
				}

				searchData := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"tool":      "logs_search",
					"parameters": map[string]interface{}{
						"query":     params.Query,
						"regex":     params.Regex,
						"level":     params.Level,
						"processId": params.ProcessID,
						"since":     params.Since,
						"limit":     params.Limit,
					},
					"count":   len(filtered),
					"results": filtered,
				}

				data, err := json.MarshalIndent(searchData, "", "  ")
				if err != nil {
					result["error"] = fmt.Sprintf("Failed to marshal search results: %v", err)
					return result, nil
				}

				if err := os.WriteFile(params.OutputFile, data, 0644); err != nil {
					absPath, _ := filepath.Abs(params.OutputFile)
					cwd, _ := os.Getwd()
					result["error"] = fmt.Sprintf("Failed to write search results to file: %v\nFile path: %s\nAbsolute path: %s\nWorking directory: %s\nData size: %d bytes", err, params.OutputFile, absPath, cwd, len(data))
					return result, nil
				}
				result["file_written"] = params.OutputFile
				result["message"] = fmt.Sprintf("Search results written to %s (%d entries)", params.OutputFile, len(filtered))
			}

			// For backward compatibility, return results directly if no file output
			if params.OutputFile == "" {
				return filtered, nil
			}

			return result, nil
		},
	}
}

func (s *StreamableServer) registerProxyTools() {
	// proxy_requests - Get HTTP requests
	s.tools["proxy_requests"] = MCPTool{
		Name: "proxy_requests",
		Description: `Get HTTP requests captured by the proxy server with detailed debugging information.

Captures all HTTP traffic with headers, timing, and status. Supports filtering and file output.

For detailed documentation and examples, use: about tool="proxy_requests"`,
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
				},
				"output_file": {
					"type": "string",
					"description": "Optional file path to write request data (e.g., 'requests.json', 'debug/api-requests.json')"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				ProcessName string `json:"processName"`
				Status      string `json:"status"`
				Limit       int    `json:"limit"`
				OutputFile  string `json:"output_file"`
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

			result := map[string]interface{}{
				"requests": requests,
			}

			// Write to file if requested
			if params.OutputFile != "" {
				// Validate output path for security
				if err := validateOutputPath(params.OutputFile); err != nil {
					result["error"] = fmt.Sprintf("Invalid output file path: %v", err)
					return result, nil
				}

				requestData := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"tool":      "proxy_requests",
					"parameters": map[string]interface{}{
						"processName": params.ProcessName,
						"status":      params.Status,
						"limit":       params.Limit,
					},
					"count":    len(requests),
					"requests": requests,
				}

				data, err := json.MarshalIndent(requestData, "", "  ")
				if err != nil {
					result["error"] = fmt.Sprintf("Failed to marshal request data: %v", err)
					return result, nil
				}

				if err := os.WriteFile(params.OutputFile, data, 0644); err != nil {
					absPath, _ := filepath.Abs(params.OutputFile)
					cwd, _ := os.Getwd()
					result["error"] = fmt.Sprintf("Failed to write request data to file: %v\nFile path: %s\nAbsolute path: %s\nWorking directory: %s\nData size: %d bytes", err, params.OutputFile, absPath, cwd, len(data))
					return result, nil
				}
				result["file_written"] = params.OutputFile
				result["message"] = fmt.Sprintf("Request data written to %s (%d entries)", params.OutputFile, len(requests))
			}

			// For backward compatibility, return requests directly if no file output
			if params.OutputFile == "" {
				return requests, nil
			}

			return result, nil
		},
	}

	// telemetry_sessions - Get browser telemetry sessions
	s.tools["telemetry_sessions"] = MCPTool{
		Name: "telemetry_sessions",
		Description: `Get browser telemetry sessions with performance metrics, JavaScript errors, and Core Web Vitals.

Analyzes frontend performance and user experience data. Supports file output for large datasets.

For detailed documentation and examples, use: about tool="telemetry_sessions"`,
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
				},
				"output_file": {
					"type": "string",
					"description": "Optional file path to write telemetry session data (e.g., 'sessions.json', 'debug/telemetry-sessions.json')"
				}
			}
		}`),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var params struct {
				ProcessName string `json:"processName"`
				SessionID   string `json:"sessionId"`
				Limit       int    `json:"limit"`
				OutputFile  string `json:"output_file"`
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

			result := map[string]interface{}{
				"sessions": sessions,
			}

			// Write to file if requested
			if params.OutputFile != "" {
				// Validate output path for security
				if err := validateOutputPath(params.OutputFile); err != nil {
					result["error"] = fmt.Sprintf("Invalid output file path: %v", err)
					return result, nil
				}

				sessionData := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"tool":      "telemetry_sessions",
					"parameters": map[string]interface{}{
						"processName": params.ProcessName,
						"sessionId":   params.SessionID,
						"limit":       params.Limit,
					},
					"count":    len(sessions),
					"sessions": sessions,
				}

				data, err := json.MarshalIndent(sessionData, "", "  ")
				if err != nil {
					result["error"] = fmt.Sprintf("Failed to marshal session data: %v", err)
					return result, nil
				}

				if err := os.WriteFile(params.OutputFile, data, 0644); err != nil {
					absPath, _ := filepath.Abs(params.OutputFile)
					cwd, _ := os.Getwd()
					result["error"] = fmt.Sprintf("Failed to write session data to file: %v\nFile path: %s\nAbsolute path: %s\nWorking directory: %s\nData size: %d bytes", err, params.OutputFile, absPath, cwd, len(data))
					return result, nil
				}
				result["file_written"] = params.OutputFile
				result["message"] = fmt.Sprintf("Session data written to %s (%d entries)", params.OutputFile, len(sessions))
			}

			// For backward compatibility, return sessions directly if no file output
			if params.OutputFile == "" {
				return sessions, nil
			}

			return result, nil
		},
	}

	// telemetry_events - Stream telemetry events
	s.tools["telemetry_events"] = MCPTool{
		Name: "telemetry_events",
		Description: `Stream real-time browser telemetry events including console logs, errors, and performance metrics.

Provides live monitoring of browser activity with event filtering and file output support.

For detailed documentation and examples, use: about tool="telemetry_events"`,
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
				},
				"output_file": {
					"type": "string",
					"description": "Optional file path to write telemetry event data (e.g., 'events.json', 'debug/telemetry-events.json')"
				}
			}
		}`),
		Streaming: true,
		StreamingHandler: func(args json.RawMessage, send func(interface{})) (interface{}, error) {
			var params struct {
				SessionID  string `json:"sessionId"`
				EventType  string `json:"eventType"`
				Follow     bool   `json:"follow"`
				Limit      int    `json:"limit"`
				OutputFile string `json:"output_file"`
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
				SessionID  string `json:"sessionId"`
				Limit      int    `json:"limit"`
				OutputFile string `json:"output_file"`
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

			result := map[string]interface{}{
				"events": events,
			}

			// Write to file if requested
			if params.OutputFile != "" {
				// Validate output path for security
				if err := validateOutputPath(params.OutputFile); err != nil {
					result["error"] = fmt.Sprintf("Invalid output file path: %v", err)
					return result, nil
				}

				eventData := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"tool":      "telemetry_events",
					"parameters": map[string]interface{}{
						"sessionId": params.SessionID,
						"limit":     params.Limit,
					},
					"count":  len(events),
					"events": events,
				}

				data, err := json.MarshalIndent(eventData, "", "  ")
				if err != nil {
					result["error"] = fmt.Sprintf("Failed to marshal event data: %v", err)
					return result, nil
				}

				if err := os.WriteFile(params.OutputFile, data, 0644); err != nil {
					absPath, _ := filepath.Abs(params.OutputFile)
					cwd, _ := os.Getwd()
					result["error"] = fmt.Sprintf("Failed to write event data to file: %v\nFile path: %s\nAbsolute path: %s\nWorking directory: %s\nData size: %d bytes", err, params.OutputFile, absPath, cwd, len(data))
					return result, nil
				}
				result["file_written"] = params.OutputFile
				result["message"] = fmt.Sprintf("Event data written to %s (%d entries)", params.OutputFile, len(events))
			}

			// For backward compatibility, return events directly if no file output
			if params.OutputFile == "" {
				return events, nil
			}

			return result, nil
		},
	}
}

func (s *StreamableServer) registerBrowserTools() {
	// browser_open - Open URL in browser with proxy
	s.tools["browser_open"] = MCPTool{
		Name: "browser_open",
		Description: `Open a URL in the default browser with automatic proxy configuration and telemetry collection.

Automatically configures proxy settings for request monitoring and cross-platform browser support.

For detailed documentation and examples, use: about tool="browser_open"`,
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
		Description: `Send a refresh command to connected browser tabs for immediate code change testing.

Refreshes browser tabs via WebSocket connections. Useful for seeing updates after code changes.

For detailed documentation and examples, use: about tool="browser_refresh"`,
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
		Description: `Navigate browser tabs to different URLs while maintaining proxy monitoring and telemetry.

Allows programmatic navigation to test different routes and pages with continued monitoring.

For detailed documentation and examples, use: about tool="browser_navigate"`,
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
		Description: `Execute JavaScript code in the live browser context for debugging, testing, and interactive development.

Provides browser-based REPL for testing JavaScript and inspecting page state.

For detailed documentation and examples, use: about tool="repl_execute"`,
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
		Description: `Capture screenshots of browser tabs for debugging, documentation, and visual testing.

Supports PNG, JPEG, WebP formats with automatic file saving and base64 output.

For detailed documentation and examples, use: about tool="browser_screenshot"`,
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
				// Note: Using standard string concatenation instead of template literals
				// because Go raw string literals can't contain JavaScript template literals
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
				// Note: JavaScript template literals would be preferred but can't be used
				// inside Go raw string literals due to conflicting backtick usage
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
				// Note: Standard approach used instead of template literals due to Go/JS backtick conflict
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

func (s *StreamableServer) registerUtilityTools() {
	// about - Comprehensive information about Brummer
	s.tools["about"] = MCPTool{
		Name: "about",
		Description: `Get comprehensive information about Brummer and its development workflow capabilities.

Provides detailed overview, use cases, and examples. Supports both console display and markdown file output.

For specific tool information, use: about tool="toolname"`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"output_file": {
					"type": "string",
					"description": "Optional file path to write the about information as markdown (e.g., 'README.md', 'docs/brummer-overview.md')"
				}
			}
		}`),
		Handler: func(arguments json.RawMessage) (interface{}, error) {
			var params struct {
				OutputFile string `json:"output_file"`
			}

			if len(arguments) > 0 {
				if err := json.Unmarshal(arguments, &params); err != nil {
					return nil, fmt.Errorf("invalid arguments: %w", err)
				}
			}

			// Generate comprehensive about content
			aboutContent := generateAboutContent()

			result := map[string]interface{}{
				"content": aboutContent,
			}

			// Write to file if requested
			if params.OutputFile != "" {
				// Validate output path for security
				if err := validateOutputPath(params.OutputFile); err != nil {
					return map[string]interface{}{
						"content": aboutContent,
						"error":   fmt.Sprintf("Invalid output file path: %v", err),
					}, nil
				}

				if err := os.WriteFile(params.OutputFile, []byte(aboutContent), 0644); err != nil {
					absPath, _ := filepath.Abs(params.OutputFile)
					cwd, _ := os.Getwd()
					return map[string]interface{}{
						"content": aboutContent,
						"error":   fmt.Sprintf("Failed to write about information to file: %v\nFile path: %s\nAbsolute path: %s\nWorking directory: %s\nContent size: %d bytes", err, params.OutputFile, absPath, cwd, len(aboutContent)),
					}, nil
				}
				result["file_written"] = params.OutputFile
				result["message"] = fmt.Sprintf("About information written to %s", params.OutputFile)
			}

			return result, nil
		},
	}
}

func generateAboutContent() string {
	return `# Brummer: AI-Enhanced Development Environment Manager

**Brummer** is a comprehensive development environment manager designed specifically to enhance AI-assisted development workflows. It provides intelligent process monitoring, advanced debugging tools, and seamless integration with AI coding assistants like Claude.

## What is Brummer?

Brummer is a terminal-based development orchestrator that transforms how developers work with AI assistants. It automatically monitors your development processes, captures detailed logs and errors, provides browser automation, and exposes everything through a standardized MCP (Model Context Protocol) interface that AI assistants can use to help you debug, optimize, and build better software.

**Core Philosophy**: Bridge the gap between AI assistants and your development environment by providing rich, contextual information about your running processes, errors, network requests, and browser state.

## Key Capabilities

### 🔍 **Intelligent Error Detection & Analysis**
- **Automatic Error Clustering**: Groups related error messages (like multi-line stack traces) using time-based analysis
- **Contextual Error Information**: Captures process context, timing, and related log entries
- **Error Pattern Recognition**: Identifies common error patterns across different frameworks and languages
- **Structured Error Export**: Makes errors easily accessible to AI assistants for analysis and solutions

**Example Workflow:**

    Developer: "My React app is failing to start"
    AI (via Brummer): Uses logs_search to find startup errors, analyzes the stack trace, 
                      identifies missing dependencies, and suggests exact fix commands

### 📋 **Advanced Log Management & Search**
- **Real-time Log Streaming**: Monitor multiple processes simultaneously with filtering
- **Intelligent Log Search**: Regex-powered search across historical logs with context
- **Process-specific Filtering**: Focus on logs from specific services or processes
- **Time-based Analysis**: Find correlations between events across different processes

**Example Workflow:**

    Developer: "Why is my API responding slowly?"
    AI (via Brummer): Uses logs_search to find database timeout patterns, correlates with 
                      proxy_requests to identify slow endpoints, suggests optimization strategies

### 🌐 **Browser Automation & Debugging**
- **Live JavaScript REPL**: Execute code directly in your running application's browser context
- **Request Monitoring**: Capture and analyze all HTTP requests with detailed timing information
- **Browser State Inspection**: Access DOM, localStorage, global variables, and component state
- **Visual Testing**: Take screenshots and monitor visual changes during development

**Example Workflow:**

    Developer: "The login form isn't working properly"
    AI (via Brummer): Uses repl_execute to inspect form state, checks network requests 
                      via proxy_requests, identifies CORS issues, provides fix

### 🚀 **Multi-Project Coordination (Hub Mode)**
- **Instance Discovery**: Automatically detects and coordinates multiple project instances
- **Cross-Service Debugging**: Debug issues that span multiple microservices
- **Unified Tool Access**: Access tools from any project instance through a single interface
- **Distributed Development**: Perfect for microservice architectures and monorepos

**Example Workflow:**

    Developer: "My frontend can't connect to the backend"
    AI (via Brummer): Uses hub_proxy_requests to check API calls from frontend instance,
                      hub_logs_search to find backend errors, identifies port mismatch

## How Brummer Improves AI Development Workflows

### **1. Error Resolution Acceleration**
Traditional workflow:
- Error occurs → Developer copies error to AI → AI suggests generic solutions → Trial and error

**With Brummer:**
- Error occurs → AI automatically accesses full error context → AI provides specific, actionable solutions based on your exact environment

### **2. Real-time Debugging Assistance**
Traditional workflow:
- Issue occurs → Developer describes symptoms → AI asks clarifying questions → Slow back-and-forth

**With Brummer:**
- Issue occurs → AI directly inspects your application state, logs, and network traffic → Immediate, precise diagnosis

### **3. Contextual Code Suggestions**
Traditional workflow:
- AI suggests code → Developer tests manually → Multiple iterations to get it working

**With Brummer:**
- AI suggests code → AI tests directly in your browser via REPL → AI refines based on actual behavior → Working solution faster

### **4. Performance Optimization**
Traditional workflow:
- Performance issue → Developer manually gathers metrics → AI analyzes incomplete data

**With Brummer:**
- Performance issue → AI accesses real-time proxy data, log patterns, and browser metrics → Comprehensive optimization recommendations

## Practical Examples

### **Example 1: Database Connection Issues**

    # Developer notices app hanging
    Developer: "My app seems to be hanging on startup"

    # AI uses Brummer to investigate
    AI: logs_search({"query": "connect|database|timeout", "level": "error"})
    # Finds: "Connection timeout after 5000ms to database localhost:5432"

    AI: "I found a database connection timeout. Your app is trying to connect to 
        PostgreSQL on localhost:5432 but timing out. Is PostgreSQL running?"

    # AI provides specific solution
    AI: scripts_run({"name": "db:start"})  # Starts database if script exists

### **Example 2: API Integration Debugging**

    # API calls failing
    Developer: "My API calls are returning 401 errors"

    # AI investigates using proxy monitoring
    AI: proxy_requests({"status": "error"})
    # Finds requests missing Authorization header

    AI: repl_execute({"code": "localStorage.getItem('authToken')"})
    # Discovers token is null

    AI: "Your API calls are failing because the auth token isn't being stored. 
        The localStorage shows null for 'authToken'. Let me check your login flow..."

### **Example 3: Build Process Optimization**

    # Slow build times
    Developer: "My build is taking forever"

    # AI analyzes build logs and patterns
    AI: logs_search({"query": "webpack|build|compile", "since": "1h"})
    # Identifies large bundle sizes and unused imports

    AI: repl_execute({"code": "webpack.getStats().chunks.map(c => ({name: c.name, size: c.size}))"})
    # Gets actual bundle size data

    AI: "Your build is slow because webpack is processing large unused dependencies. 
        Based on your bundle analysis, consider lazy loading these modules..."

## Integration Examples

### **Claude Desktop Configuration**

    {
      "servers": {
        "brummer": {
          "command": "brum",
          "args": ["--no-tui", "--port", "7777"]
        }
      }
    }

### **Multi-Project Hub Setup**

    # Terminal 1: Frontend
    cd frontend/ && brum dev

    # Terminal 2: Backend  
    cd backend/ && brum -p 7778 start

    # Terminal 3: Hub coordinator
    brum --mcp  # Coordinates both instances

### **Continuous Integration**

    # GitHub Actions example
    - name: Start development environment
      run: brum --no-tui test &
      
    - name: Wait for services
      run: |
        # Use MCP tools to verify services are ready
        curl -X POST http://localhost:7777/mcp \
          -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scripts_status"}}'

## Advanced Features

### **Time-based Error Clustering**
Brummer uses intelligent time-gap analysis to group related error messages:
- **Multi-line stack traces** are automatically assembled into single, coherent errors
- **Related warnings and errors** occurring within time windows are grouped
- **Context preservation** maintains the relationship between different error components

### **Automatic URL Detection & Proxy Management**
- **Smart URL discovery** from process logs (detects "Server running on http://localhost:3000")
- **Automatic proxy setup** for request monitoring without manual configuration
- **Shareable development URLs** for team collaboration and testing

### **Browser Integration**
- **Automatic browser launching** with proxy configuration for monitoring
- **Live JavaScript execution** for testing and debugging without leaving your AI assistant
- **Screenshot capabilities** for visual regression testing and bug reporting

### **Process Lifecycle Management**
- **Intelligent restart detection** prevents duplicate processes
- **Resource cleanup** automatically manages orphaned processes
- **Exit code analysis** provides detailed failure information

## Documentation & Resources

### **Full Documentation**
📖 **GitHub Repository**: [https://github.com/standardbeagle/brummer](https://github.com/standardbeagle/brummer)

### **Quick Start Guides**
- **Installation**: npm install -g @standardbeagle/brummer or go install github.com/standardbeagle/brummer/cmd/brum@latest
- **Basic Usage**: cd your-project && brum dev
- **MCP Integration**: Configure your AI assistant to connect to http://localhost:7777/mcp
- **Hub Mode**: Use brum --mcp for coordinating multiple project instances

### **Community & Support**
- **Issues & Feature Requests**: GitHub Issues
- **Documentation**: README.md and docs/ directory
- **Examples**: examples/ directory with real-world usage patterns

---

**Brummer transforms AI-assisted development from reactive problem-solving to proactive, context-aware development assistance. By providing AI assistants with direct access to your development environment, Brummer enables faster debugging, more accurate solutions, and enhanced development productivity.**

*Generated by Brummer about tool - bridging AI assistants and development environments*`
}
