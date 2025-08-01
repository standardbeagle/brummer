package aicoder

import (
	"context"
	"fmt"
)

// AIProvider represents an interface for AI code generation providers
type AIProvider interface {
	// Name returns the provider name (e.g., "claude", "openai", "local")
	Name() string

	// GenerateCode generates code based on the prompt and options
	GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error)

	// StreamGenerate generates code in a streaming fashion
	StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig(config ProviderConfig) error

	// GetCapabilities returns the provider's capabilities
	GetCapabilities() ProviderCapabilities
}

// GenerateOptions contains options for code generation
type GenerateOptions struct {
	Model            string
	MaxTokens        int
	Temperature      float64
	WorkspaceContext []string // Files to include as context
	SystemPrompt     string
	StopSequences    []string
}

// GenerateResult represents the result of a code generation request
type GenerateResult struct {
	Code         string
	Summary      string
	TokensUsed   int
	Model        string
	FinishReason string
}

// GenerateUpdate represents a streaming update during code generation
type GenerateUpdate struct {
	Content      string
	TokensUsed   int
	FinishReason string
	Error        error
}

// ProviderCapabilities describes what a provider can do
type ProviderCapabilities struct {
	SupportsStreaming bool
	MaxContextTokens  int
	MaxOutputTokens   int
	SupportedModels   []string
}

// ProviderRegistry manages available AI providers
type ProviderRegistry struct {
	providers map[string]AIProvider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]AIProvider),
	}
}

// Register registers a new AI provider
func (r *ProviderRegistry) Register(name string, provider AIProvider) error {
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}
	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (AIProvider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// List returns all registered provider names
func (r *ProviderRegistry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
