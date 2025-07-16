package mcp

import (
	"context"
	"encoding/json"
	"os"

	"github.com/standardbeagle/brummer/internal/config"
)

// HubClientInterface defines the common interface for both persistent and regular hub clients
type HubClientInterface interface {
	Initialize(ctx context.Context) error
	CallTool(ctx context.Context, toolName string, args map[string]interface{}) (json.RawMessage, error)
	ListTools(ctx context.Context) (json.RawMessage, error)
	ListResources(ctx context.Context) (json.RawMessage, error)
	ReadResource(ctx context.Context, uri string) (json.RawMessage, error)
	ListPrompts(ctx context.Context) (json.RawMessage, error)
	GetPrompt(ctx context.Context, name string, args map[string]interface{}) (json.RawMessage, error)
	Ping(ctx context.Context) error
	Close() error
}

// NewHubClientInterface creates the appropriate client based on configuration
func NewHubClientInterface(port int) (HubClientInterface, error) {
	// Check environment variable first for easier testing
	if os.Getenv("BRUMMER_USE_ROBUST_NETWORKING") == "true" {
		return NewPersistentHubClient(port)
	}

	// Check config file
	cfg, err := config.Load()
	if err != nil {
		// Fall back to regular client on config error
		return NewHubClient(port)
	}

	if cfg.GetUseRobustNetworking() {
		return NewPersistentHubClient(port)
	}

	return NewHubClient(port)
}