package handlers

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/commands"
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressHandler manages progress tracking for long-running operations
type ProgressHandler struct {
	activeProgress   map[string]*Progress
	completedProgress []*Progress
	mu               sync.RWMutex
	maxHistory       int
	updateInterval   time.Duration
	lastUpdate       atomic.Value // time.Time
	ticker           *time.Ticker
	stopChan         chan struct{}
}

// Progress represents the progress of an operation
type Progress struct {
	ID              string                 `json:"id"`
	CommandType     commands.CommandType   `json:"commandType"`
	CommandID       string                 `json:"commandId"`
	Current         int                    `json:"current"`
	Total           int                    `json:"total"`
	Percentage      float64                `json:"percentage"`
	Message         string                 `json:"message"`
	StartTime       time.Time              `json:"startTime"`
	LastUpdate      time.Time              `json:"lastUpdate"`
	EstimatedEnd    time.Time              `json:"estimatedEnd,omitempty"`
	Status          ProgressStatus         `json:"status"`
	SubProgress     map[string]*SubProgress `json:"subProgress,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SubProgress represents sub-progress within a main progress
type SubProgress struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Current    int       `json:"current"`
	Total      int       `json:"total"`
	Percentage float64   `json:"percentage"`
	Status      ProgressStatus `json:"status"`
}

// ProgressStatus represents the status of a progress operation
type ProgressStatus int

const (
	ProgressPending ProgressStatus = iota
	ProgressRunning
	ProgressPaused
	ProgressCompleted
	ProgressFailed
	ProgressCancelled
)

// ProgressUpdate represents a progress update
type ProgressUpdate struct {
	ProgressID   string                 `json:"progressId"`
	Current      int                    `json:"current"`
	Total        int                    `json:"total"`
	Percentage   float64                `json:"percentage"`
	Message      string                 `json:"message"`
	EstimatedEnd time.Time              `json:"estimatedEnd,omitempty"`
	SubProgress  map[string]*SubProgress `json:"subProgress,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ProgressStats represents statistics about progress operations
type ProgressStats struct {
	TotalOperations    int                    `json:"totalOperations"`
	ActiveOperations   int                    `json:"activeOperations"`
	CompletedOperations int                   `json:"completedOperations"`
	AverageDuration    time.Duration          `json:"averageDuration"`
	FastestOperation   time.Duration          `json:"fastestOperation"`
	SlowestOperation   time.Duration          `json:"slowestOperation"`
	OperationsByType   map[commands.CommandType]int `json:"operationsByType"`
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(maxHistory int, updateInterval time.Duration) *ProgressHandler {
	handler := &ProgressHandler{
		activeProgress:   make(map[string]*Progress),
		completedProgress: make([]*Progress, 0, maxHistory),
		maxHistory:       maxHistory,
		updateInterval:   updateInterval,
		stopChan:         make(chan struct{}),
	}
	handler.lastUpdate.Store(time.Time{})

	// Start the background ticker
	handler.startTicker()

	return handler
}

// startTicker initializes ticker but does not start automatic goroutine
// Timeout checking must be called manually to maintain Bubble Tea MVU pattern
func (h *ProgressHandler) startTicker() {
	h.ticker = time.NewTicker(h.updateInterval)
	// Ticker is available but timeout checking must be called manually
}

// Stop stops the progress handler
func (h *ProgressHandler) Stop() {
	close(h.stopChan)
}

// StartProgress starts tracking progress for a command
func (h *ProgressHandler) StartProgress(commandID string, commandType commands.CommandType, total int, message string) *Progress {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress := &Progress{
		ID:          fmt.Sprintf("progress-%s", commandID),
		CommandType: commandType,
		CommandID:   commandID,
		Current:     0,
		Total:       total,
		Percentage:  0.0,
		Message:     message,
		StartTime:   time.Now(),
		LastUpdate:  time.Now(),
		Status:      ProgressRunning,
		SubProgress: make(map[string]*SubProgress),
		Metadata:    make(map[string]interface{}),
	}

	if total > 0 {
		progress.EstimatedEnd = progress.StartTime.Add(h.estimateDuration(progress))
	}

	h.activeProgress[progress.ID] = progress
	h.lastUpdate.Store(time.Now())

	return progress
}

// UpdateProgress updates the progress of an operation
func (h *ProgressHandler) UpdateProgress(progressID string, current int, total int, message string) tea.Cmd {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return nil
	}

	// Update progress values
	progress.Current = current
	progress.Total = total
	progress.Message = message
	progress.LastUpdate = time.Now()

	// Calculate percentage
	if total > 0 {
		progress.Percentage = float64(current) / float64(total) * 100
	} else {
		progress.Percentage = 0
	}

	// Update estimated end time
	if total > 0 && current > 0 {
		elapsed := time.Since(progress.StartTime)
		estimatedTotal := time.Duration(float64(elapsed) * float64(total) / float64(current))
		progress.EstimatedEnd = progress.StartTime.Add(estimatedTotal)
	}

	// Check if completed
	if current >= total && total > 0 {
		progress.Status = ProgressCompleted
		return h.completeProgress(progress)
	}

	h.lastUpdate.Store(time.Now())

	// Return progress update command
	return func() tea.Msg {
		// TODO: Fix type mismatch in messages package
		// return messages.NewCommandProgressMessage(
		//	progress.CommandID,
		//	progress.CommandType,
		//	int(progress.Percentage),
		//	progress.Message,
		//	commands.SystemOperationContext(0), // Version will be set by state handler
		// )
		return tea.Msg(nil)
	}
}

// AddSubProgress adds a sub-progress to a main progress
func (h *ProgressHandler) AddSubProgress(progressID, subID, name string, total int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return
	}

	subProgress := &SubProgress{
		ID:         subID,
		Name:       name,
		Current:    0,
		Total:      total,
		Percentage: 0,
		Status:     ProgressRunning,
	}

	progress.SubProgress[subID] = subProgress
	h.lastUpdate.Store(time.Now())
}

// UpdateSubProgress updates a sub-progress
func (h *ProgressHandler) UpdateSubProgress(progressID, subID string, current int, total int) tea.Cmd {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return nil
	}

	subProgress, exists := progress.SubProgress[subID]
	if !exists {
		return nil
	}

	subProgress.Current = current
	subProgress.Total = total

	if total > 0 {
		subProgress.Percentage = float64(current) / float64(total) * 100
		if current >= total {
			subProgress.Status = ProgressCompleted
		}
	}

	// Update main progress based on sub-progress
	h.calculateMainProgress(progress)
	h.lastUpdate.Store(time.Now())

	return func() tea.Msg {
		// TODO: Fix type mismatch in messages package
		// return messages.NewCommandProgressMessage(
		//	progress.CommandID,
		//	progress.CommandType,
		//	int(progress.Percentage),
		//	progress.Message,
		//	commands.SystemOperationContext(0),
		// )
		return tea.Msg(nil)
	}
}

// calculateMainProgress calculates the main progress based on sub-progress
func (h *ProgressHandler) calculateMainProgress(progress *Progress) {
	if len(progress.SubProgress) == 0 {
		return
	}

	totalPercentage := 0.0
	for _, sub := range progress.SubProgress {
		totalPercentage += sub.Percentage
	}

	progress.Percentage = totalPercentage / float64(len(progress.SubProgress))
	progress.Current = int(float64(progress.Total) * progress.Percentage / 100)
}

// SetProgressMetadata sets metadata for a progress operation
func (h *ProgressHandler) SetProgressMetadata(progressID string, key string, value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return
	}

	progress.Metadata[key] = value
	h.lastUpdate.Store(time.Now())
}

// PauseProgress pauses a progress operation
func (h *ProgressHandler) PauseProgress(progressID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return false
	}

	progress.Status = ProgressPaused
	return true
}

// ResumeProgress resumes a paused progress operation
func (h *ProgressHandler) ResumeProgress(progressID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists || progress.Status != ProgressPaused {
		return false
	}

	progress.Status = ProgressRunning
	return true
}

// CancelProgress cancels a progress operation
func (h *ProgressHandler) CancelProgress(progressID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return false
	}

	progress.Status = ProgressCancelled
	return true
}

// FailProgress marks a progress operation as failed
func (h *ProgressHandler) FailProgress(progressID string, reason string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress, exists := h.activeProgress[progressID]
	if !exists {
		return false
	}

	progress.Status = ProgressFailed
	progress.Message = reason
	return true
}

// completeProgress moves a progress operation to completed history
func (h *ProgressHandler) completeProgress(progress *Progress) tea.Cmd {
	// Remove from active
	delete(h.activeProgress, progress.ID)

	// Add to completed history
	if len(h.completedProgress) >= h.maxHistory {
		h.completedProgress = h.completedProgress[1:]
	}
	h.completedProgress = append(h.completedProgress, progress)

	return func() tea.Msg {
		// TODO: Fix type mismatch in messages package
		// return messages.NewCommandCompletedMessage(
		//	progress.CommandID,
		//	progress.CommandType,
		//	ProgressResult{
		//		ID:         progress.ID,
		//		Success:    true,
		//		Duration:   time.Since(progress.StartTime),
		//		FinalStats: h.calculateFinalStats(progress),
		//	},
		//	time.Since(progress.StartTime),
		//	commands.SystemOperationContext(0),
		// )
		return tea.Msg(nil)
	}
}

// calculateFinalStats calculates final statistics for a completed progress
func (h *ProgressHandler) calculateFinalStats(progress *Progress) map[string]interface{} {
	return map[string]interface{}{
		"total_steps":      progress.Total,
		"completed_steps":  progress.Current,
		"duration":         time.Since(progress.StartTime),
		"average_step_time": time.Since(progress.StartTime) / time.Duration(progress.Current),
		"sub_progress_count": len(progress.SubProgress),
	}
}

// estimateDuration estimates the total duration based on current progress
func (h *ProgressHandler) estimateDuration(progress *Progress) time.Duration {
	if progress.Current == 0 {
		return time.Minute // Default estimate
	}

	elapsed := time.Since(progress.StartTime)
	if progress.Total > 0 {
		return time.Duration(float64(elapsed) * float64(progress.Total) / float64(progress.Current))
	}
	return elapsed
}

// checkTimeouts checks for progress operations that have timed out
func (h *ProgressHandler) checkTimeouts() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	timeout := 5 * time.Minute // Default timeout

	for _, progress := range h.activeProgress {
		if progress.Status == ProgressRunning && now.Sub(progress.StartTime) > timeout {
			progress.Status = ProgressFailed
			progress.Message = "Operation timed out"

			// Remove from active and add to completed
			delete(h.activeProgress, progress.ID)
			if len(h.completedProgress) >= h.maxHistory {
				h.completedProgress = h.completedProgress[1:]
			}
			h.completedProgress = append(h.completedProgress, progress)
		}
	}
}

// GetProgress returns a progress operation by ID
func (h *ProgressHandler) GetProgress(progressID string) (*Progress, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check active progress
	if progress, exists := h.activeProgress[progressID]; exists {
		// Return a copy
		return h.copyProgress(progress), true
	}

	// Check completed progress
	for _, progress := range h.completedProgress {
		if progress.ID == progressID {
			return h.copyProgress(progress), true
		}
	}

	return nil, false
}

// GetActiveProgress returns all active progress operations
func (h *ProgressHandler) GetActiveProgress() map[string]*Progress {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*Progress)
	for id, progress := range h.activeProgress {
		result[id] = h.copyProgress(progress)
	}
	return result
}

// GetCompletedProgress returns recent completed progress operations
func (h *ProgressHandler) GetCompletedProgress(limit int) []*Progress {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit >= len(h.completedProgress) {
		result := make([]*Progress, len(h.completedProgress))
		for i, progress := range h.completedProgress {
			result[i] = h.copyProgress(progress)
		}
		return result
	}

	result := make([]*Progress, limit)
	start := len(h.completedProgress) - limit
	for i := 0; i < limit; i++ {
		result[i] = h.copyProgress(h.completedProgress[start+i])
	}
	return result
}

// copyProgress creates a deep copy of a progress object
func (h *ProgressHandler) copyProgress(original *Progress) *Progress {
	copy := &Progress{
		ID:             original.ID,
		CommandType:    original.CommandType,
		CommandID:      original.CommandID,
		Current:        original.Current,
		Total:          original.Total,
		Percentage:     original.Percentage,
		Message:        original.Message,
		StartTime:      original.StartTime,
		LastUpdate:     original.LastUpdate,
		EstimatedEnd:   original.EstimatedEnd,
		Status:         original.Status,
		SubProgress:    make(map[string]*SubProgress),
		Metadata:       make(map[string]interface{}),
	}

	for k, v := range original.SubProgress {
		copy.SubProgress[k] = &SubProgress{
			ID:         v.ID,
			Name:       v.Name,
			Current:    v.Current,
			Total:      v.Total,
			Percentage: v.Percentage,
			Status:     v.Status,
		}
	}

	for k, v := range original.Metadata {
		copy.Metadata[k] = v
	}

	return copy
}

// GetStats returns progress handler statistics
func (h *ProgressHandler) GetStats() ProgressStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := ProgressStats{
		TotalOperations:     len(h.activeProgress) + len(h.completedProgress),
		ActiveOperations:    len(h.activeProgress),
		CompletedOperations:  len(h.completedProgress),
		OperationsByType:    make(map[commands.CommandType]int),
	}

	var totalDuration time.Duration
	var minDuration time.Duration = math.MaxInt64
	var maxDuration time.Duration
	var durationCount int

	// Count active operations
	for _, progress := range h.activeProgress {
		stats.OperationsByType[progress.CommandType]++
	}

	// Count completed operations and calculate durations
	for _, progress := range h.completedProgress {
		stats.OperationsByType[progress.CommandType]++
		duration := progress.LastUpdate.Sub(progress.StartTime)
		totalDuration += duration
		durationCount++

		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	if durationCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(durationCount)
		stats.FastestOperation = minDuration
		stats.SlowestOperation = maxDuration
	}

	return stats
}

// Cleanup removes old completed progress operations
func (h *ProgressHandler) Cleanup(maxAge time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	updated := make([]*Progress, 0, len(h.completedProgress))

	for _, progress := range h.completedProgress {
		if progress.LastUpdate.After(cutoff) {
			updated = append(updated, progress)
		}
	}

	h.completedProgress = updated
}

// ProgressResult represents the result of a completed progress operation
type ProgressResult struct {
	ID         string                 `json:"id"`
	Success    bool                   `json:"success"`
	Duration   time.Duration          `json:"duration"`
	FinalStats map[string]interface{} `json:"finalStats"`
}

// FormatProgress formats a progress value as a string
func FormatProgress(current, total int) string {
	if total <= 0 {
		return fmt.Sprintf("%d/?", current)
	}
	percentage := float64(current) / float64(total) * 100
	return fmt.Sprintf("%d/%d (%.1f%%)", current, total, percentage)
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// EstimateTimeRemaining estimates the time remaining for a progress operation
func (h *ProgressHandler) EstimateTimeRemaining(progress *Progress) time.Duration {
	if progress.Current <= 0 || progress.Percentage <= 0 {
		return 0
	}

	elapsed := time.Since(progress.StartTime)
	if progress.Percentage >= 100 {
		return 0
	}

	remaining := time.Duration(float64(elapsed) * (100.0-progress.Percentage) / progress.Percentage)
	return remaining
}

// CreateProgressBar creates a textual progress bar
func CreateProgressBar(current, total int, width int) string {
	if total <= 0 {
		return strings.Repeat("?", width)
	}

	percentage := float64(current) / float64(total)
	filled := int(float64(width) * percentage)
	empty := width - filled

	return fmt.Sprintf("[%s%s] %.1f%%",
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
		percentage*100)
}