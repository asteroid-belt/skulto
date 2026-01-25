package vector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/philippgille/chromem-go"
)

// ChromemStore implements VectorStore using chromem-go.
// This is the default implementation - zero external dependencies.
type ChromemStore struct {
	db         *chromem.DB
	collection *chromem.Collection
	dataDir    string
}

// NewChromemStore creates a new chromem-go vector store.
func NewChromemStore(cfg Config) (*ChromemStore, error) {
	if cfg.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY required for embeddings")
	}

	// Determine data directory
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".skulto", "vectors")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create vector dir: %w", err)
	}

	// Create persistent DB
	db, err := chromem.NewPersistentDB(dataDir, false)
	if err != nil {
		return nil, fmt.Errorf("create chromem db: %w", err)
	}

	// Create or get collection with OpenAI embedding function
	embeddingFunc := chromem.NewEmbeddingFuncOpenAI(cfg.OpenAIKey, chromem.EmbeddingModelOpenAI3Small)

	collection, err := db.GetOrCreateCollection("skills", nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}

	return &ChromemStore{
		db:         db,
		collection: collection,
		dataDir:    dataDir,
	}, nil
}

// AddSkill embeds and stores a skill's content.
func (s *ChromemStore) AddSkill(ctx context.Context, skill *models.Skill) (string, error) {
	content := PrepareContent(skill)
	content = TruncateToTokens(content, 8000)
	hash := ContentHash(content)

	doc := chromem.Document{
		ID:      skill.ID,
		Content: content,
		Metadata: map[string]string{
			"title":        skill.Title,
			"content_hash": hash,
		},
	}

	// AddDocuments handles embedding internally
	if err := s.collection.AddDocuments(ctx, []chromem.Document{doc}, runtime.NumCPU()); err != nil {
		return "", fmt.Errorf("add document: %w", err)
	}

	return hash, nil
}

// AddSkillBatch embeds and stores multiple skills efficiently.
func (s *ChromemStore) AddSkillBatch(ctx context.Context, skills []models.Skill) (int, []error) {
	docs := make([]chromem.Document, 0, len(skills))
	var errs []error

	for i := range skills {
		skill := &skills[i]
		content := PrepareContent(skill)
		content = TruncateToTokens(content, 8000)
		hash := ContentHash(content)

		docs = append(docs, chromem.Document{
			ID:      skill.ID,
			Content: content,
			Metadata: map[string]string{
				"title":        skill.Title,
				"content_hash": hash,
			},
		})
	}

	if err := s.collection.AddDocuments(ctx, docs, runtime.NumCPU()); err != nil {
		errs = append(errs, err)
		return 0, errs
	}

	return len(docs), errs
}

// Search finds similar skills by query text.
func (s *ChromemStore) Search(ctx context.Context, query string, limit int, threshold float32) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 50
	}

	// Query handles embedding internally
	// Cap limit to collection size to avoid chromem error
	count := s.collection.Count()
	if limit > count {
		limit = count
	}
	if limit == 0 {
		return []SearchHit{}, nil
	}

	results, err := s.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	hits := make([]SearchHit, 0, len(results))
	for _, r := range results {
		// Filter by threshold
		if r.Similarity < threshold {
			continue
		}

		hits = append(hits, SearchHit{
			SkillID:     r.ID,
			Score:       r.Similarity,
			ContentHash: r.Metadata["content_hash"],
		})
	}

	return hits, nil
}

// Delete removes a skill's embedding by ID.
func (s *ChromemStore) Delete(ctx context.Context, skillID string) error {
	return s.collection.Delete(ctx, nil, nil, skillID)
}

// Count returns total indexed skill count.
func (s *ChromemStore) Count(ctx context.Context) (int64, error) {
	return int64(s.collection.Count()), nil
}

// Close releases resources.
func (s *ChromemStore) Close() error {
	// chromem-go persists automatically, no explicit close needed
	return nil
}
