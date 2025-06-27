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
	"sync/atomic"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// ImprovedHubClient adds timeout handling, retries, and circuit breaker
type ImprovedHubClient struct {
	baseURL    string
	httpClient *http.Client
	
	// Circuit breaker
	circuitMu       sync.RWMutex
	circuitState    CircuitState
	failureCount    atomic.Int32
	lastFailureTime atomic.Int64
	successCount    atomic.Int32
	
	// Configuration
	config ClientConfig
	
	// Metrics
	totalRequests   atomic.Uint64
	failedRequests  atomic.Uint64
	timeoutRequests atomic.Uint64
}

// ClientConfig configures the improved hub client
type ClientConfig struct {
	// Timeouts
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
	
	// Retries
	MaxRetries    int
	RetryDelay    time.Duration
	RetryBackoff  float64
	
	// Circuit breaker
	FailureThreshold   int
	RecoveryTimeout    time.Duration
	HalfOpenSuccesses  int
	
	// Connection pooling
	MaxIdleConns        int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	DisableKeepAlives   bool
}

// DefaultClientConfig provides sensible defaults
var DefaultClientConfig = ClientConfig{
	ConnectTimeout:     5 * time.Second,
	RequestTimeout:     30 * time.Second,
	MaxRetries:         3,
	RetryDelay:         100 * time.Millisecond,
	RetryBackoff:       2.0,
	FailureThreshold:   5,
	RecoveryTimeout:    30 * time.Second,
	HalfOpenSuccesses:  3,
	MaxIdleConns:       10,
	MaxConnsPerHost:    2,
	IdleConnTimeout:    90 * time.Second,
	DisableKeepAlives:  false,
}

// NewImprovedHubClient creates a robust hub client
func NewImprovedHubClient(port int, config *ClientConfig) (*ImprovedHubClient, error) {
	if config == nil {
		config = &DefaultClientConfig
	}
	
	// Create transport with timeouts
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   config.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		DisableKeepAlives:     config.DisableKeepAlives,
		ResponseHeaderTimeout: config.RequestTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
	
	return &ImprovedHubClient{
		baseURL: fmt.Sprintf("http://localhost:%d/mcp", port),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.RequestTimeout,
		},
		config:       *config,
		circuitState: CircuitClosed,
	}, nil
}

// Initialize sends the initialize request with retries
func (c *ImprovedHubClient) Initialize(ctx context.Context) error {
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
	
	_, err := c.sendRequestWithRetry(ctx, request)
	return err
}

// CallTool invokes a tool with timeout and retries
func (c *ImprovedHubClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (json.RawMessage, error) {
	// Apply per-operation timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.RequestTimeout)
		defer cancel()
	}
	
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}
	
	return c.sendRequestWithRetry(ctx, request)
}

// sendRequestWithRetry implements retry logic with exponential backoff
func (c *ImprovedHubClient) sendRequestWithRetry(ctx context.Context, request interface{}) (json.RawMessage, error) {
	delay := c.config.RetryDelay
	
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Check circuit breaker
		if !c.canMakeRequest() {
			c.failedRequests.Add(1)
			return nil, fmt.Errorf("circuit breaker is open")
		}
		
		result, err := c.sendRequest(ctx, request)
		if err == nil {
			c.recordSuccess()
			return result, nil
		}
		
		// Check if error is retryable
		if !isRetryableError(err) {
			c.recordFailure()
			return nil, err
		}
		
		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			c.recordFailure()
			return nil, ctx.Err()
		}
		
		// Last attempt, don't wait
		if attempt == c.config.MaxRetries {
			c.recordFailure()
			return nil, err
		}
		
		// Wait with exponential backoff
		select {
		case <-time.After(delay):
			delay = time.Duration(float64(delay) * c.config.RetryBackoff)
		case <-ctx.Done():
			c.recordFailure()
			return nil, ctx.Err()
		}
	}
	
	c.recordFailure()
	return nil, fmt.Errorf("max retries exceeded")
}

// sendRequest sends a single request without retries
func (c *ImprovedHubClient) sendRequest(ctx context.Context, request interface{}) (json.RawMessage, error) {
	c.totalRequests.Add(1)
	
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
	
	// Add request ID for tracing
	if reqMap, ok := request.(map[string]interface{}); ok {
		if id, ok := reqMap["id"]; ok {
			req.Header.Set("X-Request-ID", fmt.Sprint(id))
		}
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			c.timeoutRequests.Add(1)
		}
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read body with size limit
	bodyReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB limit
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
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
	
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	if result.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
	}
	
	return result.Result, nil
}

// Circuit breaker methods

func (c *ImprovedHubClient) canMakeRequest() bool {
	c.circuitMu.RLock()
	state := c.circuitState
	c.circuitMu.RUnlock()
	
	switch state {
	case CircuitClosed:
		return true
		
	case CircuitOpen:
		// Check if recovery timeout has passed
		lastFailure := time.Unix(0, c.lastFailureTime.Load())
		if time.Since(lastFailure) > c.config.RecoveryTimeout {
			c.transitionToHalfOpen()
			return true
		}
		return false
		
	case CircuitHalfOpen:
		return true
		
	default:
		return false
	}
}

func (c *ImprovedHubClient) recordSuccess() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()
	
	c.failureCount.Store(0)
	
	if c.circuitState == CircuitHalfOpen {
		successCount := c.successCount.Add(1)
		if successCount >= int32(c.config.HalfOpenSuccesses) {
			c.circuitState = CircuitClosed
			c.successCount.Store(0)
		}
	}
}

func (c *ImprovedHubClient) recordFailure() {
	c.failedRequests.Add(1)
	failures := c.failureCount.Add(1)
	c.lastFailureTime.Store(time.Now().UnixNano())
	
	if failures >= int32(c.config.FailureThreshold) {
		c.transitionToOpen()
	}
}

func (c *ImprovedHubClient) transitionToOpen() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()
	
	c.circuitState = CircuitOpen
	c.successCount.Store(0)
}

func (c *ImprovedHubClient) transitionToHalfOpen() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()
	
	c.circuitState = CircuitHalfOpen
	c.successCount.Store(0)
	c.failureCount.Store(0)
}

// GetMetrics returns client metrics
func (c *ImprovedHubClient) GetMetrics() map[string]interface{} {
	c.circuitMu.RLock()
	state := c.circuitState
	c.circuitMu.RUnlock()
	
	stateStr := "unknown"
	switch state {
	case CircuitClosed:
		stateStr = "closed"
	case CircuitOpen:
		stateStr = "open"
	case CircuitHalfOpen:
		stateStr = "half-open"
	}
	
	return map[string]interface{}{
		"total_requests":   c.totalRequests.Load(),
		"failed_requests":  c.failedRequests.Load(),
		"timeout_requests": c.timeoutRequests.Load(),
		"circuit_state":    stateStr,
		"failure_count":    c.failureCount.Load(),
	}
}

// Close closes the HTTP client and cleans up resources
func (c *ImprovedHubClient) Close() error {
	// Close idle connections
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// Helper functions

func isRetryableError(err error) bool {
	// Network errors are retryable
	if _, ok := err.(net.Error); ok {
		return true
	}
	
	// Specific HTTP status codes are retryable
	// (would need to parse error message or enhance error types)
	
	return false
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

// Additional methods for other MCP operations
func (c *ImprovedHubClient) ListTools(ctx context.Context) (json.RawMessage, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}
	
	return c.sendRequestWithRetry(ctx, request)
}

func (c *ImprovedHubClient) Ping(ctx context.Context) error {
	// Use shorter timeout for ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "ping",
		"params":  map[string]interface{}{},
	}
	
	_, err := c.sendRequestWithRetry(pingCtx, request)
	return err
}