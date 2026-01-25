package llm

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMessageConstructors(t *testing.T) {
	t.Run("NewSystemMessage", func(t *testing.T) {
		msg := NewSystemMessage("You are a helpful assistant")
		assert.Equal(t, "system", msg.Role)
		assert.Equal(t, "You are a helpful assistant", msg.Content)
	})

	t.Run("NewUserMessage", func(t *testing.T) {
		msg := NewUserMessage("Hello!")
		assert.Equal(t, "user", msg.Role)
		assert.Equal(t, "Hello!", msg.Content)
	})

	t.Run("NewAssistantMessage", func(t *testing.T) {
		msg := NewAssistantMessage("Hi there!")
		assert.Equal(t, "assistant", msg.Role)
		assert.Equal(t, "Hi there!", msg.Content)
	})
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.LLMConfig
		expected string
	}{
		{
			name:     "no keys",
			cfg:      config.LLMConfig{},
			expected: "",
		},
		{
			name: "anthropic only",
			cfg: config.LLMConfig{
				AnthropicAPIKey: "sk-ant-xxx",
			},
			expected: "anthropic",
		},
		{
			name: "openai only",
			cfg: config.LLMConfig{
				OpenAIAPIKey: "sk-xxx",
			},
			expected: "openai",
		},
		{
			name: "openrouter only",
			cfg: config.LLMConfig{
				OpenRouterAPIKey: "sk-or-xxx",
			},
			expected: "openrouter",
		},
		{
			name: "anthropic priority over openai",
			cfg: config.LLMConfig{
				AnthropicAPIKey: "sk-ant-xxx",
				OpenAIAPIKey:    "sk-xxx",
			},
			expected: "anthropic",
		},
		{
			name: "openai priority over openrouter",
			cfg: config.LLMConfig{
				OpenAIAPIKey:     "sk-xxx",
				OpenRouterAPIKey: "sk-or-xxx",
			},
			expected: "openai",
		},
		{
			name: "all keys - anthropic wins",
			cfg: config.LLMConfig{
				AnthropicAPIKey:  "sk-ant-xxx",
				OpenAIAPIKey:     "sk-xxx",
				OpenRouterAPIKey: "sk-or-xxx",
			},
			expected: "anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectProvider(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.LLMConfig
		expected bool
	}{
		{
			name:     "no keys",
			cfg:      config.LLMConfig{},
			expected: false,
		},
		{
			name: "anthropic key",
			cfg: config.LLMConfig{
				AnthropicAPIKey: "sk-ant-xxx",
			},
			expected: true,
		},
		{
			name: "openai key",
			cfg: config.LLMConfig{
				OpenAIAPIKey: "sk-xxx",
			},
			expected: true,
		},
		{
			name: "openrouter key",
			cfg: config.LLMConfig{
				OpenRouterAPIKey: "sk-or-xxx",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConfigured(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewProvider_NoConfig(t *testing.T) {
	cfg := config.LLMConfig{}
	_, err := NewProvider(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LLM provider configured")
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	cfg := config.LLMConfig{
		DefaultProvider: "unknown",
	}
	_, err := NewProvider(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestNewProviderWithOverrides(t *testing.T) {
	t.Run("provider override to openai", func(t *testing.T) {
		cfg := config.LLMConfig{
			AnthropicAPIKey: "sk-ant-xxx",
			OpenAIAPIKey:    "sk-xxx",
		}
		// Even though anthropic has priority, we override to openai
		provider, err := NewProviderWithOverrides(cfg, "openai", "")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Name())
	})

	t.Run("provider override to openrouter", func(t *testing.T) {
		cfg := config.LLMConfig{
			AnthropicAPIKey:  "sk-ant-xxx",
			OpenRouterAPIKey: "sk-or-xxx",
		}
		// Override to openrouter
		provider, err := NewProviderWithOverrides(cfg, "openrouter", "")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openrouter", provider.Name())
	})

	t.Run("missing key for override", func(t *testing.T) {
		cfg := config.LLMConfig{
			AnthropicAPIKey: "sk-ant-xxx",
		}
		// Override to openai but no openai key
		_, err := NewProviderWithOverrides(cfg, "openai", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OPENAI_API_KEY not set")
	})

	t.Run("missing openrouter key for override", func(t *testing.T) {
		cfg := config.LLMConfig{
			AnthropicAPIKey: "sk-ant-xxx",
		}
		// Override to openrouter but no openrouter key
		_, err := NewProviderWithOverrides(cfg, "openrouter", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OPENROUTER_API_KEY not set")
	})
}

func TestProviderTypes(t *testing.T) {
	assert.Equal(t, ProviderType("anthropic"), ProviderAnthropic)
	assert.Equal(t, ProviderType("openai"), ProviderOpenAI)
	assert.Equal(t, ProviderType("openrouter"), ProviderOpenRouter)
}

func TestNewProvider_Anthropic(t *testing.T) {
	cfg := config.LLMConfig{
		AnthropicAPIKey: "sk-ant-test-key",
	}
	provider, err := NewProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "anthropic", provider.Name())
	assert.Equal(t, DefaultAnthropicModel, provider.DefaultModel())
	assert.Contains(t, provider.Models(), "claude-3-haiku-20240307")
}

func TestNewProvider_AnthropicWithModel(t *testing.T) {
	cfg := config.LLMConfig{
		AnthropicAPIKey: "sk-ant-test-key",
		DefaultModel:    "claude-3-5-sonnet-20241022",
	}
	provider, err := NewProvider(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "anthropic", provider.Name())
}
