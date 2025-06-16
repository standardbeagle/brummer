package logs

import (
	"testing"
	"time"
)

func TestLogCollapsing(t *testing.T) {
	store := NewStore(100)

	// Add some identical consecutive logs
	store.Add("proc1", "test", "This is a test message", false)
	time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	store.Add("proc1", "test", "This is a test message", false)
	time.Sleep(1 * time.Millisecond)
	store.Add("proc1", "test", "This is a test message", false)

	// Add a different log
	store.Add("proc1", "test", "This is a different message", false)

	// Add more identical logs
	store.Add("proc1", "test", "Another repeated message", false)
	time.Sleep(1 * time.Millisecond)
	store.Add("proc1", "test", "Another repeated message", false)

	// Get collapsed logs
	collapsed := store.GetAllCollapsed()

	// Should have 3 collapsed entries
	if len(collapsed) != 3 {
		t.Errorf("Expected 3 collapsed entries, got %d", len(collapsed))
	}

	// First entry should be collapsed with count 3
	if !collapsed[0].IsCollapsed {
		t.Error("First entry should be collapsed")
	}
	if collapsed[0].Count != 3 {
		t.Errorf("First entry should have count 3, got %d", collapsed[0].Count)
	}
	if collapsed[0].Content != "This is a test message" {
		t.Errorf("First entry content incorrect: %s", collapsed[0].Content)
	}

	// Second entry should not be collapsed (single occurrence)
	if collapsed[1].IsCollapsed {
		t.Error("Second entry should not be collapsed")
	}
	if collapsed[1].Count != 1 {
		t.Errorf("Second entry should have count 1, got %d", collapsed[1].Count)
	}
	if collapsed[1].Content != "This is a different message" {
		t.Errorf("Second entry content incorrect: %s", collapsed[1].Content)
	}

	// Third entry should be collapsed with count 2
	if !collapsed[2].IsCollapsed {
		t.Error("Third entry should be collapsed")
	}
	if collapsed[2].Count != 2 {
		t.Errorf("Third entry should have count 2, got %d", collapsed[2].Count)
	}
	if collapsed[2].Content != "Another repeated message" {
		t.Errorf("Third entry content incorrect: %s", collapsed[2].Content)
	}
}

func TestLogCollapsingByProcess(t *testing.T) {
	store := NewStore(100)

	// Add logs from different processes
	store.Add("proc1", "test1", "Message A", false)
	store.Add("proc2", "test2", "Message A", false) // Same content, different process
	store.Add("proc1", "test1", "Message A", false) // Same as first

	// Get collapsed logs for proc1
	collapsed := store.GetByProcessCollapsed("proc1")

	// Should have 1 collapsed entry with count 2
	if len(collapsed) != 1 {
		t.Errorf("Expected 1 collapsed entry for proc1, got %d", len(collapsed))
	}

	if !collapsed[0].IsCollapsed {
		t.Error("Entry should be collapsed")
	}
	if collapsed[0].Count != 2 {
		t.Errorf("Entry should have count 2, got %d", collapsed[0].Count)
	}

	// Get collapsed logs for proc2
	collapsed2 := store.GetByProcessCollapsed("proc2")

	// Should have 1 non-collapsed entry
	if len(collapsed2) != 1 {
		t.Errorf("Expected 1 entry for proc2, got %d", len(collapsed2))
	}

	if collapsed2[0].IsCollapsed {
		t.Error("Entry should not be collapsed")
	}
	if collapsed2[0].Count != 1 {
		t.Errorf("Entry should have count 1, got %d", collapsed2[0].Count)
	}
}

func TestLogCollapsingWithDifferentLevels(t *testing.T) {
	store := NewStore(100)

	// Add identical content but one as error
	store.Add("proc1", "test", "Test message", false)
	store.Add("proc1", "test", "Test message", true) // Same content but error
	store.Add("proc1", "test", "Test message", false)

	// Get collapsed logs
	collapsed := store.GetAllCollapsed()

	// Should have 3 collapsed entries (error and non-error are separate, so we get: non-error, error, non-error)
	if len(collapsed) != 3 {
		t.Errorf("Expected 3 collapsed entries, got %d", len(collapsed))
		for i, entry := range collapsed {
			t.Logf("Entry %d: Content='%s', IsError=%v, Count=%d", i, entry.Content, entry.IsError, entry.Count)
		}
		return
	}

	// First should be single non-error
	if collapsed[0].IsCollapsed {
		t.Error("First entry should not be collapsed")
	}
	if collapsed[0].IsError {
		t.Error("First entry should not be error")
	}

	// Second should be single error
	if collapsed[1].IsCollapsed {
		t.Error("Second entry should not be collapsed")
	}
	if !collapsed[1].IsError {
		t.Error("Second entry should be error")
	}

	// Third should be single non-error (different from first because interrupted by error)
	if collapsed[2].IsCollapsed {
		t.Error("Third entry should not be collapsed")
	}
	if collapsed[2].IsError {
		t.Error("Third entry should not be error")
	}
}

func TestLogCollapsingTimestamps(t *testing.T) {
	store := NewStore(100)

	// Add logs with measurable time differences
	start := time.Now()
	store.Add("proc1", "test", "Repeated message", false)

	time.Sleep(10 * time.Millisecond)
	middle := time.Now()
	store.Add("proc1", "test", "Repeated message", false)

	time.Sleep(10 * time.Millisecond)
	end := time.Now()
	store.Add("proc1", "test", "Repeated message", false)

	// Get collapsed logs
	collapsed := store.GetAllCollapsed()

	if len(collapsed) != 1 {
		t.Errorf("Expected 1 collapsed entry, got %d", len(collapsed))
		return
	}

	entry := collapsed[0]

	// Check timestamps
	if entry.FirstSeen.Before(start) || entry.FirstSeen.After(middle) {
		t.Error("FirstSeen timestamp should be close to start time")
	}

	if entry.LastSeen.Before(middle) || entry.LastSeen.After(end.Add(time.Millisecond)) {
		t.Error("LastSeen timestamp should be close to end time")
	}

	if !entry.FirstSeen.Before(entry.LastSeen) {
		t.Error("FirstSeen should be before LastSeen")
	}
}

func TestAreLogsIdentical(t *testing.T) {
	store := NewStore(100)

	now := time.Now()

	log1 := LogEntry{
		ID:          "1",
		ProcessID:   "proc1",
		ProcessName: "test",
		Timestamp:   now,
		Content:     "Test message",
		Level:       LevelInfo,
		IsError:     false,
	}

	log2 := LogEntry{
		ID:          "2", // Different ID
		ProcessID:   "proc1",
		ProcessName: "test",
		Timestamp:   now.Add(time.Second), // Different timestamp
		Content:     "Test message",
		Level:       LevelInfo,
		IsError:     false,
	}

	log3 := LogEntry{
		ID:          "3",
		ProcessID:   "proc2", // Different process
		ProcessName: "test",
		Timestamp:   now,
		Content:     "Test message",
		Level:       LevelInfo,
		IsError:     false,
	}

	log4 := LogEntry{
		ID:          "4",
		ProcessID:   "proc1",
		ProcessName: "test",
		Timestamp:   now,
		Content:     "Different message", // Different content
		Level:       LevelInfo,
		IsError:     false,
	}

	// log1 and log2 should be identical (ignore ID and timestamp)
	if !store.areLogsIdentical(log1, log2) {
		t.Error("log1 and log2 should be identical")
	}

	// log1 and log3 should not be identical (different process)
	if store.areLogsIdentical(log1, log3) {
		t.Error("log1 and log3 should not be identical")
	}

	// log1 and log4 should not be identical (different content)
	if store.areLogsIdentical(log1, log4) {
		t.Error("log1 and log4 should not be identical")
	}
}
