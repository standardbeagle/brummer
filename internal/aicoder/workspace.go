package aicoder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceManager handles workspace operations for AI coders
type WorkspaceManager struct {
	baseDir string
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager(baseDir string) (*WorkspaceManager, error) {
	// Expand home directory if needed
	if strings.HasPrefix(baseDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(home, baseDir[1:])
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &WorkspaceManager{
		baseDir: absPath,
	}, nil
}

// CreateWorkspace creates a new workspace directory for an AI coder
func (w *WorkspaceManager) CreateWorkspace(coderID string) (string, error) {
	workspaceDir := filepath.Join(w.baseDir, coderID)

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create initial directories
	for _, dir := range []string{"src", "docs", "tests", ".aicoder"} {
		if err := os.MkdirAll(filepath.Join(workspaceDir, dir), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create metadata file
	metadataPath := filepath.Join(workspaceDir, ".aicoder", "metadata.json")
	metadata := fmt.Sprintf(`{
  "coder_id": "%s",
  "created_at": "%s",
  "version": "1.0"
}`, coderID, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	return workspaceDir, nil
}

// ValidatePath validates that a path is within the workspace directory
func (w *WorkspaceManager) ValidatePath(workspaceDir, requestedPath string) error {
	// Clean and resolve the requested path
	cleanPath := filepath.Clean(requestedPath)

	// If it's not absolute, join with workspace dir
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(workspaceDir, cleanPath)
	}

	// Resolve any symlinks
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// If file doesn't exist yet, just use the cleaned path
		if os.IsNotExist(err) {
			resolvedPath = cleanPath
		} else {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Ensure the path is within the workspace
	relPath, err := filepath.Rel(workspaceDir, resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to determine relative path: %w", err)
	}

	// Check for directory traversal
	if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, "/../") {
		return fmt.Errorf("path traversal detected: %s", requestedPath)
	}

	return nil
}

// WriteFile writes content to a file within the workspace
func (w *WorkspaceManager) WriteFile(workspaceDir, relativePath string, content []byte) error {
	// Validate the path
	if err := w.ValidatePath(workspaceDir, relativePath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	fullPath := filepath.Join(workspaceDir, relativePath)

	// Create parent directory if needed
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReadFile reads a file from the workspace
func (w *WorkspaceManager) ReadFile(workspaceDir, relativePath string) ([]byte, error) {
	// Validate the path
	if err := w.ValidatePath(workspaceDir, relativePath); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	fullPath := filepath.Join(workspaceDir, relativePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// ListFiles lists all files in the workspace
func (w *WorkspaceManager) ListFiles(workspaceDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

// CopyFile copies a file from source to destination within the workspace
func (w *WorkspaceManager) CopyFile(workspaceDir, srcPath, dstPath string) error {
	// Validate both paths
	if err := w.ValidatePath(workspaceDir, srcPath); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := w.ValidatePath(workspaceDir, dstPath); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	srcFullPath := filepath.Join(workspaceDir, srcPath)
	dstFullPath := filepath.Join(workspaceDir, dstPath)

	// Open source file
	src, err := os.Open(srcFullPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Create destination directory if needed
	dstDir := filepath.Dir(dstFullPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dst, err := os.Create(dstFullPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// CleanupWorkspace removes a workspace directory
func (w *WorkspaceManager) CleanupWorkspace(workspaceDir string) error {
	// Ensure the workspace is within our base directory
	relPath, err := filepath.Rel(w.baseDir, workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to determine relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("workspace directory is outside base directory")
	}

	// Remove the workspace
	if err := os.RemoveAll(workspaceDir); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	return nil
}

// GetWorkspaceSize returns the total size of a workspace in bytes
func (w *WorkspaceManager) GetWorkspaceSize(workspaceDir string) (int64, error) {
	var size int64

	err := filepath.Walk(workspaceDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate workspace size: %w", err)
	}

	return size, nil
}
