package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/pkg/ports"
)

// JSON-RPC 2.0 Message Types
type JSONRPCMessage struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// StreamableServer is the new MCP server with streaming support
type StreamableServer struct {
	mu        sync.RWMutex
	router    *mux.Router
	sessions  map[string]*ClientSession
	tools     map[string]MCPTool
	resources map[string]Resource
	prompts   map[string]Prompt

	// Brummer components
	port        int
	processMgr  *process.Manager
	logStore    *logs.Store
	proxyServer *proxy.Server
	eventBus    *events.EventBus

	// Server info
	serverInfo   ServerInfo
	capabilities ServerCapabilities

	// Message ID counter
	messageID atomic.Int64

	// HTTP server
	server *http.Server

	// WebSocket upgrader
	wsUpgrader websocket.Upgrader
}

type ClientSession struct {
	ID              string
	Context         context.Context
	Cancel          context.CancelFunc
	ResponseWriter  http.ResponseWriter
	Flusher         http.Flusher
	StreamingActive bool
	mu              sync.Mutex
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type LoggingCapability struct {
	// Logging configuration
}

// MCPTool definition with streaming support
type MCPTool struct {
	Name             string
	Description      string
	InputSchema      json.RawMessage
	Handler          func(json.RawMessage) (interface{}, error)
	Streaming        bool
	StreamingHandler func(json.RawMessage, func(interface{})) (interface{}, error)
}

// Resource definition
type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// Prompt definition
type Prompt struct {
	Name        string
	Description string
	Arguments   []PromptArgument
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// NewStreamableServer creates a new MCP server with streaming support
func NewStreamableServer(port int, processMgr *process.Manager, logStore *logs.Store, proxyServer *proxy.Server, eventBus *events.EventBus) *StreamableServer {
	s := &StreamableServer{
		router:      mux.NewRouter(),
		sessions:    make(map[string]*ClientSession),
		tools:       make(map[string]MCPTool),
		resources:   make(map[string]Resource),
		prompts:     make(map[string]Prompt),
		port:        port,
		processMgr:  processMgr,
		logStore:    logStore,
		proxyServer: proxyServer,
		eventBus:    eventBus,
		serverInfo: ServerInfo{
			Name:    "brummer-mcp",
			Version: "2.0.0",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: true,
			},
			Resources: &ResourcesCapability{
				Subscribe:   true,
				ListChanged: true,
			},
			Prompts: &PromptsCapability{
				ListChanged: true,
			},
			Logging: &LoggingCapability{},
		},
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	s.setupRoutes()
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

func (s *StreamableServer) setupRoutes() {
	// Main MCP endpoint
	s.router.HandleFunc("/mcp", s.handleRequest).Methods("POST", "GET")

	// Legacy endpoints for backward compatibility
	s.router.HandleFunc("/mcp/connect", s.handleLegacyConnect).Methods("POST")
	s.router.HandleFunc("/mcp/events", s.handleLegacySSE).Methods("GET")
	s.router.HandleFunc("/mcp/logs", s.handleLegacyGetLogs).Methods("GET")
	s.router.HandleFunc("/mcp/processes", s.handleLegacyGetProcesses).Methods("GET")
	s.router.HandleFunc("/mcp/scripts", s.handleLegacyGetScripts).Methods("GET")
	s.router.HandleFunc("/mcp/execute", s.handleLegacyExecuteScript).Methods("POST")
	s.router.HandleFunc("/mcp/stop", s.handleLegacyStopProcess).Methods("POST")

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *StreamableServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Set appropriate headers
	w.Header().Set("Content-Type", "application/json")

	// Handle GET requests for SSE streaming
	if r.Method == "GET" {
		s.handleStreamingConnection(w, r)
		return
	}

	// Handle POST requests for standard JSON-RPC
	var messages []JSONRPCMessage
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}
	defer r.Body.Close()

	// Try to decode as array first
	if err := json.Unmarshal(body, &messages); err != nil {
		// Try single message
		var msg JSONRPCMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			s.sendError(w, nil, -32700, "Parse error", err.Error())
			return
		}
		messages = []JSONRPCMessage{msg}
	}

	// Process each message
	responses := make([]JSONRPCMessage, 0)
	streaming := false

	for _, msg := range messages {
		response, isStreaming := s.processMessage(&msg, w, r)
		if isStreaming {
			streaming = true
			// For streaming responses, we'll handle them separately
			continue
		}
		if response != nil {
			responses = append(responses, *response)
		}
	}

	// Send non-streaming responses
	if !streaming && len(responses) > 0 {
		if len(responses) == 1 {
			json.NewEncoder(w).Encode(responses[0])
		} else {
			json.NewEncoder(w).Encode(responses)
		}
	}
}

func (s *StreamableServer) handleStreamingConnection(w http.ResponseWriter, r *http.Request) {
	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(r.Context())

	session := &ClientSession{
		ID:              sessionID,
		Context:         ctx,
		Cancel:          cancel,
		ResponseWriter:  w,
		Flusher:         flusher,
		StreamingActive: true,
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		cancel()
	}()

	// Send initial connection event
	s.sendSSEEvent(session, "message", JSONRPCMessage{
		Jsonrpc: "2.0",
		Method:  "connection/established",
		Params:  json.RawMessage(fmt.Sprintf(`{"sessionId":"%s"}`, sessionID)),
	})

	// Set up heartbeat
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Subscribe to events
	eventChan := make(chan events.Event, 100)
	s.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		select {
		case eventChan <- e:
		default:
			// Channel full, skip
		}
	})
	s.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		select {
		case eventChan <- e:
		default:
		}
	})
	s.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		select {
		case eventChan <- e:
		default:
		}
	})

	defer close(eventChan)

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			return

		case <-heartbeat.C:
			s.sendSSEEvent(session, "ping", map[string]string{"timestamp": time.Now().Format(time.RFC3339)})

		case event := <-eventChan:
			// Send event notification
			notification := JSONRPCMessage{
				Jsonrpc: "2.0",
				Method:  fmt.Sprintf("notifications/%s", strings.ReplaceAll(string(event.Type), ".", "/")),
				Params:  mustMarshal(event.Data),
			}
			s.sendSSEEvent(session, "message", notification)
		}
	}
}

func (s *StreamableServer) processMessage(msg *JSONRPCMessage, w http.ResponseWriter, r *http.Request) (*JSONRPCMessage, bool) {
	// Handle different methods
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg), false

	case "tools/list":
		return s.handleToolsList(msg), false

	case "tools/call":
		return s.handleToolCall(msg, w, r)

	case "resources/list":
		return s.handleResourcesList(msg), false

	case "resources/read":
		return s.handleResourceRead(msg), false

	case "resources/subscribe":
		return s.handleResourceSubscribe(msg), false

	case "resources/unsubscribe":
		return s.handleResourceUnsubscribe(msg), false

	case "prompts/list":
		return s.handlePromptsList(msg), false

	case "prompts/get":
		return s.handlePromptGet(msg), false

	default:
		return s.createErrorResponse(msg.ID, -32601, "Method not found", nil), false
	}
}

func (s *StreamableServer) handleInitialize(msg *JSONRPCMessage) *JSONRPCMessage {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    s.capabilities,
		"serverInfo":      s.serverInfo,
	}

	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

func (s *StreamableServer) sendSSEEvent(session *ClientSession, eventType string, data interface{}) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.StreamingActive {
		return fmt.Errorf("session not streaming")
	}

	// Format SSE event
	fmt.Fprintf(session.ResponseWriter, "event: %s\n", eventType)

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(session.ResponseWriter, "data: %s\n\n", dataBytes)
	session.Flusher.Flush()

	return nil
}

func (s *StreamableServer) createErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCMessage {
	return &JSONRPCMessage{
		Jsonrpc: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func (s *StreamableServer) sendError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	response := s.createErrorResponse(id, code, message, data)
	json.NewEncoder(w).Encode(response)
}

// Broadcast notifications to all streaming clients
func (s *StreamableServer) BroadcastNotification(method string, params interface{}) {
	s.mu.RLock()
	sessions := make([]*ClientSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		if session.StreamingActive {
			sessions = append(sessions, session)
		}
	}
	s.mu.RUnlock()

	notification := JSONRPCMessage{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  mustMarshal(params),
	}

	for _, session := range sessions {
		go s.sendSSEEvent(session, "message", notification)
	}
}

func (s *StreamableServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "healthy",
		"sessions": len(s.sessions),
		"mode":     "streamable",
	})
}

// Start starts the MCP server
func (s *StreamableServer) Start() error {
	// Try to find an available port, starting from the requested port
	availablePort, err := ports.FindAvailablePort(s.port)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	
	// Update the port if it changed
	if availablePort != s.port {
		s.port = availablePort
	}
	
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(s.router),
	}

	// MCP Streamable HTTP server starting (logs disabled for TUI compatibility)

	// Set up event broadcasting
	go s.setupEventBroadcasting()

	return s.server.ListenAndServe()
}

// Stop stops the MCP server
func (s *StreamableServer) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}
	return nil
}

// GetPort returns the current port the server is running on
func (s *StreamableServer) GetPort() int {
	return s.port
}

func (s *StreamableServer) setupEventBroadcasting() {
	// Subscribe to relevant events and broadcast them
	s.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		s.BroadcastNotification("notifications/logs/new", e.Data)
	})

	s.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		s.BroadcastNotification("notifications/process/started", e.Data)
	})

	s.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		s.BroadcastNotification("notifications/process/exited", e.Data)
	})

	s.eventBus.Subscribe(events.ErrorDetected, func(e events.Event) {
		s.BroadcastNotification("notifications/error/detected", e.Data)
	})
}

// Helper functions
func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}


// IsRunning returns true if the MCP server is currently running
func (s *StreamableServer) IsRunning() bool {
	return s.server != nil
}
