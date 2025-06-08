package proxy

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/beagle/brummer/pkg/events"
	"github.com/elazarl/goproxy"
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
}

// ProxyMode defines the proxy operation mode
type ProxyMode string

const (
	ProxyModeFull    ProxyMode = "full"    // Traditional HTTP proxy
	ProxyModeReverse ProxyMode = "reverse" // Reverse proxy for detected URLs
)

// URLMapping represents a reverse proxy mapping
type URLMapping struct {
	TargetURL   string      // e.g., "http://localhost:3000"
	ProxyPort   int         // e.g., 8889
	ProxyURL    string      // e.g., "http://localhost:8889"
	ProcessName string
	CreatedAt   time.Time
	Server      *http.Server // The HTTP server for this mapping
}

//go:embed monitor.js
var monitoringScript string

// Server manages the HTTP proxy server
type Server struct {
	port      int
	mode      ProxyMode
	proxy     *goproxy.ProxyHttpServer
	server    *http.Server
	eventBus  *events.EventBus
	
	mu          sync.RWMutex
	requests    []Request
	urlMap      map[string]string // Maps URL to process name
	
	// Reverse proxy specific fields
	urlMappings map[string]*URLMapping // Maps target URL to mapping
	nextPort    int                    // Next available port for reverse proxy
	basePort    int                    // Base port for reverse proxy mode
	
	// Telemetry
	telemetry   *TelemetryStore
	enableTelemetry bool
	
	running     bool
}

// NewServer creates a new proxy server
func NewServer(port int, eventBus *events.EventBus) *Server {
	return NewServerWithMode(port, ProxyModeFull, eventBus)
}

// NewServerWithMode creates a new proxy server with specified mode
func NewServerWithMode(port int, mode ProxyMode, eventBus *events.EventBus) *Server {
	s := &Server{
		port:        port,
		mode:        mode,
		eventBus:    eventBus,
		requests:    make([]Request, 0, 1000),
		urlMap:      make(map[string]string),
		urlMappings: make(map[string]*URLMapping),
		basePort:    port,
		nextPort:    port + 1000, // Start allocating from port+1000 for reverse proxy URLs
		telemetry:   NewTelemetryStore(),
		enableTelemetry: true,
	}
	
	if mode == ProxyModeFull {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = false
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
		
		// Store request info in context
		ctx.UserData = &Request{
			ID:          reqID,
			Method:      r.Method,
			URL:         r.URL.String(),
			Host:        r.Host,
			Path:        r.URL.Path,
			StartTime:   startTime,
			ProcessName: processName,
		}
		
		return r, nil
	})
	
	// Handle responses
	s.proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if ctx.UserData != nil {
			req := ctx.UserData.(*Request)
			req.Duration = time.Since(req.StartTime)
			req.StatusCode = resp.StatusCode
			
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
	
	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp
	}
	resp.Body.Close()
	
	// Decompress if needed
	encoding := resp.Header.Get("Content-Encoding")
	if encoding == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			resp.Body = ioutil.NopCloser(bytes.NewReader(body))
			return resp
		}
		body, err = ioutil.ReadAll(reader)
		reader.Close()
		if err != nil {
			resp.Body = ioutil.NopCloser(bytes.NewReader(body))
			return resp
		}
	}
	
	// Convert body to string for manipulation
	bodyStr := string(body)
	
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
	resp.Body = ioutil.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
	
	// Remove content security policy that might block our script
	resp.Header.Del("Content-Security-Policy")
	resp.Header.Del("X-Content-Security-Policy")
	
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
	
	// In full proxy mode, ensure goproxy is initialized
	if s.mode == ProxyModeFull && s.proxy == nil {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = false
		s.proxy = proxy
		s.setupHandlers()
	}
	
	// Create a custom handler that serves PAC file and proxies everything else
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr: addr,
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
		modeStr := string(s.mode)
		log.Printf("Starting %s proxy server on port %d", modeStr, s.port)
		log.Printf("PAC file available at: http://localhost:%d/proxy.pac", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Proxy server error: %v", err)
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
		for url, mapping := range s.urlMappings {
			if mapping.Server != nil {
				log.Printf("Stopping reverse proxy for %s on port %d", url, mapping.ProxyPort)
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

// createURLProxyHandler creates a handler for a specific URL mapping
func (s *Server) createURLProxyHandler(mapping *URLMapping) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle telemetry endpoint
		if r.URL.Path == "/__brummer_telemetry__" && r.Method == "POST" {
			s.handleTelemetry(w, r)
			return
		}
		
		// Parse target URL
		targetURL, err := url.Parse(mapping.TargetURL)
		if err != nil {
			log.Printf("Error parsing target URL %s: %v", mapping.TargetURL, err)
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
			return
		}
		
		// Build the full target URL with the request path
		targetURLStr := fmt.Sprintf("%s://%s%s", targetURL.Scheme, targetURL.Host, r.URL.Path)
		if r.URL.RawQuery != "" {
			targetURLStr += "?" + r.URL.RawQuery
		}
		
		// Create the proxied request
		proxyReq, err := http.NewRequest(r.Method, targetURLStr, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}
	
		// Copy headers
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}
		
		// Set forwarding headers
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
		proxyReq.Header.Set("X-Forwarded-Host", r.Host)
		proxyReq.Header.Set("X-Forwarded-Proto", "http")
		
		// Record request start
		startTime := time.Now()
		reqID := fmt.Sprintf("%d", time.Now().UnixNano())
		
		// Perform the request
		client := &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects automatically
			},
		}
		
		log.Printf("Proxying request: %s %s -> %s", r.Method, r.URL.Path, targetURLStr)
		resp, err := client.Do(proxyReq)
		
		duration := time.Since(startTime)
		
		// Create request record
		req := Request{
			ID:          reqID,
			Method:      r.Method,
			URL:         targetURLStr,
			Host:        targetURL.Host,
			Path:        r.URL.Path,
			StartTime:   startTime,
			Duration:    duration,
			ProcessName: mapping.ProcessName,
		}
		
		if err != nil {
			req.Error = err.Error()
			s.addRequest(req)
			http.Error(w, "Failed to proxy request", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		
		req.StatusCode = resp.StatusCode
		if resp.ContentLength > 0 {
			req.Size = resp.ContentLength
		}
		
		// Store the request
		s.addRequest(req)
		
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
		
		// Check if this is an HTML response that we should inject telemetry into
		contentType := resp.Header.Get("Content-Type")
		isHTML := strings.Contains(contentType, "text/html")
		
		// Just read telemetry enabled directly - it's a bool, so it's atomic
		if s.enableTelemetry && isHTML && resp.StatusCode == 200 {
			// Read the entire response body for injection
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read response", http.StatusInternalServerError)
				return
			}
			
			// Handle gzip encoding
			encoding := resp.Header.Get("Content-Encoding")
			if encoding == "gzip" {
				reader, err := gzip.NewReader(bytes.NewReader(body))
				if err == nil {
					body, _ = ioutil.ReadAll(reader)
					reader.Close()
				}
			}
			
			// Inject telemetry script
			bodyStr := string(body)
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
`, mapping.ProcessName, s.port, monitoringScript)
			
			// Try to inject before </body> or </html>
			injected := false
			for _, tag := range []string{"</body>", "</html>"} {
				if idx := strings.LastIndex(strings.ToLower(bodyStr), tag); idx != -1 {
					bodyStr = bodyStr[:idx] + injectionScript + bodyStr[idx:]
					injected = true
					break
				}
			}
			
			// If no suitable tag found, append to end
			if !injected {
				bodyStr += injectionScript
			}
			
			// Update content length
			modifiedBody := []byte(bodyStr)
			resp.Header.Del("Content-Length")
			resp.Header.Del("Content-Encoding") // Remove gzip encoding since we decoded it
			
			// Copy headers except the ones we're modifying
			for key, values := range resp.Header {
				if key != "Content-Length" && key != "Content-Encoding" {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
			}
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))
			
			// Remove CSP headers that might block our script
			w.Header().Del("Content-Security-Policy")
			w.Header().Del("Content-Security-Policy-Report-Only")
			
			w.WriteHeader(resp.StatusCode)
			w.Write(modifiedBody)
		} else {
			// For non-HTML responses, just copy as-is
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			
			w.WriteHeader(resp.StatusCode)
			
			// Copy response body
			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					w.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}
	}
}

// RegisterURL associates a URL with a process name and returns the proxy URL
func (s *Server) RegisterURL(urlStr, processName string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Normalize URL
	normalized := normalizeURL(urlStr)
	s.urlMap[normalized] = processName
	
	// In reverse proxy mode, create a separate server for each URL
	if s.mode == ProxyModeReverse {
		// Check if we already have a mapping for this URL
		if existing, exists := s.urlMappings[normalized]; exists {
			return existing.ProxyURL
		}
		
		// Allocate a new port
		port := s.nextPort
		s.nextPort++
		
		// Create mapping
		mapping := &URLMapping{
			TargetURL:   normalized,
			ProxyPort:   port,
			ProxyURL:    fmt.Sprintf("http://localhost:%d", port),
			ProcessName: processName,
			CreatedAt:   time.Now(),
		}
		
		// Create and start a new server for this URL
		handler := s.createURLProxyHandler(mapping)
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: handler,
		}
		mapping.Server = server
		
		// Start the server
		go func() {
			log.Printf("Starting reverse proxy for %s on port %d", normalized, port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Reverse proxy error for %s: %v", normalized, err)
			}
		}()
		
		s.urlMappings[normalized] = mapping
		
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
	body, err := ioutil.ReadAll(r.Body)
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