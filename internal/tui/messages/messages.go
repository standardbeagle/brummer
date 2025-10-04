package messages

// CommandResultMsg is a message that indicates the result of a command.
type CommandResultMsg struct {
	Success bool
	Message string
}

// NavigateToViewMsg is a message that requests a view change.
type NavigateToViewMsg struct {
	View string
}

// ShowHelpMsg is a message that requests to show the help view.
type ShowHelpMsg struct{}

// ServiceStatusUpdateMsg is a message that updates the status of a service in the header.
type ServiceStatusUpdateMsg struct {
	Service string // "mcp", "proxy", "ai_coders", "processes"
	Status  string // "running", "stopped", "error", "starting"
	Details string // Optional details like port count, error message, etc.
}
