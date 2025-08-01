package aicoder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// TerminalProvider implements the AIProvider interface for local terminal/shell execution
type TerminalProvider struct {
	shell      string
	workingDir string
	env        []string
	mu         sync.Mutex
}

// NewTerminalProvider creates a new Terminal provider
func NewTerminalProvider(shell string) *TerminalProvider {
	if shell == "" {
		// Detect default shell
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash" // Default to bash
		}
	}

	return &TerminalProvider{
		shell:      shell,
		workingDir: ".",
		env:        os.Environ(),
	}
}

// Name returns the provider name
func (p *TerminalProvider) Name() string {
	return "terminal"
}

// GetCapabilities returns Terminal's capabilities
func (p *TerminalProvider) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  1000000, // No real limit for terminal
		MaxOutputTokens:   1000000, // No real limit for terminal
		SupportedModels: []string{
			"bash",
			"sh",
			"zsh",
			"fish",
			"python",
			"node",
			"ruby",
		},
	}
}

// GenerateCode executes commands in the terminal and returns the output
func (p *TerminalProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// The prompt is treated as a command or script to execute
	// First, let's see if it's a multi-line script or a single command
	lines := strings.Split(prompt, "\n")

	var cmd *exec.Cmd
	var output strings.Builder
	var summary string

	// Determine shell to use (options.Model can override default)
	shell := p.shell
	if options.Model != "" && options.Model != "terminal" {
		shell = options.Model
	}

	// If it's a single line, execute directly
	if len(lines) == 1 && !strings.Contains(prompt, ";") && !strings.Contains(prompt, "&&") {
		cmd = exec.CommandContext(ctx, shell, "-c", prompt)
		summary = fmt.Sprintf("Executed command: %s", prompt)
	} else {
		// Multi-line script - write to temp file and execute
		tmpFile, err := os.CreateTemp("", "aicoder-script-*.sh")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp script: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		// Write script content
		if _, err := tmpFile.WriteString(prompt); err != nil {
			return nil, fmt.Errorf("failed to write script: %w", err)
		}
		tmpFile.Close()

		// Make executable
		os.Chmod(tmpFile.Name(), 0755)

		// Execute script
		cmd = exec.CommandContext(ctx, shell, tmpFile.Name())
		summary = "Executed multi-line script"
	}

	// Set working directory
	if p.workingDir != "" && p.workingDir != "." {
		cmd.Dir = p.workingDir
	}

	// Set environment
	cmd.Env = p.env

	// Capture both stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Read output
	var wg sync.WaitGroup
	wg.Add(2)

	// Read stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			output.WriteString(scanner.Text() + "\n")
		}
	}()

	// Read stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			output.WriteString("[stderr] " + scanner.Text() + "\n")
		}
	}()

	// Wait for command to complete
	wg.Wait()
	err = cmd.Wait()

	// Build result
	result := &GenerateResult{
		Code:         output.String(),
		Summary:      summary,
		TokensUsed:   len(output.String()), // Approximate by character count
		Model:        shell,
		FinishReason: "complete",
	}

	// Include exit code in summary if command failed
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Summary = fmt.Sprintf("%s (exit code: %d)", summary, exitErr.ExitCode())
		} else {
			result.Summary = fmt.Sprintf("%s (error: %v)", summary, err)
		}
	}

	return result, nil
}

// StreamGenerate executes commands with real-time streaming output
func (p *TerminalProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	ch := make(chan GenerateUpdate, 100)

	go func() {
		defer close(ch)

		p.mu.Lock()
		defer p.mu.Unlock()

		// Determine shell to use
		shell := p.shell
		if options.Model != "" && options.Model != "terminal" {
			shell = options.Model
		}

		// Create command
		var cmd *exec.Cmd
		lines := strings.Split(prompt, "\n")

		if len(lines) == 1 && !strings.Contains(prompt, ";") && !strings.Contains(prompt, "&&") {
			cmd = exec.CommandContext(ctx, shell, "-c", prompt)
		} else {
			// Multi-line script
			tmpFile, err := os.CreateTemp("", "aicoder-script-*.sh")
			if err != nil {
				ch <- GenerateUpdate{Error: fmt.Errorf("failed to create temp script: %w", err)}
				return
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(prompt); err != nil {
				ch <- GenerateUpdate{Error: fmt.Errorf("failed to write script: %w", err)}
				return
			}
			tmpFile.Close()
			os.Chmod(tmpFile.Name(), 0755)

			cmd = exec.CommandContext(ctx, shell, tmpFile.Name())
		}

		// Set working directory and environment
		if p.workingDir != "" && p.workingDir != "." {
			cmd.Dir = p.workingDir
		}
		cmd.Env = p.env

		// Create pipes
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ch <- GenerateUpdate{Error: fmt.Errorf("failed to create stdout pipe: %w", err)}
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			ch <- GenerateUpdate{Error: fmt.Errorf("failed to create stderr pipe: %w", err)}
			return
		}

		// Start command
		if err := cmd.Start(); err != nil {
			ch <- GenerateUpdate{Error: fmt.Errorf("failed to start command: %w", err)}
			return
		}

		// Stream output
		done := make(chan bool, 2)

		// Stream stdout
		go func() {
			reader := bufio.NewReader(stdout)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						select {
						case ch <- GenerateUpdate{Content: fmt.Sprintf("[error reading stdout: %v]\n", err)}:
						case <-ctx.Done():
							return
						}
					}
					break
				}
				select {
				case ch <- GenerateUpdate{Content: line, TokensUsed: len(line)}:
				case <-ctx.Done():
					return
				}
			}
			done <- true
		}()

		// Stream stderr
		go func() {
			reader := bufio.NewReader(stderr)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						select {
						case ch <- GenerateUpdate{Content: fmt.Sprintf("[error reading stderr: %v]\n", err)}:
						case <-ctx.Done():
							return
						}
					}
					break
				}
				select {
				case ch <- GenerateUpdate{Content: "[stderr] " + line, TokensUsed: len(line)}:
				case <-ctx.Done():
					return
				}
			}
			done <- true
		}()

		// Wait for streams to finish
		<-done
		<-done

		// Wait for command to complete
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				ch <- GenerateUpdate{
					Content:      fmt.Sprintf("\n[Process exited with code %d]\n", exitErr.ExitCode()),
					FinishReason: "error",
				}
			} else {
				ch <- GenerateUpdate{
					Content:      fmt.Sprintf("\n[Process error: %v]\n", err),
					FinishReason: "error",
				}
			}
		} else {
			ch <- GenerateUpdate{
				Content:      "\n[Process completed successfully]\n",
				FinishReason: "complete",
			}
		}
	}()

	return ch, nil
}

// ValidateConfig validates the provider configuration
func (p *TerminalProvider) ValidateConfig(config ProviderConfig) error {
	// Terminal provider doesn't need API keys
	// Just validate the shell/model if specified
	if config.Model != "" {
		// Check if the shell exists
		if _, err := exec.LookPath(config.Model); err != nil {
			// It might be a built-in shell command, so don't fail hard
			// Just warn
			return nil
		}
	}

	return nil
}

// SetWorkingDirectory sets the working directory for terminal commands
func (p *TerminalProvider) SetWorkingDirectory(dir string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workingDir = dir
}

// SetEnvironment sets additional environment variables
func (p *TerminalProvider) SetEnvironment(env map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Start with current environment
	p.env = os.Environ()

	// Add/override with provided variables
	for key, value := range env {
		p.env = append(p.env, fmt.Sprintf("%s=%s", key, value))
	}
}
