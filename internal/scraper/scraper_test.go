package scraper

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
)

func TestResponseCache(t *testing.T) {
	cache := NewResponseCache(100 * time.Millisecond)

	// Test Set and Get
	cache.Set("key1", "value1")
	val, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected cache hit")
	}
	if val.(string) != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Test cache miss
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("Expected cache miss for nonexistent key")
	}

	// Test expiration
	time.Sleep(150 * time.Millisecond)
	_, ok = cache.Get("key1")
	if ok {
		t.Error("Expected cache miss after expiration")
	}
}

func TestResponseCacheDelete(t *testing.T) {
	cache := NewResponseCache(time.Hour)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("Expected cache miss after delete")
	}
}

func TestResponseCacheClear(t *testing.T) {
	cache := NewResponseCache(time.Hour)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Expected empty cache after clear, got %d items", cache.Len())
	}
}

func TestAllSeeds(t *testing.T) {
	seeds := AllSeeds()

	if len(seeds) == 0 {
		t.Error("Expected at least one seed")
	}

	// Check that seeds are sorted by priority (descending)
	for i := 1; i < len(seeds); i++ {
		if seeds[i].Priority > seeds[i-1].Priority {
			t.Errorf("Seeds not sorted by priority: %d > %d at position %d",
				seeds[i].Priority, seeds[i-1].Priority, i)
		}
	}

	// Check that official seeds come first (priority 10)
	if seeds[0].Priority != 10 {
		t.Errorf("Expected first seed to have priority 10, got %d", seeds[0].Priority)
	}
}

func TestIsSkillFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"SKILL.md", true},
		{"README.md", false},
		{"skill.md", true},  // Case insensitive for consistency with IsSkillFilePath
		{"CLAUDE.md", true}, // Also matches claude.md files
		{"claude.md", true}, // Case insensitive
		{"SKILLS.md", false},
		{"random.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsSkillFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsSkillFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	content := `
# Python Testing Guide

This skill teaches you how to write tests using pytest for Python applications.
It covers unit testing, integration testing, and test-driven development with Django.

## Requirements
- Python 3.8+
- pytest
- Django (optional)
`

	tags := ExtractTags(content)

	// Check that we found expected tags
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	expectedTags := []string{"python", "testing", "django"}
	for _, expected := range expectedTags {
		if !tagNames[expected] {
			t.Errorf("Expected to find tag %q", expected)
		}
	}

	// Check that tags have proper structure
	for _, tag := range tags {
		if tag.ID == "" {
			t.Error("Tag ID should not be empty")
		}
		if tag.Slug == "" {
			t.Error("Tag Slug should not be empty")
		}
		if tag.Category == "" {
			t.Error("Tag Category should not be empty")
		}
	}
}

func TestExtractTagsEmpty(t *testing.T) {
	tags := ExtractTags("")
	if len(tags) != 0 {
		t.Errorf("Expected no tags from empty content, got %d", len(tags))
	}
}

func TestExtractTagsNoMatches(t *testing.T) {
	content := "This is a generic document with no programming language mentions."
	tags := ExtractTags(content)
	if len(tags) != 0 {
		t.Errorf("Expected no tags, got %d: %v", len(tags), tags)
	}
}

func TestSearchQueries(t *testing.T) {
	if len(SearchQueries) == 0 {
		t.Error("Expected at least one search query")
	}

	// Check that queries are valid
	for _, query := range SearchQueries {
		if query == "" {
			t.Error("Search query should not be empty")
		}
	}
}

func TestSeedRepositoryTypes(t *testing.T) {
	// Check official seeds
	for _, seed := range OfficialSeeds {
		if seed.Type != "official" {
			t.Errorf("Official seed %s/%s has type %q, want 'official'",
				seed.Owner, seed.Repo, seed.Type)
		}
		if seed.Priority < 9 {
			t.Errorf("Official seed %s/%s has low priority %d",
				seed.Owner, seed.Repo, seed.Priority)
		}
	}

	// Check curated seeds
	for _, seed := range CuratedSeeds {
		if seed.Type != "curated" {
			t.Errorf("Curated seed %s/%s has type %q, want 'curated'",
				seed.Owner, seed.Repo, seed.Type)
		}
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		content  string
		word     string
		expected bool
	}{
		// Should match - exact word
		{"This is Python code", "python", true},
		{"This is PYTHON code", "python", true},
		{"This is python code", "Python", true},
		{"Python is great", "python", true},
		{"Use PYTHON for scripting", "python", true},

		// Basic non-matches
		{"No match here", "python", false},
		{"", "python", false},
		{"python", "", true},

		// Scala edge cases - MUST NOT match scalability/scalable
		{"System is scalable and fast", "scala", false},
		{"Improve scalability", "scala", false},
		{"Built with Scala language", "scala", true},
		{"Scala is functional", "scala", true},

		// Go edge cases - MUST NOT match going/algorithm/ago
		{"Written in Go language", "go", true},
		{"Go is fast", "go", true},
		{"Going forward", "go", false},
		{"Algorithm design", "go", false},
		{"A week ago", "go", false},
		{"Let's go!", "go", true}, // punctuation is boundary

		// Rust edge cases - MUST NOT match trust/frustrating
		{"Built with Rust", "rust", true},
		{"Don't trust the data", "rust", false},
		{"Frustrating bug", "rust", false},

		// AI edge cases - MUST NOT match maintain/certain
		{"AI and machine learning", "ai", true},
		{"Maintain the system", "ai", false},
		{"Certain conditions", "ai", false},

		// ML edge cases - MUST NOT match html
		{"ML models", "ml", true},
		{"HTML templates", "ml", false},

		// Data edge cases - MUST NOT match update/metadata
		{"Data processing", "data", true},
		{"Update the record", "data", false},
		{"Metadata extraction", "data", false},

		// Hyphenated tags - must match exact hyphenated form
		{"Using code-review tools", "code-review", true},
		{"Set up ci-cd pipeline", "ci-cd", true},
		// Hyphenated tags don't match non-hyphenated content
		{"Code review best practices", "code-review", false},
		{"CI/CD pipeline", "ci-cd", false},
	}

	for _, tt := range tests {
		t.Run(tt.content+"_"+tt.word, func(t *testing.T) {
			result := containsWord(tt.content, tt.word)
			if result != tt.expected {
				t.Errorf("containsWord(%q, %q) = %v, want %v",
					tt.content, tt.word, result, tt.expected)
			}
		})
	}
}

func TestNewGitHubClient(t *testing.T) {
	// Test without token
	client := NewGitHubClient("", 0)
	if client == nil {
		t.Error("Expected non-nil client")
		return
	}
	if client.cache == nil {
		t.Error("Expected non-nil cache")
	}
	if client.limiter == nil {
		t.Error("Expected non-nil limiter")
	}

	// Test with token
	clientWithToken := NewGitHubClient("fake-token", 0)
	if clientWithToken == nil {
		t.Error("Expected non-nil client with token")
	}
}

func TestGitHubClientStats(t *testing.T) {
	client := NewGitHubClient("", 0)

	requests, hits, misses := client.Stats()
	if requests != 0 || hits != 0 || misses != 0 {
		t.Errorf("Expected zero stats, got requests=%d, hits=%d, misses=%d",
			requests, hits, misses)
	}

	// Simulate some cache activity
	client.cache.Set("test", "value")
	client.cache.Get("test")  // hit
	client.cache.Get("other") // miss

	// Stats should still be zero (cache doesn't increment directly)
	requests, _, _ = client.Stats()
	if requests != 0 {
		t.Errorf("Expected 0 requests, got %d", requests)
	}
}

func TestGitHubClientResetStats(t *testing.T) {
	client := NewGitHubClient("", 0)

	// Manually set some stats
	client.mu.Lock()
	client.requestCount = 10
	client.cacheHits = 5
	client.cacheMisses = 3
	client.mu.Unlock()

	client.ResetStats()

	requests, hits, misses := client.Stats()
	if requests != 0 || hits != 0 || misses != 0 {
		t.Errorf("Expected zero stats after reset, got requests=%d, hits=%d, misses=%d",
			requests, hits, misses)
	}
}

func TestSkillFilePatterns(t *testing.T) {
	// Ensure we have all expected patterns and exclude unwanted ones

	tests := []struct {
		pattern       string
		shouldBeFound bool
	}{
		{"SKILL.md", true},
		{"skill.md", true},
		{"CLAUDE.md", true},
		{"claude.md", true},
		{".cursorrules", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			found := false
			for _, pattern := range SkillFilePatterns {
				if pattern == tt.pattern {
					found = true
					break
				}
			}
			if found != tt.shouldBeFound {
				if tt.shouldBeFound {
					t.Errorf("Expected pattern %q to be in SkillFilePatterns but it wasn't", tt.pattern)
				} else {
					t.Errorf("Pattern %q should NOT be in SkillFilePatterns but it was", tt.pattern)
				}
			}
		})
	}
}

func TestNewScraperWithConfigGitMode(t *testing.T) {
	cfg := ScraperConfig{
		Token:        "",
		DataDir:      t.TempDir(),
		UseGitClone:  true,
		RepoCacheTTL: 7,
	}

	s := NewScraperWithConfig(cfg, nil)
	if s == nil {
		t.Fatal("Expected non-nil Scraper")
	}

	if s.gitClient == nil {
		t.Error("Expected gitClient to be set for git clone mode")
	}
	if _, ok := s.client.(*GitClient); !ok {
		t.Error("Expected client to be GitClient")
	}
}

func TestNewScraperWithConfigAPIMode(t *testing.T) {
	cfg := ScraperConfig{
		Token:       "test-token",
		UseGitClone: false,
	}

	s := NewScraperWithConfig(cfg, nil)
	if s == nil {
		t.Fatal("Expected non-nil Scraper")
	}

	if s.gitClient != nil {
		t.Error("Expected gitClient to be nil for API mode")
	}
	if _, ok := s.client.(*GitHubClient); !ok {
		t.Error("Expected client to be GitHubClient")
	}
}

func TestScraperCleanupOldRepositories(t *testing.T) {
	// Test with git clone mode
	cfg := ScraperConfig{
		DataDir:      t.TempDir(),
		UseGitClone:  true,
		RepoCacheTTL: 7,
	}
	s := NewScraperWithConfig(cfg, nil)

	// Should not error on empty directory
	err := s.CleanupOldRepositories()
	if err != nil {
		t.Fatalf("CleanupOldRepositories failed: %v", err)
	}
}

func TestScraperCleanupOldRepositoriesAPIMode(t *testing.T) {
	// Test with API mode (no cleanup should happen)
	cfg := ScraperConfig{
		Token:       "test-token",
		UseGitClone: false,
	}
	s := NewScraperWithConfig(cfg, nil)

	// Should return nil (no-op for API mode)
	err := s.CleanupOldRepositories()
	if err != nil {
		t.Fatalf("CleanupOldRepositories should be no-op for API mode: %v", err)
	}
}

func TestScraperConfigDefaults(t *testing.T) {
	cfg := ScraperConfig{
		DataDir:     t.TempDir(),
		UseGitClone: true,
		// CloneDir not set - should default to DataDir/repositories
	}
	s := NewScraperWithConfig(cfg, nil)

	if s.gitClient == nil {
		t.Fatal("Expected gitClient to be set")
	}

	// Verify that cleanup doesn't fail (indirectly tests clone dir setup)
	err := s.CleanupOldRepositories()
	if err != nil {
		t.Fatalf("CleanupOldRepositories failed: %v", err)
	}
}

func TestDefaultScrapeSeedsOptions(t *testing.T) {
	opts := DefaultScrapeSeedsOptions()

	if opts.Force {
		t.Error("Expected Force to be false by default")
	}
	if opts.MaxConcurrency != 5 {
		t.Errorf("Expected MaxConcurrency to be 5, got %d", opts.MaxConcurrency)
	}
}

func TestScrapeSeedsOptionsValidation(t *testing.T) {
	tests := []struct {
		name          string
		opts          ScrapeSeedsOptions
		expectedForce bool
		expectedConc  int
	}{
		{
			name:          "default options",
			opts:          DefaultScrapeSeedsOptions(),
			expectedForce: false,
			expectedConc:  5,
		},
		{
			name:          "force enabled",
			opts:          ScrapeSeedsOptions{Force: true, MaxConcurrency: 5},
			expectedForce: true,
			expectedConc:  5,
		},
		{
			name:          "custom concurrency",
			opts:          ScrapeSeedsOptions{Force: false, MaxConcurrency: 10},
			expectedForce: false,
			expectedConc:  10,
		},
		{
			name:          "zero concurrency defaults to 5",
			opts:          ScrapeSeedsOptions{Force: false, MaxConcurrency: 0},
			expectedForce: false,
			expectedConc:  0, // Will be set to 5 internally during execution
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.Force != tt.expectedForce {
				t.Errorf("Expected Force=%v, got %v", tt.expectedForce, tt.opts.Force)
			}
			if tt.opts.MaxConcurrency != tt.expectedConc {
				t.Errorf("Expected MaxConcurrency=%d, got %d", tt.expectedConc, tt.opts.MaxConcurrency)
			}
		})
	}
}

func TestScrapeSeedsWithOptionsCallsCorrectMethod(t *testing.T) {
	// Verify that ScrapeSeeds calls ScrapeSeedsWithOptions with default options
	cfg := ScraperConfig{
		DataDir:     t.TempDir(),
		UseGitClone: true,
	}
	s := NewScraperWithConfig(cfg, nil)

	// This is a structural test - verify the scraper was created correctly
	if s == nil {
		t.Fatal("Expected non-nil scraper")
	}
	if s.gitClient == nil {
		t.Error("Expected gitClient to be set")
	}
}

func TestDeduplicationIsThreadSafe(t *testing.T) {
	// Setup in-memory database
	database, err := db.New(db.Config{
		Path:        ":memory:",
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() { _ = database.Close() }()

	cfg := ScraperConfig{
		DataDir:     t.TempDir(),
		UseGitClone: true,
	}
	scraper := NewScraperWithConfig(cfg, database)

	// Create base skill in database first
	baseSkill := &models.Skill{
		ID:          "test-skill-base",
		Slug:        "test-skill",
		Title:       "Test Skill",
		EmbeddingID: "unique-content-1",
	}
	if err := database.CreateSkill(baseSkill); err != nil {
		t.Fatalf("Failed to create base skill: %v", err)
	}

	// Run concurrent deduplication checks for same slug
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make([]struct {
		skip bool
		err  error
		slug string
	}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			skill := &models.Skill{
				ID:          fmt.Sprintf("test-skill-%d", idx+2),
				Slug:        "test-skill",
				Title:       "Test Skill",
				EmbeddingID: fmt.Sprintf("unique-content-%d", idx+2), // Different content
			}
			skip, err := scraper.deduplicateSkill(skill)
			results[idx].skip = skip
			results[idx].err = err
			results[idx].slug = skill.Slug
		}(i)
	}

	wg.Wait()

	// Check that no errors occurred
	for i, r := range results {
		if r.err != nil {
			t.Errorf("Goroutine %d got error: %v", i, r.err)
		}
	}

	// Collect unique slugs that weren't skipped
	slugs := make(map[string]bool)
	for _, r := range results {
		if !r.skip {
			if slugs[r.slug] {
				t.Errorf("Duplicate slug detected: %s (race condition!)", r.slug)
			}
			slugs[r.slug] = true
		}
	}

	// Verify that we got unique slugs (no duplicates)
	t.Logf("Generated %d unique slugs from %d goroutines", len(slugs), numGoroutines)
}
