package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/beagle/brummer/internal/logs"
	"github.com/beagle/brummer/internal/process"
	"github.com/beagle/brummer/pkg/events"
)

type Server struct {
	port       int
	processMgr *process.Manager
	logStore   *logs.Store
	eventBus   *events.EventBus
	
	clients         map[string]*Client
	tokens          map[string]string // token -> clientID mapping
	browserClients  map[string]time.Time // clientID -> last activity
	mu              sync.RWMutex
	
	server     *http.Server
}

type Client struct {
	ID       string
	Name     string
	SSE      chan Event
	Commands chan Command
}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Command struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewServer(port int, processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus) *Server {
	s := &Server{
		port:           port,
		processMgr:     processMgr,
		logStore:       logStore,
		eventBus:       eventBus,
		clients:        make(map[string]*Client),
		tokens:         make(map[string]string),
		browserClients: make(map[string]time.Time),
	}
	
	// Start browser client cleanup routine
	go s.cleanupInactiveBrowsers()
	
	return s
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// MCP protocol endpoints
	mux.HandleFunc("/mcp/connect", s.handleConnect)
	mux.HandleFunc("/mcp/events", s.handleSSE)
	mux.HandleFunc("/mcp/command", s.handleCommand)
	
	// MCP convenience endpoints
	mux.HandleFunc("/mcp/logs", s.handleGetLogs)
	mux.HandleFunc("/mcp/processes", s.handleGetProcesses)
	mux.HandleFunc("/mcp/scripts", s.handleGetScripts)
	mux.HandleFunc("/mcp/execute", s.handleExecuteScript)
	mux.HandleFunc("/mcp/stop", s.handleStopProcess)
	mux.HandleFunc("/mcp/search", s.handleSearchLogs)
	mux.HandleFunc("/mcp/filters", s.handleFilters)
	
	// Browser extension API (not MCP protocol)
	mux.HandleFunc("/api/browser-log", s.handleBrowserLog)
	mux.HandleFunc("/api/ping", s.handlePing)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: corsMiddleware(mux),
	}

	s.subscribeToEvents()
	
	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
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
	client := &Client{
		ID:       clientID,
		Name:     req.ClientName,
		SSE:      make(chan Event, 100),
		Commands: make(chan Command, 10),
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

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
			"browserLog": "/api/browser-log",
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

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("clientId")
	
	s.mu.RLock()
	client, exists := s.clients[clientID]
	s.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case event := <-client.SSE:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			
		case <-r.Context().Done():
			s.mu.Lock()
			delete(s.clients, clientID)
			s.mu.Unlock()
			return
		}
	}
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := s.processCommand(cmd)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) processCommand(cmd Command) Response {
	switch cmd.Method {
	case "getLogs":
		processID, _ := cmd.Params["processId"].(string)
		var logs []logs.LogEntry
		if processID != "" {
			logs = s.logStore.GetByProcess(processID)
		} else {
			logs = s.logStore.GetAll()
		}
		return Response{ID: cmd.ID, Result: logs}
		
	case "getProcesses":
		processes := s.processMgr.GetAllProcesses()
		return Response{ID: cmd.ID, Result: processes}
		
	case "getScripts":
		scripts := s.processMgr.GetScripts()
		return Response{ID: cmd.ID, Result: scripts}
		
	case "executeScript":
		scriptName, _ := cmd.Params["script"].(string)
		if scriptName == "" {
			return Response{ID: cmd.ID, Error: &Error{Code: -32602, Message: "Invalid params"}}
		}
		
		process, err := s.processMgr.StartScript(scriptName)
		if err != nil {
			return Response{ID: cmd.ID, Error: &Error{Code: -32603, Message: err.Error()}}
		}
		
		return Response{ID: cmd.ID, Result: process}
		
	case "stopProcess":
		processID, _ := cmd.Params["processId"].(string)
		if processID == "" {
			return Response{ID: cmd.ID, Error: &Error{Code: -32602, Message: "Invalid params"}}
		}
		
		if err := s.processMgr.StopProcess(processID); err != nil {
			return Response{ID: cmd.ID, Error: &Error{Code: -32603, Message: err.Error()}}
		}
		
		return Response{ID: cmd.ID, Result: map[string]bool{"success": true}}
		
	case "searchLogs":
		query, _ := cmd.Params["query"].(string)
		if query == "" {
			return Response{ID: cmd.ID, Error: &Error{Code: -32602, Message: "Invalid params"}}
		}
		
		results := s.logStore.Search(query)
		return Response{ID: cmd.ID, Result: results}
		
	default:
		return Response{ID: cmd.ID, Error: &Error{Code: -32601, Message: "Method not found"}}
	}
}

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleGetProcesses(w http.ResponseWriter, r *http.Request) {
	processes := s.processMgr.GetAllProcesses()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

func (s *Server) handleGetScripts(w http.ResponseWriter, r *http.Request) {
	scripts := s.processMgr.GetScripts()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scripts)
}

func (s *Server) handleExecuteScript(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleStopProcess(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleSearchLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}

	results := s.logStore.Search(query)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleFilters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filters := s.logStore.GetFilters()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filters)
		
	case http.MethodPost:
		var filter struct {
			Name          string `json:"name"`
			Type          string `json:"type"`
			Pattern       string `json:"pattern"`
			PriorityBoost int    `json:"priorityBoost"`
			CaseSensitive bool   `json:"caseSensitive"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// Add filter logic here
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) subscribeToEvents() {
	s.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		s.broadcast(Event{Type: "process.started", Data: e})
	})
	
	s.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		s.broadcast(Event{Type: "process.exited", Data: e})
	})
	
	s.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		s.broadcast(Event{Type: "log.line", Data: e})
	})
	
	s.eventBus.Subscribe(events.ErrorDetected, func(e events.Event) {
		s.broadcast(Event{Type: "error.detected", Data: e})
	})
	
	s.eventBus.Subscribe(events.BuildEvent, func(e events.Event) {
		s.broadcast(Event{Type: "build.event", Data: e})
	})
	
	s.eventBus.Subscribe(events.TestFailed, func(e events.Event) {
		s.broadcast(Event{Type: "test.failed", Data: e})
	})
	
	s.eventBus.Subscribe(events.TestPassed, func(e events.Event) {
		s.broadcast(Event{Type: "test.passed", Data: e})
	})
}

func (s *Server) broadcast(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, client := range s.clients {
		select {
		case client.SSE <- event:
		default:
			// Client channel is full, skip
		}
	}
}

func (s *Server) handleBrowserLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for bearer token
	authHeader := r.Header.Get("Authorization")
	var clientID string
	
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		s.mu.RLock()
		cid, exists := s.tokens[token]
		s.mu.RUnlock()
		
		if !exists {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		clientID = cid
		// Track browser activity
		s.mu.Lock()
		s.browserClients[clientID] = time.Now()
		s.mu.Unlock()
	}

	var req struct {
		ClientID string `json:"clientId"`
		LogData  struct {
			Type      string                 `json:"type"`
			Level     string                 `json:"level"`
			Message   string                 `json:"message"`
			Details   map[string]interface{} `json:"details"`
			URL       string                 `json:"url"`
			Timestamp string                 `json:"timestamp"`
			Source    string                 `json:"source"`
			Tab       struct {
				ID    int    `json:"id"`
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"tab"`
		} `json:"logData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use clientID from bearer token if available, otherwise from request
	if clientID == "" {
		clientID = req.ClientID
	}
	
	// Validate client exists
	if clientID != "" {
		s.mu.RLock()
		_, exists := s.clients[clientID]
		s.mu.RUnlock()

		if !exists {
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}
	}

	// Format browser log for Brummer's log system
	processName := "[browser]"
	if req.LogData.Tab.ID != 0 {
		processName = fmt.Sprintf("[browser:%d]", req.LogData.Tab.ID)
	}
	
	// Create a formatted log message
	var logMessage string
	switch req.LogData.Type {
	case "console":
		logMessage = fmt.Sprintf("[%s] %s", req.LogData.Level, req.LogData.Message)
	case "javascript-error":
		if details := req.LogData.Details; details != nil {
			if filename, ok := details["filename"].(string); ok && filename != "" {
				logMessage = fmt.Sprintf("JS Error: %s (%s:%v:%v)", 
					req.LogData.Message, filename, details["lineno"], details["colno"])
			} else {
				logMessage = fmt.Sprintf("JS Error: %s", req.LogData.Message)
			}
		} else {
			logMessage = fmt.Sprintf("JS Error: %s", req.LogData.Message)
		}
	case "promise-rejection":
		logMessage = fmt.Sprintf("Promise Rejection: %s", req.LogData.Message)
	case "resource-error":
		logMessage = fmt.Sprintf("Resource Error: %s", req.LogData.Message)
	case "network-request":
		if details := req.LogData.Details; details != nil {
			status := ""
			if s, ok := details["status"].(float64); ok {
				status = fmt.Sprintf(" (%d)", int(s))
			}
			duration := ""
			if d, ok := details["duration"].(float64); ok {
				duration = fmt.Sprintf(" %dms", int(d))
			}
			logMessage = fmt.Sprintf("Network: %s%s%s", req.LogData.Message, status, duration)
		} else {
			logMessage = fmt.Sprintf("Network: %s", req.LogData.Message)
		}
	case "network-error":
		logMessage = fmt.Sprintf("Network Error: %s", req.LogData.Message)
	case "navigation":
		logMessage = fmt.Sprintf("Navigation: %s", req.LogData.Message)
	default:
		logMessage = fmt.Sprintf("[%s] %s", req.LogData.Type, req.LogData.Message)
	}

	// Add browser context to message
	if req.LogData.Tab.Title != "" {
		logMessage = fmt.Sprintf("%s | Page: %s (%s)", logMessage, req.LogData.Tab.Title, req.LogData.Tab.URL)
	} else {
		logMessage = fmt.Sprintf("%s | Page: %s", logMessage, req.LogData.Tab.URL)
	}

	// Determine if it's an error
	isError := req.LogData.Level == "error" || 
		       req.LogData.Type == "javascript-error" || 
		       req.LogData.Type == "promise-rejection" || 
		       req.LogData.Type == "resource-error" || 
		       req.LogData.Type == "network-error"

	// Add to log store
	s.logStore.Add("browser", processName, logMessage, isError)

	// Broadcast event
	s.broadcast(Event{
		Type: "log.line",
		Data: map[string]interface{}{
			"processId":   "browser",
			"processName": processName,
			"content":     logMessage,
			"isError":     isError,
			"timestamp":   req.LogData.Timestamp,
			"source":      req.LogData.Source,
			"type":        req.LogData.Type,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// GenerateURLToken creates a token for a specific URL that will be opened
func (s *Server) GenerateURLToken(processName string) string {
	token := fmt.Sprintf("bt_%s_%d", processName, time.Now().UnixNano())
	
	// Create a temporary client for this browser tab
	clientID := generateID()
	client := &Client{
		ID:       clientID,
		Name:     fmt.Sprintf("Browser Tab (%s)", processName),
		SSE:      make(chan Event, 100),
		Commands: make(chan Command, 10),
	}
	
	s.mu.Lock()
	s.clients[clientID] = client
	s.tokens[token] = clientID
	s.mu.Unlock()
	
	return token
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.port
}

// HasActiveBrowsers returns true if any browser clients are currently active
func (s *Server) HasActiveBrowsers() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	cutoff := time.Now().Add(-30 * time.Second) // Consider active if seen in last 30 seconds
	for _, lastSeen := range s.browserClients {
		if lastSeen.After(cutoff) {
			return true
		}
	}
	return false
}

// cleanupInactiveBrowsers removes browser clients that haven't been active
func (s *Server) cleanupInactiveBrowsers() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-60 * time.Second) // Remove after 60 seconds of inactivity
		for clientID, lastSeen := range s.browserClients {
			if lastSeen.Before(cutoff) {
				delete(s.browserClients, clientID)
				// Also cleanup associated client and token
				if token := s.findTokenForClient(clientID); token != "" {
					delete(s.tokens, token)
				}
				delete(s.clients, clientID)
			}
		}
		s.mu.Unlock()
	}
}

// findTokenForClient finds the token associated with a client ID
func (s *Server) findTokenForClient(clientID string) string {
	for token, cid := range s.tokens {
		if cid == clientID {
			return token
		}
	}
	return ""
}

// handlePing handles ping requests from browser extensions
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for bearer token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	s.mu.RLock()
	clientID, exists := s.tokens[token]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Update browser activity
	s.mu.Lock()
	s.browserClients[clientID] = time.Now()
	s.mu.Unlock()

	// Parse request body
	var req struct {
		Timestamp string `json:"timestamp"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Send pong response
	response := map[string]interface{}{
		"status":    "pong",
		"timestamp": time.Now().Format(time.RFC3339),
		"clientId":  clientID,
		"active":    true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}