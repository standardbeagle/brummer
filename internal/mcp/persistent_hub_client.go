package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// PersistentHubClient is an enhanced HTTP client with persistent connections for long-running sessions
type PersistentHubClient struct {
	baseURL     string
	transport   *http.Transport
	client      *http.Client
	connMu      sync.Mutex
	established bool
}

// NewPersistentHubClient creates a new persistent HTTP client for connecting to an instance
func NewPersistentHubClient(port int) (*PersistentHubClient, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}

	// Create transport optimized for long-running persistent connections
	transport := &http.Transport{
		// Connection pooling settings
		MaxIdleConns:        10,                // Total idle connections across all hosts
		MaxIdleConnsPerHost: 1,                 // Only 1 persistent connection per instance
		IdleConnTimeout:     24 * time.Hour,    // Keep connections alive for hours
		MaxConnsPerHost:     2,                 // Limit total connections per host

		// Keep-alive settings for long-running connections
		DisableKeepAlives: false,

		// Connection timeouts
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,

		// Network interruption handling
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			conn, err := d.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			
			// Configure socket options for better connection stability
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				tcpConn.SetKeepAlive(true)
				tcpConn.SetKeepAlivePeriod(30 * time.Second)
			}
			
			return conn, nil
		},

		// Optimize for persistent connections
		DisableCompression: false,
		ForceAttemptHTTP2:  false, // Stick with HTTP/1.1 for better multiplexing control
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   0, // No global timeout for persistent connections
	}

	return &PersistentHubClient{
		baseURL:     fmt.Sprintf("http://localhost:%d/mcp", port),
		transport:   transport,
		client:      client,
		established: false,
	}, nil
}

// Initialize sends the initialize request to establish connection
func (c *PersistentHubClient) Initialize(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

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
				"name":    "brummer-hub-persistent",
				"version": "1.0",
			},
		},
	}

	// Use shorter timeout for initialization
	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.sendRequestWithContext(initCtx, request)
	if err != nil {
		return fmt.Errorf("initialize connection: %w", err)
	}

	c.established = true
	return nil
}

// CallTool invokes a tool on the instance with request multiplexing
func (c *PersistentHubClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(), // Unique ID for request multiplexing
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	return c.sendRequestWithContext(ctx, request)
}

// ListTools gets the list of available tools from the instance
func (c *PersistentHubClient) ListTools(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	return c.sendRequestWithContext(ctx, request)
}

// ListResources gets the list of available resources from the instance
func (c *PersistentHubClient) ListResources(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "resources/list",
		"params":  map[string]interface{}{},
	}

	return c.sendRequestWithContext(ctx, request)
}

// ReadResource reads a specific resource from the instance
func (c *PersistentHubClient) ReadResource(ctx context.Context, uri string) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": uri,
		},
	}

	return c.sendRequestWithContext(ctx, request)
}

// ListPrompts gets the list of available prompts from the instance
func (c *PersistentHubClient) ListPrompts(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "prompts/list",
		"params":  map[string]interface{}{},
	}

	return c.sendRequestWithContext(ctx, request)
}

// GetPrompt gets a specific prompt from the instance
func (c *PersistentHubClient) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "prompts/get",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}

	return c.sendRequestWithContext(ctx, request)
}

// Ping sends a ping to check if the persistent connection is alive
func (c *PersistentHubClient) Ping(ctx context.Context) error {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "ping",
		"params":  map[string]interface{}{},
	}

	// Use shorter timeout for ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.sendRequestWithContext(pingCtx, request)
	return err
}

// sendRequestWithContext sends a JSON-RPC request with proper context handling
func (c *PersistentHubClient) sendRequestWithContext(ctx context.Context, request interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers to maintain persistent connections
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.client.Do(req)
	if err != nil {
		// Check if this is a connection-related error
		if netErr, ok := err.(net.Error); ok {
			return nil, fmt.Errorf("network error: %w", netErr)
		}
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

// IsEstablished returns whether the connection has been established
func (c *PersistentHubClient) IsEstablished() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.established
}

// Close closes the persistent HTTP client and cleans up connections
func (c *PersistentHubClient) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
	c.established = false
	return nil
}

// GetConnectionStats returns statistics about the persistent connection
func (c *PersistentHubClient) GetConnectionStats() map[string]interface{} {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	return map[string]interface{}{
		"established":      c.established,
		"baseURL":         c.baseURL,
		"maxIdleConns":    c.transport.MaxIdleConns,
		"maxConnsPerHost": c.transport.MaxConnsPerHost,
		"idleConnTimeout": c.transport.IdleConnTimeout.String(),
	}
}