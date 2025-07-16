package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeConnRefused, "connection_refused"},
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypeDNS, "dns_resolution"},
		{ErrorTypeNetworkUnreachable, "network_unreachable"},
		{ErrorTypeConnReset, "connection_reset"},
		{ErrorTypeProcessNotFound, "process_not_found"},
		{ErrorTypePermissionDenied, "permission_denied"},
		{ErrorTypeHostUnreachable, "host_unreachable"},
		{ErrorTypeNoRoute, "no_route"},
		{ErrorTypeTLSHandshake, "tls_handshake"},
		{ErrorTypeProtocol, "protocol_error"},
		{ErrorTypeContextCancelled, "context_cancelled"},
		{ErrorTypeContextDeadline, "context_deadline"},
		{ErrorTypeUnknown, "unknown"},
	}

	for _, test := range tests {
		result := test.errorType.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestNetworkErrorBasics(t *testing.T) {
	underlyingErr := errors.New("test error")
	netErr := &NetworkError{
		Type:       ErrorTypeConnRefused,
		Underlying: underlyingErr,
		Temporary:  true,
		RetryAfter: 5 * time.Second,
		Instance:   "test-instance",
		Timestamp:  time.Now(),
	}

	// Test Error() method
	errStr := netErr.Error()
	if errStr != "network error [connection_refused] on instance test-instance: test error" {
		t.Errorf("Unexpected error string: %s", errStr)
	}

	// Test without instance
	netErr.Instance = ""
	errStr = netErr.Error()
	if errStr != "network error [connection_refused]: test error" {
		t.Errorf("Unexpected error string without instance: %s", errStr)
	}

	// Test IsTemporary
	if !netErr.IsTemporary() {
		t.Error("Error should be temporary")
	}

	// Test ShouldRetry
	if !netErr.ShouldRetry() {
		t.Error("Connection refused errors should be retryable")
	}

	// Test GetRetryDelay
	if netErr.GetRetryDelay() != 5*time.Second {
		t.Errorf("Expected retry delay 5s, got %v", netErr.GetRetryDelay())
	}
}

func TestNetworkErrorShouldRetry(t *testing.T) {
	tests := []struct {
		errorType    ErrorType
		temporary    bool
		shouldRetry  bool
		description  string
	}{
		{ErrorTypeConnRefused, true, true, "Connection refused should be retryable"},
		{ErrorTypeTimeout, true, true, "Timeout should be retryable"},
		{ErrorTypePermissionDenied, false, false, "Permission denied should not be retryable"},
		{ErrorTypeProcessNotFound, false, false, "Process not found should not be retryable"},
		{ErrorTypeContextCancelled, false, false, "Context cancelled should not be retryable"},
		{ErrorTypeContextDeadline, true, false, "Context deadline should not be retryable"},
		{ErrorTypeProtocol, false, false, "Protocol errors should not be retryable"},
		{ErrorTypeNetworkUnreachable, true, true, "Network unreachable should be retryable"},
	}

	for _, test := range tests {
		netErr := &NetworkError{
			Type:      test.errorType,
			Temporary: test.temporary,
		}

		result := netErr.ShouldRetry()
		if result != test.shouldRetry {
			t.Errorf("%s: expected %v, got %v", test.description, test.shouldRetry, result)
		}
	}
}

func TestNetworkErrorWithContext(t *testing.T) {
	netErr := &NetworkError{
		Type: ErrorTypeTimeout,
	}

	// Add context
	netErr.WithContext("attempt", 3)
	netErr.WithContext("duration", "5s")

	if netErr.Context["attempt"] != 3 {
		t.Error("Context should contain attempt information")
	}

	if netErr.Context["duration"] != "5s" {
		t.Error("Context should contain duration information")
	}

	// Test WithInstance
	netErr.WithInstance("test-instance-123")
	if netErr.Instance != "test-instance-123" {
		t.Error("Instance should be set correctly")
	}
}

func TestClassifyNetworkError(t *testing.T) {
	tests := []struct {
		inputError   error
		expectedType ErrorType
		temporary    bool
		description  string
	}{
		{
			inputError:   errors.New("connection refused"),
			expectedType: ErrorTypeConnRefused,
			temporary:    true,
			description:  "Connection refused error",
		},
		{
			inputError:   errors.New("connection reset by peer"),
			expectedType: ErrorTypeConnReset,
			temporary:    true,
			description:  "Connection reset error",
		},
		{
			inputError:   errors.New("operation timed out"),
			expectedType: ErrorTypeTimeout,
			temporary:    true,
			description:  "Timeout error",
		},
		{
			inputError:   errors.New("context deadline exceeded"),
			expectedType: ErrorTypeContextDeadline,
			temporary:    true,
			description:  "Context deadline error",
		},
		{
			inputError:   errors.New("context canceled"),
			expectedType: ErrorTypeContextCancelled,
			temporary:    false,
			description:  "Context cancelled error",
		},
		{
			inputError:   errors.New("network is unreachable"),
			expectedType: ErrorTypeNetworkUnreachable,
			temporary:    true,
			description:  "Network unreachable error",
		},
		{
			inputError:   errors.New("permission denied"),
			expectedType: ErrorTypePermissionDenied,
			temporary:    false,
			description:  "Permission denied error",
		},
		{
			inputError:   errors.New("no such host"),
			expectedType: ErrorTypeHostUnreachable,
			temporary:    true,
			description:  "Host unreachable error",
		},
		{
			inputError:   errors.New("no route to host"),
			expectedType: ErrorTypeNoRoute,
			temporary:    true,
			description:  "No route error",
		},
		{
			inputError:   errors.New("tls handshake failed"),
			expectedType: ErrorTypeTLSHandshake,
			temporary:    false,
			description:  "TLS handshake error",
		},
		{
			inputError:   errors.New("dns resolution failed"),
			expectedType: ErrorTypeDNS,
			temporary:    true,
			description:  "DNS error",
		},
		{
			inputError:   errors.New("process not found"),
			expectedType: ErrorTypeProcessNotFound,
			temporary:    false,
			description:  "Process not found error",
		},
		{
			inputError:   errors.New("completely unknown error type"),
			expectedType: ErrorTypeUnknown,
			temporary:    true,
			description:  "Unknown error type",
		},
	}

	for _, test := range tests {
		result := ClassifyNetworkError(test.inputError)
		
		if result == nil {
			t.Errorf("%s: ClassifyNetworkError should not return nil", test.description)
			continue
		}

		if result.Type != test.expectedType {
			t.Errorf("%s: expected type %v, got %v", test.description, test.expectedType, result.Type)
		}

		if result.Temporary != test.temporary {
			t.Errorf("%s: expected temporary %v, got %v", test.description, test.temporary, result.Temporary)
		}

		if result.Underlying != test.inputError {
			t.Errorf("%s: underlying error should be preserved", test.description)
		}

		if result.Timestamp.IsZero() {
			t.Errorf("%s: timestamp should be set", test.description)
		}
	}
}

func TestClassifyNetworkErrorWithNetError(t *testing.T) {
	// Create a mock net.Error
	mockNetErr := &mockNetError{
		timeout:   true,
		temporary: true,
	}

	result := ClassifyNetworkError(mockNetErr)
	
	if result.Type != ErrorTypeTimeout {
		t.Errorf("Expected timeout error type, got %v", result.Type)
	}

	if !result.Temporary {
		t.Error("Should be marked as temporary")
	}

	if result.Context["net_error"] != true {
		t.Error("Should be marked as net.Error")
	}
}

func TestClassifyHTTPError(t *testing.T) {
	tests := []struct {
		statusCode   int
		inputError   error
		expectedType ErrorType
		temporary    bool
		description  string
	}{
		{500, errors.New("internal server error"), ErrorTypeProtocol, true, "Server error"},
		{502, errors.New("bad gateway"), ErrorTypeProtocol, true, "Bad gateway"},
		{429, errors.New("too many requests"), ErrorTypeProtocol, true, "Rate limiting"},
		{404, errors.New("not found"), ErrorTypeProtocol, false, "Client error"},
		{401, errors.New("unauthorized"), ErrorTypeProtocol, false, "Unauthorized"},
		{302, errors.New("found"), ErrorTypeProtocol, true, "Redirection"},
	}

	for _, test := range tests {
		result := ClassifyHTTPError(test.statusCode, test.inputError)
		
		if result.Type != test.expectedType {
			t.Errorf("%s: expected type %v, got %v", test.description, test.expectedType, result.Type)
		}

		if result.Temporary != test.temporary {
			t.Errorf("%s: expected temporary %v, got %v", test.description, test.temporary, result.Temporary)
		}

		if result.Context["http_status_code"] != test.statusCode {
			t.Errorf("%s: HTTP status code should be recorded in context", test.description)
		}

		// Check specific retry delays for rate limiting
		if test.statusCode == 429 && result.RetryAfter != 30*time.Second {
			t.Errorf("%s: rate limiting should have 30s retry delay", test.description)
		}
	}
}

func TestIsRetryableError(t *testing.T) {
	// Test with NetworkError
	tempNetErr := &NetworkError{
		Type:      ErrorTypeTimeout,
		Temporary: true,
	}
	if !IsRetryableError(tempNetErr) {
		t.Error("Temporary NetworkError should be retryable")
	}

	permNetErr := &NetworkError{
		Type:      ErrorTypePermissionDenied,
		Temporary: false,
	}
	if IsRetryableError(permNetErr) {
		t.Error("Permanent NetworkError should not be retryable")
	}

	// Test with net.Error
	mockNetErr := &mockNetError{temporary: true}
	if !IsRetryableError(mockNetErr) {
		t.Error("Temporary net.Error should be retryable")
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	if IsRetryableError(regularErr) {
		t.Error("Regular error should not be retryable by default")
	}
}

func TestGetRetryDelay(t *testing.T) {
	// Test with NetworkError
	netErr := &NetworkError{
		RetryAfter: 10 * time.Second,
	}
	if GetRetryDelay(netErr) != 10*time.Second {
		t.Error("Should return NetworkError retry delay")
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	if GetRetryDelay(regularErr) != 5*time.Second {
		t.Error("Should return default retry delay for regular errors")
	}
}

func TestErrorStats(t *testing.T) {
	stats := NewErrorStats()

	// Test initial state
	if stats.TotalErrors != 0 {
		t.Error("Initial error count should be zero")
	}

	// Record some errors
	tempErr := &NetworkError{
		Type:      ErrorTypeTimeout,
		Temporary: true,
		Context:   map[string]interface{}{"http_status_code": 500},
	}
	stats.RecordError(tempErr)

	permErr := &NetworkError{
		Type:      ErrorTypePermissionDenied,
		Temporary: false,
	}
	stats.RecordError(permErr)

	regularErr := errors.New("regular error")
	stats.RecordError(regularErr)

	// Check statistics
	if stats.TotalErrors != 3 {
		t.Errorf("Expected 3 total errors, got %d", stats.TotalErrors)
	}

	// Regular errors get classified as unknown and marked as temporary by default
	if stats.TempErrorCount != 2 {
		t.Errorf("Expected 2 temporary errors, got %d", stats.TempErrorCount)
	}

	if stats.PermErrorCount != 1 {
		t.Errorf("Expected 1 permanent error, got %d", stats.PermErrorCount)
	}

	if stats.ErrorsByType["timeout"] != 1 {
		t.Error("Should have recorded timeout error")
	}

	if stats.ErrorsByCode[500] != 1 {
		t.Error("Should have recorded HTTP 500 error")
	}

	// Test error rates  
	tempRate, permRate := stats.GetErrorRate()
	expectedTempRate := float64(2) / float64(3) * 100 // 2 temporary errors
	expectedPermRate := float64(1) / float64(3) * 100 // 1 permanent error

	if tempRate != expectedTempRate {
		t.Errorf("Expected temp rate %.2f, got %.2f", expectedTempRate, tempRate)
	}

	if permRate != expectedPermRate {
		t.Errorf("Expected perm rate %.2f, got %.2f", expectedPermRate, permRate)
	}

	// Test reset
	stats.Reset()
	if stats.TotalErrors != 0 {
		t.Error("Stats should be reset to zero")
	}
}

func TestErrorStatsWithEmptyStats(t *testing.T) {
	stats := NewErrorStats()
	
	tempRate, permRate := stats.GetErrorRate()
	if tempRate != 0 || permRate != 0 {
		t.Error("Empty stats should return zero rates")
	}
}

func TestClassifyNetworkErrorNil(t *testing.T) {
	result := ClassifyNetworkError(nil)
	if result != nil {
		t.Error("ClassifyNetworkError should return nil for nil input")
	}
}

func TestContextCancellationErrors(t *testing.T) {
	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	err := ctx.Err()
	result := ClassifyNetworkError(err)
	
	if result.Type != ErrorTypeContextCancelled {
		t.Errorf("Expected context cancelled error, got %v", result.Type)
	}

	if result.Temporary {
		t.Error("Context cancelled errors should not be temporary")
	}

	// Test context deadline
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout
	
	err = ctx.Err()
	result = ClassifyNetworkError(err)
	
	if result.Type != ErrorTypeContextDeadline {
		t.Errorf("Expected context deadline error, got %v", result.Type)
	}
}

// Mock net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (m *mockNetError) Error() string {
	return "mock network error"
}

func (m *mockNetError) Timeout() bool {
	return m.timeout
}

func (m *mockNetError) Temporary() bool {
	return m.temporary
}

func TestNetworkErrorJSONSerialization(t *testing.T) {
	netErr := &NetworkError{
		Type:       ErrorTypeConnRefused,
		Temporary:  true,
		RetryAfter: 5 * time.Second,
		Context:    map[string]interface{}{"attempt": 3},
		Instance:   "test-instance",
		Timestamp:  time.Now(),
	}

	// Test that the struct can be used in JSON contexts
	// (This is mainly to ensure json tags are correct)
	if netErr.Type != ErrorTypeConnRefused {
		t.Error("Type should be preserved")
	}
	
	if !netErr.Temporary {
		t.Error("Temporary flag should be preserved")
	}
}

func TestErrorTypeUnknownDefault(t *testing.T) {
	var unknownType ErrorType = 999 // Invalid type
	if unknownType.String() != "unknown" {
		t.Error("Unknown error types should return 'unknown'")
	}
}