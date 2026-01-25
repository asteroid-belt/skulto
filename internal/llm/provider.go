// Package llm provides interfaces and implementations for LLM providers.
package llm

import (
	"context"
	"fmt"

	"github.com/asteroid-belt/skulto/internal/config"
)

// Provider defines the interface for LLM providers.
type Provider interface {
	// Chat sends messages and returns a streaming response.
	Chat(ctx context.Context, messages []Message, opts ChatOptions) (*StreamReader, error)

	// ChatSync sends messages and waits for complete response.
	ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error)

	// Name returns the provider name (e.g., "anthropic", "openai").
	Name() string

	// Models returns available model IDs for this provider.
	Models() []string

	// DefaultModel returns the default model for this provider.
	DefaultModel() string
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // Message content
}

// NewSystemMessage creates a system message.
func NewSystemMessage(content string) Message {
	return Message{Role: "system", Content: content}
}

// NewUserMessage creates a user message.
func NewUserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// NewAssistantMessage creates an assistant message.
func NewAssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// ChatOptions configures a chat request.
type ChatOptions struct {
	Model       string  // Model to use (empty = provider default)
	MaxTokens   int     // Maximum tokens in response
	Temperature float64 // Sampling temperature (0-1)
	Stream      bool    // Enable streaming response
}

// Response represents a complete chat response.
type Response struct {
	Content      string // Response content
	Model        string // Model used
	FinishReason string // Why generation stopped
	Usage        Usage  // Token usage
}

// Usage tracks token usage for a request.
type Usage struct {
	PromptTokens     int // Tokens in prompt
	CompletionTokens int // Tokens in completion
	TotalTokens      int // Total tokens
}

// ProviderType represents supported LLM providers.
type ProviderType string

const (
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderOpenAI     ProviderType = "openai"
	ProviderOpenRouter ProviderType = "openrouter"
)

// NewProvider creates a provider based on configuration.
// It auto-detects the provider if not explicitly set.
func NewProvider(cfg config.LLMConfig) (Provider, error) {
	return NewProviderWithOverrides(cfg, "", "")
}

// NewProviderWithOverrides creates a provider with optional overrides.
func NewProviderWithOverrides(cfg config.LLMConfig, providerOverride, modelOverride string) (Provider, error) {
	providerName := providerOverride
	if providerName == "" {
		providerName = cfg.DefaultProvider
	}

	// Auto-detect provider if not specified
	if providerName == "" {
		providerName = detectProvider(cfg)
	}

	if providerName == "" {
		return nil, fmt.Errorf("no LLM provider configured: set ANTHROPIC_API_KEY, OPENAI_API_KEY, or OPENROUTER_API_KEY")
	}

	model := modelOverride
	if model == "" {
		model = cfg.DefaultModel
	}

	switch ProviderType(providerName) {
	case ProviderAnthropic:
		if cfg.AnthropicAPIKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return NewAnthropicProvider(cfg.AnthropicAPIKey, model)

	case ProviderOpenAI:
		if cfg.OpenAIAPIKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return NewOpenAIProvider(cfg.OpenAIAPIKey, model)

	case ProviderOpenRouter:
		if cfg.OpenRouterAPIKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY not set")
		}
		return NewOpenRouterProvider(cfg.OpenRouterAPIKey, model)

	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: anthropic, openai, openrouter)", providerName)
	}
}

// detectProvider determines which provider to use based on available API keys.
// Priority: Anthropic > OpenAI > OpenRouter
func detectProvider(cfg config.LLMConfig) string {
	if cfg.AnthropicAPIKey != "" {
		return string(ProviderAnthropic)
	}
	if cfg.OpenAIAPIKey != "" {
		return string(ProviderOpenAI)
	}
	if cfg.OpenRouterAPIKey != "" {
		return string(ProviderOpenRouter)
	}
	return ""
}

// IsConfigured returns true if any LLM provider is configured.
func IsConfigured(cfg config.LLMConfig) bool {
	return cfg.AnthropicAPIKey != "" || cfg.OpenAIAPIKey != "" || cfg.OpenRouterAPIKey != ""
}

// NewAnthropicProvider creates an Anthropic provider.
func NewAnthropicProvider(apiKey, model string) (Provider, error) {
	return NewAnthropicProviderImpl(apiKey, model)
}

// placeholder implementations - will be replaced in Phase 1B/1C

// NewOpenAIProvider creates an OpenAI provider.
func NewOpenAIProvider(apiKey, model string) (Provider, error) {
	return newOpenAIProvider(apiKey, model)
}

// NewOpenRouterProvider creates an OpenRouter provider.
func NewOpenRouterProvider(apiKey, model string) (Provider, error) {
	return newOpenRouterProvider(apiKey, model)
}
