package scraper

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
)

// LicenseInfo holds detected license information for a repository.
type RepositoryLicenseInfo struct {
	Type string // SPDX identifier (e.g., "MIT", "Apache-2.0")
	URL  string // Direct link to LICENSE file in GitHub
	File string // The LICENSE file name found
}

// ScrapeResult contains the results of a scraping operation.
type ScrapeResult struct {
	SourcesProcessed int
	SourcesSkipped   int
	SkillsFound      int
	SkillsNew        int
	SkillsUpdated    int
	Errors           []error
	Duration         time.Duration
}

// ScraperStats contains scraper statistics.
type ScraperStats struct {
	APIRequests   int
	CacheHits     int
	CacheMisses   int
	SkillsIndexed int64
	SourcesCount  int64
	LastSyncAt    time.Time
}

// Client defines the interface for repository data fetching.
// Both GitHubClient (API-based) and GitClient (git clone-based) implement this.
type Client interface {
	GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepoInfo, error)
	ListSkillFiles(ctx context.Context, owner, repo, path string) ([]*SkillFile, error)
	GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error)
	GetLicenseFile(ctx context.Context, owner, repo, ref string) (fileName, licenseURL, rawURL string, err error)
	Stats() (requests, cacheHits, cacheMisses int)
	ResetStats()
	ClearCache()
}

// ScraperConfig holds configuration for the scraper.
type ScraperConfig struct {
	Token        string // GitHub token (optional for public repos with git clone)
	DataDir      string // Base data directory (~/.skulto), repositories cloned to DataDir/repositories
	RepoCacheTTL int    // Days to keep cloned repos
	UseGitClone  bool   // Use git clone instead of GitHub API
}

// Scraper orchestrates the GitHub scraping pipeline.
type Scraper struct {
	client       Client
	gitClient    *GitClient // Keep reference for cleanup operations
	parser       *SkillParser
	db           *db.DB
	config       ScraperConfig
	dedupMutex   sync.Mutex        // Protects deduplication checks to prevent race conditions
	claimedSlugs map[string]string // Maps slug -> skillID for pending inserts (not yet in DB)
}

// NewScraperWithConfig creates a new scraper with full configuration support.
func NewScraperWithConfig(cfg ScraperConfig, database *db.DB) *Scraper {
	s := &Scraper{
		parser:       NewSkillParser(),
		db:           database,
		config:       cfg,
		claimedSlugs: make(map[string]string),
	}

	// Clone directory is always DataDir/repositories
	cloneDir := ""
	if cfg.DataDir != "" {
		cloneDir = filepath.Join(cfg.DataDir, "repositories")
	}

	// Choose client based on configuration
	if cfg.UseGitClone {
		gitClient := NewGitClient(cfg.Token, cloneDir)
		s.client = gitClient
		s.gitClient = gitClient
	} else {
		s.client = NewGitHubClient(cfg.Token, 0)
	}

	return s
}

// CleanupOldRepositories removes repositories that haven't been accessed recently.
// Only works when using git clone-based scraping.
func (s *Scraper) CleanupOldRepositories() error {
	if s.gitClient == nil {
		return nil // Not using git clone, nothing to clean up
	}

	maxAge := time.Duration(s.config.RepoCacheTTL) * 24 * time.Hour
	if maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour // Default to 7 days
	}

	return s.gitClient.Cleanup(maxAge)
}

// ProgressCallback is called to report scraping progress.
// completed is the number of repositories finished, total is the total count,
// and repoName is the name of the repository that just completed (empty if just starting).
type ProgressCallback func(completed, total int, repoName string)

// ScrapeSeedsOptions configures the ScrapeSeeds behavior.
type ScrapeSeedsOptions struct {
	// Force bypasses the commit SHA check and re-scans all skills
	Force bool
	// MaxConcurrency limits the number of parallel repository scrapes (default: 5)
	MaxConcurrency int
	// OnProgress is called after each repository completes (thread-safe)
	OnProgress ProgressCallback
}

// DefaultScrapeSeedsOptions returns sensible defaults.
func DefaultScrapeSeedsOptions() ScrapeSeedsOptions {
	return ScrapeSeedsOptions{
		Force:          false,
		MaxConcurrency: 5,
		OnProgress:     nil,
	}
}

// ScrapeSeeds scrapes all seed repositories with default options.
func (s *Scraper) ScrapeSeeds(ctx context.Context) (*ScrapeResult, error) {
	return s.ScrapeSeedsWithOptions(ctx, DefaultScrapeSeedsOptions())
}

// ScrapeSeedsWithOptions scrapes all seed repositories with the given options.
// Uses parallel goroutines with a semaphore for concurrency control.
func (s *Scraper) ScrapeSeedsWithOptions(ctx context.Context, opts ScrapeSeedsOptions) (*ScrapeResult, error) {
	start := time.Now()
	result := &ScrapeResult{}

	seeds := AllSeeds()
	if len(seeds) == 0 {
		result.Duration = time.Since(start)
		return result, nil
	}

	total := len(seeds)

	// Set default concurrency
	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}

	// Report initial progress
	if opts.OnProgress != nil {
		opts.OnProgress(0, total, "")
	}

	// Atomic counter for completed repos
	var completed int32

	// Create semaphore for concurrency control
	sem := make(chan struct{}, maxConcurrency)

	// Channel for collecting results
	type repoResult struct {
		seed   SeedRepository
		result *ScrapeResult
		err    error
	}
	results := make(chan repoResult, len(seeds))

	// WaitGroup to track completion
	var wg sync.WaitGroup

	// Launch goroutines for each seed
	for _, seed := range seeds {
		wg.Add(1)
		go func(seed SeedRepository) {
			defer wg.Done()

			// Check context before acquiring semaphore
			select {
			case <-ctx.Done():
				results <- repoResult{seed: seed, err: ctx.Err()}
				return
			case sem <- struct{}{}: // Acquire semaphore
				defer func() { <-sem }() // Release on exit
			}

			// Check context again after acquiring
			select {
			case <-ctx.Done():
				results <- repoResult{seed: seed, err: ctx.Err()}
				return
			default:
			}

			// Scrape the repository
			res, err := s.scrapeRepositoryWithOptions(ctx, seed.Owner, seed.Repo, opts.Force)

			// Report progress after completion
			if opts.OnProgress != nil {
				c := int(atomic.AddInt32(&completed, 1))
				repoName := fmt.Sprintf("%s/%s", seed.Owner, seed.Repo)
				opts.OnProgress(c, total, repoName)
			}

			results <- repoResult{seed: seed, result: res, err: err}
		}(seed)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for res := range results {
		if res.err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s/%s: %w", res.seed.Owner, res.seed.Repo, res.err))
			continue
		}
		if res.result != nil {
			result.SourcesProcessed++
			result.SourcesSkipped += res.result.SourcesSkipped
			result.SkillsFound += res.result.SkillsFound
			result.SkillsNew += res.result.SkillsNew
			result.SkillsUpdated += res.result.SkillsUpdated
		}
	}

	result.Duration = time.Since(start)

	// Update sync metadata
	_ = s.db.SetSyncMeta(models.SyncMetaLastFullSync, time.Now().Format(time.RFC3339))

	return result, nil
}

// ScrapeRepository scrapes a single repository for skill files.
func (s *Scraper) ScrapeRepository(ctx context.Context, owner, repo string) (*ScrapeResult, error) {
	return s.scrapeRepositoryWithOptions(ctx, owner, repo, false)
}

// scrapeRepositoryWithOptions scrapes a single repository with optional force re-scan.
// If force is true, the commit SHA check is bypassed and all skills are re-scanned.
func (s *Scraper) scrapeRepositoryWithOptions(ctx context.Context, owner, repo string, force bool) (*ScrapeResult, error) {
	repoStart := time.Now()
	result := &ScrapeResult{}

	// Get repository info
	repoInfo, err := s.client.GetRepositoryInfo(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get repo info: %w", err)
	}

	// Fetch license information for the repository (once, not per-skill)
	licenseInfo := s.fetchRepositoryLicense(ctx, owner, repo, repoInfo.DefaultBranch)

	// Create/update source in database
	source := &models.Source{
		ID:            fmt.Sprintf("%s/%s", owner, repo),
		Owner:         owner,
		Repo:          repo,
		FullName:      repoInfo.FullName,
		Description:   repoInfo.Description,
		URL:           fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		CloneURL:      repoInfo.CloneURL,
		Stars:         repoInfo.Stars,
		Forks:         repoInfo.Forks,
		Watchers:      repoInfo.Watchers,
		DefaultBranch: repoInfo.DefaultBranch,
		LastCommitSHA: repoInfo.CommitSHA,
		LicenseType:   licenseInfo.Type,
		LicenseURL:    licenseInfo.URL,
		LicenseFile:   licenseInfo.File,
	}

	// Determine source type
	for _, seed := range OfficialSeeds {
		if seed.Owner == owner && seed.Repo == repo {
			source.IsOfficial = true
			source.Priority = seed.Priority
			break
		}
	}
	for _, seed := range CuratedSeeds {
		if seed.Owner == owner && seed.Repo == repo {
			source.IsCurated = true
			source.Priority = seed.Priority
			break
		}
	}

	// Check if source already exists with the same commit (skip check if force=true)
	existingSource, err := s.db.GetSource(source.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, fmt.Errorf("get source: %w", err)
	}

	// If source exists and commit hasn't changed, skip scraping (unless force=true)
	if !force && existingSource != nil && existingSource.LastCommitSHA == repoInfo.CommitSHA && repoInfo.CommitSHA != "" {
		result.SourcesSkipped = 1
		// Still update the metadata
		now := time.Now()
		source.LastScrapedAt = &now
		// Preserve existing skill count
		source.SkillCount = existingSource.SkillCount
		if err := s.db.UpsertSource(source); err != nil {
			return nil, fmt.Errorf("update source metadata: %w", err)
		}
		result.Duration = time.Since(repoStart)
		return result, nil
	}

	if err := s.db.UpsertSource(source); err != nil {
		return nil, fmt.Errorf("upsert source: %w", err)
	}

	result.SourcesProcessed = 1

	// Find skill path for this seed
	skillPath := ""
	for _, seed := range AllSeeds() {
		if seed.Owner == owner && seed.Repo == repo {
			skillPath = seed.SkillPath
			break
		}
	}

	// List skill files in repository
	skillFiles, err := s.client.ListSkillFiles(ctx, owner, repo, skillPath)
	if err != nil {
		return nil, fmt.Errorf("list skill files: %w", err)
	}

	result.SkillsFound = len(skillFiles)

	// Batch process skills using a transaction for better performance
	type skillData struct {
		skill    *models.Skill
		tags     []models.Tag
		existing *models.Skill
	}
	var skillBatch []skillData

	// First pass: parse all skills
	for _, sf := range skillFiles {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Check if skill already exists
		existing, err := s.db.GetSkill(sf.ID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("get skill %s: %w", sf.ID, err))
			continue
		}

		// Get file content
		content, err := s.client.GetFileContent(ctx, owner, repo, sf.Path, "")
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("get content %s: %w", sf.Path, err))
			continue
		}

		// Parse skill
		skill, err := s.parser.Parse(content, sf)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("parse %s: %w", sf.Path, err))
			continue
		}

		// Check for duplicate slug with different content
		isDuplicate, err := s.deduplicateSkill(skill)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("dedupe %s: %w", sf.Path, err))
			continue
		}
		if isDuplicate {
			// Skip - identical content already exists
			continue
		}

		// Copy repo metadata to skill
		skill.Stars = repoInfo.Stars
		skill.Forks = repoInfo.Forks

		// Use source author (repository owner) as skill author if not already set
		if skill.Author == "" {
			skill.Author = owner
		}

		// Extract tags from content
		tags := ExtractTags(content)

		skillBatch = append(skillBatch, skillData{
			skill:    skill,
			tags:     tags,
			existing: existing,
		})
	}

	// Second pass: batch upsert skills in a transaction
	if len(skillBatch) > 0 {
		err := s.db.Transaction(func(tx *db.DB) error {
			for _, sd := range skillBatch {
				if err := tx.UpsertSkillWithTags(sd.skill, sd.tags); err != nil {
					return fmt.Errorf("upsert skill %s: %w", sd.skill.ID, err)
				}
			}
			return nil
		})
		if err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			// Count new vs updated
			for _, sd := range skillBatch {
				if sd.existing == nil {
					result.SkillsNew++
				} else {
					result.SkillsUpdated++
				}
			}
		}
	}

	// Update source skill count
	if err := s.db.UpdateSourceSkillCount(source.ID); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("update skill count: %w", err))
	}

	// Update last scraped time
	now := time.Now()
	source.LastScrapedAt = &now
	if err := s.db.UpsertSource(source); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("update last scraped: %w", err))
	}

	result.Duration = time.Since(repoStart)

	return result, nil
}

// ScrapeSearch runs search queries to discover new skill repositories.
// Note: This only works with GitHubClient (API-based). When using GitClient
// (git clone-based), this returns an empty result as search requires the GitHub API.
func (s *Scraper) ScrapeSearch(ctx context.Context) (*ScrapeResult, error) {
	start := time.Now()
	result := &ScrapeResult{}

	// Search only works with GitHubClient
	githubClient, ok := s.client.(*GitHubClient)
	if !ok {
		// Using GitClient - search is not available
		result.Duration = time.Since(start)
		return result, nil
	}

	// Track unique repos to avoid duplicates
	seen := make(map[string]bool)

	for _, query := range SearchQueries {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		files, err := githubClient.SearchSkillFiles(ctx, query)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("search %q: %w", query, err))
			continue
		}

		for _, sf := range files {
			if seen[sf.RepoName] {
				continue
			}
			seen[sf.RepoName] = true

			// Scrape the discovered repository
			res, err := s.ScrapeRepository(ctx, sf.Owner, sf.Repo)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", sf.RepoName, err))
				continue
			}

			result.SourcesProcessed++
			result.SkillsFound += res.SkillsFound
			result.SkillsNew += res.SkillsNew
			result.SkillsUpdated += res.SkillsUpdated
		}
	}

	result.Duration = time.Since(start)

	return result, nil
}

// Sync performs a full sync with delta support.
func (s *Scraper) Sync(ctx context.Context) (*ScrapeResult, error) {
	start := time.Now()
	result := &ScrapeResult{}

	// First, scrape all seeds
	seedResult, err := s.ScrapeSeeds(ctx)
	if err != nil {
		return nil, err
	}

	result.SourcesProcessed += seedResult.SourcesProcessed
	result.SkillsFound += seedResult.SkillsFound
	result.SkillsNew += seedResult.SkillsNew
	result.SkillsUpdated += seedResult.SkillsUpdated
	result.Errors = append(result.Errors, seedResult.Errors...)

	// Then run search discovery (if we have API quota left)
	requests, _, _ := s.client.Stats()
	if requests < 100 { // Conservative limit
		searchResult, err := s.ScrapeSearch(ctx)
		if err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.SourcesProcessed += searchResult.SourcesProcessed
			result.SkillsFound += searchResult.SkillsFound
			result.SkillsNew += searchResult.SkillsNew
			result.SkillsUpdated += searchResult.SkillsUpdated
			result.Errors = append(result.Errors, searchResult.Errors...)
		}
	}

	result.Duration = time.Since(start)

	// Update sync metadata
	_ = s.db.SetSyncMeta(models.SyncMetaLastFullSync, time.Now().Format(time.RFC3339))

	// Update total skills count
	stats, _ := s.db.GetStats()
	if stats != nil {
		_ = s.db.SetSyncMeta(models.SyncMetaTotalSkills, fmt.Sprintf("%d", stats.TotalSkills))
	}

	return result, nil
}

// deduplicateSkill checks for existing skills with the same slug.
// Returns true if the skill should be skipped (identical content exists).
// If different content exists with same slug, appends a number to make it unique.
// Thread-safe: uses mutex and in-memory claimed slugs map to prevent race conditions
// when called from parallel goroutines during batch processing.
func (s *Scraper) deduplicateSkill(skill *models.Skill) (bool, error) {
	s.dedupMutex.Lock()
	defer s.dedupMutex.Unlock()

	baseSlug := skill.Slug
	suffix := 1

	for {
		// Check if slug is already claimed by another pending skill (not yet in DB)
		if claimedByID, claimed := s.claimedSlugs[skill.Slug]; claimed && claimedByID != skill.ID {
			// Slug is claimed by another skill - try next suffix
			suffix++
			skill.Slug = fmt.Sprintf("%s-%d", baseSlug, suffix)
			if suffix > 100 {
				return false, fmt.Errorf("too many slug collisions for %s", baseSlug)
			}
			continue
		}

		existing, err := s.db.GetSkillBySlug(skill.Slug)
		if err != nil {
			return false, err
		}

		if existing == nil {
			// No conflict - slug is unique, claim it
			s.claimedSlugs[skill.Slug] = skill.ID
			return false, nil
		}

		// Check if it's the same skill (same ID)
		if existing.ID == skill.ID {
			// Same skill, just updating
			s.claimedSlugs[skill.Slug] = skill.ID
			return false, nil
		}

		// Different skill with same slug - check content
		if existing.EmbeddingID == skill.EmbeddingID {
			// Identical content - skip this duplicate
			return true, nil
		}

		// Different content - try next suffix
		suffix++
		skill.Slug = fmt.Sprintf("%s-%d", baseSlug, suffix)

		// Safety limit to prevent infinite loop
		if suffix > 100 {
			return false, fmt.Errorf("too many slug collisions for %s", baseSlug)
		}
	}
}

// RetagAll re-extracts tags for all skills in the database.
// This should be called after updating the tagging algorithm.
func (s *Scraper) RetagAll(ctx context.Context) (int, error) {
	return s.db.RetagAllSkills(ExtractTags)
}

// Stats returns scraper and client statistics.
func (s *Scraper) Stats() ScraperStats {
	requests, hits, misses := s.client.Stats()

	stats := ScraperStats{
		APIRequests: requests,
		CacheHits:   hits,
		CacheMisses: misses,
	}

	// Get database stats
	dbStats, err := s.db.GetStats()
	if err == nil && dbStats != nil {
		stats.SkillsIndexed = dbStats.TotalSkills
		stats.SourcesCount = dbStats.TotalSources
	}

	// Get last sync time
	if lastSync, err := s.db.GetSyncMeta(models.SyncMetaLastFullSync); err == nil && lastSync != "" {
		if t, err := time.Parse(time.RFC3339, lastSync); err == nil {
			stats.LastSyncAt = t
		}
	}

	return stats
}

// ExtractTags extracts tags from skill content based on keyword matching.
func ExtractTags(content string) []models.Tag {
	var tags []models.Tag
	seen := make(map[string]bool)

	// Check for each predefined tag
	for category, tagNames := range models.PredefinedTags {
		for _, name := range tagNames {
			if seen[name] {
				continue
			}

			// Simple case-insensitive substring match
			if containsWord(content, name) {
				tags = append(tags, models.Tag{
					ID:       name,
					Name:     name,
					Slug:     name,
					Category: string(category),
					Color:    models.TagColors[category],
				})
				seen[name] = true
			}
		}
	}

	// Sort tags by category for consistency
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Category != tags[j].Category {
			return tags[i].Category < tags[j].Category
		}
		return tags[i].Name < tags[j].Name
	})

	return tags
}

// tagPatterns caches compiled regex patterns for performance.
var tagPatterns = make(map[string]*regexp.Regexp)
var tagPatternsMu sync.RWMutex

// getTagPattern returns a compiled regex for word boundary matching.
// Patterns are cached for reuse.
func getTagPattern(word string) *regexp.Regexp {
	tagPatternsMu.RLock()
	if pattern, ok := tagPatterns[word]; ok {
		tagPatternsMu.RUnlock()
		return pattern
	}
	tagPatternsMu.RUnlock()

	// Escape special regex characters in the word
	escaped := regexp.QuoteMeta(word)
	// Use (?i) for case-insensitive and \b for word boundaries
	pattern := regexp.MustCompile(`(?i)\b` + escaped + `\b`)

	tagPatternsMu.Lock()
	tagPatterns[word] = pattern
	tagPatternsMu.Unlock()

	return pattern
}

// containsWord checks if content contains the word as a complete word (not substring).
// Uses regex word boundaries to avoid false positives like "scala" matching "scalability".
func containsWord(content, word string) bool {
	if word == "" {
		return true // Empty word matches everything (backward compat)
	}
	pattern := getTagPattern(word)
	return pattern.MatchString(content)
}

// fetchRepositoryLicense fetches and detects license information for a repository.
// Returns empty RepositoryLicenseInfo if license not found or on error (non-blocking).
func (s *Scraper) fetchRepositoryLicense(ctx context.Context, owner, repo, ref string) RepositoryLicenseInfo {
	// Fetch LICENSE file from repository
	fileName, licenseURL, _, err := s.client.GetLicenseFile(ctx, owner, repo, ref)
	if err != nil {
		// Gracefully handle errors - license is optional
		return RepositoryLicenseInfo{}
	}

	// If no license file found, return empty
	if fileName == "" {
		return RepositoryLicenseInfo{}
	}

	// Fetch the license content and detect type
	licenseContent, err := s.client.GetFileContent(ctx, owner, repo, fileName, ref)
	if err != nil {
		// Gracefully handle errors - we still have the file name and URL
		return RepositoryLicenseInfo{
			File: fileName,
			URL:  licenseURL,
			Type: "Unknown",
		}
	}

	// Detect license type from content
	detectedType := DetectLicenseType(licenseContent)

	return RepositoryLicenseInfo{
		Type: detectedType,
		URL:  licenseURL,
		File: fileName,
	}
}

// GetDirectoryContents fetches the contents of an optional directory from a repository.
// Returns nil if the directory doesn't exist (not an error).
// Uses cached git data when available to avoid API rate limits.
// Supports nested subdirectories with recursive traversal.
func (s *Scraper) GetDirectoryContents(ctx context.Context, owner, repo, dirPath string, dirName models.OptionalDirName) (*models.OptionalDir, error) {
	// Only works with GitClient
	if s.gitClient == nil {
		return nil, nil // No git client, skip
	}

	optDir := &models.OptionalDir{
		Name:  dirName,
		Files: make([]models.OptionalFile, 0),
	}

	// Recursively collect files from the directory and all subdirectories
	if err := s.collectDirectoryFilesRecursively(ctx, owner, repo, dirPath, "", optDir); err != nil {
		return nil, err
	}

	if len(optDir.Files) == 0 {
		return nil, nil
	}

	return optDir, nil
}

// collectDirectoryFilesRecursively traverses a directory and all subdirectories,
// collecting files while maintaining relative paths.
// dirPath is the absolute path in the repository, relPath is the relative path within the optional dir.
func (s *Scraper) collectDirectoryFilesRecursively(ctx context.Context, owner, repo, dirPath, relPath string, optDir *models.OptionalDir) error {
	// List files in the current directory
	entries, err := s.gitClient.ListDirectoryContents(ctx, owner, repo, dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Build full paths
		var currentDirPath, currentRelPath string
		if dirPath == "" {
			currentDirPath = entry.Name
		} else {
			currentDirPath = dirPath + "/" + entry.Name
		}

		if relPath == "" {
			currentRelPath = entry.Name
		} else {
			currentRelPath = relPath + "/" + entry.Name
		}

		if entry.IsDir {
			// Recursively process subdirectory
			if err := s.collectDirectoryFilesRecursively(ctx, owner, repo, currentDirPath, currentRelPath, optDir); err != nil {
				// Log but don't fail - continue processing other files/dirs
				continue
			}
		} else {
			// Process file
			// Check file size limit
			if entry.Size > models.MaxOptionalFileSize {
				continue
			}

			content, size, err := s.gitClient.ReadFileBytes(ctx, owner, repo, currentDirPath)
			if err != nil {
				continue // Skip files we can't read
			}

			optDir.Files = append(optDir.Files, models.OptionalFile{
				Name:    entry.Name,
				Path:    currentRelPath, // Relative path within the optional dir (may include subdirs)
				Content: content,
				Size:    size,
			})
		}
	}

	return nil
}
