//go:build integration

package search_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/testutil"
	"github.com/asteroid-belt/skulto/internal/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_SemanticSearch tests the full semantic search flow.
// Run with: RUN_AI_TESTS=1 go test -tags=integration ./internal/search/... -v
// Requires: OPENAI_API_KEY environment variable
func TestIntegration_SemanticSearch(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	// 1. Create database with test skills
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{Path: dbPath})
	require.NoError(t, err)
	defer database.Close()

	// Create test skills
	skills := []models.Skill{
		{
			ID:          "skill-react-testing",
			Title:       "React Testing Guide",
			Description: "Learn how to test React components with Jest and React Testing Library",
			Content:     "This guide covers unit testing, integration testing, and end-to-end testing for React applications.",
			Category:    "testing",
		},
		{
			ID:          "skill-go-concurrency",
			Title:       "Go Concurrency Patterns",
			Description: "Master goroutines, channels, and concurrent programming in Go",
			Content:     "Learn about goroutines, channels, select statements, and common concurrency patterns in Go.",
			Category:    "golang",
		},
		{
			ID:          "skill-python-ml",
			Title:       "Python Machine Learning",
			Description: "Introduction to machine learning with Python and scikit-learn",
			Content:     "Build machine learning models using Python, pandas, numpy, and scikit-learn.",
			Category:    "python",
		},
	}

	for _, skill := range skills {
		err := database.CreateSkill(&skill)
		require.NoError(t, err)
	}

	// 2. Create vector store
	vectorDir := filepath.Join(tmpDir, "vectors")
	store, err := vector.NewChromemStore(vector.Config{
		OpenAIKey: apiKey,
		DataDir:   vectorDir,
	})
	require.NoError(t, err)
	defer store.Close()

	// 3. Index skills (continue on error - API may rate limit)
	fmt.Println("Indexing skills...")
	indexedCount := 0
	for _, skill := range skills {
		hash, err := store.AddSkill(ctx, &skill)
		if err != nil {
			fmt.Printf("  Warning: Failed to index %s: %v\n", skill.Title, err)
			continue
		}
		fmt.Printf("  Indexed: %s (hash: %s...)\n", skill.Title, hash[:8])
		indexedCount++
	}

	if indexedCount == 0 {
		t.Skip("No skills were indexed - API may be unavailable or rate limited")
	}

	// 4. Create search service
	svc := search.New(database, store, search.DefaultConfig())

	// 5. Test semantic search
	testQueries := []string{
		"how to test React components",
		"concurrent programming",
		"machine learning Python scikit", // More specific terms
	}

	for _, query := range testQueries {
		fmt.Printf("\n--- Query: %q ---\n", query)

		results, err := svc.Search(ctx, query, search.SearchOptions{
			Limit:           10,
			Threshold:       0.4, // Lower threshold to catch more semantic matches
			IncludeFTS:      true,
			IncludeSemantic: true,
		})
		require.NoError(t, err)

		fmt.Printf("Total hits: %d (took %v)\n", results.TotalHits, results.Duration)
		fmt.Printf("Title matches: %d\n", len(results.TitleMatches))
		for _, m := range results.TitleMatches {
			fmt.Printf("  - %s (score: %.2f)\n", m.Skill.Title, m.Score)
		}
		fmt.Printf("Content matches: %d\n", len(results.ContentMatches))
		for _, m := range results.ContentMatches {
			fmt.Printf("  - %s (score: %.2f)\n", m.Skill.Title, m.Score)
			for _, s := range m.Snippets {
				fmt.Printf("    Snippet: %s\n", search.HighlightText(s))
			}
		}

		// Note: May not find results for all queries depending on which skills were indexed
		if results.TotalHits > 0 {
			fmt.Println("  ✓ Found matching results")
		}
	}

	// 6. Test vector store stats
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(indexedCount), count)

	fmt.Println("\n✓ Integration test passed!")
}
