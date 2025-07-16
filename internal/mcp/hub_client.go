package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HubClient is an HTTP client for connecting to instance MCP servers
type HubClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHubClient creates a new HTTP client for connecting to an instance
func NewHubClient(port int) (*HubClient, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}
	return &HubClient{
		baseURL: fmt.Sprintf("http://localhost:%d/mcp", port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Initialize sends the initialize request to establish connection
func (c *HubClient) Initialize(ctx context.Context) error {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "1.0",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]string{
				"name":    "brummer-hub",
				"version": "1.0",
			},
		},
	}

	_, err := c.sendRequest(ctx, request)
	return err
}

// CallTool invokes a tool on the instance
func (c *HubClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(), // Unique ID
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	return c.sendRequest(ctx, request)
}

// ListTools gets the list of available tools from the instance
func (c *HubClient) ListTools(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	return c.sendRequest(ctx, request)
}

// ListResources gets the list of available resources from the instance
func (c *HubClient) ListResources(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "resources/list",
		"params":  map[string]interface{}{},
	}

	return c.sendRequest(ctx, request)
}

// ReadResource reads a specific resource from the instance
func (c *HubClient) ReadResource(ctx context.Context, uri string) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": uri,
		},
	}

	return c.sendRequest(ctx, request)
}

// ListPrompts gets the list of available prompts from the instance
func (c *HubClient) ListPrompts(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "prompts/list",
		"params":  map[string]interface{}{},
	}

	return c.sendRequest(ctx, request)
}

// GetPrompt gets a specific prompt from the instance
func (c *HubClient) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "prompts/get",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}

	return c.sendRequest(ctx, request)
}

// Ping sends a ping to check if the connection is alive
func (c *HubClient) Ping(ctx context.Context) error {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "ping",
		"params":  map[string]interface{}{},
	}

	_, err := c.sendRequest(ctx, request)
	return err
}

// sendRequest sends a JSON-RPC request and returns the result
func (c *HubClient) sendRequest(ctx context.Context, request interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data,omitempty"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
	}

	return result.Result, nil
}

// Close closes the HTTP client
func (c *HubClient) Close() error {
	// Nothing to close for HTTP client
	return nil
}
