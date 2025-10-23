//go:build !windows

package pty

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PTYProgram wraps tea.Program for PTY-based testing
type PTYProgram struct {
	mu      sync.RWMutex
	program *tea.Program
	pty     *PTYLayerImpl
	model   tea.Model
	options []tea.ProgramOption
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewPTYProgram creates a new PTY program wrapper
func NewPTYProgram(model tea.Model, pty *PTYLayerImpl, options ...tea.ProgramOption) *PTYProgram {
	ctx, cancel := context.WithCancel(context.Background())

	// Add PTY-specific options for integration with BubbleTea
	ptyOptions := []tea.ProgramOption{
		tea.WithoutSignalHandler(),
		tea.WithContext(ctx),
		tea.WithInput(pty),              // Use PTY as input
		tea.WithOutput(pty),             // Use PTY as output
	}

	// Combine with user-provided options
	allOptions := append(ptyOptions, options...)

	return &PTYProgram{
		pty:     pty,
		model:   model,
		options: allOptions,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the PTY program
func (pp *PTYProgram) Start() error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.running {
		return fmt.Errorf("program already running")
	}

	if !pp.pty.IsAllocated() {
		return fmt.Errorf("PTY not allocated")
	}

	// Set PTY size before starting program
	size := pp.pty.GetSize() // Get PTY size for debug
	_ = size // Use size to avoid unused variable warning
	initialOptions := append(pp.options, tea.WithAltScreen())

	pp.program = tea.NewProgram(pp.model, initialOptions...)
	pp.running = true

	return nil
}

// Run runs the PTY program to completion
func (pp *PTYProgram) Run() (tea.Model, error) {
	if err := pp.Start(); err != nil {
		return nil, err
	}

	defer pp.Stop()

	model, err := pp.program.Run()
	if err != nil {
		return nil, err
	}

	pp.model = model
	return model, nil
}

// Stop stops the PTY program
func (pp *PTYProgram) Stop() error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if !pp.running {
		return nil
	}

	if pp.program != nil {
		pp.program.Quit()
		pp.program.ReleaseTerminal()
		pp.program = nil
	}

	pp.cancel()
	pp.running = false

	return nil
}

// IsRunning returns true if the program is running
func (pp *PTYProgram) IsRunning() bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	return pp.running
}

// GetModel returns the current model
func (pp *PTYProgram) GetModel() tea.Model {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	return pp.model
}

// GetContext returns the program context
func (pp *PTYProgram) GetContext() context.Context {
	return pp.ctx
}

// PTYInputController manages keyboard input simulation through PTY
type PTYInputController struct {
	mu       sync.RWMutex
	pty      *PTYLayerImpl
	queue    []tea.KeyMsg
	running  bool
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPTYInputController creates a new PTY input controller
func NewPTYInputController(pty *PTYLayerImpl) *PTYInputController {
	ctx, cancel := context.WithCancel(context.Background())

	return &PTYInputController{
		pty:     pty,
		queue:   make([]tea.KeyMsg, 0, 100),
		done:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the input controller
func (pic *PTYInputController) Start() {
	pic.mu.Lock()
	defer pic.mu.Unlock()

	if pic.running {
		return
	}

	pic.running = true
	go pic.processQueue()
}

// Stop stops the input controller
func (pic *PTYInputController) Stop() {
	pic.mu.Lock()
	defer pic.mu.Unlock()

	if !pic.running {
		return
	}

	pic.cancel()
	<-pic.done
	pic.running = false
}

// SendKey sends a key press to the PTY
func (pic *PTYInputController) SendKey(key tea.KeyMsg) error {
	pic.mu.Lock()
	defer pic.mu.Unlock()

	if !pic.running {
		return fmt.Errorf("input controller not running")
	}

	pic.queue = append(pic.queue, key)
	return nil
}

// SendKeys sends multiple key presses to the PTY
func (pic *PTYInputController) SendKeys(keys []tea.KeyMsg) error {
	pic.mu.Lock()
	defer pic.mu.Unlock()

	if !pic.running {
		return fmt.Errorf("input controller not running")
	}

	pic.queue = append(pic.queue, keys...)
	return nil
}

// processQueue processes the key queue
func (pic *PTYInputController) processQueue() {
	defer close(pic.done)

	for {
		select {
		case <-pic.ctx.Done():
			return
		default:
			pic.mu.Lock()
			if len(pic.queue) == 0 {
				pic.mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				continue
			}

			key := pic.queue[0]
			pic.queue = pic.queue[1:]
			pic.mu.Unlock()

			if err := pic.pty.SimulateKeyPress(key); err != nil {
				// Log error but continue processing
				fmt.Printf("Warning: failed to simulate key press: %v\n", err)
			}

			// Small delay between keystrokes for realism
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// PTYRenderer captures TUI output without display
type PTYRenderer struct {
	mu      sync.RWMutex
	pty     *PTYLayerImpl
	buffer  []byte
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewPTYRenderer creates a new PTY renderer
func NewPTYRenderer(pty *PTYLayerImpl) *PTYRenderer {
	ctx, cancel := context.WithCancel(context.Background())

	return &PTYRenderer{
		pty:    pty,
		buffer: make([]byte, 0, 4096),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the renderer
func (pr *PTYRenderer) Start() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.running {
		return
	}

	pr.running = true
	go pr.captureOutput()
}

// Stop stops the renderer
func (pr *PTYRenderer) Stop() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.running {
		return
	}

	pr.cancel()
	pr.running = false
}

// GetContent returns the captured content
func (pr *PTYRenderer) GetContent() string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return string(pr.buffer)
}

// ClearBuffer clears the content buffer
func (pr *PTYRenderer) ClearBuffer() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.buffer = pr.buffer[:0]
}

// captureOutput continuously captures output from PTY
func (pr *PTYRenderer) captureOutput() {
	for {
		select {
		case <-pr.ctx.Done():
			return
		default:
			data, err := pr.pty.ReadOutput()
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			if len(data) > 0 {
				pr.mu.Lock()
				pr.buffer = append(pr.buffer, data...)
				// Keep only last 4KB
				if len(pr.buffer) > 4096 {
					pr.buffer = pr.buffer[len(pr.buffer)-4096:]
				}
				pr.mu.Unlock()
			}

			time.Sleep(10 * time.Millisecond)
		}
	}
}

// PTYBridge converts PTY events to tea.Msg format
type PTYBridge struct {
	mu       sync.RWMutex
	pty      *PTYLayerImpl
	msgChan  chan tea.Msg
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPTYBridge creates a new PTY bridge
func NewPTYBridge(pty *PTYLayerImpl) *PTYBridge {
	ctx, cancel := context.WithCancel(context.Background())

	return &PTYBridge{
		pty:     pty,
		msgChan: make(chan tea.Msg, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the bridge
func (pb *PTYBridge) Start() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.running {
		return
	}

	pb.running = true
	go pb.monitorPTY()
}

// Stop stops the bridge
func (pb *PTYBridge) Stop() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if !pb.running {
		return
	}

	pb.cancel()
	pb.running = false
	close(pb.msgChan)
}

// Messages returns the message channel
func (pb *PTYBridge) Messages() <-chan tea.Msg {
	return pb.msgChan
}

// monitorPTY monitors PTY for events and converts them to tea.Msg
func (pb *PTYBridge) monitorPTY() {
	for {
		select {
		case <-pb.ctx.Done():
			return
		default:
			// Read PTY output
			data, err := pb.pty.ReadOutput()
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			if len(data) > 0 {
				// Convert output to appropriate tea.Msg
				msg := pb.convertOutputToMsg(data)
				if msg != nil {
					select {
					case pb.msgChan <- msg:
					case <-pb.ctx.Done():
						return
					default:
						// Channel full, drop message
					}
				}
			}

			time.Sleep(10 * time.Millisecond)
		}
	}
}

// convertOutputToMsg converts PTY output to tea.Msg
func (pb *PTYBridge) convertOutputToMsg(data []byte) tea.Msg {
	// This is a simplified implementation
	// In a real implementation, you'd parse the output and create appropriate messages

	// For now, just create a generic window size message when content changes
	if len(data) > 0 {
		size := pb.pty.GetSize()
		return tea.WindowSizeMsg{
			Width:  int(size.Cols),
			Height: int(size.Rows),
		}
	}

	return nil
}

// ptyRenderer implements tea.Renderer interface for PTY integration
type ptyRenderer struct {
	pty *PTYLayerImpl
}

// Render renders the model to the PTY
func (pr ptyRenderer) Render(model tea.Model) string {
	// For PTY integration, we don't actually render to a terminal
	// Instead, we capture the output for testing validation
	if pr.pty != nil {
		// This would normally render the model to string
		// For testing, we'll capture model state for validation
		return fmt.Sprintf("PTY rendered model: %T", model)
	}
	return ""
}

// Clear clears the PTY screen
func (pr ptyRenderer) Clear() {
	// PTY doesn't need explicit clearing in testing context
}

// Release releases PTY resources
func (pr ptyRenderer) Release() {
	// Resources are managed by PTY layer
}

// NewPTYRendererWithInput creates a PTY renderer with input handling
func NewPTYRendererWithInput(pty *PTYLayerImpl) ptyRenderer {
	return ptyRenderer{pty: pty}
}

// PTYInputHandler converts BubbleTea input to PTY-compatible format
type PTYInputHandler struct {
	pty *PTYLayerImpl
}

// NewPTYInputHandler creates a new PTY input handler
func NewPTYInputHandler(pty *PTYLayerImpl) *PTYInputHandler {
	return &PTYInputHandler{pty: pty}
}

// HandleInput processes BubbleTea input events
func (pih *PTYInputHandler) HandleInput(msg tea.Msg) error {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return pih.pty.SimulateKeyPress(msg)
	case tea.WindowSizeMsg:
		size := PTYSize{
			Rows: uint16(msg.Height),
			Cols: uint16(msg.Width),
		}
		return pih.pty.SetSize(size)
	default:
		// Ignore other message types for now
		return nil
	}
}

// CreatePTYBubbleTeaIntegration creates a complete PTY-BubbleTea integration
func CreatePTYBubbleTeaIntegration(model tea.Model, pty *PTYLayerImpl) (*PTYProgram, error) {
	if !pty.IsAllocated() {
		return nil, fmt.Errorf("PTY must be allocated before integration")
	}

	// Input handler is available for future integration
	_ = NewPTYInputHandler(pty)

	// Create custom program options that integrate with PTY
	options := []tea.ProgramOption{
		tea.WithoutSignalHandler(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}

	// Create the PTY program
	program := NewPTYProgram(model, pty, options...)

	// Start input handling
	go func() {
		for {
			// This would be connected to the actual BubbleTea event loop
			// For now, it's a placeholder for the integration pattern
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return program, nil
}