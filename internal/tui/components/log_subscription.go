package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
)

// LogSubscription manages real-time log streaming using Bubble Tea's subscription pattern
type LogSubscription struct {
	logStore *logs.Store
	lastIdx  int
}

// NewLogSubscription creates a subscription that streams new logs as they arrive
func NewLogSubscription(logStore *logs.Store) *LogSubscription {
	return &LogSubscription{
		logStore: logStore,
		lastIdx:  0,
	}
}

// Subscribe returns a Bubble Tea subscription that streams new logs
func (s *LogSubscription) Subscribe() tea.Cmd {
	return tea.Every(100*time.Millisecond, func(t time.Time) tea.Msg {
		if s.logStore == nil {
			return nil
		}

		// Get all logs
		allLogs := s.logStore.GetAll()

		// Check if we have new logs
		if len(allLogs) <= s.lastIdx {
			return nil // No new logs
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
	})
}

// Reset resets the subscription to start from the beginning
func (s *LogSubscription) Reset() {
	s.lastIdx = 0
}

// SetPosition sets the subscription position (useful for filtering)
func (s *LogSubscription) SetPosition(idx int) {
	s.lastIdx = idx
}
