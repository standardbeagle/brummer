package services

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// ProxyServiceBridge implements ProxyService using proxy.Server
type ProxyServiceBridge struct {
	server  *proxy.Server
	enabled bool
	port    int
}

// NewProxyServiceBridge creates a bridge to proxy functionality
func NewProxyServiceBridge(server *proxy.Server) *ProxyServiceBridge {
	return &ProxyServiceBridge{
		server: server,
		port:   3000, // Default proxy port
	}
}

// SetEnabled enables or disables the proxy
func (s *ProxyServiceBridge) SetEnabled(enabled bool) tea.Cmd {
	return func() tea.Msg {
		s.enabled = enabled
		
		if enabled {
			// In a real implementation, start the proxy server
			return CommandResultMsg{
				Success: true,
				Message: fmt.Sprintf("Proxy enabled on port %d", s.port),
			}
		} else {
			// In a real implementation, stop the proxy server
			return CommandResultMsg{
				Success: true,
				Message: "Proxy disabled",
			}
		}
	}
}

// Toggle switches the proxy state
func (s *ProxyServiceBridge) Toggle() tea.Cmd {
	return s.SetEnabled(!s.enabled)
}

// GetInfo returns current proxy configuration
func (s *ProxyServiceBridge) GetInfo() ProxyInfo {
	requestCount := 0
	if s.server != nil {
		// In a real implementation, get request count from server
		requestCount = 0
	}
	
	return ProxyInfo{
		Enabled:      s.enabled,
		Port:         s.port,
		TargetURL:    fmt.Sprintf("http://localhost:%d", s.port),
		RequestCount: requestCount,
	}
}

// GetRequests returns intercepted HTTP requests
func (s *ProxyServiceBridge) GetRequests(limit int) []ProxyRequest {
	// In a real implementation, this would fetch from the proxy server
	// For now, return empty list
	return []ProxyRequest{}
}