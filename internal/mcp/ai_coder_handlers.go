package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// getContext returns a context for the current request
// In a real implementation, this might include request-specific context
func (s *StreamableServer) getContext() context.Context {
	return context.Background()
}

// Additional handler helpers for AI coder tools

// writeJSONToFile writes JSON data to a file with security validation
func (s *StreamableServer) writeJSONToFile(filePath string, data interface{}) error {
	// Security: prevent path traversal
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Ensure file is within project directory
	if !strings.HasPrefix(absPath, wd) {
		return fmt.Errorf("file path must be within project directory")
	}

	// Create directory if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(absPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// formatAICoderSummary creates a human-readable summary of AI coder status
func formatAICoderSummary(coders []map[string]interface{}) string {
	if len(coders) == 0 {
		return "No AI coders found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active AI Coders (%d):\n\n", len(coders)))

	for i, coder := range coders {
		sb.WriteString(fmt.Sprintf("%d. Session: %s\n", i+1, coder["session_id"]))
		sb.WriteString(fmt.Sprintf("   ID: %s\n", coder["id"]))
		sb.WriteString(fmt.Sprintf("   Status: %s\n", coder["status"]))
		sb.WriteString(fmt.Sprintf("   Provider: %s\n", coder["provider"]))
		sb.WriteString(fmt.Sprintf("   Progress: %s\n", coder["progress"]))
		sb.WriteString(fmt.Sprintf("   Task: %s\n", truncateString(coder["task"].(string), 80)))
		if i < len(coders)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// truncateString truncates a string to maxLen with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// validateWorkspacePath ensures a file path is safe to access within a workspace
func validateWorkspacePath(workspaceDir, filePath string) error {
	// Prevent path traversal
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Construct full path
	fullPath := filepath.Join(workspaceDir, filePath)

	// Ensure it's within workspace
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return fmt.Errorf("invalid workspace directory: %w", err)
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if !strings.HasPrefix(absPath, absWorkspace) {
		return fmt.Errorf("file path must be within workspace directory")
	}

	return nil
}

// readWorkspaceFile safely reads a file from an AI coder's workspace
func readWorkspaceFile(workspaceDir, filePath string) ([]byte, error) {
	if err := validateWorkspacePath(workspaceDir, filePath); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(workspaceDir, filePath)

	// Check file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Don't read directories
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory: %s", filePath)
	}

	// Limit file size to prevent memory issues
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	// Read file
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return io.ReadAll(file)
}

// listWorkspaceFiles returns a list of files in the workspace directory
func listWorkspaceFiles(workspaceDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the workspace directory itself
		if path == workspaceDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Add files (not directories)
		if !info.IsDir() {
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workspace files: %w", err)
	}

	return files, nil
}