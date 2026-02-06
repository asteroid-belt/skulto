package scraper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

const (
	// DefaultCacheTTL is the default TTL for cached responses.
	DefaultCacheTTL = 24 * time.Hour

	// AuthenticatedRateLimit is requests per minute with token.
	AuthenticatedRateLimit = 20

	// UnauthenticatedRateLimit is requests per minute without token.
	UnauthenticatedRateLimit = 5
)

// ResponseCache provides TTL-based caching for GitHub API responses.
type ResponseCache struct {
	data map[string]cacheEntry
	ttl  time.Duration
	mu   sync.RWMutex
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// NewResponseCache creates a new cache with the specified TTL.
func NewResponseCache(ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		data: make(map[string]cacheEntry),
		ttl:  ttl,
	}
}

// Get retrieves a value from the cache if it exists and hasn't expired.
func (c *ResponseCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in the cache.
func (c *ResponseCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache.
func (c *ResponseCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear removes all entries from the cache.
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry)
}

// Len returns the number of entries in the cache.
func (c *ResponseCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// RepoInfo contains repository metadata.
type RepoInfo struct {
	FullName      string
	Description   string
	Stars         int
	Forks         int
	Watchers      int
	DefaultBranch string
	CommitSHA     string
	UpdatedAt     time.Time
	License       string
	CloneURL      string
}

// GitHubClient wraps GitHub API with rate limiting and caching.
type GitHubClient struct {
	rest    *github.Client
	limiter *rate.Limiter
	cache   *ResponseCache
	mu      sync.RWMutex

	// Stats tracking
	requestCount int
	cacheHits    int
	cacheMisses  int
}

// NewGitHubClient creates a new GitHub client with authentication.
func NewGitHubClient(token string, rateLimit int) *GitHubClient {
	var httpClient *http.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	// Determine rate limit based on authentication
	if rateLimit <= 0 {
		if token != "" {
			rateLimit = AuthenticatedRateLimit
		} else {
			rateLimit = UnauthenticatedRateLimit
		}
	}

	// Create rate limiter: rateLimit requests per minute
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rateLimit)), rateLimit)

	client := github.NewClient(httpClient)

	return &GitHubClient{
		rest:    client,
		limiter: limiter,
		cache:   NewResponseCache(DefaultCacheTTL),
	}
}

// SearchSkillFiles searches GitHub for skill files matching the query.
func (c *GitHubClient) SearchSkillFiles(ctx context.Context, query string) ([]*SkillFile, error) {
	cacheKey := fmt.Sprintf("search:%s", query)

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		return cached.([]*SkillFile), nil
	}

	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	opts := &github.SearchOptions{
		Sort:  "indexed",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allFiles []*SkillFile

	for {
		c.mu.Lock()
		c.requestCount++
		c.mu.Unlock()

		result, resp, err := c.rest.Search.Code(ctx, query, opts)
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				return nil, fmt.Errorf("GitHub rate limit exceeded, retry after %v", resp.Rate.Reset.Time)
			}
			return nil, fmt.Errorf("search code: %w", err)
		}

		for _, item := range result.CodeResults {
			repoName := item.GetRepository().GetFullName()
			path := item.GetPath()

			sf := &SkillFile{
				ID:       generateSkillID(repoName, path),
				Path:     path,
				RepoName: repoName,
				Owner:    item.GetRepository().GetOwner().GetLogin(),
				Repo:     item.GetRepository().GetName(),
				URL:      item.GetHTMLURL(),
				SHA:      item.GetSHA(),
			}
			allFiles = append(allFiles, sf)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage

		// Rate limit between pagination
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}
	}

	// Cache results
	c.cache.Set(cacheKey, allFiles)

	return allFiles, nil
}

// GetFileContent fetches raw file content from a repository.
func (c *GitHubClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	cacheKey := fmt.Sprintf("content:%s/%s:%s@%s", owner, repo, path, ref)

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		return cached.(string), nil
	}

	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait: %w", err)
	}

	c.mu.Lock()
	c.requestCount++
	c.mu.Unlock()

	opts := &github.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}

	fileContent, _, resp, err := c.rest.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("get contents: %w", err)
	}

	// Check rate limit headers
	if resp.Rate.Remaining < 100 {
		log.Printf("github: rate limit low: %d remaining", resp.Rate.Remaining)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("decode content: %w", err)
	}

	// Cache result
	c.cache.Set(cacheKey, content)

	return content, nil
}

// GetRepositoryInfo fetches repository metadata.
func (c *GitHubClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepoInfo, error) {
	cacheKey := fmt.Sprintf("repo:%s/%s", owner, repo)

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		return cached.(*RepoInfo), nil
	}

	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	c.mu.Lock()
	c.requestCount++
	c.mu.Unlock()

	repository, _, err := c.rest.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	license := ""
	if repository.License != nil {
		license = repository.License.GetName()
	}

	// Get latest commit SHA on default branch
	commitSHA := ""
	if defaultBranch := repository.GetDefaultBranch(); defaultBranch != "" {
		// Wait for rate limiter for the commit fetch
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}
		c.mu.Lock()
		c.requestCount++
		c.mu.Unlock()

		branch, _, err := c.rest.Repositories.GetBranch(ctx, owner, repo, defaultBranch, 0)
		if err == nil && branch != nil && branch.Commit != nil {
			commitSHA = branch.Commit.GetSHA()
		}
	}

	info := &RepoInfo{
		FullName:      repository.GetFullName(),
		Description:   repository.GetDescription(),
		Stars:         repository.GetStargazersCount(),
		Forks:         repository.GetForksCount(),
		Watchers:      repository.GetWatchersCount(),
		DefaultBranch: repository.GetDefaultBranch(),
		CommitSHA:     commitSHA,
		UpdatedAt:     repository.GetUpdatedAt().Time,
		License:       license,
		CloneURL:      repository.GetCloneURL(),
	}

	// Cache result
	c.cache.Set(cacheKey, info)

	return info, nil
}

// ListSkillFiles lists all skill files in a repository.
func (c *GitHubClient) ListSkillFiles(ctx context.Context, owner, repo, path string) ([]*SkillFile, error) {
	cacheKey := fmt.Sprintf("tree:%s/%s:%s", owner, repo, path)

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		return cached.([]*SkillFile), nil
	}

	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	c.mu.Lock()
	c.requestCount++
	c.mu.Unlock()

	// Get the repository to find default branch
	repository, _, err := c.rest.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	branch := repository.GetDefaultBranch()
	if branch == "" {
		branch = "main"
	}

	// Get the tree recursively
	tree, _, err := c.rest.Git.GetTree(ctx, owner, repo, branch, true)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}

	repoName := fmt.Sprintf("%s/%s", owner, repo)
	var skillFiles []*SkillFile

	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}

		entryPath := entry.GetPath()

		// Check if path matches skill file patterns
		// If a specific path is provided, only look in that path
		if path != "" && !strings.HasPrefix(entryPath, path) {
			continue
		}

		// Check if the filename matches skill patterns
		parts := strings.Split(entryPath, "/")
		filename := parts[len(parts)-1]

		if IsSkillFile(filename) {
			sf := &SkillFile{
				ID:       generateSkillID(repoName, entryPath),
				Path:     entryPath,
				RepoName: repoName,
				Owner:    owner,
				Repo:     repo,
				URL:      fmt.Sprintf("https://github.com/%s/blob/%s/%s", repoName, branch, entryPath),
				SHA:      entry.GetSHA(),
			}
			skillFiles = append(skillFiles, sf)
		}
	}

	// Cache result
	c.cache.Set(cacheKey, skillFiles)

	return skillFiles, nil
}

// Stats returns client statistics.
func (c *GitHubClient) Stats() (requests, cacheHits, cacheMisses int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.requestCount, c.cacheHits, c.cacheMisses
}

// ResetStats resets the client statistics.
func (c *GitHubClient) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestCount = 0
	c.cacheHits = 0
	c.cacheMisses = 0
}

// ClearCache clears the response cache.
func (c *GitHubClient) ClearCache() {
	c.cache.Clear()
}

// GetLicenseFile fetches the LICENSE file from a repository and returns its content and metadata.
// Tries multiple license file names in order: LICENSE, LICENSE.md, LICENSE.txt, COPYING, COPYING.md, COPYING.txt, LICENSE.rst
// Returns (content, fileName, rawURL, error). If no license file is found, returns ("", "", "", nil).
func (c *GitHubClient) GetLicenseFile(ctx context.Context, owner, repo, ref string) (string, string, string, error) {
	cacheKey := fmt.Sprintf("license:%s/%s@%s", owner, repo, ref)

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		if info, ok := cached.(*LicenseInfo); ok {
			return info.FileName, info.URL, info.RawURL, nil
		}
	}

	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()

	// Try each license file name
	for _, fileName := range LicenseFileNames() {
		// Wait for rate limiter
		if err := c.limiter.Wait(ctx); err != nil {
			return "", "", "", fmt.Errorf("rate limit wait: %w", err)
		}

		c.mu.Lock()
		c.requestCount++
		c.mu.Unlock()

		opts := &github.RepositoryContentGetOptions{}
		if ref != "" {
			opts.Ref = ref
		}

		// Use GetContents to check if file exists and get its content
		fileContent, directoryContent, resp, err := c.rest.Repositories.GetContents(ctx, owner, repo, fileName, opts)
		if err != nil {
			// File not found - continue to next license file name
			if resp != nil && resp.StatusCode == 404 {
				continue
			}
			// Other errors - log and continue
			continue
		}

		// Skip if this is a directory (shouldn't happen for license files)
		if directoryContent != nil {
			continue
		}

		// Successfully found a license file
		_, err = fileContent.GetContent()
		if err != nil {
			// Failed to decode content - continue to next
			continue
		}

		// Build URLs
		branch := ref
		if branch == "" {
			branch = "main"
		}
		licenseURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, branch, fileName)
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, fileName)

		// Cache the result
		licenseInfo := &LicenseInfo{
			FileName: fileName,
			URL:      licenseURL,
			RawURL:   rawURL,
		}
		c.cache.Set(cacheKey, licenseInfo)

		return fileName, licenseURL, rawURL, nil
	}

	// No license file found - cache empty result
	c.cache.Set(cacheKey, &LicenseInfo{})
	return "", "", "", nil
}
