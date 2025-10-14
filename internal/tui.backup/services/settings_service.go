package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// SettingsService provides configuration management and settings operations
// Integrates with ServiceBridge to provide settings-related events and UI updates
type SettingsService struct {
	// Core components
	logStore   *logs.Store
	eventBus   *events.EventBus
	config     *config.Config
	workingDir string

	// Service references (for status information)
	processMgr  *process.Manager
	proxyServer *proxy.Server
	mcpServer   interface {
		IsRunning() bool
		GetPort() int
	}

	// State management (using atomic operations for lock-free design)
	lastRefreshTime atomic.Value // time.Time

	// Configuration
	settingsConfig SettingsConfig

	// Context management
	ctx           context.Context
	cancel        context.CancelFunc
	subscriptions []func()
}

// SettingsConfig contains configuration for the settings service
type SettingsConfig struct {
	RefreshInterval  time.Duration `toml:"refresh_interval"`
	EnableMCPInstall bool          `toml:"enable_mcp_install"`
	ConfigPaths      []string      `toml:"config_paths"`
}

// DefaultSettingsConfig returns default settings configuration
func DefaultSettingsConfig() SettingsConfig {
	return SettingsConfig{
		RefreshInterval:  30 * time.Second,
		EnableMCPInstall: true,
		ConfigPaths: []string{
			filepath.Join(os.Getenv("HOME"), ".config", "claude", "claude_desktop_config.json"),
			filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Claude", "claude_desktop_config.json"),
			filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json"),
		},
	}
}

// NewSettingsService creates a new settings service
func NewSettingsService(
	logStore *logs.Store,
	eventBus *events.EventBus,
	config *config.Config,
	workingDir string,
	processMgr *process.Manager,
	proxyServer *proxy.Server,
	mcpServer interface {
		IsRunning() bool
		GetPort() int
	},
	settingsConfig SettingsConfig,
) *SettingsService {
	ctx, cancel := context.WithCancel(context.Background())

	ss := &SettingsService{
		logStore:       logStore,
		eventBus:       eventBus,
		config:         config,
		workingDir:     workingDir,
		processMgr:     processMgr,
		proxyServer:    proxyServer,
		mcpServer:      mcpServer,
		settingsConfig: settingsConfig,
		ctx:            ctx,
		cancel:         cancel,
		subscriptions:  []func(){},
	}

	// Initialize atomic values
	ss.lastRefreshTime.Store(time.Now())

	return ss
}

// GetSettingsInfo returns comprehensive settings information
func (ss *SettingsService) GetSettingsInfo() *messages.SettingsInfo {
	info := &messages.SettingsInfo{
		WorkingDir:  ss.workingDir,
		LastRefresh: ss.lastRefreshTime.Load().(time.Time),
	}

	// Server information
	info.ServerInfo = ss.getServerInfo()

	// Package manager information
	if ss.processMgr != nil {
		installedMgrs := ss.processMgr.GetInstalledPackageManagers()
		currentMgr := ss.processMgr.GetCurrentPackageManager()

		available := make([]string, len(installedMgrs))
		for i, mgr := range installedMgrs {
			available[i] = string(mgr.Manager)
		}

		info.PackageMgr = messages.PackageMgrInfo{
			Available: available,
			Current:   string(currentMgr),
		}
	}

	// Check MCP configuration status
	info.MCPConfigured = ss.checkMCPConfiguration()

	return info
}

// RefreshSettings refreshes settings information and emits an event
func (ss *SettingsService) RefreshSettings() {
	ss.lastRefreshTime.Store(time.Now())

	settingsInfo := ss.GetSettingsInfo()

	// Emit settings refreshed event
	ss.emitSettingsRefreshedEvent(settingsInfo)
}

// InstallMCPConfig installs MCP configuration to the default config file
func (ss *SettingsService) InstallMCPConfig() error {
	if !ss.settingsConfig.EnableMCPInstall {
		return fmt.Errorf("MCP installation is disabled")
	}

	// Find the first available config path
	for _, configPath := range ss.settingsConfig.ConfigPaths {
		if _, err := os.Stat(configPath); err == nil {
			return ss.InstallMCPConfigToFile(configPath)
		}
	}

	return fmt.Errorf("no suitable config file found")
}

// InstallMCPConfigToFile installs MCP configuration to a specific file
func (ss *SettingsService) InstallMCPConfigToFile(filePath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}

	config := mcp.GenerateBrummerConfig(execPath, 7777)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	var existingData map[string]interface{}
	if err := json.Unmarshal(data, &existingData); err != nil {
		return fmt.Errorf("error parsing JSON in %s: %w", filePath, err)
	}

	if existingData["mcpServers"] == nil {
		existingData["mcpServers"] = make(map[string]interface{})
	}

	servers := existingData["mcpServers"].(map[string]interface{})
	servers["brummer"] = map[string]interface{}{
		"command": config.Command,
		"args":    config.Args,
		"env":     config.Env,
	}

	updatedData, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}

	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", filePath, err)
	}

	// Log success
	if ss.logStore != nil {
		ss.logStore.Add("system", "System", fmt.Sprintf("✅ MCP configuration added to %s", filePath), false)
	}

	// Emit MCP installed event
	ss.emitMCPInstalledEvent(filePath)

	return nil
}

// GetServiceName returns the service name for ServiceBridge integration
func (ss *SettingsService) GetServiceName() string {
	return "settings"
}

// GetServiceStatus returns the current status of the settings service
func (ss *SettingsService) GetServiceStatus() ServiceStatus {
	status := "idle"
	details := "ready"

	// Check if MCP is configured
	if ss.checkMCPConfiguration() {
		details += " | MCP configured"
	}

	return ServiceStatus{
		Service: ss.GetServiceName(),
		Status:  status,
		Details: details,
	}
}

// SubscribeToEvents subscribes the settings service to relevant system events
func (ss *SettingsService) SubscribeToEvents() {
	if ss.eventBus == nil {
		return
	}

	// Subscribe to events that might affect settings
	ss.eventBus.Subscribe(events.ProcessStarted, func(evt events.Event) {
		// Process start might affect available package managers
		ss.RefreshSettings()
	})

	ss.eventBus.Subscribe(events.ProcessExited, func(evt events.Event) {
		// Process exit might affect available package managers
		ss.RefreshSettings()
	})

	ss.eventBus.Subscribe(events.MCPConnected, func(evt events.Event) {
		// MCP connection status changed
		ss.RefreshSettings()
	})

	ss.eventBus.Subscribe(events.MCPDisconnected, func(evt events.Event) {
		// MCP connection status changed
		ss.RefreshSettings()
	})
}

// Stop stops the settings service and cleans up resources
func (ss *SettingsService) Stop() error {
	// Cancel context
	ss.cancel()

	// Clean up event subscriptions
	for _, unsub := range ss.subscriptions {
		if unsub != nil {
			unsub()
		}
	}
	ss.subscriptions = nil

	return nil
}

// Helper methods

func (ss *SettingsService) getServerInfo() messages.ServerInfo {
	info := messages.ServerInfo{}

	// MCP Server info
	if ss.mcpServer != nil {
		mcpPort := 7777
		mcpStatus := "🔴 Not Running"

		if ss.mcpServer.IsRunning() {
			mcpStatus = "🟢 Running"
			mcpPort = ss.mcpServer.GetPort()
		}

		info.MCPServer = messages.MCPServerInfo{
			Running: ss.mcpServer.IsRunning(),
			Port:    mcpPort,
			URL:     fmt.Sprintf("http://localhost:%d/mcp", mcpPort),
			Status:  mcpStatus,
		}
	}

	// Proxy Server info
	if ss.proxyServer != nil && ss.proxyServer.IsRunning() && ss.proxyServer.GetMode() == proxy.ProxyModeFull {
		info.ProxyServer = messages.ProxyServerInfo{
			Running: ss.proxyServer.IsRunning(),
			Port:    ss.proxyServer.GetPort(),
			Mode:    string(ss.proxyServer.GetMode()),
			PacURL:  ss.proxyServer.GetPACURL(),
		}
	}

	return info
}

func (ss *SettingsService) checkMCPConfiguration() bool {
	// Check if MCP is configured in any of the known config files
	for _, configPath := range ss.settingsConfig.ConfigPaths {
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				continue
			}

			var configData map[string]interface{}
			if err := json.Unmarshal(data, &configData); err != nil {
				continue
			}

			if servers, ok := configData["mcpServers"].(map[string]interface{}); ok {
				if _, exists := servers["brummer"]; exists {
					return true
				}
			}
		}
	}
	return false
}

// Event emission methods

func (ss *SettingsService) emitSettingsRefreshedEvent(info *messages.SettingsInfo) {
	if ss.eventBus != nil {
		ss.eventBus.Publish(events.Event{
			Type: "settings.refreshed",
			Data: map[string]interface{}{
				"settings_info": info,
			},
		})
	}
}

func (ss *SettingsService) emitMCPInstalledEvent(filePath string) {
	if ss.eventBus != nil {
		ss.eventBus.Publish(events.Event{
			Type: "settings.mcp_installed",
			Data: map[string]interface{}{
				"config_file": filePath,
			},
		})
	}
}
