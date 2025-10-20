package state

import (
	"time"
)

// ProxyEvent represents a single HTTP request/response captured by proxy
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
type ProxyEvent struct {
	ID             string            `json:"id"`               // Unique event identifier
	Timestamp      time.Time         `json:"timestamp"`        // When request was captured
	Method         string            `json:"method"`          // HTTP method (GET, POST, etc.)
	URL            string            `json:"url"`             // Request URL
	StatusCode     int               `json:"status_code"`     // Response status code
	RequestHeaders map[string]string `json:"request_headers"` // HTTP request headers
	ResponseHeaders map[string]string `json:"response_headers"`// HTTP response headers
	RequestBody    string            `json:"request_body"`    // Request body (truncated if large)
	ResponseBody   string            `json:"response_body"`   // Response body (truncated if large)
	Duration       time.Duration     `json:"duration"`        // Request/response round-trip time
	Size           int               `json:"size"`            // Response body size in bytes
}

// NewProxyEvent creates a new ProxyEvent with the given parameters
func NewProxyEvent(id string, method, url string, statusCode int) ProxyEvent {
	return ProxyEvent{
		ID:             id,
		Timestamp:      time.Now(),
		Method:         method,
		URL:            url,
		StatusCode:     statusCode,
		RequestHeaders: make(map[string]string),
		ResponseHeaders: make(map[string]string),
		RequestBody:    "",
		ResponseBody:   "",
		Duration:       0,
		Size:           0,
	}
}

// WithRequestHeaders sets the request headers (immutable)
func (pe ProxyEvent) WithRequestHeaders(headers map[string]string) ProxyEvent {
	newHeaders := make(map[string]string)
	for k, v := range headers {
		newHeaders[k] = v
	}

	newEvent := pe
	newEvent.RequestHeaders = newHeaders
	return newEvent
}

// WithResponseHeaders sets the response headers (immutable)
func (pe ProxyEvent) WithResponseHeaders(headers map[string]string) ProxyEvent {
	newHeaders := make(map[string]string)
	for k, v := range headers {
		newHeaders[k] = v
	}

	newEvent := pe
	newEvent.ResponseHeaders = newHeaders
	return newEvent
}

// WithRequestBody sets the request body (immutable)
func (pe ProxyEvent) WithRequestBody(body string) ProxyEvent {
	// Truncate if too large
	maxBodySize := 10000
	if len(body) > maxBodySize {
		body = body[:maxBodySize] + "...[truncated]"
	}

	newEvent := pe
	newEvent.RequestBody = body
	return newEvent
}

// WithResponseBody sets the response body (immutable)
func (pe ProxyEvent) WithResponseBody(body string) ProxyEvent {
	// Truncate if too large
	maxBodySize := 10000
	if len(body) > maxBodySize {
		body = body[:maxBodySize] + "...[truncated]"
	}

	newEvent := pe
	newEvent.ResponseBody = body
	newEvent.Size = len(body)
	return newEvent
}

// WithDuration sets the duration (immutable)
func (pe ProxyEvent) WithDuration(duration time.Duration) ProxyEvent {
	newEvent := pe
	newEvent.Duration = duration
	return newEvent
}

// WithTimestamp sets the timestamp (immutable)
func (pe ProxyEvent) WithTimestamp(timestamp time.Time) ProxyEvent {
	newEvent := pe
	newEvent.Timestamp = timestamp
	return newEvent
}

// GetContentType returns the content type from response headers
func (pe ProxyEvent) GetContentType() string {
	if contentType, exists := pe.ResponseHeaders["Content-Type"]; exists {
		return contentType
	}
	if contentType, exists := pe.ResponseHeaders["content-type"]; exists {
		return contentType
	}
	return ""
}

// IsSuccess returns true if the status code indicates success (2xx)
func (pe ProxyEvent) IsSuccess() bool {
	return pe.StatusCode >= 200 && pe.StatusCode < 300
}

// IsError returns true if the status code indicates an error (4xx, 5xx)
func (pe ProxyEvent) IsError() bool {
	return pe.StatusCode >= 400
}

// IsRedirect returns true if the status code indicates a redirect (3xx)
func (pe ProxyEvent) IsRedirect() bool {
	return pe.StatusCode >= 300 && pe.StatusCode < 400
}

// GetStatusCategory returns a human-readable category for the status code
func (pe ProxyEvent) GetStatusCategory() string {
	if pe.IsSuccess() {
		return "Success"
	} else if pe.IsError() {
		if pe.StatusCode >= 500 {
			return "Server Error"
		}
		return "Client Error"
	} else if pe.IsRedirect() {
		return "Redirect"
	} else if pe.StatusCode == 100 || pe.StatusCode == 101 {
		return "Informational"
	}
	return "Unknown"
}

// Validate checks if the ProxyEvent has valid data
func (pe ProxyEvent) Validate() []string {
	var issues []string

	if pe.ID == "" {
		issues = append(issues, "ID cannot be empty")
	}

	if pe.Method == "" {
		issues = append(issues, "Method cannot be empty")
	}

	if pe.URL == "" {
		issues = append(issues, "URL cannot be empty")
	}

	if pe.StatusCode < 100 || pe.StatusCode > 599 {
		issues = append(issues, "StatusCode must be between 100 and 599")
	}

	if pe.Timestamp.IsZero() {
		issues = append(issues, "Timestamp cannot be zero time")
	}

	if pe.Duration < 0 {
		issues = append(issues, "Duration cannot be negative")
	}

	if pe.Size < 0 {
		issues = append(issues, "Size cannot be negative")
	}

	return issues
}

// Copy creates a deep copy of the ProxyEvent
func (pe ProxyEvent) Copy() ProxyEvent {
	// Copy request headers
	newRequestHeaders := make(map[string]string)
	for k, v := range pe.RequestHeaders {
		newRequestHeaders[k] = v
	}

	// Copy response headers
	newResponseHeaders := make(map[string]string)
	for k, v := range pe.ResponseHeaders {
		newResponseHeaders[k] = v
	}

	return ProxyEvent{
		ID:             pe.ID,
		Timestamp:      pe.Timestamp,
		Method:         pe.Method,
		URL:            pe.URL,
		StatusCode:     pe.StatusCode,
		RequestHeaders: newRequestHeaders,
		ResponseHeaders: newResponseHeaders,
		RequestBody:    pe.RequestBody,
		ResponseBody:   pe.ResponseBody,
		Duration:       pe.Duration,
		Size:           pe.Size,
	}
}