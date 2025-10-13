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

// Component-Service Communication Messages for File Browser
// These messages facilitate communication between FileBrowserService and FileBrowserComponent

// FileBrowserShownMsg is sent when a file browser dialog is opened
type FileBrowserShownMsg struct {
	DialogID    string    `json:"dialog_id"`
	InitialPath string    `json:"initial_path"`
	CallbackID  string    `json:"callback_id"`
	Time        time.Time `json:"time"`
}

// FileBrowserHiddenMsg is sent when a file browser dialog is closed
type FileBrowserHiddenMsg struct {
	DialogID string    `json:"dialog_id"`
	Trigger  string    `json:"trigger"` // "selection", "cancel", "escape"
	Time     time.Time `json:"time"`
}

// FileSelectionChangedMsg is sent when file selection changes
type FileSelectionChangedMsg struct {
	DialogID       string      `json:"dialog_id"`
	SelectedPath   string      `json:"selected_path"`
	SelectedIndex  int         `json:"selected_index"`
	FileInfo       interface{} `json:"file_info"` // FileInfo struct from service
	Time           time.Time   `json:"time"`
}

// FileBrowserNavigatedMsg is sent when directory navigation occurs
type FileBrowserNavigatedMsg struct {
	DialogID    string      `json:"dialog_id"`
	OldPath     string      `json:"old_path"`
	NewPath     string      `json:"new_path"`
	FileList    interface{} `json:"file_list"` // []FileInfo from service
	Time        time.Time   `json:"time"`
}

// FileBrowserFilterChangedMsg is sent when file filter is modified
type FileBrowserFilterChangedMsg struct {
	DialogID   string    `json:"dialog_id"`
	FilterType string    `json:"filter_type"`
	Filter     string    `json:"filter"`
	Time       time.Time `json:"time"`
}
