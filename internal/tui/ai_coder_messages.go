package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
)

// TUI Messages for AI Coder events

// AICoderListUpdatedMsg is sent when the list of AI coders changes
type AICoderListUpdatedMsg struct {
	Coders []*aicoder.AICoderProcess
}

// AICoderStatusUpdatedMsg is sent when an AI coder's status changes
type AICoderStatusUpdatedMsg struct {
	CoderID string
	Status  aicoder.AICoderStatus
	Message string
}

// AICoderSelectedMsg is sent when an AI coder is selected
type AICoderSelectedMsg struct {
	CoderID string
}

// AICoderCreatedMsg is sent when a new AI coder is created
type AICoderCreatedMsg struct {
	Coder *aicoder.AICoderProcess
}

// AICoderDeletedMsg is sent when an AI coder is deleted
type AICoderDeletedMsg struct {
	CoderID string
}

// AICoderCommandSentMsg is sent after a command is sent to an AI coder
type AICoderCommandSentMsg struct {
	CoderID string
	Command string
	Success bool
	Error   string
}

// AICoderProgressUpdatedMsg is sent when an AI coder's progress changes
type AICoderProgressUpdatedMsg struct {
	CoderID  string
	Progress float64
	Message  string
}

// AICoderOutputMsg is sent when an AI coder produces output
type AICoderOutputMsg struct {
	CoderID string
	Output  string
	IsError bool
}

// Command functions that return tea.Cmd

func (v AICoderView) refreshCoders() tea.Cmd {
	return func() tea.Msg {
		if v.manager == nil {
			return AICoderListUpdatedMsg{Coders: []*aicoder.AICoderProcess{}}
		}
		coders := v.manager.ListCoders()
		return AICoderListUpdatedMsg{Coders: coders}
	}
}

func (v AICoderView) createNewCoderWithTask(task string) tea.Cmd {
	return func() tea.Msg {
		req := aicoder.CreateCoderRequest{
			Task:     task,
			Provider: "mock", // TODO: Get from config when available
			Name:     fmt.Sprintf("coder-%d", len(v.coders)+1),
		}

		coder, err := v.manager.CreateCoder(context.Background(), req)
		if err != nil {
			return AICoderCommandSentMsg{
				Success: false,
				Error:   fmt.Sprintf("Failed to create AI coder: %v", err),
			}
		}

		return AICoderCreatedMsg{Coder: coder}
	}
}

func (v AICoderView) deleteCoder(coderID string) tea.Cmd {
	return func() tea.Msg {
		err := v.manager.DeleteCoder(coderID)
		if err != nil {
			return AICoderCommandSentMsg{
				CoderID: coderID,
				Success: false,
				Error:   fmt.Sprintf("Failed to delete AI coder: %v", err),
			}
		}

		return AICoderDeletedMsg{CoderID: coderID}
	}
}

func (v AICoderView) startCoder(coderID string) tea.Cmd {
	return func() tea.Msg {
		err := v.manager.StartCoder(coderID)
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		return AICoderCommandSentMsg{
			CoderID: coderID,
			Command: "start",
			Success: success,
			Error:   errorMsg,
		}
	}
}

func (v AICoderView) pauseCoder(coderID string) tea.Cmd {
	return func() tea.Msg {
		err := v.manager.PauseCoder(coderID)
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		return AICoderCommandSentMsg{
			CoderID: coderID,
			Command: "pause",
			Success: success,
			Error:   errorMsg,
		}
	}
}

func (v AICoderView) resumeCoder(coderID string) tea.Cmd {
	return func() tea.Msg {
		err := v.manager.ResumeCoder(coderID)
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		return AICoderCommandSentMsg{
			CoderID: coderID,
			Command: "resume",
			Success: success,
			Error:   errorMsg,
		}
	}
}

func (v AICoderView) sendCommand(coderID, command string) tea.Cmd {
	return func() tea.Msg {
		// For now, use UpdateCoderTask to send commands
		// TODO: Implement proper command interface when available
		err := v.manager.UpdateCoderTask(coderID, command)
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		return AICoderCommandSentMsg{
			CoderID: coderID,
			Command: command,
			Success: success,
			Error:   errorMsg,
		}
	}
}

// Event handling will be implemented in Task 06 - Event System Integration
// TODO: Implement event conversion and subscription when event system is ready

// Helper function to convert AI coder events to TUI messages (placeholder)
// func convertAICoderEvent(event aicoder.Event) tea.Msg {
// 	// Implementation will be added in Task 06
// 	return nil
// }

// subscribeToAICoderEvents returns a tea.Cmd that subscribes to AI coder events (placeholder)
// func subscribeToAICoderEvents(manager *aicoder.AICoderManager) tea.Cmd {
// 	return func() tea.Msg {
// 		// This would connect to the event system when integrated
// 		// For now, return nil as event system integration is Task 06
// 		return nil
// 	}
// }