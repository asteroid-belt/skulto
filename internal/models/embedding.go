package models

import "time"

// Embedding represents a vector embedding for a skill.
type Embedding struct {
	ID        string    `json:"id" db:"id"`
	SkillID   string    `json:"skill_id" db:"skill_id"`
	Vector    []float32 `json:"-" db:"-"` // 1536-dim for text-embedding-3-small
	Model     string    `json:"model" db:"model"`
	Dimension int       `json:"dimension" db:"dimension"`

	// Hash for cache invalidation
	ContentHash string `json:"content_hash" db:"content_hash"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// EmbeddingConfig holds embedding service configuration.
type EmbeddingConfig struct {
	Model     string `json:"model" yaml:"model"`
	Dimension int    `json:"dimension" yaml:"dimension"`
	BatchSize int    `json:"batch_size" yaml:"batch_size"`
	MaxTokens int    `json:"max_tokens" yaml:"max_tokens"`
	RateLimit int    `json:"rate_limit" yaml:"rate_limit"` // requests per minute
}

// DefaultEmbeddingConfig returns sensible defaults.
func DefaultEmbeddingConfig() EmbeddingConfig {
	return EmbeddingConfig{
		Model:     "text-embedding-3-small",
		Dimension: 1536,
		BatchSize: 100,
		MaxTokens: 8191,
		RateLimit: 3000, // 3000 RPM for text-embedding-3-small
	}
}

// EmbeddingModels maps model names to their dimensions.
var EmbeddingModelDimensions = map[string]int{
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,
}
