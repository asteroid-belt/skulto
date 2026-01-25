package search

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/vector"
	"github.com/stretchr/testify/assert"
)

// mockVectorStore implements vector.VectorStore for indexer testing.
type mockVectorStore struct {
	skills     []models.Skill
	addErr     error   // Error to return from AddSkill
	batchErrs  []error // Errors to return from AddSkillBatch
	callCount  int     // Track number of AddSkillBatch calls
	failFirst  int     // Number of times to fail before succeeding
	searchHits []vector.SearchHit
}

func (m *mockVectorStore) AddSkill(ctx context.Context, skill *models.Skill) (string, error) {
	if m.addErr != nil {
		return "", m.addErr
	}
	m.skills = append(m.skills, *skill)
	return vector.ContentHash(vector.PrepareContent(skill)), nil
}

func (m *mockVectorStore) AddSkillBatch(ctx context.Context, skills []models.Skill) (int, []error) {
	m.callCount++

	// Simulate transient failures
	if m.failFirst > 0 && m.callCount <= m.failFirst {
		return 0, []error{errors.New("transient error")}
	}

	if m.batchErrs != nil {
		return 0, m.batchErrs
	}

	m.skills = append(m.skills, skills...)
	return len(skills), nil
}

func (m *mockVectorStore) Search(ctx context.Context, query string, limit int, threshold float32) ([]vector.SearchHit, error) {
	return m.searchHits, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, skillID string) error {
	return nil
}

func (m *mockVectorStore) Count(ctx context.Context) (int64, error) {
	return int64(len(m.skills)), nil
}

func (m *mockVectorStore) Close() error {
	return nil
}

func TestDefaultIndexerConfig(t *testing.T) {
	cfg := DefaultIndexerConfig()

	assert.Equal(t, 50, cfg.BatchSize)
	assert.Equal(t, 3, cfg.RetryAttempts)
	assert.Equal(t, time.Second, cfg.RetryBaseDelay)
}

func TestNewIndexer_DefaultsApplied(t *testing.T) {
	// Test that zero values get defaults
	indexer := NewIndexer(nil, nil, IndexerConfig{})

	assert.Equal(t, 50, indexer.config.BatchSize)
	assert.Equal(t, 3, indexer.config.RetryAttempts)
	assert.Equal(t, time.Second, indexer.config.RetryBaseDelay)
}

func TestNewIndexer_CustomConfig(t *testing.T) {
	cfg := IndexerConfig{
		BatchSize:      100,
		RetryAttempts:  5,
		RetryBaseDelay: 2 * time.Second,
	}

	indexer := NewIndexer(nil, nil, cfg)

	assert.Equal(t, 100, indexer.config.BatchSize)
	assert.Equal(t, 5, indexer.config.RetryAttempts)
	assert.Equal(t, 2*time.Second, indexer.config.RetryBaseDelay)
}

func TestIndexer_GetPendingCount_NilDB(t *testing.T) {
	indexer := NewIndexer(nil, nil, DefaultIndexerConfig())

	// This will panic with nil db, which is expected behavior
	// In real usage, db should never be nil
	assert.Panics(t, func() {
		_, _ = indexer.GetPendingCount()
	})
}

func TestProgress_Struct(t *testing.T) {
	p := Progress{
		Total:     100,
		Completed: 50,
		Failed:    5,
		Skipped:   10,
		Duration:  time.Minute,
	}

	assert.Equal(t, 100, p.Total)
	assert.Equal(t, 50, p.Completed)
	assert.Equal(t, 5, p.Failed)
	assert.Equal(t, 10, p.Skipped)
	assert.Equal(t, time.Minute, p.Duration)
}

func TestMockVectorStore_AddSkill(t *testing.T) {
	store := &mockVectorStore{}
	ctx := context.Background()

	skill := &models.Skill{
		ID:    "test-1",
		Title: "Test Skill",
	}

	hash, err := store.AddSkill(ctx, skill)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, store.skills, 1)
	assert.Equal(t, "Test Skill", store.skills[0].Title)
}

func TestMockVectorStore_AddSkillBatch(t *testing.T) {
	store := &mockVectorStore{}
	ctx := context.Background()

	skills := []models.Skill{
		{ID: "1", Title: "Skill 1"},
		{ID: "2", Title: "Skill 2"},
		{ID: "3", Title: "Skill 3"},
	}

	added, errs := store.AddSkillBatch(ctx, skills)

	assert.Empty(t, errs)
	assert.Equal(t, 3, added)
	assert.Len(t, store.skills, 3)
}

func TestMockVectorStore_AddSkillBatch_WithRetry(t *testing.T) {
	// Test that failFirst causes failures and then succeeds
	store := &mockVectorStore{failFirst: 2} // Fail first 2 calls
	ctx := context.Background()

	skills := []models.Skill{
		{ID: "1", Title: "Skill 1"},
	}

	// First call should fail
	added, errs := store.AddSkillBatch(ctx, skills)
	assert.Equal(t, 0, added)
	assert.Len(t, errs, 1)
	assert.Equal(t, 1, store.callCount)

	// Second call should also fail
	added, errs = store.AddSkillBatch(ctx, skills)
	assert.Equal(t, 0, added)
	assert.Len(t, errs, 1)
	assert.Equal(t, 2, store.callCount)

	// Third call should succeed
	added, errs = store.AddSkillBatch(ctx, skills)
	assert.Equal(t, 1, added)
	assert.Empty(t, errs)
	assert.Equal(t, 3, store.callCount)
}

func TestMockVectorStore_Count(t *testing.T) {
	store := &mockVectorStore{}
	ctx := context.Background()

	// Add some skills
	store.skills = []models.Skill{
		{ID: "1"},
		{ID: "2"},
	}

	count, err := store.Count(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
