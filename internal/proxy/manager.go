package proxy

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/beagle/brummer/internal/logs"
	"github.com/beagle/brummer/internal/mcp"
	"github.com/beagle/brummer/pkg/events"
)

type ProxyManager struct {
	mcpServer      *mcp.Server
	eventBus       *events.EventBus
	logStore       *logs.Store
	activeProxies  map[string]*ProxyInfo // processID -> ProxyInfo
	portAllocator  *PortAllocator
	mu             sync.RWMutex
}

type ProxyInfo struct {
	ProcessID    string
	ProcessName  string
	TargetURL    string
	ProxyURL     string
	ProxyPort    int
	Server       *ProxyServer
	StartTime    time.Time
	Status       string // "starting", "running", "stopped", "failed"
}

type PortAllocator struct {
	startPort int
	endPort   int
	usedPorts map[int]bool
	mu        sync.Mutex
}

var (
	// Pattern to detect development server URLs
	devServerPattern = regexp.MustCompile(`(?i)(?:server|app|dev|listening|running|started).*?(?:on|at|@)?\s*(https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0)(?::\d+)?(?:/[^\s]*)?)`)
	// Additional patterns for specific frameworks
	nextDevPattern = regexp.MustCompile(`(?i)ready.*?started server on.*?(https?://[^\s]+)`)
	vitePattern    = regexp.MustCompile(`(?i)Local:\s*(https?://[^\s]+)`)
	webpackPattern = regexp.MustCompile(`(?i)Project is running at\s*(https?://[^\s]+)`)
	createReactPattern = regexp.MustCompile(`(?i)compiled.*?you can now view.*?(https?://[^\s]+)`)
)

func NewProxyManager(mcpServer *mcp.Server, eventBus *events.EventBus, logStore *logs.Store) *ProxyManager {
	pm := &ProxyManager{
		mcpServer:     mcpServer,
		eventBus:      eventBus,
		logStore:      logStore,
		activeProxies: make(map[string]*ProxyInfo),
		portAllocator: &PortAllocator{
			startPort: 8080,
			endPort:   8099,
			usedPorts: make(map[int]bool),
		},
	}

	// Subscribe to log events to detect dev servers
	eventBus.Subscribe(events.LogLine, func(e events.Event) {
		processID := e.ProcessID
		if line, ok := e.Data["line"].(string); ok {
			pm.checkForDevServer(processID, line)
		}
	})

	// Subscribe to process exit events to cleanup proxies
	eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		pm.StopProxy(e.ProcessID)
	})

	return pm
}

func (pm *ProxyManager) checkForDevServer(processID string, logLine string) {
	// Check if we already have a proxy for this process
	pm.mu.RLock()
	if _, exists := pm.activeProxies[processID]; exists {
		pm.mu.RUnlock()
		return
	}
	pm.mu.RUnlock()

	// Try to extract URL from log line
	var detectedURL string
	
	// Try specific patterns first
	patterns := []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{nextDevPattern, "Next.js"},
		{vitePattern, "Vite"},
		{webpackPattern, "Webpack"},
		{createReactPattern, "Create React App"},
		{devServerPattern, "Generic"},
	}

	for _, p := range patterns {
		if matches := p.pattern.FindStringSubmatch(logLine); len(matches) > 1 {
			detectedURL = matches[1]
			pm.logStore.Add("proxy", "Proxy Manager", 
				fmt.Sprintf("🔍 Detected %s dev server: %s", p.name, detectedURL), false)
			break
		}
	}

	if detectedURL == "" {
		return
	}

	// Validate URL
	parsedURL, err := url.Parse(detectedURL)
	if err != nil || parsedURL.Host == "" {
		return
	}

	// Only proxy local development servers
	host := strings.Split(parsedURL.Host, ":")[0]
	if host != "localhost" && host != "127.0.0.1" && host != "0.0.0.0" {
		return
	}

	// Get process info
	processName := "unknown"
	if proc, exists := pm.getProcessInfo(processID); exists {
		processName = proc
	}

	// Start proxy for this dev server
	go pm.StartProxy(processID, processName, detectedURL)
}

func (pm *ProxyManager) StartProxy(processID, processName, targetURL string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check again if proxy already exists
	if _, exists := pm.activeProxies[processID]; exists {
		return fmt.Errorf("proxy already exists for process %s", processID)
	}

	// Allocate port
	proxyPort, err := pm.portAllocator.allocatePort()
	if err != nil {
		pm.logStore.Add("proxy", "Proxy Manager", 
			fmt.Sprintf("❌ Failed to allocate port for proxy: %v", err), true)
		return err
	}

	// Create proxy info
	proxyInfo := &ProxyInfo{
		ProcessID:   processID,
		ProcessName: processName,
		TargetURL:   targetURL,
		ProxyPort:   proxyPort,
		ProxyURL:    fmt.Sprintf("http://localhost:%d", proxyPort),
		StartTime:   time.Now(),
		Status:      "starting",
	}

	// Create and start proxy server
	proxyServer, err := NewProxyServer(targetURL, proxyPort, pm.mcpServer, processName)
	if err != nil {
		pm.portAllocator.releasePort(proxyPort)
		pm.logStore.Add("proxy", "Proxy Manager", 
			fmt.Sprintf("❌ Failed to create proxy: %v", err), true)
		return err
	}

	proxyInfo.Server = proxyServer
	pm.activeProxies[processID] = proxyInfo

	// Start proxy in background
	go func() {
		pm.logStore.Add("proxy", "Proxy Manager", 
			fmt.Sprintf("🚀 Starting proxy on %s → %s for %s", 
				proxyInfo.ProxyURL, targetURL, processName), false)
		
		proxyInfo.Status = "running"
		
		// Publish proxy started event
		pm.eventBus.Publish(events.Event{
			Type:      events.ProxyStarted,
			ProcessID: processID,
			Data: map[string]interface{}{
				"processName": processName,
				"targetURL":   targetURL,
				"proxyURL":    proxyInfo.ProxyURL,
				"proxyPort":   proxyPort,
			},
		})

		// This blocks until proxy stops
		if err := proxyServer.Start(); err != nil {
			pm.logStore.Add("proxy", "Proxy Manager", 
				fmt.Sprintf("❌ Proxy error: %v", err), true)
			proxyInfo.Status = "failed"
		} else {
			proxyInfo.Status = "stopped"
		}

		// Cleanup
		pm.mu.Lock()
		delete(pm.activeProxies, processID)
		pm.portAllocator.releasePort(proxyPort)
		pm.mu.Unlock()

		// Publish proxy stopped event
		pm.eventBus.Publish(events.Event{
			Type:      events.ProxyStopped,
			ProcessID: processID,
			Data: map[string]interface{}{
				"processName": processName,
			},
		})
	}()

	return nil
}

func (pm *ProxyManager) StopProxy(processID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proxyInfo, exists := pm.activeProxies[processID]
	if !exists {
		return nil // No proxy to stop
	}

	pm.logStore.Add("proxy", "Proxy Manager", 
		fmt.Sprintf("🛑 Stopping proxy for %s", proxyInfo.ProcessName), false)

	// Stop the proxy server
	if proxyInfo.Server != nil {
		proxyInfo.Server.Stop()
	}

	// Cleanup
	delete(pm.activeProxies, processID)
	pm.portAllocator.releasePort(proxyInfo.ProxyPort)

	return nil
}

func (pm *ProxyManager) GetActiveProxies() map[string]*ProxyInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return a copy to avoid race conditions
	proxies := make(map[string]*ProxyInfo)
	for k, v := range pm.activeProxies {
		proxies[k] = v
	}
	return proxies
}

func (pm *ProxyManager) GetProxyForProcess(processID string) (*ProxyInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	proxy, exists := pm.activeProxies[processID]
	return proxy, exists
}

// Helper to get process name from process manager
func (pm *ProxyManager) getProcessInfo(processID string) (string, bool) {
	// Look for process name in log store's process map
	// The process name is typically part of the processID format: "scriptname-timestamp"
	parts := strings.Split(processID, "-")
	if len(parts) > 0 {
		return parts[0], true
	}
	return processID, true
}

// Port allocator methods
func (pa *PortAllocator) allocatePort() (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	for port := pa.startPort; port <= pa.endPort; port++ {
		if pa.usedPorts[port] {
			continue
		}

		// Check if port is actually available
		if isPortAvailable(port) {
			pa.usedPorts[port] = true
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", pa.startPort, pa.endPort)
}

func (pa *PortAllocator) releasePort(port int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	delete(pa.usedPorts, port)
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}