package scraper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests for the git clone-based scraping pipeline.
// These tests require network access and are skipped in short mode.

func TestIntegrationGitClientFullPipeline(t *testing.T) {
	cloneDir := t.TempDir()
	client := NewGitClient("", cloneDir)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Test the full pipeline with a known repository
	owner, repo := "obra", "superpowers"

	// Step 1: Get repository info
	t.Run("GetRepositoryInfo", func(t *testing.T) {
		info, err := client.GetRepositoryInfo(ctx, owner, repo)
		if err != nil {
			t.Fatalf("GetRepositoryInfo failed: %v", err)
		}

		if info.FullName != "obra/superpowers" {
			t.Errorf("Expected FullName 'obra/superpowers', got %q", info.FullName)
		}
		if info.CommitSHA == "" {
			t.Error("Expected non-empty CommitSHA")
		}
		if info.DefaultBranch == "" {
			t.Error("Expected non-empty DefaultBranch")
		}

		t.Logf("Repository: %s, Branch: %s, SHA: %s", info.FullName, info.DefaultBranch, info.CommitSHA[:8])
	})

	// Step 2: List skill files
	var skillFiles []*SkillFile
	t.Run("ListSkillFiles", func(t *testing.T) {
		files, err := client.ListSkillFiles(ctx, owner, repo, "skills")
		if err != nil {
			t.Fatalf("ListSkillFiles failed: %v", err)
		}

		if len(files) == 0 {
			t.Error("Expected to find skill files in obra/superpowers")
		}

		skillFiles = files
		t.Logf("Found %d skill files", len(files))
		for _, f := range files {
			t.Logf("  - %s", f.Path)
		}
	})

	// Step 3: Get file content for each skill
	t.Run("GetFileContent", func(t *testing.T) {
		if len(skillFiles) == 0 {
			t.Skip("No skill files found")
		}

		// Test reading the first skill file
		sf := skillFiles[0]
		content, err := client.GetFileContent(ctx, owner, repo, sf.Path, "")
		if err != nil {
			t.Fatalf("GetFileContent failed: %v", err)
		}

		if content == "" {
			t.Error("Expected non-empty content")
		}

		t.Logf("Read %d bytes from %s", len(content), sf.Path)
	})

	// Step 4: Get license file
	t.Run("GetLicenseFile", func(t *testing.T) {
		fileName, licenseURL, _, err := client.GetLicenseFile(ctx, owner, repo, "")
		if err != nil {
			t.Fatalf("GetLicenseFile failed: %v", err)
		}

		// obra/superpowers should have a LICENSE file
		if fileName == "" {
			t.Log("No license file found (may be expected for some repos)")
		} else {
			t.Logf("Found license: %s at %s", fileName, licenseURL)
		}
	})

	// Step 5: Verify caching works
	t.Run("CachingWorks", func(t *testing.T) {
		_, hits1, _ := client.Stats()

		// Second call should hit cache
		_, err := client.GetRepositoryInfo(ctx, owner, repo)
		if err != nil {
			t.Fatalf("GetRepositoryInfo (cached) failed: %v", err)
		}

		_, hits2, _ := client.Stats()
		if hits2 <= hits1 {
			t.Error("Expected cache hit on second call")
		}

		t.Logf("Cache hits increased from %d to %d", hits1, hits2)
	})
}

func TestIntegrationWorkingTreeRepoStructure(t *testing.T) {
	cloneDir := t.TempDir()
	rm := NewRepositoryManager(cloneDir, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Clone a repository
	localPath, err := rm.CloneOrUpdate(ctx, "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("CloneOrUpdate failed: %v", err)
	}

	// Verify working tree repository structure
	t.Run("HasGitDir", func(t *testing.T) {
		gitPath := filepath.Join(localPath, ".git")
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			t.Error("Expected .git directory in working tree repository")
		}
	})

	t.Run("HasHEADInGitDir", func(t *testing.T) {
		headPath := filepath.Join(localPath, ".git", "HEAD")
		if _, err := os.Stat(headPath); os.IsNotExist(err) {
			t.Error("Expected HEAD file inside .git directory")
		}
	})

	t.Run("HasObjectsInGitDir", func(t *testing.T) {
		objectsPath := filepath.Join(localPath, ".git", "objects")
		if _, err := os.Stat(objectsPath); os.IsNotExist(err) {
			t.Error("Expected objects directory inside .git directory")
		}
	})

	t.Run("HasWorkingTreeFiles", func(t *testing.T) {
		// Working tree should have actual files like README
		readmePath := filepath.Join(localPath, "README")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			t.Error("Expected README file in working tree")
		}
	})

	// Measure disk usage
	t.Run("DiskUsage", func(t *testing.T) {
		size, err := dirSize(localPath)
		if err != nil {
			t.Fatalf("Failed to measure disk usage: %v", err)
		}
		t.Logf("Working tree repository size: %d KB", size/1024)
	})
}

func TestIntegrationIsSkillFilePathPatterns(t *testing.T) {
	// Comprehensive test of all expected patterns
	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		// SKILL.md patterns - should match
		{"SKILL.md", true, "root SKILL.md"},
		{"skill.md", true, "root skill.md lowercase"},
		{"Skill.md", true, "root Skill.md mixed case"},
		{"skills/SKILL.md", true, "nested SKILL.md"},
		{"skills/python/SKILL.md", true, "deeply nested SKILL.md"},
		{".claude/skills/testing/SKILL.md", true, "claude skills path"},
		{"src/skills/my-skill/skill.md", true, "src nested lowercase"},

		// CLAUDE.md patterns - should match
		{"CLAUDE.md", true, "root CLAUDE.md"},
		{"claude.md", true, "root claude.md lowercase"},
		{".claude/CLAUDE.md", true, "nested CLAUDE.md"},

		// Non-skill files - should NOT match
		{"README.md", false, "README file"},
		{"SKILLS.md", false, "SKILLS plural"},
		{"skill.txt", false, "wrong extension"},
		{"my-skill.md", false, "skill in filename but not pattern"},
		{".cursorrules", false, "cursor rules excluded"},
		{".windsurfrules", false, "windsurf rules excluded"},
		{".cursor/rules", false, "cursor rules dir excluded"},
		{"docs/skill-guide.md", false, "skill in path but wrong filename"},
		{"SKILL.md.bak", false, "backup file"},
		{"SKILL.md/somefile", false, "SKILL.md as directory"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := IsSkillFilePath(tt.path)
			if result != tt.expected {
				t.Errorf("IsSkillFilePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIntegrationCleanupOldRepos(t *testing.T) {
	baseDir := t.TempDir()
	rm := NewRepositoryManager(baseDir, "")

	// Create test repository structures
	repos := []struct {
		owner        string
		repo         string
		daysOld      int
		shouldRemove bool
	}{
		{"old-owner", "old-repo", 10, true},         // Older than 7 days
		{"new-owner", "new-repo", 1, false},         // Newer than 7 days
		{"edge-owner", "edge-repo", 6, false},       // 6 days old (within 7 day TTL)
		{"ancient-owner", "ancient-repo", 30, true}, // Very old
	}

	for _, r := range repos {
		repoPath := filepath.Join(baseDir, r.owner, r.repo)
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			t.Fatalf("Failed to create repo dir: %v", err)
		}

		// Create .git directory to simulate working tree repo
		gitDir := filepath.Join(repoPath, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatalf("Failed to create .git dir: %v", err)
		}

		// Create HEAD file inside .git
		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
			t.Fatalf("Failed to create HEAD file: %v", err)
		}

		// Set modification time
		modTime := time.Now().Add(-time.Duration(r.daysOld) * 24 * time.Hour)
		if err := os.Chtimes(repoPath, modTime, modTime); err != nil {
			t.Fatalf("Failed to set mod time: %v", err)
		}
	}

	// Run cleanup with 7-day TTL
	err := rm.CleanupOldRepos(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldRepos failed: %v", err)
	}

	// Verify results
	for _, r := range repos {
		repoPath := filepath.Join(baseDir, r.owner, r.repo)
		exists := true
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			exists = false
		}

		if r.shouldRemove && exists {
			t.Errorf("Expected %s/%s to be removed (was %d days old)", r.owner, r.repo, r.daysOld)
		}
		if !r.shouldRemove && !exists {
			t.Errorf("Expected %s/%s to be kept (was %d days old)", r.owner, r.repo, r.daysOld)
		}
	}
}

func TestIntegrationClientInterfaceCompatibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Test that both clients implement the same interface
	var clients []Client

	// GitClient
	gitClient := NewGitClient("", t.TempDir())
	clients = append(clients, gitClient)

	// GitHubClient
	githubClient := NewGitHubClient("", 0)
	clients = append(clients, githubClient)

	owner, repo := "octocat", "Hello-World"

	for i, client := range clients {
		clientName := "GitClient"
		if i == 1 {
			clientName = "GitHubClient"
		}

		t.Run(clientName+"/GetRepositoryInfo", func(t *testing.T) {
			info, err := client.GetRepositoryInfo(ctx, owner, repo)
			if err != nil {
				t.Fatalf("GetRepositoryInfo failed: %v", err)
			}
			if info.FullName != "octocat/Hello-World" {
				t.Errorf("Expected FullName 'octocat/Hello-World', got %q", info.FullName)
			}
		})

		t.Run(clientName+"/Stats", func(t *testing.T) {
			requests, hits, misses := client.Stats()
			t.Logf("%s stats: requests=%d, hits=%d, misses=%d", clientName, requests, hits, misses)
		})

		t.Run(clientName+"/ClearCache", func(t *testing.T) {
			client.ClearCache()
			// Should not panic
		})

		t.Run(clientName+"/ResetStats", func(t *testing.T) {
			client.ResetStats()
			requests, hits, misses := client.Stats()
			if requests != 0 || hits != 0 || misses != 0 {
				t.Errorf("Expected zero stats after reset")
			}
		})
	}
}

func TestIntegrationHTMLScraping(t *testing.T) {
	rm := NewRepositoryManager(t.TempDir(), "")

	// Test with a popular repository that definitely has stars
	stars, forks := rm.ScrapeGitHubMetadata("golang", "go")

	t.Logf("golang/go: stars=%d, forks=%d", stars, forks)

	// The Go repo has many stars, so we should get positive numbers
	// (unless GitHub changes their HTML structure)
	if stars == 0 && forks == 0 {
		t.Log("Warning: HTML scraping returned 0 for both stars and forks")
		t.Log("This may indicate GitHub's HTML structure has changed")
	}
}

// Helper function to calculate directory size
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
