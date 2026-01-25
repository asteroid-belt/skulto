package config

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BaseDir: DefaultBaseDir(),

		GitHub: GitHubConfig{
			RateLimit:    30,
			UseGraphQL:   true,
			CacheHours:   24,
			RepoCacheTTL: 7, // Keep cloned repos for 7 days
			UseGitClone:  true,
		},

		Embedding: DefaultVectorConfig(),

		LLM: DefaultLLMConfig(),
	}
}

// EmbeddingModels defines available embedding models.
var EmbeddingModels = map[string]int{
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,
}
