package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// DiagnosticReport contains detailed information about discovery state
type DiagnosticReport struct {
	Timestamp       time.Time                 `json:"timestamp"`
	InstancesDir    string                    `json:"instances_dir"`
	DirPermissions  string                    `json:"dir_permissions"`
	DirExists       bool                      `json:"dir_exists"`
	FileCount       int                       `json:"file_count"`
	ValidInstances  map[string]*Instance      `json:"valid_instances"`
	InvalidFiles    map[string]string         `json:"invalid_files"`
	FilePermissions map[string]string         `json:"file_permissions"`
	ProcessStatus   map[string]ProcessStatus  `json:"process_status"`
	LockFileInfo    *LockFileInfo             `json:"lock_file_info"`
	SystemInfo      SystemInfo                `json:"system_info"`
	Errors          []string                  `json:"errors"`
}

// ProcessStatus contains information about a process
type ProcessStatus struct {
	PID     int    `json:"pid"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

// LockFileInfo contains information about the discovery lock file
type LockFileInfo struct {
	Exists     bool      `json:"exists"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	Locked     bool      `json:"locked"`
	LockError  string    `json:"lock_error,omitempty"`
}

// SystemInfo contains system-level information
type SystemInfo struct {
	TempDir         string `json:"temp_dir"`
	XDGRuntimeDir   string `json:"xdg_runtime_dir"`
	DefaultDir      string `json:"default_instances_dir"`
	CurrentUser     string `json:"current_user"`
	FileSystemType  string `json:"file_system_type,omitempty"`
}

// GenerateDiagnosticReport creates a comprehensive diagnostic report
func GenerateDiagnosticReport(instancesDir string) (*DiagnosticReport, error) {
	report := &DiagnosticReport{
		Timestamp:       time.Now(),
		InstancesDir:    instancesDir,
		ValidInstances:  make(map[string]*Instance),
		InvalidFiles:    make(map[string]string),
		FilePermissions: make(map[string]string),
		ProcessStatus:   make(map[string]ProcessStatus),
		Errors:          []string{},
		SystemInfo: SystemInfo{
			TempDir:       os.TempDir(),
			XDGRuntimeDir: os.Getenv("XDG_RUNTIME_DIR"),
			DefaultDir:    GetDefaultInstancesDir(),
			CurrentUser:   getCurrentUser(),
		},
	}

	// Check directory existence and permissions
	if info, err := os.Stat(instancesDir); err != nil {
		if os.IsNotExist(err) {
			report.DirExists = false
			report.Errors = append(report.Errors, fmt.Sprintf("Directory does not exist: %s", instancesDir))
		} else {
			report.Errors = append(report.Errors, fmt.Sprintf("Cannot stat directory: %v", err))
		}
	} else {
		report.DirExists = true
		report.DirPermissions = info.Mode().String()
		
		// Check if we can read the directory
		if entries, err := os.ReadDir(instancesDir); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Cannot read directory: %v", err))
		} else {
			report.FileCount = len(entries)
			
			// Process each file
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				
				filename := entry.Name()
				fullPath := filepath.Join(instancesDir, filename)
				
				// Get file permissions
				if info, err := os.Stat(fullPath); err == nil {
					report.FilePermissions[filename] = info.Mode().String()
				}
				
				// Skip non-JSON files
				if filepath.Ext(filename) != ".json" {
					continue
				}
				
				// Skip lock file
				if filename == ".discovery.lock" {
					report.LockFileInfo = getLockFileInfo(fullPath)
					continue
				}
				
				// Try to read and parse the file
				instanceID := extractInstanceID(filename)
				if instanceID == "" {
					report.InvalidFiles[filename] = "Invalid filename format"
					continue
				}
				
				data, err := os.ReadFile(fullPath)
				if err != nil {
					report.InvalidFiles[filename] = fmt.Sprintf("Read error: %v", err)
					continue
				}
				
				var instance Instance
				if err := json.Unmarshal(data, &instance); err != nil {
					report.InvalidFiles[filename] = fmt.Sprintf("JSON parse error: %v", err)
					continue
				}
				
				// Validate instance
				if err := validateInstance(&instance); err != nil {
					report.InvalidFiles[filename] = fmt.Sprintf("Validation error: %v", err)
					continue
				}
				
				// Check process status
				report.ProcessStatus[instanceID] = checkProcessStatus(instance.ProcessInfo.PID)
				
				// Add to valid instances
				report.ValidInstances[instanceID] = &instance
			}
		}
	}
	
	return report, nil
}

// PrintDiagnosticReport writes a human-readable diagnostic report
func PrintDiagnosticReport(w io.Writer, report *DiagnosticReport) {
	fmt.Fprintf(w, "Discovery Diagnostic Report\n")
	fmt.Fprintf(w, "==========================\n\n")
	
	fmt.Fprintf(w, "Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(w, "Instances Directory: %s\n", report.InstancesDir)
	fmt.Fprintf(w, "Directory Exists: %v\n", report.DirExists)
	fmt.Fprintf(w, "Directory Permissions: %s\n", report.DirPermissions)
	fmt.Fprintf(w, "File Count: %d\n\n", report.FileCount)
	
	fmt.Fprintf(w, "System Information:\n")
	fmt.Fprintf(w, "  Current User: %s\n", report.SystemInfo.CurrentUser)
	fmt.Fprintf(w, "  Temp Dir: %s\n", report.SystemInfo.TempDir)
	fmt.Fprintf(w, "  XDG_RUNTIME_DIR: %s\n", report.SystemInfo.XDGRuntimeDir)
	fmt.Fprintf(w, "  Default Instances Dir: %s\n\n", report.SystemInfo.DefaultDir)
	
	if len(report.Errors) > 0 {
		fmt.Fprintf(w, "ERRORS:\n")
		for _, err := range report.Errors {
			fmt.Fprintf(w, "  - %s\n", err)
		}
		fmt.Fprintf(w, "\n")
	}
	
	if report.LockFileInfo != nil {
		fmt.Fprintf(w, "Lock File:\n")
		fmt.Fprintf(w, "  Path: %s\n", report.LockFileInfo.Path)
		fmt.Fprintf(w, "  Exists: %v\n", report.LockFileInfo.Exists)
		if report.LockFileInfo.Exists {
			fmt.Fprintf(w, "  Size: %d bytes\n", report.LockFileInfo.Size)
			fmt.Fprintf(w, "  Modified: %s\n", report.LockFileInfo.ModTime.Format(time.RFC3339))
			fmt.Fprintf(w, "  Locked: %v\n", report.LockFileInfo.Locked)
		}
		if report.LockFileInfo.LockError != "" {
			fmt.Fprintf(w, "  Lock Error: %s\n", report.LockFileInfo.LockError)
		}
		fmt.Fprintf(w, "\n")
	}
	
	if len(report.ValidInstances) > 0 {
		fmt.Fprintf(w, "Valid Instances (%d):\n", len(report.ValidInstances))
		for id, inst := range report.ValidInstances {
			status := report.ProcessStatus[id]
			fmt.Fprintf(w, "  %s:\n", id)
			fmt.Fprintf(w, "    Name: %s\n", inst.Name)
			fmt.Fprintf(w, "    Port: %d\n", inst.Port)
			fmt.Fprintf(w, "    PID: %d (Running: %v)\n", inst.ProcessInfo.PID, status.Running)
			fmt.Fprintf(w, "    Started: %s\n", inst.StartedAt.Format(time.RFC3339))
			fmt.Fprintf(w, "    Last Ping: %s (%.1f minutes ago)\n", 
				inst.LastPing.Format(time.RFC3339),
				time.Since(inst.LastPing).Minutes())
			if status.Error != "" {
				fmt.Fprintf(w, "    Process Error: %s\n", status.Error)
			}
		}
		fmt.Fprintf(w, "\n")
	}
	
	if len(report.InvalidFiles) > 0 {
		fmt.Fprintf(w, "Invalid Files (%d):\n", len(report.InvalidFiles))
		for filename, reason := range report.InvalidFiles {
			fmt.Fprintf(w, "  %s: %s\n", filename, reason)
		}
		fmt.Fprintf(w, "\n")
	}
	
	if len(report.FilePermissions) > 0 {
		fmt.Fprintf(w, "File Permissions:\n")
		for filename, perms := range report.FilePermissions {
			fmt.Fprintf(w, "  %s: %s\n", filename, perms)
		}
		fmt.Fprintf(w, "\n")
	}
}

// VerifyDiscoverySetup performs comprehensive verification of discovery setup
func VerifyDiscoverySetup(instancesDir string) error {
	report, err := GenerateDiagnosticReport(instancesDir)
	if err != nil {
		return fmt.Errorf("failed to generate diagnostic report: %w", err)
	}
	
	// Check for critical issues
	var issues []string
	
	if !report.DirExists {
		issues = append(issues, "instances directory does not exist")
	}
	
	if report.DirExists && !canWriteToDirectory(instancesDir) {
		issues = append(issues, "cannot write to instances directory")
	}
	
	if report.LockFileInfo != nil && report.LockFileInfo.Locked {
		issues = append(issues, "discovery lock file is currently locked")
	}
	
	// Check for stale instances
	now := time.Now()
	for id, inst := range report.ValidInstances {
		if now.Sub(inst.LastPing) > StaleInstanceTimeout {
			status := report.ProcessStatus[id]
			if !status.Running {
				issues = append(issues, fmt.Sprintf("instance %s has stale ping and dead process", id))
			}
		}
	}
	
	if len(issues) > 0 {
		return fmt.Errorf("discovery setup issues: %s", strings.Join(issues, "; "))
	}
	
	return nil
}

// Helper functions

func getLockFileInfo(lockPath string) *LockFileInfo {
	info := &LockFileInfo{
		Path: lockPath,
	}
	
	if stat, err := os.Stat(lockPath); err == nil {
		info.Exists = true
		info.Size = stat.Size()
		info.ModTime = stat.ModTime()
		
		// Try to acquire lock to see if it's currently locked
		// This is non-blocking check
		if file, err := os.OpenFile(lockPath, os.O_RDWR, 0); err == nil {
			defer file.Close()
			
			// Try to get an exclusive lock (non-blocking)
			err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				// We got the lock, so it wasn't locked
				info.Locked = false
				syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
			} else {
				// Lock acquisition failed, file is locked
				info.Locked = true
				info.LockError = err.Error()
			}
		}
	}
	
	return info
}

func checkProcessStatus(pid int) ProcessStatus {
	status := ProcessStatus{PID: pid}
	
	if pid <= 0 {
		status.Error = "Invalid PID"
		return status
	}
	
	// Check using /proc filesystem (Linux)
	procPath := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(procPath); err != nil {
		if os.IsNotExist(err) {
			status.Running = false
		} else {
			status.Error = err.Error()
		}
	} else {
		status.Running = true
	}
	
	return status
}

func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

func canWriteToDirectory(dir string) bool {
	// Try to create a temp file
	tempFile := filepath.Join(dir, ".write-test-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	file, err := os.Create(tempFile)
	if err != nil {
		return false
	}
	
	file.Close()
	os.Remove(tempFile)
	return true
}

// DiagnoseDiscoveryIssue provides specific diagnosis for common issues
func DiagnoseDiscoveryIssue(instancesDir string) (string, error) {
	report, err := GenerateDiagnosticReport(instancesDir)
	if err != nil {
		return "", err
	}
	
	var diagnosis strings.Builder
	
	// No instances found
	if len(report.ValidInstances) == 0 && len(report.InvalidFiles) == 0 {
		if !report.DirExists {
			diagnosis.WriteString("The instances directory does not exist.\n")
			diagnosis.WriteString("This is normal if no instances have been started yet.\n")
			diagnosis.WriteString(fmt.Sprintf("Directory path: %s\n", instancesDir))
		} else if report.FileCount == 0 {
			diagnosis.WriteString("The instances directory exists but contains no instance files.\n")
			diagnosis.WriteString("Instances should automatically register when started with 'brum' command.\n")
		} else {
			diagnosis.WriteString("Files found in directory but none are valid instance files.\n")
			diagnosis.WriteString("Instance files should have .json extension and valid JSON content.\n")
		}
		return diagnosis.String(), nil
	}
	
	// Some instances found but with issues
	staleCount := 0
	deadCount := 0
	
	for id, inst := range report.ValidInstances {
		status := report.ProcessStatus[id]
		if !status.Running {
			deadCount++
		}
		if time.Since(inst.LastPing) > StaleInstanceTimeout {
			staleCount++
		}
	}
	
	if deadCount > 0 {
		diagnosis.WriteString(fmt.Sprintf("%d instance(s) have dead processes.\n", deadCount))
		diagnosis.WriteString("These instances may have crashed or been terminated.\n")
		diagnosis.WriteString("Run cleanup to remove stale instance files.\n\n")
	}
	
	if staleCount > 0 {
		diagnosis.WriteString(fmt.Sprintf("%d instance(s) have not updated their ping recently.\n", staleCount))
		diagnosis.WriteString("These instances may be hung or not running properly.\n\n")
	}
	
	if len(report.InvalidFiles) > 0 {
		diagnosis.WriteString(fmt.Sprintf("%d invalid file(s) found in instances directory.\n", len(report.InvalidFiles)))
		diagnosis.WriteString("These files may be corrupted or have invalid format.\n")
		for file, reason := range report.InvalidFiles {
			diagnosis.WriteString(fmt.Sprintf("  - %s: %s\n", file, reason))
		}
		diagnosis.WriteString("\n")
	}
	
	if diagnosis.Len() == 0 {
		diagnosis.WriteString("No obvious issues found with discovery setup.\n")
		diagnosis.WriteString(fmt.Sprintf("Found %d valid instance(s).\n", len(report.ValidInstances)))
	}
	
	return diagnosis.String(), nil
}