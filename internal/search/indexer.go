package search

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/vector"
)

// Indexer handles bulk and incremental indexing with retry support.
type Indexer struct {
	db     *db.DB
	store  vector.VectorStore
	config IndexerConfig
}

// IndexerConfig holds indexer settings.
type IndexerConfig struct {
	BatchSize      int
	RetryAttempts  int
	RetryBaseDelay time.Duration // Base delay for exponential backoff
}

// DefaultIndexerConfig returns sensible defaults.
func DefaultIndexerConfig() IndexerConfig {
	return IndexerConfig{
		BatchSize:      50,
		RetryAttempts:  3,
		RetryBaseDelay: time.Second,
	}
}

// Progress reports indexing progress.
type Progress struct {
	Total     int
	Completed int
	Failed    int
	Skipped   int
	Duration  time.Duration
}

// NewIndexer creates a new indexer.
func NewIndexer(database *db.DB, store vector.VectorStore, cfg IndexerConfig) *Indexer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = 3
	}
	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay = time.Second
	}

	return &Indexer{
		db:     database,
		store:  store,
		config: cfg,
	}
}

// IndexAll indexes all skills with progress reporting.
func (idx *Indexer) IndexAll(ctx context.Context, progress chan<- Progress) error {
	start := time.Now()

	// Get all skills
	skills, err := idx.db.ListSkills(100000, 0)
	if err != nil {
		return fmt.Errorf("list skills: %w", err)
	}

	if len(skills) == 0 {
		return nil
	}

	total := len(skills)
	completed := 0
	failed := 0
	skipped := 0

	// Process in batches
	for i := 0; i < len(skills); i += idx.config.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := min(i+idx.config.BatchSize, len(skills))
		batch := skills[i:end]

		batchCompleted, batchSkipped, batchFailed := idx.indexBatchWithRetry(ctx, batch)
		completed += batchCompleted
		skipped += batchSkipped
		failed += batchFailed

		if progress != nil {
			progress <- Progress{
				Total:     total,
				Completed: completed,
				Failed:    failed,
				Skipped:   skipped,
				Duration:  time.Since(start),
			}
		}
	}

	return nil
}

// indexBatchWithRetry indexes a batch with exponential backoff retry.
func (idx *Indexer) indexBatchWithRetry(ctx context.Context, skills []models.Skill) (completed, skipped, failed int) {
	// Filter skills that need indexing (check content hash)
	var toIndex []models.Skill
	for i := range skills {
		skill := &skills[i]
		content := vector.PrepareContent(skill)
		hash := vector.ContentHash(content)
		if skill.EmbeddingID == hash {
			skipped++
			continue
		}
		toIndex = append(toIndex, *skill)
	}

	if len(toIndex) == 0 {
		return 0, skipped, 0
	}

	// Try with exponential backoff
	for attempt := 0; attempt < idx.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, ...
			delay := idx.config.RetryBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))

			select {
			case <-ctx.Done():
				return completed, skipped, len(toIndex) - completed
			case <-time.After(delay):
			}
		}

		// Use VectorStore's batch method (handles embedding internally)
		batchCompleted, errs := idx.store.AddSkillBatch(ctx, toIndex)
		if len(errs) == 0 {
			// Success - update skills with embedding IDs
			for i := range toIndex {
				content := vector.PrepareContent(&toIndex[i])
				hash := vector.ContentHash(content)
				toIndex[i].EmbeddingID = hash
				if err := idx.db.UpdateSkill(&toIndex[i]); err != nil {
					log.Printf("failed to update skill %s embedding ID: %v", toIndex[i].ID, err)
					failed++
				} else {
					completed++
				}
			}
			return completed, skipped, failed
		}

		// Partial success - some may have worked
		if batchCompleted > 0 {
			completed += batchCompleted
		}
	}

	// All retries exhausted
	return completed, skipped, len(toIndex) - completed
}

// IndexPending indexes only skills without embeddings.
func (idx *Indexer) IndexPending(ctx context.Context, progress chan<- Progress) error {
	start := time.Now()

	skills, err := idx.db.GetSkillsWithoutEmbedding(idx.config.BatchSize * 10)
	if err != nil {
		return fmt.Errorf("get pending skills: %w", err)
	}

	if len(skills) == 0 {
		return nil
	}

	total := len(skills)
	completed := 0
	failed := 0

	for i := 0; i < len(skills); i += idx.config.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := min(i+idx.config.BatchSize, len(skills))
		batch := skills[i:end]

		batchCompleted, _, batchFailed := idx.indexBatchWithRetry(ctx, batch)
		completed += batchCompleted
		failed += batchFailed

		if progress != nil {
			progress <- Progress{
				Total:     total,
				Completed: completed,
				Failed:    failed,
				Duration:  time.Since(start),
			}
		}
	}

	return nil
}

// GetPendingCount returns the number of skills needing indexing.
func (idx *Indexer) GetPendingCount() (int, error) {
	return idx.db.CountSkillsWithoutEmbedding()
}
