package components

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
)

// DebouncedLogSubscription manages real-time log streaming with debouncing
// This replaces polling with an event-driven approach that batches rapid updates
type DebouncedLogSubscription struct {
	logStore *logs.Store
	lastIdx  int

	// Debouncing mechanism
	mu            sync.Mutex
	pendingUpdate bool
	debounceTimer *time.Timer
	debounceDelay time.Duration

	// Channel for triggering updates
	updateChan chan struct{}
	stopChan   chan struct{}
}

// NewDebouncedLogSubscription creates an event-driven subscription with debouncing
func NewDebouncedLogSubscription(logStore *logs.Store) *DebouncedLogSubscription {
	return &DebouncedLogSubscription{
		logStore:      logStore,
		lastIdx:       0,
		debounceDelay: 50 * time.Millisecond,  // Batch updates within 50ms window
		updateChan:    make(chan struct{}, 1), // Buffered to avoid blocking
		stopChan:      make(chan struct{}),
	}
}

// NotifyNewLogs triggers an update (will be debounced)
func (s *DebouncedLogSubscription) NotifyNewLogs() {
	// Non-blocking send - if channel is full, update is already pending
	select {
	case s.updateChan <- struct{}{}:
	default:
	}
}

// Subscribe returns a Bubble Tea command that listens for log updates
func (s *DebouncedLogSubscription) Subscribe() tea.Cmd {
	return func() tea.Msg {
		// Add a timeout fallback to check for new logs periodically
		// This ensures we don't miss logs from components that don't call NotifyNewLogs()
		fallbackTimer := time.NewTimer(1 * time.Second)
		defer fallbackTimer.Stop()

		// Listen for update notifications
		select {
		case <-s.updateChan:
			// Debounce: wait a bit to batch multiple rapid updates
			s.mu.Lock()
			if s.debounceTimer != nil {
				s.debounceTimer.Stop()
			}
			s.debounceTimer = time.NewTimer(s.debounceDelay)
			s.mu.Unlock()

			// Wait for debounce timer or more updates
			for {
				select {
				case <-s.debounceTimer.C:
					// Timer expired, send the update
					return s.createUpdateMsg()

				case <-s.updateChan:
					// More updates arrived, reset timer
					s.mu.Lock()
					if !s.debounceTimer.Stop() {
						// Timer already fired, need to drain channel
						select {
						case <-s.debounceTimer.C:
						default:
						}
					}
					s.debounceTimer.Reset(s.debounceDelay)
					s.mu.Unlock()

				case <-s.stopChan:
					return nil
				}
			}

		case <-fallbackTimer.C:
			// Timeout fallback - check for new logs even without notification
			// This catches logs from components that don't call NotifyNewLogs()
			return s.createUpdateMsg()

		case <-s.stopChan:
			return nil
		}
	}
}

// createUpdateMsg creates a LogsUpdateMsg with new logs since last check
func (s *DebouncedLogSubscription) createUpdateMsg() tea.Msg {
	if s.logStore == nil {
		// Re-subscribe to keep listening
		return ContinueLogSubscriptionMsg{}
	}

	// Get all logs
	allLogs := s.logStore.GetAll()

	// Check if we have new logs
	if len(allLogs) <= s.lastIdx {
		// No new logs, but keep listening
		return ContinueLogSubscriptionMsg{}
	}

	// Get only new logs since last check
	newLogs := allLogs[s.lastIdx:]

	// Convert to pointers for the view
	entries := make([]*logs.LogEntry, len(newLogs))
	for i := range newLogs {
		entry := newLogs[i] // Copy to avoid reference issues
		entries[i] = &entry
	}

	// Update our position
	s.lastIdx = len(allLogs)

	// Return message with only NEW logs to append
	return LogsUpdateMsg{
		Entries: entries,
		Append:  true, // Append instead of replace
	}
}

// Reset resets the subscription to start from the beginning
func (s *DebouncedLogSubscription) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastIdx = 0
}

// SetPosition sets the subscription position (useful for filtering)
func (s *DebouncedLogSubscription) SetPosition(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastIdx = idx
}

// Stop cleanly shuts down the subscription
func (s *DebouncedLogSubscription) Stop() {
	close(s.stopChan)
	s.mu.Lock()
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}
	s.mu.Unlock()
}

// ContinueLogSubscriptionMsg signals to re-subscribe for more updates
type ContinueLogSubscriptionMsg struct{}
