package mcp

import (
	"encoding/json"
	"strings"
)

// registerResources registers all available MCP resources
func (s *StreamableServer) registerResources() {
	// Log resources
	s.resources["logs://recent"] = Resource{
		URI:         "logs://recent",
		Name:        "Recent Logs",
		Description: "Recent log entries from all processes",
		MimeType:    "application/json",
	}

	s.resources["logs://errors"] = Resource{
		URI:         "logs://errors",
		Name:        "Error Logs",
		Description: "Recent error log entries",
		MimeType:    "application/json",
	}

	// Telemetry resources
	s.resources["telemetry://sessions"] = Resource{
		URI:         "telemetry://sessions",
		Name:        "Telemetry Sessions",
		Description: "Active browser telemetry sessions",
		MimeType:    "application/json",
	}

	s.resources["telemetry://errors"] = Resource{
		URI:         "telemetry://errors",
		Name:        "Browser Errors",
		Description: "JavaScript errors from browser sessions",
		MimeType:    "application/json",
	}

	s.resources["telemetry://console-errors"] = Resource{
		URI:         "telemetry://console-errors",
		Name:        "Console Errors",
		Description: "Console error output (console.error calls) from browser sessions",
		MimeType:    "application/json",
	}

	// Proxy resources
	s.resources["proxy://requests"] = Resource{
		URI:         "proxy://requests",
		Name:        "HTTP Requests",
		Description: "Recent HTTP requests captured by proxy",
		MimeType:    "application/json",
	}

	s.resources["proxy://mappings"] = Resource{
		URI:         "proxy://mappings",
		Name:        "URL Mappings",
		Description: "Active reverse proxy URL mappings",
		MimeType:    "application/json",
	}

	// Process resources
	s.resources["processes://active"] = Resource{
		URI:         "processes://active",
		Name:        "Active Processes",
		Description: "Currently running processes",
		MimeType:    "application/json",
	}

	s.resources["scripts://available"] = Resource{
		URI:         "scripts://available",
		Name:        "Available Scripts",
		Description: "Scripts defined in package.json",
		MimeType:    "application/json",
	}
}

// Resource list handler
func (s *StreamableServer) handleResourcesList(msg *JSONRPCMessage) *JSONRPCMessage {
	resources := make([]map[string]interface{}, 0, len(s.resources))

	for uri, resource := range s.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         uri,
			"name":        resource.Name,
			"description": resource.Description,
			"mimeType":    resource.MimeType,
		})
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"resources": resources,
		},
	}
}

// Resource read handler
func (s *StreamableServer) handleResourceRead(msg *JSONRPCMessage) *JSONRPCMessage {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.createErrorResponse(msg.ID, -32602, "Invalid params", nil)
	}

	resource, ok := s.resources[params.URI]
	if !ok {
		return s.createErrorResponse(msg.ID, -32602, "Resource not found", nil)
	}

	// Get resource content based on URI
	var content interface{}
	var err error

	switch params.URI {
	case "logs://recent":
		content = s.getRecentLogs(100)

	case "logs://errors":
		content = s.getErrorLogs(50)

	case "telemetry://sessions":
		content = s.getTelemetrySessions()

	case "telemetry://errors":
		content = s.getBrowserErrors()

	case "telemetry://console-errors":
		content = s.getConsoleErrors()

	case "proxy://requests":
		content = s.getProxyRequests(100)

	case "proxy://mappings":
		content = s.getProxyMappings()

	case "processes://active":
		content = s.getActiveProcesses()

	case "scripts://available":
		content = s.getAvailableScripts()

	default:
		return s.createErrorResponse(msg.ID, -32603, "Resource handler not implemented", nil)
	}

	if err != nil {
		return s.createErrorResponse(msg.ID, -32603, err.Error(), nil)
	}

	// Convert content to JSON string
	contentBytes, _ := json.Marshal(content)

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      params.URI,
					"mimeType": resource.MimeType,
					"text":     string(contentBytes),
				},
			},
		},
	}
}

// Resource subscribe handler
func (s *StreamableServer) handleResourceSubscribe(msg *JSONRPCMessage, sessionID string) *JSONRPCMessage {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.createErrorResponse(msg.ID, -32602, "Invalid params", nil)
	}

	_, ok := s.resources[params.URI]
	if !ok {
		return s.createErrorResponse(msg.ID, -32602, "Resource not found", nil)
	}

	// Use the session ID passed as parameter

	s.subscriptionsMu.Lock()
	if _, exists := s.subscriptions[sessionID]; !exists {
		s.subscriptions[sessionID] = make(map[string]bool)
	}
	s.subscriptions[sessionID][params.URI] = true
	s.subscriptionsMu.Unlock()

	// Update session's own subscriptions
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if ok {
		session.mu.Lock()
		session.subscriptions[params.URI] = true
		session.mu.Unlock()
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result:  map[string]interface{}{"subscribed": true},
	}
}

// Resource unsubscribe handler
func (s *StreamableServer) handleResourceUnsubscribe(msg *JSONRPCMessage, sessionID string) *JSONRPCMessage {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.createErrorResponse(msg.ID, -32602, "Invalid params", nil)
	}

	// Use the session ID passed as parameter

	s.subscriptionsMu.Lock()
	if subs, exists := s.subscriptions[sessionID]; exists {
		delete(subs, params.URI)
		if len(subs) == 0 {
			delete(s.subscriptions, sessionID)
		}
	}
	s.subscriptionsMu.Unlock()

	// Update session's own subscriptions
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if ok {
		session.mu.Lock()
		delete(session.subscriptions, params.URI)
		session.mu.Unlock()
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result:  map[string]interface{}{"unsubscribed": true},
	}
}

// Resource content getters

func (s *StreamableServer) getRecentLogs(limit int) []interface{} {
	logs := s.logStoreGetAllInterface()
	if len(logs) > limit {
		return logs[len(logs)-limit:]
	}
	return logs
}

func (s *StreamableServer) getErrorLogs(limit int) []interface{} {
	allLogs := s.logStoreGetAllInterface()
	errorLogs := make([]interface{}, 0)

	for i := len(allLogs) - 1; i >= 0 && len(errorLogs) < limit; i-- {
		if logMap, ok := allLogs[i].(map[string]interface{}); ok {
			if priority, ok := logMap["priority"].(int); ok && priority >= 3 {
				errorLogs = append(errorLogs, allLogs[i])
			} else if text, ok := logMap["text"].(string); ok {
				lowerText := strings.ToLower(text)
				if strings.Contains(lowerText, "error") || strings.Contains(lowerText, "fail") {
					errorLogs = append(errorLogs, allLogs[i])
				}
			}
		}
	}

	return errorLogs
}

func (s *StreamableServer) getTelemetrySessions() []interface{} {
	if s.proxyServer == nil || s.proxyServer.GetTelemetryStore() == nil {
		return []interface{}{}
	}

	sessions := make([]interface{}, 0)
	for _, session := range s.proxyServer.GetTelemetryStore().GetAllSessions() {
		sessions = append(sessions, session.GetMetricsSummary())
	}

	return sessions
}

func (s *StreamableServer) getBrowserErrors() []interface{} {
	if s.proxyServer == nil || s.proxyServer.GetTelemetryStore() == nil {
		return []interface{}{}
	}

	errors := make([]interface{}, 0)
	for _, session := range s.proxyServer.GetTelemetryStore().GetAllSessions() {
		for _, event := range session.Events {
			if event.Type == "javascript_error" || event.Type == "unhandled_rejection" {
				errors = append(errors, map[string]interface{}{
					"sessionId": session.SessionID,
					"url":       session.URL,
					"timestamp": event.Timestamp,
					"type":      event.Type,
					"data":      event.Data,
				})
			}
		}
	}

	return errors
}

func (s *StreamableServer) getConsoleErrors() []interface{} {
	if s.proxyServer == nil || s.proxyServer.GetTelemetryStore() == nil {
		return []interface{}{}
	}

	telemetry := s.proxyServer.GetTelemetryStore()
	errors := make([]interface{}, 0)

	// Get all sessions and extract console.error calls
	for _, session := range telemetry.GetAllSessions() {
		for _, event := range session.Events {
			// Check if this is a console output event with error level
			if event.Type == "console_output" {
				if level, ok := event.Data["level"].(string); ok && level == "error" {
					errors = append(errors, map[string]interface{}{
						"sessionId": session.SessionID,
						"url":       session.URL,
						"timestamp": event.Timestamp,
						"type":      "console.error",
						"message":   event.Data["message"],
						"stack":     event.Data["stack"],
					})
				}
			}
		}
	}

	return errors
}

func (s *StreamableServer) getProxyRequests(limit int) []interface{} {
	if s.proxyServer == nil {
		return []interface{}{}
	}

	requests := s.proxyServer.GetRequests()
	result := make([]interface{}, 0, len(requests))

	start := 0
	if len(requests) > limit {
		start = len(requests) - limit
	}

	for i := start; i < len(requests); i++ {
		result = append(result, requests[i])
	}

	return result
}

func (s *StreamableServer) getProxyMappings() []interface{} {
	if s.proxyServer == nil {
		return []interface{}{}
	}

	mappings := s.proxyServer.GetURLMappings()
	result := make([]interface{}, 0, len(mappings))

	for _, mapping := range mappings {
		result = append(result, map[string]interface{}{
			"targetUrl":   mapping.TargetURL,
			"proxyUrl":    mapping.ProxyURL,
			"proxyPort":   mapping.ProxyPort,
			"processName": mapping.ProcessName,
			"createdAt":   mapping.CreatedAt,
		})
	}

	return result
}

func (s *StreamableServer) getActiveProcesses() []interface{} {
	processes := s.processMgr.GetAllProcesses()
	result := make([]interface{}, 0, len(processes))

	for _, p := range processes {
		result = append(result, map[string]interface{}{
			"id":        p.ID,
			"name":      p.Name,
			"script":    p.Script,
			"status":    string(p.Status),
			"startTime": p.StartTime,
			"exitCode":  p.ExitCode,
		})
	}

	return result
}

func (s *StreamableServer) getAvailableScripts() map[string]string {
	return s.processMgr.GetScripts()
}

// Broadcast resource updates
func (s *StreamableServer) broadcastResourceUpdate(uri string) {
	s.BroadcastNotification("notifications/resources/updated", map[string]interface{}{
		"uri": uri,
	})
}
