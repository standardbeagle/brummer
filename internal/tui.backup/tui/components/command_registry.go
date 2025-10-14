package components

import (
	"sync"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// CommandRegistry collects commands from all services
// This decouples the UI from service implementations
type CommandRegistry struct {
	commands map[string]messages.Command
	mu       sync.RWMutex
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]messages.Command),
	}
}

// Register adds a command to the registry
func (r *CommandRegistry) Register(cmd messages.Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Name] = cmd
}

// RegisterMultiple adds multiple commands at once
func (r *CommandRegistry) RegisterMultiple(cmds []messages.Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, cmd := range cmds {
		r.commands[cmd.Name] = cmd
	}
}

// Get retrieves a command by name
func (r *CommandRegistry) Get(name string) (messages.Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[name]
	return cmd, ok
}

// All returns all registered commands
func (r *CommandRegistry) All() []messages.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmds := make([]messages.Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// GetByCategory returns all commands in a category
func (r *CommandRegistry) GetByCategory(category string) []messages.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var cmds []messages.Command
	for _, cmd := range r.commands {
		if cmd.Category == category {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// GetNames returns all command names for autocomplete
func (r *CommandRegistry) GetNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}
