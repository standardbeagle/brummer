package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// registerPrompts registers all available MCP prompts
func (s *StreamableServer) registerPrompts() {
	// Debug Error prompt
	s.prompts["debug_error"] = Prompt{
		Name:        "Debug Error",
		Description: "Analyze error logs and suggest fixes",
		Arguments: []PromptArgument{
			{
				Name:        "error_message",
				Description: "The error message to debug",
				Required:    true,
			},
			{
				Name:        "context",
				Description: "Additional context about when the error occurred",
				Required:    false,
			},
		},
	}

	// Performance Analysis prompt
	s.prompts["performance_analysis"] = Prompt{
		Name:        "Performance Analysis",
		Description: "Analyze telemetry data for performance issues",
		Arguments: []PromptArgument{
			{
				Name:        "session_id",
				Description: "Telemetry session ID to analyze",
				Required:    false,
			},
			{
				Name:        "metric_type",
				Description: "Specific metric to focus on (memory, load_time, etc)",
				Required:    false,
			},
		},
	}

	// API Troubleshooting prompt
	s.prompts["api_troubleshooting"] = Prompt{
		Name:        "API Troubleshooting",
		Description: "Examine proxy requests to debug API issues",
		Arguments: []PromptArgument{
			{
				Name:        "endpoint",
				Description: "API endpoint pattern to analyze",
				Required:    false,
			},
			{
				Name:        "status_code",
				Description: "Filter by HTTP status code",
				Required:    false,
			},
		},
	}

	// Script Configuration prompt
	s.prompts["script_configuration"] = Prompt{
		Name:        "Script Configuration",
		Description: "Help configure npm scripts for common tasks",
		Arguments: []PromptArgument{
			{
				Name:        "task_type",
				Description: "Type of task (dev, build, test, lint, etc)",
				Required:    true,
			},
			{
				Name:        "framework",
				Description: "Framework being used (react, vue, angular, etc)",
				Required:    false,
			},
		},
	}
}

// Prompt list handler
func (s *StreamableServer) handlePromptsList(msg *JSONRPCMessage) *JSONRPCMessage {
	prompts := make([]map[string]interface{}, 0, len(s.prompts))

	for name, prompt := range s.prompts {
		promptData := map[string]interface{}{
			"name":        name,
			"description": prompt.Description,
		}

		if len(prompt.Arguments) > 0 {
			args := make([]map[string]interface{}, len(prompt.Arguments))
			for i, arg := range prompt.Arguments {
				args[i] = map[string]interface{}{
					"name":        arg.Name,
					"description": arg.Description,
					"required":    arg.Required,
				}
			}
			promptData["arguments"] = args
		}

		prompts = append(prompts, promptData)
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"prompts": prompts,
		},
	}
}

// Prompt get handler
func (s *StreamableServer) handlePromptGet(msg *JSONRPCMessage) *JSONRPCMessage {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.createErrorResponse(msg.ID, -32602, "Invalid params", nil)
	}

	prompt, ok := s.prompts[params.Name]
	if !ok {
		return s.createErrorResponse(msg.ID, -32602, "Prompt not found", nil)
	}

	// Generate prompt messages based on the template and arguments
	messages := s.generatePromptMessages(prompt, params.Arguments)

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"description": prompt.Description,
			"messages":    messages,
		},
	}
}

func (s *StreamableServer) generatePromptMessages(prompt Prompt, args map[string]interface{}) []map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	// Generate system message based on prompt type
	switch prompt.Name {
	case "Debug Error":
		errorMsg, _ := args["error_message"].(string)
		context, _ := args["context"].(string)

		// Get recent error logs
		errorLogs := s.getErrorLogs(20)

		systemMsg := "You are helping debug an error in a development environment. Analyze the error and provide actionable suggestions."
		if context != "" {
			systemMsg += "\n\nContext: " + context
		}

		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": systemMsg,
		})

		userContent := "Error: " + errorMsg + "\n\nRecent error logs:\n"
		for _, log := range errorLogs {
			if logMap, ok := log.(map[string]interface{}); ok {
				userContent += logMap["content"].(string) + "\n"
			}
		}

		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": userContent,
		})

	case "Performance Analysis":
		sessionID, _ := args["session_id"].(string)
		metricType, _ := args["metric_type"].(string)

		systemMsg := "You are analyzing web application performance metrics. Focus on identifying bottlenecks and suggesting optimizations."
		if metricType != "" {
			systemMsg += " Pay special attention to " + metricType + " metrics."
		}

		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": systemMsg,
		})

		// Get telemetry data
		var telemetryData interface{}
		if sessionID != "" {
			if s.proxyServer != nil && s.proxyServer.GetTelemetryStore() != nil {
				session, exists := s.proxyServer.GetTelemetryStore().GetSession(sessionID)
				if exists {
					telemetryData = session.GetMetricsSummary()
				}
			}
		} else {
			telemetryData = s.getTelemetrySessions()
		}

		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": "Analyze the following telemetry data:\n" + jsonString(telemetryData),
		})

	case "API Troubleshooting":
		endpoint, _ := args["endpoint"].(string)
		statusCode, _ := args["status_code"].(string)

		systemMsg := "You are debugging API issues. Analyze HTTP requests and responses to identify problems."
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": systemMsg,
		})

		// Get proxy requests
		requests := s.getProxyRequests(50)

		// Filter if needed
		if endpoint != "" || statusCode != "" {
			filtered := make([]interface{}, 0)
			for _, req := range requests {
				if reqMap, ok := req.(map[string]interface{}); ok {
					include := true
					if endpoint != "" {
						if url, ok := reqMap["URL"].(string); ok && !containsString(url, endpoint) {
							include = false
						}
					}
					if statusCode != "" {
						if code, ok := reqMap["StatusCode"].(int); ok && fmt.Sprintf("%d", code) != statusCode {
							include = false
						}
					}
					if include {
						filtered = append(filtered, req)
					}
				}
			}
			requests = filtered
		}

		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": "Analyze these API requests:\n" + jsonString(requests),
		})

	case "Script Configuration":
		taskType, _ := args["task_type"].(string)
		framework, _ := args["framework"].(string)

		systemMsg := "You are helping configure npm scripts for a " + taskType + " task."
		if framework != "" {
			systemMsg += " The project uses " + framework + "."
		}

		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": systemMsg,
		})

		// Get current scripts
		scripts := s.processMgr.GetScripts()

		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": "Current package.json scripts:\n" + jsonString(scripts) + "\n\nSuggest configuration for " + taskType,
		})
	}

	return messages
}

func jsonString(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > 0 && strings.Contains(str, substr))
}
