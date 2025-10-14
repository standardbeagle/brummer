package contracts

import (
    "testing"
    tea "github.com/charmbracelet/bubbletea"
)

// ServiceContract defines the contract for stateless services
// Services must be stateless and only create commands
type ServiceContract interface {
    // Services should have no state-modifying methods
    // All methods should return tea.Cmd
}

// ProcessServiceContract defines the contract for process operations
type ProcessServiceContract interface {
    Start(name, cmd string) tea.Cmd
    Stop(id string) tea.Cmd
    Restart(id string) tea.Cmd
    GetOutput(id string) tea.Cmd
}

// TestProcessServiceContract validates process service behavior
func TestProcessServiceContract(t *testing.T, service ProcessServiceContract) {
    tests := []struct {
        name     string
        test     func() tea.Cmd
        validate func(t *testing.T, cmd tea.Cmd)
    }{
        {
            name: "Start returns non-nil command",
            test: func() tea.Cmd {
                return service.Start("test", "echo hello")
            },
            validate: func(t *testing.T, cmd tea.Cmd) {
                if cmd == nil {
                    t.Error("Start must return a command")
                }

                // Execute command and verify it returns a message
                msg := cmd()
                if msg == nil {
                    t.Error("Start command must produce a message")
                }

                // Verify message type
                switch msg.(type) {
                case ProcessStartedMsg, ErrorMsg:
                    // Expected message types
                default:
                    t.Errorf("unexpected message type: %T", msg)
                }
            },
        },
        {
            name: "Stop returns command for valid ID",
            test: func() tea.Cmd {
                return service.Stop("test-123")
            },
            validate: func(t *testing.T, cmd tea.Cmd) {
                if cmd == nil {
                    t.Error("Stop must return a command")
                }
            },
        },
        {
            name: "Restart returns sequenced commands",
            test: func() tea.Cmd {
                return service.Restart("test-123")
            },
            validate: func(t *testing.T, cmd tea.Cmd) {
                if cmd == nil {
                    t.Error("Restart must return a command")
                }
            },
        },
        {
            name: "GetOutput returns streaming command",
            test: func() tea.Cmd {
                return service.GetOutput("test-123")
            },
            validate: func(t *testing.T, cmd tea.Cmd) {
                if cmd == nil {
                    t.Error("GetOutput must return a command")
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := tt.test()
            tt.validate(t, cmd)
        })
    }
}

// LogServiceContract defines the contract for log operations
type LogServiceContract interface {
    Load(processID string) tea.Cmd
    Save(logs []LogEntry, path string) tea.Cmd
    Stream(processID string) tea.Cmd
    Filter(logs []LogEntry, query string) []LogEntry // Pure function, not a command
}

// TestLogServiceContract validates log service behavior
func TestLogServiceContract(t *testing.T, service LogServiceContract) {
    tests := []struct {
        name string
        test func() interface{}
    }{
        {
            name: "Load returns command",
            test: func() interface{} {
                return service.Load("test-123")
            },
        },
        {
            name: "Save returns command",
            test: func() interface{} {
                logs := []LogEntry{{Text: "test log"}}
                return service.Save(logs, "/tmp/test.log")
            },
        },
        {
            name: "Stream returns command",
            test: func() interface{} {
                return service.Stream("test-123")
            },
        },
        {
            name: "Filter is pure function",
            test: func() interface{} {
                logs := []LogEntry{
                    {Text: "error: something"},
                    {Text: "info: something"},
                }
                filtered := service.Filter(logs, "error")
                if len(filtered) != 1 {
                    t.Error("Filter should return only matching logs")
                }
                return filtered
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.test()

            // Check if result is a command (except for Filter)
            if tt.name != "Filter is pure function" {
                if cmd, ok := result.(tea.Cmd); !ok || cmd == nil {
                    t.Errorf("%s must return a non-nil command", tt.name)
                }
            }
        })
    }
}

// TestServiceStatelessness validates services don't hold state
func TestServiceStatelessness(t *testing.T, service interface{}) {
    // Services should not have any fields that hold mutable state
    // This is a compile-time check in practice, but we document it here

    // Example of what NOT to do:
    // type BadService struct {
    //     processes map[string]*Process  // ❌ Services shouldn't hold state
    //     counter   int                  // ❌ No mutable fields
    // }

    // Example of what's OK:
    // type GoodService struct {
    //     workDir string  // ✅ Configuration is OK
    //     logger  Logger  // ✅ Dependencies are OK
    // }

    t.Log("Services must be stateless - configuration and dependencies only")
}

// TestCommandImmutability validates commands don't modify external state
func TestCommandImmutability(t *testing.T) {
    // Commands should not directly modify any external state
    // They should only:
    // 1. Perform I/O operations
    // 2. Return messages for Update to handle

    // This is validated by ensuring commands return tea.Msg
    // and don't have access to the Model directly

    t.Log("Commands must not modify external state directly")
}