package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// NetworkCommand represents a network operation command
type NetworkCommand struct {
	*BaseCommand
	request     *HTTPRequest
	operationType NetworkOperationType
}

// NetworkOperationType represents the type of network operation
type NetworkOperationType int

const (
	NetworkGET NetworkOperationType = iota
	NetworkPOST
	NetworkPUT
	NetworkDELETE
	NetworkHEAD
	NetworkOPTIONS
	NetworkPATCH
)

// NetworkResultMsg represents the result of a network command
type NetworkResultMsg struct {
	ID           string             `json:"id"`
	Type         CommandType        `json:"type"`
	Result       interface{}        `json:"result,omitempty"`
	Error        error              `json:"error,omitempty"`
	Context      OperationContext   `json:"context"`
	Duration     time.Duration      `json:"duration"`
	OperationType NetworkOperationType `json:"operationType"`
	URL          string             `json:"url"`
	StatusCode   int                `json:"statusCode,omitempty"`
}

// HTTPRequest represents an HTTP request
type HTTPRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        interface{}        `json:"body,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	FollowRedirects bool           `json:"followRedirects"`
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int               `json:"statusCode"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}        `json:"body,omitempty"`
	Duration   time.Duration     `json:"duration"`
	Size       int64             `json:"size"`
	Request    HTTPRequest       `json:"request"`
}

// NewNetworkCommand creates a new network command
func NewNetworkCommand(id string, request *HTTPRequest, opType NetworkOperationType, context OperationContext) *NetworkCommand {
	return &NetworkCommand{
		BaseCommand:   NewBaseCommand(id, CommandTypeNetwork, PriorityMedium, context),
		request:       request,
		operationType: opType,
	}
}

// Run executes the network command
func (c *NetworkCommand) Run() tea.Msg {
	start := time.Now()

	// Create command context for timeout handling
	cmdCtx := NewCommandContext(c.Context())
	defer cmdCtx.Cleanup()

	var result interface{}
	var err error

	// Execute HTTP request with timeout
	result, err = c.executeHTTPRequest(cmdCtx)
	duration := time.Since(start)

	// Check for timeout
	if cmdCtx.IsTimedOut() {
		err = fmt.Errorf("network operation timed out after %v", duration)
	}

	return NetworkResultMsg{
		ID:           c.ID(),
		Type:         c.Type(),
		Result:       result,
		Error:        err,
		Context:      c.Context(),
		Duration:     duration,
		OperationType: c.operationType,
		URL:          c.request.URL,
	}
}

// executeHTTPRequest executes the HTTP request
func (c *NetworkCommand) executeHTTPRequest(cmdCtx *CommandContext) (*HTTPResponse, error) {
	// Parse URL
	parsedURL, err := url.Parse(c.request.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", c.request.URL, err)
	}

	// Create request body
	var body io.Reader
	if c.request.Body != nil {
		switch v := c.request.Body.(type) {
		case string:
			body = strings.NewReader(v)
		case []byte:
			body = bytes.NewReader(v)
		default:
			// Try to marshal as JSON
			jsonBody, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = bytes.NewReader(jsonBody)
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(cmdCtx.Context(), c.request.Method, parsedURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range c.request.Headers {
		req.Header.Set(key, value)
	}

	// Set default headers
	if req.Header.Get("Content-Type") == "" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Brummer/1.0")
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: c.request.Timeout,
	}

	// Configure redirects
	if !c.request.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Execute request
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response body
	var parsedBody interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonBody interface{}
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			parsedBody = jsonBody
		} else {
			parsedBody = string(respBody)
		}
	} else {
		parsedBody = string(respBody)
	}

	// Build response headers map
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       parsedBody,
		Duration:   duration,
		Size:       int64(len(respBody)),
		Request:    *c.request,
	}, nil
}

// Factory functions for common network operations

// NewGETCommand creates a GET request command
func NewGETCommand(id string, url string, headers map[string]string, timeout time.Duration, context OperationContext) *NetworkCommand {
	request := &HTTPRequest{
		Method:      "GET",
		URL:         url,
		Headers:     headers,
		Timeout:     timeout,
		FollowRedirects: true,
	}
	return NewNetworkCommand(id, request, NetworkGET, context)
}

// NewPOSTCommand creates a POST request command
func NewPOSTCommand(id string, url string, body interface{}, headers map[string]string, timeout time.Duration, context OperationContext) *NetworkCommand {
	request := &HTTPRequest{
		Method:      "POST",
		URL:         url,
		Headers:     headers,
		Body:        body,
		Timeout:     timeout,
		FollowRedirects: true,
	}
	return NewNetworkCommand(id, request, NetworkPOST, context)
}

// NewPUTCommand creates a PUT request command
func NewPUTCommand(id string, url string, body interface{}, headers map[string]string, timeout time.Duration, context OperationContext) *NetworkCommand {
	request := &HTTPRequest{
		Method:      "PUT",
		URL:         url,
		Headers:     headers,
		Body:        body,
		Timeout:     timeout,
		FollowRedirects: true,
	}
	return NewNetworkCommand(id, request, NetworkPUT, context)
}

// NewDELETECommand creates a DELETE request command
func NewDELETECommand(id string, url string, headers map[string]string, timeout time.Duration, context OperationContext) *NetworkCommand {
	request := &HTTPRequest{
		Method:      "DELETE",
		URL:         url,
		Headers:     headers,
		Timeout:     timeout,
		FollowRedirects: true,
	}
	return NewNetworkCommand(id, request, NetworkDELETE, context)
}

// NewHealthCheckCommand creates a health check command
func NewHealthCheckCommand(id string, url string, timeout time.Duration, context OperationContext) *NetworkCommand {
	headers := map[string]string{
		"Accept": "application/json",
	}
	return NewGETCommand(id, url, headers, timeout, context)
}

// NewAPICallCommand creates a general API call command
func NewAPICallCommand(id string, method string, url string, body interface{}, headers map[string]string, timeout time.Duration, context OperationContext) *NetworkCommand {
	request := &HTTPRequest{
		Method:      method,
		URL:         url,
		Headers:     headers,
		Body:        body,
		Timeout:     timeout,
		FollowRedirects: true,
	}

	var opType NetworkOperationType
	switch strings.ToUpper(method) {
	case "GET":
		opType = NetworkGET
	case "POST":
		opType = NetworkPOST
	case "PUT":
		opType = NetworkPUT
	case "DELETE":
		opType = NetworkDELETE
	case "HEAD":
		opType = NetworkHEAD
	case "OPTIONS":
		opType = NetworkOPTIONS
	case "PATCH":
		opType = NetworkPATCH
	default:
		opType = NetworkGET
	}

	return NewNetworkCommand(id, request, opType, context)
}

// Network utility functions

func IsValidURL(rawURL string) bool {
	_, err := url.ParseRequestURI(rawURL)
	return err == nil
}

func GetURLHost(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsed.Host, nil
}

func IsHTTPSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func IsHTTPRedirect(statusCode int) bool {
	return statusCode >= 300 && statusCode < 400
}

func IsHTTPClientError(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500
}

func IsHTTPServerError(statusCode int) bool {
	return statusCode >= 500
}

// Network error types

type NetworkError struct {
	URL        string        `json:"url"`
	Method     string        `json:"method"`
	StatusCode int           `json:"statusCode"`
	Message    string        `json:"message"`
	Duration   time.Duration `json:"duration"`
	Retryable  bool          `json:"retryable"`
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("Network error: %s %s -> %d: %s (took %v)",
		e.Method, e.URL, e.StatusCode, e.Message, e.Duration)
}

type TimeoutError struct {
	URL      string        `json:"url"`
	Method   string        `json:"method"`
	Timeout  time.Duration `json:"timeout"`
	Duration time.Duration `json:"duration"`
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("Network timeout: %s %s timed out after %v (took %v)",
		e.Method, e.URL, e.Timeout, e.Duration)
}

type ConnectionError struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Cause  string `json:"cause"`
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("Connection error: %s %s -> %s", e.Method, e.URL, e.Cause)
}

// Message interface implementation
func (m NetworkResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the network operation completed successfully
func (m NetworkResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the network operation failed
func (m NetworkResultMsg) IsFailure() bool {
	return m.Error != nil
}

// IsSuccessStatus returns true if the HTTP status code indicates success
func (m NetworkResultMsg) IsSuccessStatus() bool {
	if resp, ok := m.Result.(*HTTPResponse); ok {
		return IsHTTPSuccess(resp.StatusCode)
	}
	return false
}

// String returns a string representation of the network result
func (m NetworkResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("NetworkResult{id=%s, type=%v, url=%s, duration=%v}",
			m.ID, m.OperationType, m.URL, m.Duration)
	}
	return fmt.Sprintf("NetworkResult{id=%s, type=%v, url=%s, error=%v, duration=%v}",
		m.ID, m.OperationType, m.URL, m.Error, m.Duration)
}

// HTTP request builder
type HTTPRequestBuilder struct {
	request *HTTPRequest
}

func NewHTTPRequestBuilder() *HTTPRequestBuilder {
	return &HTTPRequestBuilder{
		request: &HTTPRequest{
			Method:         "GET",
			Timeout:        30 * time.Second,
			FollowRedirects: true,
			Headers:        make(map[string]string),
		},
	}
}

func (b *HTTPRequestBuilder) Method(method string) *HTTPRequestBuilder {
	b.request.Method = method
	return b
}

func (b *HTTPRequestBuilder) URL(url string) *HTTPRequestBuilder {
	b.request.URL = url
	return b
}

func (b *HTTPRequestBuilder) Header(key, value string) *HTTPRequestBuilder {
	b.request.Headers[key] = value
	return b
}

func (b *HTTPRequestBuilder) Headers(headers map[string]string) *HTTPRequestBuilder {
	for k, v := range headers {
		b.request.Headers[k] = v
	}
	return b
}

func (b *HTTPRequestBuilder) Body(body interface{}) *HTTPRequestBuilder {
	b.request.Body = body
	return b
}

func (b *HTTPRequestBuilder) Timeout(timeout time.Duration) *HTTPRequestBuilder {
	b.request.Timeout = timeout
	return b
}

func (b *HTTPRequestBuilder) FollowRedirects(follow bool) *HTTPRequestBuilder {
	b.request.FollowRedirects = follow
	return b
}

func (b *HTTPRequestBuilder) Build() *HTTPRequest {
	return b.request
}