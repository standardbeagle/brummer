package messages

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

func TestProcessStartedMsg(t *testing.T) {
	now := time.Now()
	msg := ProcessStartedMsg{
		ProcessID: "test-123",
		Name:      "dev",
		Script:    "npm run dev",
		StartTime: now,
	}

	if msg.ProcessID != "test-123" {
		t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
	}
	if msg.Name != "dev" {
		t.Errorf("Expected Name dev, got %s", msg.Name)
	}
	if msg.Script != "npm run dev" {
		t.Errorf("Expected Script 'npm run dev', got %s", msg.Script)
	}
	if msg.StartTime != now {
		t.Errorf("Expected StartTime %v, got %v", now, msg.StartTime)
	}
}

func TestProcessStoppedMsg(t *testing.T) {
	now := time.Now()
	msg := ProcessStoppedMsg{
		ProcessID: "test-123",
		Name:      "dev",
		ExitCode:  0,
		EndTime:   now,
	}

	if msg.ProcessID != "test-123" {
		t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
	}
	if msg.ExitCode != 0 {
		t.Errorf("Expected ExitCode 0, got %d", msg.ExitCode)
	}
}

func TestProcessOutputMsg(t *testing.T) {
	now := time.Now()
	msg := ProcessOutputMsg{
		ProcessID: "test-123",
		Line:      "Hello World",
		IsError:   false,
		Timestamp: now,
	}

	if msg.ProcessID != "test-123" {
		t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
	}
	if msg.Line != "Hello World" {
		t.Errorf("Expected Line 'Hello World', got %s", msg.Line)
	}
	if msg.IsError {
		t.Error("Expected IsError false, got true")
	}
}

func TestProcessStateChangedMsg(t *testing.T) {
	msg := ProcessStateChangedMsg{
		ProcessID: "test-123",
		OldState:  process.StatusPending,
		NewState:  process.StatusRunning,
	}

	if msg.ProcessID != "test-123" {
		t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
	}
	if msg.OldState != process.StatusPending {
		t.Errorf("Expected OldState StatusPending, got %v", msg.OldState)
	}
	if msg.NewState != process.StatusRunning {
		t.Errorf("Expected NewState StatusRunning, got %v", msg.NewState)
	}
}

func TestLogAddedMsg(t *testing.T) {
	entry := &logs.LogEntry{
		ID:          "entry-1",
		ProcessID:   "test-123",
		ProcessName: "dev",
		Timestamp:   time.Now(),
		Content:     "Test log",
		IsError:     false,
	}

	msg := LogAddedMsg{
		Entry: entry,
	}

	if msg.Entry.ID != "entry-1" {
		t.Errorf("Expected Entry.ID entry-1, got %s", msg.Entry.ID)
	}
	if msg.Entry.Content != "Test log" {
		t.Errorf("Expected Entry.Content 'Test log', got %s", msg.Entry.Content)
	}
}

func TestLogsClearedMsg(t *testing.T) {
	// Test clearing specific process
	msg := LogsClearedMsg{
		ProcessID: "test-123",
	}
	if msg.ProcessID != "test-123" {
		t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
	}

	// Test clearing all logs
	msgAll := LogsClearedMsg{
		ProcessID: "",
	}
	if msgAll.ProcessID != "" {
		t.Errorf("Expected empty ProcessID, got %s", msgAll.ProcessID)
	}
}

func TestURLDetectedMsg(t *testing.T) {
	now := time.Now()
	msg := URLDetectedMsg{
		URL:        "http://localhost:3000",
		ProcessID:  "test-123",
		Type:       "http",
		Timestamp:  now,
		HasProxy:   true,
		ProxyPort:  8080,
		ProxyURL:   "http://localhost:8080",
		ProxyLabel: "Dev Server",
	}

	if msg.URL != "http://localhost:3000" {
		t.Errorf("Expected URL http://localhost:3000, got %s", msg.URL)
	}
	if msg.Type != "http" {
		t.Errorf("Expected Type http, got %s", msg.Type)
	}
	if !msg.HasProxy {
		t.Error("Expected HasProxy true, got false")
	}
	if msg.ProxyPort != 8080 {
		t.Errorf("Expected ProxyPort 8080, got %d", msg.ProxyPort)
	}
}

func TestErrorOccurredMsg(t *testing.T) {
	now := time.Now()
	msg := ErrorOccurredMsg{
		ProcessID:   "test-123",
		ProcessName: "dev",
		Error:       "Connection failed",
		Timestamp:   now,
	}

	if msg.ProcessID != "test-123" {
		t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
	}
	if msg.Error != "Connection failed" {
		t.Errorf("Expected Error 'Connection failed', got %s", msg.Error)
	}
}

func TestServiceHealthMsg(t *testing.T) {
	msg := ServiceHealthMsg{
		Service: "process",
		Healthy: true,
		Message: "All systems operational",
	}

	if msg.Service != "process" {
		t.Errorf("Expected Service process, got %s", msg.Service)
	}
	if !msg.Healthy {
		t.Error("Expected Healthy true, got false")
	}
	if msg.Message != "All systems operational" {
		t.Errorf("Expected Message 'All systems operational', got %s", msg.Message)
	}

	// Test unhealthy service
	unhealthy := ServiceHealthMsg{
		Service: "proxy",
		Healthy: false,
		Message: "Connection timeout",
	}

	if unhealthy.Healthy {
		t.Error("Expected Healthy false, got true")
	}
}
