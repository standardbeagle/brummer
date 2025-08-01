package aicoder

import (
	"context"
	"fmt"
)

// MockProvider implements a mock AI provider for testing
type MockProvider struct {
	name         string
	capabilities ProviderCapabilities
}

// NewMockProvider creates a new mock provider
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name: name,
		capabilities: ProviderCapabilities{
			SupportsStreaming: true,
			MaxContextTokens:  100000,
			MaxOutputTokens:   4096,
			SupportedModels:   []string{"mock-model"},
		},
	}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	// Mock implementation
	return &GenerateResult{
		Code:         "// Mock generated code\nfunc main() {\n\tfmt.Println(\"Hello from AI\")\n}",
		Summary:      "Generated mock code based on prompt",
		TokensUsed:   100,
		Model:        options.Model,
		FinishReason: "complete",
	}, nil
}

func (m *MockProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	ch := make(chan GenerateUpdate)

	go func() {
		defer close(ch)

		// Simulate streaming response
		updates := []string{
			"// Mock generated code\n",
			"func main() {\n",
			"\tfmt.Println(\"Hello from AI\")\n",
			"}",
		}

		for _, content := range updates {
			select {
			case <-ctx.Done():
				ch <- GenerateUpdate{Error: ctx.Err()}
				return
			case ch <- GenerateUpdate{Content: content, TokensUsed: 25}:
			}
		}

		ch <- GenerateUpdate{FinishReason: "complete", TokensUsed: 100}
	}()

	return ch, nil
}

func (m *MockProvider) ValidateConfig(config ProviderConfig) error {
	if config.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func (m *MockProvider) GetCapabilities() ProviderCapabilities {
	return m.capabilities
}
