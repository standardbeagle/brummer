package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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

// MCPServer is the MCP server with streaming support
type MCPServer struct {
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

	// REPL response handlers
	replMu            sync.RWMutex
	replResponseChans map[string]chan interface{}

	// Resource subscriptions
	subscriptionsMu sync.RWMutex
	subscriptions   map[string]map[string]bool // sessionID -> resource URI -> subscribed

	// Resource update handlers
	updateHandlersMu sync.RWMutex
	updateHandlers   map[string]chan ResourceUpdate // sessionID -> update channel

	// Session tracking for connection events
	seenSessions map[string]bool // Track which sessions we've seen to avoid duplicate connection events

	// Connection manager for hub mode
	connectionManager *ConnectionManager

	// AI Coder manager
	aiCoderManager interface{} // Will be *aicoder.AICoderManager when available

	// Message queue (lock-free implementation)
	messageQueue *MessageQueueLockFree
}

type ClientSession struct {
	ID              string
	Context         context.Context
	Cancel          context.CancelFunc
	ResponseWriter  http.ResponseWriter
	Flusher         http.Flusher
	StreamingActive bool
	mu              sync.Mutex
	subscriptions   map[string]bool // resource URI -> subscribed
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

// ResourceUpdate represents a resource change notification
type ResourceUpdate struct {
	URI      string      `json:"uri"`
	Contents interface{} `json:"contents"`
}

// NewMCPServer creates a new MCP server with streaming support
func NewMCPServer(port int, processMgr *process.Manager, logStore *logs.Store, proxyServer *proxy.Server, eventBus *events.EventBus) *MCPServer {
	s := &MCPServer{
		router:            mux.NewRouter(),
		sessions:          make(map[string]*ClientSession),
		tools:             make(map[string]MCPTool),
		resources:         make(map[string]Resource),
		prompts:           make(map[string]Prompt),
		port:              port,
		processMgr:        processMgr,
		logStore:          logStore,
		proxyServer:       proxyServer,
		eventBus:          eventBus,
		replResponseChans: make(map[string]chan interface{}),
		subscriptions:     make(map[string]map[string]bool),
		updateHandlers:    make(map[string]chan ResourceUpdate),
		seenSessions:      make(map[string]bool),
		messageQueue:      NewMessageQueueLockFree(),
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
	s.setupResourceUpdateHandlers()
	s.setupMessageQueueEventHandlers()

	return s
}

func (s *MCPServer) setupRoutes() {
	// Main MCP endpoint implementing Streamable HTTP Transport
	// - POST with Accept: application/json → Standard JSON-RPC response
	// - POST with Accept: text/event-stream → SSE stream response
	// - GET with Accept: text/event-stream → SSE streaming connection
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

func (s *MCPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Debug logging for incoming requests
	if s.logStore != nil && IsDebugEnabled() {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("📥 MCP Request: %s %s (Accept: %s)", r.Method, r.URL.Path, r.Header.Get("Accept")), false)
	}

	// Check Accept header for proper content negotiation
	acceptHeader := r.Header.Get("Accept")
	// acceptsJSON := strings.Contains(acceptHeader, "application/json") || acceptHeader == "" || acceptHeader == "*/*"
	acceptsSSE := strings.Contains(acceptHeader, "text/event-stream")

	// Handle GET requests for SSE streaming
	if r.Method == "GET" {
		if !acceptsSSE {
			http.Error(w, "GET requests must accept text/event-stream", http.StatusNotAcceptable)
			return
		}
		s.handleStreamingConnection(w, r)
		return
	}

	// Handle POST requests
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if client wants SSE response
	wantsSSE := acceptsSSE && strings.Contains(acceptHeader, "text/event-stream")

	// Set appropriate headers
	if wantsSSE {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
	} else {
		w.Header().Set("Content-Type", "application/json")
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

	// Extract session ID if provided
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Check if this is the first time we've seen this session
	s.mu.Lock()
	isFirstTimeSession := !s.seenSessions[sessionID]
	if isFirstTimeSession {
		s.seenSessions[sessionID] = true
	}
	s.mu.Unlock()

	// Track session for POST requests (if first time seeing this session)
	if isFirstTimeSession {
		connectionType := "HTTP"
		if wantsSSE {
			connectionType = "HTTP+SSE"
		}

		s.eventBus.Publish(events.Event{
			Type: events.MCPConnected,
			Data: map[string]interface{}{
				"sessionId":      sessionID,
				"connectedAt":    time.Now(),
				"clientInfo":     r.Header.Get("User-Agent"),
				"connectionType": connectionType,
				"method":         r.Method,
			},
		})
	}

	// Process messages
	if wantsSSE {
		// Handle SSE response for POST request
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send initial SSE comment
		fmt.Fprintf(w, ": MCP Streamable HTTP Transport\n\n")
		flusher.Flush()

		// Process each message and send responses via SSE
		for _, msg := range messages {
			response, _ := s.processMessage(&msg, w, r, sessionID)
			if response != nil {
				data, _ := json.Marshal(response)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}

		// Keep connection open for potential server-initiated messages
		// (In a real implementation, you might want to handle this differently)
		return
	}

	// Standard JSON response
	responses := make([]JSONRPCMessage, 0)
	for _, msg := range messages {
		response, _ := s.processMessage(&msg, w, r, sessionID)
		if response != nil {
			responses = append(responses, *response)
		}
	}

	// Send JSON responses
	if len(responses) > 0 {
		if len(responses) == 1 {
			json.NewEncoder(w).Encode(responses[0])
		} else {
			json.NewEncoder(w).Encode(responses)
		}
	}
}

func (s *MCPServer) handleStreamingConnection(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers as per MCP spec
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Extract or create session ID from header
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	ctx, cancel := context.WithCancel(r.Context())

	session := &ClientSession{
		ID:              sessionID,
		Context:         ctx,
		Cancel:          cancel,
		ResponseWriter:  w,
		Flusher:         flusher,
		StreamingActive: true,
		subscriptions:   make(map[string]bool),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.seenSessions[sessionID] = true // Mark this session as seen
	s.mu.Unlock()

	// Publish connection event
	s.eventBus.Publish(events.Event{
		Type: events.MCPConnected,
		Data: map[string]interface{}{
			"sessionId":      sessionID,
			"connectedAt":    time.Now(),
			"clientInfo":     r.Header.Get("User-Agent"),
			"connectionType": "SSE",
			"method":         r.Method,
		},
	})

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		cancel()

		// Publish disconnection event
		s.eventBus.Publish(events.Event{
			Type: events.MCPDisconnected,
			Data: map[string]interface{}{
				"sessionId":      sessionID,
				"disconnectedAt": time.Now(),
			},
		})
	}()

	// Send initial SSE comment with session info as per spec
	fmt.Fprintf(w, ": MCP Streamable HTTP Transport\n")
	fmt.Fprintf(w, ": Session-Id: %s\n\n", sessionID)
	flusher.Flush()

	// Set up heartbeat
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Subscribe to events
	eventChan := make(chan events.Event, 100)
	resourceUpdateChan := make(chan ResourceUpdate, 100)

	// Register this session for resource updates
	s.registerResourceUpdateHandler(sessionID, resourceUpdateChan)
	defer s.unregisterResourceUpdateHandler(sessionID)

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
	defer close(resourceUpdateChan)

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

		case update := <-resourceUpdateChan:
			// Send resource update notification
			notification := JSONRPCMessage{
				Jsonrpc: "2.0",
				Method:  "notifications/resources/updated",
				Params:  mustMarshal(update),
			}
			s.sendSSEEvent(session, "message", notification)
		}
	}
}

func (s *MCPServer) processMessage(msg *JSONRPCMessage, w http.ResponseWriter, r *http.Request, sessionID string) (*JSONRPCMessage, bool) {
	startTime := time.Now()
	var response *JSONRPCMessage
	var isStreaming bool

	// Track the activity
	defer func() {
		// Publish MCP activity event
		activityData := map[string]interface{}{
			"sessionId": sessionID,
			"method":    msg.Method,
			"params":    string(msg.Params),
			"duration":  time.Since(startTime),
		}

		if response != nil {
			if response.Error != nil {
				activityData["error"] = response.Error.Message
			} else if response.Result != nil {
				// Marshal result for activity tracking
				if resultBytes, err := json.Marshal(response.Result); err == nil {
					resultStr := string(resultBytes)
					if len(resultStr) > 200 {
						resultStr = resultStr[:197] + "..."
					}
					activityData["response"] = resultStr
				}
			}
		}

		s.eventBus.Publish(events.Event{
			Type: events.MCPActivity,
			Data: activityData,
		})
	}()

	// Log the method being called
	if s.logStore != nil && IsDebugEnabled() {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("🔧 MCP Method: %s (ID: %v)", msg.Method, msg.ID), false)
	}

	// Handle different methods
	switch msg.Method {
	case "initialize":
		response = s.handleInitialize(msg)
		return response, false

	case "tools/list":
		response = s.handleToolsList(msg)
		return response, false

	case "tools/call":
		response, isStreaming = s.handleToolCall(msg, w, r)
		return response, isStreaming

	case "resources/list":
		response = s.handleResourcesList(msg)
		return response, false

	case "resources/read":
		response = s.handleResourceRead(msg)
		return response, false

	case "resources/subscribe":
		response = s.handleResourceSubscribe(msg, sessionID)
		return response, false

	case "resources/unsubscribe":
		response = s.handleResourceUnsubscribe(msg, sessionID)
		return response, false

	case "prompts/list":
		response = s.handlePromptsList(msg)
		return response, false

	case "prompts/get":
		response = s.handlePromptGet(msg)
		return response, false

	default:
		response = s.createErrorResponse(msg.ID, -32601, "Method not found", nil)
		return response, false
	}
}

func (s *MCPServer) handleInitialize(msg *JSONRPCMessage) *JSONRPCMessage {
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

func (s *MCPServer) sendSSEEvent(session *ClientSession, eventType string, data interface{}) error {
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

func (s *MCPServer) createErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCMessage {
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

func (s *MCPServer) sendError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	response := s.createErrorResponse(id, code, message, data)
	json.NewEncoder(w).Encode(response)
}

// Broadcast notifications to all streaming clients
func (s *MCPServer) BroadcastNotification(method string, params interface{}) {
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

func (s *MCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "healthy",
		"sessions": len(s.sessions),
		"mode":     "streamable",
	})
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
	// Log start attempt
	if s.logStore != nil {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("🚀 Starting MCP server on port %d...", s.port), false)
	}

	// Try to find an available port, starting from the requested port
	availablePort, err := ports.FindAvailablePort(s.port)
	if err != nil {
		if s.logStore != nil {
			s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("❌ Failed to find available port: %v", err), true)
		}
		return fmt.Errorf("failed to find available port: %w", err)
	}

	// Update the port if it changed
	if availablePort != s.port {
		if s.logStore != nil {
			s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("⚠️  Port %d unavailable, using port %d instead", s.port, availablePort), false)
		}
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

	// Log successful start
	if s.logStore != nil {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("✅ MCP server started on http://localhost:%d/mcp", s.port), false)
	}

	// Publish system message event for successful start
	if s.eventBus != nil {
		successMsg := fmt.Sprintf("✅ MCP server started on http://localhost:%d/mcp", s.port)
		s.eventBus.Publish(events.Event{
			Type: events.EventType("system.message"),
			Data: map[string]interface{}{
				"level":   "success",
				"context": "MCP Server",
				"message": successMsg,
			},
		})
	}

	// Start the server (blocking call)
	err = s.server.ListenAndServe()
	if err != nil && s.logStore != nil {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("❌ MCP server error: %v", err), true)
	}
	return err
}

// Stop stops the MCP server
func (s *MCPServer) Stop() error {
	// Stop message queue
	if s.messageQueue != nil {
		s.messageQueue.Stop()
	}
	
	if s.server != nil {
		// Use a timeout context to prevent shutdown from hanging indefinitely
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetPort returns the current port the server is running on
func (s *MCPServer) GetPort() int {
	return s.port
}

func (s *MCPServer) setupEventBroadcasting() {
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

	// Subscribe to REPL responses
	s.eventBus.Subscribe(events.EventType("repl.response"), func(e events.Event) {
		if responseID, ok := e.Data["responseId"].(string); ok {
			s.handleREPLResponse(responseID, e.Data)
		}
	})
}

// Helper functions
func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}

// IsRunning returns true if the MCP server is currently running
func (s *MCPServer) IsRunning() bool {
	if s.server == nil {
		return false
	}
	// Check if we can actually connect to the port
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// registerREPLResponse registers a channel to receive REPL response for the given ID
func (s *MCPServer) registerREPLResponse(responseID string) chan interface{} {
	s.replMu.Lock()
	defer s.replMu.Unlock()

	responseChan := make(chan interface{}, 1)
	s.replResponseChans[responseID] = responseChan
	return responseChan
}

// unregisterREPLResponse removes the response channel for the given ID
func (s *MCPServer) unregisterREPLResponse(responseID string) {
	s.replMu.Lock()
	defer s.replMu.Unlock()

	if ch, exists := s.replResponseChans[responseID]; exists {
		close(ch)
		delete(s.replResponseChans, responseID)
	}
}

// handleREPLResponse processes an incoming REPL response from the browser
func (s *MCPServer) handleREPLResponse(responseID string, response interface{}) {
	s.replMu.RLock()
	responseChan, exists := s.replResponseChans[responseID]
	s.replMu.RUnlock()

	if exists && responseChan != nil {
		select {
		case responseChan <- response:
			// Response sent successfully
		default:
			// Channel is full or closed, ignore
		}
	}
}

// registerResourceUpdateHandler registers a channel to receive resource updates for a session
func (s *MCPServer) registerResourceUpdateHandler(sessionID string, ch chan ResourceUpdate) {
	s.updateHandlersMu.Lock()
	s.updateHandlers[sessionID] = ch
	s.updateHandlersMu.Unlock()
}

// unregisterResourceUpdateHandler removes the update handler for a session
func (s *MCPServer) unregisterResourceUpdateHandler(sessionID string) {
	s.updateHandlersMu.Lock()
	delete(s.updateHandlers, sessionID)
	s.updateHandlersMu.Unlock()
}

// notifyResourceUpdate notifies all subscribed sessions about a resource update
func (s *MCPServer) notifyResourceUpdate(uri string, contents interface{}) {
	update := ResourceUpdate{
		URI:      uri,
		Contents: contents,
	}

	s.subscriptionsMu.RLock()
	defer s.subscriptionsMu.RUnlock()

	s.updateHandlersMu.RLock()
	defer s.updateHandlersMu.RUnlock()

	// Check each session's subscriptions
	for sessionID, subs := range s.subscriptions {
		if subs[uri] {
			// Session is subscribed to this resource
			if handler, exists := s.updateHandlers[sessionID]; exists {
				select {
				case handler <- update:
					// Update sent
				default:
					// Channel full, skip
				}
			}
		}
	}
}

// setupResourceUpdateHandlers sets up event listeners to trigger resource updates
func (s *MCPServer) setupResourceUpdateHandlers() {
	// Listen for log events
	s.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		// Update logs resources
		s.notifyResourceUpdate("logs://recent", s.getRecentLogs(100))

		// Check if it's an error log
		if isError, ok := e.Data["isError"].(bool); ok && isError {
			s.notifyResourceUpdate("logs://errors", s.getErrorLogs(100))
		}
	})

	// Listen for process events
	s.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		s.notifyResourceUpdate("processes://active", s.getActiveProcesses())
	})

	s.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		s.notifyResourceUpdate("processes://active", s.getActiveProcesses())
	})

	// For now, we'll update proxy resources periodically or on demand
	// since there's no ProxyRequest event type yet
}

// RegisterTool dynamically registers a tool at runtime
func (s *MCPServer) RegisterTool(name string, tool MCPTool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if tool already exists
	if _, exists := s.tools[name]; exists {
		return fmt.Errorf("tool %s already exists", name)
	}

	s.tools[name] = tool

	// Notify connected clients about tool list change
	s.BroadcastNotification("notifications/tools/list_changed", nil)

	// Log the registration
	if s.logStore != nil {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("🔧 Registered tool: %s", name), false)
	}

	return nil
}

// UnregisterTool removes a tool at runtime
func (s *MCPServer) UnregisterTool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if tool exists
	if _, exists := s.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(s.tools, name)

	// Notify connected clients about tool list change
	s.BroadcastNotification("notifications/tools/list_changed", nil)

	// Log the unregistration
	if s.logStore != nil {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("🔧 Unregistered tool: %s", name), false)
	}

	return nil
}

// RegisterToolsFromInstance registers multiple tools from a connected instance
func (s *MCPServer) RegisterToolsFromInstance(instanceID string, tools []MCPTool) error {
	for _, tool := range tools {
		// Prefix tool name with instance ID to avoid conflicts
		prefixedName := fmt.Sprintf("%s/%s", instanceID, tool.Name)
		tool.Name = prefixedName
		if err := s.RegisterTool(prefixedName, tool); err != nil {
			// If registration fails, unregister any tools we already added
			s.UnregisterToolsFromInstance(instanceID)
			return fmt.Errorf("failed to register tool %s: %w", tool.Name, err)
		}
	}
	return nil
}

// UnregisterToolsFromInstance removes all tools belonging to an instance
func (s *MCPServer) UnregisterToolsFromInstance(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := instanceID + "_"
	var toRemove []string

	// Find all tools with the instance prefix
	for name := range s.tools {
		if strings.HasPrefix(name, prefix) {
			toRemove = append(toRemove, name)
		}
	}

	// Remove the tools
	for _, name := range toRemove {
		delete(s.tools, name)
	}

	if len(toRemove) > 0 {
		// Notify connected clients about tool list change
		s.BroadcastNotification("notifications/tools/list_changed", nil)

		// Log the unregistration
		if s.logStore != nil && IsDebugEnabled() {
			s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("🔧 Unregistered %d tools from instance %s", len(toRemove), instanceID), false)
		}
	}

	return nil
}

// SetConnectionManager sets the connection manager for hub mode
func (s *MCPServer) SetConnectionManager(cm *ConnectionManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectionManager = cm
}

// RegisterResource dynamically registers a resource at runtime
func (s *MCPServer) RegisterResource(uri string, resource Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if resource already exists
	if _, exists := s.resources[uri]; exists {
		return fmt.Errorf("resource %s already exists", uri)
	}

	s.resources[uri] = resource

	// Notify connected clients about resource list change
	s.BroadcastNotification("notifications/resources/list_changed", nil)

	// Log the registration
	if s.logStore != nil && IsDebugEnabled() {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("📚 Registered resource: %s", uri), false)
	}

	return nil
}

// UnregisterResource removes a resource at runtime
func (s *MCPServer) UnregisterResource(uri string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if resource exists
	if _, exists := s.resources[uri]; !exists {
		return fmt.Errorf("resource %s not found", uri)
	}

	delete(s.resources, uri)

	// Notify connected clients about resource list change
	s.BroadcastNotification("notifications/resources/list_changed", nil)

	// Log the unregistration
	if s.logStore != nil && IsDebugEnabled() {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("📚 Unregistered resource: %s", uri), false)
	}

	return nil
}

// RegisterResourcesFromInstance registers multiple resources from a connected instance
func (s *MCPServer) RegisterResourcesFromInstance(instanceID string, resources []ResourceWithHandler) error {
	for _, resource := range resources {
		// Resource URI is already prefixed in ProxyResource
		if err := s.RegisterResource(resource.URI, resource.Resource); err != nil {
			// If registration fails, unregister any resources we already added
			s.UnregisterResourcesFromInstance(instanceID)
			return fmt.Errorf("failed to register resource %s: %w", resource.URI, err)
		}
	}
	return nil
}

// UnregisterResourcesFromInstance removes all resources belonging to an instance
func (s *MCPServer) UnregisterResourcesFromInstance(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := instanceID + "_"
	var toRemove []string

	// Find all resources with the instance prefix
	for uri := range s.resources {
		if strings.HasPrefix(uri, prefix) {
			toRemove = append(toRemove, uri)
		}
	}

	// Remove the resources
	for _, uri := range toRemove {
		delete(s.resources, uri)
	}

	if len(toRemove) > 0 {
		// Notify connected clients about resource list change
		s.BroadcastNotification("notifications/resources/list_changed", nil)

		// Log the unregistration
		if s.logStore != nil && IsDebugEnabled() {
			s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("📚 Unregistered %d resources from instance %s", len(toRemove), instanceID), false)
		}
	}

	return nil
}

// RegisterPrompt dynamically registers a prompt at runtime
func (s *MCPServer) RegisterPrompt(name string, prompt Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if prompt already exists
	if _, exists := s.prompts[name]; exists {
		return fmt.Errorf("prompt %s already exists", name)
	}

	s.prompts[name] = prompt

	// Notify connected clients about prompt list change
	s.BroadcastNotification("notifications/prompts/list_changed", nil)

	// Log the registration
	if s.logStore != nil && IsDebugEnabled() {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("💭 Registered prompt: %s", name), false)
	}

	return nil
}

// UnregisterPrompt removes a prompt at runtime
func (s *MCPServer) UnregisterPrompt(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if prompt exists
	if _, exists := s.prompts[name]; !exists {
		return fmt.Errorf("prompt %s not found", name)
	}

	delete(s.prompts, name)

	// Notify connected clients about prompt list change
	s.BroadcastNotification("notifications/prompts/list_changed", nil)

	// Log the unregistration
	if s.logStore != nil && IsDebugEnabled() {
		s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("💭 Unregistered prompt: %s", name), false)
	}

	return nil
}

// RegisterPromptsFromInstance registers multiple prompts from a connected instance
func (s *MCPServer) RegisterPromptsFromInstance(instanceID string, prompts []PromptWithHandler) error {
	for _, prompt := range prompts {
		// Prompt name is already prefixed in ProxyPrompt
		if err := s.RegisterPrompt(prompt.Name, prompt.Prompt); err != nil {
			// If registration fails, unregister any prompts we already added
			s.UnregisterPromptsFromInstance(instanceID)
			return fmt.Errorf("failed to register prompt %s: %w", prompt.Name, err)
		}
	}
	return nil
}

// UnregisterPromptsFromInstance removes all prompts belonging to an instance
func (s *MCPServer) UnregisterPromptsFromInstance(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := instanceID + "_"
	var toRemove []string

	// Find all prompts with the instance prefix
	for name := range s.prompts {
		if strings.HasPrefix(name, prefix) {
			toRemove = append(toRemove, name)
		}
	}

	// Remove the prompts
	for _, name := range toRemove {
		delete(s.prompts, name)
	}

	if len(toRemove) > 0 {
		// Notify connected clients about prompt list change
		s.BroadcastNotification("notifications/prompts/list_changed", nil)

		// Log the unregistration
		if s.logStore != nil && IsDebugEnabled() {
			s.logStore.Add("mcp-server", "MCP", fmt.Sprintf("💭 Unregistered %d prompts from instance %s", len(toRemove), instanceID), false)
		}
	}

	return nil
}

// SetAICoderManager sets the AI coder manager for the server
func (s *MCPServer) SetAICoderManager(manager interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.aiCoderManager = manager
}

// CallTool is a test helper method to call tools directly
func (s *MCPServer) CallTool(ctx context.Context, toolName string, arguments json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	tool, exists := s.tools[toolName]
	s.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}
	
	if tool.Handler == nil {
		return nil, fmt.Errorf("tool '%s' has no handler", toolName)
	}
	
	return tool.Handler(arguments)
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Mcp-Session-Id")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
