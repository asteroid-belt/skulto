// Package config handles application configuration management.
package config

import (
	"os"
	"path/filepath"
)

// Config holds all application configuration.
type Config struct {
	// Base directory for all Skulto data (~/.skulto)
	BaseDir string

	// GitHub API settings
	GitHub GitHubConfig

	// Embedding/Vector Store settings
	Embedding VectorConfig

	// LLM settings for skill builder
	LLM LLMConfig
}

// LLMConfig holds LLM provider configuration for skill building.
type LLMConfig struct {
	// API keys for different providers
	AnthropicAPIKey  string
	OpenAIAPIKey     string
	OpenRouterAPIKey string

	// Default provider: "anthropic", "openai", "openrouter" (auto-detected if empty)
	DefaultProvider string
	// Default model (provider-specific, uses sensible default if empty)
	DefaultModel string
}

// DefaultLLMConfig returns sensible defaults for LLM configuration.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		// API keys read from env vars in Load()
		DefaultProvider: "", // Auto-detect based on available keys
		DefaultModel:    "", // Provider-specific defaults
	}
}

// VectorConfig holds vector store configuration.
type VectorConfig struct {
	// OpenAI API key for embeddings (OPENAI_API_KEY env var)
	APIKey string
	// Model for embeddings (default: text-embedding-3-small)
	Model string
	// DataDir for chromem-go persistence (default: ~/.skulto/vectors)
	DataDir string
	// MinSimilarity threshold for search (default: 0.6)
	MinSimilarity float32
	// Enabled toggles semantic search (default: false until API key set)
	Enabled bool
}

// DefaultVectorConfig returns sensible defaults.
func DefaultVectorConfig() VectorConfig {
	return VectorConfig{
		Model:         "text-embedding-3-small",
		DataDir:       "", // Will use ~/.skulto/vectors
		MinSimilarity: 0.6,
		Enabled:       false,
	}
}

// GitHubConfig holds GitHub API settings.
type GitHubConfig struct {
	Token      string
	RateLimit  int
	UseGraphQL bool
	CacheHours int

	// Git clone-based scraping
	RepoCacheTTL int  // Days to keep cloned repos (default: 7)
	UseGitClone  bool // Use git clone instead of API (default: true)
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Read environment variables
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cfg.GitHub.Token = token
	}

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.Embedding.APIKey = apiKey
		cfg.Embedding.Enabled = true
		cfg.LLM.OpenAIAPIKey = apiKey
	}

	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.LLM.AnthropicAPIKey = apiKey
	}

	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		cfg.LLM.OpenRouterAPIKey = apiKey
	}

	// Ensure directories exist
	if err := ensureDirectories(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ensureDirectories creates required directories if they don't exist.
func ensureDirectories(cfg *Config) error {
	dirs := []string{
		cfg.BaseDir,
		filepath.Join(cfg.BaseDir, "repositories"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
