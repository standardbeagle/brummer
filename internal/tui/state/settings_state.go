package state

import (
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// SettingsState manages all settings-related UI state
// This is a sub-model that encapsulates settings display according to Elm/Redux patterns
type SettingsState struct {
	// Settings data cache
	SettingsInfo     *messages.SettingsInfo
	CacheTimestamp   time.Time
	NeedsRefresh     bool

	// UI state
	Loading          bool
	LastError        string
	LastUpdateTime   time.Time

	// Display settings
	ShowDetails      bool
	AutoRefresh      bool
	RefreshInterval  time.Duration

	// Version for atomic updates and state tracking
	version          uint64
}

// NewSettingsState creates a new settings sub-model with initial state
func NewSettingsState() *SettingsState {
	return &SettingsState{
		SettingsInfo:    nil,
		CacheTimestamp:  time.Time{},
		NeedsRefresh:    true,
		Loading:         false,
		LastError:       "",
		LastUpdateTime:  time.Time{},
		ShowDetails:     true,
		AutoRefresh:     false,
		RefreshInterval: 30 * time.Second,
		version:         0,
	}
}

// UpdateSettingsInfo updates the cached settings information (immutable)
func (s *SettingsState) UpdateSettingsInfo(settingsInfo *messages.SettingsInfo) *SettingsState {

	newState := s.copy()
	newState.SettingsInfo = settingsInfo
	newState.CacheTimestamp = time.Now()
	newState.LastUpdateTime = time.Now()
	newState.NeedsRefresh = false
	newState.Loading = false
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *SettingsState) SetLoading(loading bool) *SettingsState {

	newState := s.copy()
	newState.Loading = loading
	if loading {
		newState.NeedsRefresh = true
	}
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *SettingsState) SetError(err error) *SettingsState {

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.Loading = false
	newState.incrementVersion()
	return newState
}

// SetShowDetails updates the detail view setting (immutable)
func (s *SettingsState) SetShowDetails(show bool) *SettingsState {

	newState := s.copy()
	newState.ShowDetails = show
	newState.incrementVersion()
	return newState
}

// SetAutoRefresh updates the auto-refresh setting (immutable)
func (s *SettingsState) SetAutoRefresh(refresh bool) *SettingsState {

	newState := s.copy()
	newState.AutoRefresh = refresh
	newState.incrementVersion()
	return newState
}

// SetRefreshInterval updates the refresh interval (immutable)
func (s *SettingsState) SetRefreshInterval(interval time.Duration) *SettingsState {

	newState := s.copy()
	newState.RefreshInterval = interval
	newState.incrementVersion()
	return newState
}

// MarkForRefresh marks that the settings data needs refreshing (immutable)
func (s *SettingsState) MarkForRefresh() *SettingsState {

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// ClearError clears any existing error (immutable)
func (s *SettingsState) ClearError() *SettingsState {

	newState := s.copy()
	newState.LastError = ""
	newState.incrementVersion()
	return newState
}

// IsCacheValid checks if the cached settings data is still valid
func (s *SettingsState) IsCacheValid(maxAge time.Duration) bool {

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetMCPServerInfo returns MCP server information
func (s *SettingsState) GetMCPServerInfo() *messages.MCPServerInfo {

	if s.SettingsInfo == nil {
		return nil
	}
	return &s.SettingsInfo.ServerInfo.MCPServer
}

// GetProxyServerInfo returns proxy server information
func (s *SettingsState) GetProxyServerInfo() *messages.ProxyServerInfo {

	if s.SettingsInfo == nil {
		return nil
	}
	return &s.SettingsInfo.ServerInfo.ProxyServer
}

// GetPackageManagers returns package manager information
func (s *SettingsState) GetPackageManagers() *messages.PackageMgrInfo {

	if s.SettingsInfo == nil {
		return nil
	}
	return &s.SettingsInfo.PackageMgr
}

// GetWorkingDirectory returns the working directory
func (s *SettingsState) GetWorkingDirectory() string {

	if s.SettingsInfo == nil {
		return ""
	}
	return s.SettingsInfo.WorkingDir
}

// IsMCPConfigured returns whether MCP is configured
func (s *SettingsState) IsMCPConfigured() bool {

	if s.SettingsInfo == nil {
		return false
	}
	return s.SettingsInfo.MCPConfigured
}

// GetStateSummary returns a summary of current state
func (s *SettingsState) GetStateSummary() map[string]interface{} {

	summary := map[string]interface{}{
		"loading":           s.Loading,
		"last_error":        s.LastError,
		"last_update_time":  s.LastUpdateTime,
		"show_details":      s.ShowDetails,
		"auto_refresh":      s.AutoRefresh,
		"refresh_interval":  s.RefreshInterval,
		"needs_refresh":     s.NeedsRefresh,
		"cache_timestamp":   s.CacheTimestamp,
	}

	if s.SettingsInfo != nil {
		summary["mcp_configured"] = s.SettingsInfo.MCPConfigured
		summary["mcp_port"] = s.SettingsInfo.ServerInfo.MCPServer.Port
		summary["proxy_running"] = s.SettingsInfo.ServerInfo.ProxyServer.Running
		summary["package_mgr_current"] = s.SettingsInfo.PackageMgr.Current
		summary["working_dir"] = s.SettingsInfo.WorkingDir
	}

	return summary
}

// Validate checks for state consistency
func (s *SettingsState) Validate() []string {

	var issues []string

	// Basic validation
	if s.RefreshInterval <= 0 {
		issues = append(issues, "invalid refresh interval")
	}

	if s.SettingsInfo != nil {
		// Validate MCP port
		if s.SettingsInfo.ServerInfo.MCPServer.Port <= 0 || s.SettingsInfo.ServerInfo.MCPServer.Port > 65535 {
			issues = append(issues, "invalid MCP server port")
		}

		// Validate proxy port if running
		if s.SettingsInfo.ServerInfo.ProxyServer.Running {
			if s.SettingsInfo.ServerInfo.ProxyServer.Port <= 0 || s.SettingsInfo.ServerInfo.ProxyServer.Port > 65535 {
				issues = append(issues, "invalid proxy server port")
			}
		}

		// Validate working directory
		if s.SettingsInfo.WorkingDir == "" {
			issues = append(issues, "empty working directory")
		}
	}

	return issues
}

// Getters for current state values
func (s *SettingsState) GetSettingsInfo() *messages.SettingsInfo {

	if s.SettingsInfo == nil {
		return nil
	}
	// Return a copy to prevent mutation
	settingsCopy := *s.SettingsInfo
	return &settingsCopy
}

func (s *SettingsState) GetLoading() bool {
	return s.Loading
}

func (s *SettingsState) GetError() string {
	return s.LastError
}

func (s *SettingsState) GetShowDetails() bool {
	return s.ShowDetails
}

func (s *SettingsState) GetAutoRefresh() bool {
	return s.AutoRefresh
}

func (s *SettingsState) GetRefreshInterval() time.Duration {
	return s.RefreshInterval
}

func (s *SettingsState) GetNeedsRefresh() bool {
	return s.NeedsRefresh
}

func (s *SettingsState) GetLastUpdateTime() time.Time {
	return s.LastUpdateTime
}

func (s *SettingsState) GetVersion() uint64 {
	return s.version
}

// Helper methods
func (s *SettingsState) copy() *SettingsState {
	newState := *s

	// Deep copy SettingsInfo if present
	if s.SettingsInfo != nil {
		settingsCopy := *s.SettingsInfo
		newState.SettingsInfo = &settingsCopy
	}

	return &newState
}

func (s *SettingsState) incrementVersion() {
	s.version++
}