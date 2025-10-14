package messages

// Command represents a slash command that can be executed in the TUI
type Command struct {
	Name        string              // Command name without "/" (e.g., "run", "show", "ai")
	Description string              // Human-readable description
	Usage       string              // Usage example (e.g., "/run <script>")
	Category    string              // Category for grouping (e.g., "Process", "Logs", "AI")
	Arguments   []CommandArgument   // Expected arguments
	Execute     func(args []string) // Execution handler
}

// CommandArgument describes an argument to a command
type CommandArgument struct {
	Name        string   // Argument name (e.g., "script", "pattern")
	Description string   // What this argument is for
	Required    bool     // Is this argument required?
	Suggestions []string // Autocomplete suggestions (can be dynamic)
}

// CommandExecuteMsg is sent when a command should be executed
type CommandExecuteMsg struct {
	CommandName string
	Arguments   []string
}

// CommandRegisteredMsg is sent when new commands are registered
type CommandRegisteredMsg struct {
	Commands []Command
}
