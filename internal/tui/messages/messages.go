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
