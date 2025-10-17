package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// IOCommand represents an I/O operation command
type IOCommand struct {
	*BaseCommand
	operation func() (interface{}, error)
	filePath  string
	operationType IOOperationType
}

// IOOperationType represents the type of I/O operation
type IOOperationType int

const (
	IOReadFile IOOperationType = iota
	IOWriteFile
	IOReadDir
	IOMkdir
	IORemove
	IOStat
	IOCopy
)

// IOResultMsg represents the result of an I/O command
type IOResultMsg struct {
	ID       string        `json:"id"`
	Type     CommandType    `json:"type"`
	Result   interface{}   `json:"result,omitempty"`
	Error    error         `json:"error,omitempty"`
	Context  OperationContext `json:"context"`
	Duration time.Duration `json:"duration"`
	FilePath string        `json:"filePath,omitempty"`
	OperationType IOOperationType `json:"operationType"`
}

// NewIOCommand creates a new I/O command
func NewIOCommand(id string, filePath string, opType IOOperationType, operation func() (interface{}, error), context OperationContext) *IOCommand {
	return &IOCommand{
		BaseCommand: NewBaseCommand(id, CommandTypeIO, PriorityMedium, context),
		operation:   operation,
		filePath:    filePath,
		operationType: opType,
	}
}

// Run executes the I/O command
func (c *IOCommand) Run() tea.Msg {
	start := time.Now()

	// Create command context for timeout handling
	cmdCtx := NewCommandContext(c.Context())
	defer cmdCtx.Cleanup()

	var result interface{}
	var err error

	// Execute operation with timeout
	result, err = cmdCtx.ExecuteWithTimeout(c.operation)
	duration := time.Since(start)

	// Check for timeout
	if cmdCtx.IsTimedOut() {
		err = fmt.Errorf("I/O operation timed out after %v", duration)
	}

	return IOResultMsg{
		ID:           c.ID(),
		Type:         c.Type(),
		Result:       result,
		Error:        err,
		Context:      c.Context(),
		Duration:     duration,
		FilePath:     c.filePath,
		OperationType: c.operationType,
	}
}

// NewReadFileCommand creates a command to read a file
func NewReadFileCommand(id string, filePath string, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		return FileData{
			Path:     filePath,
			Content:  data,
			Size:     len(data),
			Modified: getFileModTime(filePath),
		}, nil
	}

	return NewIOCommand(id, filePath, IOReadFile, operation, context)
}

// NewWriteFileCommand creates a command to write a file
func NewWriteFileCommand(id string, filePath string, content []byte, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", filePath, err)
		}

		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}

		return WriteResult{
			Path:   filePath,
			Bytes:  len(content),
			Status: "success",
		}, nil
	}

	return NewIOCommand(id, filePath, IOWriteFile, operation, context)
}

// NewReadDirCommand creates a command to read a directory
func NewReadDirCommand(id string, dirPath string, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
		}

		var dirEntries []DirEntry
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue // Skip entries we can't get info for
			}

			dirEntries = append(dirEntries, DirEntry{
				Name:    entry.Name(),
				IsDir:   entry.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
			})
		}

		return DirData{
			Path:    dirPath,
			Entries: dirEntries,
			Count:   len(dirEntries),
		}, nil
	}

	return NewIOCommand(id, dirPath, IOReadDir, operation, context)
}

// NewMkdirCommand creates a command to create a directory
func NewMkdirCommand(id string, dirPath string, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}

		return MkdirResult{
			Path:   dirPath,
			Status: "created",
		}, nil
	}

	return NewIOCommand(id, dirPath, IOMkdir, operation, context)
}

// NewRemoveCommand creates a command to remove a file or directory
func NewRemoveCommand(id string, path string, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		// Check if it exists first
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}

		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("failed to remove %s: %w", path, err)
		}

		return RemoveResult{
			Path:   path,
			Status: "removed",
		}, nil
	}

	return NewIOCommand(id, path, IORemove, operation, context)
}

// NewStatCommand creates a command to get file/directory information
func NewStatCommand(id string, path string, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		return FileStat{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
			Mode:    info.Mode(),
		}, nil
	}

	return NewIOCommand(id, path, IOStat, operation, context)
}

// NewCopyCommand creates a command to copy a file
func NewCopyCommand(id string, srcPath, dstPath string, context OperationContext) *IOCommand {
	operation := func() (interface{}, error) {
		src, err := os.Open(srcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open source file %s: %w", srcPath, err)
		}
		defer src.Close()

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", dstPath, err)
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination file %s: %w", dstPath, err)
		}
		defer dst.Close()

		bytesCopied, err := io.Copy(dst, src)
		if err != nil {
			return nil, fmt.Errorf("failed to copy from %s to %s: %w", srcPath, dstPath, err)
		}

		return CopyResult{
			SrcPath:      srcPath,
			DstPath:      dstPath,
			BytesCopied:  bytesCopied,
			Status:       "success",
		}, nil
	}

	return NewIOCommand(id, srcPath, IOCopy, operation, context)
}

// Data structures for I/O operations

type FileData struct {
	Path     string    `json:"path"`
	Content  []byte    `json:"content"`
	Size     int       `json:"size"`
	Modified time.Time `json:"modified"`
}

type WriteResult struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	Status string `json:"status"`
}

type DirEntry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Mode    os.FileMode `json:"mode"`
}

type DirData struct {
	Path    string     `json:"path"`
	Entries []DirEntry `json:"entries"`
	Count   int        `json:"count"`
}

type MkdirResult struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type RemoveResult struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type FileStat struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
	Mode    os.FileMode `json:"mode"`
}

type CopyResult struct {
	SrcPath      string `json:"srcPath"`
	DstPath      string `json:"dstPath"`
	BytesCopied  int64  `json:"bytesCopied"`
	Status       string `json:"status"`
}

// Helper functions

func getFileModTime(filePath string) time.Time {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// Message interface implementation
func (m IOResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the I/O operation completed successfully
func (m IOResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the I/O operation failed
func (m IOResultMsg) IsFailure() bool {
	return m.Error != nil
}

// String returns a string representation of the I/O result
func (m IOResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("IOResult{id=%s, type=%v, path=%s, duration=%v}",
			m.ID, m.OperationType, m.FilePath, m.Duration)
	}
	return fmt.Sprintf("IOResult{id=%s, type=%v, path=%s, error=%v, duration=%v}",
		m.ID, m.OperationType, m.FilePath, m.Error, m.Duration)
}