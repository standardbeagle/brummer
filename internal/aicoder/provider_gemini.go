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

// GeminiProvider implements the AIProvider interface for Google's Gemini models
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewGeminiProvider creates a new Gemini AI provider
func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if apiKey == "" {
		// Try to get from environment
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if model == "" {
		model = "gemini-1.5-pro" // Default to Gemini 1.5 Pro
	}

	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		model:   model,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// GetCapabilities returns Gemini's capabilities
func (p *GeminiProvider) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  1000000, // Gemini 1.5 supports up to 1M tokens
		MaxOutputTokens:   8192,
		SupportedModels: []string{
			"gemini-1.5-pro",
			"gemini-1.5-flash",
			"gemini-1.0-pro",
			"gemini-pro",
		},
	}
}

// GenerateCode generates code using Gemini API
func (p *GeminiProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured")
	}

	// Build the system prompt
	systemPrompt := "You are an expert software engineer helping to implement code. " +
		"Generate clean, well-documented code following best practices. " +
		"Include error handling and tests where appropriate."

	if options.SystemPrompt != "" {
		systemPrompt = options.SystemPrompt
	}

	// Build the full prompt with system context
	fullPrompt := systemPrompt + "\n\n" + prompt

	// Build the request
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": fullPrompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": options.MaxTokens,
			"temperature":     options.Temperature,
		},
	}

	if len(options.StopSequences) > 0 {
		requestBody["generationConfig"].(map[string]interface{})["stopSequences"] = options.StopSequences
	}

	// Marshal request
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

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
				return nil, fmt.Errorf("Gemini API error: %v", errorObj["message"])
			}
		}
		return nil, fmt.Errorf("Gemini API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	// Extract text from parts
	var code string
	for _, part := range response.Candidates[0].Content.Parts {
		code += part.Text
	}

	return &GenerateResult{
		Code:         code,
		Summary:      fmt.Sprintf("Generated with %s", p.model),
		TokensUsed:   response.UsageMetadata.TotalTokenCount,
		Model:        p.model,
		FinishReason: response.Candidates[0].FinishReason,
	}, nil
}

// StreamGenerate generates code with streaming support
func (p *GeminiProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	ch := make(chan GenerateUpdate)

	go func() {
		defer close(ch)

		// For now, use non-streaming version
		// Gemini supports streaming, but we'll implement it later
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
func (p *GeminiProvider) ValidateConfig(config ProviderConfig) error {
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
