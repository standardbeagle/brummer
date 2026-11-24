package components

import "time"

// Test timing constants for consistent async operations
const (
	// AsyncProcessingDelay is the standard delay for async log processing
	// The log store uses an async worker pattern, so we need to wait
	// for logs to be processed before asserting
	AsyncProcessingDelay = 20 * time.Millisecond

	// DebounceSettleTime is the time to wait for debounced operations to complete
	// The debounced log subscription uses a 50ms window, so we wait slightly longer
	DebounceSettleTime = 60 * time.Millisecond

	// StressTestTimeout is the maximum time for stress test operations
	StressTestTimeout = 30 * time.Second

	// ComponentInitTimeout is the time to wait for component initialization
	ComponentInitTimeout = 100 * time.Millisecond

	// ProcessStartupDelay is the time to wait for a process to start
	ProcessStartupDelay = 50 * time.Millisecond

	// LogFlushTimeout is the maximum time to wait for logs to be flushed
	LogFlushTimeout = 200 * time.Millisecond
)

// Test configuration constants
const (
	// DefaultConcurrentWriters is the default number of concurrent log writers in stress tests
	DefaultConcurrentWriters = 10

	// DefaultLogsPerWriter is the default number of logs each writer generates
	DefaultLogsPerWriter = 50

	// DefaultConcurrentReaders is the default number of concurrent log readers
	DefaultConcurrentReaders = 5

	// MaxLogEntries is the maximum number of log entries in tests
	MaxLogEntries = 10000

	// DefaultTestWidth is the default terminal width for tests
	DefaultTestWidth = 120

	// DefaultTestHeight is the default terminal height for tests
	DefaultTestHeight = 30
)

// WaitForAsync waits for async operations to complete
func WaitForAsync() {
	time.Sleep(AsyncProcessingDelay)
}

// WaitForDebounce waits for debounced operations to settle
func WaitForDebounce() {
	time.Sleep(DebounceSettleTime)
}

// WaitForProcessStartup waits for a process to initialize
func WaitForProcessStartup() {
	time.Sleep(ProcessStartupDelay)
}

// WaitForComponentInit waits for component initialization
func WaitForComponentInit() {
	time.Sleep(ComponentInitTimeout)
}
