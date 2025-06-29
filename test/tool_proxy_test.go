package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolProxyIntegration(t *testing.T) {
	
	// Create connection manager
	connMgr := mcp.NewConnectionManager()
	defer connMgr.Stop()
	
	// Create a test instance
	instance := &discovery.Instance{
		ID:        "test-instance",
		Name:      "Test Instance",
		Directory: "/test/dir",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	
	// Register the instance
	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	
	// Create mock tool info
	toolInfo := mcp.ToolInfo{
		Name:        "test/tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type": "object"}`),
	}
	
	// Create proxy tool
	proxyTool := mcp.ProxyTool("test-instance", toolInfo, connMgr)
	
	// Verify tool properties
	assert.Equal(t, "test-instance_test/tool", proxyTool.Name)
	assert.Contains(t, proxyTool.Description, "[test-instance]")
	assert.Contains(t, proxyTool.Description, "A test tool")
	assert.NotNil(t, proxyTool.Handler)
	assert.False(t, proxyTool.Streaming)
}

func TestExtractInstanceAndTool(t *testing.T) {
	tests := []struct {
		name          string
		prefixedName  string
		wantInstance  string
		wantTool      string
		wantErr       bool
	}{
		{
			name:         "valid prefixed name",
			prefixedName: "instance-123_scripts/run",
			wantInstance: "instance-123",
			wantTool:     "scripts/run",
			wantErr:      false,
		},
		{
			name:         "no prefix", 
			prefixedName: "scriptsrun",
			wantInstance: "",
			wantTool:     "",
			wantErr:      true,
		},
		{
			name:         "empty string",
			prefixedName: "",
			wantInstance: "",
			wantTool:     "",
			wantErr:      true,
		},
		{
			name:         "tool with underscore",
			prefixedName: "instance-123_scripts_run",
			wantInstance: "instance-123",
			wantTool:     "scripts_run",
			wantErr:      false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceID, toolName, err := mcp.ExtractInstanceAndTool(tt.prefixedName)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantInstance, instanceID)
				assert.Equal(t, tt.wantTool, toolName)
			}
		})
	}
}

func TestRegisterInstanceTools(t *testing.T) {
	// This test would require a more complete mock setup
	// including a mock HubClient that returns tools
	// For now, we'll skip the full integration test
	t.Skip("Requires full mock setup")
}