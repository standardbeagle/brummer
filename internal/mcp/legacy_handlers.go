package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
)

// Legacy HTTP handlers for backward compatibility

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

	clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	
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
			"events":     "/legacy/events",
			"logs":       "/legacy/logs",
			"processes":  "/legacy/processes",
			"scripts":    "/legacy/scripts",
			"execute":    "/legacy/execute",
			"stop":       "/legacy/stop",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StreamableServer) handleLegacySSE(w http.ResponseWriter, r *http.Request) {
	// Legacy SSE endpoint - not used in MCP mode
	http.Error(w, "SSE not supported in MCP mode", http.StatusNotImplemented)
}

func (s *StreamableServer) handleLegacyGetLogs(w http.ResponseWriter, r *http.Request) {
	processID := r.URL.Query().Get("processId")
	priority := r.URL.Query().Get("priority")

	var logs []interface{}

	if priority != "" {
		logs = make([]interface{}, 0)
		// Get high priority logs
		for _, log := range s.logStore.GetHighPriority(30) {
			logs = append(logs, s.logEntryToInterface(log))
		}
	} else if processID != "" {
		logs = s.logStoreGetByProcessInterface(processID)
	} else {
		logs = s.logStoreGetAllInterface()
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

// Helper methods for log store interface conversion

func (s *StreamableServer) logStoreGetAllInterface() []interface{} {
	logs := s.logStore.GetAll()
	result := make([]interface{}, len(logs))
	for i, log := range logs {
		result[i] = s.logEntryToInterface(log)
	}
	return result
}

func (s *StreamableServer) logStoreGetByProcessInterface(processID string) []interface{} {
	logs := s.logStore.GetByProcess(processID)
	result := make([]interface{}, len(logs))
	for i, log := range logs {
		result[i] = s.logEntryToInterface(log)
	}
	return result
}

func (s *StreamableServer) logEntryToInterface(entry logs.LogEntry) map[string]interface{} {
	return map[string]interface{}{
		"id":          entry.ID,
		"processId":   entry.ProcessID,
		"processName": entry.ProcessName,
		"content":     entry.Content,
		"timestamp":   entry.Timestamp.Format(time.RFC3339),
		"isError":     entry.IsError,
		"priority":    entry.Priority,
		"level":       entry.Level,
		"tags":        entry.Tags,
	}
}