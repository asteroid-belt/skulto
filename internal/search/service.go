package search

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/vector"
)

// Service provides unified search functionality combining FTS5 and semantic search.
type Service struct {
	db     *db.DB
	store  vector.VectorStore
	config Config
}

// Config holds search service configuration.
type Config struct {
	MinSimilarity float32
	MaxResults    int
	MaxSnippets   int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MinSimilarity: 0.6,
		MaxResults:    50,
		MaxSnippets:   3,
	}
}

// New creates a new search service.
// The store parameter can be nil if semantic search is not available.
func New(database *db.DB, store vector.VectorStore, cfg Config) *Service {
	if cfg.MinSimilarity <= 0 {
		cfg.MinSimilarity = 0.6
	}
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = 50
	}
	if cfg.MaxSnippets <= 0 {
		cfg.MaxSnippets = 3
	}

	return &Service{
		db:     database,
		store:  store,
		config: cfg,
	}
}

// Search performs hybrid search combining FTS5 and semantic search.
// Errors in one search method don't fail the entire search - graceful degradation.
func (s *Service) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResults, error) {
	start := time.Now()

	results := &SearchResults{
		Query:          query,
		TitleMatches:   []SkillMatch{},
		ContentMatches: []SkillMatch{},
	}

	// Track seen skill IDs to avoid duplicates
	seen := make(map[string]bool)

	// Semantic search (graceful degradation on error)
	if opts.IncludeSemantic && s.store != nil {
		hits, err := s.semanticSearch(ctx, query, opts)
		if err != nil {
			log.Printf("semantic search warning: %v (continuing with FTS only)", err)
		} else {
			s.categorizeResults(ctx, hits, results, query, seen)
		}
	}

	// FTS search (graceful degradation on error)
	if opts.IncludeFTS {
		ftsResults, err := s.db.SearchSkills(query, opts.Limit)
		if err != nil {
			log.Printf("FTS search warning: %v (continuing with semantic only)", err)
		} else {
			s.mergeWithFTS(ftsResults, results, query, seen)
		}
	}

	results.Duration = time.Since(start)
	results.TotalHits = len(results.TitleMatches) + len(results.ContentMatches)

	return results, nil
}

// semanticSearch performs vector similarity search.
func (s *Service) semanticSearch(ctx context.Context, query string, opts SearchOptions) ([]vector.SearchHit, error) {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = s.config.MinSimilarity
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = s.config.MaxResults
	}

	return s.store.Search(ctx, query, limit, threshold)
}

// categorizeResults categorizes semantic search hits into title vs content matches.
func (s *Service) categorizeResults(ctx context.Context, hits []vector.SearchHit, results *SearchResults, query string, seen map[string]bool) {
	queryLower := strings.ToLower(query)

	for _, hit := range hits {
		if seen[hit.SkillID] {
			continue
		}
		seen[hit.SkillID] = true

		// Fetch the full skill from database (includes tags)
		skill, err := s.db.GetSkill(hit.SkillID)
		if err != nil || skill == nil {
			continue
		}

		match := SkillMatch{
			Skill: *skill,
			Score: hit.Score,
		}

		// Determine if it's a title/tag or content match
		if isTitleOrTagMatch(skill, queryLower) {
			match.MatchType = MatchTypeTitle
			results.TitleMatches = append(results.TitleMatches, match)
		} else {
			match.MatchType = MatchTypeContent
			// Extract snippets from content
			content := buildSearchableContent(skill)
			match.Snippets = ExtractSnippets(content, query, s.config.MaxSnippets)
			results.ContentMatches = append(results.ContentMatches, match)
		}
	}
}

// mergeWithFTS adds FTS results that aren't already present from semantic search.
func (s *Service) mergeWithFTS(skills []models.Skill, results *SearchResults, query string, seen map[string]bool) {
	queryLower := strings.ToLower(query)

	for _, skill := range skills {
		if seen[skill.ID] {
			continue
		}
		seen[skill.ID] = true

		match := SkillMatch{
			Skill: skill,
			Score: 0.5, // Default score for FTS results
		}

		// Determine if it's a title, tag, or content match
		if isTitleOrTagMatch(&skill, queryLower) {
			match.MatchType = MatchTypeTitle
			results.TitleMatches = append(results.TitleMatches, match)
		} else {
			match.MatchType = MatchTypeContent
			content := buildSearchableContent(&skill)
			match.Snippets = ExtractSnippets(content, query, s.config.MaxSnippets)
			results.ContentMatches = append(results.ContentMatches, match)
		}
	}
}

// isTitleOrTagMatch checks if the query matches the skill title or any of its tags.
func isTitleOrTagMatch(skill *models.Skill, queryLower string) bool {
	// Check title
	titleLower := strings.ToLower(skill.Title)
	if strings.Contains(titleLower, queryLower) || containsAnyTerm(titleLower, queryLower) {
		return true
	}

	// Check tags
	for _, tag := range skill.Tags {
		tagNameLower := strings.ToLower(tag.Name)
		tagSlugLower := strings.ToLower(tag.Slug)
		if strings.Contains(tagNameLower, queryLower) || strings.Contains(tagSlugLower, queryLower) {
			return true
		}
		if containsAnyTerm(tagNameLower, queryLower) || containsAnyTerm(tagSlugLower, queryLower) {
			return true
		}
	}

	return false
}

// buildSearchableContent builds a searchable text from skill fields.
func buildSearchableContent(skill *models.Skill) string {
	var parts []string

	if skill.Description != "" {
		parts = append(parts, skill.Description)
	}
	if skill.Summary != "" {
		parts = append(parts, skill.Summary)
	}
	if skill.Content != "" {
		parts = append(parts, skill.Content)
	}

	return strings.Join(parts, " ")
}

// containsAnyTerm checks if text contains any word from the query.
func containsAnyTerm(text, query string) bool {
	terms := strings.Fields(query)
	for _, term := range terms {
		if len(term) >= 3 && strings.Contains(text, term) {
			return true
		}
	}
	return false
}

// IndexSkill indexes a skill in the vector store.
func (s *Service) IndexSkill(ctx context.Context, skill *models.Skill) error {
	if s.store == nil {
		return nil // Vector search disabled
	}

	hash, err := s.store.AddSkill(ctx, skill)
	if err != nil {
		return err
	}

	// Update skill with embedding reference if changed
	if skill.EmbeddingID != hash {
		skill.EmbeddingID = hash
		if err := s.db.UpdateSkill(skill); err != nil {
			return err
		}
	}

	return nil
}

// IndexSkillBatch indexes multiple skills in the vector store.
func (s *Service) IndexSkillBatch(ctx context.Context, skills []models.Skill) (int, []error) {
	if s.store == nil {
		return 0, nil
	}

	return s.store.AddSkillBatch(ctx, skills)
}

// Stats returns indexing statistics.
func (s *Service) Stats(ctx context.Context) (*IndexStats, error) {
	stats := &IndexStats{}

	// Get total skills from database
	dbStats, err := s.db.GetStats()
	if err != nil {
		return nil, err
	}
	stats.TotalSkills = dbStats.TotalSkills
	stats.LastIndexedAt = dbStats.LastUpdated

	// Get indexed count from vector store
	if s.store != nil {
		stats.VectorStoreReady = true
		indexed, err := s.store.Count(ctx)
		if err == nil {
			stats.IndexedSkills = indexed
		}
	}

	// Calculate pending
	pending, err := s.db.CountSkillsWithoutEmbedding()
	if err == nil {
		stats.PendingSkills = int64(pending)
	}

	return stats, nil
}

// HasVectorStore returns true if semantic search is available.
func (s *Service) HasVectorStore() bool {
	return s.store != nil
}
