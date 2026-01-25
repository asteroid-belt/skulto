//go:build functional

package search_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/testutil"
	"github.com/asteroid-belt/skulto/internal/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctional_Phase4_BackgroundIndexer tests the complete Phase 4 implementation.
//
// Run with:
//
//	RUN_AI_TESTS=1 go test -tags=functional ./internal/search/... -v -run Phase4
//
// This test verifies:
//  1. BackgroundIndexer creation and lifecycle
//  2. Non-blocking start behavior
//  3. Progress reporting via channels
//  4. Integration with VectorStore
//  5. Graceful shutdown
//  6. Pending count detection
func TestFunctional_Phase4_BackgroundIndexer(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping functional test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Phase 4 Functional Test: Background Indexer            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// ═══════════════════════════════════════════════════════════════════════
	// Step 1: Create test database with skills that need indexing
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 1: Creating test database with unindexed skills ━━━")

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{Path: dbPath})
	require.NoError(t, err)
	defer database.Close()

	// Create skills WITHOUT embeddings (EmbeddingID is empty)
	testSkills := []models.Skill{
		{
			ID:          "phase4-skill-1",
			Title:       "Background Indexing Test 1",
			Description: "First skill for Phase 4 testing",
			Content:     "This skill tests background indexing with goroutines and channels.",
			Category:    "testing",
		},
		{
			ID:          "phase4-skill-2",
			Title:       "Background Indexing Test 2",
			Description: "Second skill for Phase 4 testing",
			Content:     "This skill verifies non-blocking TUI launch behavior.",
			Category:    "testing",
		},
		{
			ID:          "phase4-skill-3",
			Title:       "Background Indexing Test 3",
			Description: "Third skill for Phase 4 testing",
			Content:     "This skill confirms progress reporting works correctly.",
			Category:    "testing",
		},
	}

	for _, skill := range testSkills {
		err := database.UpsertSkillWithTags(&skill, nil)
		require.NoError(t, err)
		fmt.Printf("  ✓ Created skill: %s (EmbeddingID: %q)\n", skill.Title, skill.EmbeddingID)
	}

	// Verify skills need indexing
	pendingBefore, err := database.CountSkillsWithoutEmbedding()
	require.NoError(t, err)
	fmt.Printf("  Skills pending embedding: %d\n", pendingBefore)
	assert.Equal(t, len(testSkills), pendingBefore)

	// ═══════════════════════════════════════════════════════════════════════
	// Step 2: Initialize VectorStore
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 2: Initializing VectorStore ━━━")

	vectorDir := filepath.Join(tmpDir, "vectors")
	store, err := vector.NewChromemStore(vector.Config{
		OpenAIKey: apiKey,
		DataDir:   vectorDir,
	})
	require.NoError(t, err)
	fmt.Printf("  ✓ VectorStore initialized at: %s\n", vectorDir)

	// ═══════════════════════════════════════════════════════════════════════
	// Step 3: Create BackgroundIndexer
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 3: Creating BackgroundIndexer ━━━")

	cfg := search.IndexerConfig{
		BatchSize:      10,
		RetryAttempts:  3,
		RetryBaseDelay: time.Second,
	}

	indexer := search.NewBackgroundIndexer(database, store, cfg)
	require.NotNil(t, indexer)
	fmt.Println("  ✓ BackgroundIndexer created")

	// Verify VectorStore accessor
	assert.Equal(t, store, indexer.VectorStore())
	fmt.Println("  ✓ VectorStore() accessor works")

	// Verify pending count
	pending, err := indexer.GetPendingCount()
	require.NoError(t, err)
	assert.Equal(t, len(testSkills), pending)
	fmt.Printf("  ✓ GetPendingCount() returns %d\n", pending)

	// ═══════════════════════════════════════════════════════════════════════
	// Step 4: Test Non-Blocking Start
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 4: Testing non-blocking Start() ━━━")

	progressCh := make(chan search.IndexProgress, 20)

	startTime := time.Now()
	err = indexer.Start(ctx, progressCh)
	startDuration := time.Since(startTime)

	require.NoError(t, err)
	fmt.Printf("  ✓ Start() returned in %v (should be < 50ms)\n", startDuration.Round(time.Millisecond))
	assert.Less(t, startDuration, 100*time.Millisecond, "Start should be non-blocking")

	// Verify IsRunning
	assert.True(t, indexer.IsRunning())
	fmt.Println("  ✓ IsRunning() returns true")

	// ═══════════════════════════════════════════════════════════════════════
	// Step 5: Test Progress Reporting
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 5: Testing progress reporting ━━━")

	var progressUpdates []search.IndexProgress
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for progress := range progressCh {
			progressUpdates = append(progressUpdates, progress)
			fmt.Printf("  Progress: Running=%v, Completed=%d/%d, Failed=%d, Message=%q\n",
				progress.Running, progress.Completed, progress.Total, progress.Failed, progress.Message)

			// Stop collecting when done
			if !progress.Running {
				return
			}
		}
	}()

	// Wait for indexing to complete (with timeout)
	done := make(chan struct{})
	go func() {
		indexer.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("  ✓ Indexing completed")
	case <-time.After(2 * time.Minute):
		t.Fatal("Timeout waiting for indexing to complete")
	}

	// Close progress channel to stop the goroutine
	close(progressCh)
	wg.Wait()

	// Verify we got progress updates
	assert.NotEmpty(t, progressUpdates, "Should have received progress updates")
	fmt.Printf("  ✓ Received %d progress updates\n", len(progressUpdates))

	// Verify final progress
	if len(progressUpdates) > 0 {
		final := progressUpdates[len(progressUpdates)-1]
		assert.False(t, final.Running, "Final progress should have Running=false")
		fmt.Printf("  ✓ Final state: Completed=%d, Failed=%d\n", final.Completed, final.Failed)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// Step 6: Verify Skills Were Indexed
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 6: Verifying skills were indexed ━━━")

	// Check vector store count
	vectorCount, err := store.Count(ctx)
	require.NoError(t, err)
	fmt.Printf("  Vectors in store: %d\n", vectorCount)
	assert.Equal(t, int64(len(testSkills)), vectorCount)

	// Verify IsRunning is now false
	assert.False(t, indexer.IsRunning())
	fmt.Println("  ✓ IsRunning() returns false after completion")

	// ═══════════════════════════════════════════════════════════════════════
	// Step 7: Test Semantic Search Works
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 7: Testing semantic search on indexed skills ━━━")

	// Create search service with the vector store
	searchSvc := search.New(database, store, search.DefaultConfig())

	results, err := searchSvc.Search(ctx, "goroutines and channels", search.SearchOptions{
		Limit:           10,
		Threshold:       0.3,
		IncludeFTS:      true,
		IncludeSemantic: true,
	})
	require.NoError(t, err)

	fmt.Printf("  Query: \"goroutines and channels\"\n")
	fmt.Printf("  Results: %d total (%d title, %d content)\n",
		results.TotalHits, len(results.TitleMatches), len(results.ContentMatches))

	// Should find the skill about goroutines
	found := false
	for _, m := range results.TitleMatches {
		if m.Skill.ID == "phase4-skill-1" {
			found = true
			fmt.Printf("  ✓ Found expected skill in title matches: %s\n", m.Skill.Title)
		}
	}
	for _, m := range results.ContentMatches {
		if m.Skill.ID == "phase4-skill-1" {
			found = true
			fmt.Printf("  ✓ Found expected skill in content matches: %s\n", m.Skill.Title)
		}
	}
	assert.True(t, found, "Should find skill about goroutines")

	// ═══════════════════════════════════════════════════════════════════════
	// Step 8: Test Graceful Close
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n━━━ Step 8: Testing graceful Close() ━━━")

	err = indexer.Close()
	require.NoError(t, err)
	fmt.Println("  ✓ Close() completed without error")

	// ═══════════════════════════════════════════════════════════════════════
	// Summary
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Phase 4 Test Summary                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("  ✓ BackgroundIndexer creation and lifecycle")
	fmt.Println("  ✓ Non-blocking Start() behavior")
	fmt.Println("  ✓ Progress reporting via channels")
	fmt.Println("  ✓ VectorStore integration")
	fmt.Println("  ✓ Pending count detection")
	fmt.Println("  ✓ Semantic search on indexed skills")
	fmt.Println("  ✓ Graceful shutdown")
	fmt.Println("")
	fmt.Println("  All Phase 4 components verified successfully!")
	fmt.Println("")
}

// TestFunctional_Phase4_StartAlreadyRunning verifies Start is idempotent.
func TestFunctional_Phase4_StartAlreadyRunning(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping functional test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	fmt.Println("\n━━━ Testing: Start() when already running ━━━")

	// Setup database
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{Path: dbPath})
	require.NoError(t, err)
	defer database.Close()

	// Create a skill
	skill := models.Skill{ID: "test-1", Title: "Test", Content: "Test content"}
	require.NoError(t, database.UpsertSkillWithTags(&skill, nil))

	// Setup vector store
	vectorDir := filepath.Join(tmpDir, "vectors")
	store, err := vector.NewChromemStore(vector.Config{
		OpenAIKey: apiKey,
		DataDir:   vectorDir,
	})
	require.NoError(t, err)

	// Create indexer
	indexer := search.NewBackgroundIndexer(database, store, search.DefaultIndexerConfig())

	progressCh := make(chan search.IndexProgress, 20)

	// Start first time
	err = indexer.Start(ctx, progressCh)
	require.NoError(t, err)
	fmt.Println("  ✓ First Start() succeeded")

	// Start second time (should be no-op)
	err = indexer.Start(ctx, progressCh)
	require.NoError(t, err)
	fmt.Println("  ✓ Second Start() succeeded (no-op)")

	// Wait for completion
	indexer.Wait()

	// Cleanup
	close(progressCh)
	require.NoError(t, indexer.Close())
	fmt.Println("  ✓ Cleanup completed")
}

// TestFunctional_Phase4_NoPendingSkills tests behavior when no skills need indexing.
func TestFunctional_Phase4_NoPendingSkills(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping functional test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	fmt.Println("\n━━━ Testing: No pending skills scenario ━━━")

	// Setup database with NO skills
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{Path: dbPath})
	require.NoError(t, err)
	defer database.Close()

	// Setup vector store
	vectorDir := filepath.Join(tmpDir, "vectors")
	store, err := vector.NewChromemStore(vector.Config{
		OpenAIKey: apiKey,
		DataDir:   vectorDir,
	})
	require.NoError(t, err)

	// Create indexer
	indexer := search.NewBackgroundIndexer(database, store, search.DefaultIndexerConfig())

	// Verify no pending
	pending, err := indexer.GetPendingCount()
	require.NoError(t, err)
	assert.Equal(t, 0, pending)
	fmt.Printf("  ✓ GetPendingCount() returns %d\n", pending)

	progressCh := make(chan search.IndexProgress, 10)

	// Start - should complete quickly
	err = indexer.Start(ctx, progressCh)
	require.NoError(t, err)

	// Wait with short timeout
	done := make(chan struct{})
	go func() {
		indexer.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("  ✓ Indexer completed quickly (no work to do)")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout - should complete quickly when no skills")
	}

	// Should get a "no skills" or completion message
	select {
	case progress := <-progressCh:
		fmt.Printf("  ✓ Received progress: Running=%v, Message=%q\n", progress.Running, progress.Message)
		assert.False(t, progress.Running)
	case <-time.After(time.Second):
		// May not receive any message if nothing to do
		fmt.Println("  ✓ No progress message (nothing to index)")
	}

	close(progressCh)
	require.NoError(t, indexer.Close())
}

// TestFunctional_Phase4_ContextCancellation tests cancellation behavior.
func TestFunctional_Phase4_ContextCancellation(t *testing.T) {
	testutil.SkipAITests(t)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping functional test")
	}

	tmpDir := t.TempDir()

	fmt.Println("\n━━━ Testing: Context cancellation / Stop() ━━━")

	// Setup database with many skills
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{Path: dbPath})
	require.NoError(t, err)
	defer database.Close()

	// Create many skills to ensure indexing takes a while
	for i := 0; i < 20; i++ {
		skill := models.Skill{
			ID:      fmt.Sprintf("cancel-test-%d", i),
			Title:   fmt.Sprintf("Cancellation Test Skill %d", i),
			Content: fmt.Sprintf("Content for skill %d with lots of text to embed", i),
		}
		require.NoError(t, database.UpsertSkillWithTags(&skill, nil))
	}

	// Setup vector store
	vectorDir := filepath.Join(tmpDir, "vectors")
	store, err := vector.NewChromemStore(vector.Config{
		OpenAIKey: apiKey,
		DataDir:   vectorDir,
	})
	require.NoError(t, err)

	// Create indexer with small batch to make it take longer
	indexer := search.NewBackgroundIndexer(database, store, search.IndexerConfig{
		BatchSize:      2, // Small batch = more iterations
		RetryAttempts:  1,
		RetryBaseDelay: time.Second,
	})

	progressCh := make(chan search.IndexProgress, 50)

	// Start indexing
	ctx := context.Background()
	err = indexer.Start(ctx, progressCh)
	require.NoError(t, err)
	fmt.Println("  ✓ Started indexing")

	// Wait a tiny bit then stop
	time.Sleep(100 * time.Millisecond)

	// Stop should not block
	startStop := time.Now()
	indexer.Stop()
	stopDuration := time.Since(startStop)
	fmt.Printf("  ✓ Stop() returned in %v\n", stopDuration.Round(time.Millisecond))

	// Wait for the indexer to actually stop
	done := make(chan struct{})
	go func() {
		indexer.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("  ✓ Indexer stopped successfully")
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for indexer to stop")
	}

	// IsRunning should be false
	assert.False(t, indexer.IsRunning())
	fmt.Println("  ✓ IsRunning() returns false after Stop()")

	close(progressCh)
	require.NoError(t, indexer.Close())
}
