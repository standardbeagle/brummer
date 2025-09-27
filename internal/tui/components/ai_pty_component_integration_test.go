package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
)

// Test that starting a terminal PTY produces output and populates the viewport.
func TestAIPTY_StartTerminalProducesOutput(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	comp := NewAIPTYComponent(logStore)
	comp.SetSize(100, 30)

	// Start a terminal PTY (falls back to /bin/bash -i)
	_, cmd := comp.Update(AIPTYStartMsg{Provider: "terminal"})
	if cmd == nil {
		t.Fatalf("expected a command to subscribe after starting PTY")
	}

	// Run first subscription command (waits for OutputChan)
	// Give the shell a moment to emit a prompt
	done := make(chan struct{})
	var msg tea.Msg
	go func() {
		msg = cmd()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for initial PTY output")
	}

	if msg == nil {
		t.Fatalf("expected a message from subscription, got nil")
	}

	// Feed the message back into Update to refresh viewport
	_, next := comp.Update(msg)
	// We may or may not have further output; tolerate nil next
	_ = next

	// The viewport should have some content now (prompt or our echo)
	view := comp.View()
	if len(view) == 0 {
		t.Fatalf("expected non-empty view after PTY start")
	}
}

