package scraper

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GitClient wraps git repository operations with caching.
// It provides the same interface as GitHubClient but uses local git clones
// instead of the GitHub API, eliminating rate limits for public repositories.
type GitClient struct {
	repoManager *RepositoryManager
	cache       *ResponseCache
	token       string
	cloneDir    string
	mu          sync.RWMutex

	// Stats tracking
	requestCount int
	cacheHits    int
	cacheMisses  int
}

// NewGitClient creates a new git-based client.
func NewGitClient(token string, cloneDir string) *GitClient {
	return &GitClient{
		repoManager: NewRepositoryManager(cloneDir, token),
		cache:       NewResponseCache(DefaultCacheTTL),
		token:       token,
		cloneDir:    cloneDir,
	}
}

// GetRepositoryInfo fetches repository metadata using git clone.
// Returns the same RepoInfo structure as GitHubClient for compatibility.
func (gc *GitClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepoInfo, error) {
	cacheKey := fmt.Sprintf("git:repo:%s/%s", owner, repo)

	// Check cache
	if cached, ok := gc.cache.Get(cacheKey); ok {
		gc.mu.Lock()
		gc.cacheHits++
		gc.mu.Unlock()
		return cached.(*RepoInfo), nil
	}

	gc.mu.Lock()
	gc.cacheMisses++
	gc.requestCount++
	gc.mu.Unlock()

	// Clone or update repository
	localPath, err := gc.repoManager.CloneOrUpdate(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("clone/update repository: %w", err)
	}

	// Get repository info from local clone
	repoInfo, err := gc.repoManager.GetRepositoryInfo(localPath)
	if err != nil {
		return nil, fmt.Errorf("get repository info: %w", err)
	}

	// Transform to RepoInfo (compatible with GitHubClient)
	result := &RepoInfo{
		FullName:      repoInfo.FullName,
		Description:   repoInfo.Description,
		Stars:         repoInfo.Stars,
		Forks:         repoInfo.Forks,
		Watchers:      0, // Not available from git clone
		DefaultBranch: repoInfo.DefaultBranch,
		CommitSHA:     repoInfo.CommitSHA,
		UpdatedAt:     repoInfo.UpdatedAt,
		License:       "", // Will be set separately via GetLicenseFile
		CloneURL:      repoInfo.CloneURL,
	}

	// Cache result
	gc.cache.Set(cacheKey, result)

	return result, nil
}

// ListSkillFiles lists all skill files in a repository.
// Returns the same []*SkillFile structure as GitHubClient for compatibility.
func (gc *GitClient) ListSkillFiles(ctx context.Context, owner, repo, path string) ([]*SkillFile, error) {
	cacheKey := fmt.Sprintf("git:tree:%s/%s:%s", owner, repo, path)

	// Check cache
	if cached, ok := gc.cache.Get(cacheKey); ok {
		gc.mu.Lock()
		gc.cacheHits++
		gc.mu.Unlock()
		return cached.([]*SkillFile), nil
	}

	gc.mu.Lock()
	gc.cacheMisses++
	gc.requestCount++
	gc.mu.Unlock()

	// Ensure repository is cloned/updated
	localPath, err := gc.repoManager.CloneOrUpdate(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("clone/update repository: %w", err)
	}

	// List skill files from local repository
	filePaths, err := gc.repoManager.ListSkillFiles(localPath, path)
	if err != nil {
		return nil, fmt.Errorf("list skill files: %w", err)
	}

	// Get commit SHA for URLs
	commitSHA, _ := gc.repoManager.GetCommitSHA(localPath)

	repoName := fmt.Sprintf("%s/%s", owner, repo)
	var skillFiles []*SkillFile

	for _, filePath := range filePaths {
		sf := &SkillFile{
			ID:       generateSkillID(repoName, filePath),
			Path:     filePath,
			RepoName: repoName,
			Owner:    owner,
			Repo:     repo,
			URL:      fmt.Sprintf("https://github.com/%s/blob/%s/%s", repoName, commitSHA, filePath),
			SHA:      commitSHA, // Use commit SHA as file SHA
		}
		skillFiles = append(skillFiles, sf)
	}

	// Cache result
	gc.cache.Set(cacheKey, skillFiles)

	return skillFiles, nil
}

// GetFileContent fetches raw file content from a repository.
func (gc *GitClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	// Use ref in cache key if provided, otherwise use "HEAD"
	refKey := ref
	if refKey == "" {
		refKey = "HEAD"
	}
	cacheKey := fmt.Sprintf("git:content:%s/%s:%s@%s", owner, repo, path, refKey)

	// Check cache
	if cached, ok := gc.cache.Get(cacheKey); ok {
		gc.mu.Lock()
		gc.cacheHits++
		gc.mu.Unlock()
		return cached.(string), nil
	}

	gc.mu.Lock()
	gc.cacheMisses++
	gc.requestCount++
	gc.mu.Unlock()

	// Ensure repository is cloned/updated
	localPath, err := gc.repoManager.CloneOrUpdate(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("clone/update repository: %w", err)
	}

	// Read file from local repository
	content, err := gc.repoManager.ReadFile(localPath, path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	// Cache result
	gc.cache.Set(cacheKey, content)

	return content, nil
}

// GetLicenseFile fetches the LICENSE file from a repository.
// Returns (fileName, licenseURL, rawURL, error) for compatibility with GitHubClient.
func (gc *GitClient) GetLicenseFile(ctx context.Context, owner, repo, ref string) (string, string, string, error) {
	refKey := ref
	if refKey == "" {
		refKey = "HEAD"
	}
	cacheKey := fmt.Sprintf("git:license:%s/%s@%s", owner, repo, refKey)

	// Check cache
	if cached, ok := gc.cache.Get(cacheKey); ok {
		gc.mu.Lock()
		gc.cacheHits++
		gc.mu.Unlock()
		if info, ok := cached.(*LicenseInfo); ok {
			return info.FileName, info.URL, info.RawURL, nil
		}
	}

	gc.mu.Lock()
	gc.cacheMisses++
	gc.requestCount++
	gc.mu.Unlock()

	// Ensure repository is cloned/updated
	localPath, err := gc.repoManager.CloneOrUpdate(ctx, owner, repo)
	if err != nil {
		return "", "", "", fmt.Errorf("clone/update repository: %w", err)
	}

	// Get license file from local repository
	fileName, content, err := gc.repoManager.GetLicenseFile(localPath)
	if err != nil {
		return "", "", "", fmt.Errorf("get license file: %w", err)
	}

	// No license file found
	if fileName == "" {
		gc.cache.Set(cacheKey, &LicenseInfo{})
		return "", "", "", nil
	}

	// Get commit SHA for URLs
	commitSHA, _ := gc.repoManager.GetCommitSHA(localPath)
	if commitSHA == "" {
		commitSHA = "main"
	}

	// Build URLs
	licenseURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, commitSHA, fileName)
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, commitSHA, fileName)

	// Detect license type from content
	licenseType := DetectLicenseType(content)

	// Cache the result
	licenseInfo := &LicenseInfo{
		Type:     licenseType,
		FileName: fileName,
		URL:      licenseURL,
		RawURL:   rawURL,
	}
	gc.cache.Set(cacheKey, licenseInfo)

	return fileName, licenseURL, rawURL, nil
}

// ListDirectoryContents lists files in a directory within a repository.
// Returns nil if the directory doesn't exist (not an error).
func (gc *GitClient) ListDirectoryContents(ctx context.Context, owner, repo, dirPath string) ([]DirEntry, error) {
	cacheKey := fmt.Sprintf("git:dir:%s/%s:%s", owner, repo, dirPath)

	// Check cache
	if cached, ok := gc.cache.Get(cacheKey); ok {
		gc.mu.Lock()
		gc.cacheHits++
		gc.mu.Unlock()
		return cached.([]DirEntry), nil
	}

	gc.mu.Lock()
	gc.cacheMisses++
	gc.requestCount++
	gc.mu.Unlock()

	// Ensure repository is cloned/updated
	localPath, err := gc.repoManager.CloneOrUpdate(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("clone/update repository: %w", err)
	}

	// List directory from local repository
	entries, err := gc.repoManager.ListDirectory(localPath, dirPath)
	if err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}

	// Cache result (even if nil)
	gc.cache.Set(cacheKey, entries)

	return entries, nil
}

// ReadFileBytes reads a file and returns its bytes and size.
func (gc *GitClient) ReadFileBytes(ctx context.Context, owner, repo, path string) ([]byte, int64, error) {
	cacheKey := fmt.Sprintf("git:bytes:%s/%s:%s", owner, repo, path)

	// Check cache
	if cached, ok := gc.cache.Get(cacheKey); ok {
		gc.mu.Lock()
		gc.cacheHits++
		gc.mu.Unlock()
		if data, ok := cached.(*fileData); ok {
			return data.Content, data.Size, nil
		}
	}

	gc.mu.Lock()
	gc.cacheMisses++
	gc.requestCount++
	gc.mu.Unlock()

	// Ensure repository is cloned/updated
	localPath, err := gc.repoManager.CloneOrUpdate(ctx, owner, repo)
	if err != nil {
		return nil, 0, fmt.Errorf("clone/update repository: %w", err)
	}

	// Read file from local repository
	content, size, err := gc.repoManager.ReadFileBytes(localPath, path)
	if err != nil {
		return nil, 0, fmt.Errorf("read file: %w", err)
	}

	// Cache result
	gc.cache.Set(cacheKey, &fileData{Content: content, Size: size})

	return content, size, nil
}

// fileData holds cached file content and size.
type fileData struct {
	Content []byte
	Size    int64
}

// Stats returns client statistics.
func (gc *GitClient) Stats() (requests, cacheHits, cacheMisses int) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.requestCount, gc.cacheHits, gc.cacheMisses
}

// ResetStats resets the client statistics.
func (gc *GitClient) ResetStats() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.requestCount = 0
	gc.cacheHits = 0
	gc.cacheMisses = 0
}

// ClearCache clears the response cache.
func (gc *GitClient) ClearCache() {
	gc.cache.Clear()
}

// Cleanup removes old repositories that haven't been accessed recently.
func (gc *GitClient) Cleanup(maxAge time.Duration) error {
	return gc.repoManager.CleanupOldRepos(maxAge)
}

// GetCloneDir returns the clone directory path.
func (gc *GitClient) GetCloneDir() string {
	return gc.cloneDir
}
