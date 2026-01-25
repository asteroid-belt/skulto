package vector

import (
	"context"

	"github.com/asteroid-belt/skulto/internal/models"
)

// VectorStore abstracts vector storage with built-in embedding support.
// Implementations handle both embedding generation and storage.
type VectorStore interface {
	// AddSkill embeds and stores a skill's content.
	// Returns the content hash used for cache invalidation.
	AddSkill(ctx context.Context, skill *models.Skill) (contentHash string, err error)

	// AddSkillBatch embeds and stores multiple skills efficiently.
	// Returns number of successful additions and any errors.
	AddSkillBatch(ctx context.Context, skills []models.Skill) (added int, errs []error)

	// Search finds similar skills by query text.
	// Handles query embedding internally.
	Search(ctx context.Context, query string, limit int, threshold float32) ([]SearchHit, error)

	// Delete removes a skill's embedding by ID.
	Delete(ctx context.Context, skillID string) error

	// Count returns total indexed skill count.
	Count(ctx context.Context) (int64, error)

	// Close releases resources.
	Close() error
}

// SearchHit represents a search result with similarity score.
type SearchHit struct {
	SkillID     string
	Score       float32 // Cosine similarity (0.0-1.0)
	ContentHash string  // For cache validation
}

// Config holds vector store configuration.
type Config struct {
	// DataDir is where chromem-go persists vectors (default: ~/.skulto/vectors)
	DataDir string

	// OpenAI settings for embeddings
	OpenAIKey string
	Model     string // default: "text-embedding-3-small"
}

// New creates a VectorStore using chromem-go.
func New(cfg Config) (VectorStore, error) {
	return NewChromemStore(cfg)
}
