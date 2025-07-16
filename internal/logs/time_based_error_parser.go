package logs

import (
	"fmt"
	"strings"
	"time"
)

// TimeBasedErrorCluster represents a group of log lines that form one error
type TimeBasedErrorCluster struct {
	ID          string
	ProcessID   string
	ProcessName string
	StartTime   time.Time
	EndTime     time.Time
	Lines       []LogEntry
	ErrorType   string
	Message     string
	Severity    string
}

// TimeBasedErrorParser groups error lines using time gaps
type TimeBasedErrorParser struct {
	// Active clusters being built per process
	activeClusters map[string]*TimeBasedErrorCluster
	
	// Completed error clusters
	completedClusters []TimeBasedErrorCluster
	
	// Time gap threshold to trigger cluster completion
	timeGapThreshold time.Duration
	
	// Minimum lines to consider as an error cluster
	minClusterSize int
}

func NewTimeBasedErrorParser() *TimeBasedErrorParser {
	return &TimeBasedErrorParser{
		activeClusters:   make(map[string]*TimeBasedErrorCluster),
		completedClusters: make([]TimeBasedErrorCluster, 0),
		timeGapThreshold: 200 * time.Millisecond, // 200ms gap triggers completion
		minClusterSize:   1, // Even single lines can be errors
	}
}

// ProcessLogEntry processes a log entry and potentially completes error clusters
func (p *TimeBasedErrorParser) ProcessLogEntry(entry LogEntry, processName string, isError bool) *TimeBasedErrorCluster {
	// Only process error lines
	if !isError && entry.Level < LevelError {
		return nil
	}
	
	processKey := entry.ProcessID
	
	// Check if we have an active cluster for this process
	if activeCluster, exists := p.activeClusters[processKey]; exists {
		// Check time gap
		timeSinceLastLine := entry.Timestamp.Sub(activeCluster.EndTime)
		
		if timeSinceLastLine > p.timeGapThreshold {
			// Time gap detected - finalize the current cluster
			p.finalizeCluster(activeCluster)
			completed := *activeCluster
			delete(p.activeClusters, processKey)
			
			// Start a new cluster with this entry
			p.startNewCluster(entry, processName, processKey)
			
			return &completed
		} else {
			// Add to existing cluster
			activeCluster.Lines = append(activeCluster.Lines, entry)
			activeCluster.EndTime = entry.Timestamp
			p.updateClusterAnalysis(activeCluster)
			return nil
		}
	} else {
		// Start new cluster
		p.startNewCluster(entry, processName, processKey)
		return nil
	}
}

// ForceCompleteAll completes all active clusters (useful for shutdown)
func (p *TimeBasedErrorParser) ForceCompleteAll() []TimeBasedErrorCluster {
	var completed []TimeBasedErrorCluster
	
	for processKey, cluster := range p.activeClusters {
		p.finalizeCluster(cluster)
		completed = append(completed, *cluster)
		delete(p.activeClusters, processKey)
	}
	
	return completed
}

// GetCompletedClusters returns all completed error clusters
func (p *TimeBasedErrorParser) GetCompletedClusters() []TimeBasedErrorCluster {
	return p.completedClusters
}

func (p *TimeBasedErrorParser) startNewCluster(entry LogEntry, processName, processKey string) {
	cluster := &TimeBasedErrorCluster{
		ID:          fmt.Sprintf("%s-%d", processKey, entry.Timestamp.UnixNano()),
		ProcessID:   entry.ProcessID,
		ProcessName: processName,
		StartTime:   entry.Timestamp,
		EndTime:     entry.Timestamp,
		Lines:       []LogEntry{entry},
	}
	
	p.updateClusterAnalysis(cluster)
	p.activeClusters[processKey] = cluster
}

func (p *TimeBasedErrorParser) finalizeCluster(cluster *TimeBasedErrorCluster) {
	// Only finalize if it meets minimum size
	if len(cluster.Lines) >= p.minClusterSize {
		p.completedClusters = append(p.completedClusters, *cluster)
		
		// Keep only last 100 clusters to prevent memory growth
		if len(p.completedClusters) > 100 {
			p.completedClusters = p.completedClusters[1:]
		}
	}
}

func (p *TimeBasedErrorParser) updateClusterAnalysis(cluster *TimeBasedErrorCluster) {
	if len(cluster.Lines) == 0 {
		return
	}
	
	// Combine all content from the cluster
	var allContent []string
	for _, line := range cluster.Lines {
		allContent = append(allContent, line.Content)
	}
	
	combinedContent := strings.Join(allContent, "\n")
	
	// Simple error type detection on the combined content
	cluster.ErrorType = p.detectErrorType(combinedContent)
	cluster.Message = p.extractMainMessage(cluster.Lines[0].Content) // Use first line as primary message
	cluster.Severity = p.determineSeverity(combinedContent)
}

func (p *TimeBasedErrorParser) detectErrorType(content string) string {
	content = strings.ToLower(content)
	
	// Check for specific error types in order of specificity
	errorTypes := map[string][]string{
		"MongoError": {"mongoerror", "mongo", "mongodb"},
		"TypeError": {"typeerror", "cannot read property", "is not a function"},
		"ReferenceError": {"referenceerror", "is not defined"},
		"SyntaxError": {"syntaxerror", "unexpected token", "unexpected end"},
		"NetworkError": {"fetcherror", "enotfound", "connection", "network"},
		"CompilationError": {"compilation failed", "build failed", "compile error"},
		"LintError": {"eslint", "lint error", "tslint"},
		"RuntimeError": {"runtime error", "panic", "exception"},
	}
	
	for errorType, keywords := range errorTypes {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				return errorType
			}
		}
	}
	
	return "Error"
}

func (p *TimeBasedErrorParser) extractMainMessage(firstLine string) string {
	// Strip common prefixes
	cleaned := p.stripLogPrefixes(firstLine)
	
	// Limit message length for display
	if len(cleaned) > 200 {
		return cleaned[:197] + "..."
	}
	
	return cleaned
}

func (p *TimeBasedErrorParser) stripLogPrefixes(content string) string {
	// TODO: Implement sophisticated prefix stripping
	// For now, just return the content as-is
	cleaned := content
	
	return strings.TrimSpace(cleaned)
}

func (p *TimeBasedErrorParser) determineSeverity(content string) string {
	content = strings.ToLower(content)
	
	if strings.Contains(content, "critical") || strings.Contains(content, "fatal") || strings.Contains(content, "panic") {
		return "critical"
	}
	if strings.Contains(content, "error") || strings.Contains(content, "fail") {
		return "error"
	}
	if strings.Contains(content, "warn") {
		return "warning"
	}
	
	return "error" // Default for unknown errors
}

// ConvertToErrorContext converts a cluster to the existing ErrorContext format for compatibility
func (cluster *TimeBasedErrorCluster) ToErrorContext() ErrorContext {
	var rawLines []string
	var stackLines []string
	var contextLines []string
	
	for _, line := range cluster.Lines {
		rawLines = append(rawLines, line.Content)
		
		// Simple heuristics for stack vs context
		if strings.Contains(line.Content, " at ") || strings.Contains(line.Content, ".js:") || strings.Contains(line.Content, ".ts:") {
			stackLines = append(stackLines, line.Content)
		} else {
			contextLines = append(contextLines, line.Content)
		}
	}
	
	return ErrorContext{
		ID:          cluster.ID,
		ProcessID:   cluster.ProcessID,
		ProcessName: cluster.ProcessName,
		Timestamp:   cluster.StartTime,
		Type:        cluster.ErrorType,
		Message:     cluster.Message,
		Stack:       stackLines,
		Context:     contextLines,
		Severity:    cluster.Severity,
		Language:    "javascript", // Default for now
		Raw:         rawLines,
	}
}