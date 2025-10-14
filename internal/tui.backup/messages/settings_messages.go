package messages

import (
	"time"
)

// Settings-related messages for the TUI system

// SettingsInfo represents comprehensive settings information
type SettingsInfo struct {
	ServerInfo    ServerInfo     `json:"server_info"`
	PackageMgr    PackageMgrInfo `json:"package_manager"`
	WorkingDir    string         `json:"working_directory"`
	LastRefresh   time.Time      `json:"last_refresh"`
	MCPConfigured bool           `json:"mcp_configured"`
}

// ServerInfo contains server-related settings
type ServerInfo struct {
	MCPServer   MCPServerInfo   `json:"mcp_server"`
	ProxyServer ProxyServerInfo `json:"proxy_server,omitempty"`
}

// MCPServerInfo contains MCP server information
type MCPServerInfo struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
	Status  string `json:"status"`
}

// ProxyServerInfo contains proxy server information
type ProxyServerInfo struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Mode    string `json:"mode"`
	PacURL  string `json:"pac_url,omitempty"`
}

// PackageMgrInfo contains package manager information
type PackageMgrInfo struct {
	Available []string `json:"available"`
	Current   string   `json:"current"`
}

// SettingsRefreshedMsg indicates settings information has been refreshed
type SettingsRefreshedMsg struct {
	SettingsInfo *SettingsInfo
	Timestamp    time.Time
}

// SettingsMCPInstalledMsg indicates MCP configuration has been installed
type SettingsMCPInstalledMsg struct {
	ConfigFile string
	Timestamp  time.Time
}

// SettingsErrorMsg indicates a settings operation error
type SettingsErrorMsg struct {
	Operation string
	Error     string
	Timestamp time.Time
}

// SettingsUpdateMsg requests a settings update
type SettingsUpdateMsg struct {
	Refresh    bool
	InstallMCP bool
	ConfigFile string
}

// SettingsStatusUpdateMsg indicates settings service status change
type SettingsStatusUpdateMsg struct {
	Status    string
	Details   string
	Timestamp time.Time
}
