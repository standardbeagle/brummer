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
		Description: `List all available npm/yarn/pnpm/bun scripts from package.json in the current project.

**When to use:**
- User asks "what scripts are available?" or "what can I run?"
- Before starting any script to see available options
- When exploring a new project to understand its capabilities
- To check script names before using scripts_run

**Workflow context:**
This is typically the first tool you'll use in a development session. It shows all available scripts like 'dev', 'build', 'test', 'lint', etc. Use this information to determine which scripts to run with scripts_run.

**Few-shot examples:**
1. User: "What scripts are available in this project?"
   → Use: scripts_list with {}
   
2. User: "How do I start the development server?"
   → First use: scripts_list to see available scripts
   → Then use: scripts_run with the appropriate script name
   
3. User: "What build commands are available?"
   → Use: scripts_list to show all scripts, then identify build-related ones

**Returns:** Array of script objects with name, command, and description fields.

**Best practices:**
- Always call this before scripts_run if you're unsure about script names
- No parameters needed - it scans the current project automatically
- Works with npm, yarn, pnpm, and bun projects`,
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

**When to use:**
- User wants to start development server: "start the dev server", "run the project" 
- User wants to run tests: "run tests", "test the code"
- User wants to build: "build the project", "create production build"
- User wants to run any npm/yarn/pnpm/bun script
- NEVER use direct bash commands like 'npm run dev' - always use this tool instead

**Workflow context:**
This is the primary tool for starting development processes. It automatically:
- Detects URLs in logs and creates proxy mappings for sharing
- Captures all output for monitoring and debugging  
- Manages process lifecycle (start/stop/restart)
- Handles duplicate script prevention
- Supports real-time log streaming

**Few-shot examples:**
1. User: "Start the development server"
   → Use: scripts_run with {"name": "dev"}
   
2. User: "Run the tests"
   → Use: scripts_run with {"name": "test"}
   
3. User: "Build the project for production" 
   → Use: scripts_run with {"name": "build"}
   
4. User: "Start the backend API"
   → First check scripts_list, then use scripts_run with {"name": "start"} or {"name": "server"}

**Duplicate handling:**
If script is already running, returns current status with:
- Process ID and runtime information
- Proxy URLs if detected
- Commands to stop/restart the process

**Best practices:**
- Always use this instead of direct bash npm/yarn commands
- Check scripts_list first if unsure about script names
- Use scripts_status to check what's currently running
- Monitor logs with logs_stream after starting long-running processes

**Error scenarios:**
- Script doesn't exist → Use scripts_list to see available options
- Process fails to start → Check logs_search for error details
- Already running → Use scripts_stop to stop, then restart`,
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

**When to use:**
- User wants to stop a running process: "stop the dev server", "kill the process"
- Before restarting a script that's already running
- When switching between different development tasks
- To free up ports or resources
- Before starting a different version of the same script

**Workflow context:**
Part of the process lifecycle management. Often used in sequence:
1. scripts_status to see what's running
2. scripts_stop to stop specific processes  
3. scripts_run to start new/different processes

**Few-shot examples:**
1. User: "Stop the development server"
   → First use: scripts_status to get process ID
   → Then use: scripts_stop with {"processId": "dev-1697123456"}
   
2. User: "Kill all running processes"
   → Use: scripts_status to see all processes
   → Use: scripts_stop for each processId
   
3. User: "Restart the server" 
   → Use: scripts_status to get processId
   → Use: scripts_stop with processId
   → Use: scripts_run to start again

**Process ID discovery:**
Get processId from:
- scripts_status output (shows all running processes)
- scripts_run output (when starting a process)
- Error messages that mention already running processes

**Best practices:**
- Always get processId from scripts_status first
- Wait a moment after stopping before restarting
- Use scripts_status to verify the process actually stopped
- Graceful shutdown - lets processes clean up properly

**Error scenarios:**
- Invalid processId → Use scripts_status to get current process IDs
- Process already stopped → No action needed, safe to ignore
- Permission issues → Process may require manual intervention`,
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
		Description: `Check the status of running scripts with detailed process information and proxy URLs.

**When to use:**
- User asks "what's running?", "is the server started?", "show me running processes"
- Before starting a script to avoid duplicates
- To get process IDs for stopping specific scripts
- To see proxy URLs for accessing running services
- For debugging process issues or port conflicts
- To check uptime and process health

**Workflow context:**
Central tool for process monitoring. Used frequently to:
- Check current state before taking actions
- Get process IDs for scripts_stop operations
- Find proxy URLs for browser access
- Monitor process health and uptime

**Few-shot examples:**
1. User: "What's currently running?"
   → Use: scripts_status with {}
   
2. User: "Is the dev server running?"
   → Use: scripts_status with {"name": "dev"}
   
3. User: "How do I access my app?"
   → Use: scripts_status to get proxy URLs for running services
   
4. User: "Stop all processes"
   → First use: scripts_status to get all process IDs
   → Then use: scripts_stop for each process

**Return information:**
- Process ID (needed for scripts_stop)
- Script name and status (running/stopped/failed)
- Start time and uptime duration
- Proxy URLs for web services (if detected)
- Management commands for stopping/restarting

**Parameter options:**
- No parameters: Returns all running processes
- {"name": "scriptname"}: Returns status for specific script only

**Best practices:**
- Use regularly to monitor development environment state
- Check this before scripts_run to avoid duplicate processes
- Use proxy URLs from output to access running web services
- Essential for getting process IDs needed by scripts_stop

**Proxy URL features:**
Automatically shows shareable URLs for detected web services:
- Original URL (e.g., http://localhost:3000)
- Proxy URL (e.g., http://localhost:20888)
- Service labels for easy identification`,
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

**When to use:**
- User wants to monitor live output: "show me the logs", "watch the output", "monitor the server"
- Debugging issues as they happen in real-time
- Monitoring build processes, test runs, or deployments
- Watching for specific errors or events
- Following logs after starting a script with scripts_run

**Workflow context:**
Often used after scripts_run to monitor the started process. Essential for:
- Real-time debugging and troubleshooting
- Monitoring long-running processes like dev servers
- Watching for URL detection and service startup
- Following build or deployment progress

**Few-shot examples:**
1. User: "Show me what's happening in the dev server"
   → Use: logs_stream with {"processId": "dev-1697123456", "follow": true}
   
2. User: "Monitor all processes for errors"
   → Use: logs_stream with {"level": "error", "follow": true}
   
3. User: "Watch the build output"
   → First start build with scripts_run
   → Then use: logs_stream with processId from build process
   
4. User: "Show recent logs then keep watching"
   → Use: logs_stream with {"limit": 50, "follow": true}

**Parameter combinations:**
- {"follow": true} - Stream all new logs from all processes
- {"processId": "abc-123", "follow": true} - Stream from specific process
- {"level": "error", "follow": true} - Only error-level messages
- {"limit": 100, "follow": false} - Get recent logs without streaming

**Streaming behavior:**
- Starts with historical logs (up to limit)
- Then streams new logs in real-time
- Automatically detects URLs and important events
- Streams for up to 5 minutes per session

**Best practices:**
- Use processId filter for focused debugging on specific services
- Use level filter to focus on errors or warnings only
- Combine with scripts_status to get process IDs
- Stream after starting long-running processes to monitor startup

**Log levels:**
- "all" (default) - All log messages
- "error" - Error messages only
- "warn" - Warning messages
- "info" - Informational messages

**Common patterns:**
1. Start script → Get processId → Stream logs for that process
2. See error → Use logs_search to find when it started
3. Use error level filtering during debugging sessions`,
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
		Description: `Search through historical logs using text patterns, regex, and advanced filtering.

**When to use:**
- User reports an error: "I'm getting an error", "something's not working", "find the problem"
- Debugging specific issues: "when did this start failing?", "find connection errors"
- Investigating patterns: "how often does X occur?", "show me all timeouts"
- Historical analysis: "what happened before the crash?", "trace this issue"
- Code debugging: "find logs with 'function_name'", "search for API errors"

**Workflow context:**
Key tool for retrospective debugging. Often used when:
- User reports issues but doesn't know when they started
- Need to trace error patterns over time
- Investigating intermittent problems
- Finding context around specific events

**Few-shot examples:**
1. User: "I'm getting connection errors"
   → Use: logs_search with {"query": "connection", "level": "error"}
   
2. User: "When did the server start failing?"
   → Use: logs_search with {"query": "fail|error|crash", "regex": true}
   
3. User: "Find all 404 errors from today"
   → Use: logs_search with {"query": "404", "since": "2024-01-15T00:00:00Z"}
   
4. User: "Show me what happened before the crash"
   → First search for crash: {"query": "crash|fatal|exit", "regex": true}
   → Then search time before crash with "since" parameter

**Advanced search patterns:**
- Text search: {"query": "connection timeout"}
- Regex search: {"query": "ERROR.*database.*failed", "regex": true}
- Time-bounded: {"query": "error", "since": "2024-01-15T10:00:00Z"}
- Process-specific: {"query": "error", "processId": "dev-123"}
- Combined filters: {"query": "API", "level": "error", "limit": 20}

**Search capabilities:**
- Full-text search across all log messages
- Regular expression patterns for complex matching
- Time-range filtering with RFC3339 timestamps
- Log level filtering (error, warn, info)
- Process-specific filtering
- Result limiting and pagination

**Best practices:**
- Start with simple text search, then refine with regex if needed
- Use level filtering to focus on errors/warnings
- Combine with time ranges for incident investigation
- Use processId filter when debugging specific services
- Check timestamps to understand event sequences

**Common debugging patterns:**
1. Error reported → Search for error keywords → Narrow by time/process
2. Intermittent issue → Regex search for patterns → Check frequency
3. Service failure → Search for process name + error → Find root cause
4. Performance issue → Search for "slow|timeout|delay" → Identify bottlenecks

**Regex examples:**
- "(error|failed|exception)" - Common error patterns
- "\\b\\d{3}\\b" - HTTP status codes  
- "timeout.*connection" - Connection timeout patterns
- "database.*error" - Database-related errors`,
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
		Description: `Get HTTP requests captured by the proxy server with detailed debugging information.

**When to use:**
- User asks about API calls: "what requests is my app making?", "show me the network traffic"
- Debugging API issues: "why is my API failing?", "check the HTTP responses"
- Performance analysis: "which requests are slow?", "find bottlenecks"
- Authentication debugging: "check if auth headers are sent", "verify API tokens"
- CORS or network troubleshooting

**Workflow context:**
Essential for full-stack development debugging. Works with the proxy server to:
- Intercept and analyze all HTTP traffic from your app
- Debug API integration issues
- Monitor third-party service calls
- Analyze request/response patterns

**Few-shot examples:**
1. User: "My API calls aren't working"
   → Use: proxy_requests with {"status": "error"}
   
2. User: "What requests is my frontend making?"
   → Use: proxy_requests with {"processName": "dev", "limit": 20}
   
3. User: "Show me recent network activity"
   → Use: proxy_requests with {"limit": 50}
   
4. User: "Find slow API calls"
   → Use: proxy_requests with {} then analyze response times

**Filter options:**
- {"processName": "dev"} - Requests from specific process (e.g., dev server)
- {"status": "error"} - Only failed requests (4xx, 5xx status codes)
- {"status": "success"} - Only successful requests (2xx, 3xx)
- {"limit": 100} - Limit number of results returned

**Request information includes:**
- HTTP method, URL, and status code
- Request and response headers
- Response time and payload size
- Timestamp and originating process
- Error details for failed requests

**Best practices:**
- Use processName filter to focus on specific services
- Check error status when debugging API issues
- Analyze response times for performance problems
- Review headers for authentication and CORS issues
- Combine with browser tools for complete request lifecycle

**Common debugging scenarios:**
1. API errors → Filter by "error" status → Check error codes and messages
2. Slow performance → Review all requests → Identify high response times
3. Auth issues → Check request headers → Verify tokens and cookies
4. CORS problems → Examine preflight requests → Check response headers

**Integration with other tools:**
- Use after scripts_run to monitor started services
- Combine with browser_open to see full request flow
- Use with telemetry_sessions for complete debugging picture`,
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
		Description: `Get browser telemetry sessions with performance metrics, JavaScript errors, and user interaction data.

**When to use:**
- User reports frontend issues: "my app is slow", "JavaScript errors", "page performance problems"
- Performance monitoring: "how fast is my app?", "check page load times"
- Error debugging: "find client-side errors", "JavaScript console errors"
- User experience analysis: "monitor user interactions", "track page metrics"
- Memory leak investigation: "check memory usage", "find performance bottlenecks"

**Workflow context:**
Provides client-side telemetry data captured when users access your app through the proxy. Essential for:
- Frontend performance optimization
- JavaScript error tracking and debugging
- User experience monitoring
- Resource usage analysis

**Few-shot examples:**
1. User: "My website feels slow"
   → Use: telemetry_sessions with {} to see performance metrics
   
2. User: "Are there JavaScript errors?"
   → Use: telemetry_sessions then check error counts in results
   
3. User: "Show me the latest user session"
   → Use: telemetry_sessions with {"limit": 1}
   
4. User: "Monitor my dev environment performance"
   → Use: telemetry_sessions with {"processName": "dev"}

**Session data includes:**
- Page load and render performance metrics
- JavaScript errors and console messages
- Memory usage and resource consumption
- User interaction events (clicks, navigation)
- Core Web Vitals (LCP, FID, CLS)
- Network timing information

**Filter options:**
- {"processName": "dev"} - Sessions from specific development process
- {"sessionId": "abc-123"} - Get specific session details
- {"limit": 10} - Limit number of sessions returned

**Performance metrics:**
- First Contentful Paint (FCP)
- Largest Contentful Paint (LCP)
- Cumulative Layout Shift (CLS)
- First Input Delay (FID)
- Total Blocking Time (TBT)
- Memory usage and heap size

**Best practices:**
- Monitor sessions regularly during development
- Check error counts and types for debugging
- Analyze performance metrics for optimization
- Use sessionId for detailed investigation of specific issues
- Combine with telemetry_events for real-time monitoring

**Common analysis patterns:**
1. Performance issues → Check LCP, FID, CLS metrics → Identify bottlenecks
2. JavaScript errors → Review error counts → Use telemetry_events for details
3. Memory leaks → Monitor memory usage trends → Identify growing sessions
4. User experience → Analyze interaction timing → Optimize critical paths`,
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
		Description: `Stream real-time browser telemetry events including console logs, errors, performance metrics, and user interactions.

**When to use:**
- Real-time debugging: "watch for JavaScript errors", "monitor console output", "see what's happening in the browser"
- Performance monitoring: "track page performance live", "watch for slow operations"
- User interaction tracking: "monitor user clicks and navigation", "see real-time usage"
- Error detection: "catch errors as they happen", "monitor for exceptions"
- Development workflow: streaming events while testing features

**Workflow context:**
Provides real-time visibility into browser activity. Essential for:
- Live debugging during development
- Monitoring user testing sessions
- Catching intermittent errors
- Performance regression detection

**Few-shot examples:**
1. User: "Watch for JavaScript errors while I test"
   → Use: telemetry_events with {"eventType": "error", "follow": true}
   
2. User: "Monitor console output in real-time"
   → Use: telemetry_events with {"eventType": "console", "follow": true}
   
3. User: "Show me recent browser activity"
   → Use: telemetry_events with {"limit": 50, "follow": false}
   
4. User: "Track performance events for this session"
   → Use: telemetry_events with {"sessionId": "session-123", "eventType": "performance", "follow": true}

**Event types:**
- "console" - Console.log, console.error, console.warn messages
- "error" - JavaScript errors and exceptions
- "performance" - Timing metrics, resource loads, Core Web Vitals
- "interaction" - User clicks, form submissions, navigation
- "all" (default) - All event types

**Streaming parameters:**
- {"follow": true} - Stream new events in real-time (default)
- {"follow": false} - Get historical events only
- {"sessionId": "abc-123"} - Filter to specific browser session
- {"eventType": "error"} - Filter to specific event types
- {"limit": 50} - Number of historical events to include

**Event data structure:**
- Timestamp and event type
- Session and page context
- Detailed event payload (error messages, performance metrics, etc.)
- User agent and browser information

**Best practices:**
- Use eventType filters to focus on specific concerns (errors, performance)
- Stream during active development and testing
- Combine with sessionId when investigating specific user sessions
- Use historical mode for analysis, streaming mode for monitoring

**Real-time debugging workflow:**
1. Start streaming events before testing
2. Filter by error type when debugging issues
3. Use performance events to track optimization
4. Monitor console events for development insights

**Common event patterns:**
- Error debugging: Filter "error" events, watch for exceptions
- Performance tuning: Filter "performance" events, monitor metrics
- User testing: Stream "interaction" events, watch user behavior
- Console monitoring: Filter "console" events for debug output`,
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
		Description: `Open a URL in the default browser with automatic proxy configuration for request monitoring and telemetry.

**When to use:**
- User wants to test their app: "open my app", "test the website", "launch the development server"
- Need to access running services with monitoring enabled
- Testing with proxy for debugging HTTP requests and performance
- Opening detected URLs from running processes
- Cross-platform browser launching (Windows, Mac, Linux, WSL2)

**Workflow context:**
Integrates browser access with development workflow. Automatically:
- Configures proxy for request interception and telemetry
- Creates shareable URLs for team collaboration
- Enables automatic monitoring of frontend performance
- Connects browser sessions to development processes

**Few-shot examples:**
1. User: "Open my development server"
   → First check scripts_status to get proxy URLs
   → Use: browser_open with detected URL from status
   
2. User: "Test my app with monitoring"
   → Use: browser_open with {"url": "http://localhost:3000", "processName": "dev"}
   
3. User: "Launch the website for debugging"
   → Use: browser_open with URL, then monitor with proxy_requests and telemetry_events
   
4. User: "Open my API documentation"
   → Use: browser_open with {"url": "http://localhost:8080/docs"}

**Automatic proxy features:**
- Creates reverse proxy URL for monitoring if proxy server is running
- Registers URL with specified process name for request tracking
- Enables telemetry collection for performance analysis
- Returns both original and proxy URLs for reference

**Cross-platform support:**
- **Windows**: Uses cmd /c start command
- **macOS**: Uses open command
- **Linux**: Uses xdg-open command  
- **WSL2**: Automatically detects and uses Windows commands

**Return information:**
- Original URL that was requested
- Proxy URL for monitoring (if proxy is enabled)
- Success confirmation of browser launch

**Best practices:**
- Use processName to associate browser sessions with development processes
- Combine with proxy_requests to monitor HTTP traffic
- Use telemetry_sessions to track browser performance
- Check scripts_status first to get automatically detected URLs
- Use proxy URLs for team sharing and collaboration

**Integration workflow:**
1. Start development process with scripts_run
2. Check scripts_status for detected URLs
3. Open browser with browser_open
4. Monitor with proxy_requests and telemetry tools

**Common scenarios:**
- Development testing: Open localhost URLs with full monitoring
- Team collaboration: Share proxy URLs with team members
- Performance testing: Open with telemetry for analysis
- API testing: Open documentation or testing interfaces`,
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

**When to use:**
- User makes code changes: "refresh the browser", "reload the page", "see my changes"
- Testing code changes without manual browser switching
- After modifying CSS, JavaScript, or HTML files
- Development workflow automation for rapid iteration
- Force reload after configuration changes

**Workflow context:**
Essential for efficient development workflow. Enables:
- Instant testing of code changes
- Automated refresh after file modifications
- Remote browser control for development
- Seamless transition between coding and testing

**Few-shot examples:**
1. User: "I updated the CSS, show me the changes"
   → Use: browser_refresh with {}
   
2. User: "Refresh the browser to see my updates"
   → Use: browser_refresh with {}
   
3. User: "Reload that specific tab I was testing"
   → Use: browser_refresh with {"sessionId": "session-abc123"}
   
4. User: "Force refresh after changing config"
   → Use: browser_refresh to reload with latest configuration

**Refresh behavior:**
- Sends refresh command via WebSocket to connected browser tabs
- Works with browsers that accessed app through proxy URLs
- Maintains session state and telemetry connections
- Triggers standard browser refresh (F5 equivalent)

**Session targeting:**
- {} (no sessionId) - Refreshes all connected browser tabs
- {"sessionId": "abc-123"} - Refreshes only the specified session

**Best practices:**
- Use after making code changes to see immediate results
- Combine with file watching for automated refresh workflows
- Use specific sessionId when testing with multiple browser tabs
- Essential for rapid development iteration cycles

**Integration patterns:**
1. Edit code → browser_refresh → see changes immediately
2. Deploy changes → browser_refresh → test updated functionality
3. Configuration update → browser_refresh → verify new settings
4. CSS/JS changes → browser_refresh → visual confirmation

**Prerequisites:**
- Browser must be opened via browser_open with proxy enabled
- WebSocket connection established through proxy server
- Browser tabs accessing app through proxy URLs

**Error handling:**
- Returns success confirmation when command is sent
- No error if no browsers are connected (safe operation)
- Works best with modern browsers supporting WebSocket`,
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

**When to use:**
- User wants to test different pages: "go to the about page", "navigate to dashboard", "test the login flow"
- Testing routing and navigation in single-page applications
- Verifying different app sections or user flows
- Remote navigation control during development and testing
- Maintaining monitoring context across page changes

**Workflow context:**
Enables comprehensive app testing while preserving:
- Proxy connection for HTTP request monitoring
- Telemetry collection across navigation
- Session continuity for performance analysis
- Development debugging context

**Few-shot examples:**
1. User: "Test the user profile page"
   → Use: browser_navigate with {"url": "/profile"}
   
2. User: "Go to the admin dashboard"
   → Use: browser_navigate with {"url": "/admin/dashboard"}
   
3. User: "Navigate that testing tab to the checkout page"
   → Use: browser_navigate with {"url": "/checkout", "sessionId": "test-session-123"}
   
4. User: "Test the API documentation page"
   → Use: browser_navigate with {"url": "http://localhost:3000/docs"}

**URL format support:**
- Relative paths: "/about", "/dashboard", "/api/docs"
- Absolute URLs: "http://localhost:3000/page", "https://example.com"
- Query parameters: "/search?q=test&category=all"
- Hash fragments: "/page#section", "/#/spa-route"

**Session targeting:**
- {} - Navigate all connected browser tabs
- {"sessionId": "abc-123"} - Navigate only specified session
- Maintains telemetry and monitoring context across navigation

**Navigation behavior:**
- Sends navigate command via WebSocket to browser tabs
- Preserves proxy configuration and monitoring
- Maintains session state for telemetry collection
- Works with both single-page and multi-page applications

**Best practices:**
- Use relative paths for same-origin navigation
- Test critical user flows by navigating through app sections
- Combine with telemetry_events to monitor navigation performance
- Use specific sessionId when testing with multiple browser instances

**Testing workflows:**
1. Navigation testing: Navigate → Check page load → Verify functionality
2. Performance testing: Navigate → Monitor telemetry → Analyze metrics
3. User flow testing: Sequential navigation through app sections
4. Route testing: Navigate to different SPA routes and verify rendering

**Integration with monitoring:**
- Use proxy_requests to see navigation-triggered API calls
- Use telemetry_events to monitor page performance
- Use telemetry_sessions to analyze navigation patterns
- Combine with browser_refresh for complete testing control

**Error handling:**
- Returns success when navigation command is sent
- Browser handles actual navigation and error states
- Monitor telemetry for navigation success/failure
- Invalid URLs are handled by browser (404, network errors)`,
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

**When to use:**
- User wants to debug JavaScript: "check the value of X", "test this function", "inspect the DOM"
- Interactive testing: "call this API", "modify the page", "test user interactions"
- State inspection: "what's in localStorage?", "check global variables", "inspect component state"
- Real-time development: "try this code", "test before implementing", "quick prototyping"
- Troubleshooting: "why isn't this working?", "debug the frontend", "check console state"

**Workflow context:**
Provides live browser console access for:
- Interactive debugging without switching to browser DevTools
- Real-time code testing and experimentation
- State inspection and manipulation
- API testing and data exploration
- DOM manipulation and UI testing

**Few-shot examples:**
1. User: "What's the current page title?"
   → Use: repl_execute with {"code": "document.title"}
   
2. User: "Check if my function works"
   → Use: repl_execute with {"code": "myFunction('test').then(console.log)"}
   
3. User: "Get data from my API"
   → Use: repl_execute with {"code": "await fetch('/api/users').then(r => r.json())"}
   
4. User: "Check what's in localStorage"
   → Use: repl_execute with {"code": "Object.keys(localStorage)"}
   
5. User: "Test this code snippet"
   → Use: repl_execute with {"code": "const result = myApp.processData(input); return result;"}

**JavaScript capabilities:**
- **Synchronous code**: Direct value returns, function calls, variable access
- **Asynchronous code**: Full async/await support for Promises and API calls
- **Multi-line code**: Complex logic with proper statement separation
- **DOM access**: Full document and window object access
- **Global scope**: Access to all loaded libraries and app globals
- **Error handling**: Returns JavaScript errors for debugging

**Code examples by category:**

*DOM Inspection:*
- "document.querySelector('#app').innerHTML"
- "Array.from(document.querySelectorAll('button')).map(b => b.textContent)"

*API Testing:*
- "await fetch('/api/data').then(r => r.json())"
- "await myApp.api.getUserProfile(123)"

*State Debugging:*
- "window.appState || 'No global state'"
- "localStorage.getItem('user_session')"

*Function Testing:*
- "myFunction('test_input')"
- "Object.keys(window.myLibrary)"

**Session targeting:**
- {} - Execute in most recent browser session
- {"sessionId": "abc-123"} - Execute in specific browser session

**Response format:**
- Returns JavaScript execution result as JSON
- Includes error details for failed executions
- Supports all JSON-serializable return types
- Timeout protection (5 second limit)

**Best practices:**
- Start with simple expressions, then build complexity
- Use await for asynchronous operations
- Return values for inspection (use return statement)
- Test code incrementally for complex debugging
- Use console.log for side effects, return for values

**Advanced patterns:**
1. **Multi-step debugging**: Execute series of commands to isolate issues
2. **State modification**: Change variables to test different scenarios
3. **API exploration**: Test endpoints and examine response structures
4. **Performance testing**: Time operations with performance.now()
5. **Event simulation**: Trigger events to test event handlers

**Error scenarios:**
- JavaScript syntax errors → Returns error details for correction
- Runtime exceptions → Shows stack trace and error message
- Timeout errors → Code took longer than 5 seconds to execute
- No browser connection → Proxy server not available or no sessions

**Integration workflow:**
1. Identify issue or testing need
2. Execute diagnostic code to understand current state
3. Test solutions interactively
4. Implement final solution in codebase
5. Verify with additional REPL testing`,
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

**When to use:**
- User wants visual documentation: "take a screenshot", "capture the current page", "show me what it looks like"
- Visual regression testing: "capture before/after changes", "document the UI state"
- Bug reporting: "screenshot the error", "show the broken layout", "capture the issue"
- Documentation: "screenshot for docs", "capture the interface", "show the feature"
- Design review: "capture the current design", "show the layout"

**Workflow context:**
Essential for:
- Visual debugging and documentation
- Capturing UI states for comparison
- Creating visual records of development progress
- Bug reporting with visual evidence
- Design and layout verification

**Few-shot examples:**
1. User: "Take a screenshot of the current page"
   → Use: browser_screenshot with {"format": "png"}
   
2. User: "Capture the full page including what's below the fold"
   → Use: browser_screenshot with {"fullPage": true, "format": "png"}
   
3. User: "Screenshot just the main content area"
   → Use: browser_screenshot with {"selector": "#main-content", "format": "png"}
   
4. User: "Take a high-quality screenshot for documentation"
   → Use: browser_screenshot with {"format": "png", "fullPage": true}

**Screenshot options:**

*Format options:*
- "png" (default) - Lossless, best for UI screenshots
- "jpeg" - Compressed, good for photos/images
- "webp" - Modern format, good compression

*Capture modes:*
- **Viewport** (default): Visible area only
- **Full page**: Entire page including scrolled content
- **Element**: Specific element by CSS selector

*Quality settings:*
- PNG: No quality setting (lossless)
- JPEG/WebP: Quality 0-100 (default: 90)

**Parameter combinations:**
- {"format": "png"} - Standard viewport screenshot
- {"fullPage": true, "format": "png"} - Full page capture
- {"selector": ".component", "format": "png"} - Element-specific capture
- {"format": "jpeg", "quality": 85} - Compressed screenshot
- {"sessionId": "abc-123", "format": "png"} - Specific browser session

**Technical implementation:**
- Uses html2canvas library for browser-based capture
- Automatically loads html2canvas if not present
- Supports high-DPI displays with proper scaling
- Handles cross-origin content when possible
- Returns base64-encoded image data

**Best practices:**
- Use PNG for UI screenshots (crisp text and graphics)
- Use JPEG for image-heavy content to reduce size
- Use fullPage for complete documentation
- Use selector for focused component screenshots
- Specify sessionId when working with multiple browser tabs

**Limitations and alternatives:**
- **Browser security**: Some content may not be capturable
- **Cross-origin**: External images may not appear
- **Performance**: Full page capture of large pages may be slow

**Alternative approaches when browser_screenshot has limitations:**
1. **Browser DevTools**: F12 → Elements → Right-click → "Capture node screenshot"
2. **OS tools**: Windows (Win+Shift+S), Mac (Cmd+Shift+4), Linux (varies)
3. **Browser extensions**: Full-page screenshot extensions
4. **Manual capture**: Use browser's built-in screenshot features

**Error handling:**
- **Element not found**: Returns error with suggestion to check selector
- **Library load failure**: Provides fallback suggestions
- **Timeout**: 5-second limit for screenshot generation
- **Browser compatibility**: Works best with modern browsers

**Return format:**
- Success: Returns base64-encoded image data URL
- Error: Returns error message with helpful suggestions
- Timeout: Provides alternative screenshot methods

**Integration patterns:**
1. Visual testing: Navigate → Screenshot → Compare changes
2. Bug reporting: Reproduce issue → Screenshot → Document problem
3. Documentation: Set up UI state → Screenshot → Add to docs
4. Design review: Implement changes → Screenshot → Share for feedback`,
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
