#!/bin/bash

echo "Testing TUI startup..."
echo "Environment:"
echo "  TERM: $TERM"
echo "  PWD: $(pwd)"
echo "  TTY available: $(tty 2>/dev/null || echo 'no')"
echo ""

# Test if we can create a simple TTY-based program
cat > test_tty.go << 'EOF'
package main

import (
	"fmt"
	"os"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return "Simple TUI Test\nPress Ctrl+C or Esc to exit\n"
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("TUI Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("TUI exited successfully")
}
EOF

echo "Compiling test TUI..."
go build -o test_tty test_tty.go

echo "Running test TUI..."
timeout 5s ./test_tty || echo "Test TUI finished or timed out"

echo ""
echo "Cleaning up..."
rm -f test_tty test_tty.go