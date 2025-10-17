package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CommandFactory provides a centralized way to create commands
type CommandFactory struct {
	nextID     int
	context    OperationContext
	version    uint64
	model      interface{}
}

// NewCommandFactory creates a new command factory
func NewCommandFactory(context OperationContext, version uint64, model interface{}) *CommandFactory {
	return &CommandFactory{
		nextID:  1,
		context: context,
		version: version,
		model:   model,
	}
}

// GenerateID generates a unique command ID
func (f *CommandFactory) GenerateID() string {
	id := fmt.Sprintf("cmd-%d", f.nextID)
	f.nextID++
	return id
}

// UpdateVersion updates the state version for new commands
func (f *CommandFactory) UpdateVersion(version uint64) {
	f.version = version
	f.context.Version = version
}

// SetContext updates the operation context for new commands
func (f *CommandFactory) SetContext(context OperationContext) {
	f.context = context
}

// GetContext returns the current operation context
func (f *CommandFactory) GetContext() OperationContext {
	return f.context
}

// IO Command Factory Methods

// NewReadFileCommand creates a file read command
func (f *CommandFactory) NewReadFileCommand(filePath string) *IOCommand {
	id := f.GenerateID()
	return NewReadFileCommand(id, filePath, f.context)
}

// NewWriteFileCommand creates a file write command
func (f *CommandFactory) NewWriteFileCommand(filePath string, content []byte) *IOCommand {
	id := f.GenerateID()
	return NewWriteFileCommand(id, filePath, content, f.context)
}

// NewReadDirCommand creates a directory read command
func (f *CommandFactory) NewReadDirCommand(dirPath string) *IOCommand {
	id := f.GenerateID()
	return NewReadDirCommand(id, dirPath, f.context)
}

// NewMkdirCommand creates a directory creation command
func (f *CommandFactory) NewMkdirCommand(dirPath string) *IOCommand {
	id := f.GenerateID()
	return NewMkdirCommand(id, dirPath, f.context)
}

// NewRemoveCommand creates a file/directory removal command
func (f *CommandFactory) NewRemoveCommand(path string) *IOCommand {
	id := f.GenerateID()
	return NewRemoveCommand(id, path, f.context)
}

// NewStatCommand creates a file stat command
func (f *CommandFactory) NewStatCommand(path string) *IOCommand {
	id := f.GenerateID()
	return NewStatCommand(id, path, f.context)
}

// NewCopyCommand creates a file copy command
func (f *CommandFactory) NewCopyCommand(srcPath, dstPath string) *IOCommand {
	id := f.GenerateID()
	return NewCopyCommand(id, srcPath, dstPath, f.context)
}

// Process Command Factory Methods

// NewStartScriptCommand creates a script start command
func (f *CommandFactory) NewStartScriptCommand(scriptName string, args []string, workingDir string) *ProcessCommand {
	id := f.GenerateID()
	return NewStartScriptCommand(id, scriptName, args, workingDir, f.context)
}

// NewStartNPMCommand creates an npm script start command
func (f *CommandFactory) NewStartNPMCommand(scriptName string, workingDir string) *ProcessCommand {
	id := f.GenerateID()
	return NewStartNPMCommand(id, scriptName, workingDir, f.context)
}

// NewStartNodeCommand creates a Node.js process start command
func (f *CommandFactory) NewStartNodeCommand(scriptFile string, args []string, workingDir string) *ProcessCommand {
	id := f.GenerateID()
	return NewStartNodeCommand(id, scriptFile, args, workingDir, f.context)
}

// NewStopProcessCommand creates a process stop command
func (f *CommandFactory) NewStopProcessCommand(pid int) *ProcessCommand {
	id := f.GenerateID()
	return NewStopProcessCommand(id, pid, f.context)
}

// NewProcessCommand creates a generic process command
func (f *CommandFactory) NewProcessCommand(command []string, workingDir string) *ProcessCommand {
	id := f.GenerateID()
	return NewProcessCommand(id, command, ProcessStart, workingDir, f.context)
}

// Network Command Factory Methods

// NewGETCommand creates a GET request command
func (f *CommandFactory) NewGETCommand(url string, headers map[string]string, timeout time.Duration) *NetworkCommand {
	id := f.GenerateID()
	return NewGETCommand(id, url, headers, timeout, f.context)
}

// NewPOSTCommand creates a POST request command
func (f *CommandFactory) NewPOSTCommand(url string, body interface{}, headers map[string]string, timeout time.Duration) *NetworkCommand {
	id := f.GenerateID()
	return NewPOSTCommand(id, url, body, headers, timeout, f.context)
}

// NewPUTCommand creates a PUT request command
func (f *CommandFactory) NewPUTCommand(url string, body interface{}, headers map[string]string, timeout time.Duration) *NetworkCommand {
	id := f.GenerateID()
	return NewPUTCommand(id, url, body, headers, timeout, f.context)
}

// NewDELETECommand creates a DELETE request command
func (f *CommandFactory) NewDELETECommand(url string, headers map[string]string, timeout time.Duration) *NetworkCommand {
	id := f.GenerateID()
	return NewDELETECommand(id, url, headers, timeout, f.context)
}

// NewHealthCheckCommand creates a health check command
func (f *CommandFactory) NewHealthCheckCommand(url string, timeout time.Duration) *NetworkCommand {
	id := f.GenerateID()
	return NewHealthCheckCommand(id, url, timeout, f.context)
}

// NewAPICallCommand creates a general API call command
func (f *CommandFactory) NewAPICallCommand(method string, url string, body interface{}, headers map[string]string, timeout time.Duration) *NetworkCommand {
	id := f.GenerateID()
	return NewAPICallCommand(id, method, url, body, headers, timeout, f.context)
}

// System Command Factory Methods

// NewSystemInfoCommand creates a system info command
func (f *CommandFactory) NewSystemInfoCommand() *SystemCommand {
	id := f.GenerateID()
	return NewSystemInfoCommand(id, f.context)
}

// NewEnvironmentCommand creates an environment variables command
func (f *CommandFactory) NewEnvironmentCommand() *SystemCommand {
	id := f.GenerateID()
	return NewEnvironmentCommand(id, f.context)
}

// NewUptimeCommand creates a system uptime command
func (f *CommandFactory) NewUptimeCommand() *SystemCommand {
	id := f.GenerateID()
	return NewUptimeCommand(id, f.context)
}

// NewMemoryCommand creates a memory info command
func (f *CommandFactory) NewMemoryCommand() *SystemCommand {
	id := f.GenerateID()
	return NewMemoryCommand(id, f.context)
}

// NewCPUCommand creates a CPU info command
func (f *CommandFactory) NewCPUCommand() *SystemCommand {
	id := f.GenerateID()
	return NewCPUCommand(id, f.context)
}

// NewTempCommand creates a temperature info command
func (f *CommandFactory) NewTempCommand() *SystemCommand {
	id := f.GenerateID()
	return NewTempCommand(id, f.context)
}

// NewDiskCommand creates a disk info command
func (f *CommandFactory) NewDiskCommand(path string) *SystemCommand {
	id := f.GenerateID()
	return NewDiskCommand(id, path, f.context)
}

// Retry Wrapper Factory Methods

// NewRetryableIOCommand creates a retryable I/O command
func (f *CommandFactory) NewRetryableIOCommand(command Command) *RetryWrapper {
	return NewRetryableIOCommand(command)
}

// NewRetryableNetworkCommand creates a retryable network command
func (f *CommandFactory) NewRetryableNetworkCommand(command Command) *RetryWrapper {
	return NewRetryableNetworkCommand(command)
}

// NewRetryableProcessCommand creates a retryable process command
func (f *CommandFactory) NewRetryableProcessCommand(command Command) *RetryWrapper {
	return NewRetryableProcessCommand(command)
}

// NewRetryableSystemCommand creates a retryable system command
func (f *CommandFactory) NewRetryableSystemCommand(command Command) *RetryWrapper {
	return NewRetryableSystemCommand(command)
}

// Batch Command Factory Methods

// NewSequentialBatch creates a sequential batch command
func (f *CommandFactory) NewSequentialBatch(commands []Command) *BatchCommand {
	id := f.GenerateID()
	return NewSequentialBatch(id, commands, f.context, f.model)
}

// NewParallelBatch creates a parallel batch command
func (f *CommandFactory) NewParallelBatch(commands []Command) *BatchCommand {
	id := f.GenerateID()
	return NewParallelBatch(id, commands, f.context, f.model)
}

// NewLimitedParallelBatch creates a limited parallel batch command
func (f *CommandFactory) NewLimitedParallelBatch(commands []Command, limit int) *BatchCommand {
	id := f.GenerateID()
	return NewLimitedParallelBatch(id, commands, limit, f.context, f.model)
}

// NewFailFastBatch creates a fail-fast batch command
func (f *CommandFactory) NewFailFastBatch(commands []Command, mode BatchExecutionMode) *BatchCommand {
	id := f.GenerateID()
	return NewFailFastBatch(id, commands, mode, f.context, f.model)
}

// NewBatchCommandBuilder creates a new batch command builder
func (f *CommandFactory) NewBatchCommandBuilder() *BatchCommandBuilder {
	id := f.GenerateID()
	return NewBatchCommandBuilder(id, f.context, f.model)
}

// High-level convenience methods

// NewFileOperationBatch creates a batch for common file operations
func (f *CommandFactory) NewFileOperationBatch(operations []FileOperation) *BatchCommand {
	commands := make([]Command, len(operations))

	for i, op := range operations {
		switch op.Type {
		case FileOperationRead:
			commands[i] = f.NewReadFileCommand(op.Path)
		case FileOperationWrite:
			commands[i] = f.NewWriteFileCommand(op.Path, op.Content)
		case FileOperationCopy:
			commands[i] = f.NewCopyCommand(op.Path, op.DestinationPath)
		case FileOperationRemove:
			commands[i] = f.NewRemoveCommand(op.Path)
		case FileOperationMkdir:
			commands[i] = f.NewMkdirCommand(op.Path)
		}
	}

	return f.NewSequentialBatch(commands)
}

// NewProcessStartupBatch creates a batch for starting multiple processes
func (f *CommandFactory) NewProcessStartupBatch(processes []ProcessStartup) *BatchCommand {
	commands := make([]Command, len(processes))

	for i, proc := range processes {
		switch proc.Type {
		case ProcessStartupScript:
			commands[i] = f.NewStartScriptCommand(proc.ScriptName, proc.Args, proc.WorkingDir)
		case ProcessStartupNPM:
			commands[i] = f.NewStartNPMCommand(proc.ScriptName, proc.WorkingDir)
		case ProcessStartupNode:
			commands[i] = f.NewStartNodeCommand(proc.ScriptFile, proc.Args, proc.WorkingDir)
		case ProcessStartupCommand:
			commands[i] = f.NewProcessCommand(proc.Command, proc.WorkingDir)
		}
	}

	return f.NewParallelBatch(commands)
}

// NewNetworkCheckBatch creates a batch for checking multiple URLs
func (f *CommandFactory) NewNetworkCheckBatch(urls []string, timeout time.Duration) *BatchCommand {
	commands := make([]Command, len(urls))

	for i, url := range urls {
		commands[i] = f.NewHealthCheckCommand(url, timeout)
	}

	return f.NewLimitedParallelBatch(commands, 5) // Limit to 5 concurrent requests
}

// NewSystemInfoBatch creates a batch for collecting system information
func (f *CommandFactory) NewSystemInfoBatch() *BatchCommand {
	commands := []Command{
		f.NewSystemInfoCommand(),
		f.NewEnvironmentCommand(),
		f.NewUptimeCommand(),
		f.NewMemoryCommand(),
		f.NewCPUCommand(),
		f.NewTempCommand(),
	}

	return f.NewParallelBatch(commands)
}

// Project-specific convenience methods

// NewProjectScanCommand creates a command to scan a project directory
func (f *CommandFactory) NewProjectScanCommand(projectPath string) *BatchCommand {
	commands := []Command{
		f.NewReadDirCommand(projectPath),
		f.NewStatCommand(projectPath),
		f.NewSystemInfoCommand(),
	}

	// Check for package.json
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		commands = append(commands, f.NewReadFileCommand(packageJsonPath))
	}

	// Check for common config files
	configFiles := []string{
		"tsconfig.json",
		"webpack.config.js",
		"vite.config.js",
		"next.config.js",
		"tailwind.config.js",
	}

	for _, configFile := range configFiles {
		configPath := filepath.Join(projectPath, configFile)
		if _, err := os.Stat(configPath); err == nil {
			commands = append(commands, f.NewReadFileCommand(configPath))
		}
	}

	return f.NewSequentialBatch(commands)
}

// NewDevEnvironmentSetupCommand creates a command to set up a development environment
func (f *CommandFactory) NewDevEnvironmentSetupCommand(projectPath string) *BatchCommand {
	commands := []Command{
		// Check if node_modules exists
		f.NewStatCommand(filepath.Join(projectPath, "node_modules")),
		// Check package.json
		f.NewStatCommand(filepath.Join(projectPath, "package.json")),
		// Get system info
		f.NewSystemInfoCommand(),
		// Check available disk space
		f.NewDiskCommand(projectPath),
	}

	return f.NewSequentialBatch(commands)
}

// NewComprehensiveHealthCheckCommand creates a comprehensive health check command
func (f *CommandFactory) NewComprehensiveHealthCheckCommand(includeNetwork bool, urls []string) *BatchCommand {
	commands := []Command{
		f.NewSystemInfoCommand(),
		f.NewMemoryCommand(),
		f.NewCPUCommand(),
		f.NewDiskCommand("."),
	}

	if includeNetwork && len(urls) > 0 {
		networkCommands := make([]Command, len(urls))
		for i, url := range urls {
			networkCommands[i] = f.NewHealthCheckCommand(url, 10*time.Second)
		}
		// Create a batch for network commands and wrap it as a command
		networkBatch := f.NewLimitedParallelBatch(networkCommands, 3)
		commands = append(commands, networkBatch)
	}

	return f.NewParallelBatch(commands)
}

// Support structures for convenience methods

// FileOperation represents a file operation
type FileOperation struct {
	Type           FileOperationType
	Path           string
	Content        []byte
	DestinationPath string
}

type FileOperationType int

const (
	FileOperationRead FileOperationType = iota
	FileOperationWrite
	FileOperationCopy
	FileOperationRemove
	FileOperationMkdir
)

// ProcessStartup represents a process startup configuration
type ProcessStartup struct {
	Type        ProcessStartupType
	ScriptName  string
	ScriptFile  string
	Command     []string
	Args        []string
	WorkingDir  string
}

type ProcessStartupType int

const (
	ProcessStartupScript ProcessStartupType = iota
	ProcessStartupNPM
	ProcessStartupNode
	ProcessStartupCommand
)

// Utility functions for creating operations

// NewReadOperation creates a file read operation
func NewReadOperation(path string) FileOperation {
	return FileOperation{
		Type: FileOperationRead,
		Path: path,
	}
}

// NewWriteOperation creates a file write operation
func NewWriteOperation(path string, content []byte) FileOperation {
	return FileOperation{
		Type:    FileOperationWrite,
		Path:    path,
		Content: content,
	}
}

// NewCopyOperation creates a file copy operation
func NewCopyOperation(srcPath, dstPath string) FileOperation {
	return FileOperation{
		Type:           FileOperationCopy,
		Path:           srcPath,
		DestinationPath: dstPath,
	}
}

// NewRemoveOperation creates a file remove operation
func NewRemoveOperation(path string) FileOperation {
	return FileOperation{
		Type: FileOperationRemove,
		Path: path,
	}
}

// NewMkdirOperation creates a directory creation operation
func NewMkdirOperation(path string) FileOperation {
	return FileOperation{
		Type: FileOperationMkdir,
		Path: path,
	}
}

// NewScriptStartup creates a script startup configuration
func NewScriptStartup(scriptName string, args []string, workingDir string) ProcessStartup {
	return ProcessStartup{
		Type:       ProcessStartupScript,
		ScriptName: scriptName,
		Args:       args,
		WorkingDir: workingDir,
	}
}

// NewNPMStartup creates an npm startup configuration
func NewNPMStartup(scriptName string, workingDir string) ProcessStartup {
	return ProcessStartup{
		Type:       ProcessStartupNPM,
		ScriptName: scriptName,
		WorkingDir: workingDir,
	}
}

// NewNodeStartup creates a Node.js startup configuration
func NewNodeStartup(scriptFile string, args []string, workingDir string) ProcessStartup {
	return ProcessStartup{
		Type:       ProcessStartupNode,
		ScriptFile: scriptFile,
		Args:       args,
		WorkingDir: workingDir,
	}
}

// NewCommandStartup creates a command startup configuration
func NewCommandStartup(command []string, workingDir string) ProcessStartup {
	return ProcessStartup{
		Type:       ProcessStartupCommand,
		Command:    command,
		WorkingDir: workingDir,
	}
}