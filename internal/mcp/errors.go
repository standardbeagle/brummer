package mcp

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// NetworkError represents a structured network error with classification and retry information
type NetworkError struct {
	Type       ErrorType              `json:"type"`
	Underlying error                  `json:"-"`
	Temporary  bool                   `json:"temporary"`
	RetryAfter time.Duration          `json:"retry_after"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Instance   string                 `json:"instance,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ErrorType represents different categories of network errors
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeConnRefused
	ErrorTypeTimeout
	ErrorTypeDNS
	ErrorTypeNetworkUnreachable
	ErrorTypeConnReset
	ErrorTypeProcessNotFound
	ErrorTypePermissionDenied
	ErrorTypeHostUnreachable
	ErrorTypeNoRoute
	ErrorTypeTLSHandshake
	ErrorTypeProtocol
	ErrorTypeContextCancelled
	ErrorTypeContextDeadline
)

// String returns a human-readable representation of the error type
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeConnRefused:
		return "connection_refused"
	case ErrorTypeTimeout:
		return "timeout"
	case ErrorTypeDNS:
		return "dns_resolution"
	case ErrorTypeNetworkUnreachable:
		return "network_unreachable"
	case ErrorTypeConnReset:
		return "connection_reset"
	case ErrorTypeProcessNotFound:
		return "process_not_found"
	case ErrorTypePermissionDenied:
		return "permission_denied"
	case ErrorTypeHostUnreachable:
		return "host_unreachable"
	case ErrorTypeNoRoute:
		return "no_route"
	case ErrorTypeTLSHandshake:
		return "tls_handshake"
	case ErrorTypeProtocol:
		return "protocol_error"
	case ErrorTypeContextCancelled:
		return "context_cancelled"
	case ErrorTypeContextDeadline:
		return "context_deadline"
	default:
		return "unknown"
	}
}

// Error implements the error interface
func (ne *NetworkError) Error() string {
	if ne.Instance != "" {
		return fmt.Sprintf("network error [%s] on instance %s: %v", ne.Type.String(), ne.Instance, ne.Underlying)
	}
	return fmt.Sprintf("network error [%s]: %v", ne.Type.String(), ne.Underlying)
}

// IsTemporary returns whether the error is temporary and should be retried
func (ne *NetworkError) IsTemporary() bool {
	return ne.Temporary
}

// ShouldRetry returns whether the operation should be retried based on error type and conditions
func (ne *NetworkError) ShouldRetry() bool {
	if !ne.Temporary {
		return false
	}

	// Don't retry certain error types
	switch ne.Type {
	case ErrorTypePermissionDenied, ErrorTypeProcessNotFound, ErrorTypeProtocol:
		return false
	case ErrorTypeContextCancelled, ErrorTypeContextDeadline:
		return false
	default:
		return true
	}
}

// GetRetryDelay returns the recommended delay before retrying
func (ne *NetworkError) GetRetryDelay() time.Duration {
	return ne.RetryAfter
}

// WithContext adds context information to the error
func (ne *NetworkError) WithContext(key string, value interface{}) *NetworkError {
	if ne.Context == nil {
		ne.Context = make(map[string]interface{})
	}
	ne.Context[key] = value
	return ne
}

// WithInstance sets the instance ID for the error
func (ne *NetworkError) WithInstance(instanceID string) *NetworkError {
	ne.Instance = instanceID
	return ne
}

// ClassifyNetworkError analyzes an error and returns a structured NetworkError
func ClassifyNetworkError(err error) *NetworkError {
	if err == nil {
		return nil
	}

	netErr := &NetworkError{
		Type:       ErrorTypeUnknown,
		Underlying: err,
		Temporary:  false,
		Timestamp:  time.Now(),
		Context:    make(map[string]interface{}),
	}

	// Analyze error message for specific patterns first (more specific)
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "context canceled"):
		netErr.Type = ErrorTypeContextCancelled
		netErr.Temporary = false // Don't retry cancelled operations
		return netErr

	case strings.Contains(errStr, "context deadline exceeded"):
		netErr.Type = ErrorTypeContextDeadline
		netErr.Temporary = true
		netErr.RetryAfter = 1 * time.Second
		return netErr
	}

	// Check for standard net.Error interface
	if nerr, ok := err.(net.Error); ok {
		netErr.Context["net_error"] = true

		if nerr.Timeout() {
			netErr.Type = ErrorTypeTimeout
			netErr.Temporary = true
			netErr.RetryAfter = 5 * time.Second
			return netErr
		}

		if nerr.Temporary() {
			netErr.Temporary = true
			netErr.RetryAfter = 2 * time.Second
		}
	}

	// Continue with other pattern matching
	switch {
	case strings.Contains(errStr, "connection refused"):
		netErr.Type = ErrorTypeConnRefused
		netErr.Temporary = true
		netErr.RetryAfter = 10 * time.Second

	case strings.Contains(errStr, "connection reset"):
		netErr.Type = ErrorTypeConnReset
		netErr.Temporary = true
		netErr.RetryAfter = 2 * time.Second

	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "timed out"):
		netErr.Type = ErrorTypeTimeout
		netErr.Temporary = true
		netErr.RetryAfter = 5 * time.Second

	case strings.Contains(errStr, "network is unreachable"):
		netErr.Type = ErrorTypeNetworkUnreachable
		netErr.Temporary = true
		netErr.RetryAfter = 15 * time.Second

	case strings.Contains(errStr, "host unreachable") || strings.Contains(errStr, "no such host"):
		netErr.Type = ErrorTypeHostUnreachable
		netErr.Temporary = true
		netErr.RetryAfter = 30 * time.Second

	case strings.Contains(errStr, "no route to host"):
		netErr.Type = ErrorTypeNoRoute
		netErr.Temporary = true
		netErr.RetryAfter = 30 * time.Second

	case strings.Contains(errStr, "permission denied"):
		netErr.Type = ErrorTypePermissionDenied
		netErr.Temporary = false // Don't retry permission errors

	case strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate"):
		netErr.Type = ErrorTypeTLSHandshake
		netErr.Temporary = false // Don't retry TLS errors

	case strings.Contains(errStr, "dns") || strings.Contains(errStr, "name resolution"):
		netErr.Type = ErrorTypeDNS
		netErr.Temporary = true
		netErr.RetryAfter = 10 * time.Second

	case strings.Contains(errStr, "process") && strings.Contains(errStr, "not found"):
		netErr.Type = ErrorTypeProcessNotFound
		netErr.Temporary = false // Don't retry if process doesn't exist

	default:
		// For unknown errors, be conservative about retrying
		netErr.Type = ErrorTypeUnknown
		netErr.Temporary = true
		netErr.RetryAfter = 30 * time.Second
	}

	return netErr
}

// ClassifyHTTPError analyzes HTTP-related errors and returns structured NetworkError
func ClassifyHTTPError(statusCode int, err error) *NetworkError {
	netErr := ClassifyNetworkError(err)
	if netErr == nil {
		netErr = &NetworkError{
			Type:      ErrorTypeProtocol,
			Timestamp: time.Now(),
			Context:   make(map[string]interface{}),
		}
	}

	netErr.Context["http_status_code"] = statusCode

	// Classify based on HTTP status codes
	switch {
	case statusCode >= 500 && statusCode < 600:
		// Server errors - usually temporary
		netErr.Type = ErrorTypeProtocol
		netErr.Temporary = true
		netErr.RetryAfter = 5 * time.Second
		netErr.Context["http_category"] = "server_error"

	case statusCode == 429:
		// Rate limiting - temporary but longer delay
		netErr.Type = ErrorTypeProtocol
		netErr.Temporary = true
		netErr.RetryAfter = 30 * time.Second
		netErr.Context["http_category"] = "rate_limited"

	case statusCode >= 400 && statusCode < 500:
		// Client errors - usually not retryable
		netErr.Type = ErrorTypeProtocol
		netErr.Temporary = false
		netErr.Context["http_category"] = "client_error"

	case statusCode >= 300 && statusCode < 400:
		// Redirection - might be retryable depending on implementation
		netErr.Type = ErrorTypeProtocol
		netErr.Temporary = true
		netErr.RetryAfter = 1 * time.Second
		netErr.Context["http_category"] = "redirection"
	}

	return netErr
}

// IsRetryableError checks if an error should be retried based on type and context
func IsRetryableError(err error) bool {
	if netErr, ok := err.(*NetworkError); ok {
		return netErr.ShouldRetry()
	}

	// For non-NetworkError types, use basic heuristics
	if nerr, ok := err.(net.Error); ok {
		return nerr.Temporary()
	}

	// Default to not retrying unknown error types
	return false
}

// GetRetryDelay extracts retry delay from structured errors
func GetRetryDelay(err error) time.Duration {
	if netErr, ok := err.(*NetworkError); ok {
		return netErr.GetRetryDelay()
	}

	// Default retry delay for unstructured errors
	return 5 * time.Second
}

// ErrorStats tracks error statistics for monitoring and debugging
type ErrorStats struct {
	TotalErrors    int            `json:"total_errors"`
	ErrorsByType   map[string]int `json:"errors_by_type"`
	ErrorsByCode   map[int]int    `json:"errors_by_code,omitempty"`
	LastError      *NetworkError  `json:"last_error,omitempty"`
	LastErrorTime  time.Time      `json:"last_error_time"`
	TempErrorCount int            `json:"temporary_error_count"`
	PermErrorCount int            `json:"permanent_error_count"`
}

// NewErrorStats creates a new error statistics tracker
func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		ErrorsByType: make(map[string]int),
		ErrorsByCode: make(map[int]int),
	}
}

// RecordError adds an error to the statistics
func (es *ErrorStats) RecordError(err error) {
	es.TotalErrors++
	es.LastErrorTime = time.Now()

	if netErr, ok := err.(*NetworkError); ok {
		es.LastError = netErr

		typeStr := netErr.Type.String()
		es.ErrorsByType[typeStr]++

		if netErr.Temporary {
			es.TempErrorCount++
		} else {
			es.PermErrorCount++
		}

		if httpCode, exists := netErr.Context["http_status_code"]; exists {
			if code, ok := httpCode.(int); ok {
				es.ErrorsByCode[code]++
			}
		}
	} else {
		// For unstructured errors, classify them first
		classified := ClassifyNetworkError(err)
		if classified != nil {
			es.LastError = classified
			typeStr := classified.Type.String()
			es.ErrorsByType[typeStr]++

			if classified.Temporary {
				es.TempErrorCount++
			} else {
				es.PermErrorCount++
			}
		} else {
			es.ErrorsByType["unknown"]++
			es.PermErrorCount++
		}
	}
}

// GetErrorRate returns the percentage of errors that are temporary vs permanent
func (es *ErrorStats) GetErrorRate() (tempRate, permRate float64) {
	if es.TotalErrors == 0 {
		return 0, 0
	}

	tempRate = float64(es.TempErrorCount) / float64(es.TotalErrors) * 100
	permRate = float64(es.PermErrorCount) / float64(es.TotalErrors) * 100
	return tempRate, permRate
}

// Reset clears all error statistics
func (es *ErrorStats) Reset() {
	es.TotalErrors = 0
	es.ErrorsByType = make(map[string]int)
	es.ErrorsByCode = make(map[int]int)
	es.LastError = nil
	es.LastErrorTime = time.Time{}
	es.TempErrorCount = 0
	es.PermErrorCount = 0
}
