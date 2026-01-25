package scraper

import (
	"context"
	"testing"
	"time"
)

func TestNewGitClient(t *testing.T) {
	cloneDir := t.TempDir()
	client := NewGitClient("test-token", cloneDir)

	if client == nil {
		t.Fatal("Expected non-nil GitClient")
	}
	if client.cache == nil {
		t.Error("Expected non-nil cache")
	}
	if client.repoManager == nil {
		t.Error("Expected non-nil repoManager")
	}
	if client.cloneDir != cloneDir {
		t.Errorf("Expected cloneDir %q, got %q", cloneDir, client.cloneDir)
	}
	if client.token != "test-token" {
		t.Errorf("Expected token 'test-token', got %q", client.token)
	}
}

func TestGitClientStats(t *testing.T) {
	client := NewGitClient("", t.TempDir())

	requests, hits, misses := client.Stats()
	if requests != 0 || hits != 0 || misses != 0 {
		t.Errorf("Expected zero stats, got requests=%d, hits=%d, misses=%d",
			requests, hits, misses)
	}
}

func TestGitClientResetStats(t *testing.T) {
	client := NewGitClient("", t.TempDir())

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

func TestGitClientClearCache(t *testing.T) {
	client := NewGitClient("", t.TempDir())

	// Add some cache entries
	client.cache.Set("key1", "value1")
	client.cache.Set("key2", "value2")

	if client.cache.Len() != 2 {
		t.Errorf("Expected 2 cache entries, got %d", client.cache.Len())
	}

	client.ClearCache()

	if client.cache.Len() != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", client.cache.Len())
	}
}

func TestGitClientGetCloneDir(t *testing.T) {
	cloneDir := t.TempDir()
	client := NewGitClient("", cloneDir)

	if got := client.GetCloneDir(); got != cloneDir {
		t.Errorf("Expected cloneDir %q, got %q", cloneDir, got)
	}
}

func TestGitClientCleanup(t *testing.T) {
	client := NewGitClient("", t.TempDir())

	// Cleanup on empty directory should not error
	err := client.Cleanup(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
}

// Integration tests - require network access

func TestGitClientGetRepositoryInfo(t *testing.T) {
	client := NewGitClient("", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	info, err := client.GetRepositoryInfo(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("GetRepositoryInfo failed: %v", err)
	}

	if info.FullName != "octocat/Hello-World" {
		t.Errorf("Expected FullName 'octocat/Hello-World', got %q", info.FullName)
	}
	if info.CommitSHA == "" {
		t.Error("Expected non-empty CommitSHA")
	}
	if info.CloneURL == "" {
		t.Error("Expected non-empty CloneURL")
	}

	// Verify caching works
	requests1, hits1, _ := client.Stats()

	info2, err := client.GetRepositoryInfo(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("GetRepositoryInfo (cached) failed: %v", err)
	}

	if info2.FullName != info.FullName {
		t.Error("Cached result doesn't match")
	}

	requests2, hits2, _ := client.Stats()
	if requests2 != requests1 {
		t.Error("Expected no new requests for cached result")
	}
	if hits2 <= hits1 {
		t.Error("Expected cache hit for second call")
	}
}

func TestGitClientListSkillFiles(t *testing.T) {
	client := NewGitClient("", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use obra/superpowers which has skill files
	files, err := client.ListSkillFiles(ctx, "obra", "superpowers", "skills")
	if err != nil {
		t.Fatalf("ListSkillFiles failed: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected to find skill files")
	}

	// Verify file structure
	for _, sf := range files {
		if sf.ID == "" {
			t.Error("Expected non-empty ID")
		}
		if sf.Path == "" {
			t.Error("Expected non-empty Path")
		}
		if sf.Owner != "obra" {
			t.Errorf("Expected Owner 'obra', got %q", sf.Owner)
		}
		if sf.Repo != "superpowers" {
			t.Errorf("Expected Repo 'superpowers', got %q", sf.Repo)
		}
		if sf.URL == "" {
			t.Error("Expected non-empty URL")
		}
	}

	t.Logf("Found %d skill files", len(files))
}

func TestGitClientGetFileContent(t *testing.T) {
	client := NewGitClient("", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	content, err := client.GetFileContent(ctx, "octocat", "Hello-World", "README", "")
	if err != nil {
		t.Fatalf("GetFileContent failed: %v", err)
	}

	if content == "" {
		t.Error("Expected non-empty content")
	}

	// Verify caching
	_, hits1, _ := client.Stats()

	content2, err := client.GetFileContent(ctx, "octocat", "Hello-World", "README", "")
	if err != nil {
		t.Fatalf("GetFileContent (cached) failed: %v", err)
	}

	if content2 != content {
		t.Error("Cached result doesn't match")
	}

	_, hits2, _ := client.Stats()
	if hits2 <= hits1 {
		t.Error("Expected cache hit for second call")
	}
}

func TestGitClientGetLicenseFile(t *testing.T) {
	client := NewGitClient("", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use anthropics/anthropic-cookbook which has a LICENSE file and is reasonably sized
	fileName, licenseURL, rawURL, err := client.GetLicenseFile(ctx, "anthropics", "anthropic-cookbook", "")
	if err != nil {
		t.Fatalf("GetLicenseFile failed: %v", err)
	}

	if fileName == "" {
		t.Error("Expected non-empty fileName")
	}
	if licenseURL == "" {
		t.Error("Expected non-empty licenseURL")
	}
	if rawURL == "" {
		t.Error("Expected non-empty rawURL")
	}

	t.Logf("Found license: %s", fileName)
	t.Logf("License URL: %s", licenseURL)
}

func TestGitClientNoLicenseFile(t *testing.T) {
	client := NewGitClient("", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// octocat/Hello-World doesn't have a LICENSE file
	fileName, _, _, err := client.GetLicenseFile(ctx, "octocat", "Hello-World", "")
	if err != nil {
		t.Fatalf("GetLicenseFile failed: %v", err)
	}

	if fileName != "" {
		t.Errorf("Expected empty fileName for repo without license, got %q", fileName)
	}
}

func TestGitClientCaching(t *testing.T) {
	client := NewGitClient("", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First call - should be a miss
	_, err := client.GetRepositoryInfo(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("GetRepositoryInfo failed: %v", err)
	}

	requests1, hits1, misses1 := client.Stats()
	t.Logf("After first call: requests=%d, hits=%d, misses=%d", requests1, hits1, misses1)

	if misses1 == 0 {
		t.Error("Expected at least one cache miss")
	}

	// Second call - should be a hit
	_, err = client.GetRepositoryInfo(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("GetRepositoryInfo (cached) failed: %v", err)
	}

	requests2, hits2, misses2 := client.Stats()
	t.Logf("After second call: requests=%d, hits=%d, misses=%d", requests2, hits2, misses2)

	if hits2 <= hits1 {
		t.Error("Expected cache hit for second call")
	}
	if requests2 != requests1 {
		t.Error("Expected no new requests for cached result")
	}
	if misses2 != misses1 {
		t.Error("Expected no new misses for cached result")
	}
}
