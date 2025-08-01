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

// CLIToolProvider implements the AIProvider interface for generic CLI tools
type CLIToolProvider struct {
	name        string
	command     string
	flagMapping map[string]string // maps Brummer config keys to CLI flags
	args        []string          // base arguments for the command
	workingDir  string
	env         []string
	mu          sync.Mutex
}

// Note: CLIToolConfig is now defined in types.go

// NewCLIToolProvider creates a new CLI tool provider
func NewCLIToolProvider(name string, config CLIToolConfig) *CLIToolProvider {
	if config.Command == "" {
		config.Command = name // default to provider name
	}

	if config.FlagMapping == nil {
		config.FlagMapping = getDefaultFlagMapping(name)
	}

	return &CLIToolProvider{
		name:        name,
		command:     config.Command,
		flagMapping: config.FlagMapping,
		args:        config.BaseArgs,
		workingDir:  config.WorkingDir,
		env:         buildEnvironment(config.Environment),
	}
}

// Name returns the provider name
func (p *CLIToolProvider) Name() string {
	return p.name
}

// GetCapabilities returns CLI tool capabilities
func (p *CLIToolProvider) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  1000000, // No real limit for CLI tools
		MaxOutputTokens:   1000000, // No real limit for CLI tools
		SupportedModels:   getSupportedModels(p.name),
	}
}

// GenerateCode executes the CLI tool and returns the output
func (p *CLIToolProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Build command arguments
	args := p.buildArgs(prompt, options)

	// Create command
	cmd := exec.CommandContext(ctx, p.command, args...)

	// Set working directory
	if p.workingDir != "" {
		cmd.Dir = p.workingDir
	} else if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}

	// Set environment
	cmd.Env = p.env

	// Capture output
	var output strings.Builder
	var errorOutput strings.Builder

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
		return nil, fmt.Errorf("failed to start %s: %w", p.command, err)
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
			errorOutput.WriteString(scanner.Text() + "\n")
		}
	}()

	// Wait for command to complete
	wg.Wait()
	err = cmd.Wait()

	// Build result
	result := &GenerateResult{
		Code:         output.String(),
		Summary:      fmt.Sprintf("Executed %s", p.command),
		TokensUsed:   len(output.String()), // Approximate by character count
		Model:        options.Model,
		FinishReason: "complete",
	}

	// Include stderr in summary if there were errors
	if errorOutput.Len() > 0 {
		result.Summary += fmt.Sprintf(" (stderr: %s)", errorOutput.String())
	}

	// Include exit code in summary if command failed
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Summary += fmt.Sprintf(" (exit code: %d)", exitErr.ExitCode())
			result.FinishReason = "error"
		} else {
			result.Summary += fmt.Sprintf(" (error: %v)", err)
			result.FinishReason = "error"
		}
	}

	return result, nil
}

// StreamGenerate executes CLI tool with real-time streaming output
func (p *CLIToolProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	ch := make(chan GenerateUpdate, 100)

	go func() {
		defer close(ch)

		p.mu.Lock()
		defer p.mu.Unlock()

		// Build command arguments
		args := p.buildArgs(prompt, options)

		// Create command
		cmd := exec.CommandContext(ctx, p.command, args...)

		// Set working directory
		if p.workingDir != "" {
			cmd.Dir = p.workingDir
		} else if wd, err := os.Getwd(); err == nil {
			cmd.Dir = wd
		}

		// Set environment
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
			ch <- GenerateUpdate{Error: fmt.Errorf("failed to start %s: %w", p.command, err)}
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
					Content:      fmt.Sprintf("\n[%s exited with code %d]\n", p.command, exitErr.ExitCode()),
					FinishReason: "error",
				}
			} else {
				ch <- GenerateUpdate{
					Content:      fmt.Sprintf("\n[%s error: %v]\n", p.command, err),
					FinishReason: "error",
				}
			}
		} else {
			ch <- GenerateUpdate{
				Content:      fmt.Sprintf("\n[%s completed successfully]\n", p.command),
				FinishReason: "complete",
			}
		}
	}()

	return ch, nil
}

// ValidateConfig validates the CLI tool configuration
func (p *CLIToolProvider) ValidateConfig(config ProviderConfig) error {
	// Check if the command exists
	if _, err := exec.LookPath(p.command); err != nil {
		return fmt.Errorf("command '%s' not found in PATH: %w", p.command, err)
	}

	return nil
}

// buildArgs constructs the command arguments based on the prompt and options
func (p *CLIToolProvider) buildArgs(prompt string, options GenerateOptions) []string {
	args := make([]string, len(p.args))
	copy(args, p.args)

	// Map Brummer options to CLI flags
	if options.Model != "" {
		if flag, exists := p.flagMapping["model"]; exists {
			args = append(args, flag, options.Model)
		}
	}

	if options.MaxTokens > 0 {
		if flag, exists := p.flagMapping["max_tokens"]; exists {
			args = append(args, flag, fmt.Sprintf("%d", options.MaxTokens))
		}
	}

	if options.Temperature > 0 {
		if flag, exists := p.flagMapping["temperature"]; exists {
			args = append(args, flag, fmt.Sprintf("%.2f", options.Temperature))
		}
	}

	// Add workspace context files
	for _, file := range options.WorkspaceContext {
		if flag, exists := p.flagMapping["context_file"]; exists {
			args = append(args, flag, file)
		} else {
			// Default to just adding the file as an argument
			args = append(args, file)
		}
	}

	// Add the prompt/message
	if flag, exists := p.flagMapping["message"]; exists {
		args = append(args, flag, prompt)
	} else {
		// Some tools take the message as a positional argument
		args = append(args, prompt)
	}

	return args
}

// getDefaultFlagMapping returns default flag mappings for known CLI tools
func getDefaultFlagMapping(toolName string) map[string]string {
	switch strings.ToLower(toolName) {
	case "aider":
		return map[string]string{
			"model":       "--model",
			"message":     "--message",
			"max_tokens":  "--max-tokens",  // if supported
			"temperature": "--temperature", // if supported
		}
	case "cursor":
		return map[string]string{
			"model":   "--model",
			"message": "--prompt",
		}
	case "codeium":
		return map[string]string{
			"model":   "--model",
			"message": "--prompt",
		}
	case "github-copilot":
		return map[string]string{
			"message": "--prompt",
		}
	default:
		return map[string]string{
			"model":   "--model",
			"message": "--prompt",
		}
	}
}

// getSupportedModels returns supported models for known CLI tools
func getSupportedModels(toolName string) []string {
	switch strings.ToLower(toolName) {
	case "aider":
		return []string{
			"gpt-4-turbo-preview",
			"gpt-4",
			"gpt-3.5-turbo",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		}
	case "cursor":
		return []string{
			"gpt-4",
			"gpt-3.5-turbo",
			"claude-3-sonnet",
		}
	default:
		return []string{"default"}
	}
}

// buildEnvironment builds the environment variables for the command
func buildEnvironment(envVars map[string]string) []string {
	env := os.Environ()

	for key, value := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}
