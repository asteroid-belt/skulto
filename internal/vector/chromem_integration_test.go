package vector

import (
	"context"
	"os"
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChromemStore_AddAndSearch(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	// Use temp directory
	tmpDir := t.TempDir()

	store, err := NewChromemStore(Config{
		OpenAIKey: apiKey,
		DataDir:   tmpDir,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Add a skill
	skill := &models.Skill{
		ID:          "test-skill-1",
		Title:       "React Testing Guide",
		Description: "Learn how to test React components with Jest",
		Content:     "This guide covers unit testing, integration testing...",
	}

	hash, err := store.AddSkill(ctx, skill)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Search for it
	hits, err := store.Search(ctx, "testing React components", 10, 0.5)
	require.NoError(t, err)

	assert.NotEmpty(t, hits)
	// chromem-go might return hits in any order if scores are close, but here we expect high relevance
	assert.Equal(t, "test-skill-1", hits[0].SkillID)
	assert.Greater(t, hits[0].Score, float32(0.5))
}

func TestChromemStore_NoAPIKey(t *testing.T) {
	_, err := NewChromemStore(Config{
		OpenAIKey: "",
		DataDir:   t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY")
}

func TestChromemStore_Persistence(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create store and add skill
	store1, err := NewChromemStore(Config{OpenAIKey: apiKey, DataDir: tmpDir})
	require.NoError(t, err)

	skill := &models.Skill{ID: "persist-test", Title: "Persistence Test"}
	_, err = store1.AddSkill(ctx, skill)
	require.NoError(t, err)
	_ = store1.Close()

	// Reopen and verify skill still exists
	store2, err := NewChromemStore(Config{OpenAIKey: apiKey, DataDir: tmpDir})
	require.NoError(t, err)
	defer func() { _ = store2.Close() }()

	count, _ := store2.Count(ctx)
	assert.Equal(t, int64(1), count)
}
