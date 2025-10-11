package messages

import (
	"time"
)

// File browser related messages for the TUI system

// FileBrowserPathChangedMsg indicates the current directory path has changed
type FileBrowserPathChangedMsg struct {
	Path      string
	Timestamp time.Time
}

// FileBrowserDirectoryListedMsg indicates a directory listing has been completed
type FileBrowserDirectoryListedMsg struct {
	Path      string
	FileCount int
	Error     string
}

// FileBrowserFileEventMsg indicates a file system event has occurred
type FileBrowserFileEventMsg struct {
	Path      string
	Operation string // "create", "write", "remove", "rename"
	Timestamp time.Time
	FileInfo  interface{} // *FileInfo
}

// FileBrowserWatchStartedMsg indicates file watching has started for a path
type FileBrowserWatchStartedMsg struct {
	Path      string
	Timestamp time.Time
}

// FileBrowserWatchStoppedMsg indicates file watching has stopped for a path
type FileBrowserWatchStoppedMsg struct {
	Path      string
	Timestamp time.Time
}

// FileBrowserSearchCompletedMsg indicates a file search has completed
type FileBrowserSearchCompletedMsg struct {
	Pattern    string
	Path       string
	MatchCount int
	TookMs     int64
}

// FileBrowserWorkspaceDetectedMsg indicates a workspace root has been detected
type FileBrowserWorkspaceDetectedMsg struct {
	RootPath  string
	Timestamp time.Time
}

// FileBrowserErrorMsg indicates a file browser operation error
type FileBrowserErrorMsg struct {
	Operation string
	Path      string
	Error     string
	Timestamp time.Time
}
