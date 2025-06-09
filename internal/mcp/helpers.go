package mcp

import (
	"encoding/json"
	"net/http"
	
	"github.com/beagle/brummer/internal/logs"
)

// Helper methods to convert logs to interface{} for MCP

func (s *StreamableServer) logStoreGetAllInterface() []interface{} {
	entries := s.logStore.GetAll()
	result := make([]interface{}, len(entries))
	for i, entry := range entries {
		result[i] = map[string]interface{}{
			"id":          entry.ID,
			"timestamp":   entry.Timestamp,
			"processId":   entry.ProcessID,
			"processName": entry.ProcessName,
			"content":     entry.Content,
			"level":       entry.Level,
			"isError":     entry.IsError,
			"tags":        entry.Tags,
			"priority":    entry.Priority,
		}
	}
	return result
}

func (s *StreamableServer) logStoreGetByProcessInterface(processID string) []interface{} {
	entries := s.logStore.GetByProcess(processID)
	result := make([]interface{}, len(entries))
	for i, entry := range entries {
		result[i] = map[string]interface{}{
			"id":          entry.ID,
			"timestamp":   entry.Timestamp,
			"processId":   entry.ProcessID,
			"processName": entry.ProcessName,
			"content":     entry.Content,
			"level":       entry.Level,
			"isError":     entry.IsError,
			"tags":        entry.Tags,
			"priority":    entry.Priority,
		}
	}
	return result
}


// Legacy endpoint handlers for backward compatibility

func (s *StreamableServer) handleLegacyConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClientName string `json:"clientName"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clientID := generateID()
	
	response := map[string]interface{}{
		"clientId": clientID,
		"capabilities": []string{
			"logs",
			"processes", 
			"scripts",
			"execute",
			"search",
			"filters",
			"events",
		},
		"endpoints": map[string]string{
			"events": "/mcp/events",
			"logs": "/mcp/logs",
			"processes": "/mcp/processes",
			"scripts": "/mcp/scripts",
			"execute": "/mcp/execute",
			"stop": "/mcp/stop",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StreamableServer) handleLegacySSE(w http.ResponseWriter, r *http.Request) {
	// Redirect to new streaming endpoint
	s.handleStreamingConnection(w, r)
}

func (s *StreamableServer) handleLegacyGetLogs(w http.ResponseWriter, r *http.Request) {
	processID := r.URL.Query().Get("processId")
	priority := r.URL.Query().Get("priority")
	
	var logs []logs.LogEntry
	
	if priority != "" {
		logs = s.logStore.GetHighPriority(30)
	} else if processID != "" {
		logs = s.logStore.GetByProcess(processID)
	} else {
		logs = s.logStore.GetAll()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *StreamableServer) handleLegacyGetProcesses(w http.ResponseWriter, r *http.Request) {
	processes := s.processMgr.GetAllProcesses()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

func (s *StreamableServer) handleLegacyGetScripts(w http.ResponseWriter, r *http.Request) {
	scripts := s.processMgr.GetScripts()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scripts)
}

func (s *StreamableServer) handleLegacyExecuteScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Script string `json:"script"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	process, err := s.processMgr.StartScript(req.Script)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(process)
}

func (s *StreamableServer) handleLegacyStopProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProcessID string `json:"processId"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.processMgr.StopProcess(req.ProcessID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

