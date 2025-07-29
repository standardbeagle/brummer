package aicoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ClaudeProvider implements the AIProvider interface for Anthropic's Claude
type ClaudeProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClaudeProvider creates a new Claude AI provider
func NewClaudeProvider(apiKey, model string) *ClaudeProvider {
	if apiKey == "" {
		// Try to get from environment
		apiKey = os.Getenv("CLAUDE_API_KEY")
	}
	
	if model == "" {
		model = "claude-3-sonnet-20240229" // Default to Sonnet
	}
	
	return &ClaudeProvider{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		model:   model,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute, // Longer timeout for code generation
		},
	}
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// GetCapabilities returns Claude's capabilities
func (p *ClaudeProvider) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  200000, // Claude 3 supports up to 200k tokens
		MaxOutputTokens:   4096,
		SupportedModels: []string{
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
			"claude-3-5-sonnet-20241022",
		},
	}
}

// GenerateCode generates code using Claude API
func (p *ClaudeProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Claude API key not configured")
	}
	
	// Build the system prompt
	systemPrompt := "You are an expert software engineer helping to implement code. " +
		"Generate clean, well-documented code following best practices. " +
		"Include error handling and tests where appropriate."
	
	if options.SystemPrompt != "" {
		systemPrompt = options.SystemPrompt
	}
	
	// Build the request
	requestBody := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"system":     systemPrompt,
		"max_tokens": options.MaxTokens,
	}
	
	if options.Temperature > 0 {
		requestBody["temperature"] = options.Temperature
	}
	
	if len(options.StopSequences) > 0 {
		requestBody["stop_sequences"] = options.StopSequences
	}
	
	// Marshal request
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(map[string]interface{}); ok {
				return nil, fmt.Errorf("Claude API error: %v", errorMsg["message"])
			}
		}
		return nil, fmt.Errorf("Claude API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Extract code from response
	var code strings.Builder
	for _, content := range response.Content {
		if content.Type == "text" {
			code.WriteString(content.Text)
		}
	}
	
	return &GenerateResult{
		Code:         code.String(),
		Summary:      fmt.Sprintf("Generated with %s", p.model),
		TokensUsed:   response.Usage.InputTokens + response.Usage.OutputTokens,
		Model:        p.model,
		FinishReason: "complete",
	}, nil
}

// StreamGenerate generates code with streaming support
func (p *ClaudeProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	ch := make(chan GenerateUpdate)
	
	go func() {
		defer close(ch)
		
		// For now, use non-streaming version
		// Claude API supports streaming, but we'll implement it later
		result, err := p.GenerateCode(ctx, prompt, options)
		if err != nil {
			ch <- GenerateUpdate{Error: err}
			return
		}
		
		// Send as single chunk
		ch <- GenerateUpdate{
			Content:      result.Code,
			TokensUsed:   result.TokensUsed,
			FinishReason: result.FinishReason,
		}
	}()
	
	return ch, nil
}

// ValidateConfig validates the provider configuration
func (p *ClaudeProvider) ValidateConfig(config ProviderConfig) error {
	if config.APIKeyEnv != "" {
		apiKey := os.Getenv(config.APIKeyEnv)
		if apiKey == "" {
			return fmt.Errorf("API key environment variable %s is not set", config.APIKeyEnv)
		}
	}
	
	// Validate model
	caps := p.GetCapabilities()
	validModel := false
	for _, model := range caps.SupportedModels {
		if config.Model == model {
			validModel = true
			break
		}
	}
	
	if config.Model != "" && !validModel {
		return fmt.Errorf("unsupported model: %s", config.Model)
	}
	
	return nil
}