package harness

import (
	"fmt"
	"time"
)

// TestHarnessConfiguration contains configuration parameters for test execution
type TestHarnessConfiguration struct {
	TestTimeout     time.Duration `json:"test_timeout"`
	KeyInputDelay   time.Duration `json:"key_input_delay"`
	StateValidation bool          `json:"state_validation"`
	ReportLevel     ReportLevel   `json:"report_level"`
	TestScope       TestScope     `json:"test_scope"`
	TerminalRows    uint16        `json:"terminal_rows"`
	TerminalCols    uint16        `json:"terminal_cols"`
	EnablePTY       bool          `json:"enable_pty"`
	DebugMode       bool          `json:"debug_mode"`
}

// DefaultTestConfig returns a default test harness configuration
func DefaultTestConfig() TestHarnessConfiguration {
	return TestHarnessConfiguration{
		TestTimeout:     30 * time.Second,
		KeyInputDelay:   100 * time.Millisecond,
		StateValidation: true,
		ReportLevel:     ReportLevelDetailed,
		TestScope:       TestScopeAll,
		TerminalRows:    24,
		TerminalCols:    80,
		EnablePTY:       true,
		DebugMode:       false,
	}
}

// Validate checks if the configuration is valid
func (c TestHarnessConfiguration) Validate() error {
	if c.TestTimeout <= 0 {
		return fmt.Errorf("test timeout must be positive")
	}

	if c.KeyInputDelay < 0 {
		return fmt.Errorf("key input delay cannot be negative")
	}

	if c.TerminalRows == 0 {
		return fmt.Errorf("terminal rows must be positive")
	}

	if c.TerminalCols == 0 {
		return fmt.Errorf("terminal cols must be positive")
	}

	return nil
}