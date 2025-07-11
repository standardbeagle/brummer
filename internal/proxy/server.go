package proxy

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/gorilla/websocket"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/pkg/ports"
)

// Request represents a proxied HTTP request with its metadata
type Request struct {
	ID          string
	Method      string
	URL         string
	Host        string
	Path        string
	StatusCode  int
	StartTime   time.Time
	Duration    time.Duration
	Size        int64
	Error       string
	ProcessName string

	// Telemetry data
	SessionID    string
	HasTelemetry bool
	Telemetry    *PageSession // Link to telemetry session if available

	// Authentication data
	HasAuth   bool                   // True if request has Authorization header
	AuthType  string                 // Type of auth (Bearer, Basic, etc.)
	JWTClaims map[string]interface{} // Decoded JWT claims if present
	JWTError  string                 // JWT decoding error if any

	// Error tracking
	IsError bool // True if status code is 4xx or 5xx

	// Request type
	IsXHR       bool   // True if X-Requested-With: XMLHttpRequest header present
	ContentType string // Response Content-Type header
}

// ProxyMode defines the proxy operation mode
type ProxyMode string

const (
	ProxyModeFull    ProxyMode = "full"    // Traditional HTTP proxy
	ProxyModeReverse ProxyMode = "reverse" // Reverse proxy for detected URLs
)

// URLMapping represents a reverse proxy mapping
type URLMapping struct {
	TargetURL    string // e.g., "http://localhost:3000"
	ProxyPort    int    // e.g., 8889
	ProxyURL     string // e.g., "http://localhost:8889"
	ProcessName  string
	Label        string // e.g., "Frontend", "API", extracted from log context
	CreatedAt    time.Time
	Server       *http.Server           // The HTTP server for this mapping
	ReverseProxy *httputil.ReverseProxy // The reverse proxy instance for this mapping
}

//go:embed monitor.js
var monitoringScript string

// Server manages the HTTP proxy server
type Server struct {
	port     int
	mode     ProxyMode
	proxy    *goproxy.ProxyHttpServer
	server   *http.Server
	eventBus *events.EventBus

	mu       sync.RWMutex
	requests []Request
	urlMap   map[string]string // Maps URL to process name

	// Reverse proxy specific fields
	urlMappings map[string]*URLMapping // Maps target URL to mapping
	nextPort    int                    // Next available port for reverse proxy
	basePort    int                    // Base port for reverse proxy mode

	// Telemetry
	telemetry       *TelemetryStore
	enableTelemetry bool

	// WebSocket connections for real-time telemetry
	wsUpgrader websocket.Upgrader
	wsClients  map[*websocket.Conn]bool
	// Note: wsClients now protected by main mu mutex to prevent deadlocks

	running bool
}

// createSilentLogger creates a logger that discards all output to prevent
// HTTP server errors from appearing in stdout/stderr during TUI mode
func (s *Server) createSilentLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

// decodeJWT attempts to decode a JWT token without verification
// This is for display purposes only - we're not validating signatures
func decodeJWT(tokenString string) (map[string]interface{}, error) {
	// JWT has 3 parts separated by dots
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode the payload (second part)
	payload := parts[1]

	// Add padding if necessary
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	// Decode base64
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	// Parse JSON
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %v", err)
	}

	return claims, nil
}

// extractAuthInfo extracts authentication information from request headers
func extractAuthInfo(r *http.Request) (hasAuth bool, authType string, jwtClaims map[string]interface{}, jwtError string) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false, "", nil, ""
	}

	hasAuth = true

	// Parse auth type and token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) < 2 {
		authType = "Unknown"
		return
	}

	authType = parts[0]
	token := parts[1]

	// If it's a Bearer token, try to decode as JWT
	if strings.EqualFold(authType, "Bearer") {
		claims, err := decodeJWT(token)
		if err != nil {
			jwtError = err.Error()
		} else {
			jwtClaims = claims
		}
	}

	return
}

// NewServer creates a new proxy server
func NewServer(port int, eventBus *events.EventBus) *Server {
	return NewServerWithMode(port, ProxyModeFull, eventBus)
}

// NewServerWithMode creates a new proxy server with specified mode
func NewServerWithMode(port int, mode ProxyMode, eventBus *events.EventBus) *Server {
	s := &Server{
		port:            port,
		mode:            mode,
		eventBus:        eventBus,
		requests:        make([]Request, 0, 1000),
		urlMap:          make(map[string]string),
		urlMappings:     make(map[string]*URLMapping),
		basePort:        port,
		nextPort:        port + 1000, // Start allocating from port+1000 for reverse proxy URLs
		telemetry:       NewTelemetryStore(),
		enableTelemetry: true,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		wsClients: make(map[*websocket.Conn]bool),
	}

	if mode == ProxyModeFull {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = false
		proxy.Logger = s.createSilentLogger()
		s.proxy = proxy
		s.setupHandlers()
	}

	return s
}

// setupHandlers configures the proxy request/response handlers
func (s *Server) setupHandlers() {
	// Handle requests
	s.proxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		startTime := time.Now()

		// Generate request ID
		reqID := fmt.Sprintf("%d", ctx.Session)

		// Get the process name for this URL
		processName := s.getProcessForURL(r.URL.String())

		// Extract authentication info
		hasAuth, authType, jwtClaims, jwtError := extractAuthInfo(r)

		// Store request info in context
		ctx.UserData = &Request{
			ID:          reqID,
			Method:      r.Method,
			URL:         r.URL.String(),
			Host:        r.Host,
			Path:        r.URL.Path,
			StartTime:   startTime,
			ProcessName: processName,
			HasAuth:     hasAuth,
			AuthType:    authType,
			JWTClaims:   jwtClaims,
			JWTError:    jwtError,
			IsXHR:       r.Header.Get("X-Requested-With") == "XMLHttpRequest",
		}

		return r, nil
	})

	// Handle responses
	s.proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if ctx.UserData != nil {
			req := ctx.UserData.(*Request)
			req.Duration = time.Since(req.StartTime)
			req.StatusCode = resp.StatusCode
			req.ContentType = resp.Header.Get("Content-Type")

			// Check if this is an error response
			req.IsError = resp.StatusCode >= 400

			// Get response size
			if resp.ContentLength > 0 {
				req.Size = resp.ContentLength
			}

			// Store the request
			s.addRequest(*req)

			// Publish event
			s.eventBus.Publish(events.Event{
				Type:      events.EventType("proxy.request"),
				ProcessID: req.ProcessName,
				Data: map[string]interface{}{
					"method":      req.Method,
					"url":         req.URL,
					"status":      req.StatusCode,
					"duration":    req.Duration.Milliseconds(),
					"size":        req.Size,
					"processName": req.ProcessName,
				},
			})

			// Inject monitoring script into HTML responses
			if s.enableTelemetry && resp != nil && resp.StatusCode == 200 {
				resp = s.injectMonitoringScript(resp, req.ProcessName)
			}
		}

		return resp
	})

	// Handle errors
	s.proxy.OnResponse(goproxy.StatusCodeIs(0)).DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if ctx.UserData != nil {
			req := ctx.UserData.(*Request)
			req.Duration = time.Since(req.StartTime)
			req.Error = "Connection failed"

			// Store the failed request
			s.addRequest(*req)
		}

		return resp
	})
}

// injectMonitoringScript injects the monitoring JavaScript into HTML responses
func (s *Server) injectMonitoringScript(resp *http.Response, processName string) *http.Response {
	// Check if response is HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return resp
	}

	// Check if this is an AJAX/fetch request - skip injection for those
	if s.isBackgroundRequest(resp.Request) {
		return resp
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp
	}
	resp.Body.Close()

	// Decompress if needed
	encoding := resp.Header.Get("Content-Encoding")
	if encoding == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))
			return resp
		}
		body, err = io.ReadAll(reader)
		reader.Close()
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))
			return resp
		}
	}

	// Convert body to string for manipulation
	bodyStr := string(body)

	// Check if script is already injected
	if strings.Contains(bodyStr, "<!-- Brummer Monitoring Script -->") {
		// Script already injected, return response as-is
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	// Create the injection script with metadata
	injectionScript := fmt.Sprintf(`
<!-- Brummer Monitoring Script -->
<script>
// Set process name and proxy host for telemetry
window.__brummerProcessName = '%s';
window.__brummerProxyHost = 'localhost:%d';
</script>
<script>
%s
</script>
<!-- End Brummer Monitoring Script -->
`, processName, s.port, monitoringScript)

	// Try to inject before </body> or </html>
	injected := false
	for _, tag := range []string{"</body>", "</html>"} {
		if idx := strings.LastIndex(strings.ToLower(bodyStr), tag); idx != -1 {
			bodyStr = bodyStr[:idx] + injectionScript + bodyStr[idx:]
			injected = true
			break
		}
	}

	// If no suitable tag found, append to the end
	if !injected {
		bodyStr += injectionScript
	}

	// Update body
	newBody := []byte(bodyStr)

	// Re-compress if needed
	if encoding == "gzip" {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, err = gw.Write(newBody)
		gw.Close()
		if err == nil {
			newBody = buf.Bytes()
		}
	}

	// Update response
	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))

	// Remove content security policy that might block our script
	resp.Header.Del("Content-Security-Policy")
	resp.Header.Del("X-Content-Security-Policy")

	return resp
}

// injectMonitoringScriptForMapping injects telemetry script using mapping-specific port
func (s *Server) injectMonitoringScriptForMapping(resp *http.Response, mapping *URLMapping, req *http.Request) *http.Response {
	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return resp
	}

	// Skip injection for XHR/AJAX requests
	if req != nil {
		// Check for common AJAX headers
		if req.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			return resp
		}
		// Check for Fetch API requests
		if req.Header.Get("Sec-Fetch-Mode") == "cors" || req.Header.Get("Sec-Fetch-Dest") == "empty" {
			return resp
		}
		// Check Accept header for non-HTML requests
		accept := req.Header.Get("Accept")
		if accept != "" && !strings.Contains(accept, "text/html") && !strings.Contains(accept, "*/*") {
			return resp
		}
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp
	}
	resp.Body.Close()

	// Decompress if needed
	encoding := resp.Header.Get("Content-Encoding")
	if encoding == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))
			return resp
		}
		body, err = io.ReadAll(reader)
		reader.Close()
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))
			return resp
		}
	}

	// Convert body to string for manipulation
	bodyStr := string(body)

	// Check if script is already injected
	if strings.Contains(bodyStr, "<!-- Brummer Monitoring Script -->") {
		// Script already injected, return response as-is
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	// Create the injection script with metadata - use mapping's proxy port, not control port
	injectionScript := fmt.Sprintf(`
<!-- Brummer Monitoring Script -->
<script>
// Set process name and proxy host for telemetry
window.__brummerProcessName = '%s';
window.__brummerProxyHost = 'localhost:%d';
</script>
<script>
%s
</script>
<!-- End Brummer Monitoring Script -->
`, mapping.ProcessName, mapping.ProxyPort, monitoringScript)

	// Try to inject before </body> or </html>
	injected := false
	for _, tag := range []string{"</body>", "</html>"} {
		if idx := strings.LastIndex(strings.ToLower(bodyStr), tag); idx != -1 {
			bodyStr = bodyStr[:idx] + injectionScript + bodyStr[idx:]
			injected = true
			break
		}
	}

	// If no suitable tag found, append to the end
	if !injected {
		bodyStr += injectionScript
	}

	// Update body
	newBody := []byte(bodyStr)

	// Re-compress if needed
	if encoding == "gzip" {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, err = gw.Write(newBody)
		gw.Close()
		if err == nil {
			newBody = buf.Bytes()
		}
	}

	// Update response
	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))

	// Remove content security policy that might block our script
	resp.Header.Del("Content-Security-Policy")
	resp.Header.Del("X-Content-Security-Policy")

	return resp
}

// rewriteURLsInResponse rewrites URLs in HTML responses to use the proxy
func (s *Server) rewriteURLsInResponse(resp *http.Response, mapping *URLMapping) *http.Response {
	// Check if response is HTML - be more flexible with content type detection
	contentType := resp.Header.Get("Content-Type")
	contentTypeLower := strings.ToLower(contentType)
	isHTML := strings.Contains(contentTypeLower, "text/html") ||
		strings.Contains(contentTypeLower, "text/plain") || // Some servers serve HTML as text/plain
		contentType == "" // If no content type, assume it might be HTML

	if !isHTML {
		// Log for debugging - this might be why rewriting isn't happening
		// URL rewriting skipped for non-HTML content
		return resp
	}

	// URL rewriting in progress

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp
	}
	resp.Body.Close()

	// Handle gzip encoding
	encoding := resp.Header.Get("Content-Encoding")
	if encoding == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			// Not gzip encoded despite header, use as-is
		} else {
			body, err = io.ReadAll(reader)
			reader.Close()
			if err != nil {
				return resp
			}
		}
	}

	// Convert body to string for manipulation
	bodyStr := string(body)

	// Parse target URL to extract domain and port
	targetURL, err := url.Parse(mapping.TargetURL)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	// Extract host (domain:port) from target URL - handles all URL forms including user:pass@domain:port
	targetHost := targetURL.Host
	proxyHost := fmt.Sprintf("localhost:%d", mapping.ProxyPort)

	// Replacing host in HTML content

	// Simple host replacement - works for all URL variations
	if targetHost != "" {
		bodyStr = strings.ReplaceAll(bodyStr, targetHost, proxyHost)
	}

	// Update body
	newBody := []byte(bodyStr)

	// Re-compress if needed
	if encoding == "gzip" {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, err = gw.Write(newBody)
		gw.Close()
		if err == nil {
			newBody = buf.Bytes()
		}
	}

	// Update response
	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))

	return resp
}

// getProcessForURL returns the process name associated with a URL
func (s *Server) getProcessForURL(urlStr string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normalized := normalizeURL(urlStr)

	// Try exact match first
	if process, ok := s.urlMap[normalized]; ok {
		return process
	}

	// Try to match by host
	if u, err := url.Parse(urlStr); err == nil {
		for mappedURL, process := range s.urlMap {
			if mu, err := url.Parse(mappedURL); err == nil {
				if u.Host == mu.Host {
					return process
				}
			}
		}
	}

	return "unknown"
}

// normalizeURL normalizes a URL for consistent mapping
func normalizeURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// Remove trailing slashes from path
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" {
		path = "/"
	}

	// Rebuild URL without query params for mapping
	normalized := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)
	return normalized
}

// addRequest stores a request in the history
func (s *Server) addRequest(req Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to find telemetry session for this URL
	if s.telemetry != nil {
		sessions := s.telemetry.GetSessionsForURL(req.URL)
		if len(sessions) > 0 {
			// Link to the most recent session for this URL
			req.SessionID = sessions[0].SessionID
			req.HasTelemetry = true
			req.Telemetry = sessions[0]
		}
	}

	s.requests = append(s.requests, req)

	// Keep only last 1000 requests
	if len(s.requests) > 1000 {
		s.requests = s.requests[len(s.requests)-1000:]
	}
}

// linkTelemetryToRequests links telemetry data to existing requests
func (s *Server) linkTelemetryToRequests(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.telemetry.GetSession(sessionID)
	if !exists || session == nil {
		return
	}

	// Find requests that match this session's URL and link them
	sessionURL := session.URL
	if sessionURL == "" {
		return
	}

	// Normalize the session URL for comparison
	normalizedSessionURL := normalizeURL(sessionURL)

	for i := range s.requests {
		req := &s.requests[i]

		// Skip if already has telemetry
		if req.HasTelemetry {
			continue
		}

		// Check if URLs match (normalized comparison)
		normalizedReqURL := normalizeURL(req.URL)
		if normalizedReqURL == normalizedSessionURL || req.URL == sessionURL {
			req.SessionID = sessionID
			req.HasTelemetry = true
			req.Telemetry = session
			// Telemetry session linked to request
		}
	}
}

// GetRequests returns all stored requests
func (s *Server) GetRequests() []Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	requests := make([]Request, len(s.requests))
	copy(requests, s.requests)
	return requests
}

// GetRequestsForProcess returns requests for a specific process
func (s *Server) GetRequestsForProcess(processName string) []Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []Request
	for _, req := range s.requests {
		if req.ProcessName == processName {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

// Start starts the proxy server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("proxy server already running")
	}

	// Try to find an available port, starting from the requested port
	availablePort, err := ports.FindAvailablePort(s.port)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	// Update the port if it changed
	if availablePort != s.port {
		s.port = availablePort
	}

	// In full proxy mode, ensure goproxy is initialized
	if s.mode == ProxyModeFull && s.proxy == nil {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = false
		proxy.Logger = s.createSilentLogger()
		s.proxy = proxy
		s.setupHandlers()
	}

	// Create a custom handler that serves PAC file and proxies everything else
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:     addr,
		ErrorLog: s.createSilentLogger(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle PAC file requests
			if r.URL.Path == "/proxy.pac" || r.URL.Path == "/pac" {
				s.servePACFile(w, r)
				return
			}

			// Handle telemetry endpoint
			if r.URL.Path == "/__brummer_telemetry__" && r.Method == "POST" {
				s.handleTelemetry(w, r)
				return
			}

			// Handle WebSocket telemetry endpoint
			if r.URL.Path == "/__brummer_ws__" {
				s.handleWebSocketTelemetry(w, r)
				return
			}

			// Handle direct browsing to proxy server (not proxy requests)
			if r.Header.Get("Host") == r.Host && (r.URL.Path == "/" || r.URL.Path == "") {
				fmt.Fprintf(w, "Brummer Proxy Server\n\n")
				fmt.Fprintf(w, "Mode: %s\n", s.mode)
				fmt.Fprintf(w, "PAC File: http://localhost:%d/proxy.pac\n\n", s.port)
				fmt.Fprintf(w, "Configure your browser's automatic proxy configuration URL to:\n")
				fmt.Fprintf(w, "http://localhost:%d/proxy.pac\n", s.port)
				return
			}

			// In reverse proxy mode, we don't handle proxy requests on the main port
			if s.mode == ProxyModeReverse {
				http.Error(w, "This is the control port. Use the dedicated proxy ports for each URL.", http.StatusBadRequest)
				return
			}

			// All proxy requests go through goproxy (full proxy mode only)
			if s.proxy != nil {
				s.proxy.ServeHTTP(w, r)
			} else {
				http.Error(w, "Proxy not initialized", http.StatusInternalServerError)
			}
		}),
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	go func() {
		// Get server reference safely
		s.mu.RLock()
		server := s.server
		s.mu.RUnlock()
		
		// Proxy server starting (logs disabled for TUI compatibility)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Proxy server error (logged internally)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}
	}()

	return nil
}

// Stop stops the proxy server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// In reverse proxy mode, stop all individual servers
	if s.mode == ProxyModeReverse {
		for _, mapping := range s.urlMappings {
			if mapping.Server != nil {
				// Stopping reverse proxy (logged internally)
				mapping.Server.Close()
			}
		}
	}

	if s.server != nil {
		return s.server.Close()
	}

	return nil
}

// IsRunning returns whether the proxy server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ClearRequests clears all stored requests
func (s *Server) ClearRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = make([]Request, 0, 1000)

	// Note: We don't stop the proxy servers, just clear the request history
}

// ClearRequestsForProcess clears requests for a specific process
func (s *Server) ClearRequestsForProcess(processName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var filtered []Request
	for _, req := range s.requests {
		if req.ProcessName != processName {
			filtered = append(filtered, req)
		}
	}
	s.requests = filtered
}

// GetPort returns the proxy server port
func (s *Server) GetPort() int {
	return s.port
}

// GetMode returns the proxy server mode
func (s *Server) GetMode() ProxyMode {
	return s.mode
}

// SwitchMode switches the proxy server between full and reverse mode
func (s *Server) SwitchMode(newMode ProxyMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mode == newMode {
		return nil // Already in the requested mode
	}

	oldMode := s.mode
	wasRunning := s.running

	// Stop the server first
	if wasRunning {
		s.running = false
		if s.server != nil {
			s.server.Close()
		}

		// In reverse proxy mode, stop all individual servers
		if oldMode == ProxyModeReverse {
			for _, mapping := range s.urlMappings {
				if mapping.Server != nil {
					// Stopping reverse proxy for mode switch (logged internally)
					mapping.Server.Close()
				}
			}
		}
	}

	// Clear existing proxy setup
	s.proxy = nil

	// Switch mode
	s.mode = newMode

	// Clear URL mappings when switching away from reverse mode
	if oldMode == ProxyModeReverse && newMode == ProxyModeFull {
		// Clear the individual reverse proxy servers
		s.urlMappings = make(map[string]*URLMapping)
		// Reset next port
		s.nextPort = s.basePort + 1000
	}

	// Set up new mode
	if newMode == ProxyModeFull {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = false
		proxy.Logger = s.createSilentLogger()
		s.proxy = proxy
		s.setupHandlers()
	}

	// Restart if it was running
	if wasRunning {
		// Create new server with updated handler
		s.server = &http.Server{
			Addr:     fmt.Sprintf(":%d", s.port),
			ErrorLog: s.createSilentLogger(),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle PAC file requests
				if r.URL.Path == "/proxy.pac" || r.URL.Path == "/pac" {
					s.servePACFile(w, r)
					return
				}

				// Handle telemetry endpoint
				if r.URL.Path == "/__brummer_telemetry__" && r.Method == "POST" {
					s.handleTelemetry(w, r)
					return
				}

				// Handle WebSocket telemetry endpoint
				if r.URL.Path == "/__brummer_ws__" {
					s.handleWebSocketTelemetry(w, r)
					return
				}

				// Handle direct browsing to proxy server (not proxy requests)
				if r.Header.Get("Host") == r.Host && (r.URL.Path == "/" || r.URL.Path == "") {
					fmt.Fprintf(w, "Brummer Proxy Server\n\n")
					fmt.Fprintf(w, "Mode: %s\n", s.mode)
					fmt.Fprintf(w, "PAC File: http://localhost:%d/proxy.pac\n\n", s.port)
					fmt.Fprintf(w, "Configure your browser's automatic proxy configuration URL to:\n")
					fmt.Fprintf(w, "http://localhost:%d/proxy.pac\n", s.port)
					return
				}

				// In reverse proxy mode, we don't handle proxy requests on the main port
				if s.mode == ProxyModeReverse {
					http.Error(w, "This is the control port. Use the dedicated proxy ports for each URL.", http.StatusBadRequest)
					return
				}

				// All proxy requests go through goproxy (full proxy mode only)
				if s.proxy != nil {
					s.proxy.ServeHTTP(w, r)
				} else {
					http.Error(w, "Proxy not initialized", http.StatusInternalServerError)
				}
			}),
		}

		s.running = true

		go func() {
			// Get server reference safely
			s.mu.RLock()
			server := s.server
			s.mu.RUnlock()
			
			// Restarting proxy server (logs disabled for TUI compatibility)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				// Proxy server error (logged internally)
				s.mu.Lock()
				s.running = false
				s.mu.Unlock()
			}
		}()

		// Publish mode switch event
		s.eventBus.Publish(events.Event{
			Type: events.EventType("proxy.mode_switched"),
			Data: map[string]interface{}{
				"old_mode": string(oldMode),
				"new_mode": string(newMode),
			},
		})
	}

	return nil
}

// createReverseProxyHandler creates an HTTP handler for reverse proxy mode
func (s *Server) createReverseProxyHandler() http.Handler {
	// In reverse proxy mode, we don't use the main port
	// Instead, each URL gets its own port
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Reverse proxy mode active. Each URL gets its own port.\n")
	})
	return mux
}

// createURLProxyHandler creates an httputil.ReverseProxy-based handler for a specific URL mapping
func (s *Server) createURLProxyHandler(mapping *URLMapping) http.Handler {
	// Parse target URL
	targetURL, err := url.Parse(mapping.TargetURL)
	if err != nil {
		// Invalid target URL for mapping (logged internally)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid target URL configuration", http.StatusInternalServerError)
		})
	}

	// Create reverse proxy with proper URL rewriting
	rp := &httputil.ReverseProxy{
		ErrorLog: s.createSilentLogger(),
		Director: func(req *http.Request) {
			// Store original URL for logging
			originalURL := fmt.Sprintf("http://%s%s", req.Host, req.URL.RequestURI())
			req.Header.Set("X-Original-URL", originalURL)

			// Rewrite the request to target the backend
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Header.Set("X-Forwarded-Proto", "http")
		},
		ModifyResponse: func(resp *http.Response) error {
			// Track the request
			startTime := time.Now()
			reqID := fmt.Sprintf("%d", time.Now().UnixNano())

			originalURL := resp.Request.Header.Get("X-Original-URL")
			if originalURL == "" {
				originalURL = resp.Request.URL.String()
			}

			// Extract authentication info
			hasAuth, authType, jwtClaims, jwtError := extractAuthInfo(resp.Request)

			// Create request record
			reqRecord := Request{
				ID:          reqID,
				Method:      resp.Request.Method,
				URL:         originalURL,
				Host:        resp.Request.Host,
				Path:        resp.Request.URL.Path,
				StartTime:   startTime,
				ProcessName: mapping.ProcessName,
				StatusCode:  resp.StatusCode,
				IsError:     resp.StatusCode >= 400,
				HasAuth:     hasAuth,
				AuthType:    authType,
				JWTClaims:   jwtClaims,
				JWTError:    jwtError,
				IsXHR:       resp.Request.Header.Get("X-Requested-With") == "XMLHttpRequest",
				ContentType: resp.Header.Get("Content-Type"),
			}

			if resp.ContentLength > 0 {
				reqRecord.Size = resp.ContentLength
			}

			// Store the request
			s.addRequest(reqRecord)

			// Publish event
			s.eventBus.Publish(events.Event{
				Type:      events.EventType("proxy.request"),
				ProcessID: reqRecord.ProcessName,
				Data: map[string]interface{}{
					"method":      reqRecord.Method,
					"url":         reqRecord.URL,
					"status":      reqRecord.StatusCode,
					"duration":    reqRecord.Duration.Milliseconds(),
					"size":        reqRecord.Size,
					"processName": reqRecord.ProcessName,
				},
			})

			// Inject monitoring script and rewrite URLs for HTML responses
			if resp.StatusCode == 200 {
				if s.enableTelemetry {
					s.injectMonitoringScriptForMapping(resp, mapping, resp.Request)
				}
				s.rewriteURLsInResponse(resp, mapping)
			}

			return nil
		},
	}

	mapping.ReverseProxy = rp

	// Handle telemetry endpoint directly
	mux := http.NewServeMux()
	mux.HandleFunc("/__brummer_telemetry__", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "OPTIONS" {
			s.handleTelemetry(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Handle WebSocket telemetry endpoint directly
	mux.HandleFunc("/__brummer_ws__", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebSocketTelemetry(w, r)
	})

	// For all other requests, use the reverse proxy
	mux.HandleFunc("/", rp.ServeHTTP)

	return mux
}

// RegisterURL associates a URL with a process name and returns the proxy URL
func (s *Server) RegisterURL(urlStr, processName string) string {
	return s.RegisterURLWithLabel(urlStr, processName, processName)
}

// RegisterURLWithLabel associates a URL with a process name and label, and returns the proxy URL
func (s *Server) RegisterURLWithLabel(urlStr, processName, label string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Normalize URL
	normalized := normalizeURL(urlStr)
	s.urlMap[normalized] = processName

	// In reverse proxy mode, create a separate server for each URL
	if s.mode == ProxyModeReverse {
		// Check if we already have a mapping for this URL
		if existing, exists := s.urlMappings[normalized]; exists {
			// Update label if it's different and more descriptive than current
			if label != processName && (existing.Label == existing.ProcessName || existing.Label == "") {
				existing.Label = label
			}
			return existing.ProxyURL
		}

		// Allocate a new port starting from nextPort
		port, err := ports.FindAvailablePort(s.nextPort)
		if err != nil {
			return normalized // Return original URL if we can't allocate a port
		}
		s.nextPort = port + 1 // Update nextPort for next allocation

		// Create mapping
		mapping := &URLMapping{
			TargetURL:   normalized,
			ProxyPort:   port,
			ProxyURL:    fmt.Sprintf("http://localhost:%d", port),
			ProcessName: processName,
			Label:       label,
			CreatedAt:   time.Now(),
		}

		// Create and start a new server for this URL
		handler := s.createURLProxyHandler(mapping)
		server := &http.Server{
			Addr:     fmt.Sprintf(":%d", port),
			ErrorLog: s.createSilentLogger(),
			Handler:  handler,
		}
		mapping.Server = server

		// Start the server
		go func() {
			var msg string
			if mapping.Label != "" && mapping.Label != mapping.ProcessName {
				msg = fmt.Sprintf("%s started for %s (%s)", mapping.ProxyURL, normalized, mapping.Label)
			} else {
				msg = fmt.Sprintf("%s started for %s", mapping.ProxyURL, normalized)
			}
			// Publish event for TUI to show in system messages instead of stdout
			s.eventBus.Publish(events.Event{
				Type: events.EventType("system.message"),
				Data: map[string]interface{}{
					"level":   "info",
					"context": "Proxy",
					"message": msg,
				},
			})
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errorMsg := fmt.Sprintf("Reverse proxy error for %s: %v", normalized, err)
				s.eventBus.Publish(events.Event{
					Type: events.EventType("system.message"),
					Data: map[string]interface{}{
						"level":   "error",
						"context": "Proxy",
						"message": errorMsg,
					},
				})
			}
		}()

		s.urlMappings[normalized] = mapping

		// Also register the proxy URL so telemetry can map it back to the process
		// This handles cases where telemetry comes from the proxied URL (localhost:20888)
		proxyURLNormalized := normalizeURL(mapping.ProxyURL)
		s.urlMap[proxyURLNormalized] = processName

		return mapping.ProxyURL
	}

	return urlStr
}

// GetProxyURL returns the proxy URL for a given target URL (reverse proxy mode only)
func (s *Server) GetProxyURL(targetURL string) string {
	if s.mode != ProxyModeReverse {
		return targetURL
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	normalized := normalizeURL(targetURL)
	if mapping, exists := s.urlMappings[normalized]; exists {
		return mapping.ProxyURL
	}

	return targetURL
}

// GetURLMappings returns all URL mappings (reverse proxy mode only)
func (s *Server) GetURLMappings() []URLMapping {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mappings := make([]URLMapping, 0, len(s.urlMappings))
	for _, mapping := range s.urlMappings {
		mappings = append(mappings, *mapping)
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(mappings)-1; i++ {
		for j := i + 1; j < len(mappings); j++ {
			if mappings[j].CreatedAt.After(mappings[i].CreatedAt) {
				mappings[i], mappings[j] = mappings[j], mappings[i]
			}
		}
	}

	return mappings
}

// servePACFile serves the Proxy Auto-Configuration file
func (s *Server) servePACFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")

	// Generate PAC file that uses the proxy with fallback to direct
	pacContent := fmt.Sprintf(`// Brummer Proxy Auto-Configuration
function FindProxyForURL(url, host) {
    // Use proxy for HTTP requests, with fallback to direct
    if (url.substring(0, 5) == "http:") {
        return "PROXY localhost:%d; DIRECT";
    }
    
    // Use proxy for HTTPS requests, with fallback to direct
    if (url.substring(0, 6) == "https:") {
        return "PROXY localhost:%d; DIRECT";
    }
    
    // Everything else goes direct
    return "DIRECT";
}`, s.port, s.port)

	fmt.Fprint(w, pacContent)
}

// GetPACURL returns the URL for the PAC file
func (s *Server) GetPACURL() string {
	return fmt.Sprintf("http://localhost:%d/proxy.pac", s.port)
}

// handleTelemetry handles incoming telemetry data from the monitoring script
func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for telemetry endpoint
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse telemetry batch
	var batch TelemetryBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		http.Error(w, "Invalid telemetry data", http.StatusBadRequest)
		return
	}

	// Get process name from referer or use default
	processName := "unknown"
	referer := r.Header.Get("Referer")
	if referer != "" {
		processName = s.getProcessForURL(referer)
	}

	// Store telemetry data
	s.telemetry.AddBatch(batch, processName)

	// Retroactively link telemetry to existing requests
	s.linkTelemetryToRequests(batch.SessionID)

	// Broadcast to WebSocket clients
	s.SendTelemetryToWebSockets(batch, processName)

	// Publish telemetry event
	s.eventBus.Publish(events.Event{
		Type:      events.EventType("telemetry.received"),
		ProcessID: processName,
		Data: map[string]interface{}{
			"sessionId":  batch.SessionID,
			"eventCount": len(batch.Events),
		},
	})

	// Return success
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

// GetTelemetryStore returns the telemetry store
func (s *Server) GetTelemetryStore() *TelemetryStore {
	return s.telemetry
}

// EnableTelemetry enables or disables telemetry collection
func (s *Server) EnableTelemetry(enable bool) {
	// Just set it directly - bool writes are atomic
	s.enableTelemetry = enable
}

// IsTelemetryEnabled returns whether telemetry is enabled
func (s *Server) IsTelemetryEnabled() bool {
	// Just read it directly - bool reads are atomic
	return s.enableTelemetry
}

// GetTelemetrySession returns telemetry data for a specific session
func (s *Server) GetTelemetrySession(sessionID string) (*PageSession, bool) {
	return s.telemetry.GetSession(sessionID)
}

// GetTelemetryForProcess returns all telemetry sessions for a process
func (s *Server) GetTelemetryForProcess(processName string) []*PageSession {
	return s.telemetry.GetSessionsForProcess(processName)
}

// ClearTelemetryForProcess clears telemetry data for a specific process
func (s *Server) ClearTelemetryForProcess(processName string) {
	s.telemetry.ClearSessionsForProcess(processName)
}

// isBackgroundRequest checks if this is an AJAX/fetch request based on headers
func (s *Server) isBackgroundRequest(req *http.Request) bool {
	if req == nil {
		return false
	}

	// Check for XMLHttpRequest header (used by jQuery and older AJAX libraries)
	if req.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return true
	}

	// Check Fetch metadata headers (modern browsers)
	fetchMode := req.Header.Get("Sec-Fetch-Mode")
	fetchDest := req.Header.Get("Sec-Fetch-Dest")

	// Skip injection for cors, no-cors, and same-origin modes (typically AJAX)
	// Only inject for navigate mode (actual page navigation)
	if fetchMode != "" && fetchMode != "navigate" {
		return true
	}

	// Skip injection for non-document destinations
	if fetchDest != "" && fetchDest != "document" {
		return true
	}

	// Check Accept header - if it's specifically asking for JSON or XML, skip
	accept := req.Header.Get("Accept")
	if strings.Contains(accept, "application/json") ||
		strings.Contains(accept, "application/xml") ||
		strings.Contains(accept, "text/xml") {
		return true
	}

	return false
}

// WebSocket message types
type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// handleWebSocketTelemetry handles WebSocket connections for real-time telemetry
func (s *Server) handleWebSocketTelemetry(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// WebSocket upgrade failed (logged internally)
		return
	}
	defer conn.Close()

	// Register client
	s.mu.Lock()
	s.wsClients[conn] = true
	clientCount := len(s.wsClients)
	s.mu.Unlock()

	// WebSocket connection events are only for internal tracking, no need to log to stdout

	// Send welcome message
	welcomeMsg := WSMessage{
		Type: "connected",
		Data: map[string]interface{}{
			"message":     "Connected to Brummer telemetry",
			"serverTime":  time.Now().UnixMilli(),
			"clientCount": clientCount,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	conn.WriteJSON(welcomeMsg)

	// Handle incoming messages (for REPL commands)
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			// WebSocket read error (logged internally)
			break
		}

		// Handle REPL commands
		s.handleWSCommand(conn, msg)
	}

	// Unregister client
	s.mu.Lock()
	delete(s.wsClients, conn)
	clientCount = len(s.wsClients)
	s.mu.Unlock()

	// WebSocket disconnection events are only for internal tracking, no need to log to stdout
}

// handleWSCommand processes incoming WebSocket commands for REPL functionality
func (s *Server) handleWSCommand(conn *websocket.Conn, msg WSMessage) {
	response := WSMessage{
		Type:      "command_response",
		Timestamp: time.Now().UnixMilli(),
	}

	switch msg.Type {
	case "ping":
		response.Data = map[string]interface{}{
			"pong":       true,
			"serverTime": time.Now().UnixMilli(),
		}

	case "status":
		response.Data = map[string]interface{}{
			"requests":         len(s.requests),
			"telemetryEnabled": s.enableTelemetry,
			"mode":             s.mode,
			"port":             s.port,
			"sessions":         s.telemetry.GetAllSessions(),
		}

	case "clear_buffer":
		response.Data = map[string]interface{}{
			"message": "Buffer cleared",
			"cleared": true,
		}

	case "get_requests":
		limit := 100 // Default limit
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if l, ok := data["limit"].(float64); ok {
				limit = int(l)
			}
		}

		requests := s.GetRequests()
		if len(requests) > limit {
			requests = requests[len(requests)-limit:]
		}

		response.Data = map[string]interface{}{
			"requests": requests,
			"total":    len(s.requests),
			"returned": len(requests),
		}

	case "get_telemetry":
		sessions := s.telemetry.GetAllSessions()
		response.Data = map[string]interface{}{
			"sessions": sessions,
			"count":    len(sessions),
		}

	case "telemetry":
		// Handle incoming telemetry data via WebSocket
		if data, ok := msg.Data.(map[string]interface{}); ok {
			// Convert the data back to TelemetryBatch format
			batch := TelemetryBatch{}

			// Extract sessionId
			if sessionId, ok := data["sessionId"].(string); ok {
				batch.SessionID = sessionId
			}

			// Extract events
			if events, ok := data["events"].([]interface{}); ok {
				for _, event := range events {
					if eventMap, ok := event.(map[string]interface{}); ok {
						telemetryEvent := TelemetryEvent{}

						if eventType, ok := eventMap["type"].(string); ok {
							telemetryEvent.Type = TelemetryEventType(eventType)
						}
						if timestamp, ok := eventMap["timestamp"].(float64); ok {
							telemetryEvent.Timestamp = int64(timestamp)
						}
						if sessionId, ok := eventMap["sessionId"].(string); ok {
							telemetryEvent.SessionID = sessionId
						}
						if url, ok := eventMap["url"].(string); ok {
							telemetryEvent.URL = url
						}
						if data, ok := eventMap["data"].(map[string]interface{}); ok {
							telemetryEvent.Data = data
						}

						batch.Events = append(batch.Events, telemetryEvent)
					}
				}
			}

			// Determine process name from metadata or use default
			processName := "unknown"
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				if urlStr, ok := metadata["url"].(string); ok {
					processName = s.getProcessForURL(urlStr)

					// If still unknown, try to extract from the URL host
					if processName == "unknown" {
						if u, err := url.Parse(urlStr); err == nil {
							// Check if this is one of our proxy URLs
							for proxyURL, mappedProcess := range s.urlMap {
								if pu, err := url.Parse(proxyURL); err == nil {
									if u.Host == pu.Host {
										processName = mappedProcess
										break
									}
								}
							}
						}
					}
				}
			}

			// Store telemetry data (same as HTTP handler)
			s.telemetry.AddBatch(batch, processName)

			// Retroactively link telemetry to existing requests
			s.linkTelemetryToRequests(batch.SessionID)

			// Broadcast to other WebSocket clients
			s.SendTelemetryToWebSockets(batch, processName)

			// Publish telemetry event
			s.eventBus.Publish(events.Event{
				Type:      events.EventType("telemetry.received"),
				ProcessID: processName,
				Data: map[string]interface{}{
					"sessionId":  batch.SessionID,
					"eventCount": len(batch.Events),
					"source":     "websocket",
				},
			})

			response.Data = map[string]interface{}{
				"status":    "ok",
				"received":  len(batch.Events),
				"sessionId": batch.SessionID,
			}
		} else {
			response.Data = map[string]interface{}{
				"error": "Invalid telemetry data format",
			}
		}

		// Send response back to client
		conn.WriteJSON(response)
		return // Don't send another response below

	case "repl_response":
		// Handle REPL response from browser
		if data, ok := msg.Data.(map[string]interface{}); ok {
			responseID, hasID := data["responseId"].(string)
			if !hasID {
				response.Data = map[string]interface{}{
					"error": "Missing responseId in REPL response",
				}
			} else {
				// Forward the response to MCP server via event bus
				s.eventBus.Publish(events.Event{
					Type: events.EventType("repl.response"),
					Data: map[string]interface{}{
						"responseId": responseID,
						"result":     data["result"],
						"error":      data["error"],
					},
				})

				response.Data = map[string]interface{}{
					"status":     "forwarded",
					"responseId": responseID,
				}
			}
		} else {
			response.Data = map[string]interface{}{
				"error": "Invalid REPL response format",
			}
		}

	default:
		response.Data = map[string]interface{}{
			"error": "Unknown command type: " + msg.Type,
		}
	}

	conn.WriteJSON(response)
}

// BroadcastToWebSockets sends a message to all connected WebSocket clients
func (s *Server) BroadcastToWebSockets(msgType string, data interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.wsClients) == 0 {
		return
	}

	msg := WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	// Send to all clients
	for conn := range s.wsClients {
		err := conn.WriteJSON(msg)
		if err != nil {
			// WebSocket broadcast error (logged internally)
			// Note: Client cleanup now safe with unified mutex hierarchy
		}
	}
}

// SendTelemetryToWebSockets broadcasts telemetry data to WebSocket clients
func (s *Server) SendTelemetryToWebSockets(batch TelemetryBatch, processName string) {
	s.BroadcastToWebSockets("telemetry", map[string]interface{}{
		"batch":       batch,
		"processName": processName,
	})
}
