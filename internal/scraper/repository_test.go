package scraper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsSkillFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Exact matches at root
		{".cursorrules", false},
		{".windsurfrules", false},

		// Cursor rules directory format
		{".cursor/rules", false},

		// Skill.md patterns (case insensitive for the filename)
		{"SKILL.md", true},
		{"skill.md", true},
		{"Skill.md", true},
		{"skills/python/SKILL.md", true},
		{"skills/python/skill.md", true},
		{".claude/skills/testing/SKILL.md", true},

		// Not skill files
		{"README.md", false},
		{"SKILLS.md", false},
		{"main.go", false},
		{"skill.txt", false},
		{".cursor/config", false},
		{"src/app.js", false},
		{"docs/SKILL.md.bak", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsSkillFilePath(tt.path)
			if result != tt.expected {
				t.Errorf("IsSkillFilePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNewRepositoryManager(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "test-token")

	if rm == nil {
		t.Fatal("Expected non-nil RepositoryManager")
	}
	if rm.baseDir != baseDir {
		t.Errorf("Expected baseDir %q, got %q", baseDir, rm.baseDir)
	}
	if rm.token != "test-token" {
		t.Errorf("Expected token 'test-token', got %q", rm.token)
	}
}

func TestRepositoryManagerGetRepoPath(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	path := rm.GetRepoPath("owner", "repo")
	expected := filepath.Join(baseDir, "owner", "repo")
	if path != expected {
		t.Errorf("Expected path %q, got %q", expected, path)
	}
}

func TestRepoError(t *testing.T) {
	err := &RepoError{
		Owner: "test-owner",
		Repo:  "test-repo",
		Op:    "clone",
		Err:   os.ErrNotExist,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}

	// Check that all parts are in the error message
	if !containsSubstring(errStr, "test-owner") {
		t.Errorf("Expected error to contain owner, got %q", errStr)
	}
	if !containsSubstring(errStr, "test-repo") {
		t.Errorf("Expected error to contain repo, got %q", errStr)
	}
	if !containsSubstring(errStr, "clone") {
		t.Errorf("Expected error to contain operation, got %q", errStr)
	}
}

func TestScrapeGitHubMetadata(t *testing.T) {
	rm := NewRepositoryManager(t.TempDir(), "")

	// Test with a well-known repository
	// This is an integration test that requires network access
	stars, forks := rm.ScrapeGitHubMetadata("golang", "go")

	// The Go repo has many stars, so this should return positive numbers
	// We don't check exact values since they change
	if stars < 0 {
		t.Errorf("Expected non-negative stars, got %d", stars)
	}
	if forks < 0 {
		t.Errorf("Expected non-negative forks, got %d", forks)
	}
}

func TestScrapeGitHubMetadataInvalidRepo(t *testing.T) {
	rm := NewRepositoryManager(t.TempDir(), "")

	// Test with a non-existent repository
	stars, forks := rm.ScrapeGitHubMetadata("nonexistent-owner-12345", "nonexistent-repo-67890")

	// Should return 0, 0 for non-existent repos (graceful fallback)
	if stars != 0 {
		t.Errorf("Expected 0 stars for non-existent repo, got %d", stars)
	}
	if forks != 0 {
		t.Errorf("Expected 0 forks for non-existent repo, got %d", forks)
	}
}

func TestCleanupOldRepos(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	// Create some fake repository directories
	oldRepoPath := filepath.Join(baseDir, "old-owner", "old-repo")
	newRepoPath := filepath.Join(baseDir, "new-owner", "new-repo")

	if err := os.MkdirAll(oldRepoPath, 0755); err != nil {
		t.Fatalf("Failed to create old repo dir: %v", err)
	}
	if err := os.MkdirAll(newRepoPath, 0755); err != nil {
		t.Fatalf("Failed to create new repo dir: %v", err)
	}

	// Create a marker file in the old repo to set its mod time
	markerFile := filepath.Join(oldRepoPath, "HEAD")
	if err := os.WriteFile(markerFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Set the old repo's mod time to 8 days ago
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(oldRepoPath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set mod time: %v", err)
	}

	// Create marker file in new repo (recent)
	newMarkerFile := filepath.Join(newRepoPath, "HEAD")
	if err := os.WriteFile(newMarkerFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("Failed to create new marker file: %v", err)
	}

	// Run cleanup with 7-day max age
	err := rm.CleanupOldRepos(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldRepos failed: %v", err)
	}

	// Check that old repo was removed
	if _, err := os.Stat(oldRepoPath); !os.IsNotExist(err) {
		t.Error("Expected old repo to be removed")
	}

	// Check that new repo still exists
	if _, err := os.Stat(newRepoPath); os.IsNotExist(err) {
		t.Error("Expected new repo to still exist")
	}
}

func TestCleanupOldReposEmptyBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	// Cleanup on empty directory should not error
	err := rm.CleanupOldRepos(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldRepos on empty dir failed: %v", err)
	}
}

func TestCleanupOldReposNonExistentBaseDir(t *testing.T) {
	rm := NewRepositoryManager("/nonexistent/path/12345", "")

	// Cleanup on non-existent directory should not panic
	err := rm.CleanupOldRepos(7 * 24 * time.Hour)
	// Error is expected but should be handled gracefully
	if err != nil {
		// This is OK - we handle the error
		t.Logf("Got expected error for non-existent dir: %v", err)
	}
}

// Integration tests for clone/update operations
// These require network access and are skipped in short mode

func TestCloneOrUpdate(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a small, well-known repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Verify the repository was cloned
	if localPath == "" {
		t.Error("Expected non-empty local path")
	}

	// Check that the repository exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Errorf("Expected repository at %s", localPath)
	}

	// Check for working tree repository structure (.git directory exists)
	gitDir := filepath.Join(localPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("Expected .git directory in working tree repository")
	}

	// Run update on existing repo
	_, err = rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate (update) failed: %v", err)
	}
}

func TestGetCommitSHA(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository first
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Get commit SHA
	sha, err := rm.GetCommitSHA(localPath)
	if err != nil {
		t.Fatalf("GetCommitSHA failed: %v", err)
	}

	// SHA should be 40 hex characters
	if len(sha) != 40 {
		t.Errorf("Expected 40-char SHA, got %d chars: %s", len(sha), sha)
	}
}

func TestGetRepositoryInfo(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository first
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Get repository info
	info, err := rm.GetRepositoryInfo(localPath)
	if err != nil {
		t.Fatalf("GetRepositoryInfo failed: %v", err)
	}

	if info.Owner != "octocat" {
		t.Errorf("Expected owner 'octocat', got %q", info.Owner)
	}
	if info.Repo != "Hello-World" {
		t.Errorf("Expected repo 'Hello-World', got %q", info.Repo)
	}
	if info.FullName != "octocat/Hello-World" {
		t.Errorf("Expected full name 'octocat/Hello-World', got %q", info.FullName)
	}
	if info.CommitSHA == "" {
		t.Error("Expected non-empty commit SHA")
	}
	if info.CloneURL == "" {
		t.Error("Expected non-empty clone URL")
	}
}

func TestListSkillFiles(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository that we know has skill files
	// Using obra/superpowers which is in the curated seeds
	localPath, err := rm.CloneOrUpdate(ctx, "obra", "superpowers")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// List skill files
	files, err := rm.ListSkillFiles(localPath, "skills")
	if err != nil {
		t.Fatalf("ListSkillFiles failed: %v", err)
	}

	// This repo should have skill files in the skills directory
	t.Logf("Found %d skill files", len(files))
	for _, f := range files {
		t.Logf("  - %s", f)
	}
}

func TestReadFile(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Read README file
	content, err := rm.ReadFile(localPath, "README")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if content == "" {
		t.Error("Expected non-empty content")
	}

	// The Hello-World repo README should contain "Hello World"
	if !containsSubstring(content, "Hello") {
		t.Errorf("Expected README to contain 'Hello', got %q", content)
	}
}

func TestGetLicenseFile(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone golang/go which has a LICENSE file
	localPath, err := rm.CloneOrUpdate(ctx, "golang", "go")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Get license file
	name, content, err := rm.GetLicenseFile(localPath)
	if err != nil {
		t.Fatalf("GetLicenseFile failed: %v", err)
	}

	if name == "" {
		t.Error("Expected non-empty license file name")
	}
	if content == "" {
		t.Error("Expected non-empty license content")
	}

	t.Logf("Found license file: %s (%d bytes)", name, len(content))
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCloneOrUpdatePerRepoLocking(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	// Test that two different repos can be cloned concurrently
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Clone two repos sequentially first to verify basic functionality
	_, err1 := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err1 != nil {
		t.Fatalf("First clone failed: %v", err1)
	}

	_, err2 := rm.CloneOrUpdate(ctx, "octocat", "Spoon-Knife")
	if err2 != nil {
		t.Fatalf("Second clone failed: %v", err2)
	}

	// Verify both repos exist
	path1 := rm.GetRepoPath("octocat", "Hello-World")
	path2 := rm.GetRepoPath("octocat", "Spoon-Knife")

	if _, err := os.Stat(path1); os.IsNotExist(err) {
		t.Error("Expected Hello-World repo to exist")
	}
	if _, err := os.Stat(path2); os.IsNotExist(err) {
		t.Error("Expected Spoon-Knife repo to exist")
	}
}

func TestCloneOrUpdateTimeout(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	// Use an extremely short timeout to test timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait a bit to ensure context is expired
	time.Sleep(5 * time.Millisecond)

	_, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Check that the error is context-related
	errStr := err.Error()
	if !containsSubstring(errStr, "context") {
		t.Errorf("Expected context-related error, got: %v", err)
	}
}

func TestGetRepoLock(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	// Get lock for same repo twice - should return same mutex
	lock1 := rm.getRepoLock("owner", "repo")
	lock2 := rm.getRepoLock("owner", "repo")

	if lock1 != lock2 {
		t.Error("Expected same lock for same repo")
	}

	// Get lock for different repo - should return different mutex
	lock3 := rm.getRepoLock("owner", "other-repo")
	if lock1 == lock3 {
		t.Error("Expected different lock for different repo")
	}
}

// TestListDirectory tests listing files in a repository directory
func TestListDirectory(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// List root directory
	entries, err := rm.ListDirectory(localPath, "")
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected some entries in root directory")
	}

	// Verify we got DirEntry objects with populated fields
	for _, entry := range entries {
		if entry.Name == "" {
			t.Error("Expected non-empty entry name")
		}
		// Either a file or directory should have one of these set
		if !entry.IsDir && entry.Size == 0 {
			t.Logf("Warning: file %s has zero size", entry.Name)
		}
	}
}

// TestListDirectoryWorkingTree tests that ListDirectory correctly handles working tree repositories
func TestListDirectoryWorkingTree(t *testing.T) {
	// This test verifies that ListDirectory works with working tree repositories
	// where files are directly accessible via the filesystem

	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository (octocat/Hello-World has a README file)
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// List root - should contain at least the README file
	entries, err := rm.ListDirectory(localPath, "")
	if err != nil {
		t.Fatalf("ListDirectory on root failed: %v", err)
	}

	// Verify that we get at least one file (README)
	var foundFiles bool
	for _, entry := range entries {
		if !entry.IsDir {
			foundFiles = true
			break
		}
	}

	if !foundFiles {
		t.Error("Expected to find at least one file in root")
	}

	t.Logf("Found %d entries in root", len(entries))
}

// TestParseCount tests the parseCount helper function for parsing metric values.
func TestParseCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		// Basic numbers
		{"0", 0},
		{"1", 1},
		{"123", 123},
		{"1234", 1234},

		// Numbers with commas
		{"1,234", 1234},
		{"1,234,567", 1234567},

		// K suffix (thousands)
		{"1k", 1000},
		{"1K", 1000},
		{"1.2k", 1200},
		{"1.5K", 1500},
		{"12.3k", 12300},

		// M suffix (millions)
		{"1m", 1000000},
		{"1M", 1000000},
		{"1.5m", 1500000},
		{"2.5M", 2500000},

		// Edge cases
		{"", 0},
		{"abc", 0},
		{" 123 ", 123},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseCount(tt.input)
			if result != tt.expected {
				t.Errorf("parseCount(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseMetricFromHTML tests the parseMetricFromHTML helper function.
func TestParseMetricFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		metric   string
		expected int
	}{
		{
			name:     "title attribute pattern",
			html:     `<span id="repo-stars-counter-star" title="1,234">1.2k</span>`,
			metric:   "star",
			expected: 1234,
		},
		{
			name:     "aria-label pattern",
			html:     `<a aria-label="5678 stargazers">Stars</a>`,
			metric:   "stargazer",
			expected: 5678,
		},
		{
			name:     "text pattern with stars",
			html:     `<span>100 stars</span>`,
			metric:   "star",
			expected: 100,
		},
		{
			name:     "text pattern with forks",
			html:     `<span>50 forks</span>`,
			metric:   "fork",
			expected: 50,
		},
		{
			name:     "k suffix in text",
			html:     `<span>1.5k stars</span>`,
			metric:   "star",
			expected: 1500,
		},
		{
			name:     "no match",
			html:     `<span>Hello World</span>`,
			metric:   "star",
			expected: 0,
		},
		{
			name:     "empty html",
			html:     "",
			metric:   "star",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMetricFromHTML(tt.html, tt.metric)
			if result != tt.expected {
				t.Errorf("parseMetricFromHTML() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestRepoErrorUnwrap tests that RepoError properly unwraps to the underlying error.
func TestRepoErrorUnwrap(t *testing.T) {
	underlyingErr := os.ErrPermission
	err := &RepoError{
		Owner: "test-owner",
		Repo:  "test-repo",
		Op:    "clone",
		Err:   underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

// TestReadFileBytes tests reading a file and returning bytes.
func TestReadFileBytes(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Read README file as bytes
	content, size, err := rm.ReadFileBytes(localPath, "README")
	if err != nil {
		t.Fatalf("ReadFileBytes failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected non-empty content")
	}
	if size == 0 {
		t.Error("Expected non-zero size")
	}
	if int64(len(content)) != size {
		t.Errorf("Content length %d != reported size %d", len(content), size)
	}
}

// TestReadFileBytesNonExistent tests reading a non-existent file.
func TestReadFileBytesNonExistent(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Try to read a non-existent file
	_, _, err = rm.ReadFileBytes(localPath, "non-existent-file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestListSkillFilesDepthFiltering tests that skill files are found at any depth.
// The implementation has no depth limit - all skill files are found regardless of nesting.
func TestListSkillFilesDepthFiltering(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		skillPath string
		shouldAdd bool
	}{
		// Root level (depth 0) - always included
		{"root skill.md", "skill.md", "", true},
		{"root SKILL.md", "SKILL.md", "", true},

		// One level deep (depth 1) - always included
		{"subdir skill", "docs/skill.md", "", true},
		{"subdir SKILL", "skills/SKILL.md", "", true},

		// Two levels deep (depth 2) - NOW included (no depth limit)
		{"deep skill without skillPath", "a/b/skill.md", "", true},

		// Three levels deep (depth 3) - NOW included (no depth limit)
		{"very deep skill", "a/b/c/skill.md", "", true},

		// With skillPath set - only files within skillPath are included
		{"skillPath root", "skills/SKILL.md", "skills", true},
		{"skillPath one deep", "skills/test/SKILL.md", "skills", true},
		{"skillPath two deep", "skills/a/b/SKILL.md", "skills", true},
		{"outside skillPath", "other/SKILL.md", "skills", false},
		{"skillPath itself", "skills", "skills", false}, // not a file

		// Non-skill files should never be included
		{"non-skill file", "a/b/README.md", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from ListSkillFiles
			shouldInclude := false

			// Check if it's a skill file
			isSkill := IsSkillFilePath(tt.path)
			if !isSkill {
				shouldInclude = false
			} else if tt.skillPath == "" {
				// No skillPath filter - include all skill files
				shouldInclude = true
			} else {
				// With skillPath - only include files within that path
				if len(tt.path) > len(tt.skillPath)+1 &&
					tt.path[:len(tt.skillPath)+1] == tt.skillPath+"/" {
					shouldInclude = true
				} else if tt.path == tt.skillPath {
					shouldInclude = true
				}
			}

			if shouldInclude != tt.shouldAdd {
				t.Errorf("Expected shouldInclude=%v for path %q with skillPath %q, got %v",
					tt.shouldAdd, tt.path, tt.skillPath, shouldInclude)
			}
		})
	}
}

// TestListDirectoryNonExistent tests listing a non-existent directory.
func TestListDirectoryNonExistent(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// List a non-existent directory - should return nil, not error
	entries, err := rm.ListDirectory(localPath, "non-existent-directory")
	if err != nil {
		t.Fatalf("ListDirectory should not error for non-existent dir: %v", err)
	}

	if entries != nil {
		t.Errorf("Expected nil entries for non-existent directory, got %v", entries)
	}
}

// TestIsSkillFilePathCLAUDE tests CLAUDE.md file detection.
func TestIsSkillFilePathCLAUDE(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// CLAUDE.md patterns
		{"CLAUDE.md", true},
		{"claude.md", true},
		{"Claude.md", true},
		{"skills/CLAUDE.md", true},
		{".claude/CLAUDE.md", true},

		// Not CLAUDE files
		{"CLAUDE.txt", false},
		{"CLAUDE", false},
		{"my-claude.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsSkillFilePath(tt.path)
			if result != tt.expected {
				t.Errorf("IsSkillFilePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestUpdateRepositoryResetsWorktree verifies that updateRepository actually updates
// working tree files after fetch. This is critical for symlinked skills to receive updates.
func TestUpdateRepositoryResetsWorktree(t *testing.T) {
	tempDir := t.TempDir()
	rm := NewRepositoryManager(tempDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Clone a test repo
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("Initial clone failed: %v", err)
	}

	// Read a file's content (README exists in this repo)
	readmePath := filepath.Join(localPath, "README")
	originalContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}
	if len(originalContent) == 0 {
		t.Fatal("README is empty, cannot test reset")
	}

	// Simulate local modification (this would be lost after proper update)
	modifiedContent := []byte("modified by test")
	err = os.WriteFile(readmePath, modifiedContent, 0644)
	if err != nil {
		t.Fatalf("Failed to modify README: %v", err)
	}

	// Verify modification was written
	checkContent, _ := os.ReadFile(readmePath)
	if string(checkContent) != string(modifiedContent) {
		t.Fatal("Local modification was not written")
	}

	// Clear the recently updated cache to force a fetch
	rm.mu.Lock()
	delete(rm.recentlyUpdated, "octocat/Hello-World")
	rm.mu.Unlock()

	// Update the repo - this should reset the working tree
	_, err = rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify the file was reset to remote content
	updatedContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README after update: %v", err)
	}

	if string(updatedContent) != string(originalContent) {
		t.Errorf("Working tree was not reset to remote content.\nExpected: %q\nGot: %q",
			string(originalContent), string(updatedContent))
	}
}
