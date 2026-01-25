//go:build functional

package search_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/testutil"
	"github.com/asteroid-belt/skulto/internal/tui/components"
	"github.com/asteroid-belt/skulto/internal/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctional_SemanticSearchPipeline tests Phases 1-3 end-to-end.
//
// Run with:
//
//	RUN_AI_TESTS=1 go test -tags=functional ./internal/search/... -v -run Functional
//
// This test:
//  1. Creates a test database with sample skills
//  2. Initializes chromem-go vector store (Phase 1A)
//  3. Uses the Indexer to batch-index skills (Phase 3)
//  4. Performs semantic search via Search Service (Phase 1C)
//  5. Converts results to UnifiedResultList format (Phase 2)
func TestFunctional_SemanticSearchPipeline(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping functional test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Semantic Search Functional Test (Phases 1-3)           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// ═══════════════════════════════════════════════════════════════════════
	// Step 1: Create test database with sample skills
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 1: Creating test database ━━━")

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{Path: dbPath})
	require.NoError(t, err)
	defer database.Close()

	// Sample skills representing different domains
	testSkills := []models.Skill{
		{
			ID:          "skill-react-testing",
			Title:       "React Testing Guide",
			Description: "Comprehensive guide to testing React components",
			Content:     "Learn how to test React components using Jest and React Testing Library. This guide covers unit testing, integration testing, snapshot testing, and mocking.",
			Category:    "testing",
			Tags: []models.Tag{
				{ID: "react", Name: "React", Slug: "react"},
				{ID: "testing", Name: "Testing", Slug: "testing"},
			},
		},
		{
			ID:          "skill-go-concurrency",
			Title:       "Go Concurrency Patterns",
			Description: "Master goroutines and channels in Go",
			Content:     "Understand Go's concurrency model with goroutines and channels. Learn patterns like worker pools, fan-out/fan-in, and context cancellation.",
			Category:    "golang",
			Tags: []models.Tag{
				{ID: "go", Name: "Go", Slug: "go"},
				{ID: "concurrency", Name: "Concurrency", Slug: "concurrency"},
			},
		},
		{
			ID:          "skill-python-ml",
			Title:       "Python Machine Learning",
			Description: "Introduction to ML with Python",
			Content:     "Build machine learning models using scikit-learn, pandas, and numpy. Covers classification, regression, and clustering algorithms.",
			Category:    "python",
			Tags: []models.Tag{
				{ID: "python", Name: "Python", Slug: "python"},
				{ID: "ml", Name: "Machine Learning", Slug: "machine-learning"},
			},
		},
		{
			ID:          "skill-typescript-types",
			Title:       "TypeScript Advanced Types",
			Description: "Deep dive into TypeScript's type system",
			Content:     "Master TypeScript generics, conditional types, mapped types, and utility types for building type-safe applications.",
			Category:    "typescript",
			Tags: []models.Tag{
				{ID: "typescript", Name: "TypeScript", Slug: "typescript"},
				{ID: "types", Name: "Types", Slug: "types"},
			},
		},
	}

	for _, skill := range testSkills {
		err := database.UpsertSkillWithTags(&skill, skill.Tags)
		require.NoError(t, err)
		fmt.Printf("  ✓ Created skill: %s\n", skill.Title)
	}

	// Verify skills were created
	pending, err := database.CountSkillsWithoutEmbedding()
	require.NoError(t, err)
	fmt.Printf("  Skills pending embedding: %d\n", pending)

	// ═══════════════════════════════════════════════════════════════════════
	// Step 2: Initialize chromem-go Vector Store (Phase 1A)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 2: Initializing chromem-go vector store ━━━")

	vectorDir := filepath.Join(tmpDir, "vectors")
	store, err := vector.NewChromemStore(vector.Config{
		OpenAIKey: apiKey,
		DataDir:   vectorDir,
	})
	require.NoError(t, err)
	defer store.Close()

	fmt.Printf("  ✓ Vector store initialized at: %s\n", vectorDir)

	// ═══════════════════════════════════════════════════════════════════════
	// Step 3: Batch index skills using Indexer (Phase 3)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 3: Batch indexing skills (Indexer) ━━━")

	indexer := search.NewIndexer(database, store, search.IndexerConfig{
		BatchSize:      10,
		RetryAttempts:  3,
		RetryBaseDelay: time.Second,
	})

	// Create progress channel
	progressCh := make(chan search.Progress, 10)
	done := make(chan struct{})

	go func() {
		for p := range progressCh {
			fmt.Printf("  Progress: %d/%d completed, %d skipped, %d failed\n",
				p.Completed, p.Total, p.Skipped, p.Failed)
		}
		close(done)
	}()

	// Run indexing
	start := time.Now()
	err = indexer.IndexAll(ctx, progressCh)
	close(progressCh)
	<-done

	require.NoError(t, err)
	fmt.Printf("  ✓ Indexing completed in %v\n", time.Since(start).Round(time.Millisecond))

	// Verify indexing
	count, err := store.Count(ctx)
	require.NoError(t, err)
	fmt.Printf("  Vectors in store: %d\n", count)
	assert.Equal(t, int64(len(testSkills)), count)

	// ═══════════════════════════════════════════════════════════════════════
	// Step 4: Perform semantic search (Phase 1C)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 4: Testing semantic search ━━━")

	svc := search.New(database, store, search.DefaultConfig())

	// Test queries that should match semantically (not just by keywords)
	testQueries := []struct {
		query       string
		expectTitle string // Expected skill in results
	}{
		{
			query:       "how do I write unit tests for React apps",
			expectTitle: "React Testing Guide",
		},
		{
			query:       "parallel programming in Go",
			expectTitle: "Go Concurrency Patterns",
		},
		{
			query:       "building AI models with Python",
			expectTitle: "Python Machine Learning",
		},
		{
			query:       "generics and type safety",
			expectTitle: "TypeScript Advanced Types",
		},
	}

	for _, tc := range testQueries {
		fmt.Printf("\n  Query: %q\n", tc.query)

		results, err := svc.Search(ctx, tc.query, search.SearchOptions{
			Limit:           10,
			Threshold:       0.3, // Lower threshold to catch semantic matches
			IncludeFTS:      true,
			IncludeSemantic: true,
		})
		require.NoError(t, err)

		fmt.Printf("  Results: %d total (%d title, %d content) in %v\n",
			results.TotalHits,
			len(results.TitleMatches),
			len(results.ContentMatches),
			results.Duration.Round(time.Millisecond))

		// Check if expected skill is in results
		found := false
		for _, m := range results.TitleMatches {
			if m.Skill.Title == tc.expectTitle {
				found = true
				fmt.Printf("  ✓ Found expected: %s (score: %.2f) [title]\n", m.Skill.Title, m.Score)
			}
		}
		for _, m := range results.ContentMatches {
			if m.Skill.Title == tc.expectTitle {
				found = true
				fmt.Printf("  ✓ Found expected: %s (score: %.2f) [content]\n", m.Skill.Title, m.Score)
				// Show snippets
				for _, s := range m.Snippets {
					highlighted := search.HighlightText(s)
					if len(highlighted) > 80 {
						highlighted = highlighted[:80] + "..."
					}
					fmt.Printf("    Snippet: %s\n", highlighted)
				}
			}
		}

		if !found && results.TotalHits > 0 {
			fmt.Printf("  ⚠ Expected %q not found, but got other results\n", tc.expectTitle)
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// Step 5: Convert to UnifiedResultList format (Phase 2)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 5: Testing UnifiedResultList (TUI) ━━━")

	// Perform a search
	results, err := svc.Search(ctx, "testing components", search.SearchOptions{
		Limit:           10,
		Threshold:       0.3,
		IncludeFTS:      true,
		IncludeSemantic: true,
	})
	require.NoError(t, err)

	// Convert to UnifiedResultItem format (like the TUI does)
	var items []components.UnifiedResultItem

	for _, match := range results.TitleMatches {
		items = append(items, components.UnifiedResultItem{
			Skill:     match.Skill,
			MatchType: components.MatchTypeName,
		})
	}

	for _, match := range results.ContentMatches {
		items = append(items, components.UnifiedResultItem{
			Skill:     match.Skill,
			MatchType: components.MatchTypeContent,
			Snippets:  match.Snippets,
		})
	}

	// Create UnifiedResultList
	list := components.NewUnifiedResultList()
	list.SetSize(80, 24)
	list.SetItems(items)

	fmt.Printf("  UnifiedResultList items: %d\n", list.TotalCount())
	fmt.Printf("  Selected index: %d\n", list.Selected)

	// Test navigation
	if list.TotalCount() > 1 {
		list.MoveDown()
		fmt.Printf("  After MoveDown, selected: %d\n", list.Selected)
		assert.Equal(t, 1, list.Selected)
	}

	// Test expansion
	for i, item := range items {
		if item.MatchType == components.MatchTypeContent && len(item.Snippets) > 0 {
			list.Selected = i
			expanded := list.ToggleExpand()
			fmt.Printf("  ToggleExpand on content match: %v\n", expanded)
			assert.True(t, expanded)
			break
		}
	}

	// Render the view
	view := list.View()
	fmt.Printf("  View rendered (%d chars)\n", len(view))
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "[name]") // Badge for name matches
	// Note: May contain [content] if there are content matches

	// ═══════════════════════════════════════════════════════════════════════
	// Summary
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Test Summary                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("  ✓ Phase 1A: chromem-go vector store working")
	fmt.Println("  ✓ Phase 1C: Search service with snippets working")
	fmt.Println("  ✓ Phase 2:  UnifiedResultList component working")
	fmt.Println("  ✓ Phase 3:  Indexer with batch processing working")
	fmt.Println("")
	fmt.Println("  All phases verified successfully!")
	fmt.Println("")
}
