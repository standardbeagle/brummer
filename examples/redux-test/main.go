package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

// Simple test to validate the Redux/Elm architecture works
type simpleModel struct {
	currentView string
	count       int
}

func (m simpleModel) Init() tea.Cmd {
	return nil
}

func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "1":
			m.currentView = "processes"
		case "2":
			m.currentView = "logs"
		case " ":
			m.count++
		}
	}
	return m, nil
}

func (m simpleModel) View() string {
	return fmt.Sprintf(
		"Redux/Elm TUI Test\n\n"+
			"Current View: %s\n"+
			"Count: %d\n\n"+
			"Press 1 for Processes, 2 for Logs, Space to increment, q to quit",
		m.currentView, m.count,
	)
}

func main() {
	p := tea.NewProgram(simpleModel{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
	fmt.Println("Redux/Elm architecture working!")
}