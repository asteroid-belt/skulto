package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingConfigDefaults(t *testing.T) {
	cfg := DefaultVectorConfig()

	assert.Equal(t, "text-embedding-3-small", cfg.Model)
	assert.Equal(t, float32(0.6), cfg.MinSimilarity)
	assert.False(t, cfg.Enabled) // Disabled by default
}

func TestEmbeddingConfigFromEnv(t *testing.T) {
	// Save and restore original env
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	_ = os.Setenv("OPENAI_API_KEY", "test-key-123")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "test-key-123", cfg.Embedding.APIKey)
	assert.True(t, cfg.Embedding.Enabled) // Enabled when API key is set
}

func TestLLMConfigDefaults(t *testing.T) {
	cfg := DefaultLLMConfig()

	assert.Empty(t, cfg.AnthropicAPIKey)
	assert.Empty(t, cfg.OpenAIAPIKey)
	assert.Empty(t, cfg.OpenRouterAPIKey)
	assert.Empty(t, cfg.DefaultProvider) // Auto-detect
	assert.Empty(t, cfg.DefaultModel)    // Provider-specific defaults
}

func TestLLMConfigFromEnv(t *testing.T) {
	// Save and restore original env
	originalAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	originalOpenRouterKey := os.Getenv("OPENROUTER_API_KEY")
	defer func() {
		if originalAnthropicKey != "" {
			_ = os.Setenv("ANTHROPIC_API_KEY", originalAnthropicKey)
		} else {
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
		}
		if originalOpenAIKey != "" {
			_ = os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
		if originalOpenRouterKey != "" {
			_ = os.Setenv("OPENROUTER_API_KEY", originalOpenRouterKey)
		} else {
			_ = os.Unsetenv("OPENROUTER_API_KEY")
		}
	}()

	_ = os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	_ = os.Setenv("OPENAI_API_KEY", "sk-openai-test")
	_ = os.Setenv("OPENROUTER_API_KEY", "sk-or-test")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "sk-ant-test", cfg.LLM.AnthropicAPIKey)
	assert.Equal(t, "sk-openai-test", cfg.LLM.OpenAIAPIKey)
	assert.Equal(t, "sk-or-test", cfg.LLM.OpenRouterAPIKey)
}

func TestDefaultConfigIncludesLLM(t *testing.T) {
	cfg := DefaultConfig()

	// Verify LLM config is included
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.LLM.DefaultProvider)
}

func TestGitHubTokenFromEnv(t *testing.T) {
	// Save and restore original env
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	_ = os.Setenv("GITHUB_TOKEN", "ghp_test123")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "ghp_test123", cfg.GitHub.Token)
}
