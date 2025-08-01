package aicoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAIProvider implements the AIProvider interface for OpenAI's GPT models
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if apiKey == "" {
		// Try to get from environment
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if model == "" {
		model = "gpt-4-turbo-preview" // Default to GPT-4 Turbo
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		model:   model,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GetCapabilities returns OpenAI's capabilities
func (p *OpenAIProvider) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  128000, // GPT-4 Turbo supports 128k tokens
		MaxOutputTokens:   4096,
		SupportedModels: []string{
			"gpt-4-turbo-preview",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-4-32k",
			"gpt-3.5-turbo",
			"gpt-3.5-turbo-16k",
		},
	}
}

// GenerateCode generates code using OpenAI API
func (p *OpenAIProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
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
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": options.MaxTokens,
	}

	if options.Temperature > 0 {
		requestBody["temperature"] = options.Temperature
	}

	if len(options.StopSequences) > 0 {
		requestBody["stop"] = options.StopSequences
	}

	// Marshal request
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
			if errorObj, ok := errorResp["error"].(map[string]interface{}); ok {
				return nil, fmt.Errorf("OpenAI API error: %v", errorObj["message"])
			}
		}
		return nil, fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return &GenerateResult{
		Code:         response.Choices[0].Message.Content,
		Summary:      fmt.Sprintf("Generated with %s", response.Model),
		TokensUsed:   response.Usage.TotalTokens,
		Model:        response.Model,
		FinishReason: response.Choices[0].FinishReason,
	}, nil
}

// StreamGenerate generates code with streaming support
func (p *OpenAIProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	ch := make(chan GenerateUpdate)

	go func() {
		defer close(ch)

		// For now, use non-streaming version
		// OpenAI supports streaming, but we'll implement it later
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
func (p *OpenAIProvider) ValidateConfig(config ProviderConfig) error {
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
