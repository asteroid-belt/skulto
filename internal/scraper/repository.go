package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// DefaultRepoTimeout is the per-repository timeout for clone/fetch operations.
const DefaultRepoTimeout = 2 * time.Minute

// RecentUpdateTTL is how long to skip fetches after a successful update.
// This prevents redundant network calls when reading multiple files from the same repo.
const RecentUpdateTTL = 60 * time.Second

// RepositoryManager handles git repository operations using working tree repositories.
type RepositoryManager struct {
	baseDir         string
	token           string
	mu              sync.RWMutex           // Protects repoLocks and recentlyUpdated maps
	repoLocks       map[string]*sync.Mutex // Per-repository locks
	recentlyUpdated map[string]time.Time   // Tracks when repos were last updated
}

// RepositoryInfo contains metadata extracted from a local repository.
type RepositoryInfo struct {
	Owner         string
	Repo          string
	FullName      string
	Description   string
	Stars         int
	Forks         int
	DefaultBranch string
	CommitSHA     string
	CloneURL      string
	LocalPath     string
	UpdatedAt     time.Time
}

// RepoError represents repository operation errors.
type RepoError struct {
	Owner string
	Repo  string
	Op    string // "clone", "fetch", "read", "open"
	Err   error
}

func (e *RepoError) Error() string {
	return fmt.Sprintf("%s/%s: %s failed: %v", e.Owner, e.Repo, e.Op, e.Err)
}

func (e *RepoError) Unwrap() error {
	return e.Err
}

// NewRepositoryManager creates a new repository manager.
func NewRepositoryManager(baseDir string, token string) *RepositoryManager {
	return &RepositoryManager{
		baseDir:         baseDir,
		token:           token,
		repoLocks:       make(map[string]*sync.Mutex),
		recentlyUpdated: make(map[string]time.Time),
	}
}

// GetRepoPath returns the local path for a repository.
func (rm *RepositoryManager) GetRepoPath(owner, repo string) string {
	return filepath.Join(rm.baseDir, owner, repo)
}

// getRepoLock returns a mutex for a specific repository (owner/repo).
// Creates a new mutex if one doesn't exist for this repo.
func (rm *RepositoryManager) getRepoLock(owner, repo string) *sync.Mutex {
	key := fmt.Sprintf("%s/%s", owner, repo)

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.repoLocks[key] == nil {
		rm.repoLocks[key] = &sync.Mutex{}
	}
	return rm.repoLocks[key]
}

// wasRecentlyUpdated checks if a repository was updated within RecentUpdateTTL.
func (rm *RepositoryManager) wasRecentlyUpdated(owner, repo string) bool {
	key := fmt.Sprintf("%s/%s", owner, repo)

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if t, ok := rm.recentlyUpdated[key]; ok {
		return time.Since(t) < RecentUpdateTTL
	}
	return false
}

// markRecentlyUpdated marks a repository as recently updated.
func (rm *RepositoryManager) markRecentlyUpdated(owner, repo string) {
	key := fmt.Sprintf("%s/%s", owner, repo)

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.recentlyUpdated[key] = time.Now()
}

// CloneOrUpdate clones a new repository or updates an existing one.
// Uses working tree repositories so files can be directly accessed via filesystem paths.
// Uses per-repository locking and timeout for better concurrency.
// Skips fetch if the repo was updated within RecentUpdateTTL to avoid redundant network calls.
func (rm *RepositoryManager) CloneOrUpdate(ctx context.Context, owner, repo string) (string, error) {
	localPath := rm.GetRepoPath(owner, repo)

	// Fast path: if repo exists and was recently updated, skip the fetch entirely
	if rm.wasRecentlyUpdated(owner, repo) {
		if _, err := os.Stat(filepath.Join(localPath, ".git")); err == nil {
			return localPath, nil
		}
	}

	// Per-repo lock (doesn't block other repos)
	repoLock := rm.getRepoLock(owner, repo)
	repoLock.Lock()
	defer repoLock.Unlock()

	// Double-check after acquiring lock (another goroutine may have just updated)
	if rm.wasRecentlyUpdated(owner, repo) {
		if _, err := os.Stat(filepath.Join(localPath, ".git")); err == nil {
			return localPath, nil
		}
	}

	// Per-repo timeout if parent doesn't have shorter deadline
	var cancel context.CancelFunc
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > DefaultRepoTimeout {
		ctx, cancel = context.WithTimeout(ctx, DefaultRepoTimeout)
		defer cancel()
	}

	// Check if repo exists (working tree repos have .git directory)
	if _, err := os.Stat(filepath.Join(localPath, ".git")); err == nil {
		// Repository exists, update it
		if err := rm.updateRepository(ctx, localPath, owner, repo); err != nil {
			return localPath, err
		}
		rm.markRecentlyUpdated(owner, repo)
		return localPath, nil
	}

	// Repository doesn't exist, clone it
	if err := rm.cloneRepository(ctx, localPath, owner, repo); err != nil {
		return localPath, err
	}
	rm.markRecentlyUpdated(owner, repo)
	return localPath, nil
}

// cloneRepository clones a new repository as a working tree (shallow clone).
func (rm *RepositoryManager) cloneRepository(ctx context.Context, localPath, owner, repo string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return &RepoError{Owner: owner, Repo: repo, Op: "clone", Err: err}
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	cloneOpts := &git.CloneOptions{
		URL:          cloneURL,
		SingleBranch: true, // Only fetch default branch
		Tags:         git.NoTags,
		Depth:        1, // Shallow clone for efficiency
	}

	// Add authentication if token provided
	if rm.token != "" {
		cloneOpts.Auth = &gitHttp.BasicAuth{
			Username: "oauth2",
			Password: rm.token,
		}
	}

	// Clone as working tree repository
	_, err := git.PlainCloneContext(ctx, localPath, false, cloneOpts)
	if err != nil {
		// Clean up partial clone on failure (best-effort)
		_ = os.RemoveAll(localPath)
		return &RepoError{Owner: owner, Repo: repo, Op: "clone", Err: err}
	}

	return nil
}

// updateRepository updates an existing repository.
func (rm *RepositoryManager) updateRepository(ctx context.Context, localPath, owner, repo string) error {
	r, err := git.PlainOpen(localPath)
	if err != nil {
		return &RepoError{Owner: owner, Repo: repo, Op: "open", Err: err}
	}

	fetchOpts := &git.FetchOptions{
		Force: true,
		Tags:  git.NoTags,
	}

	// Add authentication if token provided
	if rm.token != "" {
		fetchOpts.Auth = &gitHttp.BasicAuth{
			Username: "oauth2",
			Password: rm.token,
		}
	}

	// Fetch latest changes
	err = r.FetchContext(ctx, fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return &RepoError{Owner: owner, Repo: repo, Op: "fetch", Err: err}
	}

	// Update working tree to match remote HEAD
	// This is critical for symlinks to reflect the latest content
	if err := rm.resetToRemoteHead(r, owner, repo); err != nil {
		return err
	}

	// Update directory mod time to track access (best-effort)
	now := time.Now()
	_ = os.Chtimes(localPath, now, now)

	return nil
}

// resetToRemoteHead resets the working tree to match origin/HEAD.
// This ensures symlinked skill files are updated after fetch.
func (rm *RepositoryManager) resetToRemoteHead(r *git.Repository, owner, repo string) error {
	// Get the remote HEAD reference
	remoteRef, err := r.Reference(plumbing.NewRemoteReferenceName("origin", "HEAD"), true)
	if err != nil {
		// Try origin/main as fallback
		remoteRef, err = r.Reference(plumbing.NewRemoteReferenceName("origin", "main"), true)
		if err != nil {
			// Try origin/master as final fallback
			remoteRef, err = r.Reference(plumbing.NewRemoteReferenceName("origin", "master"), true)
			if err != nil {
				return &RepoError{Owner: owner, Repo: repo, Op: "resolve-ref", Err: err}
			}
		}
	}

	// Get worktree
	w, err := r.Worktree()
	if err != nil {
		return &RepoError{Owner: owner, Repo: repo, Op: "worktree", Err: err}
	}

	// Hard reset to the remote commit
	err = w.Reset(&git.ResetOptions{
		Commit: remoteRef.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return &RepoError{Owner: owner, Repo: repo, Op: "reset", Err: err}
	}

	return nil
}

// GetCommitSHA returns the current HEAD commit SHA.
func (rm *RepositoryManager) GetCommitSHA(localPath string) (string, error) {
	r, err := git.PlainOpen(localPath)
	if err != nil {
		return "", err
	}

	ref, err := r.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String(), nil
}

// GetRepositoryInfo extracts metadata from local repository and scrapes GitHub for stars/forks.
func (rm *RepositoryManager) GetRepositoryInfo(localPath string) (*RepositoryInfo, error) {
	// Extract owner/repo from path
	parts := strings.Split(filepath.Clean(localPath), string(filepath.Separator))
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid repository path: %s", localPath)
	}
	repo := parts[len(parts)-1]
	owner := parts[len(parts)-2]

	r, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, err
	}

	ref, err := r.Head()
	if err != nil {
		return nil, err
	}

	// Get HEAD commit object
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	// Parse default branch from ref name
	defaultBranch := strings.TrimPrefix(ref.Name().Short(), "refs/heads/")

	// Read description from README.md
	description := rm.extractDescription(commit)

	// Scrape stars/forks from GitHub HTML
	stars, forks := rm.ScrapeGitHubMetadata(owner, repo)

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	return &RepositoryInfo{
		Owner:         owner,
		Repo:          repo,
		FullName:      fmt.Sprintf("%s/%s", owner, repo),
		Description:   description,
		Stars:         stars,
		Forks:         forks,
		DefaultBranch: defaultBranch,
		CommitSHA:     ref.Hash().String(),
		CloneURL:      cloneURL,
		LocalPath:     localPath,
		UpdatedAt:     commit.Committer.When,
	}, nil
}

// extractDescription reads the first paragraph from README.md.
func (rm *RepositoryManager) extractDescription(commit *object.Commit) string {
	tree, err := commit.Tree()
	if err != nil {
		return ""
	}

	// Try README.md first, then README
	readmeNames := []string{"README.md", "README", "readme.md", "Readme.md"}
	for _, name := range readmeNames {
		file, err := tree.File(name)
		if err != nil {
			continue
		}

		reader, err := file.Reader()
		if err != nil {
			continue
		}
		defer func() { _ = reader.Close() }()

		// Read first 2KB to find description
		buf := make([]byte, 2048)
		n, _ := reader.Read(buf)
		content := string(buf[:n])

		// Find first non-heading, non-empty line
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
				continue
			}
			// Skip badges and links that start with [
			if strings.HasPrefix(line, "[") && strings.Contains(line, "](") {
				continue
			}
			// Found a description line
			if len(line) > 200 {
				line = line[:200] + "..."
			}
			return line
		}
	}

	return ""
}

// ScrapeGitHubMetadata fetches stars and forks count from GitHub HTML page.
// Returns (0, 0) on any error (graceful fallback).
func (rm *RepositoryManager) ScrapeGitHubMetadata(owner, repo string) (int, int) {
	url := fmt.Sprintf("https://github.com/%s/%s", owner, repo)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0
	}

	// Set a user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Skulto/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, 0
	}

	// Read the response body (limit to 500KB to avoid memory issues)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024))
	if err != nil {
		return 0, 0
	}

	html := string(body)

	stars := parseMetricFromHTML(html, "stargazer")
	forks := parseMetricFromHTML(html, "fork")

	return stars, forks
}

// parseMetricFromHTML extracts a metric count from GitHub HTML.
// Looks for patterns like "1.2k" or "123" near the metric keyword.
func parseMetricFromHTML(html, metric string) int {
	// Pattern 1: Look for id="repo-stars-counter-star" or similar with a title attribute
	// Example: <span id="repo-stars-counter-star" ... title="1,234" ...>1.2k</span>
	titlePattern := regexp.MustCompile(fmt.Sprintf(`(?i)%s[^>]*title="([0-9,]+)"`, metric))
	if matches := titlePattern.FindStringSubmatch(html); len(matches) > 1 {
		return parseCount(matches[1])
	}

	// Pattern 2: Look for data attributes
	// Example: <span ... data-view-component="true">1.2k</span> near "Star" or "Fork"
	dataPattern := regexp.MustCompile(fmt.Sprintf(`(?i)%s[^<]*</[^>]+>\s*<[^>]*>([0-9,.kKmM]+)`, metric))
	if matches := dataPattern.FindStringSubmatch(html); len(matches) > 1 {
		return parseCount(matches[1])
	}

	// Pattern 3: Look for "X stars" or "X forks" text patterns
	textPattern := regexp.MustCompile(fmt.Sprintf(`([0-9,.kKmM]+)\s*%ss?`, metric))
	if matches := textPattern.FindStringSubmatch(html); len(matches) > 1 {
		return parseCount(matches[1])
	}

	// Pattern 4: aria-label pattern (common in newer GitHub UI)
	ariaPattern := regexp.MustCompile(fmt.Sprintf(`aria-label="([0-9,]+)\s+%s`, metric))
	if matches := ariaPattern.FindStringSubmatch(html); len(matches) > 1 {
		return parseCount(matches[1])
	}

	return 0
}

// parseCount converts a count string (like "1.2k", "1,234") to an integer.
func parseCount(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")

	// Handle k/m suffixes
	multiplier := 1
	if strings.HasSuffix(strings.ToLower(s), "k") {
		multiplier = 1000
		s = s[:len(s)-1]
	} else if strings.HasSuffix(strings.ToLower(s), "m") {
		multiplier = 1000000
		s = s[:len(s)-1]
	}

	// Parse as float to handle "1.2k" style
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f * float64(multiplier))
	}

	// Try parsing as int
	if i, err := strconv.Atoi(s); err == nil {
		return i * multiplier
	}

	return 0
}

// ListSkillFiles finds all skill files in the repository using git tree traversal.
// Searches the entire repository tree for skill files (SKILL.md, skill.md, CLAUDE.md).
// If skillPath is set, only searches within that specific path.
func (rm *RepositoryManager) ListSkillFiles(localPath string, skillPath string) ([]string, error) {
	var skillFiles []string

	r, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, err
	}

	ref, err := r.Head()
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	// Traverse the tree - accept all skill files at any depth
	err = tree.Files().ForEach(func(f *object.File) error {
		if !IsSkillFilePath(f.Name) {
			return nil
		}

		// If skillPath is set, only accept files within that path
		if skillPath != "" {
			if !strings.HasPrefix(f.Name, skillPath+"/") && f.Name != skillPath {
				return nil
			}
		}

		skillFiles = append(skillFiles, f.Name)
		return nil
	})

	return skillFiles, err
}

// SkillMDPatterns are file patterns specifically for SKILL.md files.
// This excludes cursor/windsurf rules which are out of scope.
var SkillMDPatterns = []string{
	"SKILL.md",
	"skill.md",
	"Skill.md",
	"CLAUDE.md",
	"claude.md",
}

// IsSkillFilePath checks if a path matches skill file patterns.
// This focuses on SKILL.md and CLAUDE.md files only.
// Cursor/windsurf rules are explicitly excluded as they are out of scope.
func IsSkillFilePath(path string) bool {
	// Ensure clean relative path
	path = strings.TrimPrefix(path, "/")

	// Check filename patterns (case-insensitive for skill.md and claude.md)
	lowerPath := strings.ToLower(path)

	// Match skill.md or claude.md at any level
	if strings.HasSuffix(lowerPath, "/skill.md") || lowerPath == "skill.md" {
		return true
	}
	if strings.HasSuffix(lowerPath, "/claude.md") || lowerPath == "claude.md" {
		return true
	}

	return false
}

// ReadFile reads a file from the local bare repository.
func (rm *RepositoryManager) ReadFile(localPath, relativePath string) (string, error) {
	r, err := git.PlainOpen(localPath)
	if err != nil {
		return "", err
	}

	ref, err := r.Head()
	if err != nil {
		return "", err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	file, err := tree.File(relativePath)
	if err != nil {
		return "", err
	}

	content, err := file.Contents()
	if err != nil {
		return "", err
	}

	return content, nil
}

// DirEntry represents a file or directory entry.
type DirEntry struct {
	Name  string
	Size  int64
	IsDir bool
}

// ListDirectory lists files and directories in a path within the repository.
// Returns nil if the directory doesn't exist.
func (rm *RepositoryManager) ListDirectory(localPath, dirPath string) ([]DirEntry, error) {
	r, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, err
	}

	ref, err := r.Head()
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	// Navigate to the target directory
	if dirPath != "" && dirPath != "." {
		tree, err = tree.Tree(dirPath)
		if err != nil {
			// Directory doesn't exist
			return nil, nil
		}
	}

	var entries []DirEntry
	for _, entry := range tree.Entries {
		de := DirEntry{
			Name:  entry.Name,
			IsDir: !entry.Mode.IsFile(),
		}

		// Get file size if it's a file
		if !de.IsDir {
			if file, err := tree.File(entry.Name); err == nil {
				de.Size = file.Size
			}
		}

		entries = append(entries, de)
	}

	return entries, nil
}

// ReadFileBytes reads a file from the local repository and returns bytes.
func (rm *RepositoryManager) ReadFileBytes(localPath, relativePath string) ([]byte, int64, error) {
	r, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, 0, err
	}

	ref, err := r.Head()
	if err != nil {
		return nil, 0, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, 0, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, 0, err
	}

	file, err := tree.File(relativePath)
	if err != nil {
		return nil, 0, err
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, err
	}

	return content, file.Size, nil
}

// GetLicenseFile finds and reads the license file from git tree.
// Returns (fileName, content, error). If no license found, returns ("", "", nil).
func (rm *RepositoryManager) GetLicenseFile(localPath string) (string, string, error) {
	licenseNames := []string{
		"LICENSE", "LICENSE.md", "LICENSE.txt",
		"COPYING", "COPYING.md", "COPYING.txt",
		"LICENSE.rst", "LICENCE", "LICENCE.md",
	}

	r, err := git.PlainOpen(localPath)
	if err != nil {
		return "", "", err
	}

	ref, err := r.Head()
	if err != nil {
		return "", "", err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return "", "", err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", "", err
	}

	for _, name := range licenseNames {
		file, err := tree.File(name)
		if err != nil {
			continue
		}

		content, err := file.Contents()
		if err != nil {
			continue
		}

		return name, content, nil
	}

	return "", "", nil
}

// CleanupOldRepos removes repositories not accessed recently.
func (rm *RepositoryManager) CleanupOldRepos(maxAge time.Duration) error {
	// No global lock - allows concurrent clone/fetch operations
	cutoff := time.Now().Add(-maxAge)

	// Check if base directory exists
	entries, err := os.ReadDir(rm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Base dir doesn't exist, nothing to clean
		}
		return err
	}

	for _, ownerEntry := range entries {
		if !ownerEntry.IsDir() {
			continue
		}

		ownerPath := filepath.Join(rm.baseDir, ownerEntry.Name())
		repoEntries, err := os.ReadDir(ownerPath)
		if err != nil {
			continue
		}

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}

			repoPath := filepath.Join(ownerPath, repoEntry.Name())
			info, err := os.Stat(repoPath)
			if err != nil {
				continue
			}

			// Check modification time
			if info.ModTime().Before(cutoff) {
				// Acquire per-repo lock before deletion to prevent race with CloneOrUpdate
				repoLock := rm.getRepoLock(ownerEntry.Name(), repoEntry.Name())
				repoLock.Lock()
				_ = os.RemoveAll(repoPath)
				repoLock.Unlock()
			}
		}

		// Clean up empty owner directories (best-effort)
		remaining, _ := os.ReadDir(ownerPath)
		if len(remaining) == 0 {
			_ = os.Remove(ownerPath)
		}
	}

	return nil
}

// RemoveRepository deletes a specific repository from disk.
// Uses per-repository locking to prevent race conditions with CloneOrUpdate.
func (rm *RepositoryManager) RemoveRepository(owner, repo string) error {
	repoPath := rm.GetRepoPath(owner, repo)

	// Acquire per-repo lock before deletion
	repoLock := rm.getRepoLock(owner, repo)
	repoLock.Lock()
	defer repoLock.Unlock()

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil // Repository doesn't exist, nothing to do
	}

	// Remove the repository directory
	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("remove repository %s/%s: %w", owner, repo, err)
	}

	// Clean up empty owner directory (best-effort)
	ownerPath := filepath.Dir(repoPath)
	remaining, _ := os.ReadDir(ownerPath)
	if len(remaining) == 0 {
		_ = os.Remove(ownerPath)
	}

	return nil
}
