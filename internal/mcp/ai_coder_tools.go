package mcp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/standardbeagle/brummer/internal/aicoder"
)

// RegisterAICoderTools registers all AI coder management tools
func (s *StreamableServer) registerAICoderTools() {
	// ai_coder_create - Create and launch new AI coder instance
	s.tools["ai_coder_create"] = MCPTool{
		Name: "ai_coder_create",
		Description: `Create and launch a new AI coder instance with specified task and provider.

The AI coder will run as a persistent session that can generate code, handle errors, and integrate with Brummer's build and test systems.

For detailed documentation and examples, use: about tool="ai_coder_create"`,
		InputSchema: aiCoderCreateSchema,
		Handler: s.handleAICoderCreate,
	}

	// ai_coder_list - List active AI coders
	s.tools["ai_coder_list"] = MCPTool{
		Name: "ai_coder_list",
		Description: `List all active AI coder instances with current status and progress.

Shows running, paused, completed, and failed AI coders with their task details.

For detailed documentation and examples, use: about tool="ai_coder_list"`,
		InputSchema: aiCoderListSchema,
		Handler: s.handleAICoderList,
	}

	// ai_coder_control - Control AI coder state
	s.tools["ai_coder_control"] = MCPTool{
		Name: "ai_coder_control",
		Description: `Control AI coder state (start/pause/stop/resume).

Manages the lifecycle of AI coder instances with tmux-style session control.

For detailed documentation and examples, use: about tool="ai_coder_control"`,
		InputSchema: aiCoderControlSchema,
		Handler: s.handleAICoderControl,
	}

	// ai_coder_status - Get detailed status
	s.tools["ai_coder_status"] = MCPTool{
		Name: "ai_coder_status",
		Description: `Get detailed status and progress of a specific AI coder.

Returns comprehensive information including task, progress, workspace, and recent activity.

For detailed documentation and examples, use: about tool="ai_coder_status"`,
		InputSchema: aiCoderStatusSchema,
		Handler: s.handleAICoderStatus,
	}

	// ai_coder_workspace - Access workspace files
	s.tools["ai_coder_workspace"] = MCPTool{
		Name: "ai_coder_workspace",
		Description: `Access AI coder workspace files and directory structure.

Supports listing files and reading specific file contents from the coder's isolated workspace.

For detailed documentation and examples, use: about tool="ai_coder_workspace"`,
		InputSchema: aiCoderWorkspaceSchema,
		Handler: s.handleAICoderWorkspace,
	}

	// ai_coder_logs - Stream AI coder logs
	s.tools["ai_coder_logs"] = MCPTool{
		Name: "ai_coder_logs",
		Description: `Stream AI coder execution logs and activity.

Provides real-time or historical logs from AI coder sessions with optional file output.

For detailed documentation and examples, use: about tool="ai_coder_logs"`,
		InputSchema: aiCoderLogsSchema,
		Handler: s.handleAICoderLogs,
		Streaming: true,
		StreamingHandler: s.handleAICoderLogsStream,
	}
}

// getAICoderManager retrieves the AI coder manager from the server
// Returns nil if not available (feature not enabled)
func (s *StreamableServer) getAICoderManager() *aicoder.AICoderManager {
	// TODO: This will be set when AICoderManager is added to StreamableServer
	// For now, return nil to allow compilation
	return nil
}

// Handler functions

func (s *StreamableServer) handleAICoderCreate(args json.RawMessage) (interface{}, error) {
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		Task           string   `json:"task"`
		Provider       string   `json:"provider"`
		WorkspaceFiles []string `json:"workspace_files"`
		Name           string   `json:"name"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Task == "" {
		return nil, fmt.Errorf("task parameter is required")
	}

	// Create the AI coder
	req := aicoder.CreateCoderRequest{
		Task:           params.Task,
		Provider:       params.Provider,
		Name:           params.Name,
		WorkspaceFiles: params.WorkspaceFiles,
	}

	coder, err := manager.CreateCoder(s.getContext(), req)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to create AI coder: %v", err),
		}, nil
	}

	// Auto-start the coder
	if err := manager.StartCoder(coder.ID); err != nil {
		// Log but don't fail creation
		fmt.Printf("Warning: failed to auto-start AI coder: %v\n", err)
	}

	return map[string]interface{}{
		"id":        coder.ID,
		"name":      coder.Name,
		"status":    string(coder.Status),
		"provider":  coder.Provider,
		"workspace": coder.WorkspaceDir,
		"session":   coder.SessionID,
		"message":   fmt.Sprintf("AI coder created successfully with session ID: %s", coder.SessionID),
	}, nil
}

func (s *StreamableServer) handleAICoderList(args json.RawMessage) (interface{}, error) {
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		StatusFilter string `json:"status_filter"`
		Limit        int    `json:"limit"`
	}

	// Set defaults
	params.StatusFilter = "all"
	params.Limit = 20

	if err := json.Unmarshal(args, &params); err != nil {
		// Ignore unmarshal errors and use defaults
	}

	// Get all coders
	coders := manager.ListCoders()

	// Filter by status
	filteredCoders := make([]*aicoder.AICoderProcess, 0)
	for _, coder := range coders {
		if params.StatusFilter == "all" || string(coder.Status) == params.StatusFilter {
			filteredCoders = append(filteredCoders, coder)
		}
		if len(filteredCoders) >= params.Limit {
			break
		}
	}

	// Format response
	coderList := make([]map[string]interface{}, len(filteredCoders))
	for i, coder := range filteredCoders {
		coderList[i] = map[string]interface{}{
			"id":         coder.ID,
			"name":       coder.Name,
			"session_id": coder.SessionID,
			"status":     string(coder.Status),
			"provider":   coder.Provider,
			"task":       coder.Task,
			"progress":   fmt.Sprintf("%.1f%%", coder.Progress*100),
			"created_at": coder.CreatedAt.Format(time.RFC3339),
			"updated_at": coder.UpdatedAt.Format(time.RFC3339),
		}
	}

	return map[string]interface{}{
		"coders": coderList,
		"count":  len(coderList),
		"total":  len(coders),
	}, nil
}

func (s *StreamableServer) handleAICoderControl(args json.RawMessage) (interface{}, error) {
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		CoderID string `json:"coder_id"`
		Action  string `json:"action"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.CoderID == "" {
		return nil, fmt.Errorf("coder_id parameter is required")
	}

	if params.Action == "" {
		return nil, fmt.Errorf("action parameter is required")
	}

	var err error
	var message string

	switch params.Action {
	case "start":
		err = manager.StartCoder(params.CoderID)
		message = "AI coder started successfully"
	case "pause":
		err = manager.PauseCoder(params.CoderID)
		message = "AI coder paused successfully"
	case "resume":
		err = manager.ResumeCoder(params.CoderID)
		message = "AI coder resumed successfully"
	case "stop":
		err = manager.StopCoder(params.CoderID)
		message = "AI coder stopped successfully"
	default:
		return nil, fmt.Errorf("invalid action: %s (must be start, pause, resume, or stop)", params.Action)
	}

	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Get updated status
	coder, exists := manager.GetCoder(params.CoderID)
	if !exists {
		return map[string]interface{}{
			"message": message,
			"warning": "Could not retrieve updated status",
		}, nil
	}

	return map[string]interface{}{
		"message": message,
		"id":      coder.ID,
		"status":  string(coder.Status),
		"session": coder.SessionID,
	}, nil
}

func (s *StreamableServer) handleAICoderStatus(args json.RawMessage) (interface{}, error) {
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		CoderID string `json:"coder_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.CoderID == "" {
		return nil, fmt.Errorf("coder_id parameter is required")
	}

	coder, exists := manager.GetCoder(params.CoderID)
	if !exists {
		return map[string]interface{}{
			"error": fmt.Sprintf("AI coder %s not found", params.CoderID),
		}, nil
	}

	// Calculate runtime
	var runtime string
	if coder.Status == aicoder.StatusRunning || coder.Status == aicoder.StatusPaused {
		duration := time.Since(coder.CreatedAt)
		runtime = duration.Round(time.Second).String()
	}

	return map[string]interface{}{
		"id":              coder.ID,
		"name":            coder.Name,
		"session_id":      coder.SessionID,
		"status":          string(coder.Status),
		"provider":        coder.Provider,
		"task":            coder.Task,
		"progress":        coder.Progress,
		"progress_text":   fmt.Sprintf("%.1f%%", coder.Progress*100),
		"current_message": coder.CurrentMessage,
		"workspace":       coder.WorkspaceDir,
		"created_at":      coder.CreatedAt.Format(time.RFC3339),
		"updated_at":      coder.UpdatedAt.Format(time.RFC3339),
		"runtime":         runtime,
		"attached":        coder.AttachedSessions > 0,
		"attached_count":  coder.AttachedSessions,
	}, nil
}

func (s *StreamableServer) handleAICoderWorkspace(args json.RawMessage) (interface{}, error) {
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		CoderID  string `json:"coder_id"`
		Operation string `json:"operation"`
		FilePath string `json:"file_path"`
	}

	// Default operation
	params.Operation = "list"

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.CoderID == "" {
		return nil, fmt.Errorf("coder_id parameter is required")
	}

	coder, exists := manager.GetCoder(params.CoderID)
	if !exists {
		return map[string]interface{}{
			"error": fmt.Sprintf("AI coder %s not found", params.CoderID),
		}, nil
	}

	switch params.Operation {
	case "list":
		files, err := coder.ListWorkspaceFiles()
		if err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to list workspace files: %v", err),
			}, nil
		}

		return map[string]interface{}{
			"workspace": coder.WorkspaceDir,
			"files":     files,
			"count":     len(files),
		}, nil

	case "read":
		if params.FilePath == "" {
			return nil, fmt.Errorf("file_path parameter is required for read operation")
		}

		content, err := coder.ReadWorkspaceFile(params.FilePath)
		if err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to read file: %v", err),
			}, nil
		}

		return map[string]interface{}{
			"workspace": coder.WorkspaceDir,
			"file_path": params.FilePath,
			"content":   string(content),
		}, nil

	default:
		return nil, fmt.Errorf("invalid operation: %s (must be list or read)", params.Operation)
	}
}

func (s *StreamableServer) handleAICoderLogs(args json.RawMessage) (interface{}, error) {
	// Non-streaming version returns recent logs
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		CoderID    string `json:"coder_id"`
		Limit      int    `json:"limit"`
		OutputFile string `json:"output_file"`
	}

	// Default limit
	params.Limit = 100

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.CoderID == "" {
		return nil, fmt.Errorf("coder_id parameter is required")
	}

	coder, exists := manager.GetCoder(params.CoderID)
	if !exists {
		return map[string]interface{}{
			"error": fmt.Sprintf("AI coder %s not found", params.CoderID),
		}, nil
	}

	// Get recent logs (mock implementation for now)
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"level":     "info",
			"message":   fmt.Sprintf("AI coder %s started", coder.SessionID),
		},
		{
			"timestamp": time.Now().Add(-4 * time.Minute).Format(time.RFC3339),
			"level":     "info",
			"message":   fmt.Sprintf("Processing task: %s", coder.Task),
		},
		{
			"timestamp": time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
			"level":     "info",
			"message":   fmt.Sprintf("Progress: %.1f%%", coder.Progress*100),
		},
	}

	result := map[string]interface{}{
		"coder_id":   coder.ID,
		"session_id": coder.SessionID,
		"logs":       logs,
		"count":      len(logs),
	}

	// Handle file output if requested
	if params.OutputFile != "" {
		if err := s.writeJSONToFile(params.OutputFile, result); err != nil {
			result["file_write_error"] = err.Error()
		} else {
			result["file_written"] = params.OutputFile
		}
	}

	return result, nil
}

func (s *StreamableServer) handleAICoderLogsStream(args json.RawMessage, send func(interface{})) (interface{}, error) {
	// Streaming version - sends log updates as they occur
	manager := s.getAICoderManager()
	if manager == nil {
		return map[string]interface{}{
			"error": "AI coder feature is not enabled",
		}, fmt.Errorf("AI coder manager not available")
	}

	var params struct {
		CoderID    string `json:"coder_id"`
		Follow     bool   `json:"follow"`
		OutputFile string `json:"output_file"`
	}

	// Default to follow mode
	params.Follow = true

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.CoderID == "" {
		return nil, fmt.Errorf("coder_id parameter is required")
	}

	coder, exists := manager.GetCoder(params.CoderID)
	if !exists {
		return map[string]interface{}{
			"error": fmt.Sprintf("AI coder %s not found", params.CoderID),
		}, nil
	}

	// Send initial logs
	initialLogs := []map[string]interface{}{
		{
			"timestamp": coder.CreatedAt.Format(time.RFC3339),
			"level":     "info",
			"message":   fmt.Sprintf("AI coder session %s created", coder.SessionID),
		},
	}

	for _, log := range initialLogs {
		send(log)
	}

	// If not following, return immediately
	if !params.Follow {
		return map[string]interface{}{
			"message": "Log streaming completed",
			"count":   len(initialLogs),
		}, nil
	}

	// TODO: In a real implementation, this would subscribe to AI coder log events
	// For now, simulate with a few updates
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(2 * time.Second)
			send(map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"level":     "info",
				"message":   fmt.Sprintf("Simulated log entry %d", i+1),
			})
		}
	}()

	return map[string]interface{}{
		"message": "Streaming logs started",
		"session": coder.SessionID,
	}, nil
}

